# 0046. Long-Term Memory — Async Capture Pipeline

**Status:** Accepted
**Date:** 2025-02-22
**Deciders:** Core Team
**Technical Story:** [Long-Term Persistent Memory System PRD](../product/features/long-term-memory.md)

---

## Context

### Background

Forge currently has no mechanism for persisting knowledge across sessions. The in-session scratchpad (ADR-0032) and conversation memory (ADR-0007) both expire when the session ends.

This ADR defines the capture pipeline: the component responsible for observing conversations, deciding what is worth remembering, and writing memory files to the storage layer (ADR-0044). Capture is fully asynchronous and must never block the main agent loop.

### Problem Statement

The agent loop runs a tight cycle: user message → LLM response → tool calls → LLM response → repeat. Any synchronous work inserted into this cycle adds latency directly visible to the user. Memory capture involves an LLM call (the classifier) which can take multiple seconds — it cannot live in the hot path.

The classifier LLM receives the full conversation window (all messages since the last summarisation event, tool call content excluded) on each trigger. This gives it maximum context to identify patterns, detect corrections, and assess memory-worthiness. The classifier is responsible for its own de-duplication judgement — the pipeline does not attempt to track which messages have already been processed or maintain a sliding window of "new" messages since the last trigger. Tracking such state is unnecessary complexity; the classifier's high-threshold prompt handles noise naturally.

### Goals

- Implement an observer that hooks into the agent loop after each completed turn without blocking it, firing a capture trigger on every turn
- Implement two capture trigger sources: per-turn cadence and goal-based compaction events (ADR-0041)
- Implement the async classifier: an LLM call that determines memory-worthiness, assigns scope/category/relationships, and writes memory files via the store interface (ADR-0044)
- Pass the full conversation window (messages since last summarisation, tool call content excluded) to the classifier on each trigger
- Ensure classifier failures are non-fatal and silently skipped for that trigger
- Post-capture, signal the retrieval engine to rebuild its in-memory vector map asynchronously

### Non-Goals

- Embedding or retrieval (ADR-0045, ADR-0047)
- TUI management commands (P2 / future)
- Memory consolidation (stretch / future)
- Processing tool call content or tool results

---

## Decision Drivers

* Main agent loop must never be blocked by capture — zero latency impact on user interaction
* The classifier receives the full conversation window each turn — context richness comes from the conversation itself, not from batching multiple turns together
* Classifier failures must not crash or degrade the session
* Post-capture re-embedding must trigger automatically but also never block the agent loop
* The compaction trigger provides a natural second pass over a full goal arc with higher signal

---

## Considered Options

### Option 1: Synchronous capture at end of each turn

After each user turn completes, call the classifier before returning control to the user prompt.

**Pros:**
- Simple implementation; no goroutines
- Deterministic timing

**Cons:**
- Classifier latency (1–5 seconds) is directly visible to the user after every N turns
- Unacceptable UX impact — violates the core requirement

### Option 2: Goroutine-per-trigger (fire and forget)

On each turn completion, launch a new goroutine that calls the classifier and writes results.

**Pros:**
- Simple; zero blocking
- No shared state between capture goroutines

**Cons:**
- Concurrent writes to the same store can race if two triggers fire close together (e.g. turn completion and compaction simultaneously)
- Goroutine count is unbounded under pathological conditions
- No backpressure; if classification is slow, multiple classifier calls can overlap

### Option 3: Single persistent capture goroutine with a channel queue

A single long-lived goroutine reads trigger events from a buffered channel. The agent loop sends events non-blocking (dropping if the buffer is full). The goroutine processes events sequentially, ensuring at most one classifier call is in-flight at any time.

**Pros:**
- Zero agent loop blocking (channel send is non-blocking)
- Sequential processing eliminates write races
- Bounded concurrency — one classifier call at a time
- Clean shutdown via context cancellation

**Cons:**
- Slightly more complex setup (goroutine lifecycle management)
- A very slow classifier call can cause a cadence trigger to be dropped if it arrives while the previous is still processing — acceptable given the non-critical nature of capture

---

## Decision

**Chosen Option:** Option 3 — Single persistent capture goroutine with a channel queue

