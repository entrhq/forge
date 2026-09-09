# 0056. Prompt Caching Strategy

**Status:** Proposed
**Date:** 2026-08-21
**Deciders:** Justin Wilkin
**Technical Story:** Turn on prompt caching by default for the Anthropic provider ([ADR-0055](0055-anthropic-native-provider.md)), by redesigning prompt assembly so the content caching depends on is actually stable turn-to-turn.

---

## Context

### Background

F1. [ADR-0055](0055-anthropic-native-provider.md) split caching out of the provider ADR specifically because a naive "cache system prompt + tools + last message" heuristic does not hold in this codebase. Investigation into `pkg/agent` confirmed why:

F2. **System prompt is rebuilt every iteration, and mutated at the front.** `preparePrompt()` (`pkg/agent/iteration.go:59-115`) calls `buildSystemPrompt()` (`pkg/agent/prompts.go:10-37`) fresh on every LLM call. Long-term memory retrieval (`pkg/agent/longtermmemory/retrieval/engine.go`) is keyed on the latest user message and **prepended** ahead of the static prompt text (`iteration.go:83`), so the byte content before any cache breakpoint differs on nearly every turn.

F3. **Tool listing is non-deterministic and mutates mid-session.** `getToolsList()` (`pkg/agent/tools.go:9-25`) and `FormatToolSchemas()` (`pkg/agent/prompts/formatter.go:88-105`) iterate a Go map with randomized order, with no sort — byte offsets can shift even when the tool set hasn't changed. The set itself also changes: browser tools appear/disappear with session state (`pkg/tools/browser/registry.go:69-73`), and custom tools are rescanned from disk every turn (`pkg/agent/assistant.go:67-88`, ADR-0037), so an agent creating a tool mid-session changes the very next turn's prompt.

F4. **Context compaction rewrites the front of history, not the tail.** Three strategies (`pkg/agent/context/manager.go:123-192`) — tool-call summarization, threshold summarization, goal-batch compaction — each replace the oldest eligible messages with a `[SUMMARIZED]`/`[GOAL BATCH]` block via `conv.Clear()` + `conv.AddMultiple()`, triggered independently by token percentage, message age, or turn count. The message immediately after the system prompt is not stable; it gets silently swapped at a point the caller doesn't control.

F5. Tool definitions are inlined as XML-formatted text inside the system prompt string (`<available_tools>`), not sent through a provider's native `tools` field. There is no separate "tools" content block to give its own cache breakpoint; it is already part of the system prompt text.

### Problem Statement

Caching only pays off if the content before a breakpoint is byte-identical to the previous request's content up to that point. Today, nothing in the assembled prompt meets that bar: the most volatile content (memory retrieval) sits at the front, tool ordering is non-deterministic, and compaction rewrites history unpredictably. Turning on caching without fixing these first would mean writing to the cache on nearly every request and reading from it on almost none.

### Goals

- Redesign prompt assembly so genuinely stable content (static instructions, AGENTS.md, capability text) is separated from session-stable content (tool listing) and from per-turn content (memory retrieval, the live user turn), in that order.
- Make tool listing deterministic.
- Give `pkg/agent` a way to say *what kind of content this is* without knowing anything about Anthropic's wire format — the categorization is decided where the prompt is built, and `pkg/llm/<provider>` decides how (or whether) to act on it.
- Cache breakpoints on by default for the Anthropic provider once the above holds, using the standard 5-minute ephemeral TTL.
- Accept and document that a compaction event invalidates the cache for that turn; caching does not need to survive every possible history mutation, only pay off on the (common) turns where nothing structural changed.

### Non-Goals

- The 1-hour extended cache TTL (beta feature, 2x write cost). The default 5-minute ephemeral cache fits normal turn cadence; revisit if long idle gaps between turns turn out to be common.
- Changing `types.Message.Content` from a flat string into a content-block array. Stability is tracked as metadata on the existing `Message`, not by restructuring its content.
- Changing how compaction decides *what* to summarize or *when* to fire. This ADR only changes what happens to caching when compaction fires, not the compaction strategies themselves (ADR-0041, ADR-0043 stand).
- A generic mechanism for providers with no explicit client-side caching control (OpenAI). The stability metadata is provider-agnostic by design, but only the Anthropic provider is expected to act on it right now — see [ADR-0055](0055-anthropic-native-provider.md) on why OpenAI needs nothing here.

