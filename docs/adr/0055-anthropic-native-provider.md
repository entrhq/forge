# 0055. Anthropic Native Provider

**Status:** Proposed
**Date:** 2026-08-21
**Deciders:** Justin Wilkin
**Technical Story:** Add a native Anthropic Messages API provider to connect to any self-hosted Anthropic-Messages-API-compatible endpoint, not only via an OpenAI-compatible proxy.

---

## Context

### Background

F1. Forge has one concrete provider: `pkg/llm/openai`, an OpenAI Chat Completions client. All models, including `anthropic/claude-sonnet-4.5` (the current default), run through it via an OpenAI-compatible proxy (OpenRouter). Forge has never spoken Anthropic's native `/v1/messages` API: different request/response shape, `x-api-key`/`anthropic-version` auth instead of Bearer, different SSE events, top-level `system` field.

F2. [ADR-0003](0003-provider-abstraction-layer.md) claims Anthropic and Gemini support were implemented. They were not. This ADR corrects that record and is the first to add a second provider.

### Problem Statement

Forge cannot reach an Anthropic Messages API endpoint natively, blocking self-hosted deployments that implement that format.

### Goals

- `pkg/llm/anthropic` provider implementing `llm.Provider` against native `/v1/messages`, `base_url`-configurable to target Anthropic directly or any self-hosted endpoint implementing the same API.
- Existing OpenAI/OpenRouter configs unchanged.
- Provider selection explicit and config-driven.

### Non-Goals

- Native Gemini provider.
- Prompt caching. Splitting this out to its own ADR: whether "cache the system prompt and tools" is even a valid default depends on how stable those things are turn-to-turn, and Forge's system prompt, tool list, and context-compaction strategies all mutate mid-session in ways that need their own design pass, not a paragraph in the provider ADR. See [ADR-0056](0056-prompt-caching-strategy.md).
- Restructuring `types.Message` into content blocks.
- Settings-UI redesign. The `/settings` overlay (`pkg/executor/tui/overlay/settings.go`) hardcodes the LLM field list rather than rendering `Section.Data()` generically, so the new `provider` key is added to that list. No other UI work.
- The embedding provider (`pkg/llm/embedding`, [ADR-0045](0045-long-term-memory-embedding-provider.md)). Anthropic has no embeddings API; that provider surface is untouched.

---

## Decision Drivers

* No behavior change for existing OpenAI/OpenRouter configs.
* No inference-based provider routing. Forge already uses `anthropic/`-prefixed model strings as OpenRouter routing hints; inferring provider type from that string would break the current default setup.
* Match the `pkg/llm/openai` package shape (functional options, `BuildProvider` precedence, live-reload from [ADR-0034](0034-live-reloadable-llm-settings.md)).
* Wire-format correctness over hand-rolled types. Anthropic publishes an official Go SDK (`github.com/anthropics/anthropic-sdk-go`); building request/response/streaming handling on its typed params instead of hand-rolled JSON and SSE parsing removes an entire class of protocol bugs the OpenAI provider has to hand-maintain today.

---

## Considered Options

### O1: Explicit `provider` config field, dedicated `pkg/llm/anthropic` package

Add `Provider` (`"openai"` default | `"anthropic"`) to `LLMSection`. A factory dispatches to `openai.NewProvider` or `anthropic.NewProvider` on that field.

**Pros:** explicit, no ambiguity on wire protocol; zero behavior change by default; matches the existing `base_url`/`api_key` config and live-reload pattern.

**Cons:** one new config field and dispatch point; touches two call sites (`openai/factory.go`, `executor/tui/reload.go`).

### O2: Infer provider from `base_url` or `model` prefix

Auto-detect Anthropic vs OpenAI-compatible from URL shape or model prefix.

**Cons:** the current default model is `anthropic/claude-sonnet-4.5` over OpenRouter, an OpenAI-compatible wire format. Prefix inference misroutes it. A self-hosted endpoint's URL won't reliably match an Anthropic-specific pattern either. Rejected: it breaks the codebase's own default configuration.

---

## Decision

**Chosen: O1.**

O2 breaks the existing default config the moment routing is inferred from a model-string prefix, and base-URL sniffing does not generalize to an arbitrary self-hosted endpoint. O1 is additive, mirrors patterns already proven in this codebase, and gives an unambiguous, config-driven answer to "which wire protocol is this request using."

---

## Consequences

### Positive

- Forge reaches Anthropic and any self-hosted endpoint implementing its Messages API directly.
- Default `provider` value is `"openai"`: existing OpenRouter/OpenAI-compatible configs unaffected.
- Corrects the stale ADR-0003 claim.

### Negative

- A second provider implementation to maintain. Anthropic's auth, request/response shape, and streaming model share nothing with OpenAI's beyond package structure.
- `AnalyzeDocument` needs a second implementation using Anthropic's `image`/`document` blocks instead of OpenAI's `image_url` data URL.
- New dependency: `github.com/anthropics/anthropic-sdk-go`. First non-`openai-go` SDK dependency in `pkg/llm`.

### Neutral

- `LLMSection` gains one field and one branch at the two provider-construction sites. First precedent for provider-type dispatch in config; the next provider follows the same pattern.

---

## Implementation

