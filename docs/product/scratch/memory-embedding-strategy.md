# Memory Embedding Strategy

## Decision

Embeddings are **not persisted to disk**. They are held in-memory and (re)computed on two triggers:

1. **At agent startup** — embed the full memory store once, synchronously, before the first turn begins
2. **After each capture event** — whenever the classifier writes one or more new memory files, re-embed the full store; this happens asynchronously on the capture goroutine and does not block the main agent loop

Between these two events, the in-memory embedding cache is static and all retrieval reads from it directly.

---

## When Embeddings Are Computed

### Trigger 1: Startup

- On agent initialisation, scan both memory stores (`.forge/memory/` and `~/.forge/memory/`)
- Embed all memory files in a single batched API call
- Store result as an in-memory map: `memoryID → vector`
- Agent does not begin the first turn until this completes
- Cost: negligible (e.g. 1,000 memories × 150 tokens = 150,000 tokens = $0.003 at text-embedding-3-small pricing)
- Latency: fast enough to be imperceptible on startup (< 500ms for typical store sizes)

### Trigger 2: Post-Capture

- After the classifier writes new memory files (cadence or compaction trigger), re-embed the full store
- This runs on the existing async capture goroutine — it does not block the main agent loop
- The in-memory map is replaced atomically once the new embedding batch completes
- Any retrieval reads that happen during the re-embed window use the previous (slightly stale) map — this is acceptable

---

## What Is Not Done

- **No `.embed` sidecar files** — embeddings are never written to disk
- **No `embedding_model` field in YAML front-matter** — model provenance is not tracked per-file
- **No incremental update** — re-embed always covers the full store; partial updates add complexity with minimal benefit at typical store sizes

---

## Handling Embedding Model Changes

When a user changes `memory.embedding_model`, the in-memory cache is invalidated naturally on the next startup — the full store is re-embedded with the new model. No migration tooling is required. Old vectors are simply discarded; no stale embeddings persist across restarts.

---

## Cost Model

Re-embedding is proportional to **store size × capture frequency**, not **store size × turn frequency**. Retrieval (the per-turn hot path) never triggers an embedding API call — it reads from the in-memory map.

| Store Size | Tokens Per Full Re-Embed | Cost (text-embedding-3-small) |
|---|---|---|
| 100 memories | ~15,000 | $0.0003 |
| 500 memories | ~75,000 | $0.0015 |
| 1,000 memories | ~150,000 | $0.003 |
| 2,000 memories | ~300,000 | $0.006 |

At the default capture cadence of every 5 turns, heavy users (5,000 turns/month) trigger ~1,000 re-embeds/month. At 1,000 memories that is ~$3/month — negligible.

---

## Retrieval Path (Per Turn)

No embedding API calls are made during retrieval. The flow is:

1. HyDE hook generates N hypothetical sentences (flash LLM call — the only API call on the critical path)
2. Each hypothesis is embedded using the in-memory embedding model client (single lightweight call per hypothesis, not a store scan)
3. Hypothesis vectors are compared against the in-memory `memoryID → vector` map using cosine similarity
4. Top-k results returned, graph traversal applied, memories injected into context

The in-memory map makes step 3 a pure CPU operation — no I/O, no API calls.

---

## Summary

| Property | Value |
|---|---|
| Embeddings stored on disk? | No |
| Embedding model tracked in memory files? | No |
| When computed | Startup + after each capture event (async) |
| Per-turn API calls for retrieval | Zero (in-memory cosine similarity only) |
| Model change migration | Automatic on next startup |
| Complexity | Minimal — one in-memory map, two rebuild triggers |
