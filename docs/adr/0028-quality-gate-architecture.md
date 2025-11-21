# 28. Quality Gate Architecture for Headless Mode

**Status:** Proposed
**Date:** 2025-01-21
**Deciders:** Forge Core Team
**Technical Story:** Implement quality gates to validate autonomous changes before committing

---

## Context

Headless mode (ADR-0026) enables autonomous code modifications with safety constraints (ADR-0027). However, constraints alone cannot ensure correctness—an agent might modify exactly 10 files within all constraints but still break the build, fail tests, or introduce bugs.

**Key Insight:** Rather than treating quality gate failures as terminal errors that require human intervention, we can leverage the agent's reasoning capabilities to **autonomously fix issues and retry**. Quality gates become a feedback mechanism that helps the agent iteratively improve its work.

### Background

Traditional CI/CD treats quality checks as pass/fail gates. In Forge's headless mode, we can do better:

**Traditional Approach:**
1. Agent makes changes
2. Run quality gates
3. If failure → report error, exit
4. Human must fix issues manually

**Forge's Iterative Approach:**
1. Agent attempts task completion
2. Intercept completion, run quality gates
3. If failure → feed error back to agent loop
4. Agent reasons about failure and fixes issues
5. Agent attempts completion again
6. Repeat until gates pass or max iterations reached

### Problem Statement

We need a quality gate system that:

1. **Validates changes iteratively**: Run tests/linters after each completion attempt
2. **Feeds failures back to agent**: Treat quality issues as recoverable errors
3. **Enables autonomous fixing**: Agent can fix linting, tests, type errors, etc.
4. **Supports common toolchains**: Works with any command-line validation tool
5. **Prevents infinite loops**: Configurable iteration limits
6. **Provides observability**: Clear reporting of fix-retry cycles

### Goals

- Intercept task_completion events to run quality gates
- Feed gate failures back into agent loop as recoverable errors
- Support command-based gates (works with any tool)
- Enable iterative improvement (agent fixes issues autonomously)
- Configure retry limits to prevent infinite loops
- Generate detailed reports of fix-retry cycles
- Maximize autonomous success rate

### Real-World Example

```
Agent: "I've fixed the bug. Task complete."
Executor: "Running quality gates..."
Gate: go test ./... → FAIL: TestUserService (expected 200, got 404)

Executor: "Quality gate failed. Feeding error back to agent..."

Agent: "The test expects 200 but gets 404. Let me check the route handler..."
Agent: *reads code* "I see - I changed the route but didn't update the handler."
Agent: *applies fix*
Agent: "I've fixed the handler. Task complete."

Executor: "Running quality gates..."
Gate: go test ./... → PASS
Gate: golangci-lint run → PASS

Executor: "All gates passed! Committing changes."
```

This autonomous fix-retry cycle dramatically increases headless mode success rates.

### Non-Goals

- Building quality assessment ML models (use existing tooling)
- Language-specific gate implementations (keep generic)
- Interactive gate approval/override (purely automated)
- Guaranteeing agent can fix all issues (some may require human intervention)

---

## Decision Drivers

* **Autonomous Recovery**: Agent should fix issues autonomously when possible
* **Iteration Control**: Prevent infinite loops with retry limits
* **Flexibility**: Support any command-line validation tool
* **Clear Feedback**: Gate failures provide actionable error messages
* **Observability**: Track fix-retry cycles and success rates
* **Safety**: Rollback changes if max retries exceeded

---

## Considered Options

### Option 1: Final Validation (Traditional CI)

**Description:** Run quality gates after task_completion as final validation. If any gate fails, the entire headless run fails and reports the error.

**Pros:**
- Simple to implement
- Clear pass/fail outcome
- Matches traditional CI behavior
- No risk of infinite loops

**Cons:**
- Agent cannot fix issues autonomously
- Wastes agent work if validation fails
- Requires manual intervention
- Lower success rate for autonomous tasks
- Doesn't leverage agent's reasoning capabilities

### Option 2: Iterative Quality Gates with Agent Loop

**Description:** Intercept task_completion events and run quality gates. If any gate fails, feed the error back into the agent loop as a recoverable error. Agent reasons about the failure and attempts to fix it. Retry until gates pass or max iterations reached.

