# 0045. Long-Term Memory — Embedding Provider

**Status:** Accepted
**Date:** 2025-02-22
**Deciders:** Core Team
**Technical Story:** [Long-Term Persistent Memory System PRD](../product/features/long-term-memory.md)

---

## Context

### Background

Forge's provider abstraction layer (ADR-0003) currently exposes a single capability: generating chat completions via the `Provider` interface, implemented in `pkg/llm/openai/`. That package communicates with any OpenAI-compatible HTTP API via a configurable `baseURL`, which transparently supports OpenAI, Azure OpenAI, local models (Ollama, LM Studio), and OpenAI-compatible proxies.

Long-term memory retrieval (ADR-0047) requires a second provider capability: computing dense vector embeddings for short text inputs. Embeddings are needed in two places:

1. **Startup / post-capture** — embed the full memory store to build the in-memory vector map (async, off critical path)
2. **Per user message** — embed each HyDE hypothesis sentence to query the vector map (on critical path, but cheap: N short sentences)

All major embedding providers expose an OpenAI-compatible embeddings endpoint (`POST /v1/embeddings`). This includes OpenAI itself, Google (via the Gemini OpenAI-compatible endpoint), Mistral, Cohere, and self-hosted models via Ollama or LiteLLM. Implementing a single OpenAI-compatible embedding client with a configurable `baseURL` gives full provider flexibility without requiring provider-specific code for each vendor.

### Problem Statement

There is no embedding capability in the current provider abstraction. Adding it must:

- Not break existing `Provider` implementations (chat completions must be unaffected)
- Not couple the embedding client to any specific vendor — users should be able to point it at any OpenAI-compatible embedding endpoint
- Follow the same structural pattern as `pkg/llm/openai/` so the codebase stays consistent

### Goals

- Define a standalone `Embedder` interface in `pkg/llm/`
- Implement it in a new `pkg/llm/embedding/` sub-package using the OpenAI-compatible embeddings HTTP endpoint
- Wire `memory.embedding_model` and `memory.embedding_base_url` through the settings system
- Keep `Provider` (chat completions) entirely unchanged
- Return `nil` (not an error) when embedding is unconfigured, so callers treat it as "retrieval disabled"

### Non-Goals

- In-memory vector map construction and management (ADR-0047)
- Memory file storage (ADR-0044)
- Capture pipeline (ADR-0046)
- Vendor-specific embedding SDK integrations (everything goes through the OpenAI-compatible HTTP endpoint)
- Batch optimisation beyond what the provider API natively supports

---

## Decision Drivers

* **Consistency** — the embedding sub-package must look and feel like `pkg/llm/openai/`; same option pattern, same HTTP client approach
* **Provider agnosticism** — a single OpenAI-compatible client with configurable `baseURL` covers all realistic embedding providers
* **Backward compatibility** — `Provider` interface and all existing implementations are untouched
* **Testability** — `Embedder` must be an interface, independently mockable
* **Graceful degradation** — unconfigured embedding model returns `nil`, which the retrieval engine treats as disabled

---

## Considered Options

### Option A: Extend `Provider` with an optional `Embed()` method (interface assertion)

Add `Embed()` to a new `EmbeddingProvider` interface. Callers type-assert at runtime.

**Rejected:** Interface assertions are fragile and hard to mock. Providers that don't offer embeddings must either panic or return a stub error. Mixes two distinct API surface areas into one object.

### Option B: Vendor-specific embedder per provider (OpenAI SDK, Gemini SDK, …)

Implement `openAIEmbedder`, `geminiEmbedder`, etc., each using the vendor's SDK.

**Rejected:** All major providers expose the same OpenAI-compatible `/v1/embeddings` endpoint. Maintaining separate SDK-backed implementations is unnecessary complexity. The human review explicitly flagged the Gemini-specific implementation in the earlier draft as wrong.

### Option C: Single OpenAI-compatible embedding sub-package with configurable `baseURL` (chosen)

Define `Embedder` interface in `pkg/llm/`. Implement in `pkg/llm/embedding/` using direct HTTP to `POST {baseURL}/embeddings`, following the same pattern as `pkg/llm/openai/`. Users configure `memory.embedding_base_url` to target any provider.

**Chosen:** Consistent with existing codebase structure, provider-agnostic, simple to test, minimal new surface area.

---

## Decision

**Chosen Option:** Option C — single OpenAI-compatible embedding sub-package

### Rationale

