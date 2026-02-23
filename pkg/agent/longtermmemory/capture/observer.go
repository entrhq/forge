package capture

import "github.com/entrhq/forge/pkg/types"

// Observer hooks into the agent loop and enqueues a capture trigger after
// every completed user turn. There is no cadence counter or modulo check —
// every turn fires a trigger. The classifier is responsible for determining
// what (if anything) is worth remembering from the full conversation window.
//
// Observer is safe to use from multiple goroutines: all state is in Pipeline
// which uses a buffered channel for synchronization.
type Observer struct {
	pipeline *Pipeline
}

// NewObserver creates an Observer that enqueues a trigger on every turn completion.
func NewObserver(pipeline *Pipeline) *Observer {
	return &Observer{pipeline: pipeline}
}

// OnTurnComplete is called by the agent loop after each completed user turn.
// It strips tool call content from messages, then enqueues a TriggerKindTurn event.
//
// This method is synchronous and must return immediately — all heavy work is
// deferred to the pipeline goroutine (ADR-0046 §Observer).
//
// messages is the full conversation window (all user+assistant messages since the
// last summarisation event). Tool call content is stripped here before enqueueing.
func (o *Observer) OnTurnComplete(messages []*types.Message, sessionID string) {
	stripped := StripToolContent(messages)
	if len(stripped) == 0 {
		return
	}
	o.pipeline.Enqueue(TriggerEvent{
		Kind:      TriggerKindTurn,
		Messages:  stripped,
		SessionID: sessionID,
	})
}

// OnCompaction is called by the compaction system (ADR-0041) when a goal-arc
// compaction event fires. It enqueues a TriggerKindCompaction event regardless
// of turn state — compaction represents a natural second pass over a full goal arc.
//
// messages is the full conversation arc, tool content excluded.
func (o *Observer) OnCompaction(messages []*types.Message, sessionID string) {
	stripped := StripToolContent(messages)
	if len(stripped) == 0 {
		return
	}
	o.pipeline.Enqueue(TriggerEvent{
		Kind:      TriggerKindCompaction,
		Messages:  stripped,
		SessionID: sessionID,
	})
}
