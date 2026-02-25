package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/agent/memory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/types"
)

// ToolCallSummarizationStrategy summarizes old tool calls and their results
// to reduce context size while preserving semantic meaning.
// It uses a buffering mechanism with dual trigger conditions to reduce LLM API calls.
type ToolCallSummarizationStrategy struct {
	// messagesOldThreshold is how many messages back to start considering tool calls for the buffer.
	// For example, 20 means only tool calls that are 20+ messages old enter the buffer.
	messagesOldThreshold int

	// minToolCallsToSummarize is the minimum number of tool calls in the buffer before triggering summarization.
	// This creates batching to reduce LLM API calls.
	minToolCallsToSummarize int

	// maxToolCallDistance is the maximum age (in messages) a tool call can be before forcing summarization.
	// If any tool call exceeds this distance, all buffered tool calls are summarized regardless of buffer size.
	maxToolCallDistance int

	// eventChannel is used to emit progress events during parallel summarization
	eventChannel chan<- *types.AgentEvent
}

// NewToolCallSummarizationStrategy creates a new tool call summarization strategy with buffering.
// Parameters:
//   - messagesOldThreshold: Tool calls must be at least this many messages old to enter buffer (default: 20)
//   - minToolCallsToSummarize: Minimum buffer size before triggering summarization (default: 10)
//   - maxToolCallDistance: Maximum age before forcing summarization regardless of buffer size (default: 40)
func NewToolCallSummarizationStrategy(messagesOldThreshold, minToolCallsToSummarize, maxToolCallDistance int) *ToolCallSummarizationStrategy {
	if messagesOldThreshold <= 0 {
		messagesOldThreshold = 20
	}
	if minToolCallsToSummarize <= 0 {
		minToolCallsToSummarize = 10
	}
	if maxToolCallDistance <= 0 {
		maxToolCallDistance = 40
	}

	return &ToolCallSummarizationStrategy{
		messagesOldThreshold:    messagesOldThreshold,
		minToolCallsToSummarize: minToolCallsToSummarize,
		maxToolCallDistance:     maxToolCallDistance,
		eventChannel:            nil, // Will be set by Manager
	}
}

// SetEventChannel sets the event channel for emitting progress events during summarization.
func (s *ToolCallSummarizationStrategy) SetEventChannel(eventChan chan<- *types.AgentEvent) {
	s.eventChannel = eventChan
}

// Name returns the strategy's identifier.
func (s *ToolCallSummarizationStrategy) Name() string {
	return "ToolCallSummarization"
}

// ShouldRun checks if buffered tool calls meet trigger conditions for summarization.
// Returns true if either:
// 1. Buffer trigger: Buffer contains >= minToolCallsToSummarize tool calls
// 2. Age trigger: Any tool call is >= maxToolCallDistance messages old
func (s *ToolCallSummarizationStrategy) ShouldRun(conv *memory.ConversationMemory, currentTokens, maxTokens int) bool {
	messages := conv.GetAll()
	totalMessages := len(messages)

	if totalMessages <= s.messagesOldThreshold {
		return false // Not enough message history
	}

	// Identify old messages that can enter the buffer
	oldMessages := messages[:totalMessages-s.messagesOldThreshold]

	// Count unsummarized tool call pairs in the buffer and track the oldest qualifying position.
	bufferCount := 0
	oldestToolCallPosition := -1

	for i, msg := range oldMessages {
		// Skip if already summarized
		if isSummarized(msg) {
			continue
		}

		switch {
		case msg.Role == types.RoleAssistant && containsToolCallIndicators(msg.Content):
			bufferCount++
			if oldestToolCallPosition == -1 {
				oldestToolCallPosition = i
			}

		case msg.Role == types.RoleTool:
			bufferCount++
			if oldestToolCallPosition == -1 {
				oldestToolCallPosition = i
			}
			// Tool result messages are not counted; the paired assistant message
			// already represents this call+result pair in bufferCount.
		}
	}

	// No tool calls to summarize
	if bufferCount == 0 {
		return false
	}

	// Buffer trigger: Check if buffer size meets minimum threshold
	if bufferCount >= s.minToolCallsToSummarize {
		return true
	}

	// Age trigger: Check if oldest tool call exceeds maximum distance
	if oldestToolCallPosition >= 0 {
		// Calculate distance from current position
		distance := totalMessages - oldestToolCallPosition
		if distance >= s.maxToolCallDistance {
			return true
		}
	}

	return false
}