The existing `pkg/llm/openai/` package already demonstrates the right pattern: HTTP client, configurable base URL via functional options, API key from config or environment variable. Duplicating that pattern for embeddings gives full provider flexibility with zero vendor-specific coupling. The OpenAI embeddings endpoint (`POST /v1/embeddings`) is an industry-standard format that every major provider either implements natively or exposes via a compatibility layer.

---

## Consequences

### Positive

- Zero changes to `Provider` interface or any existing implementation
- Any OpenAI-compatible embedding endpoint (OpenAI, Gemini, Mistral, Ollama, LiteLLM, …) works without code changes — just change `memory.embedding_base_url`
- Consistent `functional-options` construction pattern across `pkg/llm/` sub-packages
- `nil` embedder is a first-class state: retrieval disabled without error propagation

### Negative

- Two config keys to set for retrieval (`memory.embedding_model` + `memory.embedding_base_url`); slightly more onboarding friction than auto-detecting from the chat provider config

### Neutral

- Embedding and chat model can be on different providers (e.g., GPT-4o for chat, `text-embedding-3-small` via a different base URL). This is a feature, not a bug.

---

## Implementation

### Embedder Interface

```go
// pkg/llm/embedder.go
package llm

import "context"

// Embedder computes dense vector embeddings for text inputs.
// Implementations must be safe for concurrent use.
//
// A nil Embedder is valid and means retrieval is disabled — callers must
// check for nil before calling any method.
type Embedder interface {
    // Embed returns a normalised float32 embedding vector for each input string.
    // The order of output vectors corresponds exactly to the order of inputs.
    // Returns an error if the provider call fails or any input exceeds the
    // model's token limit.
    Embed(ctx context.Context, inputs []string) ([][]float32, error)

    // Model returns the embedding model identifier string.
    Model() string
}
```

### Embedding Sub-Package

Follows the same structural conventions as `pkg/llm/openai/`:

```
pkg/llm/embedding/
    embedding.go        — Provider struct, NewProvider(), functional options
    embedding_test.go   — unit tests (http.Handler mock; no real API calls)
```

```go
// pkg/llm/embedding/embedding.go
package embedding

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"

    "github.com/entrhq/forge/pkg/llm"
)

const DefaultBaseURL = "https://api.openai.com/v1"

// Provider implements llm.Embedder using the OpenAI-compatible embeddings
// HTTP endpoint (POST {baseURL}/embeddings). Any provider that exposes this
// endpoint format is supported via WithBaseURL.
type Provider struct {
    httpClient *http.Client
    apiKey     string
    baseURL    string
    model      string
}

// ProviderOption configures a Provider.
type ProviderOption func(*Provider)

// WithBaseURL overrides the API base URL. Use this to target any
// OpenAI-compatible embedding endpoint:
//
//	embedding.NewProvider(key, model,
//	    embedding.WithBaseURL("https://generativelanguage.googleapis.com/v1beta/openai"))
func WithBaseURL(url string) ProviderOption {
    return func(p *Provider) { p.baseURL = url }
}

// WithHTTPClient overrides the default http.Client (useful for testing).
func WithHTTPClient(c *http.Client) ProviderOption {
    return func(p *Provider) { p.httpClient = c }
}

// NewProvider creates an embedding Provider.
//
// If apiKey is empty, OPENAI_API_KEY is read from the environment.
// If baseURL is not set via WithBaseURL, OPENAI_BASE_URL is checked,
// falling back to DefaultBaseURL (https://api.openai.com/v1).
//
// model is the embedding model identifier (e.g. "text-embedding-3-small").
func NewProvider(apiKey, model string, opts ...ProviderOption) (*Provider, error) {
    if apiKey == "" {
        apiKey = os.Getenv("OPENAI_API_KEY")
    }
    if apiKey == "" {
        return nil, fmt.Errorf("embedding: API key required (set OPENAI_API_KEY or pass via config)")
    }
    if model == "" {
        return nil, fmt.Errorf("embedding: model name must not be empty")
    }
    p := &Provider{
        httpClient: &http.Client{},
        apiKey:     apiKey,
        baseURL:    DefaultBaseURL,
        model:      model,
    }
    for _, opt := range opts {
        opt(p)
    }
    // Environment variable fallback for base URL (mirrors pkg/llm/openai pattern)
    if p.baseURL == DefaultBaseURL {
        if env := os.Getenv("OPENAI_BASE_URL"); env != "" {
            p.baseURL = env
        }
    }
    return p, nil
}

func (p *Provider) Model() string { return p.model }

// Embed sends a batch embedding request and returns one vector per input.
// The output order matches the input order.
func (p *Provider) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
    body, err := json.Marshal(map[string]any{
        "model": p.model,
        "input": inputs,
    })
    if err != nil {
        return nil, fmt.Errorf("embedding: marshal request: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/embeddings", bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("embedding: build request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+p.apiKey)

    resp, err := p.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("embedding: http: %w", err)
    }
    defer resp.Body.Close()

    raw, _ := io.ReadAll(resp.Body)
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("embedding: provider returned %d: %s", resp.StatusCode, raw)
    }

    var result struct {
        Data []struct {
            Embedding []float32 `json:"embedding"`
            Index     int       `json:"index"`
        } `json:"data"`
    }
    if err := json.Unmarshal(raw, &result); err != nil {
        return nil, fmt.Errorf("embedding: decode response: %w", err)
    }

    // Re-order by index — OpenAI guarantees order but be defensive
    out := make([][]float32, len(inputs))
    for _, d := range result.Data {
        if d.Index < len(out) {
            out[d.Index] = d.Embedding
        }
    }
    return out, nil
}
```

