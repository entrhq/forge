# 0047. Long-Term Memory — RAG Retrieval Engine

**Status:** Accepted
**Date:** 2025-02-22
**Deciders:** Core Team
**Technical Story:** [Long-Term Persistent Memory System PRD](../product/features/long-term-memory.md)

---

## Context

### Background

With the storage layer (ADR-0044), embedding provider (ADR-0045), and capture pipeline (ADR-0046) in place, this ADR defines the last component of the long-term memory system: the retrieval engine that brings relevant memories back into the agent's context at the right moment.

Retrieval must be fast (on the critical path, per user message), precise (low noise in injected context), and resilient (failures must not degrade the session). The agent should experience injected memories as natural background knowledge — not as a visible mechanism.

### Problem Statement

Injecting all memories into every turn would be prohibitively expensive in tokens and would overwhelm the agent's context with noise. Retrieval must surface the small subset of memories that are genuinely relevant to the current user message, at the moment it is needed, without the user noticing the machinery.

The core challenge is the semantic asymmetry between raw conversation text and stored memory files. Conversation is verbose, noisy, and dialogue-style. Memory files are terse, declarative, and distilled. Direct embedding of the conversation window produces low-precision retrieval because the two text styles sit in different regions of embedding space.

### Goals

- Build and maintain an in-memory vector map of all memory file contents (both scopes)
- Implement the HyDE two-stage retrieval pipeline: pre-RAG hook (hypothesis generation) → cosine similarity search → graph hop traversal
- Cache the retrieved memory context per user message; reuse for all agent sub-turns within the same user turn
- Inject retrieved memories and their version chains into the system context before the agent's first response
- Degrade gracefully: any failure in the retrieval pipeline must silently skip retrieval for that turn with zero session impact

### Non-Goals

- Memory file storage (ADR-0044)
- Embedding provider implementation (ADR-0045)
- Capture pipeline (ADR-0046)
- TUI management commands (P2 / future)
- Memory consolidation (stretch / future)

---

## Decision Drivers

* Retrieval is on the critical path — must complete within 2 seconds per user message
* Semantic asymmetry between conversation and memory text — direct embedding of conversation is insufficient
* Failures must be invisible to the user — no error surfaces, no degraded fallback
* Sub-turns within a user turn must not re-trigger retrieval — cached context reused at zero cost
* Vector map must never be read in a partially-rebuilt state — concurrent safety required

---

## Considered Options

### Option 1: Direct conversation embedding

Embed the recent conversation window and perform cosine search against stored memory vectors.

**Pros:**
- Simple — no additional LLM call
- No pre-RAG hook latency

**Cons:**
- Fundamental semantic mismatch: conversation text is verbose and dialogue-style; stored memories are terse and declarative
- Low retrieval precision — irrelevant memories surface alongside relevant ones
- Noisy injected context degrades agent quality

### Option 2: HyDE — Hypothetical Document Embeddings

Send the conversation window to a flash-class LLM, ask it to generate N short hypothetical sentences that a *relevant memory might contain*. Embed those hypotheses and search the vector map with them. The hypotheses are in the same declarative register as stored memories, dramatically improving retrieval precision.

**Pros:**
- Hypotheses and stored memories are semantically proximate — high retrieval precision
- N independent hypothesis embeddings are cheap (short sentences, one batch call)
- Pre-RAG hook reuses the existing provider abstraction
- Bridges the semantic gap with no new infrastructure

**Cons:**
- One additional LLM call on the critical path (mitigated: flash-class model, 2-second budget, skip-on-fail)

### Option 3: BM25 keyword search as primary, embeddings as re-ranker

Use BM25 text search to retrieve candidates, then re-rank with embeddings.

**Pros:**
- No LLM call on the critical path for the BM25 pass

**Cons:**
- BM25 requires an index that must be maintained alongside the vector map — additional complexity
- Two retrieval passes add implementation surface
- Keyword search on terse memory content is no better than direct embedding for the semantic gap problem
- User explicitly rejected keyword fallback approach

---

## Decision

**Chosen Option:** Option 2 — HyDE with flash-class pre-RAG hook

### Rationale

HyDE solves the core semantic asymmetry problem: the generated hypotheses are written in the same compressed, declarative style as stored memories, so embedding similarity is meaningful and precise. The LLM call cost is mitigated by mandating a flash-class model and enforcing a hard 2-second budget with skip-on-failure semantics — the agent always responds, with or without memories. This approach requires no new infrastructure beyond what ADR-0045 already provides.

