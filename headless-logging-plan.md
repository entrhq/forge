# Headless Mode Logging System - Implementation Plan

## Overview
Create a clean, structured, and beautiful logging system for headless mode that makes execution easy to understand and follow.

## 1. Logger Architecture

### Verbosity Levels
- **Quiet**: Only errors, warnings, and final summary (CI/CD friendly)
- **Normal**: Standard progress indicators (default, user-friendly)
- **Verbose**: Detailed execution information (troubleshooting)
- **Debug**: Everything including internal events (development)

### Design Principles
- Use colors and symbols (✓, ✗, •, →, 🔧, 📝, 🔀) for visual clarity
- Progressive disclosure: show more detail as verbosity increases
- Clear visual hierarchy with sections and separators
- Easy to scan and understand execution flow
- No noise in normal mode, comprehensive in debug mode

## 2. Key Logging Categories

### Execution Phases
- **Startup**: Configuration, workspace validation
- **Agent Execution**: Tool calls, file modifications
- **Quality Gates**: Gate execution, pass/fail status, retries
- **Git Operations**: Branch operations, commits, PR creation
- **Finalization**: Summary, artifacts, final status

### Specific Elements
- **Tool Calls**: 
  - Quiet: None
  - Normal: Compact bullets (• tool_name (#N))
  - Verbose: Detailed with tool name and count (🔧 Tool: tool_name (call #N))
  - Debug: Include input/output details

- **File Modifications**:
  - Normal+: Show path with line changes (📝 Modified: path (+X/-Y))
  
- **Quality Gates**:
  - Normal+: Clear pass/fail with gate name (✓/✗ gate_name: passed/failed)
  - Verbose+: Include error messages for failures
  - Show retry attempts clearly

- **Git Operations**:
  - Normal+: Show operations (🔀 Git: operation)
  - Verbose: Include details (commit hash, branch names)

## 3. Implementation Approach

### Step 1: Create Logger Package
File: `pkg/executor/headless/logger.go`
- Define `Logger` struct with configurable verbosity
- Implement methods for each logging category
- Use `fatih/color` for colored output
- Support io.Writer for testing

### Step 2: Update Executor
File: `pkg/executor/headless/executor.go`
- Add logger field to Executor struct
- Initialize logger based on config verbosity
- Replace all `log.Printf("[Headless] ...")` calls
- Organize logs into clear sections

### Step 3: Update FileTracker
File: `pkg/executor/headless/file_tracker.go`
- Move verbose logs to debug-only
- Use structured logger instead of log.Printf
- Keep only essential logs in normal mode

### Step 4: Update Other Components
- `pkg/executor/headless/git.go`: Use structured logging
- `pkg/executor/headless/pr.go`: Use structured logging
- `pkg/executor/headless/quality_gate.go`: Use structured logging

### Step 5: Add CLI Configuration
File: `cmd/forge-headless/main.go`
- Add `--log-level` flag (quiet|normal|verbose|debug)
- Pass log level to executor config
- Default to "normal"

## 4. Output Examples

### Normal Mode (Default)
```
======================================================================
  FORGE HEADLESS EXECUTION
======================================================================
Task: Fix all linting errors
Mode: write
Workspace: /path/to/project

▶ Initializing
──────────────────────────────────────────────────
  ✓ Workspace validated
  ✓ Git repository ready (branch: feature/fixes)

▶ Agent Execution
──────────────────────────────────────────────────
  • read_file (#1)
  • search_files (#2)
  📝 Modified: src/main.go (+5/-3)
  • execute_command (#3)
  📝 Modified: src/utils.go (+2/-1)

▶ Quality Gates
──────────────────────────────────────────────────
  ✓ lint: passed
  ✓ tests: passed

▶ Git Operations
──────────────────────────────────────────────────
  🔀 Git: Creating commit
  ✓ Commit created: abc123f

======================================================================
  EXECUTION SUMMARY
======================================================================
Status: ✓ SUCCESS
Task: Fix all linting errors
Duration: 45s

📊 Metrics:
  Files modified: 2
  Lines changed: +7/-4
  Tool calls: 5
  Tokens used: 12,450

🔀 Git:
  Commit: abc123f
======================================================================
```

### Quiet Mode (CI/CD)
```
⚠ Warning: Workspace has uncommitted changes
✗ Error: Quality gate 'lint' failed
```

### Verbose Mode
```
======================================================================
  FORGE HEADLESS EXECUTION
======================================================================
Task: Fix all linting errors
Mode: write
Workspace: /path/to/project

▶ Initializing
──────────────────────────────────────────────────
  ✓ Workspace validated
  → Workspace path: /path/to/project
  ✓ Git repository ready
  → Current branch: main
  → Creating branch: feature/fixes
  → Switched to branch: feature/fixes

▶ Agent Execution
──────────────────────────────────────────────────
  🔧 Tool: read_file (call #1)
  → Reading: src/main.go (lines 1-50)
  🔧 Tool: search_files (call #2)
  → Pattern: TODO|FIXME
  📝 Modified: src/main.go (+5/-3)
  🔧 Tool: execute_command (call #3)
  → Command: go fmt ./...
  📝 Modified: src/utils.go (+2/-1)

▶ Quality Gates
──────────────────────────────────────────────────
  ✓ lint: passed
  ✓ tests: passed

▶ Git Operations
──────────────────────────────────────────────────
  🔀 Git: Creating commit
  → Message: "fix: resolve linting errors"
  ✓ Commit created: abc123f

======================================================================
  EXECUTION SUMMARY
======================================================================
Status: ✓ SUCCESS
Task: Fix all linting errors
Duration: 45s

📊 Metrics:
  Files modified: 2
  Lines changed: +7/-4
  Tool calls: 5
  Tokens used: 12,450

📝 Modified Files:
  • src/main.go (+5/-3)
  • src/utils.go (+2/-1)

🎯 Quality Gates:
  ✓ lint
  ✓ tests

🔀 Git:
  Commit: abc123f
======================================================================
```

### Debug Mode
```
(Everything from verbose mode PLUS internal event details)
  [DEBUG] Event received: Type=tool_call
  [DEBUG] Tool call input: map[path:src/main.go start_line:1]
  [DEBUG] FileTracker: Pending modifications: 1
  [DEBUG] Constraint check: files_modified=1, max=10
```

## 5. Benefits

### For Users
- **Easy to scan**: Clear visual hierarchy and sections
- **Professional appearance**: Clean, organized output
- **Flexible verbosity**: Choose detail level for your needs
- **CI/CD friendly**: Quiet mode for automated environments

### For Developers
- **Debugging**: Debug mode shows everything
- **Maintenance**: Structured logging easier to update
- **Testing**: Logger can write to any io.Writer
- **Consistency**: All logging goes through same system

## 6. Configuration

### CLI Flags
```bash
forge-headless --task "Fix bugs" --log-level normal
forge-headless --task "Fix bugs" --log-level quiet  # CI/CD
forge-headless --task "Fix bugs" --log-level verbose # Troubleshooting
forge-headless --task "Fix bugs" --log-level debug   # Development
```

### Config File
```yaml
task: "Fix all linting errors"
log_level: normal  # quiet|normal|verbose|debug
```

## 7. Implementation Checklist

- [x] Create logger.go with Logger struct and methods
- [ ] Add LogLevel to Config struct
- [ ] Update Executor to use Logger
- [ ] Update FileTracker to use Logger
- [ ] Update git.go to use Logger
- [ ] Update pr.go to use Logger  
- [ ] Update quality_gate.go to use Logger
- [ ] Add --log-level CLI flag
- [ ] Add log_level to YAML config
- [ ] Test all verbosity levels
- [ ] Update documentation

## 8. Testing Strategy

- Test each verbosity level produces expected output
- Test color formatting (with/without TTY)
- Test logger with different io.Writers
- Test summary formatting with various metrics
- Ensure backward compatibility with existing configs
