# 26. Headless CI/CD Mode Architecture

**Status:** Proposed
**Date:** 2025-01-21
**Deciders:** Forge Core Team
**Technical Story:** Enable Forge to run autonomously in CI/CD pipelines without human interaction

---

## Context

Forge currently operates as an interactive TUI application where users chat with an AI agent that executes tools with manual approval. While powerful for development workflows, this design requires continuous human oversight and cannot be integrated into automated CI/CD pipelines, scheduled maintenance jobs, or webhook-triggered automation.

### Background

Modern software engineering increasingly relies on automation to maintain code quality, enforce standards, and reduce manual toil. Teams need AI-powered tools that can:
- Automatically fix linting errors and apply code formatting
- Update dependencies and apply security patches
- Generate boilerplate code and maintain consistency
- Run quality checks and create cleanup PRs

Competitive products like GitHub Copilot Workspace and Cursor focus exclusively on interactive development, leaving a gap for autonomous execution in CI/CD environments.

### Problem Statement

We need to enable Forge to run completely autonomously in non-interactive environments (CI/CD pipelines, cron jobs, webhooks) while maintaining safety, reliability, and predictability. This requires:

1. **Removing interactive dependencies**: No TUI, no approval flows, no questions to users
2. **Safety without supervision**: Constraints and quality gates ensure safe autonomous operation
3. **Deterministic execution**: Same input produces consistent output
4. **CI/CD integration**: Proper exit codes, logging, and artifact generation
5. **Git integration**: Automatic commits with proper attribution

### Goals

- Enable fully autonomous task execution without human intervention
- Provide comprehensive safety constraints to prevent runaway execution
- Integrate seamlessly with GitHub Actions and other CI/CD platforms
- Generate execution artifacts for auditing and debugging
- Support both simple CLI invocations and complex YAML configurations
- Maintain code reuse with existing agent/tool architecture

### Non-Goals

- Building a web UI or dashboard (future iteration)
- Real-time progress webhooks (P2 feature)
- Multi-stage execution with approval APIs (P2 feature)
- Advanced risk scoring or ML-based safety (future)

---

## Decision Drivers

* **Safety First**: Autonomous execution must not cause uncontrolled modifications or cost overruns
* **Developer Experience**: Simple CLI interface for common cases, YAML for complex workflows
* **Code Reuse**: Leverage existing agent loop, tools, and memory systems
* **CI/CD Native**: Must work seamlessly in GitHub Actions, GitLab CI, Jenkins, etc.
* **Observability**: Complete logging and artifact generation for debugging
* **Extensibility**: Architecture supports future features (webhooks, APIs, multi-stage execution)

---

## Considered Options

### Option 1: Separate Headless Agent Implementation

**Description:** Build a completely separate agent implementation for headless mode with its own execution logic, separate from the interactive TUI agent.

**Pros:**
- Clean separation of concerns
- No risk of breaking interactive mode
- Optimized specifically for headless execution
- Simpler to reason about each mode independently

**Cons:**
- Massive code duplication
- Tool implementations must be duplicated or made mode-aware
- Memory/context management logic duplicated
- Bug fixes must be applied in two places
- Increased maintenance burden
- Violates DRY principle

### Option 2: Mode-Aware Agent with Shared Core

**Description:** Extend the existing DefaultAgent with mode detection and configuration. Interactive mode uses TUI executor with approval, headless mode uses a new HeadlessExecutor with constraint enforcement.

**Pros:**
- Maximum code reuse (agent loop, tools, memory, prompts)
- Single source of truth for core logic
- Tools work in both modes with minor adjustments
- Bug fixes benefit both modes
- Reduced maintenance burden
- Existing ADRs and architecture remain valid

**Cons:**
- Agent code becomes slightly more complex with mode handling
- Must carefully handle mode-specific behaviors
- Risk of coupling if not designed carefully

### Option 3: Headless as a Tool Restriction Layer

**Description:** Run the existing agent with a special "headless tool registry" that only exposes non-interactive tools and enforces constraints at tool execution time.

**Pros:**
- Minimal changes to agent core
- Constraint enforcement at tool level is intuitive
- Clean separation via dependency injection

**Cons:**
- Constraints scattered across tool implementations
- Difficult to enforce global limits (token usage, execution time)
- Quality gates and rollback harder to implement
- Exit codes and artifact generation awkward to add
- Doesn't address fundamental need for different execution model

---

## Decision

**Chosen Option:** Option 2 - Mode-Aware Agent with Shared Core

### Rationale

1. **Code Reuse**: The agent loop, tool system, memory management, and prompt handling are identical between modes. Duplicating this would be wasteful.

