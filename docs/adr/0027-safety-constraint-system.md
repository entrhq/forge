# 27. Safety Constraint System for Headless Mode

**Status:** Accepted
**Date:** 2025-01-21
**Deciders:** Forge Core Team
**Technical Story:** Implement comprehensive safety constraints to prevent runaway autonomous execution

---

## Context

Headless mode (ADR-0026) enables autonomous execution without human oversight. This introduces significant risks: an agent could modify unlimited files, consume excessive tokens, or run indefinitely. We need a robust constraint system that prevents damage while allowing productive autonomous work.

### Background

Interactive mode relies on human approval to enforce safety—users review and approve each tool call. In headless mode, this safety mechanism is absent. Without constraints, a misconfigured agent could:

- Modify thousands of files (far beyond the intended scope)
- Consume hundreds of dollars in API costs
- Run for hours without completing
- Modify critical system files or configuration
- Apply changes that break the build

These scenarios are unacceptable for production use. Teams need confidence that autonomous execution stays within safe, predictable bounds.

### Problem Statement

We need a constraint system that:

1. **Prevents scope creep**: Limits number of files and lines modified
2. **Controls costs**: Enforces token usage budgets
3. **Ensures timely completion**: Timeout enforcement
4. **Restricts file access**: Glob pattern allowlists/denylists
5. **Limits tool usage**: Only allow safe, non-interactive tools
6. **Fails fast**: Detects violations immediately and aborts cleanly
7. **Provides clarity**: Clear error messages explaining which constraint was violated

### Goals

- Implement comprehensive constraint enforcement at the executor level
- Provide fail-fast behavior when constraints are violated
- Support both file-based and CLI-based constraint configuration
- Enable override capabilities for power users
- Generate detailed violation reports for debugging
- Preserve partial work for inspection when constraints are violated

### Non-Goals

- Dynamic constraint adjustment based on context (P2 feature)
- Risk scoring for proposed changes (future ML enhancement)
- Interactive approval for high-risk changes (P2 feature)
- Per-tool custom constraint rules (keep it simple for v1)

---

## Decision Drivers

* **Safety First**: Must prevent runaway execution in all scenarios
* **Fail-Fast**: Violations detected and handled immediately
* **Developer Experience**: Clear error messages and reasonable defaults
* **Flexibility**: Support both simple and complex constraint configurations
* **Performance**: Constraint checking must have minimal overhead
* **Observability**: Detailed logging of constraint evaluation

---

## Considered Options

### Option 1: Runtime Constraint Checking in Tools

**Description:** Each tool checks constraints before executing and aborts if violated.

**Pros:**
- Granular control at tool level
- Tools can implement custom constraint logic
- No central coordinator needed

**Cons:**
- Constraint logic scattered across codebase
- Difficult to enforce global limits (e.g., total token usage)
- Harder to track cumulative changes (files modified across multiple tool calls)
- No single source of truth for constraint state

### Option 2: Executor-Level Constraint Manager

**Description:** HeadlessExecutor owns a ConstraintManager that tracks all constraints and validates tool calls before execution.

**Pros:**
- Centralized constraint logic and state
- Easy to track cumulative metrics (total files, total tokens)
- Clean abort on violation (executor controls lifecycle)
- Single source of truth for constraint configuration
- Tools remain constraint-agnostic (better separation of concerns)

**Cons:**
- Executor must intercept tool calls
- Requires state tracking in executor
- Additional abstraction layer

### Option 3: Constraint Middleware Layer

**Description:** Introduce a middleware system that wraps tool execution and checks constraints before/after each call.

**Pros:**
- Clean separation via middleware pattern
- Extensible for future constraint types
- Tools unaware of constraints
- Could be reused for other cross-cutting concerns

**Cons:**
- Over-engineering for v1 needs
- Additional complexity and indirection
- May complicate debugging
- Overkill when executor already coordinates tool calls