### Provider selection

`pkg/config/llm.go`:

```go
type LLMSection struct {
    Provider             string // "openai" (default) | "anthropic"
    Model                string
    BaseURL              string
    APIKey               string
    SummarizationModel   string
    BrowserAnalysisModel string
    mu                   sync.RWMutex
}
```

`Data()`/`SetData()` gain a `provider` key. Empty resolves to `"openai"`: current behavior, unchanged.

A new factory (`pkg/llm/factory.go`) replaces both hardcoded `openai.NewProvider(...)` call sites, but the two sites differ and the ADR is explicit about that rather than implying identical behavior:

- `pkg/llm/openai/factory.go:BuildProvider` (startup path) runs the full CLI > env > config > default precedence chain today; the new dispatch adds a `provider` branch at the same point without changing that chain for `model`/`base_url`/`api_key`. The env step reads the provider-specific variable first (`OPENAI_*` or `ANTHROPIC_*`), then a provider-neutral `FORGE_API_KEY`/`FORGE_BASE_URL`, so one credential can serve an endpoint that speaks both APIs.
- `pkg/executor/tui/reload.go:reloadLLMProvider` (live `/settings` reload, per ADR-0034) has no CLI/env fallback today — it re-reads `config.GetLLM()` only. The new dispatch adds the same `provider` branch there, reading `provider` from config the same way.

### `pkg/llm/anthropic` package

Built on the official `github.com/anthropics/anthropic-sdk-go` rather than a hand-rolled HTTP/JSON client, so request/response/streaming correctness comes from the vendor SDK instead of being hand-maintained (the way `pkg/llm/openai` currently hand-builds requests and hand-parses SSE):

- `NewProvider(apiKey string, opts ...ProviderOption)`, `WithModel`, `WithBaseURL`, env fallback (`ANTHROPIC_API_KEY`, `ANTHROPIC_BASE_URL`). `WithBaseURL` maps to the SDK's `option.WithBaseURL`, targeting any self-hosted endpoint; a non-default auth header (if a target endpoint needs something other than the SDK's default `x-api-key`) goes through `option.WithHeader`.
- Requests built directly as the SDK's own `anthropic.MessageNewParams`, avoiding a hand-rolled DTO layer. Forge's `system`-role messages are extracted into the SDK's `[]anthropic.TextBlockParam` `System` field. `user` and `assistant` roles map to plain text content blocks, the same way `pkg/llm/openai` treats them. The provider never receives a `tool`-role message: Forge's tool protocol is XML text inside message content, and `normalizeRoleForLLM` (`pkg/agent/prompts/builder.go`) remaps `RoleTool` to `RoleUser` before any provider sees the payload. The SDK's native `tool_use`/`tool_result` block helpers are not used.
- Streaming via the SDK's iterator (`client.Messages.NewStreaming(ctx, params)`, `stream.Next()`/`stream.Current()`) rather than hand-rolled SSE parsing, with `anthropic.Message.Accumulate(event)` used for end-of-stream usage bookkeeping.
- `AnalyzeDocument` using the SDK's `image`/`document` block constructors (base64 source) instead of OpenAI's `image_url` data URL.
- `GetModelInfo()` sets `ModelInfo.Provider = "anthropic"`.
- `max_tokens` is required by the Messages API. Defaults are 64000 for streamed completions and 16000 for non-streamed document analysis, following Anthropic's guidance not to lowball the ceiling (current models allow 128K output; hitting the cap truncates mid-response, and non-streamed requests need headroom under SDK HTTP timeouts). A `max_tokens` config field (`LLMSection.MaxTokens`, `WithMaxTokens`) overrides both, for models with a lower cap or for cost bounds. The OpenAI provider does not send `max_tokens` and ignores the field.

### Migration Path

None. Default `provider: "openai"` is a no-op for every existing config, CLI invocation, env-var setup. Opt in with `provider: anthropic` and `base_url` pointed at the target endpoint.

### Timeline

Single PR: config field, then `pkg/llm/anthropic` (`Complete`, then `StreamCompletion`, then `AnalyzeDocument`), then factory dispatch, then docs.

---

## Validation

### Success Metrics

- Full conversation, including tool use and streaming, completes against a self-hosted Anthropic-Messages-API-compatible endpoint with `provider: anthropic` configured.
- Existing `openai` provider tests pass unmodified: zero regression for existing configs.

### Monitoring

None beyond existing logging conventions; usage/cache-token monitoring is scoped to the prompt-caching ADR.

---

## Related Decisions

- [ADR-0003](0003-provider-abstraction-layer.md): defines the `llm.Provider` interface this implementation conforms to; corrects its stale Anthropic/Gemini claim.
- [ADR-0034](0034-live-reloadable-llm-settings.md): config precedence and live-reload path the new dispatch integrates with.
- [ADR-0056](0056-prompt-caching-strategy.md): depends on this provider existing, since `cache_control` is an Anthropic Messages API feature this ADR's provider makes reachable, but the caching design itself is out of scope here.

---

## References

- Anthropic Messages API: https://docs.anthropic.com/en/api/messages
- Anthropic Go SDK: https://github.com/anthropics/anthropic-sdk-go

---

## Notes

None.

**Last Updated:** 2026-08-21