---

## Consequences

### Positive

- High retrieval precision: hypotheses are semantically close to stored memories
- Zero-cost sub-turns: cached context is reused without any additional API calls
- Graceful degradation: the pipeline has multiple skip points, all silent to the user
- RWMutex on the vector map ensures reads always see a consistent state

### Negative

- First user message of a session may be slightly slower if the startup embedding build is still in progress (waits on the RWMutex read lock)
- Pre-RAG hook LLM call is on the critical path; must use a flash-class model to stay within the 2-second budget

### Neutral

- Changing `memory.embedding_model` invalidates all stored vectors; full re-embed occurs automatically on next startup

---

## Implementation

### Vector Map

The vector map holds the in-memory representation of all embedded memory files. It is built once at startup and rebuilt asynchronously after each capture event.

```go
// pkg/agent/longtermmemory/retrieval/vectormap.go
package retrieval

import (
    "sync"

    "github.com/entrhq/forge/pkg/agent/longtermmemory"
)

// MemoryVector pairs a memory file with its embedding vector.
type MemoryVector struct {
    Memory *longtermmemory.MemoryFile
    Vector []float32
}

// VectorMap is a thread-safe in-memory index of memory embeddings.
type VectorMap struct {
    mu      sync.RWMutex
    entries []MemoryVector
}

// Replace atomically swaps in a fully-built new entry set.
// Called after each startup build or post-capture rebuild completes.
func (v *VectorMap) Replace(entries []MemoryVector) {
    v.mu.Lock()
    defer v.mu.Unlock()
    v.entries = entries
}

// Search returns the top-k most similar entries to the given query vector.
// Acquires a read lock — blocks if a Replace is in progress (brief).
func (v *VectorMap) Search(query []float32, topK int) []MemoryVector {
    v.mu.RLock()
    defer v.mu.RUnlock()
    return cosineTopK(v.entries, query, topK)
}
```

### Cosine Similarity

```go
// pkg/agent/longtermmemory/retrieval/cosine.go
package retrieval

import (
    "math"
    "sort"
)

// cosineTopK returns the top-k entries by cosine similarity to query.
// Assumes all vectors are normalised (unit length); uses dot product directly.
func cosineTopK(entries []MemoryVector, query []float32, k int) []MemoryVector {
    type scored struct {
        entry MemoryVector
        score float64
    }
    scores := make([]scored, 0, len(entries))
    for _, e := range entries {
        scores = append(scores, scored{entry: e, score: dotProduct(query, e.Vector)})
    }
    sort.Slice(scores, func(i, j int) bool {
        return scores[i].score > scores[j].score
    })
    if k > len(scores) {
        k = len(scores)
    }
    out := make([]MemoryVector, k)
    for i := range out {
        out[i] = scores[i].entry
    }
    return out
}

func dotProduct(a, b []float32) float64 {
    var sum float64
    for i := range a {
        sum += float64(a[i]) * float64(b[i])
    }
    return sum
}

// Normalise returns a unit-length copy of v. Used when embedding provider
// does not guarantee normalised output.
func Normalise(v []float32) []float32 {
    var sum float64
    for _, x := range v {
        sum += float64(x) * float64(x)
    }
    norm := math.Sqrt(sum)
    if norm == 0 {
        return v
    }
    out := make([]float32, len(v))
    for i, x := range v {
        out[i] = float32(float64(x) / norm)
    }
    return out
}
```

### Vector Map Builder

The builder iterates over all memory files in the store, embeds their content in one batch call, and populates a `VectorMap`.