### Rationale

Sequential processing with a non-blocking channel send gives the best combination of simplicity, zero agent loop impact, and safe concurrent state. The risk of dropped triggers under a slow classifier is explicitly acceptable: capture is best-effort. Missing one cadence window does not cause data loss — the next trigger will capture the buffered context. Compaction triggers are higher-priority and are never dropped (channel buffer is sized to hold at least one of each type).

---

## Consequences

### Positive

- Agent loop never blocks on capture — latency impact is zero
- At most one classifier LLM call in-flight at any time — no concurrent writes to store
- Compaction and cadence triggers are both serviced by the same pipeline — unified code path
- Classifier failures drop silently without any session impact

### Negative

- If classifier is slow and a cadence trigger arrives during processing, the trigger is dropped — a conversational window may not be captured. This is acceptable.
- Slightly more complex agent initialisation (goroutine start, channel wiring, context propagation)

### Neutral

- Post-capture re-embedding (rebuild of in-memory vector map) is signalled from the capture goroutine, keeping all async memory I/O on a single goroutine thread

---

## Implementation

### Trigger Types

```go
// pkg/agent/longtermmemory/capture/trigger.go
package capture

// TriggerKind identifies what caused a capture pass to be initiated.
type TriggerKind string

const (
    // TriggerKindTurn fires after every completed user turn.
    TriggerKindTurn       TriggerKind = "turn"
    // TriggerKindCompaction fires when a goal-arc compaction event is raised (ADR-0041).
    TriggerKindCompaction TriggerKind = "compaction"
)

// TriggerEvent carries the context snapshot the capture pipeline should analyse.
type TriggerEvent struct {
    Kind TriggerKind
    // Messages holds the full conversation window: all user and assistant messages
    // since the last summarisation event. Tool call content is stripped by the
    // observer before enqueueing. If no summarisation has occurred, this is the
    // full session history minus tool content.
    Messages  []ConversationMessage
    SessionID string
}

// ConversationMessage is a user or assistant message, stripped of tool content.
type ConversationMessage struct {
    Role    string // "user" or "assistant"
    Content string
}
```

### Observer

The observer wires into the agent loop's turn completion callback and the compaction event bus. It is responsible for:
1. Enqueueing a turn trigger on every completed turn (user message → agent response cycle)
2. Listening for compaction events from ADR-0041 and enqueueing a compaction trigger
3. Stripping tool call content from messages before enqueueing

```go
// pkg/agent/longtermmemory/capture/observer.go
package capture

// Observer hooks into the agent loop and enqueues a capture trigger after
// every completed user turn. There is no cadence counter or modulo check —
// every turn fires a trigger. The classifier is responsible for determining
// what (if anything) is worth remembering from the full conversation window.
type Observer struct {
    pipeline *Pipeline
}

// NewObserver creates an Observer that enqueues a trigger on every turn completion.
func NewObserver(pipeline *Pipeline) *Observer {
    return &Observer{pipeline: pipeline}
}

// OnTurnComplete is called by the agent loop after each completed user turn.
// It is synchronous and must return immediately — all heavy work is deferred
// to the pipeline goroutine.
//
// messages is the full conversation window (all user+assistant messages since
// the last summarisation event, tool call content already stripped).
func (o *Observer) OnTurnComplete(messages []ConversationMessage, sessionID string) {
    o.pipeline.Enqueue(TriggerEvent{
        Kind:      TriggerKindTurn,
        Messages:  messages,
        SessionID: sessionID,
    })
}

// OnCompaction is called by the compaction system (ADR-0041) when a goal-arc
// compaction event fires. It always enqueues a trigger regardless of turn state.
// messages is the full conversation arc, tool content excluded.
func (o *Observer) OnCompaction(messages []ConversationMessage, sessionID string) {
    o.pipeline.Enqueue(TriggerEvent{
        Kind:      TriggerKindCompaction,
        Messages:  messages,
        SessionID: sessionID,
    })
}
```

### Pipeline

