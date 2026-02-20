package context

import (
	"context"
	"testing"

	"github.com/entrhq/forge/pkg/agent/memory"
	"github.com/entrhq/forge/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// helpers

func newSummarizedMsg(content string) *types.Message {
	return types.NewAssistantMessage(content).WithMetadata("summarized", true)
}

func newGoalBatchMsg(content string) *types.Message {
	m := types.NewAssistantMessage(content)
	m.WithMetadata("summarized", true)
	m.WithMetadata("summary_type", "goal_batch")
	return m
}

// --- constructor ---

func TestNewGoalBatchCompactionStrategy_Defaults(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(0, 0, 0)
	assert.Equal(t, 20, s.minMessagesOldThreshold)
	assert.Equal(t, 3, s.minTurnsToCompact)
	assert.Equal(t, 6, s.maxTurnsPerBatch)
}

func TestNewGoalBatchCompactionStrategy_CustomValues(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(10, 2, 4)
	assert.Equal(t, 10, s.minMessagesOldThreshold)
	assert.Equal(t, 2, s.minTurnsToCompact)
	assert.Equal(t, 4, s.maxTurnsPerBatch)
}

func TestGoalBatchCompactionStrategy_Name(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(10, 2, 4)
	assert.Equal(t, "GoalBatchCompaction", s.Name())
}

// --- isGoalBatch / isRegularSummary ---

func TestIsGoalBatch(t *testing.T) {
	assert.True(t, isGoalBatch(newGoalBatchMsg("x")))
	assert.False(t, isGoalBatch(newSummarizedMsg("x")))
	assert.False(t, isGoalBatch(types.NewUserMessage("x")))
}

func TestIsRegularSummary(t *testing.T) {
	assert.True(t, isRegularSummary(newSummarizedMsg("x")))
	assert.False(t, isRegularSummary(newGoalBatchMsg("x")))
	assert.False(t, isRegularSummary(types.NewUserMessage("x")))
}

// --- collectCompleteTurns ---

func TestCollectCompleteTurns_Basic(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(5, 2, 6)
	messages := []*types.Message{
		types.NewUserMessage("goal 1"),
		newSummarizedMsg("[SUMMARIZED] did goal 1"),
		types.NewUserMessage("goal 2"),
		newSummarizedMsg("[SUMMARIZED] did goal 2"),
	}

	turns := s.collectCompleteTurns(messages)
	assert.Len(t, turns, 2)
	assert.Equal(t, "goal 1", turns[0].userMessage.Content)
	assert.Len(t, turns[0].summaryMessages, 1)
	assert.Equal(t, "goal 2", turns[1].userMessage.Content)
}

func TestCollectCompleteTurns_SkipsIncompleteTurn(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(5, 2, 6)
	// Last user message has no summary yet — incomplete.
	messages := []*types.Message{
		types.NewUserMessage("goal 1"),
		newSummarizedMsg("[SUMMARIZED] did goal 1"),
		types.NewUserMessage("in progress"),
	}

	turns := s.collectCompleteTurns(messages)
	assert.Len(t, turns, 1, "incomplete turn must not be collected")
}

func TestCollectCompleteTurns_SkipsGoalBatchBlocks(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(5, 2, 6)
	// Already-compacted [GOAL BATCH] blocks must be transparent.
	messages := []*types.Message{
		newGoalBatchMsg("[GOAL BATCH] old arc"),
		types.NewUserMessage("goal 2"),
		newSummarizedMsg("[SUMMARIZED] did goal 2"),
	}

	turns := s.collectCompleteTurns(messages)
	assert.Len(t, turns, 1)
	assert.Equal(t, "goal 2", turns[0].userMessage.Content)
}

func TestCollectCompleteTurns_MultipleSummariesPerTurn(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(5, 2, 6)
	messages := []*types.Message{
		types.NewUserMessage("goal 1"),
		newSummarizedMsg("[SUMMARIZED] part a"),
		newSummarizedMsg("[SUMMARIZED] part b"),
		types.NewUserMessage("goal 2"),
		newSummarizedMsg("[SUMMARIZED] did goal 2"),
	}

	turns := s.collectCompleteTurns(messages)
	assert.Len(t, turns, 2)
	assert.Len(t, turns[0].summaryMessages, 2, "both summaries belong to turn 1")
}

func TestCollectCompleteTurns_SkipsSystemMessages(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(5, 2, 6)
	messages := []*types.Message{
		types.NewSystemMessage("system prompt"),
		types.NewUserMessage("goal 1"),
		newSummarizedMsg("[SUMMARIZED] did goal 1"),
	}

	turns := s.collectCompleteTurns(messages)
	assert.Len(t, turns, 1)
	assert.Equal(t, "goal 1", turns[0].userMessage.Content)
}

// --- ShouldRun ---

func TestGoalBatchCompactionStrategy_ShouldRun_NotEnoughMessages(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(10, 2, 6)
	conv := memory.NewConversationMemory()
	conv.Add(types.NewUserMessage("goal 1"))
	conv.Add(newSummarizedMsg("[SUMMARIZED] did goal 1"))

	assert.False(t, s.ShouldRun(conv, 0, 0))
}

func TestGoalBatchCompactionStrategy_ShouldRun_BelowMinTurns(t *testing.T) {
	// minTurnsToCompact=3 but only 2 eligible turns
	s := NewGoalBatchCompactionStrategy(5, 3, 6)
	conv := memory.NewConversationMemory()

	conv.Add(types.NewUserMessage("goal 1"))
	conv.Add(newSummarizedMsg("[SUMMARIZED] did goal 1"))
	conv.Add(types.NewUserMessage("goal 2"))
	conv.Add(newSummarizedMsg("[SUMMARIZED] did goal 2"))

	// Push into eligible window
	for i := 0; i < 6; i++ {
		conv.Add(types.NewAssistantMessage("recent"))
	}

	assert.False(t, s.ShouldRun(conv, 0, 0))
}

func TestGoalBatchCompactionStrategy_ShouldRun_MeetsThreshold(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(5, 3, 6)
	conv := memory.NewConversationMemory()

	for i := 0; i < 3; i++ {
		conv.Add(types.NewUserMessage("goal"))
		conv.Add(newSummarizedMsg("[SUMMARIZED] done"))
	}
	for i := 0; i < 6; i++ {
		conv.Add(types.NewAssistantMessage("recent"))
	}

	assert.True(t, s.ShouldRun(conv, 0, 0))
}

// --- Summarize ---

func TestGoalBatchCompactionStrategy_Summarize_ReplacesWithGoalBatch(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(5, 2, 6)
	conv := memory.NewConversationMemory()

	conv.Add(types.NewUserMessage("refactor config loader"))
	conv.Add(newSummarizedMsg("[SUMMARIZED] I read loader.go, extracted helpers"))
	conv.Add(types.NewUserMessage("add yaml support"))
	conv.Add(newSummarizedMsg("[SUMMARIZED] I added yaml path, resolved conflict"))
	conv.Add(types.NewUserMessage("add validation"))
	conv.Add(newSummarizedMsg("[SUMMARIZED] I added ValidateConfig, tests pass"))

	// Recent messages to push old content into eligible window
	for i := 0; i < 6; i++ {
		conv.Add(types.NewAssistantMessage("recent"))
	}

	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).Return(
		types.NewAssistantMessage("## Goal Arc\nI refactored the config loader with yaml and validation."),
		nil,
	)

	count, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)
	assert.Equal(t, 3, count, "should report 3 turns compacted")

	// Verify a [GOAL BATCH] block now exists
	messages := conv.GetAll()
	foundGoalBatch := false
	for _, msg := range messages {
		if isGoalBatch(msg) {
			foundGoalBatch = true
			assert.Contains(t, msg.Content, "[GOAL BATCH]")
		}
	}
	assert.True(t, foundGoalBatch, "conversation must contain a [GOAL BATCH] block")

	// Original turn messages should be gone
	for _, msg := range messages {
		assert.NotEqual(t, "refactor config loader", msg.Content)
		assert.NotEqual(t, "add yaml support", msg.Content)
	}
}