---

## Decision

**Chosen Option:** Option 2 - Executor-Level Constraint Manager

### Rationale

1. **Centralized State**: Global constraints (token usage, total files modified) require centralized tracking. The executor already coordinates tool execution, making it the natural owner.

2. **Clean Lifecycle Control**: The executor controls the execution lifecycle—it can cleanly abort and preserve partial work when a constraint is violated.

3. **Separation of Concerns**: Tools focus on their core logic. Constraints are a deployment/execution concern, not a tool concern.

4. **Existing Pattern**: The executor already handles tool approval (ADR-0010). Constraint checking is conceptually similar—both gate tool execution.

5. **Simplicity**: Avoids introducing a new middleware layer while still achieving clean separation.

The constraint manager is owned by the HeadlessExecutor and consulted before each tool call. On violation, the executor immediately aborts execution while preserving partial work for inspection.

---

## Consequences

### Positive

- Centralized constraint enforcement prevents violations
- Clear ownership: executor manages constraints, tools execute logic
- Abort behavior is clean and reliable (executor-controlled)
- Easy to add new constraint types in one place
- Constraint state visible for debugging and reporting
- Tools remain simple and reusable across modes
- Partial work preserved for inspection when violations occur

### Negative

- Executor becomes more complex (constraint tracking, validation)
- Must carefully design constraint validation hooks
- State tracking adds memory overhead (minimal in practice)
- Tools cannot implement custom constraint-aware optimizations

### Neutral

- Constraint configuration part of executor config, not tool config
- Violations result in executor-level errors, not tool-level errors
- Constraint metrics included in execution artifacts

---

## Implementation

### Core Components

#### 1. ConstraintManager

```go
// ConstraintManager enforces safety limits during headless execution
type ConstraintManager struct {
    config *ConstraintConfig
    
    // Runtime state tracking
    filesModified   map[string]*FileModification
    tokensUsed      int
    startTime       time.Time
    
    mu sync.RWMutex
}

type ConstraintConfig struct {
    // File modification limits
    MaxFiles        int      // Maximum number of files that can be modified
    MaxLinesChanged int      // Maximum total lines added/removed
    AllowedPatterns []string // Glob patterns for allowed files
    DeniedPatterns  []string // Glob patterns for denied files
    
    // Tool restrictions
    AllowedTools []string // Whitelist of tool names
    
    // Resource limits
    MaxTokens int           // Maximum LLM tokens to consume
    Timeout   time.Duration // Maximum execution time
}

type FileModification struct {
    Path         string
    LinesAdded   int
    LinesRemoved int
}

// Constraint validation methods
func (cm *ConstraintManager) ValidateToolCall(toolName string, args interface{}) error
func (cm *ConstraintManager) RecordFileModification(path string, linesAdded, linesRemoved int) error
func (cm *ConstraintManager) RecordTokenUsage(tokens int) error
func (cm *ConstraintManager) CheckTimeout() error
func (cm *ConstraintManager) GetCurrentState() *ConstraintState
```

#### 2. Constraint Validation Flow

```go
// In HeadlessExecutor
func (e *HeadlessExecutor) executeToolCall(ctx context.Context, toolCall *tools.ToolCall) error {
    // 1. Check timeout before each tool call
    if err := e.constraintMgr.CheckTimeout(); err != nil {
        return &ConstraintViolation{Type: TimeoutViolation, Err: err}
    }
    
    // 2. Validate tool is allowed
    if err := e.constraintMgr.ValidateToolCall(toolCall.ToolName, toolCall.Arguments); err != nil {
        return &ConstraintViolation{Type: ToolRestrictionViolation, Err: err}
    }
    
    // 3. Execute tool
    result, err := e.agent.ExecuteTool(ctx, toolCall)
    if err != nil {
        return err
    }
    
    // 4. Record modifications (for file-modifying tools)
    if isFileModifyingTool(toolCall.ToolName) {
        modifications := extractModifications(result)
        for _, mod := range modifications {
            if err := e.constraintMgr.RecordFileModification(mod.Path, mod.LinesAdded, mod.LinesRemoved); err != nil {
                return &ConstraintViolation{Type: FileModificationViolation, Err: err}
            }
        }
    }
    
    return nil
}
```

