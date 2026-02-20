# 0043. Context Snapshot Export (`/exportcontext`)

**Status:** Accepted
**Date:** 2025-07-15
**Deciders:** Forge team
**Technical Story:** Engineers need to inspect the exact conversation payload that the agent sends to the LLM — including summarization blocks, goal-batch compactions, and raw messages — to debug context management issues without adding external instrumentation.

---

## Context

### Background

Forge uses a three-strategy context management pipeline (ADR-0014, ADR-0015, ADR-0018, ADR-0041) that continuously transforms the conversation:

- `ToolCallSummarizationStrategy` compresses old tool call / tool result pairs into `[SUMMARIZED]` assistant messages.
- `ThresholdSummarizationStrategy` half-and-half collapses the older portion of the conversation.
- `GoalBatchCompactionStrategy` merges completed user turns into `[GOAL BATCH]` blocks.

When something goes wrong (the agent loses context, repeats itself, ignores earlier information) engineers need to see exactly what messages the LLM is receiving — the transformed payload, not the original history.

### Problem Statement

There is currently no way to dump the live conversation payload to disk for offline inspection. The `/context` command shows aggregate statistics (token counts, block counts) but does not expose message content. Attaching a debugger or log scraper is cumbersome and disrupts the normal workflow.

### Goals

- Provide a single TUI slash command (`/exportcontext`) that writes the full conversation payload to a JSON file in the workspace.
- The exported file must be self-contained and human-readable enough for engineers to `jq`-filter, diff, or load into tooling.
- The command must be non-blocking and synchronous — no round-trip to the agent is required.
- Saved files must be timestamped and accumulate (never overwrite) so engineers can compare snapshots before and after summarization events.

### Non-Goals