func TestGoalBatchCompactionStrategy_Summarize_PreservesRecentMessages(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(5, 2, 6)
	conv := memory.NewConversationMemory()

	conv.Add(types.NewUserMessage("old goal 1"))
	conv.Add(newSummarizedMsg("[SUMMARIZED] done 1"))
	conv.Add(types.NewUserMessage("old goal 2"))
	conv.Add(newSummarizedMsg("[SUMMARIZED] done 2"))

	recentContent := "this is recent work"
	for i := 0; i < 6; i++ {
		conv.Add(types.NewAssistantMessage(recentContent))
	}

	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).Return(
		types.NewAssistantMessage("## Goal Arc\nCompleted two goals."),
		nil,
	)

	_, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)

	messages := conv.GetAll()
	recentFound := 0
	for _, msg := range messages {
		if msg.Content == recentContent {
			recentFound++
		}
	}
	assert.Equal(t, 6, recentFound, "all recent messages must be preserved")
}

func TestGoalBatchCompactionStrategy_Summarize_RespectsMaxTurnsPerBatch(t *testing.T) {
	// maxTurnsPerBatch=2 with 4 eligible turns — only 2 should be compacted per run
	s := NewGoalBatchCompactionStrategy(5, 2, 2)
	conv := memory.NewConversationMemory()

	for i := 0; i < 4; i++ {
		conv.Add(types.NewUserMessage("goal"))
		conv.Add(newSummarizedMsg("[SUMMARIZED] done"))
	}
	for i := 0; i < 6; i++ {
		conv.Add(types.NewAssistantMessage("recent"))
	}

	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).Return(
		types.NewAssistantMessage("## Goal Arc\nTwo goals."),
		nil,
	)

	count, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)
	assert.Equal(t, 2, count, "only maxTurnsPerBatch turns should be compacted per run")
}

