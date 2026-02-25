package retrieval

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/entrhq/forge/pkg/agent/longtermmemory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/logging"
	"github.com/entrhq/forge/pkg/types"
)

// testLogger returns a logger suitable for unit tests. It falls back to stderr
// gracefully when ~/.forge/logs is unavailable in CI environments.
func testLogger(t *testing.T) *logging.Logger {
	t.Helper()
	log, _ := logging.NewLogger("retrieval-test")
	return log
}

// buildEngine creates an Engine with the provided configuration components wired
// together. The builder's background goroutine is started against a context that
// is cancelled when the test ends.
func buildEngine(
	t *testing.T,
	store longtermmemory.MemoryStore,
	embedder *fakeEmbedder,
	provider *fakeProvider,
	topK, hopDepth, hypothesisCount int,
) *Engine {
	t.Helper()
	log := testLogger(t)
	cfg := Config{
		HypothesisProvider:   provider,
		HypothesisModel:      "fake",
		HypothesisCount:      hypothesisCount,
		TopK:                 topK,
		HopDepth:             hopDepth,
		InjectionTokenBudget: 0,
	}
	eng := New(store, embedder, cfg, log)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	eng.Start(ctx)
	return eng
}

// ── union-of-searches ────────────────────────────────────────────────────────
//
// These tests validate that the retrieval engine performs a per-hypothesis
// TopK search and merges results by keeping the highest cosine score per
// memory UUID — preserving the diversity benefit of HyDE.

// TestRetrieve_UnionFindsDistinctMemories checks that two orthogonal hypotheses
// each retrieve their closest memory and both appear in the final result set.
//
// The key benefit over average-then-search: with averaging, the centroid of
// two orthogonal vectors ([1,0] and [0,1]) becomes [0.707, 0.707], scoring
// both memories equally. Union-of-searches runs two separate TopK queries so
// each memory is found at full confidence by its dedicated hypothesis.
func TestRetrieve_UnionFindsDistinctMemories(t *testing.T) {
	// m1 lives near (1,0), m2 lives near (0,1).
	m1 := makeMemoryFile("m1", "content about databases", "")
	m2 := makeMemoryFile("m2", "content about networking", "")

	store := &fakeStore{files: []*longtermmemory.MemoryFile{m1, m2}}
	emb := newFakeEmbedder(2)
	emb.set("content about databases", Normalise([]float32{1, 0}))
	emb.set("content about networking", Normalise([]float32{0, 1}))

	// Two hypotheses pointing in orthogonal directions — each covers one memory.
	hyp1 := "database schema design"
	hyp2 := "tcp connection handling"
	emb.set(hyp1, Normalise([]float32{1, 0}))
	emb.set(hyp2, Normalise([]float32{0, 1}))

	log := testLogger(t)
	cfg := Config{
		HypothesisProvider:   &fakeProvider{response: hyp1 + "\n" + hyp2},
		HypothesisCount:      2,
		TopK:                 2, // Return up to 2; union finds both via separate searches.
		HopDepth:             0,
		InjectionTokenBudget: 0,
	}
	eng := New(store, emb, cfg, log)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	eng.Start(ctx)
	waitForIndex(t, eng, 2)

	got := eng.RetrieveForTurn(context.Background(), "turn-union", nil, "db and network question")
	if !strings.Contains(got, "content about databases") {
		t.Errorf("expected m1 in injection; got: %q", got)
	}
	if !strings.Contains(got, "content about networking") {
		t.Errorf("expected m2 in injection; got: %q", got)
	}
}