#### 3. File Pattern Matching

```go
type PatternMatcher struct {
    allowedPatterns []glob.Glob
    deniedPatterns  []glob.Glob
}

func NewPatternMatcher(allowed, denied []string) (*PatternMatcher, error) {
    // Compile glob patterns at initialization
    // Return error if patterns are invalid
}

func (pm *PatternMatcher) IsAllowed(path string) bool {
    // Denied patterns take precedence
    for _, pattern := range pm.deniedPatterns {
        if pattern.Match(path) {
            return false
        }
    }
    
    // If no allowed patterns specified, allow all (except denied)
    if len(pm.allowedPatterns) == 0 {
        return true
    }
    
    // Check if path matches any allowed pattern
    for _, pattern := range pm.allowedPatterns {
        if pattern.Match(path) {
            return true
        }
    }
    
    return false
}
```

#### 4. ConstraintViolation Error Type

```go
type ConstraintViolation struct {
    Type    ViolationType
    Err     error
    Details map[string]interface{}
}

type ViolationType int

const (
    FileModificationViolation ViolationType = iota
    ToolRestrictionViolation
    TokenLimitViolation
    TimeoutViolation
    PatternViolation
)

func (cv *ConstraintViolation) Error() string {
    return fmt.Sprintf("constraint violated: %s - %v", cv.Type, cv.Err)
}

func (cv *ConstraintViolation) ExitCode() int {
    return 2 // Constraint violation exit code
}
```

### Constraint Configuration

#### YAML Configuration

```yaml
constraints:
  # File modification limits
  max_files: 10              # Max files modified in single run
  max_lines_changed: 500     # Max total lines added/removed
  
  # File access patterns (glob syntax)
  allowed_patterns:
    - "src/**/*.go"          # Only Go files in src/
    - "pkg/**/*.go"
    - "docs/**/*.md"
  
  denied_patterns:
    - "vendor/**"            # Never touch vendored code
    - ".git/**"              # Never modify git metadata
    - "**/*_generated.go"    # Never modify generated files
  
  # Tool restrictions
  allowed_tools:
    - read_file
    - write_file
    - apply_diff
    - search_files
    - list_files
    - execute_command
  # Note: ask_question and converse automatically blocked in headless mode
  
  # Resource limits
  max_tokens: 50000          # Max LLM tokens to consume
  timeout: 300               # Max execution time (seconds)
```

#### Default Configuration

```go
func DefaultConstraintConfig() *ConstraintConfig {
    return &ConstraintConfig{
        MaxFiles:        10,
        MaxLinesChanged: 500,
        AllowedPatterns: []string{}, // Empty = allow all (except denied)
        DeniedPatterns: []string{
            ".git/**",
            "vendor/**",
            "node_modules/**",
            "**/*_generated.*",
        },
        AllowedTools: []string{
            "read_file",
            "write_file", 
            "apply_diff",
            "search_files",
            "list_files",
            "execute_command",
            "task_completion", // Loop-breaking tools always allowed
        },
        MaxTokens: 50000,
        Timeout:   5 * time.Minute,
    }
}
```

### Validation Logic

#### File Modification Tracking