func TestGoalBatchCompactionStrategy_Summarize_PreservesGoalBatchBlocks(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(5, 2, 6)
	conv := memory.NewConversationMemory()

	// An already-compacted [GOAL BATCH] block should survive untouched.
	conv.Add(newGoalBatchMsg("[GOAL BATCH] prior arc"))
	conv.Add(types.NewUserMessage("new goal 1"))
	conv.Add(newSummarizedMsg("[SUMMARIZED] done 1"))
	conv.Add(types.NewUserMessage("new goal 2"))
	conv.Add(newSummarizedMsg("[SUMMARIZED] done 2"))

	for i := 0; i < 6; i++ {
		conv.Add(types.NewAssistantMessage("recent"))
	}

	mockLLM := new(MockLLMProvider)
	mockLLM.On("Complete", mock.Anything, mock.Anything).Return(
		types.NewAssistantMessage("## Goal Arc\nTwo new goals."),
		nil,
	)

	_, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)

	messages := conv.GetAll()
	foundPriorBatch := false
	for _, msg := range messages {
		if isGoalBatch(msg) && msg.Content == "[GOAL BATCH] prior arc" {
			foundPriorBatch = true
		}
	}
	assert.True(t, foundPriorBatch, "prior [GOAL BATCH] block must be preserved")
}

func TestGoalBatchCompactionStrategy_Summarize_EmptyConversation(t *testing.T) {
	s := NewGoalBatchCompactionStrategy(5, 2, 6)
	conv := memory.NewConversationMemory()
	mockLLM := new(MockLLMProvider)

	count, err := s.Summarize(context.Background(), conv, mockLLM)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	mockLLM.AssertNotCalled(t, "Complete")
}
