package context

import (
	"context"
	"testing"

	"github.com/entrhq/forge/pkg/agent/memory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLMProvider is a mock implementation of llm.Provider for testing
type MockLLMProvider struct {
	mock.Mock
}

func (m *MockLLMProvider) StreamCompletion(ctx context.Context, messages []*types.Message) (<-chan *llm.StreamChunk, error) {
	args := m.Called(ctx, messages)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan *llm.StreamChunk), args.Error(1)
}

func (m *MockLLMProvider) Complete(ctx context.Context, messages []*types.Message) (*types.Message, error) {
	args := m.Called(ctx, messages)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Message), args.Error(1)
}

func (m *MockLLMProvider) GetModelInfo() *types.ModelInfo {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*types.ModelInfo)
}

func (m *MockLLMProvider) GetModel() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockLLMProvider) GetBaseURL() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockLLMProvider) GetAPIKey() string {
	args := m.Called()
	return args.String(0)
}

// toolCallMsg returns an assistant message containing an XML tool call for the given tool name.
func toolCallMsg(toolName string) *types.Message {
	return types.NewAssistantMessage(
		"<tool><server_name>local</server_name><tool_name>" + toolName + "</tool_name><arguments></arguments></tool>",
	)
}

// TestGroupToolCallsAndResults_AllToolsGrouped verifies that every tool call pair
// (including formerly-excluded loop-breaking tools) is grouped for summarization.
func TestGroupToolCallsAndResults_AllToolsGrouped(t *testing.T) {
	messages := []*types.Message{
		toolCallMsg("read_file"),
		types.NewToolMessage("File content here"),
		toolCallMsg("ask_question"),
		types.NewToolMessage("User answered: yes"),
		toolCallMsg("execute_command"),
		types.NewToolMessage("Command output"),
		toolCallMsg("task_completion"),
		types.NewToolMessage("Task completed"),
	}

	groups := groupToolCallsAndResults(messages)

	// All 4 tool call pairs should be grouped — no exclusions anymore.
	assert.Len(t, groups, 4, "All tool call pairs should be grouped")
	assert.Contains(t, groups[0][0].Content, "read_file")
	assert.Contains(t, groups[1][0].Content, "ask_question")
	assert.Contains(t, groups[2][0].Content, "execute_command")
	assert.Contains(t, groups[3][0].Content, "task_completion")
}

// TestGroupToolCallsAndResults_SkipsSummarized tests that already summarized messages are skipped.
func TestGroupToolCallsAndResults_SkipsSummarized(t *testing.T) {
	messages := []*types.Message{
		toolCallMsg("read_file"),
		types.NewToolMessage("File content"),
	}

	// Mark the assistant message as already summarized.
	messages[0].WithMetadata("summarized", true)

	groups := groupToolCallsAndResults(messages)

	// Only the orphaned tool result should form a group.
	assert.Len(t, groups, 1, "Should have 1 group with remaining tool result")
	assert.Len(t, groups[0], 1, "Group should have 1 message (tool result only)")
	assert.Equal(t, types.RoleTool, groups[0][0].Role, "Should be tool result message")
}

// TestGroupToolCallsAndResults_SkipsSystemMessages tests that system messages are never grouped.
func TestGroupToolCallsAndResults_SkipsSystemMessages(t *testing.T) {
	messages := []*types.Message{
		types.NewSystemMessage("System instruction"),
		toolCallMsg("read_file"),
		types.NewToolMessage("File content"),
		types.NewSystemMessage("Another system message"),
	}

	groups := groupToolCallsAndResults(messages)

	assert.Len(t, groups, 1, "Should have 1 group; system messages excluded")
	for _, group := range groups {
		for _, msg := range group {
			assert.NotEqual(t, types.RoleSystem, msg.Role, "System messages must not appear in groups")
		}
	}
}

