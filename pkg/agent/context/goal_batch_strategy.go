package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/agent/memory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/types"
)

// completeTurn represents a user message and its adjacent [SUMMARIZED] blocks —
// the atomic unit of goal-batch compaction. Turns are never split.
type completeTurn struct {
	userMessage     *types.Message
	summaryMessages []*types.Message
}

// GoalBatchCompactionStrategy compacts old completed-turn sequences (user message +
// adjacent [SUMMARIZED] block(s)) into single [GOAL BATCH] blocks. It fires
// proactively when enough old complete turns accumulate, rather than reactively
// when the context reaches a token limit.
//
// The atomic unit of compaction is a "turn": one user message plus all [SUMMARIZED]
// blocks that immediately follow it before the next user message. Turns are never split.
// [GOAL BATCH] blocks produced by a prior run are never re-compacted.
//
// This strategy is designed to sit third in the strategy chain, after
// ToolCallSummarizationStrategy and ThresholdSummarizationStrategy have already
// cleaned up raw tool call pairs and assistant blocks.
type GoalBatchCompactionStrategy struct {
	// minMessagesOldThreshold is how many messages back from the current position
	// a turn must be before it is eligible for compaction. Keeps recent history intact.
	minMessagesOldThreshold int

	// minTurnsToCompact is the minimum number of eligible complete turns required
	// to trigger compaction. Prevents firing on isolated old turns.
	minTurnsToCompact int

	// maxTurnsPerBatch is the maximum number of complete turns to compact in a
	// single LLM call. Bounds the prompt size.
	maxTurnsPerBatch int

	eventChannel chan<- *types.AgentEvent
}

// NewGoalBatchCompactionStrategy creates a new goal-batch compaction strategy.
//
// Parameters:
//   - minMessagesOldThreshold: messages back from current before a turn is eligible (default: 20)
//   - minTurnsToCompact: minimum eligible complete turns before triggering (default: 3)
//   - maxTurnsPerBatch: maximum turns to compact per LLM call (default: 6)
func NewGoalBatchCompactionStrategy(minMessagesOldThreshold, minTurnsToCompact, maxTurnsPerBatch int) *GoalBatchCompactionStrategy {
	if minMessagesOldThreshold <= 0 {
		minMessagesOldThreshold = 20
	}
	if minTurnsToCompact <= 0 {
		minTurnsToCompact = 3
	}
	if maxTurnsPerBatch <= 0 {
		maxTurnsPerBatch = 6
	}
	return &GoalBatchCompactionStrategy{
		minMessagesOldThreshold: minMessagesOldThreshold,
		minTurnsToCompact:    minTurnsToCompact,
		maxTurnsPerBatch:     maxTurnsPerBatch,
	}
}

// SetEventChannel sets the event channel for emitting progress events during compaction.
func (s *GoalBatchCompactionStrategy) SetEventChannel(eventChan chan<- *types.AgentEvent) {
	s.eventChannel = eventChan
}

// Name returns the strategy identifier.
func (s *GoalBatchCompactionStrategy) Name() string {
	return "GoalBatchCompaction"
}

// ShouldRun returns true when enough complete turns in the eligible window meet
// the age and count thresholds for compaction.
func (s *GoalBatchCompactionStrategy) ShouldRun(conv *memory.ConversationMemory, currentTokens, maxTokens int) bool {
	messages := conv.GetAll()
	if len(messages) <= s.minMessagesOldThreshold {
		return false
	}

	eligibleMessages := messages[:len(messages)-s.minMessagesOldThreshold]
	turns := s.collectCompleteTurns(eligibleMessages)
	return len(turns) >= s.minTurnsToCompact
}

// Summarize compacts the oldest eligible complete turns into a single [GOAL BATCH] block.
// Returns the number of turns compacted.
func (s *GoalBatchCompactionStrategy) Summarize(ctx context.Context, conv *memory.ConversationMemory, provider llm.Provider) (int, error) {
	messages := conv.GetAll()
	if len(messages) <= s.minMessagesOldThreshold {
		return 0, nil
	}

	eligibleMessages := messages[:len(messages)-s.minMessagesOldThreshold]
	allTurns := s.collectCompleteTurns(eligibleMessages)
	if len(allTurns) == 0 {
		return 0, nil
	}

	// Cap at maxTurnsPerBatch to bound prompt size.
	batch := allTurns
	if len(batch) > s.maxTurnsPerBatch {
		batch = batch[:s.maxTurnsPerBatch]
	}

	// Single LLM call covers the full batch.
	llmMessages := []*types.Message{
		types.NewSystemMessage(episodicMemorySystemPrompt),
		types.NewUserMessage(s.buildGoalBatchPrompt(batch)),
	}

	response, err := provider.Complete(ctx, llmMessages)
	if err != nil {
		return 0, fmt.Errorf("goal batch compaction LLM call failed: %w", err)
	}

	// Build the replacement [GOAL BATCH] message.
	goalBatch := types.NewAssistantMessage(fmt.Sprintf("[GOAL BATCH]\n%s", response.Content))
	goalBatch.WithMetadata("summarized", true)
	goalBatch.WithMetadata("summary_type", "goal_batch")
	goalBatch.WithMetadata("turn_count", len(batch))

	// Replace all batched messages with the single [GOAL BATCH] block,
	// inserting it at the position of the first batched message.
	batchSet := buildBatchSet(batch)
	newMessages := make([]*types.Message, 0, len(messages)-batchMessageCount(batch)+1)
	inserted := false
	for _, msg := range messages {
		if batchSet[msg] {
			if !inserted {
				newMessages = append(newMessages, goalBatch)
				inserted = true
			}
			continue
		}
		newMessages = append(newMessages, msg)
	}

	if !inserted {
		return 0, fmt.Errorf("goal batch compaction: failed to locate batch messages in conversation")
	}

	conv.Clear()
	conv.AddMultiple(newMessages)

	if s.eventChannel != nil {
		s.eventChannel <- types.NewContextSummarizationProgressEvent(
			s.Name(),
			len(batch),
			len(batch),
			0,
		)
	}

	return len(batch), nil
}

