package context

import (
	"context"
	"testing"

	"github.com/entrhq/forge/pkg/agent/memory"
	"github.com/entrhq/forge/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestNewThresholdSummarizationStrategy tests constructor clamping and defaults.
func TestNewThresholdSummarizationStrategy(t *testing.T) {
	tests := []struct {
		name                    string
		thresholdPercent        float64
		messagesPerSummary      int
		wantThresholdPercent    float64
		wantMessagesPerSummary  int
	}{
		{
			name:                   "valid inputs",
			thresholdPercent:       80,
			messagesPerSummary:     5,
			wantThresholdPercent:   80,
			wantMessagesPerSummary: 5,
		},
		{
			name:                   "threshold clamped below 0",
			thresholdPercent:       -10,
			messagesPerSummary:     5,
			wantThresholdPercent:   0,
			wantMessagesPerSummary: 5,
		},
		{
			name:                   "threshold clamped above 100",
			thresholdPercent:       150,
			messagesPerSummary:     5,
			wantThresholdPercent:   100,
			wantMessagesPerSummary: 5,
		},
		{
			name:                   "messagesPerSummary clamped below 1",
			thresholdPercent:       80,
			messagesPerSummary:     0,
			wantThresholdPercent:   80,
			wantMessagesPerSummary: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewThresholdSummarizationStrategy(tt.thresholdPercent, tt.messagesPerSummary)
			assert.Equal(t, tt.wantThresholdPercent, s.thresholdPercent)
			assert.Equal(t, tt.wantMessagesPerSummary, s.messagesPerSummary)
		})
	}
}

// TestThresholdStrategy_Name tests the Name method.
func TestThresholdStrategy_Name(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80, 5)
	assert.Equal(t, "ThresholdSummarization", s.Name())
}

// TestThresholdStrategy_ShouldRun tests trigger conditions.
func TestThresholdStrategy_ShouldRun(t *testing.T) {
	tests := []struct {
		name          string
		threshold     float64
		currentTokens int
		maxTokens     int
		want          bool
	}{
		{"above threshold", 80, 850, 1000, true},
		{"at threshold", 80, 800, 1000, true},
		{"below threshold", 80, 750, 1000, false},
		{"zero max tokens", 80, 800, 0, false},
		{"negative max tokens", 80, 800, -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewThresholdSummarizationStrategy(tt.threshold, 5)
			conv := memory.NewConversationMemory()
			got := s.ShouldRun(conv, tt.currentTokens, tt.maxTokens)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestThresholdStrategy_CollectMessagesToSummarize_SkipsUserMessages verifies that
// user (human) messages are never included in the summarization batch, and that
// a user message acts as a block boundary — stopping collection of the current block.
func TestThresholdStrategy_CollectMessagesToSummarize_SkipsUserMessages(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80, 10)

	messages := []*types.Message{
		types.NewUserMessage("What files are in the project?"),
		types.NewAssistantMessage("Let me check."),
		types.NewUserMessage("Also, what does main.go do?"),
		types.NewAssistantMessage("I'll read it now."),
	}

	toSummarize := s.collectMessagesToSummarize(messages)

	// Only assistant messages should be collected.
	for _, msg := range toSummarize {
		assert.Equal(t, types.RoleAssistant, msg.Role,
			"collectMessagesToSummarize must never return a user message; got: %q", msg.Content)
	}

	// The second user message is a block boundary, so collection stops after "Let me check."
	// Only the first assistant block (before user2) is returned.
	assert.Len(t, toSummarize, 1, "Should collect only the first block of assistant messages (before the next user message)")
	assert.Equal(t, "Let me check.", toSummarize[0].Content)
}

// TestThresholdStrategy_CollectMessagesToSummarize_SkipsSystemMessage verifies that
// the system message at index 0 is always skipped.
func TestThresholdStrategy_CollectMessagesToSummarize_SkipsSystemMessage(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80, 10)

	messages := []*types.Message{
		types.NewSystemMessage("You are an AI coding assistant."),
		types.NewUserMessage("Hello"),
		types.NewAssistantMessage("Hi there!"),
	}

	toSummarize := s.collectMessagesToSummarize(messages)

	for _, msg := range toSummarize {
		assert.NotEqual(t, types.RoleSystem, msg.Role,
			"collectMessagesToSummarize must never return a system message")
	}
}

// TestThresholdStrategy_CollectMessagesToSummarize_SkipsAlreadySummarized verifies
// that messages with summarized=true metadata are not collected again.
func TestThresholdStrategy_CollectMessagesToSummarize_SkipsAlreadySummarized(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80, 10)

	alreadySummarized := types.NewAssistantMessage("Old summary").WithMetadata("summarized", true)
	fresh := types.NewAssistantMessage("Fresh response")

	messages := []*types.Message{alreadySummarized, fresh}

	toSummarize := s.collectMessagesToSummarize(messages)

	assert.Len(t, toSummarize, 1, "Should only collect the non-summarized message")
	assert.Equal(t, fresh.Content, toSummarize[0].Content)
}

