# 0033. Notes Viewer TUI Command

**Status:** Proposed
**Date:** 2024-12-19
**Deciders:** Engineering Team
**Technical Story:** Implementation of `/notes` slash command to view agent scratchpad notes in TUI

---

## Context

The agent scratchpad notes system (ADR-0032) provides persistent working memory during task execution, allowing agents to record insights, decisions, and patterns. However, users currently have no way to view their notes within the TUI.

### Background

Notes can only be examined indirectly through:
- Tool execution results in conversation history
- Manual review of agent responses mentioning notes
- No direct visibility into what the agent has learned

The `/context` command successfully demonstrates agent state visualization using TUI overlays, providing a proven pattern for exposing internal agent data to users.

### Problem Statement

Users need a way to inspect agent working memory (notes) to understand the agent's reasoning, debug issues, and track multi-step task progress, but currently have no direct access to the notes system from the TUI.

### Goals

- Provide read-only viewing of agent scratchpad notes
- Enable filtering by tags to find relevant notes quickly
- Integrate seamlessly with existing TUI slash command interface
- Follow established overlay UI patterns for consistency

### Non-Goals

- Real-time auto-refresh when agent modifies notes (deferred to future)
- Note editing or deletion from TUI (deferred to future)
- Advanced search with regex or full-text queries (deferred to future)
- Note export to file functionality (separate feature)
- Persistent sidebar showing notes continuously (major UX change)

---

## Decision Drivers

* **Transparency** - Users should be able to inspect agent working memory to understand reasoning
* **Consistency** - Implementation must follow established TUI patterns (overlays, similar to `/context`)
* **Simplicity** - Initial implementation should focus on core viewing functionality, avoid over-engineering
* **Maintainability** - Should integrate cleanly with existing agent and TUI architecture
* **Low Implementation Cost** - Reuse existing components where possible to deliver value quickly

## Considered Options

### Option 1: Notes Overlay with Direct Manager Access

**Description:** Add `/notes` slash command that creates a read-only overlay displaying notes, accessing the notes manager through a new agent interface method.

**Implementation:**
- Extend agent interface with `GetNotesManager() *notes.Manager`
- Implement in `DefaultAgent` to expose existing notes manager
- Create `overlay.NotesOverlay` component for display
- Register `/notes` handler in slash command registry
- Display notes in overlay with filtering by tag

**Pros:**
- Follows established `/context` command pattern
- Minimal new abstractions - reuses existing components
- Direct access to notes data without transformation
- Low implementation cost (~200 lines of code)

**Cons:**
- **Violates EDA principles** - creates direct coupling between layers
- Exposes internal notes.Manager structure to TUI
- Couples TUI to notes package implementation details
- No abstraction layer if notes storage changes
- Inconsistent with event-driven patterns used elsewhere

### Option 2: Notes Query Service with DTO Layer

**Description:** Create a notes query service that returns DTOs (Data Transfer Objects) instead of exposing the manager directly.

**Implementation:**
- Create `NotesQueryService` interface with methods like `ListNotes(opts)`, `SearchNotes(query)`
- Implement service in agent package
- Define DTO structs for notes data
- TUI depends only on service interface and DTOs

**Pros:**
- Decouples TUI from notes implementation
- Provides abstraction for future storage changes
- Can evolve notes system without affecting TUI

**Cons:**
- Additional abstraction layer for simple read-only access
- Mapping between internal types and DTOs adds complexity
- Over-engineering for current requirements
- More files and interfaces to maintain

### Option 3: Event-Based Notes Retrieval

**Description:** Use request/response event pattern to fetch notes data when user invokes `/notes` command.

**Implementation:**
- Define `RequestNotesEvent` with filter options (tag, include_scratched, limit)
- Define `NotesDataEvent` containing list of notes matching the request
- TUI emits `RequestNotesEvent` when user types `/notes`
- Agent handles event, queries notes manager, emits `NotesDataEvent` with results
- TUI receives `NotesDataEvent` and displays overlay
- No caching required - fresh data on each invocation