### Factory

```go
// pkg/llm/embedder_factory.go
package llm

import (
    "fmt"

    "github.com/entrhq/forge/pkg/config"
    "github.com/entrhq/forge/pkg/llm/embedding"
)

// NewEmbedder constructs an Embedder from the memory config.
//
// Returns (nil, nil) when memory.embedding_model is empty — this is not an
// error; callers treat a nil Embedder as "retrieval disabled".
//
// The base URL defaults to DefaultBaseURL (OpenAI). Set memory.embedding_base_url
// to target any OpenAI-compatible embedding endpoint (Gemini, Mistral, Ollama, …).
func NewEmbedder(cfg *config.Config) (Embedder, error) {
    model := cfg.Memory.EmbeddingModel
    if model == "" {
        return nil, nil // retrieval disabled; not an error
    }
    var opts []embedding.ProviderOption
    if cfg.Memory.EmbeddingBaseURL != "" {
        opts = append(opts, embedding.WithBaseURL(cfg.Memory.EmbeddingBaseURL))
    }
    p, err := embedding.NewProvider(cfg.APIKey, model, opts...)
    if err != nil {
        return nil, fmt.Errorf("llm: embedding provider: %w", err)
    }
    return p, nil
}
```

### Configuration

New fields added to `MemoryConfig`:

```go
// pkg/config/config.go (addition)
type MemoryConfig struct {
    Enabled                  bool   `yaml:"enabled"`
    ClassifierModel          string `yaml:"classifier_model"`
    RetrievalModel           string `yaml:"retrieval_model"`
    EmbeddingModel           string `yaml:"embedding_model"`
    EmbeddingBaseURL         string `yaml:"embedding_base_url"`   // optional; defaults to OpenAI
    RetrievalTopK            int    `yaml:"retrieval_top_k"`
    RetrievalHopDepth        int    `yaml:"retrieval_hop_depth"`
    RetrievalHypothesisCount int    `yaml:"retrieval_hypothesis_count"`
    InjectionTokenBudget     int    `yaml:"injection_token_budget"` // P1; 0 = unlimited
}
```

Defaults applied at config load time:

| Key | Default | Notes |
|---|---|---|
| `memory.enabled` | `true` | |
| `memory.classifier_model` | inherits summarisation model | high-reasoning model recommended |
| `memory.retrieval_model` | `""` | empty disables retrieval |
| `memory.embedding_model` | `""` | empty disables retrieval |
| `memory.embedding_base_url` | `""` | falls back to `https://api.openai.com/v1` |
| `memory.retrieval_top_k` | `10` | |
| `memory.retrieval_hop_depth` | `1` | |
| `memory.retrieval_hypothesis_count` | `5` | |
| `memory.injection_token_budget` | `0` | 0 = no limit (P1 feature) |

### Token Budget (P1)

`memory.injection_token_budget` controls the maximum number of tokens of memory content that may be injected into the system prompt per turn. When the budget is set (> 0):

1. Retrieved memories are ordered by **descending cosine similarity score** (most relevant first)
2. Memories are appended to the injection block one at a time until the next memory would exceed the remaining budget
3. The truncation point is at a whole-memory boundary (memories are never split mid-content)
4. The injected block carries a header noting how many memories were included and how many were omitted due to budget constraints

This ensures the most relevant memories always make it into context, and the agent is never surprised by an unexpectedly large context expansion.

### Wiring at Agent Startup

