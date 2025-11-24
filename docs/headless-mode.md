# Headless Mode Guide

Forge headless mode enables autonomous code execution in non-interactive environments such as CI/CD pipelines, scheduled jobs, and automation workflows.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Safety Constraints](#safety-constraints)
- [Quality Gates](#quality-gates)
- [Git Integration](#git-integration)
- [CI/CD Integration](#cicd-integration)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

Headless mode is designed for:

- **CI/CD Automation**: Automated code improvements in pipelines
- **Scheduled Maintenance**: Regular code cleanup and updates
- **Webhook Handlers**: Automated responses to repository events
- **Batch Processing**: Large-scale code transformations

### Key Features

✅ **Safety Constraints**: Prevent runaway execution with file/line limits  
✅ **Quality Gates**: Validate changes before committing  
✅ **Git Integration**: Automatic commits with proper attribution  
✅ **Artifact Generation**: Detailed execution reports  
✅ **Read-Only Mode**: Safe code analysis without modifications  

## Quick Start

### Installation

```bash
go install github.com/entrhq/forge/cmd/forge-headless@latest
```

### Basic Usage

Run with an inline task:

```bash
export OPENAI_API_KEY=your-api-key
forge-headless -task "Fix all linting errors"
```

Run with a configuration file:

```bash
forge-headless -config forge-headless.yaml
```

### Example Configuration

Create `forge-headless.yaml`:

```yaml
task: "Fix all golangci-lint errors"
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
  - name: "Go Test"
    command: "go test ./..."
    required: true

git:
  auto_commit: true
  commit_message: "chore: automated linting fixes"
  author_name: "Forge AI"
  author_email: "forge@example.com"

artifacts:
  enabled: true
  output_dir: ".forge/artifacts"
```

## Configuration

### Execution Modes

#### Write Mode

Allows code modifications:

```yaml
mode: write
```

#### Read-Only Mode

Analysis only, no modifications:

```yaml
mode: read-only
```

### Task Description

Clear, specific task descriptions yield better results:

```yaml
# Good ✅
task: "Add error handling to all database queries in pkg/db/"

# Bad ❌
task: "Make the code better"
```

### Workspace Directory

Specify the working directory:

```yaml
workspace_dir: "/path/to/project"
```

Or use CLI flag:

```bash
forge-headless -workspace /path/to/project -task "..."
```

## Safety Constraints

Constraints prevent runaway execution and limit scope.

### File Limits

```yaml
constraints:
  # Maximum files that can be modified
  max_files: 10
  
  # Maximum total lines changed
  max_lines_changed: 500
```

### File Patterns

Use glob patterns to control which files can be modified:

```yaml
constraints:
  # Allow only these patterns
  allowed_patterns:
    - "pkg/**/*.go"
    - "internal/**/*.go"
    - "**/*.md"
  
  # Deny these patterns (takes precedence)
  denied_patterns:
    - "**/vendor/**"
    - "**/.git/**"
    - "**/node_modules/**"
```

### Tool Restrictions

Limit which tools the agent can use:

```yaml
constraints:
  allowed_tools:
    - task_completion
    - read_file
    - write_file
    - apply_diff
    - search_files
    - list_files
    # execute_command not included - disabled
```

### Resource Limits

```yaml
constraints:
  # Maximum LLM tokens
  max_tokens: 50000
  
  # Execution timeout
  timeout: 5m
```

## Quality Gates

Quality gates validate changes before they're committed.

### Required Gates

Execution fails if required gates don't pass:

```yaml
quality_gates:
  - name: "Unit Tests"
    command: "go test ./..."
    required: true
```

### Optional Gates

Informational only, don't block execution:

```yaml
quality_gates:
  - name: "Code Coverage"
    command: "go test -cover ./..."
    required: false
```

### Common Quality Gates

#### Go Projects

```yaml
quality_gates:
  - name: "Go Lint"
    command: "golangci-lint run ./..."
    required: true
  
  - name: "Go Test"
    command: "go test -v ./..."
    required: true
  
  - name: "Go Build"
    command: "go build ./..."
    required: true
  
  - name: "Go Format"
    command: "gofmt -l ."
    required: false
```

#### JavaScript/TypeScript Projects

```yaml
quality_gates:
  - name: "ESLint"
    command: "npm run lint"
    required: true
  
  - name: "Prettier"
    command: "npm run format:check"
    required: true
  
  - name: "TypeScript Check"
    command: "npm run type-check"
    required: true
  
  - name: "Tests"
    command: "npm test"
    required: true
```

#### Python Projects

```yaml
quality_gates:
  - name: "Black"
    command: "black --check ."
    required: true
  
  - name: "Flake8"
    command: "flake8 ."
    required: true
  
  - name: "MyPy"
    command: "mypy ."
    required: true
  
  - name: "Pytest"
    command: "pytest"
    required: true
```

### Rollback on Failure

If a required quality gate fails, changes are automatically rolled back:

```bash
[Headless] Quality gates failed
[Headless] Warning: failed to rollback changes: ...
```

## Git Integration

### Automatic Commits

Enable automatic git commits:

```yaml
git:
  auto_commit: true
  commit_message: "chore: automated fixes via Forge"
```

### Custom Attribution

Set git author information:

```yaml
git:
  author_name: "Forge Bot"
  author_email: "forge-bot@company.com"
```

### Dynamic Commit Messages

The commit message can reference the task:

```yaml
git:
  commit_message: "chore: {task}"
```

Or provide a static message:

```yaml
git:
  commit_message: "chore: automated linting fixes"
```

### Branch Strategy

Create changes on a feature branch:

```yaml
git:
  branch: "forge/automated-fixes"
  auto_commit: true
```

**Note**: In v1.0, automatic push is disabled for safety. You must manually push changes.

## CI/CD Integration

### GitHub Actions

Create `.github/workflows/forge-autofix.yml`:

```yaml
name: Forge Auto-Fix

on:
  schedule:
    - cron: '0 2 * * *'  # 2 AM daily
  workflow_dispatch:

jobs:
  autofix:
    runs-on: ubuntu-latest
    
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install Tools
        run: |
          go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
          go install github.com/entrhq/forge/cmd/forge-headless@latest
      
      - name: Run Forge Headless
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: |
          forge-headless -config .forge/headless-config.yaml
      
      - name: Upload Artifacts
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: forge-artifacts
          path: .forge/artifacts/
      
      - name: Create Pull Request
        if: success()
        uses: peter-evans/create-pull-request@v5
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          commit-message: "chore: automated fixes via Forge"
          title: "🤖 Automated code improvements"
          body: |
            Automated changes from Forge headless mode.
            
            See artifacts for execution details.
          branch: forge-autofix
```

### GitLab CI

Create `.gitlab-ci.yml`:

```yaml
forge-autofix:
  stage: maintenance
  image: golang:1.21
  
  before_script:
    - go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    - go install github.com/entrhq/forge/cmd/forge-headless@latest
  
  script:
    - forge-headless -config .forge/headless-config.yaml
  
  artifacts:
    paths:
      - .forge/artifacts/
    expire_in: 1 week
  
  rules:
    - if: '$CI_PIPELINE_SOURCE == "schedule"'
```

### Jenkins

```groovy
pipeline {
    agent any
    
    environment {
        OPENAI_API_KEY = credentials('openai-api-key')
    }
    
    triggers {
        cron('H 2 * * *')
    }
    
    stages {
        stage('Setup') {
            steps {
                sh 'go install github.com/entrhq/forge/cmd/forge-headless@latest'
            }
        }
        
        stage('Run Forge') {
            steps {
                sh 'forge-headless -config forge-headless.yaml'
            }
        }
        
        stage('Archive Artifacts') {
            steps {
                archiveArtifacts artifacts: '.forge/artifacts/**/*'
            }
        }
    }
}
```

## Best Practices

### 1. Start Conservative

Begin with tight constraints:

```yaml
constraints:
  max_files: 5
  max_lines_changed: 100
  timeout: 2m
```

Gradually increase as you gain confidence.

### 2. Use Required Quality Gates

Always validate critical changes:

```yaml
quality_gates:
  - name: "Tests"
    command: "go test ./..."
    required: true
```

### 3. Enable Artifacts

Keep execution history:

```yaml
artifacts:
  enabled: true
  output_dir: ".forge/artifacts"
```

### 4. Test Locally First

Before deploying to CI, test locally:

```bash
forge-headless -config forge-headless.yaml
cat .forge/artifacts/summary.md
```

### 5. Use Specific Tasks

Clear tasks yield better results:

```yaml
# Good ✅
task: "Add nil checks to all pointer dereferences in pkg/handler/"

# Better ✅✅
task: "Add nil checks before dereferencing pointers in pkg/handler/user.go and pkg/handler/auth.go"
```

### 6. Isolate Changes with Branches

Use feature branches for review:

```yaml
git:
  branch: "forge/automated-fixes"
```

### 7. Monitor Resource Usage

Check token usage in artifacts:

```bash
cat .forge/artifacts/metrics.json | jq '.tokens_used'
```

### 8. Use Read-Only Mode for Learning

When experimenting, use read-only mode:

```yaml
mode: read-only
```

## Troubleshooting

### Constraint Violations

**Error**: `constraint violation (file_count): maximum file count exceeded`

**Solution**: Increase `max_files` or narrow task scope:

```yaml
constraints:
  max_files: 20  # Increased from 10
```

### Quality Gate Failures

**Error**: Quality gates failed

**Solution**: Check the summary for details:

```bash
cat .forge/artifacts/summary.md
```

Fix the underlying issues or adjust quality gates.

### Timeout Issues

**Error**: `execution timeout exceeded`

**Solution**: Increase timeout or simplify task:

```yaml
constraints:
  timeout: 10m  # Increased from 5m
```

### File Pattern Mismatches

**Error**: `file does not match allowed patterns`

**Solution**: Adjust allowed patterns:

```yaml
constraints:
  allowed_patterns:
    - "**/*.go"      # All Go files
    - "pkg/**/*"     # Everything in pkg/
```

### Git Commit Failures

**Error**: `failed to create commit`

**Solution**: Ensure git is configured:

```bash
git config user.name "Forge AI"
git config user.email "forge@example.com"
```

Or configure in YAML:

```yaml
git:
  author_name: "Forge AI"
  author_email: "forge@example.com"
```

## Advanced Topics

### Custom Quality Gates

Create custom validation scripts:

```bash
#!/bin/bash
# custom-gate.sh
set -e

# Run custom validation
go test -race ./...
go vet ./...
staticcheck ./...
```

Add to configuration:

```yaml
quality_gates:
  - name: "Custom Validation"
    command: "./scripts/custom-gate.sh"
    required: true
```

### Multiple Configurations

Maintain separate configs for different use cases:

```
.forge/
├── lint-fixes.yaml
├── test-generation.yaml
├── doc-updates.yaml
└── security-audit.yaml
```

Run specific configurations:

```bash
forge-headless -config .forge/lint-fixes.yaml
```

### Artifact Analysis

Parse execution artifacts programmatically:

```bash
# Get files modified
jq '.files_modified[].path' .forge/artifacts/execution.json

# Get total lines changed
jq '.metrics.total_lines_changed' .forge/artifacts/execution.json

# Check if quality gates passed
jq '.quality_gate_results.all_passed' .forge/artifacts/execution.json
```

## Security Considerations

1. **API Keys**: Use secret management (GitHub Secrets, etc.)
2. **Workspace Isolation**: Run in isolated directories
3. **Tool Restrictions**: Only enable necessary tools
4. **File Patterns**: Use strict allowlists
5. **Quality Gates**: Validate all changes
6. **Review Changes**: Always review before merging

## Further Reading

- [ADR-0026: Headless Mode Architecture](../docs/adr/0026-headless-mode-architecture.md)
- [ADR-0027: Safety Constraint System](../docs/adr/0027-safety-constraint-system.md)
- [ADR-0028: Quality Gate Architecture](../docs/adr/0028-quality-gate-architecture.md)
- [ADR-0029: Headless Git Integration](../docs/adr/0029-headless-git-integration.md)
- [Examples](../examples/headless/)
