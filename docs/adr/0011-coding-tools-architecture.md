# 11. Coding Tools Architecture and Design

**Status:** Proposed
**Date:** 2025-01-05
**Deciders:** Forge Core Team
**Technical Story:** Building reusable coding tools for the Forge TUI agent and future agent implementations

---

## Context

The Forge TUI coding agent requires a comprehensive set of tools for file operations, code search, and command execution. These tools must be secure, reusable, and provide rich context for user approval flows.

### Background

Forge currently provides basic conversational tools (`task_completion`, `ask_question`, `converse`). To build a coding agent competitive with Claude Code and Cursor, we need specialized tools for:
- File system operations (read, write, list, search)
- Inline code editing (diff-based)
- Terminal command execution

These tools must be designed as reusable components, not tightly coupled to the TUI executor.

### Problem Statement

We need to design a set of coding tools that:
1. Provide comprehensive file system access
2. Support intelligent code editing (search/replace, not full rewrites)
3. Enable safe command execution
4. Enforce workspace security boundaries
5. Work seamlessly across different executor types (TUI, CLI, API)
6. Provide preview capabilities for approval flows

### Goals

- Build 6 core coding tools: read, write, list, search, diff, execute
- Enforce workspace-only file access (no operations outside CWD)
- Support diff-based editing to avoid full file rewrites
- Provide rich previews for approval decisions
- Make tools reusable across any agent configuration
- Maintain clean separation from executor implementations

### Non-Goals

- Git integration (deferred to future release)
- Multi-file atomic operations (future work)
- Advanced code analysis/refactoring (future work)
- IDE integration (future work)

---

## Decision Drivers

* **Security:** Must prevent access outside workspace
* **Reusability:** Tools should work in any agent, not just TUI
* **Developer Experience:** Must be easy to use and compose
* **Performance:** File operations should be efficient
* **Preview Support:** Enable approval flows with rich context
* **Maintainability:** Clean, well-tested code

---

## Considered Options

### Option 1: Tools in examples/

**Description:** Build tools as part of the example application.

**Pros:**
- Quick to implement
- Fewer abstractions needed

**Cons:**
- Not reusable across projects
- Tight coupling to example
- Harder to test independently
- Doesn't showcase framework capabilities

### Option 2: Tools in pkg/tools/coding

**Description:** Build tools as a reusable package that any agent can import.

**Pros:**
- Reusable across all agent implementations
- Clear separation of concerns
- Independently testable
- Demonstrates framework extensibility
- Can be imported by third-party agents

**Cons:**
- More initial structure needed
- Must design for generality

### Option 3: Tools as separate module

**Description:** Create `github.com/entrhq/forge-tools` separate repository.

**Pros:**
- Independent versioning
- Can grow ecosystem of tool packages
- Clear boundary

**Cons:**
- Premature separation
- Complicates development workflow
- Harder for users to get started
- Circular dependency risk

---

## Decision

**Chosen Option:** Option 2 - Tools in pkg/tools/coding

### Rationale

Building tools in `pkg/tools/coding` strikes the right balance:
1. **Reusability:** Anyone can `import "github.com/entrhq/forge/pkg/tools/coding"`
2. **Simplicity:** Single repository, easy to develop and test
3. **Discoverability:** Users finding Forge will find the tools
4. **Framework showcase:** Demonstrates how to build agent tools

We can always extract to separate module later if the tool ecosystem grows significantly.

---

## Consequences

### Positive

- Tools are reusable across all Forge agents
- Clear package structure encourages organization
- Easy to add more coding tools over time
- Third-party developers can reference implementation
- Single repository simplifies development

### Negative

- `pkg/tools/` now has mixed concerns (built-in vs domain-specific)
- Must maintain API stability as tools mature
- Size of main repository grows

### Neutral

- Tools package becomes part of core Forge distribution
- Need to document coding tools separately from core tools

---

## Implementation

### Package Structure

```
pkg/tools/
├── coding/               # Coding-specific tools
│   ├── read_file.go
│   ├── write_file.go
│   ├── list_files.go
│   ├── search_files.go
│   ├── apply_diff.go
│   ├── execute_command.go
│   └── coding_test.go
├── ask_question.go       # Core conversational tools
├── converse.go
├── task_completion.go
└── tool.go              # Tool interface
```

### Security Layer

