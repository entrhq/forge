package capture

import (
	"context"

	"github.com/entrhq/forge/pkg/agent/longtermmemory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/logging"
)

const triggerBufferSize = 8

// Pipeline is a single long-lived goroutine that receives TriggerEvents and
// runs the classifier asynchronously (ADR-0046 §Option 3).
//
// The agent loop submits events via Enqueue which is non-blocking: if the
// internal buffer is full the event is dropped and a debug log is emitted.
// This guarantees zero latency impact on the agent loop regardless of how
// slowly the classifier LLM responds.
type Pipeline struct {
	ch         chan TriggerEvent
	classifier *Classifier
	store      longtermmemory.MemoryStore
	rebuildFn  func()
	log        *logging.Logger
}

// NewPipeline constructs a Pipeline ready to be started.
//
// Parameters:
//   - classifierProvider: the LLM provider used for the classifier call
//   - classifierModel: if non-empty and provider implements llm.ModelCloner, this
//     model is used for each classifier call (ADR-0042 pattern)
//   - store: the memory store that receives captured MemoryFiles
//   - rebuildFn: called after each successful capture batch to signal the retrieval
//     engine to rebuild its in-memory vector map; must be non-blocking (ADR-0046)
func NewPipeline(
	classifierProvider llm.Provider,
	classifierModel string,
	store longtermmemory.MemoryStore,
	rebuildFn func(),
	log *logging.Logger,
) *Pipeline {
	if log == nil {
		log, _ = logging.NewLogger("memory")
	}
	p := &Pipeline{
		ch:        make(chan TriggerEvent, triggerBufferSize),
		store:     store,
		rebuildFn: rebuildFn,
		log:       log,
	}
	p.classifier = NewClassifier(classifierProvider, classifierModel, store, log)

	return p
}

// Logger returns the logger used by this pipeline (used by Observer for consistent logging).
func (p *Pipeline) Logger() *logging.Logger {
	return p.log
}

// Start launches the pipeline goroutine. It runs until ctx is canceled.
// Start must be called exactly once before any Enqueue calls.
func (p *Pipeline) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-p.ch:
				p.process(ctx, event)
			}
		}
	}()
}

// Enqueue submits a TriggerEvent to the pipeline for asynchronous processing.
// This method is non-blocking: if the internal buffer is full the event is
// dropped and a debug log is emitted. Callers must never block waiting on this.
func (p *Pipeline) Enqueue(event TriggerEvent) {
	select {
	case p.ch <- event:
		p.log.Debugf("memory: capture trigger enqueued (kind=%s session=%s messages=%d)", event.Kind, event.SessionID, len(event.Messages))
	default:
		p.log.Warnf("memory: capture trigger dropped — pipeline buffer full (kind=%s)", event.Kind)
	}
}

// process runs the classifier against a single TriggerEvent and writes any
// resulting MemoryFiles to the store. It is called sequentially by the pipeline
// goroutine, ensuring at most one classifier call is in-flight at any time.
//
// Classifier errors and individual write errors are non-fatal: a debug/warn log
// is emitted and the pipeline goroutine continues processing the next event.
func (p *Pipeline) process(ctx context.Context, event TriggerEvent) {
	p.log.Infof("memory: processing capture trigger (kind=%s session=%s messages=%d)", event.Kind, event.SessionID, len(event.Messages))

	memories, err := p.classifier.Classify(ctx, event)
	if err != nil {
		p.log.Warnf("memory: classifier error, capture skipped (kind=%s): %v", event.Kind, err)
		return
	}
	if len(memories) == 0 {
		p.log.Infof("memory: classifier found nothing memory-worthy this turn (kind=%s session=%s)", event.Kind, event.SessionID)
		return
	}

	p.log.Infof("memory: classifier identified %d memory candidate(s) to write", len(memories))

	wrote := 0
	for _, m := range memories {
		if writeErr := p.store.Write(ctx, m); writeErr != nil {
			p.log.Warnf("memory: failed to write memory (id=%s scope=%s): %v", m.Meta.ID, m.Meta.Scope, writeErr)
			continue
		}
		p.log.Infof("memory: wrote memory id=%s scope=%s category=%s session=%s", m.Meta.ID, m.Meta.Scope, m.Meta.Category, event.SessionID)
		wrote++
	}

	if wrote > 0 {
		p.log.Infof("memory: captured %d/%d memories for session %s", wrote, len(memories), event.SessionID)
	}

	// Signal the retrieval engine only when at least one memory was persisted.
	// rebuildFn must be non-blocking (ADR-0046 §rebuildFn contract).
	if wrote > 0 && p.rebuildFn != nil {
		p.rebuildFn()
	}
}