---

## Decision Drivers

* Caching that doesn't actually hit is worse than no caching: it pays the ~25% write premium on every request for no benefit. The redesign has to fix the structural churn, not just add breakpoints on top of it.
* The categorization of "how stable is this content" belongs where the content is assembled (`pkg/agent`), not guessed positionally by the provider. `pkg/llm/<provider>` should only decide how to translate a stability signal into its own wire format.
* No Anthropic-specific concepts leak into `pkg/agent` or `types`. `cache_control`, `ephemeral`, and breakpoint counts are `pkg/llm/anthropic` concerns.
* Minimal-diff on `types.Message`: add metadata, don't restructure content, consistent with ADR-0055's decision not to touch `Message.Content`.

---

## Considered Options

### O1: Stability metadata on `types.Message`, prompt assembly redesigned to segment by stability, provider translates

Add a `Stability` field to `types.Message` (`Volatile` default | `Session` | `Static`). `PromptBuilder` emits the system prompt as multiple ordered system-role messages instead of one string, tagged by stability class. Long-term memory retrieval moves off the system prompt entirely and is appended after history as an ephemeral user message instead of prepending to it. Tool/custom-tool/browser-guidance listings are sorted deterministically and tagged `Session`. Compaction summary blocks are tagged `Session` once written. `pkg/llm/anthropic` reads `Stability` to decide `cache_control` placement; `pkg/llm/openai` ignores the field.

**Pros:** fixes the actual root causes (F2-F4), not just adds breakpoints on top of unstable content. Categorization lives in `pkg/agent`, translation lives in the provider, matching the request. Additive to `types.Message`; no content-block restructuring.

**Cons:** touches `pkg/agent/prompts`, `pkg/agent/iteration.go`, `pkg/agent/tools.go`, and the compaction strategies' message construction, in addition to `pkg/llm/anthropic`. Larger diff than a provider-only change.

### O2: Minimal patch, no agent-side signal

Move memory injection to append instead of prepend, sort tool listing, and leave cache breakpoint placement entirely inside `pkg/llm/anthropic` guessing positions structurally (system prompt end, last message).

**Cons:** the provider has no way to know when a compaction event just rewrote history, or which parts of the system prompt are genuinely static versus session-stable versus this-turn-only. It would still cache blindly and eat unnecessary write costs on every compaction turn without a signal to size that against. Doesn't satisfy the "categorize at the source, translate at the provider" requirement.

### O3: Explicit `cache_control` markers set directly by `pkg/agent`

`pkg/agent` decides Anthropic's exact `cache_control` placement and passes it through to the provider verbatim.

**Cons:** leaks an Anthropic-specific wire concept into `pkg/agent`, which has to stay provider-agnostic since it doesn't know or care which LLM provider is configured. Breaks the moment a second caching-capable provider with different breakpoint semantics (e.g. a different max-breakpoint count) is added.

---

## Decision

**Chosen: O1.**

O2 fixes the two structural bugs but still asks the provider to guess at content it has no context for, and can't distinguish "just compacted" from "stable" without a signal — exactly the failure mode this ADR exists to avoid. O3 pushes a provider-specific concept into a layer that must stay provider-agnostic. O1 costs a larger diff than either, but it is the only option where the categorization and the translation sit in the layer that actually knows about each: `pkg/agent` knows what changed and why; `pkg/llm/anthropic` knows what Anthropic's API does with that information.

---

## Consequences

### Positive

- Prompt caching in the Anthropic provider actually hits on the common case (no compaction that turn), instead of writing on every request.
- The memory-injection-at-the-front bug is fixed as a byproduct, independent of caching: today's non-cached requests already resend a differently-ordered system prompt every turn for no functional reason.
- `Stability` is a small, reusable signal: a future caching-capable provider consumes the same metadata without `pkg/agent` changes.

### Negative

