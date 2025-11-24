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

Forge headless mode is built into the main `forge` binary - no separate installation needed!

```bash
go install github.com/entrhq/forge/cmd/forge@latest
```

### Basic Usage

Run headless mode with a configuration file:

```bash
export OPENAI_API_KEY=your-api-key
forge -headless -headless-config config.yaml
```

Or use the legacy standalone binary (deprecated):

```bash
forge-headless -config config.yaml
```

### Minimal Configuration

Create a `config.yaml`:

```yaml
task: "Fix all linting errors"
workspace_dir: .
```

Everything else will use defaults.

## Configuration

### Configuration File Format

Headless mode uses YAML configuration files:

```yaml
# Task to execute (REQUIRED)
task: "Analyze the codebase and suggest improvements"

# Execution mode (default: write)
# Options: read-only, write
mode: write

# Workspace directory (REQUIRED if not using CLI flag)
workspace_dir: /path/to/workspace

# Enable verbose logging (default: false)
verbose: false

# Output directory for artifacts (default: ./headless-output)
output_dir: ./output

# Safety constraints
constraints:
  # Maximum number of files that can be modified (default: 10)
  max_files: 10
  
  # Maximum number of lines that can be modified per file (default: 500)
  max_lines_per_file: 500
  
  # Maximum total lines that can be modified across all files (default: 2000)
  max_total_lines: 2000
  
  # Maximum execution time (default: 5m)
  # Format: duration string like "5m", "1h", "30s"
  timeout: 5m
  
  # Maximum number of iterations (tool calls) (default: 100)
  max_iterations: 100
  
  # Token usage limit (default: 100000)
  token_limit: 100000

# Quality gates (all optional)
quality_gates:
  # Require tests to pass before committing
  require_tests: true
  
  # Test command to run (default: "go test ./...")
  test_command: "go test ./..."
  
  # Require linting to pass before committing
  require_lint: true
  
  # Lint command to run (default: "golangci-lint run")
  lint_command: "golangci-lint run"
  
  # Require build to succeed before committing
  require_build: true
  
  # Build command to run (default: "go build ./...")
  build_command: "go build ./..."
  
  # Custom validation commands (all must succeed)
  custom_validations:
    - command: "npm run type-check"
      description: "TypeScript type checking"
    - command: "npm run format-check"
      description: "Code formatting validation"

# Git integration (optional)
git:
  # Enable automatic git operations (default: false)
  enabled: true
  
  # Commit changes automatically (default: false)
  auto_commit: true
  
  # Commit message template (optional, uses generated message if not provided)
  commit_message: "feat: {{.Task}}"
  
  # Git author name (optional, uses git config if not provided)
  author_name: "Forge Bot"
  
  # Git author email (optional, uses git config if not provided)
  author_email: "forge@example.com"
  
  # Create a new branch for changes (optional)
  branch: "forge/auto-improvements"
  
  # Push changes automatically (default: false)
  auto_push: false
  
  # Remote to push to (default: "origin")
  remote: "origin"
```

### CLI Overrides

Command-line flags override configuration file values:

```bash
forge -headless -headless-config config.yaml -workspace /custom/path
```

Available flags:
- `-headless`: Enable headless mode
- `-headless-config`: Path to YAML configuration file
- `-workspace`: Override workspace directory
- `-model`: Override LLM model
- `-api-key`: Override API key
- `-base-url`: Override API base URL

### Execution Modes

#### Write Mode (Default)

Full autonomous execution with code modification permissions:

```yaml
mode: write
```

- Can modify files
- Can execute commands
- Subject to safety constraints
- Requires quality gates to pass (if configured)

#### Read-Only Mode

Safe analysis mode without modification permissions:

```yaml
mode: read-only
```

- Cannot modify files
- Cannot execute write operations
- Useful for code analysis, documentation, and audits
- No quality gates required

## Safety Constraints

Safety constraints prevent runaway execution and protect your codebase.

### File Modification Limits

```yaml
constraints:
  max_files: 10              # Max files to modify
  max_lines_per_file: 500    # Max lines per file
  max_total_lines: 2000      # Max total lines
```

When limits are reached:
- Execution stops immediately
- Partial changes are preserved
- Error is logged with details
- Exit code indicates constraint violation

### Resource Limits

