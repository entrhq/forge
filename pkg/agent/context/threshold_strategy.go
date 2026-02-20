package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/agent/memory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/types"
)

// ThresholdSummarizationStrategy triggers summarization when token usage
// exceeds a configured percentage of the maximum context window.
//
// When triggered it collapses the older half of the conversation (excluding
// system messages) into a single comprehensive summary, then keeps the more
// recent half verbatim. This guarantees meaningful token reduction on every
// run and avoids the "already-summarized loop" that plagued block-by-block
// approaches.
//
// Layout after a run:
//
//	[system messages (unchanged)] [summary of older half] [recent half verbatim]
type ThresholdSummarizationStrategy struct {
	// thresholdPercent is the percentage (0-100) of max tokens that triggers summarization.
	thresholdPercent float64

	// minMessages is the minimum number of non-system messages required before
	// summarization fires. Prevents summarizing trivially short conversations where
	// a meaningful split is impossible.
	minMessages int
}

// NewThresholdSummarizationStrategy creates a new threshold-based half-compaction strategy.
// thresholdPercent should be between 0 and 100 (e.g., 80 for 80% of max tokens).
func NewThresholdSummarizationStrategy(thresholdPercent float64) *ThresholdSummarizationStrategy {
	// Clamp threshold to valid range.
	if thresholdPercent < 0 {
		thresholdPercent = 0
	}
	if thresholdPercent > 100 {
		thresholdPercent = 100
	}

	return &ThresholdSummarizationStrategy{
		thresholdPercent: thresholdPercent,
		minMessages:      4, // need at least 4 non-system messages for a meaningful split
	}
}

// Name returns the strategy name.
func (s *ThresholdSummarizationStrategy) Name() string {
	return "ThresholdSummarization"
}

// ShouldRun returns true when current token usage exceeds the threshold AND
// the conversation has enough non-system messages to perform a meaningful split.
func (s *ThresholdSummarizationStrategy) ShouldRun(conv *memory.ConversationMemory, currentTokens, maxTokens int) bool {
	if maxTokens <= 0 {
		return false
	}

	// Check token threshold.
	usagePercent := (float64(currentTokens) / float64(maxTokens)) * 100
	if usagePercent < s.thresholdPercent {
		return false
	}

	// Require enough non-system messages for a meaningful half-split.
	messages := conv.GetAll()
	nonSystemCount := 0
	for _, msg := range messages {
		if msg.Role != types.RoleSystem {
			nonSystemCount++
		}
	}
	return nonSystemCount >= s.minMessages
}

// Summarize collapses the older half of the conversation into a single summary,
// leaving the more recent half intact. System messages are always preserved.
// Returns the number of messages replaced by the summary.
func (s *ThresholdSummarizationStrategy) Summarize(ctx context.Context, conv *memory.ConversationMemory, provider llm.Provider) (int, error) {
	messages := conv.GetAll()

	// Partition into system vs. conversation messages preserving original order.
	var systemMessages []*types.Message
	var conversationMessages []*types.Message
	for _, msg := range messages {
		if msg.Role == types.RoleSystem {
			systemMessages = append(systemMessages, msg)
		} else {
			conversationMessages = append(conversationMessages, msg)
		}
	}

	if len(conversationMessages) < s.minMessages {
		return 0, nil
	}

	// Split: older half gets summarized, recent half stays verbatim.
	// For odd counts we round down so the recent half is slightly larger —
	// that is, we prefer to preserve more recent context.
	splitAt := len(conversationMessages) / 2
	olderHalf := conversationMessages[:splitAt]
	recentHalf := conversationMessages[splitAt:]

	// Single LLM call covers the entire older half.
	summary, err := s.generateSummary(ctx, olderHalf, provider)
	if err != nil {
		return 0, err
	}

	// Reassemble: system → summary → recent half.
	newMessages := make([]*types.Message, 0, len(systemMessages)+1+len(recentHalf))
	newMessages = append(newMessages, systemMessages...)
	newMessages = append(newMessages, summary)
	newMessages = append(newMessages, recentHalf...)

	conv.Clear()
	conv.AddMultiple(newMessages)

	return len(olderHalf), nil
}

// generateSummary calls the LLM to produce a single compact summary of messages.
func (s *ThresholdSummarizationStrategy) generateSummary(ctx context.Context, toSummarize []*types.Message, provider llm.Provider) (*types.Message, error) {
	prompt := s.buildSummarizationPrompt(toSummarize)

	llmMessages := []*types.Message{
		types.NewSystemMessage(episodicMemorySystemPrompt),
		types.NewUserMessage(prompt),
	}

	response, err := provider.Complete(ctx, llmMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	// The [SUMMARIZED] prefix is an explicit epistemic marker so the agent
	// knows this content is compressed recalled experience, not raw history.
	summary := types.NewAssistantMessage(fmt.Sprintf("[SUMMARIZED] %s", response.Content)).
		WithMetadata("summarized", true).
		WithMetadata("summary_count", len(toSummarize)).
		WithMetadata("summary_method", s.Name())

	return summary, nil
}

// buildSummarizationPrompt constructs a structured prompt for summarizing a block of
// messages from any role. The output is consumed by another LLM agent, so the
// instructions optimize for density, specificity, and decision traceability.
func (s *ThresholdSummarizationStrategy) buildSummarizationPrompt(messages []*types.Message) string {
	var b strings.Builder

	b.WriteString("The following messages are the earlier portion of your current conversation. " +
		"They include both user instructions and your own responses. " +
		"Write a first-person summary as if you are the agent recalling your own actions. " +
		"Use 'I' throughout — this is your episodic memory, not a report about someone else. " +
		"Use exactly six sections:\n\n")

	b.WriteString("## Milestones\n")
	b.WriteString("Work I fully completed: files I created or edited, tests I passed, commands I ran successfully, features I shipped.\n\n")

	b.WriteString("## Key Decisions\n")
	b.WriteString("What I chose and WHY — not just 'I used X' but 'I used X because Y'. Include algorithm choices, architecture choices, and trade-offs I accepted.\n\n")

	b.WriteString("## Findings\n")
	b.WriteString("Important discoveries I made: bugs I identified, constraints I uncovered, API behaviors I confirmed, patterns I observed.\n\n")

	b.WriteString("## Dead Ends\n")
	b.WriteString("Approaches I tried and abandoned. Write as personal lessons — 'I tried X — abandoned because Y' — so I will not re-attempt these paths.\n\n")

	b.WriteString("## Current State\n")
	b.WriteString("Precise description of where things stand: what I have in progress, what is partially complete, what is broken.\n\n")

	b.WriteString("## Open Items\n")
	b.WriteString("Unresolved questions, active blockers, and my confirmed next steps.\n\n")

	b.WriteString("---\n\n")
	b.WriteString("MUST PRESERVE (never omit or paraphrase): file paths, function names, variable names, error messages, test names, user instructions.\n")
	b.WriteString("MUST NOT INCLUDE: conversational filler, re-statements of the obvious, hedging language, offers of help, apologies.\n")
	b.WriteString("MUST USE: first-person voice throughout ('I tried', 'I found', 'I abandoned') — never 'the agent'.\n")
	b.WriteString("FOR LONG TOOL OUTPUTS: abridge intelligently — extract the essential signal (key errors, relevant values, test names) rather than quoting verbatim. The goal is density, not completeness.\n\n")

	b.WriteString("Messages to summarize:\n\n")
	for i, msg := range messages {
		b.WriteString(fmt.Sprintf("%d. [%s]: %s\n\n", i+1, msg.Role, msg.Content))
	}

	return b.String()
}