```go
// pkg/agent/longtermmemory/capture/pipeline.go
package capture

import (
    "context"
    "log/slog"

    "github.com/entrhq/forge/pkg/agent/longtermmemory"
    "github.com/entrhq/forge/pkg/llm"
)

const triggerBufferSize = 8

// Pipeline is a single long-lived goroutine that receives TriggerEvents
// and runs the classifier asynchronously.
type Pipeline struct {
    ch         chan TriggerEvent
    classifier *Classifier
    store      longtermmemory.MemoryStore
    rebuildFn  func() // called after each successful write to signal vector map rebuild
}

// NewPipeline constructs a Pipeline. rebuildFn is called after each capture
// batch completes; it should signal the retrieval engine to rebuild its vector map.
func NewPipeline(
    classifierLLM llm.LLMProvider,
    classifierModel string,
    store longtermmemory.MemoryStore,
    rebuildFn func(),
) *Pipeline {
    return &Pipeline{
        ch:         make(chan TriggerEvent, triggerBufferSize),
        classifier: NewClassifier(classifierLLM, classifierModel, store),
        store:      store,
        rebuildFn:  rebuildFn,
    }
}

// Start launches the pipeline goroutine. It runs until ctx is cancelled.
func (p *Pipeline) Start(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case event := <-p.ch:
                p.process(ctx, event)
            }
        }
    }()
}

// Enqueue submits a TriggerEvent to the pipeline. Non-blocking: if the buffer
// is full, the event is dropped and a debug log is emitted.
func (p *Pipeline) Enqueue(event TriggerEvent) {
    select {
    case p.ch <- event:
    default:
        slog.Debug("longtermmemory: capture trigger dropped (pipeline busy)", "kind", event.Kind)
    }
}

func (p *Pipeline) process(ctx context.Context, event TriggerEvent) {
    memories, err := p.classifier.Classify(ctx, event)
    if err != nil {
        slog.Debug("longtermmemory: classifier error (capture skipped)", "err", err, "trigger", event.Kind)
        return
    }
    if len(memories) == 0 {
        return
    }
    for _, m := range memories {
        if err := p.store.Write(ctx, m); err != nil {
            slog.Warn("longtermmemory: failed to write memory", "id", m.Meta.ID, "err", err)
        }
    }
    // Signal the retrieval engine to rebuild its in-memory vector map.
    // rebuildFn must be non-blocking (it should launch its own goroutine internally).
    if p.rebuildFn != nil {
        p.rebuildFn()
    }
}
```

### Classifier

The classifier formats the conversation window as a structured prompt and sends it to the configured LLM. The response is a JSON array of memory objects which is parsed and converted to `MemoryFile` instances.

