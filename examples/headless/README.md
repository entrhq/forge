# Forge Headless Mode Examples

This directory contains example configurations and use cases for Forge headless mode.

## What is Headless Mode?

Headless mode enables Forge to run completely autonomously in non-interactive environments such as:
- CI/CD pipelines (GitHub Actions, GitLab CI, Jenkins)
- Cron jobs for scheduled maintenance
- Webhook handlers for automated responses
- Batch processing tasks

## Quick Start

### Basic Usage

Run with an inline task:

```bash
forge-headless -task "Fix all linting errors in Go files"
```

Run with a configuration file:

```bash
forge-headless -config forge-headless.yaml
```

### Configuration File

The `forge-headless.yaml` file demonstrates a complete configuration:

```yaml
task: "Fix all linting errors"
mode: write

constraints:
  max_files: 10
  max_lines_changed: 500
  allowed_patterns:
    - "**/*.go"
  timeout: 5m

quality_gates:
  - name: "Go Lint"
    command: "golangci-lint run ./..."
    required: true

git:
  auto_commit: true
  commit_message: "chore: automated fixes"
```

## Use Cases

### 1. Automated Linting Fixes

Fix linting errors automatically in CI:

```yaml
task: "Fix all golangci-lint errors"
mode: write

constraints:
  allowed_patterns: ["**/*.go"]
  max_files: 20

quality_gates:
  - name: "Lint Check"
    command: "golangci-lint run ./..."
    required: true
```

### 2. Documentation Updates

Keep documentation in sync with code:

```yaml
task: "Update README.md to reflect latest API changes"
mode: write

constraints:
  allowed_patterns: ["**/*.md"]
  max_files: 5

quality_gates:
  - name: "Markdown Lint"
    command: "markdownlint ."
    required: true
```

### 3. Test Generation

Generate missing test cases:

```yaml
task: "Generate unit tests for all exported functions without tests"
mode: write

constraints:
  allowed_patterns: ["**/*_test.go"]
  max_files: 10

quality_gates:
  - name: "Run Tests"
    command: "go test ./..."
    required: true
  - name: "Coverage Check"
    command: "go test -cover ./..."
    required: true
```

### 4. Code Analysis (Read-Only)

Analyze code without making changes:

```yaml
task: "Analyze code complexity and suggest improvements"
mode: read-only

artifacts:
  enabled: true
  output_dir: ".forge/analysis"
```

## GitHub Actions Integration

Example workflow file (`.github/workflows/forge-autofix.yml`):

```yaml
name: Forge Auto-Fix

on:
  schedule:
    - cron: '0 2 * * *'  # Run at 2 AM daily
  workflow_dispatch:  # Manual trigger

jobs:
  autofix:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install golangci-lint
        run: |
          curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
      
      - name: Run Forge Headless
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: |
          forge-headless -config .forge/headless-config.yaml
      
      - name: Create Pull Request
        uses: peter-evans/create-pull-request@v5
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          commit-message: "chore: automated fixes via Forge"
          title: "Automated code improvements"
          body: "Auto-generated changes from Forge headless mode"
          branch: forge-autofix
```

## GitLab CI Integration

Example `.gitlab-ci.yml`:

```yaml
forge-autofix:
  stage: maintenance
  script:
    - forge-headless -config forge-headless.yaml
  rules:
    - if: '$CI_PIPELINE_SOURCE == "schedule"'
  artifacts:
    paths:
      - .forge/artifacts/
    expire_in: 1 week
```

## Safety Features

### Constraints

Headless mode enforces safety constraints to prevent runaway execution:

- **File limits**: Maximum number of files that can be modified
- **Line limits**: Maximum total lines changed
- **Pattern matching**: Allowlist/denylist for file patterns
- **Tool restrictions**: Only approved tools can be used
- **Token limits**: Maximum LLM token usage
- **Timeouts**: Execution time limits

### Quality Gates

