# 0006. Agent Scratchpad & Notes System

**Status:** Proposed
**Date:** 2025-12-01
**Deciders:** Engineering Team
**Technical Story:** Implementation of persistent scratchpad memory for agent task execution

---

## Context

The Forge agent currently lacks a persistent workspace for managing ephemeral notes during task execution. While the conversation memory tracks dialogue history and tool results, there is no dedicated mechanism for the agent to store working state, organize information, or build context incrementally across multiple tool calls.

### Background

The current agent architecture includes:
- **Conversation Memory**: Tracks dialogue history with token budget constraints
- **Tool Results**: Ephemeral outputs that can only be recalled through conversation context
- **No Persistent Workspace**: No mechanism for storing intermediate findings or task state

This limitation becomes apparent in complex scenarios requiring state accumulation across multiple steps.

### Problem Statement

Agents need a dedicated scratchpad system to:
1. Store working state across multiple tool calls within a session
2. Organize information gathered from various sources (file reads, searches, command outputs)
3. Build context incrementally for complex multi-step tasks
4. Reference past findings without re-executing expensive operations
5. Maintain task-specific notes separate from conversation history

### Goals

- Provide session-scoped note storage outside conversation memory token budgets
- Enable CRUD operations and search capabilities through tool-based interface
- Support tag-based organization for flexible categorization
- Maintain clean separation from core agent interface
- Ensure agent can learn usage patterns through tool schemas

### Non-Goals

- Cross-session persistence (notes are session-scoped only)
- Advanced full-text indexing (substring search is sufficient for MVP)
- Real-time collaboration between multiple agents
- External note export/import (future enhancement)

---

## Decision Drivers

* **Token Budget Constraints**: Conversation memory has limited capacity; notes should not consume token budget
* **Tool-Based Architecture**: Consistent with Forge's tool-centric design philosophy
* **Simplicity**: Minimize complexity for MVP; avoid over-engineering
* **Performance**: In-memory operations should be fast with minimal overhead
* **Learning Curve**: Agent should discover and learn usage through tool schemas
* **Maintainability**: Clean package structure and separation of concerns

---

## Considered Options

### Option 1: Add Notes to Conversation Memory

**Description:** Extend the existing ConversationMemory to store notes alongside messages.

**Pros:**
- No new packages or tools needed
- Reuses existing memory infrastructure
- Simple integration

**Cons:**
- Notes consume token budget (defeats primary goal)
- Pollutes conversation history with non-conversational data
- No CRUD operations or search capabilities
- Violates separation of concerns

### Option 2: Direct Agent Interface Methods

**Description:** Add note management methods directly to the Agent interface (e.g., `AddNote()`, `SearchNotes()`).

**Pros:**
- Programmatic access from code
- Type-safe API
- Fast execution

**Cons:**
- Breaks tool-based architecture pattern
- No visibility in conversation/audit trail
- Agent cannot learn usage (no tool schemas)
- Increases Agent interface complexity

### Option 3: Tool-Based Scratchpad System (Chosen)

**Description:** Implement notes as a separate memory subsystem accessible only through dedicated tools.

**Pros:**
- Notes live outside token budget constraints
- Consistent tool-based interaction pattern
- Full audit trail in conversation
- Agent learns through tool schemas
- Clean separation of concerns
- CRUD + search capabilities

**Cons:**
- Requires 7 new tools (increases system prompt size)
- Agent must learn when/how to use effectively
- No programmatic access from code

---

## Decision

**Chosen Option:** Option 3 - Tool-Based Scratchpad System

We will implement an **Agent Scratchpad & Notes System** as a session-scoped, tool-based memory layer that provides CRUD operations and search capabilities for ephemeral notes.

### Rationale

This option best aligns with Forge's tool-centric architecture while solving the core problem:

1. **Separation of Concerns**: Notes exist outside conversation memory and token budgets
2. **Consistency**: Tool-based access matches established patterns (read_file, write_file, etc.)
3. **Discoverability**: Conditional system prompt teaches agents when/how to use the feature
4. **Audit Trail**: All note operations appear in conversation history
5. **Simplicity**: In-memory, session-scoped storage minimizes complexity

