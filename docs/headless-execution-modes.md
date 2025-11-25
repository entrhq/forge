# Headless Execution Modes

Forge headless supports two execution modes that control what operations the agent can perform:

## Execution Modes

### Write Mode (Default)

**Mode:** `write`

Write mode allows the agent to perform both read and write operations on the workspace. This is the default mode and supports the full range of available tools.

**Capabilities:**
- Read files and directories
- Write and modify files
- Apply code changes with diffs
- Execute commands
- Run tests and quality gates

**Use Cases:**
- Automated code refactoring
- Bug fixes and feature implementation
- Test generation and updates
- Documentation updates
- Any task requiring code modifications

**Example Configuration:**
```yaml
mode: write  # or omit - write is default

task: |
  Refactor the authentication module to use the new JWT library

constraints:
  allowed_file_patterns:
    - "**/*.go"
  max_files: 10
  max_tokens: 50000
```

### Read-Only Mode

**Mode:** `read-only`

Read-only mode restricts the agent to observation and analysis only. The agent cannot modify files, apply changes, or execute commands that could alter the workspace.

**Capabilities:**
- Read files and examine content
- List directory structures
- Search code with patterns
- Analyze code quality and architecture
- Generate reports and recommendations

**Restrictions:**
- ❌ Cannot use `write_file` tool
- ❌ Cannot use `apply_diff` tool
- ❌ Cannot use `execute_command` tool
- ✅ Can still use loop-breaking tools (task_completion, ask_question, converse)

**Use Cases:**
- Code review and analysis
- Architecture assessment
- Security audits (read-only inspection)
- Documentation review
- Codebase exploration and understanding
- Generating improvement recommendations

**Example Configuration:**
```yaml
mode: read-only

task: |
  Analyze the codebase and provide a security audit report focusing on:
  - Authentication and authorization patterns
  - Input validation practices
  - Potential security vulnerabilities
  - Recommendations for hardening

constraints:
  allowed_tools:
    - read_file
    - list_files
    - search_files
  max_tokens: 100000
  timeout: 15m

output:
  artifact_path: ./security-audit
  create_markdown_report: true
```

## Mode Enforcement

Mode enforcement happens at multiple levels:

### 1. Constraint Manager Validation

The `ConstraintManager` validates every tool call before execution. In read-only mode:

```go
// Read-only mode check - happens before allowed_tools validation
if cm.mode == ModeReadOnly && isFileModifyingTool(toolName) {
    return &ConstraintViolation{
        Type:    ViolationReadOnlyMode,
        Message: fmt.Sprintf("tool %s is not allowed in read-only mode", toolName),
    }
}
```

### 2. System Prompt Guidance

When running in read-only mode, the agent receives special instructions in its system prompt:

```
# READ-ONLY MODE

**CRITICAL:** You are operating in READ-ONLY mode. You CANNOT modify any files or execute commands.

**Restrictions:**
-   **NO file writing**: You CANNOT create, modify, or delete files
-   **NO code changes**: You CANNOT apply diffs or patches
-   **NO command execution**: You CANNOT run commands that modify the workspace

**Your Role:**
-   **Analysis and Observation**: Read files, search code, list directories
-   **Information Gathering**: Examine code structure, dependencies, configuration
-   **Reporting**: Provide detailed analysis, findings, and recommendations
```

### 3. Tool Availability

The agent only has access to read-only tools when configured in read-only mode through the `allowed_tools` constraint.

## Best Practices

### When to Use Write Mode

Use write mode when:
- The task requires modifying code or files
- You need to run tests or build commands
- Automated fixes or refactoring are needed
- Quality gates require code changes to pass

### When to Use Read-Only Mode

Use read-only mode when:
- Performing code reviews or audits
- Analyzing architecture or design patterns
- Generating reports without modifications
- Exploring unfamiliar codebases safely
- Running security assessments
- You want guaranteed non-modification of the workspace

### Configuration Tips

1. **Combine with allowed_tools**: In read-only mode, explicitly list the read tools you want:
   ```yaml
   mode: read-only
   constraints:
     allowed_tools:
       - read_file
       - search_files
       - list_files
   ```

2. **Set appropriate timeouts**: Analysis tasks may need more time:
   ```yaml
   mode: read-only
   constraints:
     timeout: 15m
     max_tokens: 150000
   ```

3. **Use meaningful output paths**: Structure analysis results clearly:
   ```yaml
   output:
     artifact_path: ./analysis-results/{{.timestamp}}
     create_markdown_report: true
   ```

## Safety Guarantees

### Read-Only Mode Guarantees

When using read-only mode, you have absolute guarantee that:
- ✅ No files will be created, modified, or deleted
- ✅ No commands will be executed
- ✅ The workspace remains in its original state
- ✅ Only observation operations are permitted

### Write Mode Considerations

Write mode allows modifications, but still provides safety through:
- File pattern restrictions (allowed/deny patterns)
- Token and file modification limits
- Quality gate validation before completion
- Git integration for tracking changes
- Comprehensive audit logging

## Examples

See the `examples/headless/` directory for complete examples:
- `examples/headless/read-only-analysis.yaml` - Code analysis in read-only mode
- `examples/headless/basic-task.yaml` - Standard task with write mode
- `examples/headless/refactor.yaml` - Code refactoring with constraints

## Troubleshooting

### "Tool not allowed in read-only mode" Error

If you see this error, check:
1. Is your task trying to modify files? Switch to write mode.
2. Did you specify `mode: read-only` when you need write access?
3. Are you using the correct tools for analysis tasks?

### Agent Attempting Forbidden Operations

If the agent tries to use write tools in read-only mode:
1. The constraint manager will automatically reject the tool call
2. Check the system prompt is being applied correctly
3. Verify the mode configuration in your YAML file

## Migration Guide

### Converting Write Tasks to Read-Only

To convert an existing write-mode task to read-only:

1. Add `mode: read-only` to configuration
2. Remove file modification objectives from task description
3. Focus task on analysis and reporting
4. Update `allowed_tools` to only include read operations
5. Adjust output expectations (reports instead of code changes)

Before (write mode):
```yaml
task: Fix the authentication bug in the login handler

constraints:
  allowed_file_patterns: ["**/*.go"]
```

After (read-only mode):
```yaml
mode: read-only

task: Analyze the authentication implementation and identify the root cause of the login bug. Provide a detailed report with fix recommendations.

constraints:
  allowed_tools:
    - read_file
    - search_files
```