Quality gates validate changes before committing:

- **Required gates**: Execution fails if these don't pass
- **Optional gates**: Informational only, don't block
- **Automatic rollback**: Changes are reverted on required gate failure

### Git Integration

Safe git operations:

- **Automatic commits**: Changes are committed with proper attribution
- **Branch isolation**: Changes can be isolated on feature branches
- **Rollback support**: Automatic rollback on failures
- **No auto-push (v1.0)**: Manual review before pushing to remote

## Monitoring and Debugging

### Artifacts

Headless mode generates execution artifacts:

```
.forge/artifacts/
├── execution.json   # Full execution details
├── summary.md       # Human-readable summary
└── metrics.json     # Execution metrics
```

### Example Summary

```markdown
# Forge Headless Execution Summary

**Task:** Fix all linting errors
**Status:** success
**Duration:** 2m 34s

## Files Modified

- `pkg/agent/default.go` (+12/-8 lines)
- `pkg/tools/coding/read_file.go` (+5/-3 lines)

## Quality Gates

✅ **Go Lint** (required)
✅ **Go Test** (required)
✅ **Go Build** (required)

## Metrics

- **Files Modified:** 2
- **Total Lines Changed:** 28
- **Tokens Used:** 12,450
```

## Best Practices

### 1. Start Conservative

Begin with tight constraints and relax as needed:

```yaml
constraints:
  max_files: 5
  max_lines_changed: 100
  timeout: 2m
```

### 2. Use Required Quality Gates

Always validate changes before committing:

```yaml
quality_gates:
  - name: "Tests"
    command: "go test ./..."
    required: true
```

### 3. Enable Artifacts

Keep execution history for debugging:

```yaml
artifacts:
  enabled: true
  output_dir: ".forge/artifacts"
```

### 4. Test Locally First

Run headless mode locally before deploying to CI:

```bash
forge-headless -config forge-headless.yaml
```

### 5. Use Read-Only Mode for Analysis

When learning or testing, use read-only mode:

```yaml
mode: read-only
```

## Troubleshooting

### Constraint Violations

If execution fails due to constraint violations, check the artifacts:

```bash
cat .forge/artifacts/execution.json | jq '.error'
```

Common issues:
- Too many files modified → Increase `max_files`
- Too many lines changed → Increase `max_lines_changed`
- File pattern mismatch → Adjust `allowed_patterns`

### Quality Gate Failures

Check the summary for failed gates:

```bash
cat .forge/artifacts/summary.md
```

### Timeout Issues

Increase timeout for complex tasks:

```yaml
constraints:
  timeout: 10m
```

## Advanced Configuration

### Custom Author Attribution

```yaml
git:
  author_name: "Forge Bot"
  author_email: "bot@company.com"
```

### Branch Strategy

Create a feature branch for review:

```yaml
git:
  branch: "forge/automated-fixes"
  auto_commit: true
```

### Pattern Matching

Use glob patterns for fine-grained control:

```yaml
constraints:
  allowed_patterns:
    - "pkg/**/*.go"
    - "internal/**/*.go"
  denied_patterns:
    - "**/*_test.go"  # Don't modify tests
    - "**/vendor/**"
```

## Security Considerations

1. **API Keys**: Never commit API keys to version control
2. **Workspace Isolation**: Run in isolated workspace directories
3. **Tool Restrictions**: Only enable necessary tools
4. **File Patterns**: Use strict allowlists for sensitive projects
5. **Quality Gates**: Always validate critical changes

## See Also

- [ADR-0026: Headless Mode Architecture](../../docs/adr/0026-headless-mode-architecture.md)
- [ADR-0027: Safety Constraint System](../../docs/adr/0027-safety-constraint-system.md)
- [ADR-0028: Quality Gate Architecture](../../docs/adr/0028-quality-gate-architecture.md)
- [ADR-0029: Headless Git Integration](../../docs/adr/0029-headless-git-integration.md)