```go
// pkg/agent/longtermmemory/capture/classifier.go
package capture

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/entrhq/forge/pkg/agent/longtermmemory"
    "github.com/entrhq/forge/pkg/llm"
)

// Classifier uses an LLM to identify memory-worthy content from a conversation window.
type Classifier struct {
    provider llm.LLMProvider
    model    string
    store    longtermmemory.MemoryStore // used to load existing memories for supersedes/related resolution
}

// NewClassifier constructs a Classifier that uses the given LLM provider and model.
// store is used to load existing memories before each LLM call so the classifier
// can reference real memory IDs in supersedes and related fields.
func NewClassifier(provider llm.LLMProvider, model string, store longtermmemory.MemoryStore) *Classifier {
    return &Classifier{provider: provider, model: model, store: store}
}

// classifiedMemory is the JSON structure the classifier LLM returns.
type classifiedMemory struct {
    Content    string                           `json:"content"`
    Scope      longtermmemory.Scope             `json:"scope"`
    Category   longtermmemory.Category          `json:"category"`
    // Supersedes is the ID of the memory this one replaces, if any.
    // Omitted from JSON when empty so the LLM can leave it out cleanly.
    Supersedes string                           `json:"supersedes,omitempty"`
    Related    []longtermmemory.RelatedMemory   `json:"related,omitempty"`
}

// validScopes and validCategories are the allowed values for LLM-returned fields.
// Any value outside these sets is rejected with a debug log.
var validScopes = map[longtermmemory.Scope]struct{}{
    longtermmemory.ScopeRepo: {},
    longtermmemory.ScopeUser: {},
}

var validCategories = map[longtermmemory.Category]struct{}{
    longtermmemory.CategoryCodingPreferences:      {},
    longtermmemory.CategoryProjectConventions:     {},
    longtermmemory.CategoryArchitecturalDecisions: {},
    longtermmemory.CategoryUserFacts:              {},
    longtermmemory.CategoryCorrections:            {},
    longtermmemory.CategoryPatterns:               {},
}

// Classify sends the conversation window to the classifier LLM and returns
// zero or more MemoryFiles to be written. Returns nil, nil if nothing is memory-worthy.
//
// Before calling the LLM, Classify loads all existing memories from both scopes
// so the classifier can reference real memory IDs when populating supersedes and
// related fields. This is the only way supersedes/related links can be populated
// with valid IDs — the LLM has no other source of existing memory identity.
func (c *Classifier) Classify(ctx context.Context, event TriggerEvent) ([]*longtermmemory.MemoryFile, error) {
    // --- Phase 1: load existing memories for the prompt ---
    // List memories from both scopes. Failures are non-fatal; we log and continue
    // with an empty list, which means the LLM will produce no supersedes links.
    existing, err := c.loadExistingMemories(ctx)
    if err != nil {
        slog.Debug("longtermmemory: failed to load existing memories for classifier (supersedes disabled for this turn)", "err", err)
        existing = nil
    }

    // --- Phase 2: call the classifier LLM ---
    prompt := buildClassifierPrompt(event, existing)
    response, err := c.provider.Complete(ctx, c.model, classifierSystemPrompt, prompt)
    if err != nil {
        return nil, fmt.Errorf("classifier: LLM call failed: %w", err)
    }

    var classified []classifiedMemory
    if err := json.Unmarshal([]byte(response), &classified); err != nil {
        // LLM returned non-JSON or empty — treat as nothing memory-worthy
        return nil, nil
    }

    // --- Phase 3: validate, resolve predecessor version, construct MemoryFiles ---
    // Build a lookup map for O(1) predecessor access.
    existingByID := make(map[string]*longtermmemory.MemoryFile, len(existing))
    for _, m := range existing {
        existingByID[m.Meta.ID] = m
    }

    now := time.Now().UTC()
    var out []*longtermmemory.MemoryFile
    for _, cm := range classified {
        // Validate scope and category against defined constants.
        // An LLM hallucinating "global" or "misc" must not produce an invalid file.
        if _, ok := validScopes[cm.Scope]; !ok {
            slog.Debug("longtermmemory: classifier returned unknown scope (skipping)", "scope", cm.Scope)
            continue
        }
        if _, ok := validCategories[cm.Category]; !ok {
            slog.Debug("longtermmemory: classifier returned unknown category (skipping)", "category", cm.Category)
            continue
        }

        // MemoryMeta.Supersedes is *string (nil == first version, no predecessor).
        // Convert the classifier's plain string field to a pointer, or nil if absent.
        var supersedes *string
        if cm.Supersedes != "" {
            s := cm.Supersedes
            supersedes = &s
        }

        // Resolve version: use predecessor.Version + 1 so the chain is always
        // monotonically increasing, even if the predecessor has been superseded
        // multiple times already. Fall back to version 1 if no predecessor.
        version := 1
        if supersedes != nil {
            if predecessor, ok := existingByID[*supersedes]; ok {
                version = predecessor.Meta.Version + 1
            } else {
                // LLM referenced an ID that no longer exists or was hallucinated.
                // Clear the supersedes link rather than write a dangling reference.
                slog.Debug("longtermmemory: classifier supersedes ID not found (clearing link)", "id", *supersedes)
                supersedes = nil
            }
        }

        m := &longtermmemory.MemoryFile{
            Meta: longtermmemory.MemoryMeta{
                ID:         longtermmemory.NewMemoryID(),
                CreatedAt:  now,
                UpdatedAt:  now,
                Version:    version,
                Scope:      cm.Scope,
                Category:   cm.Category,
                Supersedes: supersedes,
                Related:    cm.Related,
                SessionID:  event.SessionID,
                Trigger:    longtermmemory.Trigger(event.Kind),
            },
            Content: cm.Content,
        }
        out = append(out, m)
    }
    return out, nil
}

// loadExistingMemories retrieves all memories from both scopes.
// Returns a combined slice; partial failures (one scope unavailable) are tolerated.
func (c *Classifier) loadExistingMemories(ctx context.Context) ([]*longtermmemory.MemoryFile, error) {
    repoMems, repoErr := c.store.ListByScope(ctx, longtermmemory.ScopeRepo)
    userMems, userErr := c.store.ListByScope(ctx, longtermmemory.ScopeUser)
    if repoErr != nil && userErr != nil {
        return nil, fmt.Errorf("classifier: could not load either scope: repo=%w user=%v", repoErr, userErr)
    }
    combined := make([]*longtermmemory.MemoryFile, 0, len(repoMems)+len(userMems))
    combined = append(combined, repoMems...)
    combined = append(combined, userMems...)
    return combined, nil
}
```

