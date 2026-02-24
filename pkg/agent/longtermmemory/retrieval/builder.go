package retrieval

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/entrhq/forge/pkg/agent/longtermmemory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/logging"
)

// builder manages incremental and full rebuilds of the VectorMap.
// A single goroutine owns the rebuild; concurrent trigger signals are coalesced
// via a 1-capacity channel so no rebuild is ever lost but signals never block.
type builder struct {
	store    longtermmemory.MemoryStore
	embedder llm.Embedder
	vm       *VectorMap
	log      *logging.Logger

	triggerCh chan struct{}
	building  atomic.Bool
}

func newBuilder(store longtermmemory.MemoryStore, embedder llm.Embedder, vm *VectorMap, log *logging.Logger) *builder {
	return &builder{
		store:     store,
		embedder:  embedder,
		vm:        vm,
		log:       log,
		triggerCh: make(chan struct{}, 1),
	}
}

// Trigger schedules a rebuild. If a rebuild is already queued the signal is a
// no-op, ensuring the channel never blocks the caller.
func (b *builder) Trigger() {
	select {
	case b.triggerCh <- struct{}{}:
	default:
	}
}

// Run starts the rebuild loop. It blocks until ctx is canceled.
func (b *builder) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.triggerCh:
			b.rebuild(ctx)
		}
	}
}

// rebuild fetches all memories, embeds them in batches, and swaps the VectorMap.
func (b *builder) rebuild(ctx context.Context) {
	if !b.building.CompareAndSwap(false, true) {
		// Another rebuild is already running; re-queue so the latest state is
		// captured once the current rebuild finishes.
		b.Trigger()
		return
	}
	defer b.building.Store(false)

	start := time.Now()
	files, err := b.store.List(ctx)
	if err != nil {
		b.log.Warnf("retrieval: builder: failed to list memories: %v", err)
		return
	}
	if len(files) == 0 {
		b.vm.Swap(nil)
		return
	}

	// Extract text content to embed.
	texts := make([]string, len(files))
	for i, f := range files {
		texts[i] = f.Content
	}

	// Check if the parent context was canceled before starting the embed call.
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Embed in a single batched call (providers may split internally).
	// Derive from ctx so cancellation propagates promptly on shutdown.
	bctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	vecs, err := b.embedder.Embed(bctx, texts)
	cancel()
	if err != nil {
		b.log.Warnf("retrieval: builder: embed failed: %v", err)
		return
	}
	if len(vecs) != len(files) {
		b.log.Warnf("retrieval: builder: embed count mismatch: got %d, want %d", len(vecs), len(files))
		return
	}

	entries := make([]MemoryVector, len(files))
	for i, f := range files {
		entries[i] = MemoryVector{
			Memory: f,
			Vector: Normalise(vecs[i]),
		}
	}

	b.vm.Swap(entries)
	b.log.Debugf("retrieval: builder: indexed %d memories in %s", len(entries), time.Since(start))
}