### Architecture

#### 1. Package Structure

Refactor the memory package to support specialized memory types:

```
pkg/agent/memory/
├── conversation/           # Existing conversation memory
│   └── memory.go          # ConversationMemory (moved from memory.go)
└── notes/                 # New scratchpad system
    ├── manager.go         # NotesManager with CRUD + search
    ├── note.go            # Note struct, validation, ID generation
    └── manager_test.go    # Comprehensive tests

pkg/tools/scratchpad/      # 7 scratchpad tools
├── add_note.go           # Create new note with content & tags
├── search_notes.go       # Full-text search with tag filtering
├── list_notes.go         # List all notes (optionally by tag)
├── update_note.go        # Modify existing note
├── delete_note.go        # Remove note by ID
├── scratch_note.go       # Quick note without tags
└── list_tags.go          # List all tags in use
```

#### 2. Tool-Based Access Only

The scratchpad system is **exclusively accessible through tools**. We will NOT add direct methods to the `Agent` interface or expose the `NotesManager` publicly. This design:

- Maintains clean separation of concerns
- Keeps the agent's core interface minimal
- Allows the LLM to learn usage through tool schemas
- Provides consistent interaction patterns (all via tool calls)

#### 3. Tool Registration Pattern

Following the established pattern in `cmd/forge/main.go`:

```go
// Create notes manager
notesManager := notes.NewManager()

// Create scratchpad tools with shared manager
scratchpadTools := []tools.Tool{
    scratchpad.NewAddNoteTool(notesManager),
    scratchpad.NewSearchNotesTool(notesManager),
    scratchpad.NewListNotesTool(notesManager),
    scratchpad.NewUpdateNoteTool(notesManager),
    scratchpad.NewDeleteNoteTool(notesManager),
    scratchpad.NewScratchNoteTool(notesManager),
    scratchpad.NewListTagsTool(notesManager),
}

// Register each tool
for _, tool := range scratchpadTools {
    if err := agent.RegisterTool(tool); err != nil {
        return fmt.Errorf("failed to register scratchpad tool: %w", err)
    }
}
```

#### 4. System Prompt Integration

Scratchpad usage guidelines will be added to the **base system prompt conditionally**—only when scratchpad tools are registered. This keeps the prompt clean and relevant.

Implementation approach:
- Add a new prompt section `ScratchpadGuidancePrompt` in `pkg/agent/prompts/static.go`
- Modify `PromptBuilder` to detect registered scratchpad tools
- Include guidance section only when any `tools/scratchpad/*` tool is present

Example guidance section:

```xml
<scratchpad_system>
You have access to a persistent scratchpad for managing ephemeral notes during task execution.

**When to use the scratchpad:**
- Storing findings from file analysis or searches
- Tracking multi-step task progress
- Building context incrementally
- Avoiding redundant expensive operations
- Organizing complex information

**Best practices:**
- Use descriptive tags for easy filtering
- Keep notes focused and actionable
- Update notes as information evolves
- Search before creating duplicates
- Clean up notes when tasks complete

**Available operations:**
add_note, search_notes, list_notes, update_note, delete_note, scratch_note, list_tags
</scratchpad_system>
```

#### 5. Core Data Model

```go
// Note represents a single scratchpad entry
type Note struct {
    ID        string    // "note_" + timestamp_ms
    Content   string    // Max 4000 chars (configurable)
    Tags      []string  // Max 10 tags (configurable)
    CreatedAt time.Time
    UpdatedAt time.Time
}

// Manager handles CRUD operations and search
type Manager struct {
    notes map[string]*Note  // ID -> Note
    // No mutex - sequential tool execution
}
```

**Design rationale:**
- **No mutex**: Tools execute sequentially in the agent loop; concurrent access is impossible
- **Simple ID scheme**: `note_` prefix + Unix milliseconds provides uniqueness and sortability
- **Fixed limits**: 4000 chars prevents abuse, 10 tags balances flexibility vs. complexity
- **In-memory only**: Session-scoped, no persistence (cleared on restart)

