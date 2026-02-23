// Package capture implements the async long-term memory capture pipeline (ADR-0046).
//
// The pipeline observes completed agent turns and goal-arc compaction events,
// runs a classifier LLM call to identify memory-worthy content, and writes
// resulting MemoryFile instances to the storage layer (ADR-0044).
//
// Architecture: a single persistent goroutine reads from a bounded channel,
// ensuring zero agent-loop blocking and sequential (race-free) store writes.
package capture

// TriggerKind identifies what caused a capture pass to be initiated.
type TriggerKind string

const (
	// TriggerKindTurn fires after every completed user turn.
	TriggerKindTurn TriggerKind = "turn"
	// TriggerKindCompaction fires when a goal-arc compaction event is raised (ADR-0041).
	TriggerKindCompaction TriggerKind = "compaction"
)

// TriggerEvent carries the context snapshot the capture pipeline should analyze.
type TriggerEvent struct {
	// Kind identifies what caused this trigger.
	Kind TriggerKind

	// Messages holds the full conversation window: all user and assistant messages
	// since the last summarisation event. Tool call content is stripped by the
	// observer before enqueueing. If no summarisation has occurred, this is the
	// full session history minus tool content.
	Messages []ConversationMessage

	// SessionID is the current agent session identifier.
	SessionID string
}

// ConversationMessage is a stripped user or assistant message, free of tool content.
type ConversationMessage struct {
	// Role is "user" or "assistant".
	Role string

	// Content is the human-language text of the message.
	Content string
}
