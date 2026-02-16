# Subagent Architecture: "Agents as Tools" Pattern

**Status:** Scratch/Brainstorming  
**Created:** 2025-01-16  
**Updated:** 2025-01-16  
**Author:** Forge Team  
**Pattern:** Sequential Execution, Bidirectional Communication

---

## Problem Statement

### Why Subagents?

Current Forge architecture executes tasks sequentially in a single agent loop. This works well for focused tasks but has limitations:

1. **Delegation**: Natural human workflow (delegate subtasks, wait for results) not supported
2. **Specialization**: Single agent must handle all aspects of complex tasks
3. **Isolation**: No way to delegate risky operations to sandboxed contexts
4. **Context Separation**: Complex multi-step workflows become unwieldy in a single conversation
5. **Interactive Subtasks**: Cannot delegate work that might need clarification

### "Agents as Tools" Philosophy

**NOT Multi-Agent Orchestration**: We are not building parallel execution or complex coordination patterns (at least not in this iteration).

**Instead**: Subagents are treated like **function calls** where the "function" is another agent. Parent calls child, blocks/waits for result, child can ask questions during execution.

This is analogous to:
```python
def parent_task():
    result = call_subagent("refactor auth.go")  # Blocks here
    # Child may ask questions during execution
    # Parent answers, child continues
    # Eventually returns result
    use_result(result)
```

### Use Cases

**Focused Delegation:**
- "Review this file for security issues" → spawn security-focused child agent
- "Write comprehensive tests for this module" → spawn testing-focused child agent
- "Research this API and document findings" → spawn research child agent

**Specialized Agents:**
- Parent handles high-level strategy, delegates specialized work to children
- Code review agent, testing agent, documentation agent, research agent
- Each child has focused context and specialized prompting

**Isolation & Safety:**
- Experimental operations in sandboxed child agent
- Untrusted code execution in isolated context
- If child fails, parent can handle gracefully

**Interactive Delegation:**
- Child encounters ambiguity, asks parent for clarification
- Parent provides guidance, child continues
- Back-and-forth dialog until subtask completes

**Context Management:**
- Break large task into manageable subtasks with separate contexts
- Each subagent maintains focused conversation history
- Parent doesn't pollute its context with detailed subtask work

---

## Mental Model

### Core Concepts

**Sequential "Function Call" Model:**
```
┌─────────────────────────────────────────┐
│  Parent Agent                            │
│  - Working on main task                  │
│  - Encounters subtask needing delegation │
└──────────┬──────────────────────────────┘
           │
           │ spawn_subagent("Review auth.go for security")
           ▼
┌─────────────────────────────────────────┐
│  Child Agent (Security Reviewer)         │
│  - Receives task context                 │
│  - Performs security review              │
│  - May ask parent questions              │
│  - Returns review results                │
└──────────┬──────────────────────────────┘
           │
           │ Returns result
           ▼
┌─────────────────────────────────────────┐
│  Parent Agent                            │
│  - Receives review results               │
│  - Continues main task                   │
└─────────────────────────────────────────┘

Key: ONE child at a time, parent BLOCKS until child completes
Depth limit: 2-3 levels (child can spawn grandchild if needed)
```

**Communication Model:**
- **Synchronous**: Parent blocks waiting for child to complete
- **Bidirectional**: Child can ask questions, parent can respond
- **Dialog-based**: Question/answer exchanges during execution
- **Scoped**: Each child has isolated workspace/context

**Execution Flow:**
```
Parent spawns child
    → Parent's agent loop PAUSES
    → Child's agent loop RUNS
    → Child may send questions to parent
        → Parent receives question
        → Parent responds
        → Child receives answer, continues
    → Child calls task_completion
    → Child's result returned to parent
    → Parent's agent loop RESUMES
```

**Lifecycle States:**
```
Parent Working → Spawns Child → Parent Blocked
                       ↓
                  Child Running
                       ↓
              Child asks question? ─Yes→ Parent Answers → Child Continues
                       ↓ No
                Child Completes
                       ↓
              Parent Resumes Work
```

