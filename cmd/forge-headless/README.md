# Forge Headless - Autonomous CI/CD Coding Agent

A headless, autonomous coding agent for CI/CD pipelines and automation workflows. Built on the Forge agent framework, it executes coding tasks without human intervention while operating within strict safety constraints and quality gates.

## Features

- 🤖 **Fully Autonomous** - Completes tasks without asking questions or requiring user input
- 🛡️ **Safety Constraints** - File pattern restrictions, token limits, and timeout enforcement
- 🎯 **Quality Gates** - Automated testing, linting, and build verification before completion
- 🔄 **Error Recovery** - Analyzes failures and attempts multiple solution approaches
- 📝 **Git Integration** - Automatic commits and branch creation
- 📊 **Execution Reports** - Comprehensive JSON summaries with metrics and artifacts
- 🔒 **Workspace Security** - All operations restricted to project directory
- ⚡ **Context Management** - Token-efficient execution with automatic summarization

## Use Cases

- **Automated Code Refactoring** - Modernize codebases, update dependencies, apply security patches
- **CI/CD Code Generation** - Generate boilerplate, configs, documentation
- **Automated Bug Fixes** - Fix failing tests, resolve linting errors
- **Code Quality Improvements** - Add tests, improve coverage, enhance documentation
- **Batch Operations** - Consistent changes across multiple files or repositories

## Installation

```bash
# Build from source
cd cmd/forge-headless
go build -o forge-headless

# Or install to GOPATH/bin
go install github.com/entrhq/forge/cmd/forge-headless@latest
```

## Quick Start

### Basic Usage

1. Set your OpenAI API key:
```bash
export OPENAI_API_KEY="your-api-key-here"
```

2. Run a simple task:
```bash
forge-headless -task "Add error handling to all HTTP handlers" -workspace /path/to/project
```

3. Check the execution summary:
```bash
cat execution-summary.json
```

### Using Configuration File

Create a `forge-config.yaml`:

```yaml
task: "Refactor authentication code to use JWT tokens"
workspace_dir: "/path/to/project"
mode: "autonomous"
timeout: 3600  # 1 hour

constraints:
  allowed_file_patterns:
    - "**/*.go"
    - "**/*.md"
  deny_file_patterns:
    - "vendor/**"
    - "**/node_modules/**"
  max_files: 50
  max_tokens: 100000

quality_gates:
  - name: "unit-tests"
    command: "go test ./..."
    required: true
  - name: "linting"
    command: "golangci-lint run"
    required: true

git:
  auto_commit: true
  commit_message: "refactor: migrate to JWT authentication"
  create_branch: true
  branch_name: "feature/jwt-auth"

artifacts:
  output_dir: "artifacts"
  save_logs: true
  save_diffs: true
```

Run with config:
```bash
forge-headless -config forge-config.yaml
```

## Configuration

### Command Line Flags

#### Required
- `-task` - The coding task to execute (required if not in config file)
- `-workspace` - Workspace directory (required if not in config file)

#### LLM Configuration
- `-api-key` - OpenAI API key (or set `OPENAI_API_KEY` env var)
- `-base-url` - OpenAI API base URL (or set `OPENAI_BASE_URL` env var)
- `-model` - LLM model to use (default: `gpt-4o`)

#### Execution Configuration
- `-config` - Path to YAML configuration file
- `-mode` - Execution mode: `autonomous` or `supervised` (default: `autonomous`)
- `-timeout` - Maximum execution time in seconds (default: `1800` = 30 minutes)

#### Output Configuration
- `-output` - Output file for execution summary (default: `execution-summary.json`)

### Configuration File Format

```yaml
# Task definition
task: "Your coding task description"
workspace_dir: "/path/to/workspace"
mode: "autonomous"  # or "supervised"
timeout: 1800  # seconds

# File access constraints
constraints:
  allowed_file_patterns:
    - "**/*.go"
    - "**/*.py"
    - "*.md"
  deny_file_patterns:
    - "vendor/**"
    - "**/node_modules/**"
    - "**/__pycache__/**"
  max_files: 100
  max_tokens: 200000

# Quality verification
quality_gates:
  - name: "tests"
    command: "npm test"
    required: true
    timeout: 300
  - name: "lint"
    command: "npm run lint"
    required: false

# Git operations
git:
  auto_commit: true
  commit_message: "automated: task completion"
  create_branch: true
  branch_name: "forge/automated-changes"

# Artifact generation
artifacts:
  output_dir: "forge-artifacts"
  save_logs: true
  save_diffs: true
  save_test_results: true
```

## Examples

