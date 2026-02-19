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
type ThresholdSummarizationStrategy struct {
	// thresholdPercent is the percentage (0-100) of max tokens that triggers summarization
	thresholdPercent float64

	// messagesPerSummary is how many messages to summarize in each batch
	messagesPerSummary int
}

// NewThresholdSummarizationStrategy creates a new threshold-based strategy.
// thresholdPercent should be between 0 and 100 (e.g., 80 for 80% of max tokens).
// messagesPerSummary controls how many messages to summarize in each batch.
func NewThresholdSummarizationStrategy(thresholdPercent float64, messagesPerSummary int) *ThresholdSummarizationStrategy {
	// Clamp threshold to valid range
	if thresholdPercent < 0 {
		thresholdPercent = 0
	}
	if thresholdPercent > 100 {
		thresholdPercent = 100
	}

	// Ensure we summarize at least 1 message
	if messagesPerSummary < 1 {
		messagesPerSummary = 1
	}

	return &ThresholdSummarizationStrategy{
		thresholdPercent:   thresholdPercent,
		messagesPerSummary: messagesPerSummary,
	}
}

// Name returns the strategy name
func (s *ThresholdSummarizationStrategy) Name() string {
	return "ThresholdSummarization"
}

// ShouldRun returns true when current token usage exceeds the threshold
func (s *ThresholdSummarizationStrategy) ShouldRun(conv *memory.ConversationMemory, currentTokens, maxTokens int) bool {
	if maxTokens <= 0 {
		return false
	}

	// Calculate current usage percentage
	usagePercent := (float64(currentTokens) / float64(maxTokens)) * 100

	// Trigger if we've exceeded the threshold
	return usagePercent >= s.thresholdPercent
}

// Summarize creates summaries for old messages to free up context space.
// It processes one contiguous block of assistant messages per iteration,
// stopping at user-message boundaries so that each block is summarized
// independently and inserted at the correct position in the conversation.
func (s *ThresholdSummarizationStrategy) Summarize(ctx context.Context, conv *memory.ConversationMemory, llm llm.Provider) (int, error) {
	total := 0

	for {
		messages := conv.GetAll()
		if len(messages) == 0 {
			break
		}

		// Collect the next summarizable block (stops at user-message boundaries).
		toSummarize := s.collectMessagesToSummarize(messages)
		if len(toSummarize) == 0 {
			break
		}

		// Generate a summary for this block.
		summary, err := s.generateSummary(ctx, toSummarize, llm)
		if err != nil {
			return total, err
		}

		// Replace this block's messages with the summary, preserving everything else.
		if err := s.replaceMessagesWithSummary(conv, messages, toSummarize, summary); err != nil {
			return total, err
		}

		total += len(toSummarize)
	}

	return total, nil
}

// collectMessagesToSummarize returns a single contiguous block of assistant
// messages to summarize, working from oldest to newest.
//
// User messages act as block boundaries: once we have started collecting
// assistant messages, we stop at the next user message so that the summary
// is placed correctly relative to the human turn that follows it.
// User messages that appear before any assistant messages are skipped so we
// can reach the first actual block.
func (s *ThresholdSummarizationStrategy) collectMessagesToSummarize(messages []*types.Message) []*types.Message {
	var toSummarize []*types.Message
	startIdx := 0

	// Skip system message if present.
	if len(messages) > 0 && messages[0].Role == types.RoleSystem {
		startIdx = 1
	}

	for i := startIdx; i < len(messages) && len(toSummarize) < s.messagesPerSummary; i++ {
		msg := messages[i]

		switch {
		case msg.Role == types.RoleUser:
			// A user message is a hard block boundary.
			// If we have already collected assistant messages, stop here so the
			// summary is inserted before this user turn, not after it.
			if len(toSummarize) > 0 {
				return toSummarize
			}
			// Nothing collected yet — skip this boundary and keep looking for
			// the first assistant block.

		case s.isSummarized(msg):
			// Already summarized — skip but do NOT treat as a boundary, so we
			// continue collecting from the same block.

		case msg.Role == types.RoleAssistant:
			toSummarize = append(toSummarize, msg)
		}
	}

	return toSummarize
}

// isSummarized checks if a message has already been summarized
func (s *ThresholdSummarizationStrategy) isSummarized(msg *types.Message) bool {
	if msg.Metadata == nil {
		return false
	}
	summarized, ok := msg.Metadata["summarized"].(bool)
	return ok && summarized
}