Create `pkg/security/workspace/` for shared security logic:

```go
type WorkspaceGuard struct {
    workingDir string
}

func (w *WorkspaceGuard) ValidatePath(path string) error
func (w *WorkspaceGuard) ResolvePath(path string) (string, error)
```

All coding tools use WorkspaceGuard for path validation.

### Tool Designs

#### ReadFileTool

```go
type ReadFileTool struct {
    guard *workspace.WorkspaceGuard
}

func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error)
```

**Parameters:**
- `path` (required): File path relative to workspace
- `line_range` (optional): "start-end" for partial reads

**Returns:** Line-numbered file content

#### WriteFileTool

```go
type WriteFileTool struct {
    guard *workspace.WorkspaceGuard
}
```

**Parameters:**
- `path` (required): File path relative to workspace
- `content` (required): Full file content to write

**Returns:** Success message with file info

#### ListFilesTool

```go
type ListFilesTool struct {
    guard *workspace.WorkspaceGuard
}
```

**Parameters:**
- `path` (optional): Directory path (default: ".")
- `recursive` (optional): Boolean for recursive listing
- `pattern` (optional): Glob pattern filter

**Returns:** Formatted file listing

#### SearchFilesTool

```go
type SearchFilesTool struct {
    guard *workspace.WorkspaceGuard
}
```

**Parameters:**
- `pattern` (required): Regex search pattern
- `path` (optional): Directory to search (default: ".")
- `file_pattern` (optional): File glob filter
- `context_lines` (optional): Lines of context (default: 2)

**Returns:** Matches with surrounding context

#### ApplyDiffTool

```go
type ApplyDiffTool struct {
    guard *workspace.WorkspaceGuard
}
```

**Parameters:**
- `path` (required): File to modify
- `search` (required): Exact text to find
- `replace` (required): Replacement text

**Returns:** Diff preview and success message

**Preview:** Implements `Previewable` interface to show diff before applying

#### ExecuteCommandTool

```go
type ExecuteCommandTool struct {
    guard *workspace.WorkspaceGuard
    timeout time.Duration
}
```

**Parameters:**
- `command` (required): Command to execute
- `working_dir` (optional): Relative working directory

**Returns:** Command output (stdout/stderr combined)

**Security:** Always runs in workspace, respects timeout

### Preview Interface

```go
type Previewable interface {
    Tool
    Preview(ctx context.Context, args json.RawMessage) (string, error)
}
```

Tools like `ApplyDiffTool` and `WriteFileTool` implement this to show diffs/changes before execution.

### Migration Path

1. Create `pkg/tools/coding/` package
2. Create `pkg/security/workspace/` package
3. Implement WorkspaceGuard
4. Implement each tool with comprehensive tests
5. Add preview capabilities to relevant tools
6. Document tool schemas and usage

### Timeline

- Week 1: Security layer and ReadFile/WriteFile tools
- Week 2: ListFiles and SearchFiles tools
- Week 3: ApplyDiff tool with preview
- Week 4: ExecuteCommand tool and integration testing

---

## Validation

### Success Metrics

- All 6 tools implemented with >90% test coverage
- Security layer prevents all path traversal attempts
- Tools work identically in TUI, CLI, and test environments
- ApplyDiff successfully edits files without corruption
- Command execution respects timeout and workspace boundaries

### Monitoring

- Track tool usage patterns in production
- Monitor security validation failures
- Measure performance of file operations
- Collect user feedback on tool effectiveness

---

## Related Decisions

- [ADR-0010](0010-tool-approval-mechanism.md) - Tool approval flow
- [ADR-0012](0012-enhanced-tui-executor.md) - Enhanced TUI with diff viewer
- [ADR-0009](0009-tui-executor-design.md) - TUI executor architecture

---

## References

- [Aider diff-based editing](https://aider.chat/)
- [Claude Code file operations](https://www.anthropic.com/news/claude-code)
- [Cursor file system access](https://cursor.sh)
- Go filepath security patterns

---

## Notes

Key design principle: **Tools should be dumb executors**. They don't make policy decisions (what to approve, when to run). They just perform operations safely within their constraints. Policy is handled by the agent loop and executor.

This separation allows:
- Tools to be composed in different contexts
- Executors to implement different approval UX
- Agents to have different tool configurations
- Easy testing of tool logic in isolation

**Last Updated:** 2025-01-05