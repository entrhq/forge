# 42. Summarization Model Override

**Status:** Proposed
**Date:** 2025-07-14
**Deciders:** Forge team
**Technical Story:** Allow users to specify a cheaper/faster model for context summarization independently of the main agent model, configurable through the TUI `/settings` overlay.

---

## Context

### Background

Forge uses a multi-strategy context management system (ADR-0014, ADR-0015, ADR-0018, ADR-0041) that automatically summarizes conversation history. The three strategies — `ToolCallSummarizationStrategy`, `ThresholdSummarizationStrategy`, and `GoalBatchCompactionStrategy` — all call an LLM provider to generate summaries. Currently, `context.Manager` holds a single `llm.Provider` and passes it through to every `strategy.Summarize()` call, meaning summarization always uses the same model as the main agent loop.

The `/settings` overlay (ADR-0034) lets users configure a single LLM connection: model, base URL, and API key. There is no mechanism to use a different model for summarization tasks.

### Problem Statement

Summarization tasks are semantically simpler than primary agent reasoning — compressing old tool-call pairs, collapsing assistant blocks, and compacting goal arcs. Running them on a frontier model wastes tokens on work that a smaller, cheaper model handles equally well. On long coding sessions with many summarization events, the cost impact is significant.

The naive fix — maintaining a second `llm.Provider` instance — is expensive: it would require two separate configurations (model, base URL, API key) in the settings UI, two hot-reload paths, two provider objects in memory sharing only by coincidence. Since summarization always uses the same endpoint and credentials as the main provider, this overhead is unjustified.

### Goals

- Allow a separate model string for summarization calls (e.g., `claude-haiku-3-5`) while using the same API key and base URL as the main provider.
- Add exactly one new text field ("Summarization Model") to the existing LLM section in `/settings` — nothing else changes.
- Support hot-reload: changing the summarization model in settings takes effect immediately.
- Fall back to the main provider's model when no summarization model is configured (zero required migration).
- Persist as a single string in `~/.config/forge/config.yaml` under the existing `llm` section.

### Non-Goals

- Supporting a different API endpoint or credentials for summarization — summarization always shares the main provider's connection.
- Per-strategy model configuration.
- Exposing this as a CLI flag.

---

## Decision Drivers

* **Minimal footprint** — The summarization override is a model-name string, not a second provider object. No duplication of credentials or connection management.
* **Single user-facing config surface** — Users manage one LLM connection. Adding a second independently configurable provider would be confusing and inconsistent with the existing settings design.
* **Zero regression** — Omitting the setting must produce identical behavior to today.
* **Hot-reload compatibility** — Must integrate cleanly with the live-reload path established in ADR-0034.
* **Provider reuse** — The HTTP client, connection pool, API key, and base URL are shared; only the model field differs.

---

## Considered Options

### Option 1: Second full `llm.Provider` on `context.Manager`

**Description:** Add a `summarizationLLM llm.Provider` field to `context.Manager`. Construct a second `openai.Provider` at startup and on hot-reload using the same API key and base URL but the summarization model string. Wire `SetSummarizationProvider()` through `DefaultAgent`.

**Pros:**
- Clean separation — each provider fully self-contained.

**Cons:**
- Two provider objects in code for what is conceptually one connection with a model swap.
- Settings UI must somehow expose only the model string while silently inheriting base URL and API key — confusing and fragile if the user changes base URL after the fact.
- Hot-reload must rebuild the summarization provider any time *either* the main settings *or* the summarization model changes.
- `context.Manager`, `DefaultAgent`, `reloadLLMProvider()`, and all three entry points all grow code to manage the second provider.

### Option 2: Model-override option on `Complete()` / `StreamCompletion()` calls

**Description:** Change the `llm.Provider` interface to accept per-call options (e.g., variadic `CallOption`), allowing callers to inject a model override at call time.

**Pros:**
- No extra provider objects.

**Cons:**
- Breaking change to the `llm.Provider` interface — all implementations and all call sites must be updated.
- Adds call-site complexity for an option that is only needed in one place (the context manager).
- Over-engineers the general provider interface to solve a narrow summarization use case.

### Option 3: `summarizationModel string` on `context.Manager` + `ModelCloner` optional interface

**Description:** `context.Manager` stores only a `summarizationModel string` (not a provider). A new optional `ModelCloner` interface allows a provider to produce a lightweight clone of itself with a different model. `openai.Provider` implements `CloneWithModel(model string) llm.Provider` as a shallow struct copy sharing the HTTP client and credentials. At summarization call time, the manager calls `CloneWithModel` if the interface is available and a model is configured, otherwise falls back to `m.llm` unchanged.