// TestRetrieve_UnionKeepsHighestScore checks that when the same memory is
// returned by multiple hypothesis searches, the entry with the highest score
// is kept (not duplicated, and not the lower score).
func TestRetrieve_UnionKeepsHighestScore(t *testing.T) {
	m1 := makeMemoryFile("m1", "shared memory content", "")

	store := &fakeStore{files: []*longtermmemory.MemoryFile{m1}}
	emb := newFakeEmbedder(2)
	emb.set("shared memory content", Normalise([]float32{1, 0}))

	// Both hypotheses hit m1; one is closer (score ≈ 1), one is farther.
	hypClose := "very relevant hypothesis"
	hypFar := "loosely related hypothesis"
	emb.set(hypClose, Normalise([]float32{1, 0}))   // cosine ≈ 1.0 vs m1
	emb.set(hypFar, Normalise([]float32{0.6, 0.8})) // cosine ≈ 0.6 vs m1

	log := testLogger(t)
	cfg := Config{
		HypothesisProvider:   &fakeProvider{response: hypClose + "\n" + hypFar},
		HypothesisCount:      2,
		TopK:                 5,
		HopDepth:             0,
		InjectionTokenBudget: 0,
	}
	eng := New(store, emb, cfg, log)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	eng.Start(ctx)
	waitForIndex(t, eng, 1)

	got := eng.RetrieveForTurn(context.Background(), "turn-dedup", nil, "relevant question")

	// m1 should appear exactly once in the output.
	count := strings.Count(got, "shared memory content")
	if count != 1 {
		t.Errorf("expected m1 exactly once in injection, got %d occurrences; output: %q", count, got)
	}
}

// ── expandHops ────────────────────────────────────────────────────────────────

func TestExpandHops_NoRelated(t *testing.T) {
	store := &fakeStore{}
	emb := newFakeEmbedder(2)
	eng := buildEngine(t, store, emb, &fakeProvider{response: "hypothesis"}, 5, 1, 3)

	fa := makeMemoryFile("a", "content a", "")
	eng.vm.Swap([]MemoryVector{
		{Memory: fa, Vector: Normalise([]float32{1, 0})},
	})

	initial := []MemoryVector{{Memory: fa, Vector: Normalise([]float32{1, 0})}}
	got := eng.expandHops(initial, 1)
	if len(got) != 1 {
		t.Errorf("expandHops with no related: len = %d, want 1", len(got))
	}
}

func TestExpandHops_FollowsRelatedEdge(t *testing.T) {
	store := &fakeStore{}
	emb := newFakeEmbedder(2)
	eng := buildEngine(t, store, emb, &fakeProvider{response: "hypothesis"}, 5, 1, 3)

	fb := makeMemoryFile("b", "related content", "")
	fa := makeMemoryFileWithRelated("a", "root content", []longtermmemory.RelatedMemory{
		{ID: "b"},
	})

	eng.vm.Swap([]MemoryVector{
		{Memory: fa, Vector: Normalise([]float32{1, 0})},
		{Memory: fb, Vector: Normalise([]float32{0, 1})},
	})

	initial := []MemoryVector{{Memory: fa, Vector: Normalise([]float32{1, 0})}}
	got := eng.expandHops(initial, 1)
	if len(got) != 2 {
		t.Errorf("expected 2 after hop, got %d", len(got))
	}
	ids := map[string]bool{got[0].Memory.Meta.ID: true, got[1].Memory.Meta.ID: true}
	if !ids["b"] {
		t.Error("expected related memory 'b' to be included after hop")
	}
}

func TestExpandHops_DeduplicatesVisited(t *testing.T) {
	store := &fakeStore{}
	emb := newFakeEmbedder(2)
	eng := buildEngine(t, store, emb, &fakeProvider{response: "h"}, 5, 2, 3)

	// a → b → a (cycle)
	fa := makeMemoryFileWithRelated("a", "a", []longtermmemory.RelatedMemory{{ID: "b"}})
	fb := makeMemoryFileWithRelated("b", "b", []longtermmemory.RelatedMemory{{ID: "a"}})

	eng.vm.Swap([]MemoryVector{
		{Memory: fa, Vector: Normalise([]float32{1, 0})},
		{Memory: fb, Vector: Normalise([]float32{0, 1})},
	})

	initial := []MemoryVector{{Memory: fa, Vector: Normalise([]float32{1, 0})}}
	got := eng.expandHops(initial, 2)
	// Should be exactly 2: a + b, no duplicates from cycle.
	if len(got) != 2 {
		t.Errorf("expected 2 (no duplicates), got %d", len(got))
	}
}

// ── RetrieveForTurn ──────────────────────────────────────────────────────────

func TestRetrieveForTurn_EmptyVectorMap(t *testing.T) {
	store := &fakeStore{}
	emb := newFakeEmbedder(2)
	eng := buildEngine(t, store, emb, &fakeProvider{response: "h"}, 5, 0, 3)
	// Don't populate the vector map — engine should return "" fast.

	got := eng.RetrieveForTurn(context.Background(), "turn1", nil, "hello")
	if got != "" {
		t.Errorf("expected empty string for empty index, got %q", got)
	}
}