#### 6. Search Implementation

Full-text search with tag filtering:

```go
func (m *Manager) Search(query string, tags []string) []*Note {
    var results []*Note
    
    for _, note := range m.notes {
        // Tag filtering (AND logic - all must match, case-insensitive)
        if !matchesTags(note.Tags, tags) {
            continue
        }
        
        // Content search (substring, case-insensitive)
        if query != "" && !strings.Contains(
            strings.ToLower(note.Content),
            strings.ToLower(query),
        ) {
            continue
        }
        
        results = append(results, note)
    }
    
    // Sort by relevance (query match position) then recency
    sortResults(results, query)
    return results
}
```

**Capabilities:**
- Substring matching (not full-text indexing)
- Case-insensitive for both query and tags
- Combined query + tag filtering
- Relevance scoring based on match position
- Fallback to recency for ties

#### 7. Validation & Error Messages

Detailed, instructive error messages guide the agent:

```go
// Example: Character limit exceeded
return "", nil, fmt.Errorf(
    "note content exceeds maximum length of %d characters (got %d). "+
    "Please shorten the content or split into multiple notes",
    MaxContentLength, len(content),
)

// Example: Too many tags
return "", nil, fmt.Errorf(
    "note has too many tags (max %d, got %d). "+
    "Please reduce the number of tags to focus on the most relevant categories",
    MaxTags, len(tags),
)
```

This approach helps the LLM self-correct and learn the system constraints.

#### 8. Tool Metadata

All note operations return metadata for agent awareness:

```go
// Add/Update operations
metadata := map[string]interface{}{
    "note_id":     note.ID,
    "total_notes": len(m.notes),
    "total_tags":  countUniqueTags(m.notes),
}

// Search operations
metadata := map[string]interface{}{
    "result_count": len(results),
    "query":        query,
    "tags":         tags,
}

// List operations
metadata := map[string]interface{}{
    "note_count": len(results),
    "tag_filter": tagFilter,
}
```

This provides the agent with real-time state awareness.

#### 9. Session Lifecycle

Notes are **session-scoped**:
- Created when the agent starts (via `NewManager()`)
- Cleared when the agent terminates
- No persistence across restarts
- No cross-session sharing

Future enhancement: Add optional persistence layer for note export/import.

### Tool Specifications

#### 1. add_note
```xml
<tool>
  <server_name>local</server_name>
  <tool_name>add_note</tool_name>
  <arguments>
    <content>Note content here</content>
    <tags>
      <tag>analysis</tag>
      <tag>bug-tracking</tag>
    </tags>
  </arguments>
</tool>
```

**Parameters:**
- `content` (required): String, max 4000 chars
- `tags` (optional): Array of strings, max 10 tags

**Returns:** Note ID, success message, metadata

#### 2. search_notes
```xml
<tool>
  <server_name>local</server_name>
  <tool_name>search_notes</tool_name>
  <arguments>
    <query>search term</query>
    <tags>
      <tag>analysis</tag>
    </tags>
  </arguments>
</tool>
```

**Parameters:**
- `query` (optional): Search string
- `tags` (optional): Array of tag filters (AND logic)

**Returns:** Matching notes with relevance ranking, metadata

#### 3. list_notes
```xml
<tool>
  <server_name>local</server_name>
  <tool_name>list_notes</tool_name>
  <arguments>
    <tag>optional-tag-filter</tag>
  </arguments>
</tool>
```

**Parameters:**
- `tag` (optional): Single tag filter

**Returns:** All notes (optionally filtered), sorted by recency, metadata

#### 4. update_note
```xml
<tool>
  <server_name>local</server_name>
  <tool_name>update_note</tool_name>
  <arguments>
    <id>note_1234567890</id>
    <content>Updated content</content>
    <tags>
      <tag>updated</tag>
    </tags>
  </arguments>
</tool>
```