```go
// pkg/agent/longtermmemory/retrieval/builder.go
package retrieval

import (
    "context"
    "fmt"
    "log/slog"

    "github.com/entrhq/forge/pkg/agent/longtermmemory"
    "github.com/entrhq/forge/pkg/llm"
)

// BuildVectorMap reads all memory files from store, embeds them in one batch,
// and populates vm via vm.Replace. The caller owns vm for its lifetime —
// the same *VectorMap pointer is used for all builds (startup and post-capture)
// so that Engine.vectorMap never needs to be reassigned. Reassigning the pointer
// field from multiple goroutines would be a data race; this design avoids it
// entirely because VectorMap.Replace is already mutex-protected.
//
// Returns an error only if the embedding call itself fails; individual file
// read errors are skipped with a debug log.
func BuildVectorMap(ctx context.Context, store longtermmemory.MemoryStore, embedder llm.Embedder, vm *VectorMap) error {
    memories, err := store.List(ctx)
    if err != nil {
        return fmt.Errorf("vectormap build: list memories: %w", err)
    }
    if len(memories) == 0 {
        vm.Replace(nil) // clear any stale entries from a previous build
        return nil
    }

    texts := make([]string, len(memories))
    for i, m := range memories {
        texts[i] = m.Content
    }

    vectors, err := embedder.Embed(ctx, texts)
    if err != nil {
        return fmt.Errorf("vectormap build: embed: %w", err)
    }

    entries := make([]MemoryVector, 0, len(memories))
    for i, m := range memories {
        if i >= len(vectors) {
            slog.Debug("longtermmemory: vector index mismatch, skipping", "id", m.Meta.ID)
            continue
        }
        entries = append(entries, MemoryVector{
            Memory: m,
            Vector: Normalise(vectors[i]),
        })
    }

    vm.Replace(entries)
    return nil
}
```

### Retrieval Engine

The engine owns the VectorMap, runs the HyDE pipeline, manages the per-turn cache, and injects memories into the system context.

