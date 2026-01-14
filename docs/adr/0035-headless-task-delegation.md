# ADR-0035: Headless Task Delegation

## Status

Accepted

## Context

Forge currently operates in two distinct modes:

1. **Interactive Mode**: Full conversational agent with user interaction via `ask_question` and `converse` tools
2. **Headless Mode**: Autonomous execution with safety constraints, quality gates, and artifact generation

Interactive mode excels at collaborative development but can become inefficient when handling routine, well-defined subtasks that don't require user interaction. For example:

- Running test suites and analyzing failures
- Applying linting fixes across multiple files  
- Generating documentation from code
- Performing focused refactoring tasks with clear constraints

Currently, the interactive agent must handle these tasks directly, which:
- Consumes tokens from the interactive session's context window
- Requires sequential processing when parallel execution would be faster
- Mixes routine automation with higher-level planning in a single conversation

**Existing Headless Infrastructure:**

Forge already has a robust headless execution system (`pkg/executor/headless/`) with:
- YAML-based configuration defining task, constraints, quality gates
- Safety mechanisms (file/line limits, tool restrictions, timeouts)
- Quality gate validation with automatic retry on failure
- Artifact generation (JSON results, markdown summaries, metrics)
- Git integration (auto-commit, PR creation per ADR-0031)
- Configurable output directory at `.forge/artifacts`

**Gap:**

There is no mechanism for the interactive agent to delegate focused subtasks to autonomous headless executions. The interactive agent cannot compose headless tasks as part of its problem-solving workflow.

## Decision

We will enable **Headless Task Delegation** by teaching the interactive agent to leverage existing headless infrastructure through configuration files and command execution.

### Approach: Pattern-Based Delegation (No New Tools)

Instead of creating a dedicated delegation tool, we leverage Forge's existing primitives:

1. **write_file**: Create headless config YAML in `.forge/headless/` directory
2. **execute_command**: Run `forge -headless -headless-config <path>` 
3. **read_file**: Parse artifacts from `.forge/artifacts/` for results/metrics
4. **execute_command**: Clean up temporary files after delegation completes

This approach is taught through **system prompt guidance** that explains the delegation pattern.

### Configuration File Location

Headless delegation configs are written to **`.forge/headless/`** subdirectory:
- Follows existing `.forge/` convention (already used for artifacts, git state)
- Keeps delegation configs co-located with execution artifacts
- Automatically excluded from git commits (`.forge/` in `.gitignore`)
- Easier cleanup and management than scattered `/tmp/` files

### Delegation Workflow

```
1. Interactive Agent identifies delegatable subtask
2. Agent writes headless config YAML to .forge/headless/{name}-{timestamp}.yaml
3. Agent executes: forge -headless -headless-config .forge/headless/{name}-{timestamp}.yaml
4. Headless executor runs task autonomously with safety constraints
5. Headless executor writes artifacts to .forge/artifacts/
6. Interactive agent reads artifacts to incorporate results
7. Interactive agent optionally cleans up config file
```

**Naming Convention:** Config filenames use `{name}-{timestamp}.yaml` format where:
- `{name}`: Kebab-case task identifier (e.g., `run-tests`, `fix-linting`, `generate-docs`)
- `{timestamp}`: Unix timestamp for uniqueness and ordering
- Example: `run-tests-1234567890.yaml`, `fix-linting-1234567891.yaml`

### System Prompt Guidance

The interactive agent's system prompt will include a dedicated section explaining:

- When to delegate (routine tasks, focused scope, no user input needed)
- How to structure headless configs (task, mode, constraints, quality_gates)
- Where to write configs (`.forge/headless/` directory)
- How to execute (`forge -headless -headless-config <path>`)
- How to read results (`.forge/artifacts/*.json` and `*.md`)
- Best practices (sequential delegation, cleanup, error handling)

### Initial Constraints

**Phase 1 (MVP):**
- Sequential delegation only (one headless task at a time)
- No nested delegation (headless cannot spawn sub-headless)
- Read-only and write modes both supported
- Manual cleanup of delegation configs (no auto-cleanup)

**Phase 2 (Future):**
- Parallel delegation for independent subtasks
- Automatic cleanup of successful delegation artifacts
- Delegation result caching to avoid redundant work
- Enhanced error recovery and retry strategies

## Alternatives Considered

### 1. Dedicated `delegate_task` Tool

**Approach:** Create a specialized tool that encapsulates delegation logic.

**Pros:**
- Simpler for the agent (single tool call vs multi-step pattern)
- Could enforce delegation best practices automatically
- Easier to add features like parallel execution

**Cons:**
- Adds code complexity and maintenance burden
- Less flexible than composing existing primitives
- Requires new implementation when existing infrastructure works
- Harder to debug than transparent file-based approach

**Rejected:** Over-engineering for a capability that existing tools already provide. Teaching the pattern is simpler and more maintainable.

### 2. In-Process Delegation (Shared Agent Runtime)

**Approach:** Spawn headless executions as in-process goroutines sharing the parent agent runtime.