// generateSummary calls the LLM to create a summary of the given messages
func (s *ThresholdSummarizationStrategy) generateSummary(ctx context.Context, toSummarize []*types.Message, llm llm.Provider) (*types.Message, error) {
	// Build prompt for summarization
	prompt := s.buildSummarizationPrompt(toSummarize)

	// Create messages for LLM
	llmMessages := []*types.Message{
		types.NewSystemMessage(
			"You are a technical memory encoder for an AI coding agent. " +
				"Your summaries replace a section of that agent's conversation history. " +
				"The agent must be able to continue its work seamlessly by reading your summary alone — " +
				"write for an AI consumer, not a human reader. Be dense, specific, and technical.",
		),
		types.NewUserMessage(prompt),
	}

	// Call LLM to generate summary
	response, err := llm.Complete(ctx, llmMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	// Create a single summarized message to replace the batch
	summary := types.NewAssistantMessage(response.Content).
		WithMetadata("summarized", true).
		WithMetadata("summary_count", len(toSummarize)).
		WithMetadata("summary_method", s.Name())

	return summary, nil
}

// replaceMessagesWithSummary removes the specifically-summarized assistant messages
// and inserts the summary at the position of the first one. All other messages
// (including user messages interleaved between assistant messages) are preserved.
func (s *ThresholdSummarizationStrategy) replaceMessagesWithSummary(conv *memory.ConversationMemory, messages []*types.Message, toSummarize []*types.Message, summary *types.Message) error {
	if len(toSummarize) == 0 {
		return nil
	}

	// Use pointer identity for membership checks — this is idiomatic Go and
	// avoids false matches when messages share timestamps (e.g. in fast tests).
	summarizedSet := make(map[*types.Message]bool, len(toSummarize))
	for _, msg := range toSummarize {
		summarizedSet[msg] = true
	}

	// Walk all messages: skip summarized ones, insert the summary once at the
	// position of the first summarized message, keep everything else intact.
	newMessages := make([]*types.Message, 0, len(messages)-len(toSummarize)+1)
	summaryInserted := false
	for _, msg := range messages {
		if summarizedSet[msg] {
			if !summaryInserted {
				newMessages = append(newMessages, summary)
				summaryInserted = true
			}
			// Drop this message — its content is captured in the summary.
			continue
		}
		newMessages = append(newMessages, msg)
	}

	if !summaryInserted {
		return fmt.Errorf("failed to find messages to remove")
	}

	// Clear and re-add all messages.
	conv.Clear()
	conv.AddMultiple(newMessages)

	return nil
}

// buildSummarizationPrompt creates a structured prompt for summarizing a block of
// agent messages. The output is consumed by another LLM agent (not a human), so
// the instructions optimize for density, specificity, and decision traceability
// over readability or narrative flow.
func (s *ThresholdSummarizationStrategy) buildSummarizationPrompt(messages []*types.Message) string {
	var b strings.Builder

	b.WriteString("Summarize the following agent conversation block into exactly six sections.\n")
	b.WriteString("Your output will be injected directly into an AI coding agent's context window.\n\n")

	b.WriteString("## Milestones\n")
	b.WriteString("Work that was fully completed: files created or edited, tests passed, commands run successfully, features shipped.\n\n")

	b.WriteString("## Key Decisions\n")
	b.WriteString("What was chosen and WHY — not just 'used X' but 'used X because Y'. Include algorithm choices, architecture choices, and any trade-offs accepted.\n\n")

	b.WriteString("## Findings\n")
	b.WriteString("Important discoveries: bugs identified, constraints uncovered, API behaviors confirmed, patterns observed.\n\n")

	b.WriteString("## Attempted & Abandoned\n")
	b.WriteString("Approaches that were tried and discarded, and the exact reason each was abandoned. This section prevents the agent from re-investigating known dead ends.\n\n")

	b.WriteString("## Current State\n")
	b.WriteString("Precise description of where things stand right now: what is in progress, what is partially complete, what is broken.\n\n")

	b.WriteString("## Open Items\n")
	b.WriteString("Unresolved questions, active blockers, and confirmed next steps.\n\n")

	b.WriteString("---\n\n")
	b.WriteString("MUST PRESERVE (never omit or paraphrase): file paths, function names, variable names, error messages, test names.\n")
	b.WriteString("MUST NOT INCLUDE: conversational filler, re-statements of the obvious, hedging language (\"it seems\", \"possibly\", \"I think\"), offers of help, apologies.\n\n")

	b.WriteString("Messages to summarize:\n\n")
	for i, msg := range messages {
		b.WriteString(fmt.Sprintf("%d. %s: %s\n\n", i+1, msg.Role, msg.Content))
	}

	return b.String()
}
