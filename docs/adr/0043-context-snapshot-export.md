# 43. Context Snapshot Export (`/dumpcontext`)

**Status:** Proposed
**Date:** 2025-07-15
**Deciders:** Forge team
**Technical Story:** Allow engineers to export the complete, wire-format conversation context to disk for offline debugging of context management, summarization quality, and prompt engineering.

---

## Context

### Background

Forge's context management pipeline (ADR-0014, ADR-0015, ADR-0018, ADR-0041, ADR-0042) applies up to three compaction strategies before every LLM call: tool-call summarization, threshold half-compaction, and goal-batch compaction. Each strategy rewrites messages in place. The existing `/context` command (ADR-0020) shows a token-usage summary overlay in the TUI, but it reveals nothing about the actual message content that flows to the model.

When context problems occur — the agent appears to have "forgotten" something, a summary block is low-quality, the token accounting seems wrong, or a strategy fires too aggressively — there is no way to inspect what the context actually looked like at a given moment. Engineers must reason backwards from observed agent behaviour, which is slow and unreliable.

### Problem Statement

Engineers have no way to capture and inspect the exact conversation context (messages, roles, metadata, summarization flags, token counts) that Forge sent or was about to send to the LLM. Debugging context management requires either adding temporary log statements and rebuilding or attaching a debugger, neither of which is practical in production sessions.

### Goals

- Provide a `/dumpcontext` slash command available in the TUI (and as an explicit action in headless mode) that writes the current conversation context to disk.
- Export the full message list exactly as it would be sent to the LLM: all roles, all content (including `[SUMMARIZED]` and `[GOAL BATCH]` blocks), and all message metadata (`summarized`, `summary_type`, `summary_count`, `summary_method`).
- Include contextual metadata in the export file: timestamp, workspace path, token stats, active strategy names, model name.
- Save to `.forge/context/<timestamp>-context.json` inside the current workspace so the file is co-located with the project being debugged and ignored by default (`.forge/` is gitignored).
- Report the output path to the user as a toast notification so they know where to find it.
- Operate entirely within the TUI layer — no new agent interface methods, no new event types, no new channels required.

### Non-Goals

- Live context streaming or real-time updates to the dump file.
- Automatic periodic snapshotting (on every turn, or on every summarization event).
- Exporting context in formats other than JSON (e.g. YAML, plain text, Anthropic-wire-format).
- Redacting or masking conversation content before export.
- Opening or viewing the exported file from inside the TUI.
- Headless-mode `/dumpcontext` support in the first iteration (can be wired later with the same `ExportContext()` function).

---

## Decision Drivers

* **Zero new abstractions** — `GetContextInfo()` already exists on `Agent` and returns the token breakdown. Adding a `GetMessages()` method alongside it keeps the export entirely TUI-side. No new event types, input types, or agent loop changes are needed.
* **Engineer-first UX** — The file is JSON, machine-readable, diff-able, and pasteable into bug reports. The `.forge/context/` directory is a natural home for debug artefacts alongside other forge-managed state.
* **Co-location with the work** — Saving into the workspace root (not `~/.forge/`) means the snapshot is tied to the repository being debugged and can be committed, shared, or inspected with the rest of the project.
* **Consistency** — Follows the `/context` display-overlay pattern for the command trigger. Name chosen as `/dumpcontext` to mirror the "core dump" mental model and to avoid collision with the existing `/context` display command.
* **No export-path ambiguity** — A fixed directory (`<workspace>/.forge/context/`) with a timestamped filename means multiple snapshots can coexist without overwriting.

---

## Considered Options

### Option 1: TUI-only export using `GetMessages()` on the `Agent` interface

**Description:** Add a `GetMessages() []*types.Message` method to the `Agent` interface (alongside the existing `GetContextInfo()`). The `/dumpcontext` TUI handler calls both, assembles the JSON payload, and writes it to disk directly from the slash-command handler — no new events, no new input types.

**Pros:**
- Self-contained in the TUI layer; no agent-loop impact.
- Messages are already owned by `DefaultAgent.memory`; surfacing them is a trivial one-liner.
- The export handler can be written, tested, and iterated without touching the agent event loop.
- `Agent` interface already has a `GetContextInfo()` precedent — adding `GetMessages()` is symmetrical.