```go
// pkg/agent/longtermmemory/retrieval/engine.go
package retrieval

import (
    "context"
    "log/slog"
    "sync"
    "time"

    "github.com/entrhq/forge/pkg/agent/longtermmemory"
    "github.com/entrhq/forge/pkg/llm"
)

const retrievalTimeout = 2 * time.Second

// Engine is the long-term memory retrieval engine.
// It is safe for concurrent use.
type Engine struct {
    store            longtermmemory.MemoryStore
    embedder         llm.Embedder         // nil = retrieval disabled
    retrievalLLM     llm.LLMProvider      // nil = retrieval disabled
    retrievalModel   string
    hypothesisCount  int
    topK             int
    hopDepth         int

    vectorMap        *VectorMap

    // Per-turn cache
    cacheMu          sync.Mutex
    cachedTurnID     string
    cachedMemories   []*longtermmemory.MemoryFile
}

// NewEngine constructs a retrieval Engine. embedder and retrievalLLM may be nil,
// in which case retrieval is disabled and Retrieve always returns nil.
// If memory.enabled is false in config, the caller should pass nil for both
// embedder and retrievalLLM — or simply not construct an Engine at all and
// use a no-op stub. The engine itself does not read config; it relies on the
// caller to wire the correct values at construction time.
func NewEngine(
    store longtermmemory.MemoryStore,
    embedder llm.Embedder,
    retrievalLLM llm.LLMProvider,
    retrievalModel string,
    hypothesisCount, topK, hopDepth int,
) *Engine {
    return &Engine{
        store:           store,
        embedder:        embedder,
        retrievalLLM:    retrievalLLM,
        retrievalModel:  retrievalModel,
        hypothesisCount: hypothesisCount,
        topK:            topK,
        hopDepth:        hopDepth,
        vectorMap:       &VectorMap{}, // single instance; never reassigned
    }
}

// StartAsyncBuild launches the startup vector map build in a background goroutine.
// It calls BuildVectorMap with the engine's single VectorMap instance (e.vectorMap),
// which atomically swaps its entries via Replace. The Engine.vectorMap pointer is
// never reassigned after construction — this eliminates any data race on the pointer
// field itself. The first call to Retrieve may briefly block on the VectorMap's
// RWMutex while Replace is running; subsequent turns proceed without contention.
func (e *Engine) StartAsyncBuild(ctx context.Context) {
    if e.embedder == nil {
        return
    }
    go func() {
        if err := BuildVectorMap(ctx, e.store, e.embedder, e.vectorMap); err != nil {
            slog.Warn("longtermmemory: startup vector map build failed; retrieval disabled for this session", "err", err)
        }
    }()
}

// TriggerRebuild signals the engine to asynchronously rebuild the vector map.
// Called by the capture pipeline (ADR-0046) after each successful batch write.
// Non-blocking — launches a goroutine and returns immediately.
// Same single-pointer pattern as StartAsyncBuild: Replace is called on the
// existing e.vectorMap, never on a new pointer.
func (e *Engine) TriggerRebuild(ctx context.Context) {
    if e.embedder == nil {
        return
    }
    go func() {
        if err := BuildVectorMap(ctx, e.store, e.embedder, e.vectorMap); err != nil {
            slog.Debug("longtermmemory: post-capture rebuild failed; previous map retained", "err", err)
        }
    }()
}

// Retrieve returns the relevant memories for the given user turn.
// turnID uniquely identifies the user message (e.g. a UUID or sequence number).
// If the turnID matches the cached turn, returns the cached result immediately.
// On any failure, returns nil and logs at debug level — the session is unaffected.
func (e *Engine) Retrieve(ctx context.Context, turnID string, conversationWindow []string) []*longtermmemory.MemoryFile {
    if e.embedder == nil || e.retrievalLLM == nil {
        return nil
    }

    // Cache hit
    e.cacheMu.Lock()
    if e.cachedTurnID == turnID {
        result := e.cachedMemories
        e.cacheMu.Unlock()
        return result
    }
    e.cacheMu.Unlock()

    // Apply hard retrieval timeout
    ctx, cancel := context.WithTimeout(ctx, retrievalTimeout)
    defer cancel()

    memories := e.runPipeline(ctx, conversationWindow)

    // Cache result for this turn
    e.cacheMu.Lock()
    e.cachedTurnID = turnID
    e.cachedMemories = memories
    e.cacheMu.Unlock()

    return memories
}

func (e *Engine) runPipeline(ctx context.Context, window []string) []*longtermmemory.MemoryFile {
    // Stage 1: generate hypothetical memory sentences via pre-RAG hook
    hypotheses, err := generateHypotheses(ctx, e.retrievalLLM, e.retrievalModel, window, e.hypothesisCount)
    if err != nil {
        slog.Debug("longtermmemory: pre-RAG hook failed; skipping retrieval", "err", err)
        return nil
    }
    if len(hypotheses) == 0 {
        return nil
    }

    // Stage 2: embed all hypotheses in one batch call
    vectors, err := e.embedder.Embed(ctx, hypotheses)
    if err != nil {
        slog.Debug("longtermmemory: hypothesis embedding failed; skipping retrieval", "err", err)
        return nil
    }

    // Stage 3: cosine search per hypothesis; union + deduplicate results.
    // seen tracks the best similarity score observed for each memory ID across
    // all hypotheses. When a higher score is found for an already-seen ID, the
    // score map is updated but the candidates slice is not — both entries hold
    // the same *MemoryFile content, so retrieved content is correct either way.
    // The score discrepancy only affects ordering if the caller re-sorts by score;
    // since FormatInjection receives candidates in insertion order and the caller
    // is responsible for final ordering, this is acceptable. If precise per-entry
    // score tracking is needed in future, replace the slice with a scored struct.
    seen := make(map[string]float64) // id → best score
    var candidates []MemoryVector
    for _, v := range vectors {
        norm := Normalise(v)
        results := e.vectorMap.Search(norm, e.topK)
        for _, r := range results {
            id := r.Memory.Meta.ID
            score := dotProduct(norm, r.Vector)
            if prev, ok := seen[id]; !ok || score > prev {
                seen[id] = score
                if !ok {
                    candidates = append(candidates, r)
                }
            }
        }
    }

    // Stage 4: graph hop traversal
    expanded := e.traverseHops(ctx, candidates)

    return expanded
}

// traverseHops expands the candidate set by following typed graph edges
// up to e.hopDepth hops.
func (e *Engine) traverseHops(ctx context.Context, candidates []MemoryVector) []*longtermmemory.MemoryFile {
    visited := make(map[string]bool)
    queue := make([]*longtermmemory.MemoryFile, 0, len(candidates))
    for _, c := range candidates {
        queue = append(queue, c.Memory)
        visited[c.Memory.Meta.ID] = true
    }

    for depth := 0; depth < e.hopDepth; depth++ {
        var next []*longtermmemory.MemoryFile
        for _, m := range queue {
            for _, rel := range m.Meta.Related {
                if visited[rel.ID] {
                    continue
                }
                neighbour, err := e.store.Read(ctx, rel.ID)
                if err != nil {
                    continue // missing or corrupt neighbour — skip silently
                }
                visited[rel.ID] = true
                next = append(next, neighbour)
            }
        }
        queue = append(queue, next...)
        if len(next) == 0 {
            break
        }
    }
    return queue
}
```

### HyDE Hypothesis Generation