**Pros:**
- Agent can fix issues autonomously (linting, tests, types, etc.)
- Maximizes agent capabilities
- Higher success rate for autonomous tasks
- Learning loop: agent improves through feedback
- Reduces manual intervention

**Cons:**
- More complex executor logic
- Requires careful iteration limits
- Agent might not always fix issues correctly
- Could waste tokens on unfixable issues
- Need clear observability into retry cycles

### Option 3: Selective Agent Loop Integration

**Description:** Categorize gates as "auto-fixable" vs "terminal". Auto-fixable gates (linting, formatting) feed back to agent. Terminal gates (security scans) fail immediately.

**Pros:**
- Balances autonomy with safety
- Prevents wasted effort on unfixable issues
- Faster failure for critical gates

**Cons:**
- Configuration complexity (which gates are fixable?)
- Hard to predict what agent can/cannot fix
- Requires maintaining gate categorization
- Reduces agent autonomy

---

## Decision

**Chosen Option:** Option 2 - Iterative Quality Gates with Agent Loop

**Implementation:** Command-based gates for flexibility

### Rationale

1. **Maximize Autonomous Success**: Agent can fix common issues (linting errors, test failures, type errors) autonomously, dramatically increasing headless mode success rates.

2. **Learning Feedback Loop**: Quality gate failures are teaching moments. The agent reads error output, reasons about the problem, and applies fixes—just like a human developer would.

3. **Command-Based Flexibility**:
   - Works with any validation tool (go test, npm run lint, pytest, etc.)
   - Teams use their existing tooling
   - Easy to configure (just specify commands)
   - Language-agnostic

4. **Practical Autonomous Fixes**:
   - **Linting errors**: Agent reads linter output, fixes style issues, retries
   - **Test failures**: Agent reads test failure messages, fixes bugs, retries
   - **Type errors**: Agent reads type checker output, fixes types, retries
   - **Build failures**: Agent reads compiler errors, fixes syntax, retries

5. **Safety via Iteration Limits**: Configurable max retries prevent infinite loops. If agent can't fix issues after N attempts, the run fails with clear diagnostics.

6. **Better Than Traditional CI**: Traditional CI just reports "tests failed" and requires human intervention. Forge attempts autonomous fixes first, falling back to human intervention only when necessary.

The trade-offs are acceptable because:
- Iteration limits prevent runaway token usage
- Failed attempts provide valuable debugging information
- Success rate is higher than terminal failures
- Teams can configure retry limits based on their risk tolerance

---

## Consequences

### Positive

- **Higher autonomous success rate**: Agent can fix common issues without human intervention
- **Learning feedback loop**: Agent improves through iterative error correction
- **Works with any validation tool**: Command-based gates support all toolchains
- **Simple configuration**: Just specify commands and retry limits
- **Better than traditional CI**: Attempts autonomous fixes before failing
- **Clear observability**: Reports show fix-retry cycles and what agent attempted

### Negative

- **Token usage from retries**: Failed attempts consume additional tokens
- **Complexity in executor**: Must intercept task_completion and manage retry loop
- **Not all issues fixable**: Some failures will require human intervention
- **Potential for wasted effort**: Agent might struggle with unfixable issues
- **Must execute commands via shell**: Security consideration for command injection

### Neutral

- **Configurable retry limits**: Teams control cost vs. autonomy trade-off
- **Failed attempts logged**: Provides debugging information even on final failure
- **Git rollback on max retries**: Clean failure mode preserves workspace state

---

## Implementation

### Core Components

#### 1. QualityGate Interface

```go
// QualityGate represents a validation check that runs after task completion attempts
type QualityGate interface {
    // Name returns the human-readable gate name
    Name() string
    
    // Execute runs the quality gate and returns a detailed result
    Execute(ctx context.Context, workspaceDir string) *GateResult
}

// GateResult contains the outcome of a quality gate execution
type GateResult struct {
    GateName    string
    Passed      bool
    Output      string // stdout + stderr from gate command
    ExitCode    int
    Duration    time.Duration
    Error       error  // Execution error (not validation failure)
}

// FormatForAgent returns a formatted error message suitable for feeding back to the agent
func (r *GateResult) FormatForAgent() string {
    return fmt.Sprintf(`Quality gate "%s" failed:

Exit Code: %d
Output:
%s

Please analyze the error output above and fix the issues. Once fixed, attempt task completion again.`, 
        r.GateName, r.ExitCode, r.Output)
}
```

#### 2. CommandQualityGate Implementation

```go
// CommandQualityGate executes a shell command as a quality check
type CommandQualityGate struct {
    name       string
    command    string
    timeout    time.Duration
    workingDir string // Relative to workspace, empty = workspace root
    env        map[string]string
}

func NewCommandQualityGate(config GateConfig) *CommandQualityGate {
    return &CommandQualityGate{
        name:       config.Name,
        command:    config.Command,
        timeout:    config.Timeout,
        workingDir: config.WorkingDir,
        env:        config.Env,
    }
}

func (g *CommandQualityGate) Name() string {
    return g.name
}

func (g *CommandQualityGate) Execute(ctx context.Context, workspaceDir string) *GateResult {
    start := time.Now()
    // Determine full working directory
    fullWorkDir := workspaceDir
    if g.workingDir != "" {
        fullWorkDir = filepath.Join(workspaceDir, g.workingDir)
    }
    
    // Create context with timeout
    execCtx := ctx
    if g.timeout > 0 {
        var cancel context.CancelFunc
        execCtx, cancel = context.WithTimeout(ctx, g.timeout)
        defer cancel()
    }
    
    // Execute command
    cmd := exec.CommandContext(execCtx, "sh", "-c", g.command)
    cmd.Dir = fullWorkDir
    
    // Set environment variables
    cmd.Env = os.Environ()
    for k, v := range g.env {
        cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
    }
    
    // Capture output
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    
    // Run command
    err := cmd.Run()
    
    // Combine stdout and stderr for output
    output := stdout.String()
    if stderr.Len() > 0 {
        if len(output) > 0 {
            output += "\n"
        }
        output += stderr.String()
    }
    
    result := &GateResult{
        GateName: g.name,
        Passed:   err == nil,
        Output:   output,
        Duration: time.Since(start),
    }
    
    if err != nil {
        if exitErr, ok := err.(*exec.ExitError); ok {
            result.ExitCode = exitErr.ExitCode()
        } else {
            result.Error = err
        }
    }
    
    return result
}
```

#### 3. HeadlessExecutor Integration

The HeadlessExecutor intercepts task_completion events and runs quality gates before committing changes.