```yaml
constraints:
  timeout: 5m           # Maximum execution time
  max_iterations: 100   # Maximum tool calls
  token_limit: 100000   # Maximum tokens used
```

### Working Within Constraints

For large tasks, break them into smaller chunks:

```bash
# Instead of: "Refactor entire codebase"
# Use multiple focused tasks:
forge -headless -headless-config refactor-auth.yaml
forge -headless -headless-config refactor-api.yaml
forge -headless -headless-config refactor-db.yaml
```

## Quality Gates

Quality gates ensure changes meet your standards before committing.

### Built-in Gates

```yaml
quality_gates:
  require_tests: true
  test_command: "go test ./..."
  
  require_lint: true
  lint_command: "golangci-lint run"
  
  require_build: true
  build_command: "go build ./..."
```

### Custom Validations

Add project-specific checks:

```yaml
quality_gates:
  custom_validations:
    - command: "npm run type-check"
      description: "TypeScript type checking"
    
    - command: "npm run security-scan"
      description: "Security vulnerability scan"
    
    - command: "./scripts/validate-migrations.sh"
      description: "Database migration validation"
```

### Gate Behavior

- All gates must pass for changes to be committed
- Gates run in order: tests → lint → build → custom
- First failure stops execution
- Detailed logs show which gate failed and why

### Skipping Gates (Not Recommended)

For non-critical environments only:

```yaml
quality_gates:
  require_tests: false
  require_lint: false
  require_build: false
```

## Git Integration

Automate git operations for seamless CI/CD integration.

### Basic Git Setup

```yaml
git:
  enabled: true
  auto_commit: true
  commit_message: "chore: automated code improvements"
```

### Branch Management

Create a dedicated branch for changes:

```yaml
git:
  enabled: true
  auto_commit: true
  branch: "forge/improvements-{{.Timestamp}}"
  auto_push: true
  remote: "origin"
```

Template variables:
- `{{.Task}}`: The task description
- `{{.Timestamp}}`: Unix timestamp
- `{{.Date}}`: Current date (YYYY-MM-DD)

### Commit Message Templates

Dynamic commit messages:

```yaml
git:
  commit_message: |
    feat: {{.Task}}
    
    Generated by Forge autonomous execution
    
    Files modified: {{.FilesModified}}
    Lines changed: {{.LinesChanged}}
```

### Author Attribution

```yaml
git:
  author_name: "Forge CI Bot"
  author_email: "ci-bot@company.com"
```

### Safety Features

- Git operations only run if quality gates pass
- Changes are staged but not pushed by default
- Failed quality gates trigger automatic rollback
- All git operations are logged

## CI/CD Integration

### GitHub Actions

```yaml
name: Forge Automated Improvements
on:
  schedule:
    - cron: '0 2 * * *'  # Daily at 2 AM
  workflow_dispatch:      # Manual trigger

jobs:
  improve:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Install Forge
        run: go install github.com/entrhq/forge/cmd/forge@latest
      
      - name: Run Forge Headless
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: |
          forge -headless -headless-config .forge/daily-improvements.yaml
      
      - name: Create Pull Request
        uses: peter-evans/create-pull-request@v5
        with:
          commit-message: 'chore: automated improvements'
          branch: forge/auto-improvements
          title: 'Automated Code Improvements'
          body: 'Generated by Forge autonomous execution'
```

### GitLab CI

```yaml
forge_improvements:
  image: golang:1.21
  stage: improve
  script:
    - go install github.com/entrhq/forge/cmd/forge@latest
    - forge -headless -headless-config .forge/config.yaml
  rules:
    - if: '$CI_PIPELINE_SOURCE == "schedule"'
  artifacts:
    paths:
      - headless-output/
    expire_in: 7 days
```

### Jenkins

```groovy
pipeline {
    agent any
    
    triggers {
        cron('H 2 * * *')
    }
    
    environment {
        OPENAI_API_KEY = credentials('openai-api-key')
    }
    
    stages {
        stage('Install Forge') {
            steps {
                sh 'go install github.com/entrhq/forge/cmd/forge@latest'
            }
        }
        
        stage('Run Headless') {
            steps {
                sh 'forge -headless -headless-config .forge/config.yaml'
            }
        }
        
        stage('Archive Artifacts') {
            steps {
                archiveArtifacts artifacts: 'headless-output/**/*'
            }
        }
    }
}
```

## Best Practices