- Larger implementation surface than a provider-only change: `pkg/agent/prompts`, `pkg/agent/iteration.go`, `pkg/agent/tools.go`, and the three compaction strategies' message-construction code all need the `Stability` tag applied consistently, or the signal is unreliable.
- Cache is invalidated once on any turn where compaction fires, which is somewhat non-deterministic to the caller by design (token-percentage and age-based triggers). But the resulting `[SUMMARIZED]`/`[GOAL BATCH]` block is itself stable content the moment it's written — breakpoint 3 in the translation below anchors on it, so the compacted prefix becomes a new cache-hit target on the very next turn rather than staying cold. The cost is one write-priced miss per compaction event, not a lasting hit-rate penalty.
- `BuildMessages` changes signature; its callers in `pkg/agent` update. `PromptBuilder.Build()` keeps returning a string for the display and token-accounting callers, so those are untouched.

### Neutral

- `types.Message` gains one field (`Stability`), defaulting to `Volatile` (zero value) so every existing call site that doesn't set it keeps current behavior.

---

## Implementation

### `types.Message` addition

`pkg/types/message.go`:

```go
type Stability int

const (
    StabilityVolatile Stability = iota // default; changes every turn
    StabilitySession                    // stable for the life of a session (tool list, compacted-history blocks)
    StabilityStatic                     // stable for the life of the binary (capability text, AGENTS.md)
)

type Message struct {
    Metadata  map[string]any
    Content   string
    Timestamp time.Time
    Role      MessageRole
    Stability Stability
}
```

Zero value is `StabilityVolatile`: every existing constructor (`types.NewUserMessage`, etc.) keeps producing the same behavior with no changes required.

### Prompt assembly redesign

`pkg/agent/prompts.PromptBuilder` gains `BuildSegments() []*types.Message`, returning ordered system-role messages tagged by stability. `Build() string` stays and returns the same bytes joined, for `GetSystemPrompt()`, snapshots, and token accounting, which need one string. Prompt content and order are unchanged; the split is at the first section that can change mid-session:

1. `StabilityStatic`: custom instructions, AGENTS.md/repository context, and the base prompts up to and including `ToolCallingPrompt`.
2. `StabilitySession`: the tool listing (`<available_tools>`) and everything after it: tool-use rules, scratchpad and custom-tools guidance, the custom-tools list, browser guidance. The rules text is itself fixed, but it follows the tool listing in the prompt, so it can only be as stable as the section before it. `getToolsList()` (`pkg/agent/tools.go`) and `getCustomToolsList()` (`pkg/agent/prompts.go`) sort by name before formatting, and `FormatToolSchema`/`GenerateXMLExample` (`pkg/agent/prompts/`) iterate each schema's `properties` map in sorted key order. Both are needed: live testing with only the list sorted still showed the session block changing hash on every request at a constant byte length, which was parameter order inside each tool's schema.

Long-term memory retrieval (`pkg/agent/longtermmemory/retrieval/engine.go`, invoked from `iteration.go`) no longer prepends to the system prompt. It is appended after history as an ephemeral user message, the same way error-recovery context already is. Appending, rather than editing the current user message in place, keeps the previous turn's user message byte-identical when the next turn arrives; an in-place edit would change it once its injection was dropped and invalidate the cache from that point. This is the fix for F2.

`prompts.BuildMessages` (`pkg/agent/prompts/builder.go`) is the center of the redesign. Previously it took `systemPrompt string`, emitted exactly one `types.NewSystemMessage`, then appended history with any `RoleSystem` entries stripped. New signature:

```go
func BuildMessages(systemMessages, history []*types.Message, userMessage string, ephemeralContext ...string) []*types.Message
```

The builder's tagged segments are appended first, in order. The history loop keeps stripping `RoleSystem` entries so stale system messages in memory never leak into the payload. `ephemeralContext` entries (retrieved memories, error context) are appended after history as user messages and never stored. `normalizeRoleForLLM` is unchanged.

Invariant the redesign must preserve: **system messages are first and contiguous in the slice, followed only by conversation messages.** `Provider.StreamCompletion` receives one flat `[]*types.Message` (`iteration.go:127`); the Anthropic translation below partitions system from conversation by finding the end of the leading `RoleSystem` run. That partition is only well-defined if nothing inserts a system message mid-conversation.