### Key Principles

1. **Parent Controls Lifecycle**: Parent spawns, monitors, and terminates children
2. **Isolated Context**: Each child has separate memory, workspace, and tool access
3. **Explicit Communication**: No implicit state sharing between agents
4. **Resource Bounded**: Max children per parent, max depth, max total agents
5. **Failure Isolation**: Child failure doesn't crash parent
6. **Clean Shutdown**: Graceful termination cascade (parent dies → children die)

---

## Architecture Components

### 1. Call Stack Tracker

**Purpose**: Track the agent call stack (like function call stack)

```go
type CallStack struct {
    mu sync.RWMutex
    stack []*CallFrame  // Current call stack
    maxDepth int        // Max recursion depth (default: 3)
}

type CallFrame struct {
    AgentID    string                // Unique agent ID at this level
    ParentID   string                // Parent agent ID (empty for root)
    AgentName  string                // Subagent name (e.g., "security-reviewer")
    Task       string                // Task description
    Started    time.Time             // When this agent was spawned
    Agent      *agent.DefaultAgent   // The actual agent instance
    Channels   *types.AgentChannels  // Agent's communication channels
    Cancel     context.CancelFunc    // Cancel function for cleanup
}
```

**Responsibilities:**
- Push new call frame when child spawned
- Pop call frame when child completes
- Enforce depth limits (prevent infinite recursion)
- Route questions from child to parent
- Handle parent responses back to child

### 2. Question/Answer Router

**Purpose**: Route questions from child to parent and answers back

```go
type QARouter struct {
    mu sync.RWMutex
    channels map[string]*QAChannel  // agentID → communication channel
}

type QAChannel struct {
    QuestionChan chan *Question  // Child → Parent questions
    AnswerChan   chan *Answer    // Parent → Child answers
}

type Question struct {
    ID          string
    From        string   // Child agent ID
    Question    string   // The actual question text
    Suggestions []string // Optional suggested answers
    Timestamp   time.Time
}

type Answer struct {
    QuestionID string
    Answer     string
    Timestamp  time.Time
}
```

**Responsibilities:**
- Create bidirectional channels when child spawned
- Route questions from child to parent
- Route answers from parent back to child
- Handle timeouts if parent doesn't respond
- Clean up channels when child completes

### 3. Subagent Registry & Manager

**Purpose**: Manage available subagent implementations and lifecycle

```go
type SubagentRegistry struct {
    mu          sync.RWMutex
    factories   map[string]*RegisteredFactory  // name → factory function + metadata
}

type RegisteredFactory struct {
    Name        string            // e.g., "security-reviewer", "test-writer"
    Description string            // What this agent does
    Factory     SubagentFactory   // Factory function that creates agent
}

// SubagentFactory is a function that creates a configured DefaultAgent
// It follows the EXACT same pattern as creating the main agent:
// - Call agent.NewDefaultAgent() with options
// - Register tools with ag.RegisterTool()
// - Return the configured agent
type SubagentFactory func(ctx SubagentContext) (*agent.DefaultAgent, error)

// SubagentContext provides the necessary context for creating a subagent
type SubagentContext struct {
    Provider       llm.Provider              // LLM provider (same or different from parent)
    WorkspaceDir   string                    // Isolated workspace directory
    Guard          security.WorkspaceGuard   // Workspace guard for tool isolation
    ParentInput    chan *types.Input         // Child's Input channel (for receiving answers)
    NotesManager   *notes.Manager            // Shared or isolated notes
}
```

**Subagent Manager:**

```go
type SubagentManager struct {
    registry  *SubagentRegistry
    callStack *CallStack
    config    *ManagerConfig
    
    // Shared resources that can be passed to subagents
    provider       llm.Provider
    workspaceGuard security.WorkspaceGuard
    notesManager   *notes.Manager
}

type ManagerConfig struct {
    MaxDepth         int           // Max call stack depth (default: 3)
    DefaultTimeout   time.Duration // Default child timeout (default: 30min)
    IsolateWorkspace bool          // Create separate workspace dirs per child
}
```