### Example 1: Add Tests to Untested Code

```bash
forge-headless \
  -task "Add unit tests for all exported functions in pkg/utils" \
  -workspace . \
  -timeout 1800
```

Quality gate config:
```yaml
quality_gates:
  - name: "tests"
    command: "go test ./pkg/utils/... -v"
    required: true
  - name: "coverage"
    command: "go test ./pkg/utils/... -cover"
    required: true
```

### Example 2: Fix Linting Errors

```bash
forge-headless \
  -task "Fix all golangci-lint errors" \
  -workspace /path/to/project \
  -config lint-fix.yaml
```

Config file:
```yaml
task: "Fix all linting errors reported by golangci-lint"
workspace_dir: "."
constraints:
  allowed_file_patterns: ["**/*.go"]
  deny_file_patterns: ["vendor/**", "**/*_test.go"]
quality_gates:
  - name: "lint"
    command: "golangci-lint run --timeout 5m"
    required: true
git:
  auto_commit: true
  commit_message: "fix: resolve linting errors"
```

### Example 3: Update Dependencies

```bash
forge-headless \
  -task "Update all Go dependencies to latest compatible versions" \
  -workspace . \
  -timeout 3600
```

Config:
```yaml
quality_gates:
  - name: "mod-tidy"
    command: "go mod tidy"
    required: true
  - name: "tests"
    command: "go test ./..."
    required: true
  - name: "build"
    command: "go build ./..."
    required: true
git:
  auto_commit: true
  commit_message: "deps: update Go dependencies"
  create_branch: true
  branch_name: "deps/go-update"
```

### Example 4: CI/CD Integration

GitHub Actions workflow:

```yaml
name: Automated Code Improvements
on:
  schedule:
    - cron: '0 2 * * 1'  # Weekly on Monday at 2 AM
  workflow_dispatch:

jobs:
  improve-code:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run Forge Headless
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: |
          forge-headless \
            -task "Improve test coverage in packages with <80% coverage" \
            -workspace . \
            -config .forge-ci.yaml \
            -output summary.json
      
      - name: Create Pull Request
        if: success()
        uses: peter-evans/create-pull-request@v5
        with:
          commit-message: 'test: improve test coverage'
          branch: forge/test-coverage
          title: 'Automated: Improve Test Coverage'
          body-path: summary.json
```

### Example 5: Refactoring Task

```bash
forge-headless -config refactor.yaml
```

Config:
```yaml
task: |
  Refactor the HTTP handlers in cmd/api to use middleware pattern:
  1. Extract authentication logic to middleware
  2. Extract logging to middleware
  3. Update all handlers to use the middleware chain
  4. Ensure all tests still pass

workspace_dir: "."
timeout: 3600

constraints:
  allowed_file_patterns:
    - "cmd/api/**/*.go"
    - "cmd/api/**/*_test.go"
  max_files: 30

quality_gates:
  - name: "tests"
    command: "go test ./cmd/api/... -v"
    required: true
  - name: "build"
    command: "go build ./cmd/api"
    required: true

git:
  auto_commit: true
  commit_message: "refactor: migrate handlers to middleware pattern"
  create_branch: true
  branch_name: "refactor/handler-middleware"

artifacts:
  output_dir: "refactor-artifacts"
  save_logs: true
  save_diffs: true
```

## Output and Artifacts

### Execution Summary

After completion, a JSON summary is generated:

```json
{
  "task": "Add error handling to HTTP handlers",
  "status": "success",
  "start_time": "2024-01-15T10:30:00Z",
  "end_time": "2024-01-15T10:45:00Z",
  "duration": "15m0s",
  "files_modified": [
    "cmd/api/handlers.go",
    "cmd/api/handlers_test.go"
  ],
  "metrics": {
    "files_modified": 2,
    "total_lines_changed": 87,
    "tokens_used": 12450
  },
  "quality_gates": {
    "tests": {
      "passed": true,
      "output": "PASS\nok  \tgithub.com/example/api\t0.234s"
    },
    "lint": {
      "passed": true,
      "output": ""
    }
  },
  "git_commit": "abc123def456",
  "git_branch": "forge/error-handling",
  "errors": null
}
```

### Artifact Directory Structure

```
artifacts/
├── execution-summary.json
├── logs/
│   └── execution.log
├── diffs/
│   ├── cmd_api_handlers.go.diff
│   └── cmd_api_handlers_test.go.diff
└── test-results/
    └── test-output.txt
```

## Execution Modes

### Autonomous Mode (Default)

- No user interaction required
- Agent makes all decisions independently
- Persists through errors with multiple solution attempts
- Uses task_completion when done

