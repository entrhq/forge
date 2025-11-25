# Quality Gates Retry Mechanism

## Overview

The headless executor now supports automatic retry of quality gate checks when they fail. This allows the AI agent to iteratively improve its work based on quality gate feedback, similar to how a human developer would refactor code after seeing test failures or linting errors.

## How It Works

### 1. Quality Gate Execution at Turn End

When the agent completes a turn (signals `EventTypeTurnEnd`), the executor:
1. Runs all configured quality gates
2. Evaluates the results
3. Takes action based on pass/fail status

### 2. Retry Logic

If quality gates fail:
- **First Attempts (1 to max-1)**: The executor sends feedback to the agent with:
  - Which quality gates failed
  - Error details for each failure
  - Current retry attempt number
  - Clear instructions to fix the issues and call `task_completion` when done
  
- **Max Retries Exceeded**: The executor:
  - Marks the execution as failed
  - Records all quality gate errors in the summary
  - Shuts down the agent

If quality gates pass:
- The executor proceeds with normal shutdown
- Execution is marked as successful

### 3. Feedback Message Format

The feedback message sent to the agent includes:

```
⚠️ Quality Gate Failures (attempt X/Y)

The following quality gates failed:

❌ gate-name
Error: detailed error message

[Additional failed gates...]

Please fix these issues and call task_completion when done.
Note: Do NOT call task_completion until all issues are resolved.
```

This format:
- Clearly indicates retry status
- Lists all failures with details
- Provides explicit instructions
- Prevents premature completion

## Configuration

### Per-Gate Configuration

Each quality gate can specify its own max retries:

```yaml
quality_gates:
  - name: lint
    command: golangci-lint run
    required: true
    max_retries: 3  # Gate-specific retry limit
```

### Global Configuration

Set a default max retries for all quality gates:

```yaml
executor:
  quality_gate_max_retries: 3  # Default: 3
```

Priority: Gate-specific `max_retries` overrides global `quality_gate_max_retries`.

## Implementation Details

### Key Components

1. **QualityGateRunner**: Executes quality gates and collects results
2. **QualityGateResults**: Holds pass/fail status and error details
3. **Executor.qualityGateRetryCount**: Tracks current retry attempt
4. **FormatFeedbackMessage()**: Generates user-friendly feedback for the agent

### Event Flow

```
Agent Turn End
    ↓
Run Quality Gates
    ↓
All Passed? ──Yes──→ Shutdown (Success)
    ↓ No
Retries < Max? ──No──→ Shutdown (Failed)
    ↓ Yes
Send Feedback to Agent
    ↓
Agent Retry Turn
    ↓
(Loop back to Run Quality Gates)
```

### Code Locations

- Quality gate execution: `pkg/executor/headless/executor.go` (turn end event handler)
- Retry tracking: `pkg/executor/headless/executor.go` (Executor struct)
- Feedback formatting: `pkg/executor/headless/quality_gate.go`
- Configuration: `pkg/executor/headless/config.go`

## Best Practices

### 1. Configure Appropriate Retry Limits

- **Simple fixes** (linting, formatting): 1-2 retries may suffice
- **Complex fixes** (test failures, build errors): 3-5 retries recommended
- **Avoid excessive retries**: More than 5 retries often indicates the task is too complex

### 2. Write Clear Quality Gate Commands

Quality gates should:
- Provide actionable error messages
- Be deterministic (same input → same output)
- Run quickly (avoid long-running tests in quality gates)
- Focus on measurable criteria

### 3. Use Required vs. Optional Gates

- **Required gates**: Block execution on failure (use for critical checks)
- **Optional gates**: Record warnings but don't block (use for nice-to-have checks)

### 4. Monitor Retry Patterns

Track which gates trigger retries most often to:
- Improve agent instructions
- Refine quality gate criteria
- Identify common failure patterns

## Example Scenario

### Initial Attempt
```
Agent writes code → Quality gates run → Linter fails
```

### Retry 1
```
Agent receives feedback:
"❌ lint - Error: unused variable 'x' at line 42"
→ Agent fixes the issue
→ Quality gates run → All pass
→ Execution succeeds
```

### Max Retries Scenario
```
Attempt 1: Tests fail
Attempt 2: Tests still fail
Attempt 3: Tests still fail (max retries reached)
→ Execution fails with detailed error report
```

## Limitations

1. **No Rollback Between Retries**: Changes from failed attempts are preserved
2. **Single Failure Point**: If the agent can't understand feedback, retries won't help
3. **No Progressive Hints**: Each retry gets the same feedback format (no escalating hints)

## Future Enhancements

Potential improvements:
- Progressive feedback (more detailed hints on later retries)
- Partial rollback (restore specific files on retry)
- Smart retry limits (adjust based on failure type)
- Retry metrics and analytics
- Custom feedback templates per quality gate