**Responsibilities:**
- Register subagent factory functions at startup
- Lookup factory by name when parent spawns
- Call factory to get configured DefaultAgent instance
- Create AgentChannels for child agent
- Push call frame to stack
- Start child agent in goroutine
- Block parent agent while listening to child's Event channel
- Handle EventTypeSubagentQuestion by routing to parent's LLM
- Return child's result to parent when child completes
- Clean up when child completes (pop call stack)
- Enforce depth limits

### 4. Tool Interface Extensions

**New Loop-Breaking Tool: spawn_subagent**

```go
type SpawnSubagentTool struct {
    manager *SubagentManager
}

// Parameters
type SpawnSubagentArgs struct {
    Agent    string        // Subagent name from registry (e.g., "security-reviewer")
    Task     string        // Task description for child
    Timeout  time.Duration // Max execution time (default: 30min)
    Context  string        // Optional additional context for child
}

// Returns (when child completes)
type SpawnSubagentResult struct {
    Result    string         // Child's task_completion result
    Duration  time.Duration  // How long child took
    Questions int            // Number of questions child asked parent
}
```

**Notes on spawn_subagent:**
- This is a **BLOCKING** tool - parent's agent loop pauses until child completes
- Parent looks up subagent by name in registry
- Child is created via factory function (NewDefaultAgent + RegisterTool pattern)
- Child runs its own agent loop in a goroutine
- Parent listens to child's Event channel for questions/completion
- When child calls task_completion, result is returned to parent
- Parent receives child's result and continues its own loop

**New Tool for Child Agents: ask_parent**

Child agents get a special `ask_parent` tool (instead of regular `ask_question`):

```go
type AskParentTool struct {
    parentInputChan chan *types.Input  // Send answers back to child
}

// Parameters
type AskParentArgs struct {
    Question    string   // Question for parent agent
    Suggestions []string // Optional suggested answers
}
```

**Notes on ask_parent:**
- This is a **BLOCKING** tool - child's agent loop pauses until parent responds
- Child emits `EventTypeSubagentQuestion` on its Event channel
- Parent's LLM receives question and decides:
  - Answer directly from context/knowledge
  - OR call `ask_question` to escalate to user
- Parent sends answer to child's Input channel
- Child receives answer and continues execution

**Example Flow:**

```
Child Agent: *calls ask_parent("What's the database connection string?")*
  → Emits EventTypeSubagentQuestion on Event channel
  → Blocks waiting for answer

Parent Agent: *receives question in next loop iteration*
  → Parent LLM: "I have this in context: postgres://..."
  → Sends answer to child's Input channel
  → Child receives answer and continues

Child Agent: *calls ask_parent("Should I delete the production table?")*
  → Emits EventTypeSubagentQuestion on Event channel
  → Blocks waiting for answer

Parent Agent: *receives question*
  → Parent LLM: "This is risky, I should ask the user"
  → *calls ask_question to escalate to user*
  → User responds: "No, don't delete it"
  → Sends user's answer to child's Input channel
  → Child receives answer and continues
```

```go
// In child agent context:
// Child calls: ask_question("Should I use JWT or sessions?")
// → Question routed to parent
// → Parent receives question in their event stream
// → Parent can respond via normal conversation
// → Response routed back to child
// → Child receives answer and continues
```

**No other tools needed** - the spawn_subagent + ask_question routing is sufficient for the "agents as tools" pattern.

---

## Communication Patterns

### Pattern 1: Simple Delegation (No Questions)

Parent spawns child, waits for result:

```
Parent: "I need to review auth.go for security issues"
Parent: spawn_subagent(task="Security review of auth.go")
        → Parent's loop BLOCKS
        → Child agent starts
        → Child reads file, analyzes
        → Child completes review
        → Child calls task_completion("Found 3 issues: ...")
Parent: receives result: "Found 3 issues: ..."
Parent: "Based on the review, I'll fix these issues..."
        → Parent continues main task
```

