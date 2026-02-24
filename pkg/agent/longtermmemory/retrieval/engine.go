// Package retrieval implements a HyDE-based long-term memory retrieval engine.
// On each new user turn it generates hypothetical memory sentences via a
// flash-class LLM, embeds them, performs cosine-similarity search over an
// in-memory VectorMap, and optionally follows relationship edges for 1-hop
// graph traversal. Results are injected into the system prompt before the
// first LLM call of each turn.
package retrieval

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/entrhq/forge/pkg/agent/longtermmemory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/logging"
	"github.com/entrhq/forge/pkg/types"
)

// retrievalTimeout caps the total wall-clock time allowed for one retrieval
// cycle (HyDE generation + embedding + vector search + graph hops).
// HyDE generation requires an LLM call; flash-class models typically respond
// in 2-5s, but can take longer under load. 15s gives ample headroom while
// still ensuring retrieval never blocks a user turn indefinitely.
// If you configure a large model (e.g. Sonnet) as hypothesis_model, consider
// raising this or switching to a flash-class model for lower latency.
const retrievalTimeout = 15 * time.Second

// Config holds runtime parameters for the Engine.
type Config struct {
	// HypothesisProvider is the LLM used for HyDE generation.
	HypothesisProvider llm.Provider
	// HypothesisModel overrides the provider model for hypothesis generation.
	// Leave empty to use the provider's default model.
	HypothesisModel string
	// HypothesisCount is the number of hypothetical sentences to generate.
	HypothesisCount int
	// TopK is the number of nearest-neighbour results to return before graph
	// expansion.
	TopK int
	// HopDepth is the number of relationship edges to follow after the initial
	// similarity search (0 = no graph traversal).
	HopDepth int
	// InjectionTokenBudget caps the injected context. 0 = no cap.
	InjectionTokenBudget int
}

// Engine is the public interface of the retrieval subsystem.
type Engine struct {
	cfg     Config
	vm      *VectorMap
	bld     *builder
	embedder llm.Embedder
	log     *logging.Logger

	// Per-turn cache: keyed by turnID.
	cacheMu sync.Mutex
	cache   map[string]string
}

// New creates and returns a new Engine. It immediately triggers an initial
// rebuild of the VectorMap in the background.
func New(
	store longtermmemory.MemoryStore,
	embedder llm.Embedder,
	cfg Config,
	log *logging.Logger,
) *Engine {
	vm := NewVectorMap()
	bld := newBuilder(store, embedder, vm, log)

	e := &Engine{
		cfg:      cfg,
		vm:       vm,
		bld:      bld,
		embedder: embedder,
		log:      log,
		cache:    make(map[string]string),
	}
	return e
}

// Start launches the background builder goroutine and triggers the initial
// index build. Call once after constructing the Engine.
func (e *Engine) Start(ctx context.Context) {
	go e.bld.Run(ctx)
	e.bld.Trigger()
}

// Rebuild schedules an asynchronous re-index of the VectorMap. Called by the
// capture pipeline's rebuildFn after a new memory is written.
func (e *Engine) Rebuild() {
	e.bld.Trigger()
}

// RetrieveForTurn returns the formatted memory injection string for the given
// turn. Results are cached so repeated calls within the same turn are free.
func (e *Engine) RetrieveForTurn(
	ctx context.Context,
	turnID string,
	history []*types.Message,
	userMessage string,
) string {
	// Cache hit — zero cost.
	e.cacheMu.Lock()
	if cached, ok := e.cache[turnID]; ok {
		e.cacheMu.Unlock()
		return cached
	}
	e.cacheMu.Unlock()

	result := e.retrieve(ctx, history, userMessage)

	e.cacheMu.Lock()
	// Evict any stale entries from previous turns (keep only the current one).
	e.cache = map[string]string{turnID: result}
	e.cacheMu.Unlock()

	return result
}

