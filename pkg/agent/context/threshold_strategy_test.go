package context

import (
	"context"
	"testing"

	"github.com/entrhq/forge/pkg/agent/memory"
	"github.com/entrhq/forge/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestNewThresholdSummarizationStrategy tests constructor clamping.
func TestNewThresholdSummarizationStrategy(t *testing.T) {
	tests := []struct {
		name                 string
		thresholdPercent     float64
		wantThresholdPercent float64
	}{
		{
			name:                 "valid input",
			thresholdPercent:     80,
			wantThresholdPercent: 80,
		},
		{
			name:                 "threshold clamped below 0",
			thresholdPercent:     -10,
			wantThresholdPercent: 0,
		},
		{
			name:                 "threshold clamped above 100",
			thresholdPercent:     150,
			wantThresholdPercent: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewThresholdSummarizationStrategy(tt.thresholdPercent)
			assert.Equal(t, tt.wantThresholdPercent, s.thresholdPercent)
		})
	}
}

// TestThresholdStrategy_Name tests the Name method.
func TestThresholdStrategy_Name(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80)
	assert.Equal(t, "ThresholdSummarization", s.Name())
}

// TestThresholdStrategy_ShouldRun tests trigger conditions.
func TestThresholdStrategy_ShouldRun(t *testing.T) {
	makeConv := func(nonSystemMsgCount int) *memory.ConversationMemory {
		conv := memory.NewConversationMemory()
		for i := 0; i < nonSystemMsgCount; i++ {
			if i%2 == 0 {
				conv.Add(types.NewUserMessage("user message"))
			} else {
				conv.Add(types.NewAssistantMessage("assistant message"))
			}
		}
		return conv
	}

	tests := []struct {
		name          string
		threshold     float64
		currentTokens int
		maxTokens     int
		nonSystemMsgs int
		want          bool
	}{
		{"above threshold, enough messages", 80, 850, 1000, 6, true},
		{"at threshold, enough messages", 80, 800, 1000, 6, true},
		{"below threshold", 80, 750, 1000, 6, false},
		{"zero max tokens", 80, 800, 0, 6, false},
		{"negative max tokens", 80, 800, -1, 6, false},
		{"above threshold, too few messages", 80, 850, 1000, 3, false},
		{"above threshold, exactly min messages", 80, 850, 1000, 4, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewThresholdSummarizationStrategy(tt.threshold)
			conv := makeConv(tt.nonSystemMsgs)
			got := s.ShouldRun(conv, tt.currentTokens, tt.maxTokens)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestThresholdStrategy_ShouldRun_SystemMessagesExcluded verifies that system messages
// are not counted towards the minMessages threshold.
func TestThresholdStrategy_ShouldRun_SystemMessagesExcluded(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80)
	conv := memory.NewConversationMemory()

	// Add many system messages but only 3 non-system messages (below minMessages=4).
	for i := 0; i < 10; i++ {
		conv.Add(types.NewSystemMessage("system"))
	}
	conv.Add(types.NewUserMessage("user"))
	conv.Add(types.NewAssistantMessage("assistant"))
	conv.Add(types.NewUserMessage("user"))

	// Token threshold exceeded.
	got := s.ShouldRun(conv, 900, 1000)
	assert.False(t, got, "should not trigger with only 3 non-system messages")
}

// TestThresholdStrategy_Summarize_HalfAndHalf verifies the core half-compaction invariant:
//
//   - System messages are always kept verbatim.
//   - The older half of non-system messages is collapsed into exactly one summary.
//   - The more recent half is preserved verbatim and in order.
func TestThresholdStrategy_Summarize_HalfAndHalf(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80)
	conv := memory.NewConversationMemory()

	// 6 non-system messages: older half = [0,1,2], recent half = [3,4,5].
	msgs := []*types.Message{
		types.NewUserMessage("turn 1 user"),
		types.NewAssistantMessage("turn 1 assistant"),
		types.NewUserMessage("turn 2 user"),
		types.NewAssistantMessage("turn 2 assistant"),
		types.NewUserMessage("turn 3 user"),
		types.NewAssistantMessage("turn 3 assistant"),
	}
	for _, m := range msgs {
		conv.Add(m)
	}

	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).Return(
		types.NewAssistantMessage("Half summary"),
		nil,
	)

	count, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)
	assert.Equal(t, 3, count, "should have summarized the 3 older messages")

	// Expect exactly one LLM call covering the entire older half.
	mockLLM.AssertNumberOfCalls(t, "Complete", 1)

	result := conv.GetAll()

	// Should be: [summary] [recent3] [recent4] [recent5]
	assert.Len(t, result, 4, "expected [summary, turn3user, turn3assistant, turn3user... wait: recent half is msgs[3..5]]")

	// First message is the summary.
	assert.Equal(t, types.RoleAssistant, result[0].Role)
	assert.True(t, result[0].Metadata["summarized"].(bool), "first message must be marked summarized")

	// The recent half is verbatim, in order.
	assert.Equal(t, msgs[3], result[1], "recent[0] should be turn 2 assistant verbatim")
	assert.Equal(t, msgs[4], result[2], "recent[1] should be turn 3 user verbatim")
	assert.Equal(t, msgs[5], result[3], "recent[2] should be turn 3 assistant verbatim")
}

