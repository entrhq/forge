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

// TestExtractToolName tests the extractToolName helper function
func TestExtractToolName(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "valid tool name",
			content:  `{"tool_name": "read_file", "arguments": {}}`,
			expected: "read_file",
		},
		{
			name:     "tool name with whitespace",
			content:  `{"tool_name":  "execute_command"  , "arguments": {}}`,
			expected: "execute_command",
		},
		{
			name:     "tool name in full tool call",
			content:  `<tool>{"server_name": "local", "tool_name": "ask_question", "arguments": {"question": "What?"}}</tool>`,
			expected: "ask_question",
		},
		{
			name:     "no tool name",
			content:  "Just some content",
			expected: "",
		},
		{
			name:     "incomplete JSON",
			content:  `{"tool_name": "incomplete`,
			expected: "",
		},
		{
			name:     "empty tool name",
			content:  `{"tool_name": "", "arguments": {}}`,
			expected: "",
		},
		{
			name:     "tool name with server_name",
			content:  `{"server_name": "mcp", "tool_name": "task_completion", "arguments": {"result": "done"}}`,
			expected: "task_completion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractToolName(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGroupToolCallsAndResults_WithExclusions tests that excluded tools are not grouped
func TestGroupToolCallsAndResults_WithExclusions(t *testing.T) {
	// Create test messages with mix of regular and excluded tools
	messages := []*types.Message{
		types.NewAssistantMessage(`<tool>{"tool_name": "read_file", "arguments": {"path": "test.go"}}</tool>`),
		types.NewToolMessage("File content here"),
		types.NewAssistantMessage(`<tool>{"tool_name": "ask_question", "arguments": {"question": "Continue?"}}</tool>`),
		types.NewToolMessage("User answered: yes"),
		types.NewAssistantMessage(`<tool>{"tool_name": "execute_command", "arguments": {"command": "ls"}}</tool>`),
		types.NewToolMessage("Command output"),
		types.NewAssistantMessage(`<tool>{"tool_name": "task_completion", "arguments": {"result": "done"}}</tool>`),
		types.NewToolMessage("Task completed"),
	}

	// Exclude ask_question and task_completion
	excludedTools := map[string]bool{
		"ask_question":    true,
		"task_completion": true,
	}

	groups := groupToolCallsAndResults(messages, excludedTools)

	// Should have only 2 groups (read_file and execute_command)
	// ask_question and task_completion should be excluded
	assert.Len(t, groups, 2, "Should have 2 groups (excluded tools should not be grouped)")

	// Verify first group is read_file
	assert.Contains(t, groups[0][0].Content, "read_file")
	assert.Equal(t, types.RoleTool, groups[0][1].Role)

	// Verify second group is execute_command
	assert.Contains(t, groups[1][0].Content, "execute_command")
	assert.Equal(t, types.RoleTool, groups[1][1].Role)
}

// TestGroupToolCallsAndResults_NoExclusions tests that all tools are grouped when no exclusions
func TestGroupToolCallsAndResults_NoExclusions(t *testing.T) {
	messages := []*types.Message{
		types.NewAssistantMessage(`<tool>{"tool_name": "read_file", "arguments": {}}</tool>`),
		types.NewToolMessage("File content"),
		types.NewAssistantMessage(`<tool>{"tool_name": "ask_question", "arguments": {}}</tool>`),
		types.NewToolMessage("User answered"),
	}

	// No exclusions
	excludedTools := map[string]bool{}

	groups := groupToolCallsAndResults(messages, excludedTools)

	// Should have 2 groups (both tools included)
	assert.Len(t, groups, 2, "Should have 2 groups when no exclusions")
}

// TestGroupToolCallsAndResults_SkipsSummarized tests that already summarized messages are skipped
func TestGroupToolCallsAndResults_SkipsSummarized(t *testing.T) {
	messages := []*types.Message{
		types.NewAssistantMessage(`<tool>{"tool_name": "read_file", "arguments": {}}</tool>`),
		types.NewToolMessage("File content"),
	}

	// Mark first message as summarized
	messages[0].WithMetadata("summarized", true)

	excludedTools := map[string]bool{}

	groups := groupToolCallsAndResults(messages, excludedTools)

	// Should have 1 group with just the tool result (assistant message skipped)
	// This is acceptable behavior - incomplete groups are still added
	assert.Len(t, groups, 1, "Should have 1 group with remaining tool result")
	assert.Len(t, groups[0], 1, "Group should have 1 message (tool result only)")
	assert.Equal(t, types.RoleTool, groups[0][0].Role, "Should be tool result message")
}

// TestGroupToolCallsAndResults_PreservesSystemMessages tests that system messages are not grouped
func TestGroupToolCallsAndResults_PreservesSystemMessages(t *testing.T) {
	messages := []*types.Message{
		types.NewSystemMessage("System instruction"),
		types.NewAssistantMessage(`<tool>{"tool_name": "read_file", "arguments": {}}</tool>`),
		types.NewToolMessage("File content"),
		types.NewSystemMessage("Another system message"),
	}

	excludedTools := map[string]bool{}

	groups := groupToolCallsAndResults(messages, excludedTools)

	// Should have 1 group (system messages excluded)
	assert.Len(t, groups, 1, "Should have 1 group, system messages excluded")

	// Verify no system messages in groups
	for _, group := range groups {
		for _, msg := range group {
			assert.NotEqual(t, types.RoleSystem, msg.Role, "System messages should not be in groups")
		}
	}
}

// TestNewToolCallSummarizationStrategy_DefaultExclusions tests default exclusions
func TestNewToolCallSummarizationStrategy_DefaultExclusions(t *testing.T) {
	strategy := NewToolCallSummarizationStrategy(20, 10, 40)

	// Verify default exclusions
	assert.True(t, strategy.excludedTools["task_completion"], "task_completion should be excluded by default")
	assert.True(t, strategy.excludedTools["ask_question"], "ask_question should be excluded by default")
	assert.True(t, strategy.excludedTools["converse"], "converse should be excluded by default")
	assert.Len(t, strategy.excludedTools, 3, "Should have exactly 3 default exclusions")
}

// TestNewToolCallSummarizationStrategy_CustomExclusions tests custom exclusions
func TestNewToolCallSummarizationStrategy_CustomExclusions(t *testing.T) {
	strategy := NewToolCallSummarizationStrategy(20, 10, 40, "custom_tool", "another_tool")

	// Verify custom exclusions (should replace defaults)
	assert.True(t, strategy.excludedTools["custom_tool"], "custom_tool should be excluded")
	assert.True(t, strategy.excludedTools["another_tool"], "another_tool should be excluded")
	assert.False(t, strategy.excludedTools["task_completion"], "task_completion should not be excluded when custom list provided")
	assert.Len(t, strategy.excludedTools, 2, "Should have exactly 2 custom exclusions")
}

// TestNewToolCallSummarizationStrategy_EmptyExclusions tests empty exclusion list
func TestNewToolCallSummarizationStrategy_EmptyExclusions(t *testing.T) {
	// Passing empty slice should use defaults
	strategy := NewToolCallSummarizationStrategy(20, 10, 40)

	assert.Len(t, strategy.excludedTools, 3, "Empty exclusion list should use defaults")
}

// TestSummarize_ExcludesLoopBreakingTools tests end-to-end summarization with exclusions
func TestSummarize_ExcludesLoopBreakingTools(t *testing.T) {
	strategy := NewToolCallSummarizationStrategy(5, 2, 20) // Lower thresholds for testing
	conv := memory.NewConversationMemory()

	// Add messages: mix of regular tools and excluded tools
	conv.Add(types.NewAssistantMessage(`<tool>{"tool_name": "read_file", "arguments": {}}</tool>`))
	conv.Add(types.NewToolMessage("File content here"))
	conv.Add(types.NewAssistantMessage(`<tool>{"tool_name": "ask_question", "arguments": {}}</tool>`))
	conv.Add(types.NewToolMessage("User said yes"))
	conv.Add(types.NewAssistantMessage(`<tool>{"tool_name": "execute_command", "arguments": {}}</tool>`))
	conv.Add(types.NewToolMessage("Command executed"))

	// Add recent messages to stay above threshold
	for i := 0; i < 6; i++ {
		conv.Add(types.NewUserMessage("Recent message"))
	}

	// Mock LLM
	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).Return(
		types.NewAssistantMessage("Summary of tool call"),
		nil,
	)

	ctx := context.Background()
	count, err := strategy.Summarize(ctx, conv, mockLLM)

	assert.NoError(t, err)
	assert.Equal(t, 2, count, "Should summarize 2 groups (read_file and execute_command, excluding ask_question)")

	// Verify ask_question is still in conversation (not summarized)
	messages := conv.GetAll()
	foundAskQuestion := false
	for _, msg := range messages {
		if msg.Role == types.RoleAssistant && containsToolCallIndicators(msg.Content) {
			toolName := extractToolName(msg.Content)
			if toolName == "ask_question" {
				foundAskQuestion = true
				// Verify it's NOT summarized
				summarized, _ := msg.Metadata["summarized"].(bool)
				assert.False(t, summarized, "ask_question should not be summarized")
			}
		}
	}
	assert.True(t, foundAskQuestion, "ask_question should still be in conversation")
}