**Pros:**
- Consistent with existing EDA patterns throughout codebase
- Clean separation between TUI and agent internals
- TUI only depends on event contracts, not implementation details
- Naturally supports future real-time updates if needed
- No coupling to notes.Manager structure

**Cons:**
- Requires defining two new event types
- Async request/response pattern slightly more complex than direct call
- Small latency for event round-trip (negligible for user interaction)

---

## Decision

**Chosen Option:** Option 3 - Event-Based Notes Retrieval

### Rationale

This option aligns with the system's event-driven architecture (EDA) and maintains proper separation of concerns between the TUI and agent layers. Rather than coupling the TUI directly to internal agent structures, we use the established event system for communication.

Key factors:
- Consistent with existing EDA patterns throughout the codebase
- Maintains clean separation between TUI and agent internals
- TUI only depends on event contracts, not implementation details
- Scalable pattern that supports future enhancements (real-time updates, etc.)
- Avoids exposing agent internal structures to TUI layer

---

## Consequences

### Positive

- Users can view agent working memory with familiar slash command interface
- Maintains clean separation between TUI and agent layers
- TUI only depends on Input/Event contracts, not implementation details
- Reuses existing Input channel - no new channel infrastructure
- Follows established InputType pattern consistently
- Pattern scales to other agent state viewing needs
- Helps users understand agent decision-making and debug issues
- Easy to extend with filtering and real-time updates

### Negative

- New InputType and event type to maintain
- Async request/response pattern slightly more complex than direct call
- Small latency for round-trip (negligible for user interaction)
- Type assertions needed to extract params from Input.Metadata
- Initial version is read-only (no editing or deleting notes)

### Neutral

- Similar implementation cost to direct access (~250 lines)
- Input/Event pattern provides foundation for future enhancements

---

## Implementation

### High-Level Plan

1. **Event Type Definitions** (`pkg/types/events.go`):
   ```go
   // RequestNotesEvent requests notes data from the agent
   type RequestNotesEvent struct {
       Tag              string // Optional tag filter
       IncludeScratched bool   // Include scratched notes
       Limit            int    // Max notes to return (default: 10)
   }

   // NotesDataEvent contains notes data response
   type NotesDataEvent struct {
       Notes            []NoteData // List of notes matching request
       TotalCount       int        // Total notes matching filter
       ActiveCount      int        // Count of active (non-scratched) notes
   }

   // NoteData represents a single note for display
   type NoteData struct {
       ID        string
       Content   string
       Tags      []string
       Scratched bool
       CreatedAt time.Time
       UpdatedAt time.Time
   }
   ```

2. **Event Handler in Agent** (`pkg/agent/event_handlers.go` or `pkg/agent/default.go`):
   ```go
   func (a *DefaultAgent) handleRequestNotes(req *RequestNotesEvent) {
       // Query notes manager with provided options
       opts := notes.ListOptions{
           Tag:              req.Tag,
           IncludeScratched: req.IncludeScratched,
           Limit:            req.Limit,
       }
       notesList := a.notesManager.List(opts)

       // Convert to event data
       notesData := make([]NoteData, len(notesList))
       for i, note := range notesList {
           notesData[i] = NoteData{
               ID:        note.ID,
               Content:   note.Content,
               Tags:      note.Tags,
               Scratched: note.Scratched,
               CreatedAt: note.CreatedAt,
               UpdatedAt: note.UpdatedAt,
           }
       }

       // Emit response event
       a.emitEvent(&NotesDataEvent{
           Notes:       notesData,
           TotalCount:  len(notesData),
           ActiveCount: a.notesManager.CountActive(),
       })
   }
   ```

3. **Notes Overlay Component** (`pkg/executor/tui/overlay/notes.go`):
   - Accept `[]types.NoteData` in constructor
   - Display notes in scrollable list
   - Show note ID, tags, content, timestamps
   - Format similar to context overlay