// TestThresholdStrategy_Summarize_PreservesSystemMessages verifies that system messages
// always survive summarization and appear before the summary in the output.
func TestThresholdStrategy_Summarize_PreservesSystemMessages(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80)
	conv := memory.NewConversationMemory()

	conv.Add(types.NewSystemMessage("You are an AI coding assistant."))
	conv.Add(types.NewUserMessage("msg1"))
	conv.Add(types.NewAssistantMessage("msg2"))
	conv.Add(types.NewUserMessage("msg3"))
	conv.Add(types.NewAssistantMessage("msg4"))

	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).Return(
		types.NewAssistantMessage("Summary"),
		nil,
	)

	_, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)

	result := conv.GetAll()

	// System message must come first.
	assert.Equal(t, types.RoleSystem, result[0].Role, "system message must be first")
	assert.Equal(t, "You are an AI coding assistant.", result[0].Content)

	// Summary must follow the system message.
	assert.True(t, result[1].Metadata["summarized"].(bool), "second message must be the summary")
}

// TestThresholdStrategy_Summarize_RecentHalfKeptVerbatim verifies that existing
// summaries inside the recent half are preserved as-is (not re-summarized).
func TestThresholdStrategy_Summarize_RecentHalfKeptVerbatim(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80)
	conv := memory.NewConversationMemory()

	// Make an existing summary that sits in what will become the recent half.
	existingSummary := types.NewAssistantMessage("[SUMMARIZED] prior summary").
		WithMetadata("summarized", true)

	conv.Add(types.NewUserMessage("old msg 1"))
	conv.Add(types.NewAssistantMessage("old msg 2"))
	conv.Add(types.NewUserMessage("recent msg 1")) // these three form the recent half
	conv.Add(existingSummary)
	conv.Add(types.NewUserMessage("recent msg 2"))
	conv.Add(types.NewAssistantMessage("recent msg 3"))

	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).Return(
		types.NewAssistantMessage("New half-summary"),
		nil,
	)

	_, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)

	result := conv.GetAll()

	// Exactly one LLM call — the older half is summarized in a single shot.
	mockLLM.AssertNumberOfCalls(t, "Complete", 1)

	// The existing summary in the recent half must be preserved verbatim.
	found := false
	for _, msg := range result {
		if msg == existingSummary {
			found = true
			break
		}
	}
	assert.True(t, found, "existing summary in the recent half must be kept verbatim")
}

// TestThresholdStrategy_Summarize_EmptyConversation verifies no panic on empty input.
func TestThresholdStrategy_Summarize_EmptyConversation(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80)
	conv := memory.NewConversationMemory()

	mockLLM := new(MockLLMProvider)

	count, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	mockLLM.AssertNotCalled(t, "Complete")
}

// TestThresholdStrategy_Summarize_TooFewMessages verifies that summarization is skipped
// when there are fewer than minMessages non-system messages.
func TestThresholdStrategy_Summarize_TooFewMessages(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80)
	conv := memory.NewConversationMemory()

	conv.Add(types.NewUserMessage("only message"))
	conv.Add(types.NewAssistantMessage("only reply"))
	conv.Add(types.NewUserMessage("third"))

	mockLLM := new(MockLLMProvider)

	count, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	mockLLM.AssertNotCalled(t, "Complete")
}

// TestThresholdStrategy_Summarize_OddMessageCount verifies the rounding behaviour:
// for an odd number of non-system messages the recent half is larger (we round down
// the split point so that more recent context is preserved).
func TestThresholdStrategy_Summarize_OddMessageCount(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80)
	conv := memory.NewConversationMemory()

	// 5 non-system messages → split at 2 (older), keep 3 (recent).
	msgs := []*types.Message{
		types.NewUserMessage("m0"),
		types.NewAssistantMessage("m1"),
		types.NewUserMessage("m2"),
		types.NewAssistantMessage("m3"),
		types.NewUserMessage("m4"),
	}
	for _, m := range msgs {
		conv.Add(m)
	}

	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).Return(
		types.NewAssistantMessage("summary"),
		nil,
	)

	count, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)
	assert.Equal(t, 2, count, "older half should be 2 messages (floor of 5/2)")

	result := conv.GetAll()
	// [summary] + 3 recent messages.
	assert.Len(t, result, 4)

	// Verify recent half is msgs[2..4] in order.
	assert.Equal(t, msgs[2], result[1])
	assert.Equal(t, msgs[3], result[2])
	assert.Equal(t, msgs[4], result[3])
}

// TestThresholdStrategy_NeverGetsStuck verifies the root cause of the original bug is fixed.
// After one run all conversation messages are already summarized or preserved verbatim —
// so a subsequent ShouldRun call with a sufficiently large conversation still works, and
// Summarize again reduces the message count (it does not get stuck).
func TestThresholdStrategy_NeverGetsStuck(t *testing.T) {
	s := NewThresholdSummarizationStrategy(80)
	conv := memory.NewConversationMemory()

	// Simulate a conversation where all older messages are already [SUMMARIZED].
	conv.Add(types.NewAssistantMessage("[SUMMARIZED] old summary 1").WithMetadata("summarized", true))
	conv.Add(types.NewUserMessage("user turn"))
	conv.Add(types.NewAssistantMessage("[SUMMARIZED] old summary 2").WithMetadata("summarized", true))
	conv.Add(types.NewUserMessage("another turn"))
	conv.Add(types.NewAssistantMessage("fresh response"))
	conv.Add(types.NewUserMessage("last user"))

	beforeCount := conv.Count()

	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).Return(
		types.NewAssistantMessage("consolidated summary"),
		nil,
	)

	count, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)
	assert.Greater(t, count, 0, "should have summarized some messages")
	assert.Less(t, conv.Count(), beforeCount, "conversation should be shorter after summarization")

	// Only one LLM call regardless of how many pre-existing summaries existed.
	mockLLM.AssertNumberOfCalls(t, "Complete", 1)
}