**Parameters:**
- `id` (required): Note ID
- `content` (optional): New content (replaces existing)
- `tags` (optional): New tags (replaces existing)

**Returns:** Success message, updated note, metadata

#### 5. delete_note
```xml
<tool>
  <server_name>local</server_name>
  <tool_name>delete_note</tool_name>
  <arguments>
    <id>note_1234567890</id>
  </arguments>
</tool>
```

**Parameters:**
- `id` (required): Note ID

**Returns:** Success message, metadata

#### 6. scratch_note
```xml
<tool>
  <server_name>local</server_name>
  <tool_name>scratch_note</tool_name>
  <arguments>
    <content>Quick note without tags</content>
  </arguments>
</tool>
```

**Parameters:**
- `content` (required): String, max 4000 chars

**Returns:** Note ID, success message, metadata

**Rationale:** Convenience tool for rapid note-taking without tag overhead.

#### 7. list_tags
```xml
<tool>
  <server_name>local</server_name>
  <tool_name>list_tags</tool_name>
  <arguments></arguments>
</tool>
```

**Parameters:** None

**Returns:** All unique tags in use with note counts, metadata

## Consequences

### Positive

1. **Clean Separation**: Notes exist outside conversation memory and token budgets
2. **Explicit Usage**: Tool-based access provides clear audit trail and learning signal
3. **Scalability**: Session-scoped in-memory storage has minimal overhead
4. **Flexibility**: Tag-based organization adapts to various use cases
5. **Discoverability**: Conditional system prompt teaches agents when feature is available
6. **Simplicity**: No mutex, no persistence—just fast in-memory operations
7. **Instructive Errors**: Detailed validation messages help the agent learn constraints
8. **State Awareness**: Metadata in tool results keeps agent informed

### Negative

1. **No Persistence**: Notes are lost on agent restart (acceptable for MVP)
2. **Limited Search**: Substring matching, not full-text indexing (sufficient for initial use)
3. **Tool Proliferation**: 7 new tools increase system prompt size (mitigated by conditional inclusion)
4. **Learning Curve**: Agent must learn when/how to use scratchpad effectively

### Neutral

1. **Tag Normalization**: No automatic normalization—agent learns through experience
2. **ID Scheme**: Simple timestamp-based IDs (good enough for session scope)
3. **Memory Refactor**: Moving conversation memory to subpackage (cleanup opportunity)

---

## Implementation

### Migration Path

This is a new feature with no existing implementation to migrate from. Integration steps:

1. Create new packages without touching existing code
2. Register tools optionally in application startup
3. Add conditional system prompt inclusion
4. Refactor conversation memory to subpackage (backward compatible)

### Timeline

- **Phase 1** (Core Infrastructure): 1-2 days
- **Phase 2** (Tool Implementation): 2-3 days  
- **Phase 3** (System Integration): 1-2 days
- **Phase 4** (Memory Refactor): 1 day
- **Phase 5** (Documentation): 1 day

**Total Estimated Time:** 6-9 days

---

## Validation

### Success Metrics

- Agent successfully uses scratchpad tools in complex multi-step tasks
- Notes reduce redundant file reads and searches
- Token budget usage decreases for long-running sessions
- Agent learns appropriate usage patterns within 5-10 task attempts
- Zero performance degradation from in-memory note storage

### Monitoring

- Track scratchpad tool usage frequency in agent logs
- Monitor token budget impact (should decrease overall usage)
- Measure task completion efficiency (fewer redundant operations)
- Collect user feedback on agent behavior improvements

---

## Related Decisions

- Memory system architecture (conversation memory)
- Tool interface design (`pkg/agent/tools/tool.go`)
- System prompt composition (`pkg/agent/prompts/builder.go`)

## References

- Feature spec: `docs/product/features/agent-scratchpad-notes.md`
- Requirements elicitation: `elicitation.md`
- Tool interface: `pkg/agent/tools/tool.go`
- Existing tool examples: `pkg/tools/coding/write_file.go`
- Prompt builder: `pkg/agent/prompts/builder.go`