```go
// HeadlessExecutor handles autonomous task execution with quality gates
type HeadlessExecutor struct {
    agent          *agent.DefaultAgent
    workspace      string
    constraints    *ConstraintEnforcer
    gates          []QualityGate
    maxRetries     int // Max iterations for gate failure fixes
    currentRetries int // Current retry count
    logger         *log.Logger
}

// HandleTaskCompletion intercepts task_completion and runs quality gates
func (e *HeadlessExecutor) HandleTaskCompletion(ctx context.Context, result string) error {
    e.logger.Info("Task completion attempt", "retry", e.currentRetries)
    
    // Run all quality gates
    var failedGates []*GateResult
    for _, gate := range e.gates {
        e.logger.Info("Running quality gate", "gate", gate.Name())
        gateResult := gate.Execute(ctx, e.workspace)
        
        if !gateResult.Passed {
            failedGates = append(failedGates, gateResult)
            e.logger.Warn("Quality gate failed", 
                "gate", gateResult.GateName,
                "exitCode", gateResult.ExitCode,
                "duration", gateResult.Duration)
        }
    }
    
    // All gates passed - commit and complete
    if len(failedGates) == 0 {
        e.logger.Info("All quality gates passed", "attempts", e.currentRetries+1)
        return e.commitChanges(ctx, result)
    }
    
    // Gates failed - check if we can retry
    if e.currentRetries >= e.maxRetries {
        e.logger.Error("Max retries exceeded", "maxRetries", e.maxRetries)
        return e.handleMaxRetriesExceeded(ctx, failedGates)
    }
    
    // Feed failures back to agent for fixing
    return e.feedFailuresBackToAgent(ctx, failedGates)
}

// feedFailuresBackToAgent creates an error event with gate failures and continues agent loop
func (e *HeadlessExecutor) feedFailuresBackToAgent(ctx context.Context, failedGates []*GateResult) error {
    e.currentRetries++
    
    // Build error message for agent
    var errorMsg strings.Builder
    errorMsg.WriteString(fmt.Sprintf("Task completion blocked by %d quality gate failure(s):\n\n", len(failedGates)))
    
    for i, result := range failedGates {
        errorMsg.WriteString(fmt.Sprintf("--- Gate %d: %s ---\n", i+1, result.GateName))
        errorMsg.WriteString(result.FormatForAgent())
        errorMsg.WriteString("\n\n")
    }
    
    errorMsg.WriteString(fmt.Sprintf("Retry attempt %d of %d. Please fix the issues above and attempt task completion again.\n", 
        e.currentRetries, e.maxRetries))
    
    // Create execution error event that continues the agent loop
    event := &agent.ExecutionErrorEvent{
        Error:       errors.New(errorMsg.String()),
        Recoverable: true, // Agent can attempt to fix
    }
    
    // Send to agent loop
    return e.agent.HandleEvent(ctx, event)
}

// handleMaxRetriesExceeded handles the case where agent couldn't fix issues
func (e *HeadlessExecutor) handleMaxRetriesExceeded(ctx context.Context, failedGates []*GateResult) error {
    // Rollback all changes
    if err := e.rollbackChanges(ctx); err != nil {
        e.logger.Error("Failed to rollback changes", "error", err)
    }
    
    // Build failure report
    var report strings.Builder
    report.WriteString(fmt.Sprintf("Headless execution failed after %d retry attempts.\n\n", e.maxRetries))
    report.WriteString(fmt.Sprintf("Failed quality gates (%d):\n", len(failedGates)))
    
    for _, result := range failedGates {
        report.WriteString(fmt.Sprintf("\n%s (exit code: %d)\n", result.GateName, result.ExitCode))
        report.WriteString(result.Output)
        report.WriteString("\n")
    }
    
    return fmt.Errorf("quality gates failed after max retries: %s", report.String())
}

// commitChanges commits the successful changes
func (e *HeadlessExecutor) commitChanges(ctx context.Context, result string) error {
    // Git commit logic here
    e.logger.Info("Committing changes", "result", result)
    return nil
}

// rollbackChanges reverts all uncommitted changes
func (e *HeadlessExecutor) rollbackChanges(ctx context.Context) error {
    // Git rollback logic here
    e.logger.Info("Rolling back changes")
    return nil
}
```

#### 4. Agent Event Flow

The iterative quality gate approach integrates with the agent's event system:

```
┌─────────────────────────────────────────────────────────────┐
│                      Agent Loop                              │
│                                                              │
│  1. User assigns task                                        │
│  2. Agent reasons about task                                 │
│  3. Agent makes code changes                                 │
│  4. Agent calls task_completion                              │
│     │                                                        │
│     ├──> HeadlessExecutor intercepts                         │
│     │                                                        │
│     ├──> Run quality gates                                   │
│     │    ├─ go test ./...                                    │
│     │    ├─ golangci-lint run                                │
│     │    └─ go build ./...                                   │
│     │                                                        │
│     ├──> Gates PASSED?                                       │
│     │    │                                                   │
│     │    YES ──> Commit changes, complete task               │
│     │    │                                                   │
│     │    NO ──> Format error message                         │
│     │         │                                              │
│     │         └──> Feed back to agent as ExecutionErrorEvent │
│     │              (with Recoverable=true)                   │
│     │                                                        │
│     └──> Agent receives error                                │
│          │                                                   │
│          ├──> Reads gate failure output                      │
│          ├──> Reasons about what went wrong                  │
│          ├──> Makes fixes                                    │
│          └──> Calls task_completion again (retry)            │
│               │                                              │
│               └──> Loop continues...                         │
│                                                              │
│  If max retries exceeded:                                    │
│    - Rollback all changes                                    │
│    - Return detailed failure report                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

**Key Points:**

1. **task_completion is intercepted** - Never reaches final completion until gates pass
2. **Failures are recoverable** - ExecutionErrorEvent with Recoverable=true keeps agent loop active
3. **Agent sees gate output** - Full stdout/stderr helps agent understand what to fix
4. **Automatic retry** - Agent attempts completion again after making fixes
5. **Safety limit** - Max retries prevents infinite loops

### Configuration

#### YAML Configuration

```yaml
headless:
  max_retries: 3  # Maximum quality gate fix attempts
  
