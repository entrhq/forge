# Agent Scratchpad Notes - Requirements Elicitation

**Date:** 2024-12-19  
**Feature:** Agent Scratchpad & Notes System  
**Status:** Design Phase

## Questions & Answers

### Q1: Storage Location
**Question:** Should the notes system be a new package (pkg/agent/notes) or extend the existing memory package (pkg/agent/memory)?

**Answer:** 
- Move current memory contents to `memory/conversation` subpackage
- Create new `memory/notes` subpackage for the scratchpad
- Notes are a form of memory but don't conform to the same interface as conversational memory

**Decision:** Refactor memory package structure:
```
pkg/agent/memory/
├── conversation/  (current conversation.go, memory.go)
└── notes/         (new scratchpad implementation)
```

---

### Q2: Agent Integration
**Question:** How should notes be provided to the agent in the loop?

**Answer:**
- **First iteration:** Notes retrieved/added via tools ONLY
- Notes are NOT automatically injected into context/system messages
- System prompt will explain how/what/why of the scratchpad
- Agent must explicitly call tools to interact with notes

**Decision:** Tool-based access only (no automatic context injection)

---

### Q3: Tool Registration
**Question:** Where should the 7 new tools be registered?

**Answer:**
- Follow existing pattern: coding tools are in `tools/coding`
- Create new `tools/scratchpad` package for notes tools
- Tools should follow same registration pattern

**Decision:** New package at `pkg/tools/scratchpad/`

---

### Q4: Context Budget
**Question:** How should notes interact with context compression?

**Answer:**
- Notes stored separately from chat history
- NOT included in normal token loop/counting
- Will naturally survive compression/summarization
- Should NOT be pruned or summarized

**Decision:** Notes are outside the token budget - managed separately via tools

---

### Q5: Session Lifecycle
**Question:** How is a "session" defined?

**Answer:**
- Currently: lifetime of the Agent instance
- Notes manager/storage should be defined where tools are passed to agent
- Same manager instance passed into all scratchpad tools
- All tools share the same storage instance

**Decision:** Session = Agent instance lifetime. Notes manager initialized with agent, passed to tools.

---

### Q6: Search Implementation
**Question:** What search capabilities should be implemented?

**Answer:**
- Implement ALL of:
  - Query substring matching
  - Full-text search
  - Tag filtering with AND logic
- Should support both query and tag filtering together

**Decision:** Full implementation with query, tags, and logic operators

---

## Implementation Architecture (Draft)

### Package Structure
```
pkg/
├── agent/
│   └── memory/
│       ├── conversation/
│       │   ├── memory.go          (moved from memory/)
│       │   └── conversation.go    (moved from memory/)
│       └── notes/
│           ├── manager.go         (notes CRUD + search)
│           ├── note.go            (Note struct)
│           └── manager_test.go
└── tools/
    └── scratchpad/
        ├── add_note.go
        ├── search_notes.go
        ├── list_notes.go
        ├── update_note.go
        ├── delete_note.go
        ├── scratch_note.go
        └── list_tags.go
```

### Tool Dependency Injection Pattern
- Notes manager created when Agent is initialized
- Manager passed to scratchpad tools via constructor/factory
- All tools share the same manager instance
- Manager is session-scoped (lives with Agent)

---

## Open Questions for Next Iteration

1. **Tool Factory Pattern**: How exactly are tools currently instantiated and passed dependencies? Need to examine `pkg/tools/coding` pattern.

2. **Manager Interface**: Should we define a `NotesManager` interface or use concrete type?

3. **Concurrency**: Do we need thread-safe access to notes (similar to ConversationMemory)?

4. **Tool Schema**: Need to define exact XML schema for all 7 tools following existing patterns.

5. **Error Handling**: How should validation errors (character limits, tag counts) be returned to the agent?

6. **ID Generation**: Should we use `note_[timestamp]` or a more robust UUID/ULID?

7. **Tag Normalization**: Should tags be case-insensitive? Automatically lowercased?

8. **Search Ranking**: For full-text search, what ranking algorithm? Simple relevance score?

---

## Next Steps

1. Explore `pkg/tools/coding` to understand tool implementation pattern
2. Explore agent initialization to understand dependency injection
3. Explore existing tool schemas and XML patterns
4. Propose detailed implementation approach
5. Create ADR with technical decisions