2. **Tool Portability**: Tools like apply_diff, execute_command, read_file work identically in both modes—only the approval mechanism differs.

3. **Maintainability**: A single agent implementation means bug fixes, improvements, and new features automatically benefit both modes.

4. **Architecture Alignment**: This follows the existing pattern where execution environment (TUI vs CLI) is separate from agent logic.

5. **Extensibility**: Mode-specific behavior is encapsulated in the Executor, making it easy to add new modes (API, supervised, etc.) in the future.

The key insight is that headless mode is primarily about:
- **Execution environment** (no TUI, CLI-only)
- **Approval strategy** (constraints instead of human approval)
- **Output handling** (artifacts instead of UI updates)

All of these are Executor responsibilities, not Agent responsibilities.

---

## Consequences

### Positive

- Agent loop, tools, and memory systems work unchanged in headless mode
- Existing ADRs for tool architecture, memory, context management remain valid
- New tools automatically work in both modes
- Reduced testing burden (core logic tested once)
- Clear separation: Agent handles "what to do", Executor handles "how to present/approve it"
- Future modes (API, supervised) can follow same pattern

### Negative

- Agent must support mode configuration (minor complexity increase)
- Tools may need optional mode-specific behavior (e.g., different error messages)
- Must carefully prevent interactive tools (ask_question, converse) in headless mode
- Testing matrix includes mode combinations

### Neutral

- HeadlessExecutor is a new executor implementation alongside TUI and CLI
- Configuration system extended to support headless-specific settings
- Tool approval manager gains a constraint-based approval strategy

---

## Implementation

### High-Level Architecture

```
┌────────────────────────────────────────────────────────────┐
│                      Forge CLI                              │
│    (Detects mode based on flags: --headless)               │
└──────────────────────┬─────────────────────────────────────┘
                       │
         ┌─────────────┴──────────────┐
         │                            │
    Interactive                   Headless
         │                            │
         ▼                            ▼
┌────────────────────┐      ┌─────────────────────────┐
│   TUI Executor     │      │   Headless Executor     │
│  - Approval UI     │      │  - Auto-Approval        │
│  - Event Display   │      │  - Constraints          │
│  - User Input      │      │  - Quality Gates        │
└─────────┬──────────┘      │  - Git Operations       │
          │                 │  - Artifacts            │
          │                 └──────────┬──────────────┘
          │                            │
          │                            │
          ▼                            ▼
     ┌──────────────────┐    ┌──────────────────────┐
     │  Create Agent    │    │  Create Agent with   │
     │  (All tools)     │    │  Disabled Tools      │
     └────────┬─────────┘    └──────────┬───────────┘
              │                         │
              │                         │
              └────────┬────────────────┘
                       │
                       ▼
            ┌──────────────────────────────┐
            │      DefaultAgent            │
            │     (Mode-Agnostic)          │
            │  - Agent Loop                │
            │  - Tool Calling              │
            │  - Memory                    │
            │  - Context Mgmt              │
            └──────────┬───────────────────┘
                       │
       ┌───────────────┼────────────────┐
       │               │                │
       ▼               ▼                ▼
  ┌────────┐     ┌─────────┐     ┌──────────┐
  │ Tools  │     │ Memory  │     │ Provider │
  └────────┘     └─────────┘     └──────────┘

Tool Configuration:
┌─────────────────────────────────────────────────────────┐
│ Interactive Mode:                                       │
│  Built-in: task_completion, ask_question, converse     │
│  External: read_file, write_file, apply_diff, etc.     │
├─────────────────────────────────────────────────────────┤
│ Headless Mode:                                          │
│  Built-in: task_completion                              │
│            (ask_question & converse disabled)           │
│  External: read_file, write_file, apply_diff, etc.     │
└─────────────────────────────────────────────────────────┘
```

### Component Changes

#### 1. DefaultAgent (Minor Addition - Tool Disabling Config)

```go
type DefaultAgent struct {
    // ... existing fields ...
    disabledTools map[string]bool // Tools to exclude from registration
}

// WithDisabledTools returns an option to disable specific built-in tools
func WithDisabledTools(toolNames ...string) AgentOption {
    return func(a *DefaultAgent) {
        if a.disabledTools == nil {
            a.disabledTools = make(map[string]bool)
        }
        for _, name := range toolNames {
            a.disabledTools[name] = true
        }
    }
}

// RegisterDefaultTools respects the disabled tools configuration
func (a *DefaultAgent) RegisterDefaultTools() {
    // Only register tools not in the disabled list
    if !a.disabledTools["task_completion"] {
        a.tools["task_completion"] = tools.NewTaskCompletionTool()
    }
    if !a.disabledTools["ask_question"] {
        a.tools["ask_question"] = tools.NewAskQuestionTool()
    }
    if !a.disabledTools["converse"] {
        a.tools["converse"] = tools.NewConverseTool()
    }
}

// Agent behavior is identical in all modes
// Mode-specific tool availability is configured via WithDisabledTools
// No approval logic in agent - that's executor responsibility
```

