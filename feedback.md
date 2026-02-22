# ADR-0045 Implementation Review: Long-Term Memory — Embedding Provider

**Review Date:** 2025-07-11 (Updated)
**ADR:** [0045-long-term-memory-embedding-provider.md](docs/adr/0045-long-term-memory-embedding-provider.md)
**Status:** ✅ Complete — All critical gaps resolved

---

## Summary

All items from the initial review have been addressed. The embedding provider is now fully implemented **and** wired into the agent lifecycle in both the TUI and headless entry points. The session-start misconfiguration warning is also present. Only minor documentation alignment items remain.

---

## What Is Complete ✅

### 1. `Embedder` Interface (`pkg/llm/embedder.go`)

Exact match to ADR spec:

```go
type Embedder interface {
    Embed(ctx context.Context, inputs []string) ([][]float32, error)
    Model() string
}
```

---

### 2. Embedding Sub-Package (`pkg/llm/embedding/embedding.go`)

Fully implements the ADR design: `Provider` struct, `NewProvider`, `WithBaseURL`, `WithHTTPClient` functional options, defensive index-based reordering in `Embed()`, and `DefaultBaseURL = "https://api.openai.com/v1"`.

---

### 3. Factory (`pkg/llm/embedder_factory.go`)

Graceful `(nil, nil)` degradation when config is nil, memory is disabled, or embedding model is empty. The nil-embedder contract is upheld.

> **Note — Signature Deviation (intentional):** The ADR specifies `func NewEmbedder(cfg *config.Config) (Embedder, error)`. The implementation uses `func NewEmbedder(cfg *config.MemorySection, apiKey string, opts ...embedding.ProviderOption) (Embedder, error)`. This is a deliberate improvement — it reduces coupling to the root config and improves testability. The ADR should be updated to reflect this signature.

---

### 4. Configuration (`pkg/config/memory.go`)

All required config fields are present with thread-safe getters/setters and correct defaults.

> **Note — Field Rename:** `retrieval_model` in the ADR was renamed to `hypothesis_model` in the implementation (reflected as `GetHypothesisModel()`). The ADR text still says `retrieval_model`. The ADR should be updated.

---

### 5. Mock for Testing (`pkg/llm/mock_embedder_test.go`)

Present and correct. Includes compile-time interface check `var _ llm.Embedder = (*MockEmbedder)(nil)`.

---

### 6. Agent Wiring — NOW COMPLETE ✅

Previously the most critical gap. Now resolved in both entry points:

**`cmd/forge/main.go` and `cmd/forge/headless.go`:**
```go
var embedder llm.Embedder
if memoryCfg := appconfig.GetMemory(); memoryCfg != nil {
    // misconfiguration warning (see below)
    embedder, embedErr = llm.NewEmbedder(memoryCfg, provider.GetAPIKey())
    if embedErr != nil {
        log.Printf("warning: memory retrieval disabled: embedding provider error: %v", embedErr)
        embedder = nil
    }
}
// ...
agent.WithEmbedder(embedder),
```

`DefaultAgent` in `pkg/agent/default.go` now has an `embedder llm.Embedder` field and a `WithEmbedder(e llm.Embedder) AgentOption` function. The nil-is-valid contract is documented on the field.

> **Note:** `cmd/forge-headless/` does not wire the embedder, but this binary is deprecated and can be disregarded.

---

### 7. Session-Start Misconfiguration Warning — NOW COMPLETE ✅

Previously missing. Now implemented in both `cmd/forge/main.go` and `cmd/forge/headless.go`:

```go
if hypothesisModel != "" && embeddingModel == "" {
    log.Println("warning: memory.hypothesis_model is set but memory.embedding_model is empty — retrieval is disabled")
} else if embeddingModel != "" && hypothesisModel == "" {
    log.Println("warning: memory.embedding_model is set but memory.hypothesis_model is empty — retrieval is disabled")
}
```

This correctly guides users who partially configure the retrieval system.

---

### 8. Test Coverage

- `pkg/llm/embedding/embedding_test.go` — httptest mock, index reordering, error propagation
- `pkg/llm/embedder_factory_test.go` — nil/disabled graceful degradation, successful construction
- `pkg/config/memory_test.go` — defaults, serialization, type coercion for all fields

Coverage is solid across the library layer.

---

## Remaining Items (Documentation Only)

These are not functional gaps — the code is correct. The ADR document just needs to be brought in sync:

| Item | Current ADR Text | Actual Implementation |
|---|---|---|
| Factory signature | `func NewEmbedder(cfg *config.Config) (Embedder, error)` | `func NewEmbedder(cfg *config.MemorySection, apiKey string, opts ...embedding.ProviderOption) (Embedder, error)` |
| Config field name | `retrieval_model` | `hypothesis_model` (`GetHypothesisModel()`) |

---

## Overall Findings Table

| ADR Requirement | Status | Notes |
|---|---|---|
| `Embedder` interface in `pkg/llm/` | ✅ Complete | Exact match to spec |
| `pkg/llm/embedding/` sub-package | ✅ Complete | All conventions followed |
| `embedding.go` — Provider, NewProvider, Embed, options | ✅ Complete | Defensive index reordering is a bonus |
| `embedding_test.go` — httptest mock, no real calls | ✅ Complete | Good coverage |
| `NewEmbedder` factory with graceful nil return | ✅ Complete | Signature improved but diverges from ADR doc |
| `MemoryConfig` fields | ✅ Complete | Thread-safe, correct defaults; `retrieval_model` → `hypothesis_model` |
| `MockEmbedder` with compile-time check | ✅ Complete | Exact match to spec |
| Wiring into agent startup | ✅ Complete | Both `cmd/forge/main.go` and `cmd/forge/headless.go` |
| `WithEmbedder()` agent option | ✅ Complete | `DefaultAgent` field + option function |
| Session-start misconfiguration warning | ✅ Complete | Checks both `hypothesis_model` and `embedding_model` |
| Factory signature matches ADR | ⚠️ Doc gap | Better design; ADR text needs updating |
| `retrieval_model` → `hypothesis_model` rename | ⚠️ Doc gap | ADR text needs updating |
| Retrieval engine (ADR-0047) | ⏳ Not started | Out of scope for ADR-0045 |

---

## Conclusion

ADR-0045 is functionally complete. The embedding provider is built, configured, wired into the agent at startup in all active entry points, and the misconfiguration warning guards against partial setup. The two remaining items are purely documentation alignment — the ADR text should be updated to reflect the improved factory signature and the `hypothesis_model` field rename.
