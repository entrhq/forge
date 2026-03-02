# 10. Tool Approval and Rejection Mechanism

**Status:** Proposed
**Date:** 2025-01-05
**Deciders:** Forge Core Team
**Technical Story:** Building a TUI coding agent that requires user approval for sensitive operations

---

## Context

The Forge TUI coding agent will execute powerful operations like file modifications and terminal commands. Users need the ability to review and approve/reject these operations before execution, similar to how Claude Code and Cursor handle potentially destructive actions.

### Background

Current Forge implementation executes all tools immediately upon agent request. For a coding agent with file system and command execution capabilities, this poses risks:
- Unintended file modifications
- Potentially harmful commands
- Lack of user control over agent actions

### Problem Statement

We need a mechanism that allows users to:
1. Preview tool calls before execution
2. Approve or reject individual tool invocations
3. See context (diffs, command details) before making decisions
4. Maintain conversational flow despite interruptions for approval

### Goals

- Enable user approval/rejection of sensitive tool calls
- Provide rich context for approval decisions (e.g., diffs for file changes)
- Support both automatic (trusted) and manual approval modes
- Maintain agent loop integrity during approval flow
- Keep the mechanism reusable across different executors (TUI, CLI, API)

### Non-Goals

- Building a full permission system with roles/policies (future work)
- Automatic safety analysis of commands (future work)
- Handling multi-tool transactions atomically (future work)

---

## Decision Drivers

* **User Safety:** Users must have control over potentially destructive operations
* **User Experience:** Approval flow should be seamless and informative
* **Flexibility:** Different executors may handle approvals differently
* **Simplicity:** Mechanism should be straightforward to implement and understand
* **Reusability:** Should work across TUI, CLI, and future API executors

---

## Considered Options

### Option 1: Tool-Level Approval Flag

**Description:** Each tool declares if it requires approval via an interface method.

```go
type Tool interface {
    // ... existing methods
    RequiresApproval() bool
}
```

**Pros:**
- Simple to implement
- Tools self-declare their approval needs
- Easy to understand

**Cons:**
- Binary choice (approve/reject) per tool
- No context-sensitive approval (e.g., approve writes to certain paths)
- All instances of a tool require same approval treatment

### Option 2: Approval Callback in Executor

**Description:** Executors implement an approval callback that the agent loop calls before tool execution.

```go
type ApprovalHandler interface {
    ShouldExecute(tool Tool, args json.RawMessage) (bool, error)
}
```

**Pros:**
- Executor controls approval UX
- Can provide context-specific approval UI
- Flexible per-executor implementation

**Cons:**
- Requires agent loop modification
- Coupling between agent and executor approval logic
- Harder to test in isolation

### Option 3: Event-Based Approval Flow

**Description:** Agent emits approval request events, waits for approval response events.

```go
type AgentEvent struct {
    Type EventType // EventTypeApprovalRequest, EventTypeApprovalResponse
    ToolCall *ToolCall
    Approved bool
}
```

**Pros:**
- Fits existing event-driven architecture
- Decoupled: agent doesn't know about approval mechanism
- Executors can handle approval asynchronously
- Easy to extend with additional approval metadata

**Cons:**
- More complex event flow
- Requires careful state management during approval wait
- Agent loop must pause during approval

---

## Decision

**Chosen Option:** Option 3 - Event-Based Approval Flow

### Rationale

Event-based approval aligns perfectly with Forge's existing event-driven architecture. It maintains separation of concerns: the agent loop handles tool orchestration, while executors handle user interaction for approvals.

Key benefits:
1. **Consistency:** Uses the same event channel pattern already in place
2. **Flexibility:** Different executors can implement approval UX differently (TUI diff viewer vs CLI prompts)
3. **Testability:** Approval logic can be tested independently
4. **Extensibility:** Easy to add approval metadata (diffs, previews, impact analysis)

---

## Consequences

### Positive

- Clean separation between agent logic and approval UX
- Executors control approval presentation (diff viewer, previews, etc.)
- Can enhance approval requests with rich context over time
- Fits naturally into existing event architecture
- Easy to make certain tools "auto-approve" in executor

### Negative

- Agent loop becomes more complex with approval state management
- Potential for deadlock if approval response never arrives (needs timeout)
- More events flowing through the system
- Requires careful testing of approval flows

### Neutral

- Event channel remains the primary communication mechanism
- Executors must implement approval handling if they want to support it
- Tools remain unaware of approval mechanism

---

## Implementation

### Agent Loop Changes

```go
// In agent loop, before tool execution:
1. Emit EventTypeToolApprovalRequest with tool call details
2. Wait for EventTypeToolApprovalResponse on approval channel
3. If approved: execute tool normally
4. If rejected: emit EventTypeToolRejected, continue loop
5. Timeout after 5 minutes, treat as rejection
```

### New Event Types

```go
const (
    EventTypeToolApprovalRequest EventType = "tool_approval_request"
    EventTypeToolApprovalResponse EventType = "tool_approval_response"
    EventTypeToolRejected EventType = "tool_rejected"
)
```

### Tool Metadata

Tools can optionally provide preview data:

```go
type PreviewableT interface {
    Tool
    Preview(ctx context.Context, args json.RawMessage) (string, error)
}
```

For example, `ApplyDiffTool.Preview()` returns formatted diff output.

### Migration Path

1. Add approval events to types package
2. Modify agent loop to emit approval requests
3. Add approval response channel to AgentChannels
4. Update TUI executor to handle approval requests with diff viewer
5. Default behavior: auto-approve if executor doesn't handle approval events

### Timeline

- Week 1: Core approval mechanism in agent loop
- Week 2: TUI approval handler with diff viewer
- Week 3: Tool preview interfaces for coding tools
- Week 4: Testing and refinement

---

## Validation

### Success Metrics

- User can approve/reject file changes with diff preview
- User can approve/reject commands before execution
- No approval deadlocks or timeouts in normal usage
- Approval flow adds <500ms latency to tool execution

### Monitoring

- Track approval request/response timing
- Monitor timeout occurrences
- Log approval/rejection rates per tool
- Measure user satisfaction with approval UX

---

## Related Decisions

- [ADR-0008](0008-agent-controlled-loop-termination.md) - Agent loop control flow
- [ADR-0009](0009-tui-executor-design.md) - TUI executor architecture
- [ADR-0011](0011-coding-tools-architecture.md) - Coding tools architecture
- [ADR-0012](0012-enhanced-tui-executor.md) - Enhanced TUI diff viewer

---

## References

- [Claude Code approval patterns](https://www.anthropic.com/news/claude-code)
- [Cursor AI approval flow](https://cursor.sh)
- Event-driven architecture patterns

---

## Notes

The approval mechanism should feel natural and non-intrusive. For frequently used, trusted operations, executors can implement "always approve" lists or patterns.

Future enhancements could include:
- Approval templates/policies
- Batch approval of similar operations
- Learned approval patterns
- Risk scoring for automatic approval decisions

**Last Updated:** 2025-01-05