// Summarize compresses buffered tool calls and their results using LLM-based summarization.
// It replaces grouped tool call pairs with a single [SUMMARIZED] block inserted at the
// position of the first grouped message, preserving all other messages (system, user,
// existing [SUMMARIZED] blocks, recent messages) in their original order.
//
// This ensures that on repeated runs summaries accumulate correctly:
//
//	[user1] [SUMMARIZED_A] [user2] [SUMMARIZED_B] [user3] [live messages...]
//
// rather than collapsing all history into a single ever-rewritten summary block.
func (s *ToolCallSummarizationStrategy) Summarize(ctx context.Context, conv *memory.ConversationMemory, llm llm.Provider) (int, error) {
	messages := conv.GetAll()
	if len(messages) <= s.messagesOldThreshold {
		return 0, nil
	}

	// Only consider tool calls from the "old" portion of the conversation.
	oldMessages := messages[:len(messages)-s.messagesOldThreshold]

	// Group unsummarized tool call pairs eligible for compression.
	groups := groupToolCallsAndResults(oldMessages)
	if len(groups) == 0 {
		return 0, nil
	}

	// Single batched LLM call covers all groups.
	summaries, err := s.summarizeBatch(ctx, groups, llm, findNearestUserGoal(oldMessages))
	if err != nil {
		return 0, err
	}

	// Build a set of all message pointers belonging to the groups being replaced.
	groupSet := make(map[*types.Message]bool, len(groups)*2)
	for _, group := range groups {
		for _, msg := range group {
			groupSet[msg] = true
		}
	}

	// Walk the full conversation in order. At the position of the first grouped
	// message, insert the summary block(s). Drop all other grouped messages.
	// Everything else — system, user, existing [SUMMARIZED] blocks, recent
	// messages — passes through unchanged, preserving interleaved order.
	newMessages := make([]*types.Message, 0, len(messages)-len(groupSet)+len(summaries))
	inserted := false
	for _, msg := range messages {
		if groupSet[msg] {
			if !inserted {
				newMessages = append(newMessages, summaries...)
				inserted = true
			}
			continue // drop — content is captured in the summary
		}
		newMessages = append(newMessages, msg)
	}

	conv.Clear()
	conv.AddMultiple(newMessages)

	return len(groups), nil
}

// findNearestUserGoal scans messages in reverse and returns the content of the
// most recent user message. This anchors the batch summarizer to the user's
// original request so it can reason about strategy relative to what was asked.
func findNearestUserGoal(messages []*types.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == types.RoleUser {
			return messages[i].Content
		}
	}
	return ""
}