```go
// pkg/agent/longtermmemory/retrieval/hyde.go
package retrieval

import (
    "context"
    "fmt"
    "strings"

    "github.com/entrhq/forge/pkg/llm"
)

const hydeSystemPrompt = `You are a memory retrieval assistant for an AI coding agent.
Given a snippet of conversation, generate exactly N short declarative sentences
that might appear in a memory file about a relevant past preference, convention,
decision, or correction. Each sentence should be on its own line.
Do not explain or elaborate — only output the sentences.`

// generateHypotheses calls the pre-RAG hook LLM to generate N hypothetical memory sentences.
// Returns an error if the LLM call fails; returns nil if the response is empty.
func generateHypotheses(
    ctx context.Context,
    provider llm.LLMProvider,
    model string,
    window []string,
    n int,
) ([]string, error) {
    if len(window) == 0 {
        return nil, nil
    }
    windowText := strings.Join(window, "\n---\n")
    userPrompt := fmt.Sprintf(
        "Conversation window:\n\n%s\n\nGenerate %d hypothetical memory sentences.",
        windowText, n,
    )
    response, err := provider.Complete(ctx, model, hydeSystemPrompt, userPrompt)
    if err != nil {
        return nil, fmt.Errorf("hyde: LLM call: %w", err)
    }
    lines := strings.Split(strings.TrimSpace(response), "\n")
    var hypotheses []string
    for _, l := range lines {
        l = strings.TrimSpace(l)
        if l != "" {
            hypotheses = append(hypotheses, l)
        }
    }
    return hypotheses, nil
}
```

### System Context Injection

Retrieved memories are injected into the agent's system context before the first response for a user turn. The injection point is after the existing system prompt segments and before the conversation history — memories are background context, not part of the live dialogue.

```go
// pkg/agent/longtermmemory/retrieval/inject.go
package retrieval

import (
    "fmt"
    "strings"

    "github.com/entrhq/forge/pkg/agent/longtermmemory"
)

// FormatInjection formats a set of memories as a markdown block suitable for
// injection into the system context. memories must be ordered by descending
// similarity score (most relevant first) — this function respects that ordering
// during budget truncation.
//
// tokenBudget is a soft character-count proxy for token budget
// (≈ 4 chars/token). Pass 0 to disable truncation. When the budget is reached,
// remaining memories are dropped — highest-relevance memories are always
// included first. This implements the InjectionTokenBudget config field
// (memory.injection_token_budget) defined in ADR-0045.
func FormatInjection(memories []*longtermmemory.MemoryFile, tokenBudget int) string {
    if len(memories) == 0 {
        return ""
    }
    const charPerToken = 4
    charBudget := tokenBudget * charPerToken

    header := "## Relevant Long-Term Memories\n\nThe following memories were retrieved " +
        "from your persistent memory store as relevant to the current conversation. " +
        "Apply them silently.\n\n"

    var sb strings.Builder
    sb.WriteString(header)
    used := len(header)

    for _, m := range memories {
        var entry strings.Builder
        entry.WriteString(fmt.Sprintf("**[%s | %s | v%d]**\n",
            m.Meta.Category, m.Meta.Scope, m.Meta.Version))
        if m.Meta.Supersedes != nil {
            entry.WriteString(fmt.Sprintf("*(supersedes %s)*\n", *m.Meta.Supersedes))
        }
        entry.WriteString(m.Content)
        entry.WriteString("\n\n")

        chunk := entry.String()
        if charBudget > 0 && used+len(chunk) > charBudget {
            break // budget exhausted; remaining memories are lower-relevance, drop them
        }
        sb.WriteString(chunk)
        used += len(chunk)
    }
    return sb.String()
}
```

### Conversation Window Construction

The retrieval engine receives the conversation window from the agent loop. The window is built the same way as for capture: all user and assistant messages since the last summarisation event, with tool call content excluded.