`preparePrompt()` (`iteration.go:59-115`) updates to pass the builder's `[]*types.Message` through instead of a string.

### Compaction output tagging

`pkg/agent/context`'s three strategies (`tool_call_strategy.go`, `threshold_strategy.go`, `goal_batch_strategy.go`) tag the `[SUMMARIZED]`/`[GOAL BATCH]` replacement messages they construct as `StabilitySession`: once written, a summary block doesn't change again until superseded by a later compaction pass. Each builds its summary via `types.NewAssistantMessage(...).WithMetadata(...)`; a matching `WithStability(...)` chained setter is added to `types.Message` so the tag sits in the same construction chain.

### `pkg/llm/anthropic` translation

Given the ordered message list, the provider places `cache_control: {"type": "ephemeral"}` (default 5-minute TTL, no beta header) at:

1. The last `StabilityStatic` message in the system segment, if the run transitions to `StabilitySession` or later.
2. The last system block, whatever its stability, so the tool listing caches separately from history.
3. The last `StabilitySession` message in the conversation array — the most recent compaction boundary (`[SUMMARIZED]`/`[GOAL BATCH]` block), if one exists. This caches the entire compacted prefix of history as one unit, separately from the still-changing turns after it.
4. The last message in the request overall.

Up to all 4 of Anthropic's available breakpoints; unused when a category is absent (e.g. no tools configured skips breakpoint 2, no compaction yet skips breakpoint 3). `pkg/llm/openai` ignores `Stability` entirely — no behavior change there, consistent with [ADR-0055](0055-anthropic-native-provider.md)'s scope.

### Migration Path

The `BuildMessages` signature change is internal to `pkg/agent`; no config or external-facing change. `Stability` defaults to `Volatile`, so messages constructed anywhere else behave as before. The only observable change for non-Anthropic providers is the position of retrieved memories in the request: after history as a user message rather than at the front of the system prompt.

### Timeline

Sequenced after [ADR-0055](0055-anthropic-native-provider.md) lands (the Anthropic provider needs to exist before its caching translation does): `types.Message.Stability` field → tool-listing determinism → memory-injection relocation → `PromptBuilder.BuildSegments` and `BuildMessages` signature change → compaction tagging → `pkg/llm/anthropic` breakpoint translation.

---

## Validation

### Success Metrics

- On a session with no compaction event between two consecutive turns, the second turn's Anthropic response reports non-zero `cache_read_input_tokens`.
- On the turn immediately after a compaction event, the following turn's response reports `cache_read_input_tokens` covering the compacted prefix — confirming the compaction boundary re-establishes as a cache hit within one turn, not left cold for the rest of the session.
- Tool-listing byte content is identical across two consecutive requests with an unchanged tool set (regression test on `FormatToolSchemas` output ordering).
- Existing `pkg/agent` tests pass with `PromptBuilder`'s new return shape.

### Monitoring

Debug-level log per Anthropic request with cache read/write token counts from the response `usage` block (`cache_read_input_tokens`, `cache_creation_input_tokens`), so cache hit rate is visible without new instrumentation infrastructure.

---

## Related Decisions

- [ADR-0055](0055-anthropic-native-provider.md): the provider this ADR's translation layer lives in; depends on it existing first.
- [ADR-0041](0041-goal-batch-compaction-strategy.md), [ADR-0043](0043-context-snapshot-export.md): compaction behavior this ADR tags but does not change.
- [ADR-0037](0037-custom-tools-system.md): source of the mid-session tool-list churn this ADR's determinism fix addresses.

---

## References

- Anthropic prompt caching: https://docs.anthropic.com/en/docs/build-with-claude/prompt-caching

---

## Notes

If a future compaction strategy could preserve a stable prefix instead of rewriting from the front (e.g. a rolling window that only ever truncates the tail), cache hit rate would stop being bounded by compaction frequency. Not proposed here since it changes compaction's own design, out of scope per Non-Goals.

**Last Updated:** 2026-08-21