**Cons:**
- Adds one method to the `Agent` interface, which all implementations must satisfy.
- TUI now has direct read access to the full message content (minor coupling increase).

### Option 2: Event-based export (request/response via existing channels)

**Description:** Add an `InputTypeContextDumpRequest` input type and an `EventTypeContextDump` event type. The TUI emits the request; the agent handles it, serialises the context, and emits the event containing either the file path or the JSON string.

**Pros:**
- Consistent with the `notes_request` / `notes_data` round-trip established for `/notes` (ADR-0033).
- Agent controls the serialisation, so the dump format is a stable API boundary.

**Cons:**
- Adds two new type constants (`InputTypeContextDumpRequest`, `EventTypeContextDump`) and a handler in the agent event loop for what is a read-only operation — significant overhead for a debug feature.
- The TUI must hold state between the request and the response event, adding complexity for a one-shot command.
- Agent serialising its own messages is philosophically odd — the dump is a debug artefact, not a runtime event.
- The round-trip delay is unnecessary: the TUI can read the same data synchronously.

### Option 3: Direct file write from TUI using a new `Exporter` service

**Description:** Create a standalone `pkg/agent/export/` package with an `Exporter` struct that accepts `[]*types.Message` and a metadata struct, serialises them to the canonical JSON format, and writes the file. Both the TUI and future headless integrations call `Exporter.Write(path, messages, meta)`.

**Pros:**
- Export logic is reusable from headless mode without reimplementing serialisation.
- Clear ownership — export is a separate concern from agent and TUI.

**Cons:**
- Adds a new package for a function that is realistically ~40 lines.
- Headless export is a non-goal for this iteration, so the reuse argument is premature.
- Over-engineers a debug utility.

---

## Decision

**Chosen Option:** Option 1 — TUI-only export using `GetMessages()` on the `Agent` interface.

### Rationale

The export is inherently a debug/introspection operation performed by an engineer sitting at the TUI. It requires no asynchronous coordination, no agent-loop participation, and no new runtime state. `GetContextInfo()` already establishes the pattern of surfacing read-only agent data through the interface; `GetMessages()` is the natural complement. The entire implementation is contained in three small changes: one interface method, one `DefaultAgent` implementation, and one slash-command handler.

Option 2 (event-based) would be appropriate if the export needed to trigger side-effects in the agent loop (e.g., flushing a buffer before export) or if the response needed to be asynchronous. Neither is true here. Option 3 defers to a future need (headless export) that is explicitly out of scope.

---

## Consequences

### Positive

- Engineers can capture an exact context snapshot in under a second, mid-session, without restarting or modifying code.
- The JSON file is diff-able: before/after snapshots cleanly show what a summarization strategy changed.
- Token stats and message metadata (`summarized`, `summary_type`) are included, making it trivial to verify that compaction is working correctly.
- Multiple snapshots can coexist with distinct timestamps — useful for comparing context evolution across turns.
- `.forge/context/` artefacts can be attached to bug reports or committed alongside reproduction cases.

### Negative

- `Agent` interface gains one new method (`GetMessages()`). Any external mock or test double that implements `Agent` must add this method.
- TUI has direct read access to raw message content, including any sensitive information the user may have typed.
- Dump files are written in plaintext JSON and may contain sensitive conversation data if left in the repo.

### Neutral

- `.forge/context/` directory is created on first use (created with `os.MkdirAll`).
- Timestamp format is RFC3339 with colons replaced by hyphens for filesystem compatibility (`2025-07-15T14-30-00Z`).
- The feature is TUI-only in this iteration; headless support can be added later by calling the same `GetMessages()` path.

---

## Implementation

### File Structure

```
pkg/agent/agent.go              — add GetMessages() to Agent interface
pkg/agent/default.go            — implement GetMessages() on DefaultAgent
pkg/executor/tui/slash_commands.go  — register /dumpcontext, implement handleDumpContextCommand
```

No new packages required.

### 1. `Agent` interface (`pkg/agent/agent.go`)