### Supervised Mode (Future)

- Requires approval for critical operations
- Interactive prompts for ambiguous situations
- Manual quality gate verification

## Safety and Constraints

### File Access Control

```yaml
constraints:
  # Only modify these patterns
  allowed_file_patterns:
    - "src/**/*.ts"
    - "src/**/*.tsx"
  
  # Never touch these
  deny_file_patterns:
    - "node_modules/**"
    - ".env*"
    - "dist/**"
```

### Resource Limits

```yaml
constraints:
  max_files: 50        # Maximum files that can be modified
  max_tokens: 100000   # Token budget for execution
timeout: 1800          # 30 minute maximum runtime
```

### Quality Gates

Required gates must pass for successful completion:

```yaml
quality_gates:
  - name: "tests"
    command: "npm test"
    required: true     # Task fails if this doesn't pass
    timeout: 300
  
  - name: "lint"
    command: "npm run lint"
    required: false    # Warning only if this fails
```

## Autonomous Behavior

The headless executor is designed to work through problems independently:

### Error Recovery
- Analyzes error messages and stack traces
- Tries multiple solution approaches
- Reads documentation and code to understand context
- Iterates until solution is found or constraints are hit

### Quality Gate Handling
- If tests fail, examines test output
- Makes corrections based on failure messages
- Re-runs tests after fixes
- Continues iterating until all required gates pass

### Resource Management
- Monitors token usage and optimizes operations
- Uses line-range reads for large files
- Batches similar operations
- Stays within configured limits

## Troubleshooting

### "Execution timeout exceeded"

Increase the timeout or break the task into smaller subtasks:
```bash
forge-headless -task "..." -timeout 7200  # 2 hours
```

### "Quality gate 'tests' failed"

Check if tests pass manually. The agent will try to fix them, but some issues may require human intervention:
```bash
# Run tests manually to see failures
go test ./...

# If tests are flaky, make them non-required
quality_gates:
  - name: "tests"
    required: false
```

### "Token limit exceeded"

Reduce scope or increase limit:
```yaml
constraints:
  max_tokens: 200000  # Increase budget
  allowed_file_patterns:
    - "pkg/specific/**"  # Reduce scope
```

### "No files match allowed patterns"

Check your patterns are correct:
```yaml
constraints:
  allowed_file_patterns:
    - "**/*.go"  # Recursive
    - "*.go"     # Root only
```

## Best Practices

### 1. Start Small
Begin with focused, well-defined tasks:
```bash
# Good: Specific and scoped
forge-headless -task "Add input validation to the CreateUser function"

# Too broad: May take too long or go off track
forge-headless -task "Improve the entire codebase"
```

### 2. Use Quality Gates
Always include verification:
```yaml
quality_gates:
  - name: "tests"
    command: "go test ./..."
    required: true
  - name: "build"
    command: "go build ./..."
    required: true
```

### 3. Constrain File Access
Limit scope to relevant files:
```yaml
constraints:
  allowed_file_patterns:
    - "pkg/auth/**/*.go"  # Only auth package
  deny_file_patterns:
    - "**/*_test.go"      # Don't modify tests
```

### 4. Enable Git Integration
Track changes automatically:
```yaml
git:
  auto_commit: true
  create_branch: true
  branch_name: "forge/automated-task"
```

### 5. Monitor Resources
Set appropriate limits:
```yaml
timeout: 1800
constraints:
  max_tokens: 100000
  max_files: 50
```

## Architecture

Forge Headless is built on the same core framework as the interactive Forge TUI:

- **Agent Core** - Conversation loop and tool orchestration
- **Context Manager** - Token-efficient execution with summarization
- **Coding Tools** - File operations, search, diff, command execution
- **Security Layer** - Workspace boundary enforcement
- **Headless Executor** - Autonomous execution with constraints and quality gates

Key differences from interactive Forge:
- Disabled interactive tools (ask_question, converse)
- Auto-approval for all tool calls
- Constraint enforcement (patterns, tokens, timeout)
- Quality gate integration
- Autonomous problem-solving prompts

## Development Status

- ✅ Core executor framework
- ✅ Constraint management
- ✅ Quality gates
- ✅ Git integration
- ✅ Artifact generation
- ✅ Context management
- ✅ Autonomous prompting
- 🚧 Pattern matching validation
- 📅 Supervised mode
- 📅 Advanced metrics

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## License

Apache 2.0 - See [LICENSE](../../LICENSE) for details.

---

**Version:** 0.1.0  
**Status:** 🚧 Under Active Development