// summarizeBatch compresses all eligible tool call groups into a single LLM call,
// producing one structured operation-batch summary message. Batching rather than
// per-call summarization provides two benefits:
//  1. N→1 LLM API calls, regardless of how many tool groups are present.
//  2. The summarizer sees the full operation sequence and can infer the connecting
//     intent across calls — something per-call summarization cannot do.
//
// The output is consumed by another AI agent (not a human), so the prompt optimizes
// for density, exact artifact preservation, and cross-operation context.
// userGoal is the nearest preceding user message to the batch. When provided it
// anchors the summary to what was actually asked for, enabling strategy-aware
// summarization: what was tried, what failed, and what ultimately worked.
func (s *ToolCallSummarizationStrategy) summarizeBatch(ctx context.Context, groups [][]*types.Message, provider llm.Provider, userGoal string) ([]*types.Message, error) {
	if len(groups) == 0 {
		return nil, nil
	}

	// Build the raw message block for all groups, numbered for traceability.
	var rawMessages strings.Builder
	totalOriginalChars := 0
	for i, group := range groups {
		rawMessages.WriteString(fmt.Sprintf("--- Operation %d ---\n", i+1))
		for _, msg := range group {
			rawMessages.WriteString(fmt.Sprintf("[%s]: %s\n\n", msg.Role, msg.Content))
			totalOriginalChars += len(msg.Content)
		}
	}

	// Build structured batch prompt
	var prompt strings.Builder

	// Prepend the user's original goal when available so the summarizer can
	// evaluate strategy success relative to what was actually requested.
	if userGoal != "" {
		prompt.WriteString("## User Goal\n")
		prompt.WriteString(userGoal)
		prompt.WriteString("\n\n---\n\n")
	}

	fmt.Fprintf(&prompt, "The following %d tool call(s) are from your own recent past. "+
		"Write a first-person summary as if you are the agent recalling your own actions. "+
		"Use 'I' throughout — this is your episodic memory, not a report about someone else. "+
		"Use exactly this structure:\n\n", len(groups))

	prompt.WriteString("## Strategy\n")
	prompt.WriteString("One sentence: the approach I took to address the goal.\n\n")

	prompt.WriteString("## Operations\n")
	prompt.WriteString("One line per tool call:\n")
	prompt.WriteString("- **<tool_name>** | Inputs: <key params with exact values> | Outcome: <success/failure + key result data>\n\n")

	prompt.WriteString("## Discoveries\n")
	prompt.WriteString("Facts I confirmed or found: file contents, API behavior, test results, existing code structure. Use exact values.\n\n")

	prompt.WriteString("## Dead Ends\n")
	prompt.WriteString("Approaches I tried and abandoned. For each: what I attempted and why I abandoned it. " +
		"Write as personal lessons — 'I tried X — abandoned because Y' — so I will not re-attempt these paths.\n\n")

	prompt.WriteString("## What Worked\n")
	prompt.WriteString("The approach I used that succeeded, with enough detail that I can build on it.\n\n")

	prompt.WriteString("## Critical Artifacts\n")
	prompt.WriteString("Exact file paths, function names, error strings, command outputs, line numbers, test names — " +
		"any concrete artifact I will need to reference. One item per line.\n\n")

	prompt.WriteString("## Status\n")
	prompt.WriteString("One of: COMPLETE | PARTIAL | BLOCKED — followed by one sentence on current state.\n\n")

	prompt.WriteString("---\n\n")
	prompt.WriteString("MUST PRESERVE: exact tool names, file paths, function names, error messages, command strings, line numbers, test names.\n")
	prompt.WriteString("MUST NOT INCLUDE: XML markup, role labels, conversational filler, hedging language, obvious re-statements.\n")
	prompt.WriteString("MUST USE: first-person voice throughout ('I tried', 'I found', 'I abandoned') — never 'the agent'.\n")
	prompt.WriteString("FOR LONG TOOL OUTPUTS: abridge intelligently — extract the essential signal (key errors, relevant values, test names) rather than quoting verbatim. The goal is density, not completeness.\n")
	prompt.WriteString("Omit any section that has nothing meaningful to say (e.g. Dead Ends if no approach was abandoned).\n\n")
	prompt.WriteString("Tool calls to summarize:\n\n")
	prompt.WriteString(rawMessages.String())

	// Single LLM call covering all groups
	messages := []*types.Message{
		types.NewSystemMessage(episodicMemorySystemPrompt),
		types.NewUserMessage(prompt.String()),
	}

	response, err := provider.Complete(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM batch summarization failed: %w", err)
	}

	// Emit a single progress event for the completed batch
	tokensSaved := totalOriginalChars/4 - len(response.Content)/4
	if s.eventChannel != nil {
		s.eventChannel <- types.NewContextSummarizationProgressEvent(
			s.Name(),
			len(groups),
			len(groups),
			tokensSaved,
		)
	}

	// Return as a single summarized message covering all groups
	summary := types.NewAssistantMessage(fmt.Sprintf("[SUMMARIZED] %s", response.Content))
	summary.WithMetadata("summarized", true)
	summary.WithMetadata("original_message_count", len(groups))
	summary.WithMetadata("original_group_count", len(groups))

	return []*types.Message{summary}, nil
}

// Helper functions

// containsToolCallIndicators checks if the message content contains tool call XML tags.
func containsToolCallIndicators(content string) bool {
	return strings.Contains(content, "<tool>") && strings.Contains(content, "</tool>")
}

// shouldSkipMessage returns true if the message should be skipped during grouping.
func shouldSkipMessage(msg *types.Message) bool {
	return isSummarized(msg) || msg.Role == types.RoleSystem
}

// isToolRelatedMessage checks if a message is related to a tool call or result.
func isToolRelatedMessage(msg *types.Message) bool {
	return msg.Role == types.RoleTool ||
		(msg.Role == types.RoleAssistant && containsToolCallIndicators(msg.Content))
}

// groupToolCallsAndResults groups related tool calls with their results.
// Returns groups of messages where each group represents a tool call sequence.
func groupToolCallsAndResults(messages []*types.Message) [][]*types.Message {
	groups := make([][]*types.Message, 0)
	currentGroup := make([]*types.Message, 0)

	for _, msg := range messages {
		if shouldSkipMessage(msg) {
			continue
		}

		if isToolRelatedMessage(msg) {
			currentGroup = append(currentGroup, msg)

			if msg.Role == types.RoleTool && len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
				currentGroup = make([]*types.Message, 0)
			}
		} else if len(currentGroup) > 0 {
			groups = append(groups, currentGroup)
			currentGroup = make([]*types.Message, 0)
		}
	}

	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}