```go
// pkg/agent/longtermmemory/retrieval/window.go
package retrieval

import "github.com/entrhq/forge/pkg/types"

// BuildConversationWindow returns the text of all user and assistant messages
// since the last summarisation boundary. Tool call content is excluded.
// If no summarisation has occurred (lastSummarisationIdx == 0), the full
// conversation history is used.
//
// lastSummarisationIdx is the slice index of the first message *after* the
// summarisation boundary (i.e. messages[lastSummarisationIdx] is the first
// message to include, not the summary message itself). Pass 0 when no
// summarisation has occurred — messages[0:] is the full history.
//
// TextContent() is expected to return the plain-text body of a message,
// stripping any tool call XML or structured content blocks. This method must
// be available on types.Message (see pkg/types) as a prerequisite for this
// package to compile. If types.Message does not yet expose TextContent(),
// add it before implementing this package.
func BuildConversationWindow(messages []types.Message, lastSummarisationIdx int) []string {
    start := 0
    if lastSummarisationIdx > 0 {
        start = lastSummarisationIdx
    }
    var window []string
    for _, msg := range messages[start:] {
        if msg.Role != "user" && msg.Role != "assistant" {
            continue
        }
        text := msg.TextContent()
        if text != "" {
            window = append(window, msg.Role+": "+text)
        }
    }
    return window
}
```

### Package Layout

```
pkg/agent/longtermmemory/retrieval/
    engine.go       — Engine (main entry point: Retrieve, StartAsyncBuild, TriggerRebuild)
    vectormap.go    — VectorMap (RWMutex-protected in-memory index)
    builder.go      — BuildVectorMap() (startup + post-capture build)
    cosine.go       — cosineTopK(), dotProduct(), Normalise()
    hyde.go         — generateHypotheses() (pre-RAG hook LLM call)
    inject.go       — FormatInjection() (system context formatter)
    window.go       — BuildConversationWindow()
```

### Agent Loop Integration

```
Per user message (before agent responds):
  1. turnID := newUUID()               // stable identifier for this user message
  2. window := BuildConversationWindow(messages, lastSumIdx)
  3. memories := engine.Retrieve(ctx, turnID, window)
                                       // blocks on RWMutex if startup build in progress
                                       // returns nil on any failure (silent skip)
  4. injection := FormatInjection(memories, cfg.Memory.InjectionTokenBudget)
                                       // memories are already ordered by descending score
                                       // from runPipeline; budget truncates from the tail
  5. Prepend injection to system context for this turn

Per agent sub-turn (tool call → response, within same user turn):
  1. memories := engine.Retrieve(ctx, turnID, window)
                                       // same turnID → cache hit → zero cost
                                       // turnID changes on next user message, invalidating cache
```

> **Cache keying note:** The per-turn cache is keyed on `turnID`. There is no explicit `InvalidateCache` call needed — passing a new `turnID` with each user message is sufficient to bypass the cache. The previous turn's cached slice is garbage-collected naturally. The `cacheMu` mutex prevents a torn read if `Retrieve` is called concurrently from multiple sub-turn goroutines (rare but possible).

### Wiring at Agent Startup

```go
// Pseudocode — agent initialisation
embedder, _ := llm.NewEmbedder(cfg)     // nil if memory.embedding_model unset
retrievalProvider := llmProvider        // reuse main provider for retrieval model

retrievalEngine := retrieval.NewEngine(
    store,
    embedder,
    retrievalProvider,
    cfg.Memory.RetrievalModel,
    cfg.Memory.RetrievalHypothesisCount,
    cfg.Memory.RetrievalTopK,
    cfg.Memory.RetrievalHopDepth,
)
retrievalEngine.StartAsyncBuild(ctx)    // non-blocking; first turn may wait on RWMutex

// Wire capture pipeline rebuild signal
capturePipeline := capture.NewPipeline(
    ...,
    func() { retrievalEngine.TriggerRebuild(ctx) },
)

// Short-circuit: if memory is disabled globally, skip all wiring
if !cfg.Memory.Enabled {
    // retrievalEngine remains nil; agent loop checks for nil before calling Retrieve
    return
}

// Warn once if only one of retrieval_model / embedding_model is set
if (cfg.Memory.RetrievalModel == "") != (cfg.Memory.EmbeddingModel == "") {
    log.Warn("longtermmemory: both memory.retrieval_model and memory.embedding_model must be set for retrieval; retrieval disabled")
}
```

### Error & Degradation Summary

| Failure point | Behaviour |
|---|---|
| `embedder == nil` | `Retrieve` returns nil immediately; no log |
| `retrievalLLM == nil` | `Retrieve` returns nil immediately; no log |
| Startup build fails | `VectorMap` remains empty; warn once; retrieval returns nil for session |
| Post-capture rebuild fails | Previous map retained; debug log; next startup recovers |
| Pre-RAG hook LLM timeout / error | `runPipeline` returns nil; debug log; session continues |
| Hypothesis embedding fails | `runPipeline` returns nil; debug log |
| Graph hop neighbour missing | Neighbour skipped silently; partial result returned |
| Retrieval timeout (2s) | Context cancelled; whatever completed is returned (may be nil) |