**Pros:**
- One setting, one string — no second provider object ever exists as long-lived state.
- Settings UI adds exactly one row (the model string); no new base URL or API key fields.
- Hot-reload only needs to call `manager.SetSummarizationModel(newModel)` — one line.
- `ModelCloner` is opt-in: providers that don't implement it degrade gracefully to the main model.
- The shallow clone is trivially cheap (struct copy + one string allocation) and shares the HTTP connection pool.
- Strategy interface, agent interface, and entry points require minimal changes.

**Cons:**
- `ModelCloner` is an optional interface resolved via type assertion — slightly non-obvious pattern, though it follows Go idioms well (e.g., `io.WriterTo`, `http.Flusher`).
- The cloned provider is ephemeral (created per-summarization-call) — negligible cost but worth noting.

---

## Decision

**Chosen Option:** Option 3 — `summarizationModel string` on `context.Manager` + `ModelCloner` optional interface

### Rationale

This is the minimum representation of the feature: one string, one optional interface, one new settings field. Option 1 creates unnecessary structural complexity for what is semantically just a model name swap. Option 2 breaks the provider interface contract for a narrow use case. Option 3 keeps the existing provider, settings, and agent structures intact while adding exactly the expressiveness needed — the ability to call the same endpoint with a different model name.

---

## Consequences

### Positive

- Users configure one LLM connection (as before); the summarization model is a lightweight override within that connection.
- The `/settings` LLM section gains exactly one new editable row.
- Hot-reload path is a single `SetSummarizationModel()` call — no provider rebuild required.
- Existing behavior is fully preserved when the field is absent.

### Negative

- `openai.Provider` gains a `CloneWithModel()` method. If other provider implementations are added in the future they should also implement `ModelCloner` to support this feature.
- The per-call clone is created on every summarization LLM call (not once at startup). This is negligible (struct copy) but differs from the normal "create once, reuse" provider pattern.

### Neutral

- `config.yaml` gains a new optional `summarization_model` string under the existing `llm` key.
- A new optional `ModelCloner` interface lives in `pkg/llm`.

---

## Implementation

### 1. `pkg/llm/provider.go` — Add `ModelCloner` optional interface

```go
// ModelCloner is an optional interface that LLM providers can implement to
// support lightweight per-call model overrides without constructing a full
// second provider. The returned provider shares credentials and transport with
// the original but directs calls to the given model.
type ModelCloner interface {
    CloneWithModel(model string) Provider
}
```

### 2. `pkg/llm/openai/openai.go` — Implement `ModelCloner`

```go
// CloneWithModel returns a shallow copy of p with the model field replaced.
// The clone shares the same HTTP client, API key, and base URL as p, making
// it very cheap to create. Implements llm.ModelCloner.
func (p *Provider) CloneWithModel(model string) llm.Provider {
    clone := *p // shallow copy — shares httpClient (connection pool)
    clone.model = model
    if p.modelInfo != nil {
        mi := *p.modelInfo // copy modelInfo so mutation doesn't affect original
        mi.Name = model
        clone.modelInfo = &mi
    }
    return &clone
}
```

### 3. `pkg/config/llm.go` — Add `SummarizationModel` field

```go
type LLMSection struct {
    Model              string
    BaseURL            string
    APIKey             string
    SummarizationModel string  // optional; empty means use main model
    mu sync.RWMutex
}
```

Add `summarization_model` to `Data()`, `SetData()`, and `Reset()`. Add `GetSummarizationModel()` / `SetSummarizationModel()` accessors following the existing pattern.

Update the `Description()`:
> "Configure LLM provider settings. `summarization_model` is optional — if set, context summarization uses this model instead of `model`."

### 4. `pkg/agent/context/manager.go` — Add `summarizationModel` field

```go
type Manager struct {
    strategies         []Strategy
    llm                llm.Provider
    summarizationModel string          // NEW: optional model override for summarization calls
    tokenizer          *tokenizer.Tokenizer
    maxTokens          int
    eventChannel       chan<- *types.AgentEvent
}
```

Add `SetSummarizationModel()`:

```go
// SetSummarizationModel sets the model name to use for summarization LLM calls.
// If empty, summarization uses the same model as the main provider (m.llm).
// The provider must implement llm.ModelCloner for this to take effect.
func (m *Manager) SetSummarizationModel(model string) {
    m.summarizationModel = model
}
```

Add a helper:

```go
// providerForSummarization returns the provider to use for summarization calls.
// If a summarization model override is configured and the provider implements
// llm.ModelCloner, returns a lightweight clone with the override model.
// Otherwise returns m.llm unchanged.
func (m *Manager) providerForSummarization() llm.Provider {
    if m.summarizationModel == "" {
        return m.llm
    }
    if cloner, ok := m.llm.(llm.ModelCloner); ok {
        return cloner.CloneWithModel(m.summarizationModel)
    }
    return m.llm
}
```

In `EvaluateAndSummarize()`, replace:
```go
summarizedCount, err := strategy.Summarize(ctx, conv, m.llm)
```
with:
```go
summarizedCount, err := strategy.Summarize(ctx, conv, m.providerForSummarization())
```

### 5. `pkg/agent/default.go` — Expose `SetSummarizationModel` on `DefaultAgent`

```go
// SetSummarizationModel updates the model used for context summarization calls.
// Pass an empty string to revert to the main provider model.
func (a *DefaultAgent) SetSummarizationModel(model string) {
    if a.contextManager != nil {
        a.contextManager.SetSummarizationModel(model)
    }
}
```

### 6. `pkg/executor/tui/overlay/settings.go` — Add field to LLM section

In the `llmFields` slice inside `loadSettings()`:

```go
const summarizationModelField = "summarization_model"

llmFields := []struct {
    key         string
    displayName string
}{
    {modelField,              "Model"},
    {summarizationModelField, "Summarization Model"},  // NEW
    {baseURLField,            "Base URL"},
    {apiKeyField,             "API Key"},
}
```

No other changes to the settings overlay are needed. The new field flows through the existing text-item rendering, editing, and saving logic unchanged.

### 7. `pkg/executor/tui/reload.go` — Apply summarization model on hot-reload

```go
func (m *model) reloadLLMProvider() error {
    // ... existing provider reload logic ...

    // Apply summarization model override (empty string is valid — reverts to main model)
    summarizationModel := llmConfig.GetSummarizationModel()
    if defaultAgent, ok := m.agent.(*agent.DefaultAgent); ok {
        defaultAgent.SetSummarizationModel(summarizationModel)
    }

    return nil
}
```

### Config file format

```yaml
llm:
  model: "anthropic/claude-sonnet-4.5"
  summarization_model: "anthropic/claude-haiku-3-5"  # optional
  base_url: "https://openrouter.ai/api/v1"
  api_key: "sk-..."
```

### Entry points — Read and apply at startup

In each of `cmd/forge/main.go`, `cmd/forge/headless.go`, `cmd/forge-headless/main.go`, after creating the `contextManager`:

```go
// Apply summarization model override from config (no-op if not configured)
if llmConfig := appconfig.GetLLM(); llmConfig != nil {
    if summarizationModel := llmConfig.GetSummarizationModel(); summarizationModel != "" {
        contextManager.SetSummarizationModel(summarizationModel)
    }
}
```

### Migration Path

No migration required. Existing config files without `summarization_model` produce zero behavior change: `GetSummarizationModel()` returns `""`, `providerForSummarization()` returns `m.llm` unmodified.

---

## Validation

### Success Metrics

- Setting `summarization_model` in `/settings` persists to `config.yaml` and is applied on the next summarization event.
- Hot-reload (Ctrl+S in settings) applies the new model without restarting.
- Omitting the field produces no behavior change; all existing tests pass.
- Debug logs in `context.Manager` show the correct model name for each summarization call (`m.providerForSummarization().GetModel()`).

### Monitoring

- Add a log line in `EvaluateAndSummarize()` that records which model is used per strategy run.
- Existing `pkg/agent/context` tests require no changes (strategy interface is unmodified).

---

## Related Decisions

- [ADR-0003](0003-provider-abstraction-layer.md) — Provider abstraction (`llm.Provider` interface)
- [ADR-0014](0014-composable-context-management.md) — Composable context management
- [ADR-0015](0015-buffered-tool-call-summarization.md) — Buffered tool call summarization
- [ADR-0018](0018-selective-tool-call-summarization.md) — Selective tool call summarization
- [ADR-0034](0034-live-reloadable-llm-settings.md) — Live-reloadable LLM settings
- [ADR-0041](0041-goal-batch-compaction-strategy.md) — Goal batch compaction strategy

---

## Notes

The `ModelCloner` interface is deliberately optional rather than part of `llm.Provider`. Not every provider context requires this capability, and making it optional means non-implementing providers degrade gracefully rather than failing to compile. The pattern mirrors standard Go interfaces like `io.WriterTo` and `http.Flusher`.

If a future need arises to use a completely different provider endpoint for summarization (different base URL or API key), a second `llm.Provider` field on `context.Manager` could be added later without invalidating this design — `ModelCloner` and the second-provider approach are complementary, not mutually exclusive.

**Last Updated:** 2025-07-14