### Pattern 2: Interactive Delegation (Child Asks Questions)

Parent spawns child, child needs clarification during work:

```
Parent: "Refactor the authentication system"
Parent: spawn_subagent(task="Refactor auth.go to use modern patterns")
        → Parent's loop BLOCKS
        → Child agent starts
        → Child analyzes code
        
Child:  "I found both JWT and session-based auth. Which should I prioritize?"
        (Child calls ask_question tool, which routes to parent)
        → Child's loop BLOCKS waiting for answer
        
Parent: receives question as event
Parent: responds: "Prioritize JWT, keep sessions for backward compatibility"
        → Answer routed back to child
        
Child:  receives answer, continues work
        → Child refactors code
        → Child calls task_completion("Refactored to prioritize JWT...")
        
Parent: receives result: "Refactored to prioritize JWT..."
Parent: "Great, now I'll update the tests..."
        → Parent continues main task
```

### Pattern 3: Multi-Level Delegation (Child Spawns Grandchild)

Parent delegates to child, child delegates further:

```
Parent: "Improve the authentication system"
Parent: spawn_subagent(task="Modernize auth system")
        → Child 1 starts (Auth Modernizer)
        
Child1: "I'll need comprehensive tests for the new auth flow"
Child1: spawn_subagent(task="Write tests for JWT auth flow")
        → Child 1's loop BLOCKS
        → Child 2 starts (Test Writer)
        
Child2: analyzes, writes tests
Child2: task_completion("Wrote 15 comprehensive tests")
        
Child1: receives test results
Child1: continues with modernization
Child1: task_completion("Auth system modernized with full test coverage")
        
Parent: receives result
Parent: "Excellent, now deploying changes..."
```

**Note**: NO parallel execution. Only ONE active child per parent at any time.

---

## Workspace Isolation

### Problem

Multiple agents working simultaneously need isolated workspaces to prevent conflicts.

### Solution: Workspace Hierarchy

```
workspace/                    # Root workspace (parent agent)
├── .forge/
│   ├── subagents/
│   │   ├── agent_abc123/    # Child agent workspace
│   │   │   ├── files...     # Child's working files
│   │   │   └── .forge/      # Child's metadata
│   │   └── agent_def456/    # Another child workspace
│   │       └── files...
├── src/                      # Shared source code (read-only?)
└── tests/                    # Shared tests (read-only?)
```

### Access Control

**Parent Agent:**
- Full read/write to root workspace
- Read access to child workspaces (monitoring)
- Can create/delete child workspace directories

**Child Agent:**
- Full read/write to its own workspace (`.forge/subagents/agent_<id>/`)
- Read-only access to parent workspace (shared context)
- No access to sibling workspaces

**Implementation:**
```go
type WorkspaceGuard struct {
    agentID      string
    parentWS     string        // Parent's workspace root
    childWS      string        // This child's isolated workspace
    allowedPaths []string      // Additional allowed paths
}

func (w *WorkspaceGuard) ValidatePath(path string) error {
    // Check if path is within allowed boundaries
    // Prevent path traversal attacks
    // Enforce read-only vs read-write rules
}
```

---

## Resource Management

### Limits & Quotas

```go
type ResourceLimits struct {
    // Agent limits
    MaxChildrenPerParent int           // Default: 5
    MaxHierarchyDepth    int           // Default: 3
    MaxTotalAgents       int           // Default: 20
    
    // Execution limits
    ChildDefaultTimeout  time.Duration // Default: 5min
    MaxChildTimeout      time.Duration // Default: 30min
    
    // Memory limits
    MaxMemoryPerAgent    int           // Messages in memory
    MaxContextTokens     int           // Context window size
    
    // Tool limits
    ChildToolWhitelist   []string      // Tools children can use
    ChildToolBlacklist   []string      // Tools children cannot use
}
```

### Resource Tracking