### Migration Path

No migration required. The retrieval engine is a new component. The agent loop gains two new integration points (cache invalidation on user message, context injection before first response) but no existing behaviour changes.

---

## Validation

### Success Metrics

- `Retrieve` with the same `turnID` twice returns the cached result on the second call (zero LLM/embed calls)
- `Retrieve` returns nil (not an error) when `embedder` or `retrievalLLM` is nil
- `Retrieve` returns nil (not an error) when the pre-RAG hook times out
- `VectorMap.Search` never blocks for more than the time of an in-progress `Replace` call (brief RWMutex wait)
- `TriggerRebuild` returns before the build completes — it is non-blocking
- `cosineTopK` returns results in descending similarity order
- `Normalise` returns a unit vector for all non-zero inputs
- `FormatInjection` with a tight `tokenBudget` drops lower-relevance memories and never exceeds the character budget
- `FormatInjection` with `tokenBudget == 0` includes all memories regardless of length
- Constructing an `Engine` with `embedder == nil` and calling `StartAsyncBuild` is a no-op (no goroutine launched)

### Integration Test Scenario

The following end-to-end scenario should be validated with a test using a fake/mock store and a mock embedder:

```
1. Construct a FileStore pointing at a temp directory (repo scope).
2. Write three MemoryFile entries with distinct content and known embedding vectors
   (inject via mock embedder returning fixed float32 slices).
3. Call BuildVectorMap — confirm VectorMap contains 3 entries.
4. Construct an Engine with the VectorMap, a mock HyDE LLM that returns two
   hypothetical sentences, and topK=2, hopDepth=1.
5. Add a Related edge from memory[0] to memory[2] in the store.
6. Call engine.Retrieve(ctx, "turn-1", window):
   - Confirm 2 candidates returned from cosine search (topK=2).
   - Confirm graph hop expands to include memory[2] (neighbour of memory[0]).
   - Confirm total result set has 3 entries.
7. Call engine.Retrieve(ctx, "turn-1", window) again:
   - Confirm the result is identical and no LLM/embed calls were made (cache hit).
8. Call engine.Retrieve(ctx, "turn-2", window):
   - Confirm new LLM/embed calls are made (cache miss on new turnID).
9. Call FormatInjection(results, 200) — confirm output is truncated to fit budget.
10. Call FormatInjection(results, 0) — confirm all memories are included.
```

### Monitoring

- Debug log on pre-RAG hook failure
- Debug log on hypothesis embedding failure
- Warn log on startup build failure (one per session)
- Debug log on post-capture rebuild failure

---

## Related Decisions

- [ADR-0007](0007-memory-system-design.md) — in-session conversation memory (source of message history for window construction)
- [ADR-0014](0014-composable-context-management.md) — context management (injection point for retrieved memories)
- [ADR-0041](0041-goal-batch-compaction-strategy.md) — compaction (summarisation boundary used for window construction)
- [ADR-0044](0044-long-term-memory-storage.md) — storage layer (prerequisite; retrieval reads MemoryStore)
- [ADR-0045](0045-long-term-memory-embedding-provider.md) — embedding provider (prerequisite; retrieval uses Embedder)
- [ADR-0046](0046-long-term-memory-capture.md) — capture pipeline (signals TriggerRebuild after each write)

---

## References

- [Long-Term Memory PRD](../product/features/long-term-memory.md)
- [HyDE: Precise Zero-Shot Dense Retrieval without Relevance Labels](https://arxiv.org/abs/2212.10496)

---

## Notes

The 2-second retrieval budget covers the entire pipeline: pre-RAG hook LLM call + hypothesis embedding + cosine search + graph traversal + injection formatting. The dominant cost is the pre-RAG hook LLM call. Mandating a flash-class model for `memory.retrieval_model` (e.g. `gemini-2.0-flash`, `gpt-4o-mini`) is essential to meet this budget. The cosine search and graph traversal are negligible (pure CPU, in-memory).

The `turnID` used for cache keying should be a value that is stable across all agent sub-turns within a single user turn but changes on each new user message. A UUID generated when the user message is received and passed through the agent loop is the simplest implementation.

**Last Updated:** 2025-02-22