func TestRetrieveForTurn_CachesResult(t *testing.T) {
	store := &fakeStore{
		files: []*longtermmemory.MemoryFile{
			makeMemoryFile("m1", "memory content one", ""),
		},
	}
	emb := newFakeEmbedder(2)
	emb.set("memory content one", Normalise([]float32{1, 0}))

	callCount := 0
	prov := &countingProvider{inner: &fakeProvider{response: "hypothesis about memory"}, calls: &callCount}

	log := testLogger(t)
	cfg := Config{
		HypothesisProvider:   prov,
		HypothesisModel:      "fake",
		HypothesisCount:      1,
		TopK:                 5,
		HopDepth:             0,
		InjectionTokenBudget: 0,
	}
	eng := New(store, emb, cfg, log)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	eng.Start(ctx)

	// Wait for initial build.
	waitForIndex(t, eng, 1)

	first := eng.RetrieveForTurn(context.Background(), "turn-x", nil, "test query")
	second := eng.RetrieveForTurn(context.Background(), "turn-x", nil, "test query")

	if first != second {
		t.Errorf("cache miss: first=%q second=%q", first, second)
	}
	if callCount > 1 {
		t.Errorf("provider called %d times for same turnID, want ≤1", callCount)
	}
}

func TestRetrieveForTurn_NewTurnEvictsCache(t *testing.T) {
	store := &fakeStore{
		files: []*longtermmemory.MemoryFile{
			makeMemoryFile("m1", "memory content", ""),
		},
	}
	emb := newFakeEmbedder(2)
	emb.set("memory content", Normalise([]float32{1, 0}))

	log := testLogger(t)
	cfg := Config{
		HypothesisProvider:   &fakeProvider{response: "hyp"},
		HypothesisCount:      1,
		TopK:                 5,
		InjectionTokenBudget: 0,
	}
	eng := New(store, emb, cfg, log)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	eng.Start(ctx)
	waitForIndex(t, eng, 1)

	_ = eng.RetrieveForTurn(context.Background(), "turn-1", nil, "query")
	// A different turn should not return cached result — it should re-run.
	_ = eng.RetrieveForTurn(context.Background(), "turn-2", nil, "query")

	// Verify the cache only holds turn-2 (turn-1 was evicted).
	eng.cacheMu.Lock()
	_, hasTurn1 := eng.cache["turn-1"]
	_, hasTurn2 := eng.cache["turn-2"]
	eng.cacheMu.Unlock()

	if hasTurn1 {
		t.Error("turn-1 should have been evicted from cache")
	}
	if !hasTurn2 {
		t.Error("turn-2 should be present in cache")
	}
}

func TestRetrieveForTurn_HyDEProviderError_ReturnEmpty(t *testing.T) {
	store := &fakeStore{
		files: []*longtermmemory.MemoryFile{
			makeMemoryFile("m1", "content", ""),
		},
	}
	emb := newFakeEmbedder(2)
	emb.set("content", Normalise([]float32{1, 0}))

	errProv := &fakeProvider{err: errEmbed} // reuse sentinel error

	log := testLogger(t)
	cfg := Config{
		HypothesisProvider: errProv,
		HypothesisCount:    1,
		TopK:               5,
	}
	eng := New(store, emb, cfg, log)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	eng.Start(ctx)
	waitForIndex(t, eng, 1)

	got := eng.RetrieveForTurn(context.Background(), "turn-err", nil, "query")
	if got != "" {
		t.Errorf("expected empty string on provider error, got %q", got)
	}
}