// TestThresholdStrategy_Summarize_PreservesUserMessages is an end-to-end test
// verifying that after Summarize runs, all original user messages are still
// present in the conversation memory.
func TestThresholdStrategy_Summarize_PreservesUserMessages(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80, 10)
	conv := memory.NewConversationMemory()

	userMsg1 := "What files are in the project?"
	userMsg2 := "Also, what does main.go do?"

	conv.Add(types.NewUserMessage(userMsg1))
	conv.Add(types.NewAssistantMessage("Let me check the directory."))
	conv.Add(types.NewUserMessage(userMsg2))
	conv.Add(types.NewAssistantMessage("I will read main.go for you."))

	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).Return(
		types.NewAssistantMessage("Summary of block"),
		nil,
	)

	_, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)

	messages := conv.GetAll()

	// Both user messages must still be present verbatim.
	foundUser1, foundUser2 := false, false
	for _, msg := range messages {
		if msg.Role == types.RoleUser && msg.Content == userMsg1 {
			foundUser1 = true
		}
		if msg.Role == types.RoleUser && msg.Content == userMsg2 {
			foundUser2 = true
		}
	}

	assert.True(t, foundUser1, "First user message must be preserved after summarization")
	assert.True(t, foundUser2, "Second user message must be preserved after summarization")
}

// TestThresholdStrategy_Summarize_RespectsBlockBoundaries verifies the key ordering
// invariant: assistant-message blocks before and after a user message are summarized
// independently, with each summary inserted at the correct position relative to the
// user turn that separates them.
//
// Input:
//
//	[assistant1] [assistant2] [user1] [assistant3] [assistant4]
//
// Expected output after full summarization:
//
//	[summaryA] [user1] [summaryB]
func TestThresholdStrategy_Summarize_RespectsBlockBoundaries(t *testing.T) {
	// messagesPerSummary=5 so all assistant messages in each block fit in one pass.
	s := NewThresholdSummarizationStrategy(80, 5)
	conv := memory.NewConversationMemory()

	conv.Add(types.NewAssistantMessage("tool call 1"))
	conv.Add(types.NewAssistantMessage("tool call 2"))
	conv.Add(types.NewUserMessage("user turn"))
	conv.Add(types.NewAssistantMessage("tool call 3"))
	conv.Add(types.NewAssistantMessage("tool call 4"))

	callCount := 0
	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).
		Return(types.NewAssistantMessage("summary"), nil).
		Run(func(args mock.Arguments) { callCount++ })

	_, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)

	// The LLM must have been called twice — once per block.
	assert.Equal(t, 2, callCount, "Expected one LLM call per assistant block")

	messages := conv.GetAll()

	// Exactly 3 messages should remain: summaryA, user1, summaryB.
	assert.Len(t, messages, 3, "Expected [summaryA, user1, summaryB]")
	assert.Equal(t, types.RoleAssistant, messages[0].Role, "First message should be a summary")
	assert.True(t, messages[0].Metadata["summarized"].(bool), "First message should be marked summarized")
	assert.Equal(t, types.RoleUser, messages[1].Role, "Middle message should be the user turn")
	assert.Equal(t, "user turn", messages[1].Content)
	assert.Equal(t, types.RoleAssistant, messages[2].Role, "Last message should be a summary")
	assert.True(t, messages[2].Metadata["summarized"].(bool), "Last message should be marked summarized")
}

// TestThresholdStrategy_Summarize_EmptyConversation verifies no panic on empty input.
func TestThresholdStrategy_Summarize_EmptyConversation(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80, 5)
	conv := memory.NewConversationMemory()

	mockLLM := new(MockLLMProvider)

	saved, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)
	assert.Equal(t, 0, saved)
	mockLLM.AssertNotCalled(t, "Complete")
}