4. **TUI Event Loop Integration** (`pkg/executor/tui/model.go`):
   - Add handler for `NotesDataEvent` in event processing
   - When event received, create and activate notes overlay

5. **Slash Command Handler** (`pkg/executor/tui/slash_commands.go`):
   ```go
   func handleNotesCommand(m *model, args []string) interface{} {
       // Parse command arguments
       tag := parseTagArg(args)
       includeScratched := hasFlag(args, "--all")

       // Emit request event
       m.agent.GetChannels().Input <- &types.Input{
           Event: &types.RequestNotesEvent{
               Tag:              tag,
               IncludeScratched: includeScratched,
               Limit:            10,
           },
       }

       // Overlay will be shown when NotesDataEvent is received
       return nil
   }
   ```

6. **Overlay Mode** (`pkg/types/tui.go`):
   - Add `OverlayModeNotes` constant

### Data Flow

1. User types `/notes` (optionally with tag filter or --all flag)
2. TUI parses command arguments and emits `RequestNotesEvent` to agent
3. Agent event loop receives `RequestNotesEvent`
4. Agent queries notes manager with specified filters
5. Agent converts notes to `NoteData` structs
6. Agent emits `NotesDataEvent` with notes data
7. TUI receives `NotesDataEvent` in event loop
8. TUI creates `NotesOverlay` with received data
9. TUI activates overlay for display
10. User navigates/dismisses overlay

### Display Format

```
╔══════════════════════════════════════════════════════════════════╗
║ 📝 Agent Notes                                   [ESC to close] ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                  ║
║ Active Notes (5)                                                ║
║ ────────────────────────────────────────────────────────────────║
║                                                                  ║
║ • API authentication uses OAuth2 flow                           ║
║   Tags: [auth] [api] [decision]                                ║
║   ID: abc123 | Updated: 2m ago                                  ║
║                                                                  ║
║ • Database connection pooling configuration needed              ║
║   Tags: [database] [performance] [todo]                        ║
║   ID: def456 | Updated: 5m ago                                  ║
║                                                                  ║
╚══════════════════════════════════════════════════════════════════╝
```

### Command Syntax

- `/notes` - Show all active notes (default limit: 10)
- `/notes tag:auth` - Show notes tagged with "auth"
- `/notes --all` - Include scratched notes

### Migration Path

No migration required - this is a new feature addition.

### Timeline

Target implementation: Sprint following ADR approval (~1-2 days development)

---

## Validation

### Success Metrics

- Users successfully view notes using `/notes` command without errors
- Overlay rendering performs well with up to 100 notes
- Tag filtering correctly reduces displayed notes
- Users report improved understanding of agent reasoning in feedback

### Monitoring

- Track `/notes` command usage frequency in telemetry
- Monitor for error reports related to notes viewing
- Gather user feedback on usefulness and desired enhancements

---

## Related Decisions

- [ADR-0032](0032-agent-scratchpad-notes-system.md) - Agent Scratchpad Notes System (notes package architecture)
- [ADR-0012](0012-enhanced-tui-executor.md) - Enhanced TUI Executor (overlay architecture)
- [ADR-0009](0009-tui-executor-design.md) - TUI Executor Design (slash command system)

---

## References

- Implementation reference: `/context` command in `pkg/executor/tui/slash_commands.go`
- PRD: Notes Viewer Command (product requirements document)
- Notes Manager API: `pkg/agent/memory/notes/manager.go`

---

## Notes

**Deferred Features** (explicitly out of scope for initial implementation):

1. Real-time auto-refresh when agent modifies notes
2. Note editing or deletion from TUI
3. Advanced search with regex or full-text queries
4. Note export to file functionality
5. Persistent sidebar showing notes continuously

These can be addressed in future ADRs if user demand justifies the complexity.

**Last Updated:** 2024-12-19