```go
// In agent initialisation (pkg/agent/agent.go or equivalent)
embedder, err := llm.NewEmbedder(cfg)
if err != nil {
    // Provider construction failed (bad API key, unknown base URL format, etc.)
    // Warn and disable retrieval — do not abort startup.
    slog.Warn("memory retrieval disabled: embedding provider error", "err", err)
    embedder = nil
}
// embedder may be nil — pass to retrieval engine which checks for nil before use
```

### Graceful Degradation

`NewEmbedder` returns `(nil, nil)` when `memory.embedding_model` is unset. The retrieval engine checks for a nil embedder at construction time and disables retrieval silently. A one-time warning is emitted at session start if exactly one of `memory.retrieval_model` / `memory.embedding_model` is set, since both are required for retrieval.

### Mock for Testing

```go
// pkg/llm/mock_embedder_test.go  (or a shared testutil package)
package llm_test

import (
    "context"
    "github.com/entrhq/forge/pkg/llm"
)

// Compile-time interface check.
var _ llm.Embedder = (*MockEmbedder)(nil)

// MockEmbedder is a test double for llm.Embedder.
type MockEmbedder struct {
    EmbedFn  func(ctx context.Context, inputs []string) ([][]float32, error)
    ModelStr string
}

func (m *MockEmbedder) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
    return m.EmbedFn(ctx, inputs)
}

func (m *MockEmbedder) Model() string { return m.ModelStr }
```

---

## Package Layout

```
pkg/llm/
    embedder.go              — Embedder interface
    embedder_factory.go      — NewEmbedder() factory

pkg/llm/embedding/
    embedding.go             — Provider struct, NewProvider(), Embed(), functional options
    embedding_test.go        — unit tests (httptest.Server mock; no real API calls)
```

---

## Validation

### Success Metrics

- `NewEmbedder` returns `(nil, nil)` — not an error — when `memory.embedding_model` is empty
- `embedding.Provider.Embed()` returns one vector per input in index order
- `embedding.Provider.Embed()` propagates HTTP and JSON errors without swallowing them
- `MockEmbedder` satisfies the `Embedder` interface at compile time (`var _ llm.Embedder = (*MockEmbedder)(nil)`)
- Retrieval engine unit tests use `MockEmbedder`; no real API calls in test suite
- `WithBaseURL` is the only change needed to target Gemini, Mistral, Ollama, or any other compatible provider

### Monitoring

- One-time session-start warning if retrieval model is set but embedding model is not (or vice versa)
- `Embed` call errors logged at warn level with input count and model name
- Startup embedding rebuild duration logged at debug level

---

## Related Decisions

- [ADR-0003](0003-provider-abstraction-layer.md) — existing provider abstraction (extended, not modified)
- [ADR-0017](0017-auto-approval-and-settings-system.md) — settings system (`MemoryConfig` added here)
- [ADR-0042](0042-summarization-model-override.md) — per-capability model config pattern (followed here)
- [ADR-0044](0044-long-term-memory-storage.md) — storage layer (prerequisite)
- [ADR-0046](0046-long-term-memory-capture.md) — capture pipeline (does not use Embedder)
- [ADR-0047](0047-long-term-memory-retrieval.md) — retrieval engine (primary consumer of Embedder)

---

## References

- [Long-Term Memory PRD](../product/features/long-term-memory.md)
- [OpenAI Embeddings API](https://platform.openai.com/docs/guides/embeddings)
- [Gemini OpenAI-compatible endpoint](https://ai.google.dev/gemini-api/docs/openai)
- [LiteLLM OpenAI proxy](https://docs.litellm.ai/docs/proxy/quick_start) — example of compatible local proxy

---

## Notes

**Why not a Gemini-specific SDK implementation?**

The earlier draft included a `geminiEmbedder` backed by `google.golang.org/genai`. This was removed because:
1. Gemini exposes an OpenAI-compatible endpoint (`https://generativelanguage.googleapis.com/v1beta/openai`) that the generic embedding client handles without any Gemini-specific code
2. Adding vendor SDKs for each provider creates a maintenance burden and import graph complexity that is not warranted when a universal HTTP client works equivalently
3. Consistency with the principle already established in `pkg/llm/openai/`: one OpenAI-compatible HTTP client covers the whole market

**Why `[]string` input to `Embed()`?**

The OpenAI embeddings endpoint accepts a batch of strings in a single request. The retrieval engine always embeds the full hypothesis batch (N sentences) in one call. Batch semantics are more efficient and simpler than repeated single-input calls.

**Last Updated:** 2025-02-22