quality_gates:
  # Test execution
  - name: "Run unit tests"
    command: "go test ./..."
    timeout: 300 # seconds
  
  # Linting
  - name: "Lint code"
    command: "golangci-lint run"
    timeout: 60
  
  # Build verification
  - name: "Build check"
    command: "go build ./..."
    timeout: 120
  
  # Type checking (TypeScript example)
  - name: "Type check"
    command: "npm run type-check"
    timeout: 60
    working_dir: "frontend"
  
  # Custom validation script
  - name: "Validate API schema"
    command: "./scripts/validate-schema.sh"
    timeout: 60
    working_dir: "api"
    env:
      SCHEMA_PATH: "openapi.yaml"
```

#### Common Gate Examples

**Go Projects:**
```yaml
headless:
  max_retries: 3

quality_gates:
  - name: "Tests"
    command: "go test -v -race -coverprofile=coverage.out ./..."
  
  - name: "Lint"
    command: "golangci-lint run --timeout 5m"
  
  - name: "Build"
    command: "go build -v ./..."
```

**JavaScript/TypeScript Projects:**
```yaml
headless:
  max_retries: 3

quality_gates:
  - name: "Tests"
    command: "npm test"
  
  - name: "Lint"
    command: "npm run lint"
  
  - name: "Type check"
    command: "npm run typecheck"
  
  - name: "Build"
    command: "npm run build"
```

**Python Projects:**
```yaml
headless:
  max_retries: 3

quality_gates:
  - name: "Tests"
    command: "pytest --cov=src tests/"
  
  - name: "Lint"
    command: "ruff check ."
  
  - name: "Type check"
    command: "mypy src/"
```

### Default Gates

```go
func DefaultQualityGates(language string) []GateConfig {
    switch language {
    case "go":
        return []GateConfig{
            {
                Name:    "Run tests",
                Command: "go test ./...",
                Timeout: 5 * time.Minute,
            },
            {
                Name:    "Build",
                Command: "go build ./...",
                Timeout: 2 * time.Minute,
            },
        }
    case "javascript", "typescript":
        return []GateConfig{
            {
                Name:    "Run tests",
                Command: "npm test",
                Timeout: 5 * time.Minute,
            },
        }
    case "python":
        return []GateConfig{
            {
                Name:    "Run tests",
                Command: "pytest",
                Timeout: 5 * time.Minute,
            },
        }
    default:
        return []GateConfig{} // No default gates for unknown languages
    }
}
```

### Retry Configuration

The `max_retries` setting controls how many times the agent can attempt to fix quality gate failures:

```go
type HeadlessConfig struct {
    MaxRetries int `yaml:"max_retries"` // Default: 3
}
```

**Recommended Values:**
- **3 retries** (default): Good balance for most projects. Allows agent to:
  - First attempt: Initial fix
  - Second attempt: Address edge cases or related issues
  - Third attempt: Final refinement
- **5 retries**: For complex projects with many interdependent tests
- **1 retry**: For fast feedback in simple projects
- **0 retries**: Disable iterative fixing (original pass/fail behavior)

**Token Usage Considerations:**
Each retry consumes tokens for:
- Reading gate failure output
- Agent reasoning about fixes
- Making code changes
- Re-running quality gates

A typical retry cycle might use 2,000-5,000 tokens depending on error complexity.

### Artifact Generation

Quality gate results are saved to the artifacts directory for analysis.

#### Gate Results JSON

```json
{
  "execution_id": "run-12345",
  "quality_gates": {
    "retry_attempts": 2,
    "max_retries": 3,
    "final_status": "passed",
    "total_duration_seconds": 87.6,
    "attempts": [
      {
        "attempt": 1,
        "timestamp": "2024-01-15T10:30:00Z",
        "results": [
          {
            "name": "Run unit tests",
            "passed": false,
            "duration_seconds": 23.5,
            "exit_code": 1,
            "output": "FAIL: TestUserCreation (0.00s)\n    Expected user.ID to be set\n"
          }
        ]
      },
      {
        "attempt": 2,
        "timestamp": "2024-01-15T10:31:15Z",
        "agent_fixes": "Fixed user ID initialization in CreateUser function",
        "results": [
          {
            "name": "Run unit tests",
            "passed": true,
            "duration_seconds": 24.1,
            "output": "ok  \tgithub.com/user/project\t24.089s"
          },
          {
            "name": "Lint code",
            "passed": true,
            "duration_seconds": 8.2,
            "output": ""
          },
          {
            "name": "Build check",
            "passed": true,
            "duration_seconds": 12.1,
            "output": ""
          }
        ]
      }
    ]
  }
}
```

#### Summary in Markdown

```markdown
## Quality Gates - Iterative Fix Cycle