**Pros:**
- Faster than subprocess spawning
- Could share LLM provider connection pool
- Easier state sharing between parent and child

**Cons:**
- Complex lifecycle management (context cancellation, error propagation)
- Risk of state pollution between parent and delegated tasks
- Harder to implement resource limits and isolation
- Headless executor expects clean process boundary

**Rejected:** Subprocess isolation is a feature, not a bug. Clean separation prevents context contamination and makes delegation failures containable.

### 3. Message Queue-Based Delegation

**Approach:** Use a message queue (Redis, RabbitMQ) for task delegation.

**Pros:**
- Natural support for parallel execution
- Could enable distributed execution across machines
- Better observability and monitoring

**Cons:**
- Massive infrastructure overhead for a development tool
- Requires external dependencies (deal-breaker for local dev)
- Over-engineered for single-machine use case
- Increases deployment and operational complexity

**Rejected:** Forge is a local development tool, not a distributed system. File-based delegation is sufficient and requires no external dependencies.

## Consequences

### Positive

1. **Reuses Proven Infrastructure**: Leverages existing headless executor with safety constraints, quality gates, and artifact generation
2. **Zero Code Overhead**: No new tools or systems to implement/maintain - just documentation
3. **Flexible Composition**: Agent can adapt delegation strategy based on task requirements
4. **Clear Observability**: Config files and artifacts are inspectable/debuggable
5. **Fail-Safe**: Headless safety constraints prevent runaway delegation
6. **Incremental Adoption**: Interactive agent learns delegation pattern naturally through prompt guidance

### Negative

1. **No Parallel Execution (Phase 1)**: Sequential delegation may be slower for independent subtasks
2. **Manual Cleanup Required**: `.forge/headless/` directory accumulates config files
3. **No Nested Delegation**: Headless tasks cannot spawn sub-delegations (could complicate debugging)
4. **Context Switching Overhead**: Subprocess spawn adds latency vs in-process execution
5. **Learning Curve**: Agent must learn multi-step delegation pattern instead of single tool call

### Neutral

1. **Convention Over Configuration**: Relies on `.forge/` directory convention (already established)
2. **Artifact Location**: Interactive agent must know where to read artifacts (`.forge/artifacts/`)
3. **Error Handling**: Agent responsible for detecting delegation failures via exit codes and artifacts

## Implementation Notes

### System Prompt Addition

Add section to interactive agent's system prompt:

```markdown
## Headless Task Delegation

You can delegate focused, routine subtasks to autonomous headless executions:

**When to Delegate:**
- Running test suites and analyzing failures
- Applying automated fixes (linting, formatting)
- Generating documentation or reports
- Focused refactoring with clear scope

**How to Delegate:**
1. Write headless config YAML to `.forge/headless/{name}-{timestamp}.yaml`
2. Execute: `forge -headless -headless-config .forge/headless/{name}-{timestamp}.yaml`
3. Read results from `.forge/artifacts/result.json` and `summary.md`
4. Optionally clean up config file after success

**Naming:** Use kebab-case descriptive names (run-tests, fix-linting, generate-docs) plus timestamp

**Config Structure:**
- task: Clear description of the subtask
- mode: "write" or "read-only"
- constraints: File limits, timeouts, allowed tools
- quality_gates: Validation commands (tests, linting)

**Best Practices:**
- Delegate sequentially (one task at a time in Phase 1)
- Use descriptive task names for artifact correlation
- Set conservative constraints (max_files, max_lines_changed)
- Always include quality gates for write mode
- Clean up successful delegation configs to avoid clutter
```

### Example Delegation Pattern

Interactive agent thinking:
```
I need to run the test suite and fix any failures. This is a good 
candidate for delegation since it's focused and doesn't need user input.

I'll:
1. Write a headless config for "run tests and fix failures"
2. Set quality_gates to ensure tests pass before committing
3. Execute the headless task
4. Read the results to see what was fixed
```

### Directory Structure

```
.forge/
├── headless/              # Delegation configs (gitignored)
│   ├── run-tests-1234567890.yaml
│   ├── fix-linting-1234567891.yaml
│   └── generate-docs-1234567892.yaml
├── artifacts/             # Headless execution outputs
│   ├── result.json
│   ├── summary.md
│   └── metrics.json
└── .gitignore            # Excludes headless/ and artifacts/
```

## Future Enhancements

1. **Parallel Delegation**: Execute multiple independent headless tasks concurrently
2. **Auto-Cleanup**: Remove successful delegation configs automatically  
3. **Result Caching**: Avoid re-running identical delegated tasks
4. **Delegation Templates**: Pre-defined configs for common patterns (test-fix, lint-fix, doc-gen)
5. **Progress Streaming**: Real-time updates from delegated tasks back to interactive session
6. **Delegation Budget**: Limit total delegated tokens/time per interactive session

## References

- ADR-0031: Headless Git PR Creation
- ADR-0028: Quality Gate Architecture  
- PRD: Headless Task Delegation (`docs/product/features/headless-task-delegation.md`)
- Existing Implementation: `pkg/executor/headless/`