```go
func (cm *ConstraintManager) RecordFileModification(path string, linesAdded, linesRemoved int) error {
    cm.mu.Lock()
    defer cm.mu.Unlock()
    
    // Check file pattern
    if !cm.patternMatcher.IsAllowed(path) {
        return fmt.Errorf("file %s not allowed by patterns", path)
    }
    
    // Check max files limit
    if _, exists := cm.filesModified[path]; !exists {
        if len(cm.filesModified) >= cm.config.MaxFiles {
            return fmt.Errorf("max files limit exceeded: %d >= %d", 
                len(cm.filesModified)+1, cm.config.MaxFiles)
        }
    }
    
    // Update or create file modification record
    if mod, exists := cm.filesModified[path]; exists {
        mod.LinesAdded += linesAdded
        mod.LinesRemoved += linesRemoved
    } else {
        cm.filesModified[path] = &FileModification{
            Path:         path,
            LinesAdded:   linesAdded,
            LinesRemoved: linesRemoved,
        }
    }
    
    // Check total lines changed
    totalLines := cm.calculateTotalLinesChanged()
    if totalLines > cm.config.MaxLinesChanged {
        return fmt.Errorf("max lines changed exceeded: %d > %d", 
            totalLines, cm.config.MaxLinesChanged)
    }
    
    return nil
}

func (cm *ConstraintManager) calculateTotalLinesChanged() int {
    total := 0
    for _, mod := range cm.filesModified {
        total += mod.LinesAdded + mod.LinesRemoved
    }
    return total
}
```

#### Tool Validation

```go
func (cm *ConstraintManager) ValidateToolCall(toolName string, args interface{}) error {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    
    // Always block interactive tools in headless mode
    if isInteractiveTool(toolName) {
        return fmt.Errorf("interactive tool %s not allowed in headless mode", toolName)
    }
    
    // Check if tool is in allowed list
    if len(cm.config.AllowedTools) > 0 {
        allowed := false
        for _, allowedTool := range cm.config.AllowedTools {
            if allowedTool == toolName {
                allowed = true
                break
            }
        }
        if !allowed {
            return fmt.Errorf("tool %s not in allowed list", toolName)
        }
    }
    
    return nil
}

func isInteractiveTool(toolName string) bool {
    interactiveTools := map[string]bool{
        "ask_question": true,
        "converse":     true,
    }
    return interactiveTools[toolName]
}
```

#### Timeout Checking

```go
func (cm *ConstraintManager) CheckTimeout() error {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    
    elapsed := time.Since(cm.startTime)
    if elapsed > cm.config.Timeout {
        return fmt.Errorf("execution timeout exceeded: %v > %v", elapsed, cm.config.Timeout)
    }
    
    return nil
}
```

### Constraint Violation Handling

```go
func (e *HeadlessExecutor) Run(ctx context.Context) error {
    // Execute task with constraint monitoring
    err := e.executeTask(ctx)
    
    // Handle constraint violations
    if violation, ok := err.(*ConstraintViolation); ok {
        // Log violation details
        e.logger.Error("Constraint violated", 
            "type", violation.Type,
            "details", violation.Details)
        
        // Mark execution as failed
        e.summary.Status = "failed"
        e.summary.Error = violation.Error()
        
        // Note: Partial work is preserved for inspection
        // This allows developers to review what the agent accomplished
        // before hitting the constraint limit
        e.logger.Info("Partial work preserved for inspection")
        
        // Generate violation report
        e.artifactWriter.WriteViolationReport(violation)
        
        return violation // Return with exit code 2
    }
    
    return err
}
```

### Observability

#### Constraint State Reporting