**Final Status:** ✅ All gates passed after 2 attempts

### Attempt 1 - Failed
*Timestamp: 2024-01-15T10:30:00Z*

| Gate | Status | Duration | Exit Code |
|------|--------|----------|-----------|
| Run unit tests | ❌ Fail | 23.5s | 1 |

**Error Output:**
```
FAIL: TestUserCreation (0.00s)
    Expected user.ID to be set
```

**Agent Action:** Analyzed test failure and fixed user ID initialization in CreateUser function

---

### Attempt 2 - Passed
*Timestamp: 2024-01-15T10:31:15Z*

| Gate | Status | Duration |
|------|--------|----------|
| Run unit tests | ✅ Pass | 24.1s |
| Lint code | ✅ Pass | 8.2s |
| Build check | ✅ Pass | 12.1s |

**Total Duration:** 44.4 seconds
**Total Attempts:** 2 of 3 allowed
```

---

## Validation

### Success Metrics

- **Autonomous Fix Rate**: Percentage of gate failures successfully fixed by agent without human intervention
- **Retry Efficiency**: Average number of attempts needed to pass gates (lower is better)
- **Token Efficiency**: Average tokens consumed per successful gate passage
- **False Positive Rate**: Gates that fail but shouldn't (should be near zero)
- **Time to Success**: Total time from initial task to final gate passage

### Test Scenarios

1. **First Attempt Success**: All gates pass on first try → commit immediately
2. **Single Test Failure**: Test fails → agent fixes → gates pass on retry
3. **Lint Error**: Linting fails → agent reformats code → gates pass
4. **Build Error**: Build fails → agent fixes compilation issue → gates pass
5. **Multiple Related Failures**: Tests and lint both fail → agent fixes both → gates pass
6. **Max Retries Exceeded**: Agent can't fix issue after 3 attempts → rollback and report
7. **Timeout During Gate**: Test takes > timeout → reported as error → agent can optimize or skip
8. **Non-fixable Error**: Infrastructure issue → agent recognizes and reports (no retry)

---

## Related Decisions

- [ADR-0026](0026-headless-mode-architecture.md) - Headless mode architecture (defines executor)
- [ADR-0027](0027-safety-constraint-system.md) - Safety constraints (run before gates)
- [ADR-0013](0013-streaming-command-execution.md) - Command execution patterns (similar to gates)

---

## References

- [Headless CI/CD Mode PRD](../product/features/headless-ci-mode.md)
- [GitHub Actions: Defining success and failure](https://docs.github.com/en/actions/learn-github-actions/expressions#job-status-check-functions)
- [Test-Driven Development Best Practices](https://martinfowler.com/bliki/TestDrivenDevelopment.html)

---

## Notes

Quality gates are the last line of defense before committing autonomous changes. They transform headless mode from "might work" to "validated working" code.

Future enhancements could include:
- Parallel gate execution for performance
- Gate result caching (skip if no relevant changes)
- Automatic gate selection based on changed files
- Integration with coverage tools for quality trends

**Last Updated:** 2025-01-21