#### 2. HeadlessExecutor (New)

```go
type HeadlessExecutor struct {
    agent           agent.Agent
    config          *HeadlessConfig
    constraintMgr   *ConstraintManager
    qualityGates    []QualityGate
    artifactWriter  *ArtifactWriter
    gitManager      *GitManager
    eventHandler    EventHandler
}

func NewHeadlessExecutor(config *HeadlessConfig, provider provider.Provider) (*HeadlessExecutor, error) {
    // 1. Create agent with interactive tools disabled
    //    This prevents them from appearing in system prompt entirely
    agent := agent.NewDefaultAgent(
        provider,
        agent.WithDisabledTools("ask_question", "converse"),
    )
    
    // 2. Register external tools that are safe for headless use
    agent.RegisterTool(tools.NewReadFileTool())
    agent.RegisterTool(tools.NewWriteFileTool())
    agent.RegisterTool(tools.NewApplyDiffTool())
    agent.RegisterTool(tools.NewSearchFilesTool())
    agent.RegisterTool(tools.NewListFilesTool())
    agent.RegisterTool(tools.NewExecuteCommandTool())
    
    // 3. Create executor with headless-specific components
    return &HeadlessExecutor{
        agent:          agent,
        config:         config,
        constraintMgr:  NewConstraintManager(config.Constraints),
        qualityGates:   createQualityGates(config.QualityGates),
        artifactWriter: NewArtifactWriter(config.ArtifactsDir),
        gitManager:     NewGitManager(config.WorkspaceDir),
        eventHandler:   NewHeadlessEventHandler(),
    }
}

func (e *HeadlessExecutor) Run(ctx context.Context) error {
    // 1. Validate configuration and constraints
    // 2. Check workspace state (git status)
    // 3. Subscribe to agent events for auto-approval and constraint checking
    // 4. Send task to agent
    // 5. Auto-approve tool calls (within constraints)
    // 6. Wait for completion or timeout
    // 7. Run quality gates
    // 8. Commit changes (if configured)
    // 9. Generate artifacts
    // 10. Return appropriate exit code
}

// Auto-approval happens in event handler
// No need to check for interactive tools - they're not registered
func (e *HeadlessExecutor) handleToolCallRequest(event *types.ToolCallRequestEvent) {
    // Validate against constraints
    if err := e.constraintMgr.ValidateToolCall(event.ToolName, event.Arguments); err != nil {
        e.agent.RejectToolCall(err)
        return
    }
    
    // Auto-approve (no user interaction)
    e.agent.ApproveToolCall()
}
```

#### 3. Interactive vs Headless Tool Sets

**Interactive Mode (TUI Executor):**
```go
// Create agent with all built-in tools enabled (default)
agent := agent.NewDefaultAgent(provider)

// Built-in tools registered:
// - task_completion (signals completion)
// - ask_question (asks user for clarification)
// - converse (casual conversation with user)

// External tools also registered...
```

**Headless Mode (Headless Executor):**
```go
// Create agent with interactive tools disabled
agent := agent.NewDefaultAgent(
    provider,
    agent.WithDisabledTools("ask_question", "converse"),
)

// Built-in tools registered:
// - task_completion (signals completion)
// ❌ ask_question (disabled - not in system prompt)
// ❌ converse (disabled - not in system prompt)

// External tools also registered...
```

**Benefits of This Approach:**
1. **Clean System Prompt**: Interactive tools don't appear in tool definitions at all
2. **Token Savings**: No wasted tokens documenting unavailable tools
3. **Clear Failure**: If LLM somehow references disabled tool, it fails cleanly
4. **No Runtime Blocking**: Executor doesn't need special logic to block tools
5. **Mode-Agnostic Agent**: Agent still doesn't know about "headless" - just has different tool config

#### 4. ConstraintManager (New)

```go
type ConstraintManager struct {
    maxFiles        int
    maxLinesChanged int
    allowedPatterns []string
    deniedPatterns  []string
    allowedTools    []string
    maxTokens       int
    timeout         time.Duration
    
    // Runtime tracking
    filesModified   map[string]int
    linesChanged    int
    tokensUsed      int
}

func (cm *ConstraintManager) ValidateToolCall(toolName string, args interface{}) error
func (cm *ConstraintManager) RecordModification(file string, linesAdded, linesRemoved int) error
func (cm *ConstraintManager) CheckTimeout(elapsed time.Duration) error
```