```go
// GetMessages returns a snapshot of the current conversation history.
// Messages are returned in conversation order. The returned slice is a
// copy — callers may inspect but not modify the agent's message state.
GetMessages() []*types.Message
```

### 2. `DefaultAgent` implementation (`pkg/agent/default.go`)

```go
func (a *DefaultAgent) GetMessages() []*types.Message {
    return a.memory.GetAll() // already returns a copy
}
```

### 3. Export payload (`pkg/executor/tui/slash_commands.go`)

The JSON export structure captures everything an engineer needs to reconstruct or replay the context:

```go
type contextDump struct {
    ExportedAt    string              `json:"exported_at"`     // RFC3339
    WorkspaceDir  string              `json:"workspace_dir"`
    Model         string              `json:"model"`           // from ContextInfo (future)
    TokenStats    contextDumpTokens   `json:"token_stats"`
    Strategies    []string            `json:"strategies"`      // future: from Manager
    MessageCount  int                 `json:"message_count"`
    Messages      []contextDumpMsg    `json:"messages"`
}

type contextDumpTokens struct {
    Current    int     `json:"current"`
    Max        int     `json:"max"`
    UsagePct   float64 `json:"usage_percent"`
    Free       int     `json:"free"`
}

type contextDumpMsg struct {
    Index     int                    `json:"index"`
    Role      string                 `json:"role"`
    Content   string                 `json:"content"`
    Tokens    int                    `json:"tokens"`          // per-message token count
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
```

### 4. Command registration (`pkg/executor/tui/slash_commands.go`)

```go
registerCommand(&SlashCommand{
    Name:        "dumpcontext",
    Description: "Export current context snapshot to .forge/context/",
    Type:        CommandTypeTUI,
    Handler:     handleDumpContextCommand,
    MinArgs:     0,
    MaxArgs:     0,
})
```

### 5. Handler logic

```go
func handleDumpContextCommand(m *model, args []string) interface{} {
    messages := m.agent.GetMessages()
    contextInfo := m.agent.GetContextInfo()

    // Build output directory: <workspace>/.forge/context/
    outDir := filepath.Join(m.workspaceDir, ".forge", "context")
    if err := os.MkdirAll(outDir, 0o755); err != nil {
        m.showToast("Export Failed", err.Error(), "❌", true)
        return nil
    }

    // Timestamped filename: 2025-07-15T14-30-00Z-context.json
    ts := time.Now().UTC().Format(time.RFC3339)
    ts = strings.NewReplacer(":", "-").Replace(ts)
    filename := ts + "-context.json"
    outPath := filepath.Join(outDir, filename)

    // Assemble payload
    dump := buildContextDump(m.workspaceDir, contextInfo, messages)

    data, err := json.MarshalIndent(dump, "", "  ")
    if err != nil {
        m.showToast("Export Failed", err.Error(), "❌", true)
        return nil
    }
    if err := os.WriteFile(outPath, data, 0o644); err != nil {
        m.showToast("Export Failed", err.Error(), "❌", true)
        return nil
    }

    // Relative path for readability in the toast
    rel, _ := filepath.Rel(m.workspaceDir, outPath)
    m.showToast("Context Exported", rel, "📄", false)
    return nil
}
```

### Data Flow

```
User types /dumpcontext
    ↓
TUI slash-command handler fires (synchronous)
    ↓
m.agent.GetMessages()   → []types.Message (copy from ConversationMemory)
m.agent.GetContextInfo() → ContextInfo (token stats, counts)
    ↓
Build contextDump struct
    ↓
json.MarshalIndent
    ↓
os.WriteFile → <workspace>/.forge/context/<timestamp>-context.json
    ↓
showToast("<relative path>")
```

### Example Output