// collectCompleteTurns scans messages and returns complete turns in order.
//
// A complete turn is a user message followed by at least one regular [SUMMARIZED]
// block (summarized=true, not a [GOAL BATCH]) before the next user message.
// System messages and existing [GOAL BATCH] blocks are skipped transparently.
// Incomplete turns (user message with no following summary) are never returned.
func (s *GoalBatchCompactionStrategy) collectCompleteTurns(messages []*types.Message) []completeTurn {
	var turns []completeTurn

	i := 0
	for i < len(messages) {
		msg := messages[i]

		// Skip system messages and already-compacted [GOAL BATCH] blocks.
		if msg.Role == types.RoleSystem || isGoalBatch(msg) {
			i++
			continue
		}

		// A turn starts with a user message.
		if msg.Role != types.RoleUser {
			i++
			continue
		}

		userMsg := msg
		i++

		// Collect all adjacent regular [SUMMARIZED] blocks.
		var summaries []*types.Message
		for i < len(messages) && isRegularSummary(messages[i]) {
			summaries = append(summaries, messages[i])
			i++
		}

		// Only a complete turn if at least one summary follows.
		if len(summaries) > 0 {
			turns = append(turns, completeTurn{
				userMessage:     userMsg,
				summaryMessages: summaries,
			})
		}
	}

	return turns
}

// buildGoalBatchPrompt constructs the structured compaction prompt for a batch of turns.
func (s *GoalBatchCompactionStrategy) buildGoalBatchPrompt(batch []completeTurn) string {
	var b strings.Builder

	b.WriteString("The following is a sequence of complete turns from your own recent past. " +
		"Each turn is a user instruction followed by your execution summary. " +
		"Together they represent a goal arc — the user's intent may have evolved " +
		"across turns through redirections and clarifications. " +
		"Write a goal-batch summary as if you are the agent recalling this arc. " +
		"Use 'I' throughout — this is your episodic memory, not a report about someone else.\n\n")

	b.WriteString("Use exactly these sections (omit any section with nothing meaningful to say):\n\n")

	b.WriteString("## Goal Arc\n")
	b.WriteString("What was being worked on across this sequence. Include how the goal evolved " +
		"if the user redirected or refined it — this shapes how future related work should be understood.\n\n")

	b.WriteString("## Human Direction\n")
	b.WriteString("Key instructions, corrections, and constraints the user introduced mid-arc that shaped the work. " +
		"Preserve the user's actual intent as it developed — not just my response to it.\n\n")

	b.WriteString("## What Was Achieved\n")
	b.WriteString("Concrete outcomes: what works, what was completed, what is now in place.\n\n")

	b.WriteString("## Dead Ends\n")
	b.WriteString("Approaches I tried and abandoned across this arc. Write as personal lessons — " +
		"'I tried X — abandoned because Y' — so I will not re-attempt these paths.\n\n")

	b.WriteString("## Lasting Constraints\n")
	b.WriteString("Anything from this arc that still applies to future work: architectural decisions, " +
		"things to avoid, invariants established.\n\n")

	b.WriteString("## Key Artifacts\n")
	b.WriteString("Exact file paths, function names, error strings, test names that may be referenced again. " +
		"One item per line.\n\n")

	b.WriteString("---\n\n")
	b.WriteString("MUST PRESERVE: user intent and mid-arc redirections, dead ends, lasting constraints, " +
		"key artifacts (file paths, function names, error strings, test names).\n")
	b.WriteString("MUST NOT INCLUDE: operational detail superseded within the arc, tool call mechanics, " +
		"intermediate states that were overwritten, conversational filler.\n")
	b.WriteString("MUST USE: first-person voice throughout ('I tried', 'I found', 'I abandoned') — never 'the agent'.\n\n")

	b.WriteString("Turns to compact:\n\n")
	for i, turn := range batch {
		fmt.Fprintf(&b, "--- Turn %d ---\n", i+1)
		fmt.Fprintf(&b, "[user]: %s\n\n", turn.userMessage.Content)
		for j, sm := range turn.summaryMessages {
			fmt.Fprintf(&b, "[summary %d]: %s\n\n", j+1, sm.Content)
		}
	}

	return b.String()
}

// isGoalBatch returns true if the message is a [GOAL BATCH] block produced by
// a prior compaction run.
func isGoalBatch(msg *types.Message) bool {
	if msg.Metadata == nil {
		return false
	}
	summaryType, _ := msg.Metadata["summary_type"].(string)
	return summaryType == "goal_batch"
}

// isRegularSummary returns true if the message is a [SUMMARIZED] block but not
// a [GOAL BATCH] block. These are the blocks eligible to be included in a turn.
func isRegularSummary(msg *types.Message) bool {
	return isSummarized(msg) && !isGoalBatch(msg)
}

// buildBatchSet returns a set of all message pointers in the batch for O(1) lookup.
func buildBatchSet(batch []completeTurn) map[*types.Message]bool {
	set := make(map[*types.Message]bool, batchMessageCount(batch))
	for _, turn := range batch {
		set[turn.userMessage] = true
		for _, sm := range turn.summaryMessages {
			set[sm] = true
		}
	}
	return set
}

// batchMessageCount returns the total number of raw messages across all turns in a batch.
func batchMessageCount(batch []completeTurn) int {
	total := 0
	for _, turn := range batch {
		total += 1 + len(turn.summaryMessages)
	}
	return total
}