- Real-time streaming of context to disk (continuous write, not on-demand snapshot).
- A dedicated viewer overlay inside the TUI (the JSON file is the artifact; engineers use external tools).
- Integration with the headless executor (context export is a debugging aid aimed at interactive sessions).
- Compressing or encrypting the snapshot (it lives inside the workspace, which is already the user's project directory).

---

## Decision Drivers

* **Zero agent-layer coupling** — the TUI already calls `GetContextInfo()` synchronously; adding `GetMessages()` to the same interface keeps the pattern consistent and avoids a new async event round-trip.
* **Engineer-friendly output** — JSON with per-message token counts is trivially processable with `jq` or Python; any other format (proto, msgpack) would require additional tooling.
* **Non-intrusive** — files land in `<workspace>/.forge/context/` (already gitignored by Forge convention); they never appear in `git status` and never affect the running session.
* **Simplicity** — the handler is ~50 lines, no new packages, no new types, no new channels.

---

## Considered Options

### Option 1: `GetMessages()` method on `Agent` interface (chosen)

**Description:** Add `GetMessages() []*types.Message` to the `Agent` interface. `DefaultAgent` implements it by returning `a.memory.GetAll()` (already a copy). The TUI handler calls this synchronously, builds the JSON in the handler goroutine, and writes the file.

**Pros:**
- Synchronous — no async coordination, no new channels, no risk of races.
- Consistent with the existing `GetContextInfo()` pattern.
- Single LLM call avoided entirely (no summarization, just a read).
- `memory.GetAll()` is already mutex-protected and returns a copy.

**Cons:**
- Exposes `[]*types.Message` on the public `Agent` interface — callers could inspect message internals.
- Any future `Agent` implementor must add `GetMessages()`.

### Option 2: Event-based round-trip (`InputTypeContextExportRequest` / `EventTypeContextExportReady`)

**Description:** Mirror the `/notes` pattern: TUI emits a request event, agent handles it in the event loop, emits a response event with the messages, TUI receives the response and writes the file.

**Pros:**
- Fully decoupled — TUI and agent communicate only through typed events.

**Cons:**
- Requires two new event types for what is a read-only, zero-side-effect operation.
- The async round-trip adds latency and coordination complexity with no benefit — `memory.GetAll()` is already thread-safe and copyable without the agent event loop's involvement.
- The pattern was justified for `/notes` because notes might change during the round-trip; the conversation memory changes only between agent turns, and context export is invoked by the user between turns.

### Option 3: Exporter service / middleware

**Description:** A dedicated `ContextExporter` struct registered as a middleware that intercepts every turn and can snapshot on demand.

**Pros:**
- Could support continuous export or other future modes.

**Cons:**
- Significant over-engineering for what is purely an on-demand debug snapshot.
- No identified use case for continuous export.

---

## Decision

**Chosen Option:** Option 1 — `GetMessages()` on the `Agent` interface, synchronous TUI handler.

### Rationale

The core operation (read a copy of in-memory messages) is already thread-safe and trivially expressible as a synchronous method call. An async round-trip would add complexity with no correctness benefit. Keeping the handler entirely in the TUI layer avoids polluting the agent event loop with a debug-only operation.

---

## Consequences

### Positive

- Engineers can snapshot context at any point in a session with `/exportcontext`.
- Snapshots accumulate in `.forge/context/` and can be compared across time.
- No new channels, events, or goroutines required.
- The `GetMessages()` method is generally useful (tests, introspection, future features).

### Negative

- `GetMessages()` is now part of the public `Agent` interface — third-party implementors must add the method.
- The exported JSON may be large for long sessions (expected — this is a debugging tool).

### Neutral

- `.forge/context/` directory is created on first export; subsequent exports accumulate files.
- Filename format: `<YYYYMMDD-HHMMSS>-context.json` (local time of export).

---

## Implementation

### Files changed

| File | Change |
|------|--------|
| `pkg/agent/agent.go` | Add `GetMessages() []*types.Message` and `GetSystemPrompt() string` to `Agent` interface |
| `pkg/agent/default.go` | Implement `GetMessages()` returning `a.memory.GetAll()`; implement `GetSystemPrompt()` delegating to `buildSystemPrompt()` |
| `pkg/executor/tui/slash_commands.go` | Register `/exportcontext` command; implement `handleExportContextCommand` and `buildContextSnapshot` helpers |

### Exported JSON schema

```json
{
  "exported_at": "2025-07-15T14:32:01+01:00",
  "workspace":   "/home/user/myproject",
  "token_summary": {
    "current_context": 45210,
    "max_context":    100000,
    "usage_percent":  45.2,
    "system_prompt":  12000,
    "conversation":   33210,
    "raw_messages":   20000,
    "summary_blocks": 8000,
    "goal_batch_blocks": 5210
  },
  "messages": [
    {
      "index":          0,
      "role":           "system",
      "content":        "You are Forge...",
      "tokens":         12000,
      "is_summarized":  false,
      "summary_type":   "",
      "summary_count":  0,
      "summary_method": ""
    },
    {
      "index":          1,
      "role":           "assistant",
      "content":        "[SUMMARIZED] ## Milestones\n...",
      "tokens":         3200,
      "is_summarized":  true,
      "summary_type":   "",
      "summary_count":  8,
      "summary_method": "ToolCallSummarization"
    }
  ]
}
```

### Handler pseudocode

```go
func handleExportContextCommand(m *model, args []string) interface{} {
    if m.agent == nil {
        m.showToast("Error", "Agent not available", "❌", true)
        return nil
    }

    snapshot := buildContextSnapshot(m)

    dir := filepath.Join(m.workspaceDir, ".forge", "context")
    os.MkdirAll(dir, 0o755)

    filename := time.Now().Format("20060102-150405") + "-context.json"
    path := filepath.Join(dir, filename)

    data, _ := json.MarshalIndent(snapshot, "", "  ")
    os.WriteFile(path, data, 0o644)

    m.showToast("Context exported", path, "📄", false)
    return nil
}
```

### Data flow

1. User types `/exportcontext` in TUI input.
2. TUI parses the slash command and calls `handleExportContextCommand(m, args)`.
3. Handler calls `m.agent.GetSystemPrompt()` → synthesises the full system prompt (tools, custom instructions, repository context) — this is **never** stored in conversation memory.
4. Handler calls `m.agent.GetMessages()` → `a.memory.GetAll()` (mutex-protected copy of conversation history).
5. Handler calls `m.agent.GetContextInfo()` for aggregate token statistics.
6. Handler builds `contextSnapshot` with messages[0] = system prompt, messages[1..N] = conversation history — matching exactly what the agent passes to the LLM on each turn.
7. Handler marshals to JSON and creates `<workspace>/.forge/context/` if absent.
8. Handler writes `<YYYYMMDD-HHMMSS>-context.json`.
9. Handler calls `m.showToast` with the absolute path for feedback.
10. Handler returns `nil` (no further TUI action needed).

---

## Validation

### Success Metrics

- `/exportcontext` produces a valid JSON file in `.forge/context/` within 100 ms on sessions up to 200 messages.
- `messages[0]` in the exported JSON is always the system message (role `"system"`) containing the full system prompt.
- `messages[1..N]` match exactly what `m.agent.GetMessages()` returns, in order.
- Token counts in `token_summary` match those shown by `/context`.
- Multiple invocations accumulate files (no overwrite).

---

## Related Decisions

- [ADR-0014](0014-composable-context-management.md) — Context management pipeline
- [ADR-0033](0033-notes-viewer-tui-command.md) — Notes viewer TUI command (pattern reference)
- [ADR-0020](0020-context-information-overlay.md) — `/context` overlay (token statistics source)
- [ADR-0042](0042-summarization-model-override.md) — Summarization model override

---

## Notes

**Last Updated:** 2025-07-15