```go
type ConstraintState struct {
    FilesModified   int
    LinesAdded      int
    LinesRemoved    int
    TokensUsed      int
    ElapsedTime     time.Duration
    ViolationRisk   map[string]float64 // Percentage of limit used
}

func (cm *ConstraintManager) GetCurrentState() *ConstraintState {
    cm.mu.RLock()
    defer cm.mu.RUnlock()
    
    state := &ConstraintState{
        FilesModified: len(cm.filesModified),
        TokensUsed:    cm.tokensUsed,
        ElapsedTime:   time.Since(cm.startTime),
        ViolationRisk: make(map[string]float64),
    }
    
    for _, mod := range cm.filesModified {
        state.LinesAdded += mod.LinesAdded
        state.LinesRemoved += mod.LinesRemoved
    }
    
    // Calculate violation risk percentages
    state.ViolationRisk["files"] = float64(state.FilesModified) / float64(cm.config.MaxFiles)
    state.ViolationRisk["lines"] = float64(state.LinesAdded + state.LinesRemoved) / float64(cm.config.MaxLinesChanged)
    state.ViolationRisk["tokens"] = float64(state.TokensUsed) / float64(cm.config.MaxTokens)
    state.ViolationRisk["time"] = state.ElapsedTime.Seconds() / cm.config.Timeout.Seconds()
    
    return state
}
```

#### Metrics in Artifacts

```json
{
  "execution_id": "run-12345",
  "constraints": {
    "configured": {
      "max_files": 10,
      "max_lines_changed": 500,
      "max_tokens": 50000,
      "timeout_seconds": 300
    },
    "actual": {
      "files_modified": 7,
      "lines_added": 145,
      "lines_removed": 89,
      "tokens_used": 12450,
      "elapsed_seconds": 45
    },
    "utilization": {
      "files": 0.70,
      "lines": 0.47,
      "tokens": 0.25,
      "time": 0.15
    },
    "violations": []
  }
}
```

---

## Validation

### Success Metrics

- Constraint violations abort execution cleanly with preserved partial work
- Constraint checking overhead < 5ms per tool call
- Configuration validation catches invalid patterns before execution
- Clear violation messages enable users to adjust config correctly
- Constraint state visible in real-time logs and final artifacts

### Test Scenarios

1. **Max Files Exceeded**: Agent attempts to modify 11 files when limit is 10
2. **Max Lines Exceeded**: Agent modifies files totaling 600 lines when limit is 500
3. **Pattern Violation**: Agent tries to modify `vendor/lib.go` when vendor/** is denied
4. **Tool Restriction**: Agent calls `ask_question` in headless mode
5. **Timeout**: Execution takes 6 minutes when timeout is 5 minutes
6. **Token Limit**: Agent uses 55,000 tokens when limit is 50,000

Each test verifies:
- Violation detected immediately
- Execution aborted cleanly
- Partial work preserved
- Clear error message
- Exit code 2 returned

---

## Related Decisions

- [ADR-0026](0026-headless-mode-architecture.md) - Headless mode architecture (defines executor structure)
- [ADR-0010](0010-tool-approval-mechanism.md) - Tool approval mechanism (conceptually similar gating)
- [ADR-0017](0017-auto-approval-and-settings-system.md) - Auto-approval system (similar pattern for rules)

---

## References

- [Headless CI/CD Mode PRD](../product/features/headless-ci-mode.md)
- [Go filepath.Match documentation](https://pkg.go.dev/path/filepath#Match)
- [Glob pattern syntax](https://github.com/gobwas/glob)

---

## Notes

The constraint system is designed to be strict by default but configurable for advanced use cases. Future enhancements could include:
- Dynamic constraint adjustment based on task complexity
- Risk scoring for proposed changes
- Constraint profiles (conservative, moderate, aggressive)
- Per-tool custom constraints

**Last Updated:** 2025-01-21

## Implementation Notes

**Rollback Decision (2025-01-21):** After implementation, we decided NOT to implement automatic rollback of changes on constraint violations. Instead, partial work is preserved for inspection. This decision was made because:

1. **Debugging Value**: Developers need to see what the agent accomplished before hitting limits
2. **Iterative Improvement**: Preserved work helps adjust constraints for future runs
3. **Transparency**: Easier to understand agent behavior when work is visible
4. **Git Safety**: Users can manually revert if needed using standard git tools
5. **Simplicity**: Avoids complex snapshot/restore logic

This aligns with the quality gate behavior, which also preserves changes on failure (see `quality_gate.go` line 299).