// TestSummarize_PreservesUserMessages verifies that user (human) messages in the
// "old" portion of the conversation are never dropped during tool call summarization.
// This is the core regression test for the user-message-loss bug.
func TestSummarize_PreservesUserMessages(t *testing.T) {
	strategy := NewToolCallSummarizationStrategy(5, 2, 20)
	conv := memory.NewConversationMemory()

	userMsg1 := "Please read the config file"
	userMsg2 := "Now show me the tests"

	// Interleave user messages with tool calls in the "old" window
	conv.Add(types.NewUserMessage(userMsg1))
	conv.Add(types.NewAssistantMessage(`<tool>{"tool_name": "read_file", "arguments": {}}</tool>`))
	conv.Add(types.NewToolMessage("config file content"))
	conv.Add(types.NewUserMessage(userMsg2))
	conv.Add(types.NewAssistantMessage(`<tool>{"tool_name": "execute_command", "arguments": {}}</tool>`))
	conv.Add(types.NewToolMessage("test output"))

	// Add recent messages to push the above into "old" territory
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

// TestShouldRun_WithExcludedTools tests that ShouldRun works correctly with exclusions
func TestShouldRun_WithExcludedTools(t *testing.T) {
	strategy := NewToolCallSummarizationStrategy(5, 2, 20)
	conv := memory.NewConversationMemory()

	// Add only excluded tools (should not trigger summarization)
	conv.Add(types.NewAssistantMessage(`<tool>{"tool_name": "ask_question", "arguments": {}}</tool>`))
	conv.Add(types.NewToolMessage("User answered"))
	conv.Add(types.NewAssistantMessage(`<tool>{"tool_name": "task_completion", "arguments": {}}</tool>`))
	conv.Add(types.NewToolMessage("Task done"))

	// Add recent messages
	for i := 0; i < 6; i++ {
		conv.Add(types.NewUserMessage("Recent message"))
	}

	// Note: ShouldRun doesn't check exclusions (it just counts tool calls)
	// This is expected behavior - exclusions happen during Summarize
	shouldRun := strategy.ShouldRun(conv, 1000, 2000)

	// Should run because there are enough old tool calls (even though they'll be excluded during grouping)
	assert.True(t, shouldRun, "ShouldRun should trigger based on tool call count")
}
