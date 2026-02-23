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

// log is a convenience accessor so Observer methods can call o.log().Xf(...)
func (o *Observer) log() interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
} {
	return o.pipeline.Logger()
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
	o.log().Debugf("memory: OnTurnComplete called (raw_messages=%d session=%s)", len(messages), sessionID)
	stripped := StripToolContent(messages)
	o.log().Debugf("memory: after stripping tool content (stripped_messages=%d)", len(stripped))
	if len(stripped) == 0 {
		o.log().Debugf("memory: all messages stripped — skipping capture trigger")
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
	o.log().Debugf("memory: OnCompaction called (raw_messages=%d session=%s)", len(messages), sessionID)
	stripped := StripToolContent(messages)
	o.log().Debugf("memory: after stripping tool content (stripped_messages=%d)", len(stripped))
	if len(stripped) == 0 {
		o.log().Debugf("memory: all messages stripped — skipping compaction capture trigger")
		return
	}
	o.pipeline.Enqueue(TriggerEvent{
		Kind:      TriggerKindCompaction,
		Messages:  stripped,
		SessionID: sessionID,
	})
}