### Classifier System Prompt

The system prompt is the most critical piece of the capture pipeline. It must instruct the classifier to:
- Apply a high memory-worthiness threshold (not every exchange is worth remembering)
- Correctly assign scope (repo vs user) and category
- Detect when a new memory supersedes an existing one (existing memories with their IDs are injected into the prompt by `Classify()` before the LLM call)
- Draw relationship edges thoughtfully
- Never capture sensitive literals (credentials, tokens, personal identifiers)
- Return valid JSON or an empty array

```
You are a memory classifier for an AI coding assistant called Forge.
Your job is to read a window of conversation and identify information that is
worth remembering permanently across sessions.

MEMORY-WORTHINESS CRITERIA (all must be met):
- The information represents a durable preference, convention, decision, or correction
- It is not transient (e.g. "run this command once" is not worth remembering)
- It is not already obvious from the codebase itself
- It is not sensitive (never capture credentials, tokens, passwords, or personal identifiers)

SCOPE ASSIGNMENT:
- scope: "repo" — information specific to the current project (conventions, architecture, patterns)
- scope: "user" — information about how this user works across all projects (preferences, style, habits)

CATEGORY ASSIGNMENT:
- coding-preferences — style, idiom, tooling choices
- project-conventions — repo-specific patterns and standards
- architectural-decisions — design decisions with rationale
- user-facts — facts about how the user works or thinks
- corrections — mistakes the agent made and was corrected on
- patterns — non-obvious relationships or recurring structures

RELATIONSHIP EDGES:
If a new memory refines, contradicts, or supersedes an existing memory listed in
EXISTING MEMORIES below, include the relationship in the "related" or "supersedes"
field using the exact ID from that list. Do not invent IDs.

OUTPUT FORMAT:
Return a JSON array of memory objects. Return an empty array [] if nothing is memory-worthy.
Each object must have: content (markdown string), scope, category.
Optional fields: supersedes (memory ID string), related (array of {id, relationship}).

Example:
[
  {
    "content": "User prefers `errors.As` over type assertions for error handling in all Go code.",
    "scope": "user",
    "category": "coding-preferences"
  }
]
```

`buildClassifierPrompt(event TriggerEvent, existing []*longtermmemory.MemoryFile) string` constructs the user-turn prompt. It formats the stripped conversation messages and appends an `EXISTING MEMORIES` section listing each memory as:

```
- [<id>] (<scope>/<category>) <first line of content>
```

If `existing` is empty the section is omitted entirely (no "no memories yet" noise). The existing memory list is intentionally a compact summary — only the first line of content — to avoid consuming the classifier's context budget. The LLM uses the IDs from this list verbatim when populating `supersedes` or `related` fields.

### Tool Call Content Stripping

The observer strips tool call content from messages before enqueueing. This is a preprocessing step that filters the message list to include only `role: "user"` and `role: "assistant"` messages, and further removes any assistant message segments that are tool invocations or tool results.