// TestNewToolCallSummarizationStrategy_Constructor tests constructor parameter defaults.
func TestNewToolCallSummarizationStrategy_Constructor(t *testing.T) {
	tests := []struct {
		name                    string
		threshold, min, maxDist int
		wantThreshold           int
		wantMin                 int
		wantMax                 int
	}{
		{"valid params", 20, 10, 40, 20, 10, 40},
		{"zero threshold uses default", 0, 10, 40, 20, 10, 40},
		{"zero min uses default", 20, 0, 40, 20, 10, 40},
		{"zero maxDist uses default", 20, 10, 0, 20, 10, 40},
		{"all zeros use defaults", 0, 0, 0, 20, 10, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewToolCallSummarizationStrategy(tt.threshold, tt.min, tt.maxDist)
			assert.Equal(t, tt.wantThreshold, s.messagesOldThreshold)
			assert.Equal(t, tt.wantMin, s.minToolCallsToSummarize)
			assert.Equal(t, tt.wantMax, s.maxToolCallDistance)
		})
	}
}

// TestSummarize_AllToolCallsSummarized verifies that all tool call types
// (including formerly-excluded tools) are included in summarization.
func TestSummarize_AllToolCallsSummarized(t *testing.T) {
	strategy := NewToolCallSummarizationStrategy(5, 2, 20)
	conv := memory.NewConversationMemory()

	conv.Add(toolCallMsg("read_file"))
	conv.Add(types.NewToolMessage("File content here"))
	conv.Add(toolCallMsg("ask_question"))
	conv.Add(types.NewToolMessage("User said yes"))
	conv.Add(toolCallMsg("execute_command"))
	conv.Add(types.NewToolMessage("Command executed"))

	// Add recent messages to push the above into "old" territory.
	for i := 0; i < 6; i++ {
		conv.Add(types.NewUserMessage("Recent message"))
	}

	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).Return(
		types.NewAssistantMessage("Summary of tool calls"),
		nil,
	)

	ctx := context.Background()
	count, err := strategy.Summarize(ctx, conv, mockLLM)

	assert.NoError(t, err)
	assert.Equal(t, 3, count, "Should summarize all 3 tool call groups")

	// After summarization, no raw tool call messages should remain in the old window.
	messages := conv.GetAll()
	for _, msg := range messages {
		if isSummarized(msg) {
			continue
		}
		assert.False(
			t,
			msg.Role == types.RoleAssistant && containsToolCallIndicators(msg.Content),
			"No raw tool call messages should survive summarization: %q", msg.Content,
		)
	}
}

// TestSummarize_PreservesUserMessages verifies that user (human) messages in the
// "old" portion of the conversation are never dropped during tool call summarization.
func TestSummarize_PreservesUserMessages(t *testing.T) {
	strategy := NewToolCallSummarizationStrategy(5, 2, 20)
	conv := memory.NewConversationMemory()

	userMsg1 := "Please read the config file"
	userMsg2 := "Now show me the tests"

	conv.Add(types.NewUserMessage(userMsg1))
	conv.Add(toolCallMsg("read_file"))
	conv.Add(types.NewToolMessage("config file content"))
	conv.Add(types.NewUserMessage(userMsg2))
	conv.Add(toolCallMsg("execute_command"))
	conv.Add(types.NewToolMessage("test output"))

	// Add recent messages to push the above into "old" territory.
	for i := 0; i < 6; i++ {
		conv.Add(types.NewAssistantMessage("Working on it..."))
	}

	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).Return(
		types.NewAssistantMessage("Summary of tool call"),
		nil,
	)

	ctx := context.Background()
	_, err := strategy.Summarize(ctx, conv, mockLLM)
	assert.NoError(t, err)

	messages := conv.GetAll()
	foundUser1, foundUser2 := false, false
	for _, msg := range messages {
		if msg.Role == types.RoleUser && msg.Content == userMsg1 {
			foundUser1 = true
		}
		if msg.Role == types.RoleUser && msg.Content == userMsg2 {
			foundUser2 = true
		}
	}

	assert.True(t, foundUser1, "First user message must be preserved after tool call summarization")
	assert.True(t, foundUser2, "Second user message must be preserved after tool call summarization")
}
