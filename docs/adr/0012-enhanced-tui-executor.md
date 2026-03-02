# 12. Enhanced TUI Executor with Diff Viewer

**Status:** Implemented
**Date:** 2025-01-05
**Deciders:** Forge Core Team
**Technical Story:** Extending the TUI executor to support coding workflows with diff preview, file navigation, and command output display

---

## Context

The existing Forge TUI executor provides a clean Gemini-style chat interface. For the coding agent, we need to enhance it with:
- Side-by-side diff viewer for file changes
- File tree navigation panel
- Command output display area
- Accept/reject controls integrated with tool approval mechanism

### Background

Current TUI executor ([`pkg/executor/tui/executor.go`](../../../pkg/executor/tui/executor.go)) uses Bubble Tea framework and provides:
- Viewport for conversation history
- Text area for user input
- Event-driven updates
- Salmon pink color scheme

The coding agent needs to show rich context beyond conversation text, particularly diffs and file navigation.

### Problem Statement

How do we extend the TUI to:
1. Display side-by-side diffs with syntax highlighting
2. Show file tree for workspace navigation
3. Display command output separately from conversation
4. Integrate approve/reject controls for tool calls
5. Maintain the clean, intuitive UX of the current TUI

### Goals

- Add side-by-side diff viewer component with syntax highlighting
- Add collapsible file tree panel
- Add command output display area
- Integrate with tool approval mechanism (ADR-0010)
- Support keyboard navigation for all new components
- Maintain existing chat-first interface as primary interaction
- Keep the Gemini-inspired aesthetic

### Non-Goals

- Full IDE functionality (debugging, IntelliSense, etc.)
- Git integration UI (deferred)
- Multi-pane layout customization (future work)
- Mouse-based interaction (keyboard-first for now)

---

## Decision Drivers

* **User Experience:** Must feel natural for coding workflows
* **Visual Clarity:** Diffs must be easy to read and understand
* **Integration:** Must work seamlessly with approval flow
* **Performance:** Syntax highlighting shouldn't lag
* **Maintainability:** Keep TUI code organized and testable
* **Bubble Tea Patterns:** Follow framework best practices

---

## Considered Options

### Option 1: Separate Views per Context

**Description:** Create distinct view modes (chat, diff, file tree, command) that user switches between.

**Pros:**
- Simple mental model
- Each view gets full screen space
- Easy to implement

**Cons:**
- Context switching overhead
- Can't see conversation while reviewing diff
- Breaks conversational flow
- Poor for rapid iterations

### Option 2: Multi-Pane Layout

**Description:** Split screen into fixed regions: file tree (left), chat (center), diff/command (right).

**Pros:**
- All context visible simultaneously
- IDE-like familiar layout
- No view switching needed

**Cons:**
- Reduced space per component
- Fixed layout may not suit all workflows
- Complex responsive behavior
- Harder to implement and test

### Option 3: Dynamic Overlay Components

**Description:** Keep chat-first interface, overlay diff/tree as modal components when needed.

**Pros:**
- Chat remains primary interface
- Overlays get full attention when shown
- Flexible per-workflow
- Maintains conversational flow
- Easy to dismiss and return to chat

**Cons:**
- Overlays hide conversation temporarily
- Must manage overlay state carefully
- Potential for overlay stack complexity

---

## Decision

**Chosen Option:** Option 3 - Dynamic Overlay Components

### Rationale

Coding agents are fundamentally conversational. The user describes what they want, the agent proposes changes, the user reviews and approves. This flow is:

1. **Chat-centric:** User expresses intent
2. **Context overlay:** Agent shows diff/preview
3. **Approval:** User reviews and approves/rejects
4. **Back to chat:** Conversation continues

Overlays preserve this natural flow while providing rich context when needed. The existing TUI architecture makes overlays straightforward to implement with Bubble Tea's model update pattern.

---

## Consequences

### Positive

- Maintains simple, chat-first UX
- Diffs get full screen attention during review
- Easy to add more overlay types in future
- Clean separation between conversation and previews
- Keyboard-driven workflow remains natural

### Negative

- Can't see conversation while reviewing overlay
- Overlay state adds complexity to model
- Must design good keyboard shortcuts for overlay navigation
- Potential for deep overlay stacking if not careful

### Neutral

- TUI model grows to manage overlay state
- Need clear visual indicators for overlay mode
- Must document keyboard shortcuts clearly

---

## Implementation

### Component Architecture

```
TUI Model
├── Conversation View (default)
│   ├── Viewport (chat history)
│   └── TextArea (user input)
├── Diff Overlay (when approval requested)
│   ├── Left Pane (original)
│   ├── Right Pane (modified)
│   ├── Syntax Highlighter
│   └── Accept/Reject Controls
├── File Tree Overlay (on demand)
│   ├── Directory Tree
│   ├── File Status Icons
│   └── Navigation Controls
└── Command Output Overlay (during execution)
    ├── Output Buffer
    ├── Scroll Controls
    └── ANSI Color Support
```

### Model State

```go
type model struct {
    // Existing fields
    viewport       viewport.Model
    textarea       textarea.Model
    content        *strings.Builder
    
    // New overlay state
    overlayMode    OverlayMode  // None, Diff, FileTree, CommandOutput
    diffViewer     *DiffViewer
    fileTree       *FileTree
    commandOutput  *CommandOutput
    pendingApproval *types.ToolCall
}

type OverlayMode int

const (
    OverlayNone OverlayMode = iota
    OverlayDiff
    OverlayFileTree
    OverlayCommandOutput
)
```

### Diff Viewer Component