#### 5. QualityGate Interface (New)

```go
type QualityGate interface {
    Name() string
    Required() bool // If true, failure aborts execution
    Execute(ctx context.Context, workspaceDir string) error
}

type CommandQualityGate struct {
    name     string
    command  string
    required bool
}
```

#### 6. ArtifactWriter (New)

```go
type ArtifactWriter struct {
    outputDir string
}

func (aw *ArtifactWriter) WriteExecutionSummary(summary ExecutionSummary) error
func (aw *ArtifactWriter) WriteMetrics(metrics ExecutionMetrics) error  
func (aw *ArtifactWriter) WriteChangedFiles(files []string) error
```

### Configuration Schema

```yaml
# .forge/headless-config.yml
task: "Fix linting errors and update documentation"

mode: write # read-only | write

constraints:
  max_files: 10
  max_lines_changed: 500
  timeout: 300 # seconds
  max_tokens: 50000
  
  allowed_patterns:
    - "src/**/*.go"
    - "pkg/**/*.go"
    - "docs/**/*.md"
  
  denied_patterns:
    - "vendor/**"
    - ".git/**"
  
  allowed_tools:
    - read_file
    - write_file
    - apply_diff
    - search_files
    - list_files
    - execute_command

quality_gates:
  - name: "Run tests"
    command: "go test ./..."
    required: true
  
  - name: "Lint"
    command: "golangci-lint run"
    required: false

git:
  auto_commit: true
  commit_message: "chore: automated fixes via Forge"
  branch: "" # empty = use current branch
  author_name: "Forge AI"
  author_email: "forge@example.com"

artifacts:
  enabled: true
  output_dir: ".forge/artifacts"
  json: true      # execution.json with full details
  markdown: true  # summary.md for human reading
  metrics: true   # metrics.json for analytics
```

### CLI Interface

```bash
# Simple task
forge headless --task "Fix linting errors"

# With config file
forge headless --config .forge/headless-config.yml

# Override specific settings
forge headless \
  --config .forge/headless-config.yml \
  --max-files 20 \
  --timeout 600

# Read-only mode
forge headless --task "Analyze code quality" --read-only
```

### Exit Codes

```
0 - Success (task completed, quality gates passed)
1 - General failure (task failed to complete)
2 - Constraint violation (safety limit exceeded)
3 - Quality gate failure (tests/lints failed)
4 - Timeout (execution exceeded time limit)
5 - Configuration error (invalid config)
6 - Git error (workspace dirty, conflicts)
```

### Integration with Existing Systems

#### Tool System
- Tools implement same interface (no changes needed)
- Previewable interface optional in headless mode (skip preview generation)
- Tool approval happens via ConstraintManager instead of user input

#### Memory System
- Works identically in both modes
- Context summarization still applies
- Memory stored in-process only (no persistence needed for headless)

#### Prompt System
- Same system prompts and tool definitions
- Headless mode adds constraint documentation to system prompt
- Error recovery prompts work identically

#### Provider System
- No changes needed
- Works with any LLM provider (OpenAI, Anthropic, etc.)

---

## Validation

### Success Metrics

- Agent loop code reuse > 95%
- Tool code reuse > 90%
- Headless runs complete < 5 minutes for typical tasks
- Constraint violations caught before damage (100% prevention)
- Quality gate integration works with major test frameworks
- Clear documentation and examples for common use cases

### Monitoring

- Track headless execution count and success rate
- Monitor constraint violations by type
- Measure time-to-completion for different task types
- Track quality gate pass/fail rates
- Monitor token usage in headless vs. interactive mode

---

## Related Decisions

- [ADR-0001](0001-record-architecture-decisions.md) - Establishes ADR process
- [ADR-0008](0008-agent-controlled-loop-termination.md) - Agent loop control (applies to both modes)
- [ADR-0010](0010-tool-approval-mechanism.md) - Tool approval system (extended for constraint-based approval)
- [ADR-0014](0014-composable-context-management.md) - Context management (used in both modes)
- [ADR-0017](0017-auto-approval-and-settings-system.md) - Auto-approval (conceptually similar to constraint system)

---

## References

- [Headless CI/CD Mode PRD](../product/features/headless-ci-mode.md)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Conventional Commits](https://www.conventionalcommits.org/)

---

## Notes

This ADR establishes the foundation for autonomous execution. Future ADRs will detail:
- ADR-0027: Safety Constraint System
- ADR-0028: Quality Gate Architecture  
- ADR-0029: Headless Git Integration
- ADR-0030: Artifact Generation System

**Last Updated:** 2025-01-21