```go
// pkg/agent/longtermmemory/capture/strip.go
package capture

import "github.com/entrhq/forge/pkg/types"

// StripToolContent filters a message list to retain only human-language
// user and assistant content. Tool call blocks and tool result blocks are removed.
func StripToolContent(messages []types.Message) []ConversationMessage {
    var out []ConversationMessage
    for _, msg := range messages {
        if msg.Role != "user" && msg.Role != "assistant" {
            continue
        }
        text := extractTextContent(msg)
        if text == "" {
            continue
        }
        out = append(out, ConversationMessage{Role: msg.Role, Content: text})
    }
    return out
}

// extractTextContent returns the text portions of a message, excluding
// tool call syntax and tool result blocks.
func extractTextContent(msg types.Message) string {
    // Implementation depends on the internal message representation.
    // For XML tool call format (ADR-0019), strip content between <tool> tags.
    // Returns empty string if the message is purely a tool call with no prose.
    // ... (implementation detail)
    return msg.TextContent() // assumes types.Message exposes a TextContent() helper
}
```

### Package Layout

```
pkg/agent/longtermmemory/capture/
    trigger.go      — TriggerKind, TriggerEvent, ConversationMessage types
    observer.go     — Observer (turn counter + compaction hook)
    pipeline.go     — Pipeline (goroutine, channel, process loop)
    classifier.go   — Classifier (LLM call, JSON parse, MemoryFile construction)
    strip.go        — StripToolContent()
    prompt.go       — classifierSystemPrompt, buildClassifierPrompt()
```

### Wiring at Agent Startup

```go
// Pseudocode — agent initialisation
store, _ := longtermmemory.NewFileStore(repoMemoryDir, userMemoryDir)

pipeline := capture.NewPipeline(
    llmProvider,
    cfg.Memory.ClassifierModel,
    store,
    retrievalEngine.TriggerRebuild, // non-blocking; signals async re-embed
)
pipeline.Start(ctx)

observer := capture.NewObserver(pipeline)

// Wire observer into agent loop turn completion callback
agentLoop.OnTurnComplete = observer.OnTurnComplete

// Wire observer into compaction event bus (ADR-0041)
compactionBus.Subscribe(observer.OnCompaction)
```

### Migration Path

No migration required. The capture pipeline writes to newly created directories (`.forge/memory/`, `~/.forge/memory/`) that do not exist in existing installations. No existing agent loop code is modified — the observer is wired via the existing callback/event-bus pattern.

---

## Validation

### Success Metrics

- `Pipeline.Enqueue` never blocks the caller; drops trigger with a debug log if buffer is full
- Classifier LLM failure causes the turn's capture to be silently skipped; next trigger proceeds normally
- Post-capture `rebuildFn` is called after each successful batch write
- Tool call content is absent from all classifier prompts (verified in unit tests)
- Compaction trigger always enqueues regardless of turn counter state

### Monitoring

- Debug log on dropped trigger (buffer full)
- Debug log on classifier LLM error (capture skipped for this trigger)
- Warn log on individual memory write failure (partial batch write succeeds for other memories)

---

## Related Decisions

- [ADR-0007](0007-memory-system-design.md) — in-session conversation memory (source of message history)
- [ADR-0041](0041-goal-batch-compaction-strategy.md) — compaction events (second capture trigger)
- [ADR-0042](0042-summarization-model-override.md) — classifier model config pattern
- [ADR-0044](0044-long-term-memory-storage.md) — storage layer (prerequisite; capture writes here)
- [ADR-0045](0045-long-term-memory-embedding-provider.md) — embedding provider (capture signals rebuild)
- [ADR-0047](0047-long-term-memory-retrieval.md) — retrieval engine (receives rebuild signal)

---

## References

- [Long-Term Memory PRD](../product/features/long-term-memory.md)
- [ADR-0019](0019-xml-cdata-tool-call-format.md) — XML tool call format (guides tool content stripping)

---

## Notes

The classifier prompt instructs the LLM to return an empty JSON array `[]` when nothing is memory-worthy. This is the expected common case — the classifier should have a high threshold. Returning `[]` is not an error; it is the correct response for most conversation windows.

The `rebuildFn` passed to `NewPipeline` must itself be non-blocking. The retrieval engine (ADR-0047) implements `TriggerRebuild()` as a goroutine launch, ensuring the capture goroutine is never held waiting for embedding to complete.

**Last Updated:** 2025-02-22