```go
type ResourceTracker struct {
    mu sync.RWMutex
    
    // Current usage
    totalAgents    int
    agentsByDepth  map[int]int           // depth → count
    agentsByParent map[string]int        // parentID → child count
    
    // Resource consumption
    cpuUsage       map[string]float64    // agentID → CPU %
    memoryUsage    map[string]int64      // agentID → bytes
    toolCalls      map[string]int        // agentID → call count
}

func (r *ResourceTracker) CanSpawnChild(parentID string, depth int) error {
    // Check against all limits
    // Return specific error if limit would be exceeded
}
```

### Cleanup & Garbage Collection

**Trigger conditions:**
- Child completes successfully → cleanup after result delivery
- Child fails → cleanup after error propagation
- Child timeout → forceful termination + cleanup
- Parent terminates → cascade kill all children
- System shutdown → graceful shutdown of all agents

**Cleanup steps:**
1. Cancel child's context
2. Wait for agent loop to exit (with timeout)
3. Close message channels
4. Remove from registry
5. Clean up workspace directory (optional)
6. Emit cleanup completion event

---

## Error Handling

### Child Agent Failures

**Failure Types:**
1. **Tool execution error**: Child tool call fails
2. **LLM API error**: Provider unavailable or rate-limited
3. **Timeout**: Child exceeds time limit
4. **Resource exhaustion**: Out of memory, too many tools
5. **Crash**: Panic or unexpected termination

**Parent Response:**
```go
type ChildFailureEvent struct {
    ChildID    string
    FailureType string
    Error       error
    Context     map[string]interface{}
}

// Parent receives failure event
// Options:
// 1. Retry: Spawn new child with same task
// 2. Fallback: Handle task differently
// 3. Escalate: Report to user via ask_question
// 4. Abort: Fail parent task
```

### Parent Agent Failures

**If parent fails/terminates:**
1. All children receive termination signal
2. Children attempt graceful shutdown
3. Children's contexts are cancelled
4. Partial results are preserved (if possible)
5. System cleans up all related resources

**Implementation:**
```go
func (m *SubagentManager) handleParentShutdown(parentID string) {
    children := m.registry.GetChildren(parentID)
    
    var wg sync.WaitGroup
    for _, childID := range children {
        wg.Add(1)
        go func(id string) {
            defer wg.Done()
            m.terminateAgent(id, "parent_shutdown")
        }(childID)
    }
    
    // Wait with timeout
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        // All children cleaned up
    case <-time.After(10 * time.Second):
        // Forceful cleanup
    }
}
```

---

## Memory & Context Management

### Challenge

Each agent maintains conversation history. Parent-child communication creates context dependencies.

### Strategy 1: Isolated Memory (Simple)

- Each agent has completely separate memory
- Child only knows: initial task + parent messages
- No access to parent's conversation history
- **Pro**: Simple, clean isolation
- **Con**: Child lacks context, may ask redundant questions

### Strategy 2: Context Inheritance (Complex)

- Child inherits snapshot of parent's memory at spawn time
- Future parent updates not visible to child
- **Pro**: Child has historical context
- **Con**: Memory duplication, staleness

### Strategy 3: Shared Context with Scoping (Hybrid)

- Shared read-only "knowledge base" layer
- Each agent has private working memory
- Child can reference parent's context but doesn't duplicate it
- **Pro**: Efficient, context-aware
- **Con**: More complex memory management

**Recommendation**: Start with Strategy 1, evolve to Strategy 3

---

## Implementation Phases

### Phase 1: Foundation (MVP)

**Goal**: Basic sequential "agent as tool" pattern

**Deliverables:**
- CallStack (track active agent hierarchy)
- QARouter (route questions/answers between parent and child)
- SubagentManager (spawn child, block parent, return result)
- spawn_subagent tool (blocking, waits for child completion)
- ask_question routing (child → parent instead of user)
- Basic workspace isolation
- Depth limit enforcement (max 2-3 levels)
- Timeout handling

**Use Case Enabled**: Spawn child for subtask, child can ask questions, parent receives result