### Task Design

✅ **Good Tasks**
```yaml
task: "Add error handling to all database operations in pkg/db/"
task: "Update deprecated API calls in controllers/"
task: "Add unit tests for authentication functions"
```

❌ **Poor Tasks**
```yaml
task: "Fix everything"
task: "Make it better"
task: "Refactor"  # Too vague
```

### Configuration Management

Store configurations in version control:

```
.forge/
├── daily-improvements.yaml
├── security-updates.yaml
├── test-coverage.yaml
└── documentation.yaml
```

### Monitoring and Alerts

Parse execution artifacts for monitoring:

```bash
# Check if execution succeeded
if [ $? -eq 0 ]; then
  echo "✓ Forge execution succeeded"
else
  echo "✗ Forge execution failed"
  # Send alert
fi

# Extract metrics
cat headless-output/metrics.json | jq '.files_modified'
```

### Incremental Adoption

Start with read-only mode:

```yaml
# Phase 1: Analysis only
mode: read-only
task: "Analyze code quality and suggest improvements"
```

```yaml
# Phase 2: Small changes with approval
mode: write
git:
  auto_commit: false  # Manual review required
```

```yaml
# Phase 3: Fully automated
mode: write
git:
  auto_commit: true
  auto_push: true
```

## Troubleshooting

### Common Issues

#### Constraint Violations

```
Error: constraint violation (max_files): modified 15 files, limit is 10
```

**Solution**: Increase limits or break task into smaller chunks:

```yaml
constraints:
  max_files: 20  # Increase limit
```

Or:

```bash
# Break into smaller tasks
forge -headless -headless-config task1.yaml
forge -headless -headless-config task2.yaml
```

#### Quality Gate Failures

```
Error: quality gate failed: tests
Exit code: 1
```

**Solution**: Check test output in artifacts:

```bash
cat headless-output/quality-gates/tests.log
```

Fix issues and re-run, or temporarily disable gate:

```yaml
quality_gates:
  require_tests: false  # Only for debugging
```

#### Token Limit Exceeded

```
Error: constraint violation (token_limit): used 105000 tokens, limit is 100000
```

**Solution**: Increase limit or reduce task scope:

```yaml
constraints:
  token_limit: 150000
```

#### Git Conflicts

```
Error: failed to commit changes: uncommitted changes exist
```

**Solution**: Ensure clean working directory before running:

```bash
git status  # Check for uncommitted changes
git stash   # Stash if needed
forge -headless -headless-config config.yaml
```

### Debug Mode

Enable verbose logging:

```yaml
verbose: true
```

Or via CLI:

```bash
FORGE_LOG_LEVEL=debug forge -headless -headless-config config.yaml
```

### Artifact Inspection

Check execution artifacts for details:

```bash
# Execution summary
cat headless-output/execution.json | jq .

# Metrics
cat headless-output/metrics.json | jq .

# Agent conversation log
cat headless-output/conversation.json | jq .

# Quality gate results
ls -la headless-output/quality-gates/
```

### Getting Help

1. Check logs in `headless-output/`
2. Review configuration validation errors
3. Test with a simpler task first
4. Open an issue with:
   - Configuration file
   - Error message
   - Execution artifacts

## Advanced Usage

### Multiple Configurations

Run different tasks in sequence:

```bash
for config in .forge/*.yaml; do
  forge -headless -headless-config "$config"
done
```

### Custom Validation Scripts

```yaml
quality_gates:
  custom_validations:
    - command: "./scripts/check-api-compatibility.sh"
      description: "API backward compatibility check"
    
    - command: "docker-compose -f test-compose.yml up --abort-on-container-exit"
      description: "Integration test suite"
```

### Conditional Execution

Use environment variables:

```yaml
task: "{{.TASK_DESCRIPTION}}"
workspace_dir: "{{.WORKSPACE_PATH}}"
```

```bash
export TASK_DESCRIPTION="Update dependencies"
export WORKSPACE_PATH="/path/to/project"
forge -headless -headless-config config.yaml
```

### Artifact Post-Processing

```bash
# Generate reports from artifacts
forge -headless -headless-config analyze.yaml

# Extract insights
jq -r '.summary' headless-output/execution.json > report.txt

# Send to monitoring
curl -X POST https://monitoring.example.com/metrics \
  -d @headless-output/metrics.json
```