```go
type DiffViewer struct {
    leftPane    viewport.Model  // Original content
    rightPane   viewport.Model  // Modified content
    diffLines   []DiffLine
    cursor      int
    acceptKeys  []key.Binding
    rejectKeys  []key.Binding
}

type DiffLine struct {
    Left      string
    Right     string
    Type      DiffType  // Unchanged, Added, Removed, Modified
    LineNumL  int
    LineNumR  int
}
```

**Features:**
- Syntax highlighting using [chroma](https://github.com/alecthomas/chroma)
- Side-by-side panes with synchronized scrolling
- Color-coded diff types (green=added, red=removed, yellow=modified)
- Line numbers on both sides
- Accept (Ctrl+A) / Reject (Ctrl+R) shortcuts
- Escape to cancel and return to chat

### File Tree Component

```go
type FileTree struct {
    tree        *TreeNode
    viewport    viewport.Model
    cursor      int
    expanded    map[string]bool
}

type TreeNode struct {
    Name      string
    Path      string
    IsDir     bool
    Children  []*TreeNode
    Modified  bool  // For showing git status eventually
}
```

**Features:**
- Collapsible directory tree
- Keyboard navigation (j/k, Enter to expand/collapse)
- Shows current workspace structure
- Quick file opening (Select to insert path in chat)

### Command Output Component

```go
type CommandOutput struct {
    viewport   viewport.Model
    buffer     *strings.Builder
    command    string
    running    bool
    exitCode   int
}
```

**Features:**
- Real-time command output streaming
- ANSI color support via [lipgloss](https://github.com/charmbracelet/lipgloss)
- Scrollable output history
- Shows command and exit code
- Auto-shows on ExecuteCommand approval

### Event Integration

Tool approval events trigger overlays:

```go
case *types.AgentEvent:
    switch event.Type {
    case types.EventTypeToolApprovalRequest:
        // Show appropriate overlay based on tool
        switch event.ToolName {
        case "apply_diff", "write_file":
            m.showDiffOverlay(event)
        case "execute_command":
            m.showCommandApproval(event)
        }
    case types.EventTypeToolApprovalResponse:
        m.closeOverlay()
        m.returnToChat()
    }
```

### Keyboard Shortcuts

**Conversation Mode:**
- `Enter` - Send message
- `Ctrl+C` / `Esc` - Quit
- `Ctrl+T` - Toggle file tree overlay
- `Ctrl+O` - Toggle command output overlay

**Diff Overlay:**
- `j/k` or `↓/↑` - Navigate diff lines
- `Ctrl+A` - Accept changes
- `Ctrl+R` - Reject changes
- `Esc` - Cancel (same as reject)

**File Tree Overlay:**
- `j/k` or `↓/↑` - Navigate files
- `Enter` - Expand/collapse or select file
- `Esc` - Close overlay

**Command Output:**
- `j/k` or `↓/↑` - Scroll output
- `Esc` - Close overlay

### Syntax Highlighting

Use [chroma](https://github.com/alecthomas/chroma) for syntax highlighting:

```go
import (
    "github.com/alecthomas/chroma/v2"
    "github.com/alecthomas/chroma/v2/formatters"
    "github.com/alecthomas/chroma/v2/lexers"
    "github.com/alecthomas/chroma/v2/styles"
)

func highlightCode(code string, language string) string {
    lexer := lexers.Get(language)
    if lexer == nil {
        lexer = lexers.Fallback
    }
    formatter := formatters.Get("terminal256")
    style := styles.Get("monokai")
    
    iterator, _ := lexer.Tokenise(nil, code)
    var buf strings.Builder
    formatter.Format(&buf, style, iterator)
    return buf.String()
}
```

### Migration Path

1. **Week 1:** Create overlay infrastructure in TUI model
2. **Week 2:** Implement DiffViewer component with syntax highlighting
3. **Week 3:** Implement FileTree and CommandOutput components
4. **Week 4:** Integrate with approval flow, add keyboard shortcuts
5. **Week 5:** Testing, refinement, documentation

### Testing Strategy

- Unit tests for each component (DiffViewer, FileTree, CommandOutput)
- Integration tests for overlay transitions
- Manual testing with real coding workflows
- Test different file types and diff scenarios
- Test with various terminal sizes

---

## Validation

### Success Metrics

- Diffs are readable and accurate
- Syntax highlighting works for common languages (Go, Python, JS, etc.)
- Accept/reject flow feels smooth and quick
- File tree loads workspace in <100ms
- Command output streams in real-time
- No UI lag or flickering during overlays

### Monitoring

- Track overlay usage patterns
- Monitor syntax highlighting performance
- Collect user feedback on UX
- Measure time from diff shown to approval decision

---

## Related Decisions

- [ADR-0010](0010-tool-approval-mechanism.md) - Tool approval mechanism
- [ADR-0011](0011-coding-tools-architecture.md) - Coding tools
- [ADR-0009](0009-tui-executor-design.md) - Original TUI design

---

## References

- [Bubble Tea documentation](https://github.com/charmbracelet/bubbletea)
- [Chroma syntax highlighting](https://github.com/alecthomas/chroma)
- [Lipgloss styling](https://github.com/charmbracelet/lipgloss)
- [Claude Code diff interface](https://www.anthropic.com/news/claude-code)
- [Aider diff display](https://aider.chat/)

---

## Notes

The overlay approach keeps the TUI simple while adding power when needed. Future enhancements could include:

- Customizable overlay sizes
- Persistent overlay preferences
- Split-screen mode as option
- Minimap for long diffs
- Regex search within overlays

**Key principle:** Chat remains the primary interface. Overlays are temporary, focused views that appear when needed and disappear when done.

**Last Updated:** 2025-01-05