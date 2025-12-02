package agent

import (
	"context"
	"testing"

	"github.com/entrhq/forge/pkg/agent/memory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/llm/tokenizer"
	"github.com/entrhq/forge/pkg/types"
)

// mockProvider is a minimal LLM provider for testing
type mockProvider struct{}

func (m *mockProvider) StreamCompletion(ctx context.Context, messages []*types.Message) (<-chan *llm.StreamChunk, error) {
	ch := make(chan *llm.StreamChunk)
	close(ch)
	return ch, nil
}

func (m *mockProvider) Complete(ctx context.Context, messages []*types.Message) (*types.Message, error) {
	return types.NewMessage(types.RoleAssistant, "mock response"), nil
}

func (m *mockProvider) GetModelInfo() *types.ModelInfo {
	return &types.ModelInfo{Name: "mock-model"}
}

func (m *mockProvider) GetModel() string {
	return "mock-model"
}

func (m *mockProvider) GetBaseURL() string {
	return "https://mock.api"
}

func (m *mockProvider) GetAPIKey() string {
	return "mock-key"
}

// TestMemoryAddAndRetrieve verifies that messages are properly added to memory
// and can be retrieved for token counting
func TestMemoryAddAndRetrieve(t *testing.T) {
	// Create a fresh agent with default settings
	provider := &mockProvider{}
	agent := NewDefaultAgent(provider)

	// Verify initial state - memory should be empty
	initialMessages := agent.memory.GetAll()
	if len(initialMessages) != 0 {
		t.Errorf("Expected empty memory initially, got %d messages", len(initialMessages))
	}

	// Add a user message
	userMsg := types.NewUserMessage("Hello, this is a test message")
	agent.memory.Add(userMsg)

	// Verify message was added
	messages := agent.memory.GetAll()
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message after adding user message, got %d", len(messages))
	}

	if messages[0].Role != types.RoleUser {
		t.Errorf("Expected user role, got %v", messages[0].Role)
	}

	if messages[0].Content != userMsg.Content {
		t.Errorf("Expected content %q, got %q", userMsg.Content, messages[0].Content)
	}

	// Add an assistant message
	assistantMsg := &types.Message{
		Role:    types.RoleAssistant,
		Content: "This is the assistant's response",
	}
	agent.memory.Add(assistantMsg)

	// Verify both messages are present
	messages = agent.memory.GetAll()
	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages after adding assistant message, got %d", len(messages))
	}

	if messages[1].Role != types.RoleAssistant {
		t.Errorf("Expected assistant role, got %v", messages[1].Role)
	}
}

// TestTokenCountingWithMessages verifies that token counting works correctly
// when messages are present in memory
func TestTokenCountingWithMessages(t *testing.T) {
	// Create agent with tokenizer
	provider := &mockProvider{}
	agent := NewDefaultAgent(provider)

	// Initialize tokenizer (this may fail if tiktoken can't be initialized)
	tok, err := tokenizer.New()
	if err != nil {
		t.Logf("Tokenizer initialization failed (expected in some environments): %v", err)
		// Test will verify fallback counting works
	} else {
		agent.tokenizer = tok
	}

	// Add some test messages
	agent.memory.Add(types.NewUserMessage("Write a function to add two numbers"))
	agent.memory.Add(&types.Message{
		Role:    types.RoleAssistant,
		Content: "Here's a simple function:\nfunc add(a, b int) int { return a + b }",
	})
	agent.memory.Add(types.NewUserMessage("Thanks!"))

	// Get context info which calculates tokens
	info := agent.GetContextInfo()

	// Verify we have messages
	if info.MessageCount != 3 {
		t.Errorf("Expected 3 messages, got %d", info.MessageCount)
	}

	// Verify conversation tokens are non-zero
	if info.ConversationTokens == 0 {
		t.Errorf("Expected non-zero conversation tokens, got 0. Messages in memory: %d", len(agent.memory.GetAll()))

		// Debug: print actual messages
		messages := agent.memory.GetAll()
		for i, msg := range messages {
			t.Logf("Message %d: Role=%v, Content length=%d", i, msg.Role, len(msg.Content))
		}

		// Verify tokenizer state
		t.Logf("Tokenizer nil: %v", agent.tokenizer == nil)
	}

	// Current tokens should be at least as large as conversation tokens
	if info.CurrentContextTokens < info.ConversationTokens {
		t.Errorf("Current tokens (%d) should be >= conversation tokens (%d)",
			info.CurrentContextTokens, info.ConversationTokens)
	}

	// Test the fallback counting directly if tokenizer failed
	if agent.tokenizer == nil {
		messages := agent.memory.GetAll()
		fallbackCount := 0
		for _, msg := range messages {
			fallbackCount += (len(msg.Content) + len(string(msg.Role)) + 12) / 4
		}

		if fallbackCount == 0 {
			t.Errorf("Fallback counting also returned 0, but we have %d messages", len(messages))
		}

		t.Logf("Fallback counting calculated %d tokens for %d messages", fallbackCount, len(messages))
	}
}

// TestConversationMemoryTokenCounting specifically tests the ConversationMemory implementation
func TestConversationMemoryTokenCounting(t *testing.T) {
	// Create memory directly
	mem := memory.NewConversationMemory()

	// Add test messages
	mem.Add(types.NewUserMessage("Hello"))
	mem.Add(&types.Message{Role: types.RoleAssistant, Content: "Hi there!"})
	mem.Add(types.NewUserMessage("How are you?"))

	// Verify messages are stored
	messages := mem.GetAll()
	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages in memory, got %d", len(messages))
	}

	// Create tokenizer and count tokens
	tok, err := tokenizer.New()
	if err != nil {
		t.Logf("Tokenizer init failed: %v, testing fallback", err)
		// Test fallback
		fallbackCount := 0
		for _, msg := range messages {
			fallbackCount += (len(msg.Content) + len(string(msg.Role)) + 12) / 4
		}
		if fallbackCount == 0 {
			t.Errorf("Fallback counting returned 0 for %d messages", len(messages))
		}
		t.Logf("Fallback count: %d tokens", fallbackCount)
	} else {
		// Test real tokenizer
		count := tok.CountMessagesTokens(messages)
		if count == 0 {
			t.Errorf("Tokenizer returned 0 tokens for %d messages", len(messages))
		}
		t.Logf("Tokenizer count: %d tokens", count)
	}
}

// TestEmptyMemoryTokenCounting verifies token counting works with empty memory
func TestEmptyMemoryTokenCounting(t *testing.T) {
	provider := &mockProvider{}
	agent := NewDefaultAgent(provider)

	info := agent.GetContextInfo()

	// Empty memory should have 0 conversation tokens
	if info.ConversationTokens != 0 {
		t.Errorf("Expected 0 conversation tokens for empty memory, got %d", info.ConversationTokens)
	}

	// Should still have system prompt tokens
	if info.SystemPromptTokens == 0 {
		t.Errorf("Expected non-zero system prompt tokens, got 0")
	}
}