**Success Criteria:**
- Parent spawns child with task description
- Parent's loop blocks while child runs
- Child performs work, may ask parent questions
- Parent responds to questions, child continues
- Child calls task_completion, result returned to parent
- Parent resumes with child's result

### Phase 2: Specialization & Context

**Goal**: Specialized agent types and better context handling

**Deliverables:**
- Agent specialization system ("security", "testing", "research", "coding")
- Specialized system prompts for each type
- Context inheritance strategy (what does child see from parent?)
- Workspace isolation improvements
- Error handling and graceful failures

**Use Case Enabled**: Delegate to specialized agents with appropriate prompting and context

### Phase 3: Advanced Features

**Goal**: Observability, monitoring, and resilience

**Deliverables:**
- TUI visualization of call stack (show parent → child relationship)
- Progress tracking (child status visible to parent)
- Retry mechanisms (respawn child if it fails)
- Resource metrics (tokens used, time elapsed)
- Cost attribution (track LLM costs per agent)

**Use Case Enabled**: Observable, debuggable subagent execution with clear feedback

### Phase 4: Optimization

**Goal**: Performance and developer experience

**Deliverables:**
- Context sharing optimizations (avoid duplicating parent's full history)
- Faster spawn times
- Agent templates/presets for common patterns
- Better error messages
- Documentation and examples

---

## Open Questions

1. **Tool Access Control**: Should children have restricted tool access by default? Or inherit parent's tools?

2. **LLM Provider**: Do children use same LLM provider/model as parent? Or can they use specialized models (e.g., fast model for simple tasks)?

3. **Approval System**: If child needs approval for a tool, does it propagate to parent? Or does child have its own approval context?

4. **Conversation UI**: How does TUI display parent-child interaction?
   - Show call stack in sidebar?
   - Nest child messages under parent?
   - Switch to child's view when active?

5. **Context Inheritance**: What does child see from parent's conversation?
   - Full history? (expensive)
   - Summary only? (may lack details)
   - Explicit context passed in spawn args? (controlled but manual)

6. **Cost Tracking**: How do we attribute LLM costs?
   - Track separately per agent?
   - Roll up child costs to parent?
   - Show breakdown in final report?

7. **Recursion Limits**: Beyond depth limits, what other protections?
   - Max total spawns per session?
   - Time budget for entire call stack?
   - Token budget across all agents?

8. **Error Propagation**: If child fails, how does parent handle it?
   - Receive error as result? (parent decides what to do)
   - Automatic retry? (transparent to parent)
   - Escalate to user? (break parent's loop)

9. **Result Format**: Should spawn_subagent return structured data or free-form text?
   - Free-form (child's task_completion message)
   - Structured (success/failure, result, metadata)

10. **Shared State**: Can children access parent's scratchpad notes? Browser sessions? Active commands?

---

## Success Metrics

**Functionality:**
- [ ] Spawn child agent and receive result
- [ ] Multiple children in parallel
- [ ] Parent-child bidirectional messaging
- [ ] Graceful shutdown and cleanup
- [ ] Resource limits enforced

**Performance:**
- Spawn latency: < 1s
- Message delivery: < 100ms
- Cleanup time: < 2s
- Memory overhead: < 50MB per child

**Reliability:**
- Child failure doesn't crash parent: 100%
- Resource cleanup on termination: 100%
- No orphaned agents: 100%
- No deadlocks: 100%

---

## Next Steps

1. **Create PRD**: Formalize requirements in `docs/product/features/subagent-system.md`
2. **Create ADR**: Technical design decisions in `docs/adr/00XX-subagent-architecture.md`
3. **Prototype**: Build Phase 1 MVP in feature branch
4. **Test Cases**: Design comprehensive test scenarios
5. **Documentation**: Update AGENTS.md with subagent workflow

---

## References

- Current agent loop: ADR-0008
- Tool system: ADR-0002
- Memory management: ADR-0007
- Error recovery: ADR-0006
- Headless mode: PRD (headless-ci-mode.md) - potential integration point
