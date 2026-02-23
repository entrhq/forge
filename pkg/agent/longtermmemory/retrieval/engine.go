// Package retrieval implements a HyDE-based long-term memory retrieval engine.
// On each new user turn it generates hypothetical memory sentences via a
// flash-class LLM, embeds them, performs cosine-similarity search over an
// in-memory VectorMap, and optionally follows relationship edges for 1-hop
// graph traversal. Results are injected into the system prompt before the
// first LLM call of each turn.
package retrieval

import (
	"context"
	"sync"
	"time"

	"github.com/entrhq/forge/pkg/agent/longtermmemory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/logging"
	"github.com/entrhq/forge/pkg/types"
)

const retrievalTimeout = 2 * time.Second

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

	// 2. Embed the hypotheses.
	vecs, err := e.embedder.Embed(ctx, hypotheses)
	if err != nil {
		e.log.Debugf("retrieval: embed failed (silent): %v", err)
		return ""
	}

	// 3. Average the hypothesis vectors to form a single query vector.
	query := averageVectors(vecs)
	if query == nil {
		return ""
	}
	query = Normalise(query)

	topK := e.cfg.TopK
	if topK <= 0 {
		topK = 10
	}

	// 4. Cosine similarity search.
	hits := e.vm.TopK(query, topK)

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

	return FormatInjection(memories, e.cfg.InjectionTokenBudget)
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

// averageVectors computes the element-wise mean of a slice of vectors.
func averageVectors(vecs [][]float32) []float32 {
	if len(vecs) == 0 {
		return nil
	}
	dim := len(vecs[0])
	if dim == 0 {
		return nil
	}
	avg := make([]float32, dim)
	for _, v := range vecs {
		if len(v) != dim {
			continue
		}
		for i, x := range v {
			avg[i] += x
		}
	}
	n := float32(len(vecs))
	for i := range avg {
		avg[i] /= n
	}
	return avg
}