// retrieve performs the full HyDE → embed → similarity → hop pipeline.
// It always returns within retrievalTimeout and returns "" on any error.
func (e *Engine) retrieve(
	ctx context.Context,
	history []*types.Message,
	userMessage string,
) string {
	if e.vm.Len() == 0 {
		e.log.Debugf("retrieval: skipping — vector index is empty (no memories indexed yet)")
		return ""
	}

	ctx, cancel := context.WithTimeout(ctx, retrievalTimeout)
	defer cancel()

	window := buildWindow(history, userMessage)

	hypothesisCount := e.cfg.HypothesisCount
	if hypothesisCount <= 0 {
		hypothesisCount = 5
	}

	// 1. Generate hypothetical memory sentences.
	hypotheses, err := generateHypotheses(ctx, e.cfg.HypothesisProvider, e.cfg.HypothesisModel, window, hypothesisCount)
	if err != nil {
		e.log.Debugf("retrieval: HyDE generation failed (silent): %v", err)
		return ""
	}
	if len(hypotheses) == 0 {
		return ""
	}

	// Log each hypothesis so operators can inspect what the HyDE step produced.
	for i, h := range hypotheses {
		e.log.Debugf("retrieval: hypothesis[%d]: %s", i+1, h)
	}

	// 2. Embed the hypotheses.
	vecs, err := e.embedder.Embed(ctx, hypotheses)
	if err != nil {
		e.log.Debugf("retrieval: embed failed (silent): %v", err)
		return ""
	}

	topK := e.cfg.TopK
	if topK <= 0 {
		topK = 10
	}

	// 3. Union-of-searches: run a TopK query for each hypothesis vector
	// independently, then merge by keeping the highest cosine score per
	// memory UUID. This preserves the diversity benefit of HyDE — a memory
	// that scores highly against one specific hypothesis is not buried by
	// averaging across all others.
	type scored struct {
		mv    MemoryVector
		score float64
	}
	best := make(map[string]scored, topK*len(vecs))
	for _, vec := range vecs {
		nv := Normalise(vec)
		for _, mv := range e.vm.TopK(nv, topK) {
			s := dotProduct(nv, mv.Vector)
			id := mv.Memory.Meta.ID
			if existing, ok := best[id]; !ok || s > existing.score {
				best[id] = scored{mv: mv, score: s}
			}
		}
	}

	// 4. Sort merged candidates by score descending and cap at topK.
	merged := make([]scored, 0, len(best))
	for _, s := range best {
		merged = append(merged, s)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].score > merged[j].score
	})
	if len(merged) > topK {
		merged = merged[:topK]
	}
	hits := make([]MemoryVector, len(merged))
	for i, s := range merged {
		hits[i] = s.mv
	}

	// 5. Graph hop traversal.
	hopDepth := e.cfg.HopDepth
	if hopDepth < 0 {
		hopDepth = 0
	}
	if hopDepth > 0 {
		hits = e.expandHops(hits, hopDepth)
	}

	// 6. Collect unique memory files in ranked order.
	seen := make(map[string]struct{}, len(hits))
	memories := make([]*longtermmemory.MemoryFile, 0, len(hits))
	for _, h := range hits {
		if _, ok := seen[h.Memory.Meta.ID]; ok {
			continue
		}
		seen[h.Memory.Meta.ID] = struct{}{}
		memories = append(memories, h.Memory)
	}

	injection := FormatInjection(memories, e.cfg.InjectionTokenBudget)
	if injection == "" {
		e.log.Debugf("retrieval: no memories passed similarity threshold (index_size=%d, hypotheses=%d, top_k=%d)", e.vm.Len(), len(hypotheses), topK)
	} else {
		e.log.Infof("retrieval: injecting %d memories into system prompt (%d chars, index_size=%d)", len(memories), len(injection), e.vm.Len())
	}
	return injection
}

// expandHops follows Related edges up to depth hops, appending any newly
// discovered memories to the result set.
func (e *Engine) expandHops(initial []MemoryVector, depth int) []MemoryVector {
	result := make([]MemoryVector, len(initial))
	copy(result, initial)

	frontier := initial
	visited := make(map[string]struct{}, len(initial))
	for _, mv := range initial {
		visited[mv.Memory.Meta.ID] = struct{}{}
	}

	for d := 0; d < depth; d++ {
		var next []MemoryVector
		for _, mv := range frontier {
			for _, rel := range mv.Memory.Meta.Related {
				if _, seen := visited[rel.ID]; seen {
					continue
				}
				visited[rel.ID] = struct{}{}
				if entry := e.vm.Lookup(rel.ID); entry != nil {
					result = append(result, *entry)
					next = append(next, *entry)
				}
			}
		}
		if len(next) == 0 {
			break
		}
		frontier = next
	}
	return result
}