func TestRetrieveForTurn_EmbedError_ReturnEmpty(t *testing.T) {
	store := &fakeStore{
		files: []*longtermmemory.MemoryFile{
			makeMemoryFile("m1", "content", ""),
		},
	}
	emb := newFakeEmbedder(2)
	emb.set("content", Normalise([]float32{1, 0}))

	// Embedder that succeeds for index builds but fails for query embedding.
	failEmb := &fakeEmbedder{dim: 2, vecs: map[string][]float32{
		"content": Normalise([]float32{1, 0}),
	}, embedErr: nil}

	log := testLogger(t)
	// Use non-failing provider for HyDE, but fail on query embed.
	cfg := Config{
		HypothesisProvider: &fakeProvider{response: "hyp"},
		HypothesisCount:    1,
		TopK:               5,
	}
	eng := New(store, failEmb, cfg, log)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	eng.Start(ctx)
	waitForIndex(t, eng, 1)

	// Now make the embedder fail for new calls (hypothesis embedding).
	failEmb.embedErr = errEmbed

	got := eng.RetrieveForTurn(context.Background(), "turn-embed-err", nil, "query")
	if got != "" {
		t.Errorf("expected empty string on embed error, got %q", got)
	}
}

func TestRetrieveForTurn_ReturnsMemoryInInjection(t *testing.T) {
	const memContent = "The project uses Go 1.24 for all services."
	store := &fakeStore{
		files: []*longtermmemory.MemoryFile{
			makeMemoryFile("m1", memContent, "architecture"),
		},
	}
	emb := newFakeEmbedder(2)
	emb.set(memContent, Normalise([]float32{1, 0}))

	// HyDE returns a hypothesis that will be embedded close to m1.
	hyp := "hypothesis about go version"
	emb.set(hyp, Normalise([]float32{1, 0}))

	log := testLogger(t)
	cfg := Config{
		HypothesisProvider:   &fakeProvider{response: hyp},
		HypothesisCount:      1,
		TopK:                 5,
		HopDepth:             0,
		InjectionTokenBudget: 0,
	}
	eng := New(store, emb, cfg, log)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	eng.Start(ctx)
	waitForIndex(t, eng, 1)

	got := eng.RetrieveForTurn(context.Background(), "turn-ok", nil, "what go version?")
	if !strings.Contains(got, memContent) {
		t.Errorf("injection missing memory content; got: %q", got)
	}
	if !strings.Contains(got, "<long_term_memory>") {
		t.Errorf("injection missing wrapper tag; got: %q", got)
	}
}

func TestRetrieveForTurn_WithHistory(t *testing.T) {
	const memContent = "Tests are run with make test."
	store := &fakeStore{
		files: []*longtermmemory.MemoryFile{
			makeMemoryFile("w1", memContent, "workflow"),
		},
	}
	emb := newFakeEmbedder(2)
	emb.set(memContent, Normalise([]float32{1, 0}))
	hyp := "hypothesis about testing"
	emb.set(hyp, Normalise([]float32{1, 0}))

	log := testLogger(t)
	cfg := Config{
		HypothesisProvider:   &fakeProvider{response: hyp},
		HypothesisCount:      1,
		TopK:                 5,
		InjectionTokenBudget: 0,
	}
	eng := New(store, emb, cfg, log)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	eng.Start(ctx)
	waitForIndex(t, eng, 1)

	history := []*types.Message{
		types.NewUserMessage("how do I run tests?"),
		types.NewAssistantMessage("you can use make test"),
	}
	got := eng.RetrieveForTurn(context.Background(), "turn-hist", history, "is there a shortcut?")
	if !strings.Contains(got, memContent) {
		t.Errorf("injection missing memory with history context; got: %q", got)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

// waitForIndex polls until the engine's VectorMap has at least wantLen entries
// or the deadline is exceeded.
func waitForIndex(t *testing.T, eng *Engine, wantLen int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if eng.vm.Len() >= wantLen {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for vector index to reach size %d (got %d)", wantLen, eng.vm.Len())
}

// countingProvider wraps a fakeProvider and counts Complete calls.
type countingProvider struct {
	inner *fakeProvider
	calls *int
}

func (c *countingProvider) Complete(ctx context.Context, msgs []*types.Message) (*types.Message, error) {
	*c.calls++
	return c.inner.Complete(ctx, msgs)
}

func (c *countingProvider) StreamCompletion(_ context.Context, _ []*types.Message) (<-chan *llm.StreamChunk, error) {
	ch := make(chan *llm.StreamChunk)
	close(ch)
	return ch, nil
}

func (c *countingProvider) GetModelInfo() *types.ModelInfo { return &types.ModelInfo{Name: "counting"} }
func (c *countingProvider) GetModel() string               { return "counting" }
func (c *countingProvider) GetBaseURL() string             { return "" }
func (c *countingProvider) GetAPIKey() string              { return "" }