```json
{
  "exported_at": "2025-07-15T14:30:00Z",
  "workspace_dir": "/home/user/my-project",
  "token_stats": {
    "current": 82341,
    "max": 100000,
    "usage_percent": 82.3,
    "free": 17659
  },
  "message_count": 14,
  "messages": [
    {
      "index": 0,
      "role": "system",
      "content": "You are Forge, an elite coding assistant...",
      "tokens": 4201,
      "metadata": {}
    },
    {
      "index": 1,
      "role": "user",
      "content": "Can you refactor the auth module?",
      "tokens": 9
    },
    {
      "index": 2,
      "role": "assistant",
      "content": "[SUMMARIZED] ## Strategy\nI refactored pkg/auth/...",
      "tokens": 312,
      "metadata": {
        "summarized": true,
        "summary_count": 8,
        "summary_method": "ThresholdSummarization"
      }
    }
  ]
}
```

### `.gitignore` Recommendation

The implementation does not automatically update `.gitignore`. Engineers are expected to add `.forge/` or `.forge/context/` to their project's `.gitignore` if they do not want context snapshots committed. A future enhancement could write a `.gitignore` into `.forge/` on first creation.

### Migration Path

No migration required. This is a new, additive feature. The only breaking change is the addition of `GetMessages()` to the `Agent` interface, which requires any external mock implementations to add a one-line method returning `nil` or an empty slice.

### Timeline

~0.5 day implementation. Slash command skeleton and handler are the main work; the `GetMessages()` interface addition is trivial.

---

## Validation

### Success Metrics

- `/dumpcontext` produces a valid JSON file on first use with no configuration.
- The exported `messages` array is identical (role, content, metadata) to what would be sent on the next API call.
- Token stats in the dump match the values shown in the `/context` overlay within ±1% (tokenizer is deterministic for the same inputs).
- File is written to `<workspace>/.forge/context/` and the path is reported in a toast.

### Testing

- Unit test `buildContextDump()` with a synthetic message list and known `ContextInfo`.
- Integration test: confirm the output file exists after calling the handler with a mock agent.
- Verify that `[SUMMARIZED]` and `[GOAL BATCH]` metadata fields appear correctly in the `metadata` field of exported messages.

---

## Related Decisions

- [ADR-0014](0014-composable-context-management.md) — Context management system (strategies, Manager)
- [ADR-0015](0015-buffered-tool-call-summarization.md) — Tool-call summarization strategy
- [ADR-0018](0018-selective-tool-call-summarization.md) — Selective tool-call summarization
- [ADR-0020](0020-context-information-overlay.md) — `/context` display overlay (the read-only TUI companion to this feature)
- [ADR-0033](0033-notes-viewer-tui-command.md) — `/notes` viewer (precedent for TUI slash commands surfacing agent state)
- [ADR-0041](0041-goal-batch-compaction-strategy.md) — Goal-batch compaction strategy
- [ADR-0042](0042-summarization-model-override.md) — Summarization model override

---

## References

- `pkg/agent/agent.go` — `Agent` interface, `GetContextInfo()` pattern
- `pkg/agent/default.go` — `DefaultAgent.GetContextInfo()` implementation
- `pkg/agent/memory/conversation.go` — `ConversationMemory.GetAll()` (returns a copy)
- `pkg/executor/tui/slash_commands.go` — existing command handler pattern (`handleContextCommand`)
- `pkg/types/types.go` — `Message` struct and `Metadata` field

---

## Notes

**Why `/dumpcontext` and not `/exportcontext`?**

"Dump" implies a low-level, complete, raw capture — the same mental model as a core dump or heap dump. "Export" implies a curated, potentially transformed output. Since the intent is byte-for-byte fidelity to the context as the LLM sees it, "dump" is more precise and sets the right expectations for the engineer using it.

**Why `.forge/context/` and not `~/.forge/context/`?**

The snapshot is meaningful in the context of a specific repository at a specific point in time. Putting it in the workspace keeps it adjacent to the code being debugged, makes it trivial to share (commit or archive the directory), and avoids mixing snapshots from different projects in a global directory. The workspace `.forge/` directory already stores other forge-managed local state (if applicable); this follows that convention.

**Why JSON and not a more compact format?**

JSON is universally readable, diff-able with standard tools (`jq`, `diff`), and directly representable as Go structs via `encoding/json`. The context dump is an infrequently-written debug file, not a hot-path data structure, so compactness is irrelevant. Pretty-printing with `json.MarshalIndent` makes it immediately human-readable in a text editor.

**Last Updated:** 2025-07-15
