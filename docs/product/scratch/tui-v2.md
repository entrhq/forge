# TUI v2: World-Class Terminal Interface Redesign

**Status:** Design  
**Scope:** Complete TUI architectural overhaul  
**Preserves:** All core agent functionality, slash commands, approval flows  

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Current State Audit](#current-state-audit)
3. [Competitive Analysis](#competitive-analysis)
4. [Design Principles](#design-principles)
5. [New Architecture](#new-architecture)
6. [Component Design](#component-design)
7. [Implementation Plan](#implementation-plan)
8. [What We Will NOT Do](#what-we-will-not-do)

---

## Executive Summary

The current TUI was built early in the project's life and has accumulated significant technical debt: a 676-line `Update()` function, a raw `*strings.Builder` for all content (preventing re-rendering and search), hardcoded layout constants that break on resize, ANSI-unaware string math causing visual corruption, and no theme system.

Competitors (Crush/charmbracelet, Codex CLI, Gemini CLI) have substantially higher bars: semantic color token systems, 10–15+ themes, structured message lists with caching, responsive compact-mode breakpoints, proper dialog stacks with typed interfaces, animation systems that pause off-screen, and 100+ granular components. Codex CLI's TUI alone spans 56 modular Rust source files.

This document defines the architecture and feature set for Forge's TUI v2 — an implementation that surpasses the current market by combining Crush's clean Go/BubbleTea component model with Codex's structured event store and Gemini CLI's semantic theming, while keeping our core agent/slash-command/approval flows intact.

---

## Current State Audit

### Architecture Overview

The current implementation lives in `pkg/executor/tui/` (~20 files) and follows Bubble Tea's MVU pattern. The core `model` struct has ~40 fields mixing layout state, agent state, UI component state, and content buffer state. Content is stored as a `*strings.Builder` that is concatenated and set on a `viewport.Model` on every update.

```
Header (ASCII art, 6 lines hardcoded)
Tips bar (1 line)  
TopStatus (cwd, 1 line)
Viewport (raw string content)
  [LoadingIndicator overlay]
InputBox (textarea)
BottomBar (token counts)
```

Overlays are managed by an `overlayState` stack in `update.go` with `pushOverlay`/`popOverlay`. The overlay system supports: `ToolResult`, `DiffViewer`, `CommandOutput`, `Approval`, `Notes`, `Context`, `Settings`, `Help`, `Palette`.

### Critical Bugs

#### Bug 1: ANSI-unaware string length in `buildBottomBar()`
**Location:** `pkg/executor/tui/view.go`  
**Symptom:** Bottom status bar padding is visually corrupt — columns don't align correctly.  
**Cause:** The function uses `len()` on lipgloss-styled strings. Lipgloss styles embed ANSI escape sequences (e.g., `\x1b[38;2;255;179;186m`) whose byte length inflates the raw `len()` count. Padding is calculated from this inflated count, producing negative or over-wide padding.  
**Fix:** Replace all `len()` on styled strings with `lipgloss.Width()`.

#### Bug 2: Hardcoded `headerHeight := 10` in `calculateViewportHeight()`
**Location:** `pkg/executor/tui/update.go:319`  
**Symptom:** Viewport height is wrong when terminal is narrow (header wraps), when the tips bar is hidden, or when we change header content.  
**Cause:** `calculateViewportHeight()` hardcodes `headerHeight := 10` regardless of actual rendered header height.  
**Fix:** Measure actual header height via `lipgloss.Height(m.buildHeader())` or track it as model state updated on `tea.WindowSizeMsg`.

#### Bug 3: `handleWindowResize()` drops width/height update when result list is active
**Location:** `pkg/executor/tui/update.go:335-357`  
**Symptom:** After opening a result list overlay and resizing the terminal, the entire layout breaks until the overlay is closed.  
**Cause:** `handleWindowResize()` returns early without setting `m.width`/`m.height` when `resultList.IsActive()` is true, intending to defer the resize. But the deferred path never re-runs the full resize logic.  
**Fix:** Always update `m.width`/`m.height`, then conditionally skip child resizing if that's the intent.

#### Bug 4: Unicode-unsafe `wordWrap()` and `updateTextAreaHeight()`
**Location:** `pkg/executor/tui/helpers.go:107`, `helpers.go:192`  
**Symptom:** Multi-byte Unicode input (CJK, emoji) miscalculates textarea height; word wrap breaks on non-ASCII content.  
**Cause:** Both functions use `len(line)` or `len([]rune(line))` for width calculations and never account for double-width characters (East Asian Width).  
**Fix:** Use `lipgloss.Width()` for display width measurement throughout, or `uniseg.StringWidth()` from `golang.org/x/text/unicode/norm`.

#### Bug 5: Unbounded `m.content` growth
**Location:** `pkg/executor/tui/model.go`  
**Symptom:** Memory usage grows unboundedly in long sessions; no GC pressure on old content.  
**Cause:** `m.content *strings.Builder` accumulates the entire session's rendered output as a string and is never truncated or replaced.  
**Fix:** Switch to a structured message slice; implement a rolling window or virtualized render.

#### Bug 6: `viewport.New(80, 20)` hardcoded initialization
**Location:** `pkg/executor/tui/init.go:30`  
**Symptom:** On first render before `tea.WindowSizeMsg` arrives, viewport has wrong dimensions causing a flash of incorrect layout.  
**Fix:** Initialize viewport to `(0, 0)` or query terminal dimensions at startup via `tea.WindowSize()` (BubbleTea v1.x) before model creation.

#### Bug 7: `lastToolCallID` using wall clock
**Location:** `pkg/executor/tui/events.go:166`  
**Code:** `lastToolCallID = fmt.Sprintf("%d_%s", time.Now().UnixNano(), event.ToolName)`  
**Symptom:** Tool call IDs are not stable or correlated with the agent's actual tool call IDs, making debugging and event correlation impossible.  
**Fix:** Use the actual tool call ID from the agent event.

### Architectural Debt

#### Debt 1: 676-line `Update()` with suppressed linter (`//nolint:gocyclo`)
`update.go` is a single function handling all message types with deeply nested `if`/`switch` blocks. This is the root cause of most bugs — changes in one branch affect others unpredictably. Every message type should be a named handler function.

#### Debt 2: Raw `*strings.Builder` content model
Content is stored as a single concatenated string. This means:
- No per-message re-rendering on terminal resize (everything is static text in the viewport)
- No search/filter capability
- No message-level state (collapsible tool outputs, etc.)
- Append-only — can't update in-progress streaming items without string replacement hacks

The correct model is a `[]Message` slice where each message has a type, content, and cached rendered string invalidated on width change.

#### Debt 3: Inconsistent overlay architecture
`SettingsOverlay` is a standalone 39.5KB struct that doesn't embed `BaseOverlay`. All other overlays use `BaseOverlay` composition. This means settings has its own duplicate scroll/resize/close logic. The `footerRendersViewport=true` pattern creates tight coupling between footer render functions and viewport state. `DiffViewer` has a hardcoded height formula (`viewportHeight + 9`) that breaks if the header/footer line counts change.

#### Debt 4: Dual internal/external message types
Both `toastMsg` and `tuitypes.ToastMsg` exist, requiring boilerplate conversion. Same for `operationCompleteMsg`. This was likely added to allow the agent to send messages to the TUI, but the indirection creates confusion. A single unified event type should be used.

#### Debt 5: No theme system
All colors are hardcoded as lipgloss `Color` constants in two files (`styles.go` and `types/styles.go`). There is no mechanism for users to choose a different theme, no light mode support, no high-contrast mode, and no `NO_COLOR` environment variable respect.

#### Debt 6: No compact/responsive layout
The header always takes 6+ lines. On an 80×24 terminal (common in CI/remote environments), this leaves only ~14 lines for conversation content. There are no breakpoints that switch to a compact layout.

### Missing Feature Matrix vs. Competitors

| Feature | Forge Current | Crush | Codex CLI | Gemini CLI |
|---------|:---:|:---:|:---:|:---:|
| Theme system | ✗ | ✓ | ✓ | ✓ (15 themes) |
| Light/dark mode | ✗ | ✗ | ✓ | ✓ |
| Semantic color tokens | ✗ | ✓ | ✓ | ✓ |
| Compact/responsive layout | ✗ | ✓ | ✓ | ✓ |
| Input history (↑/↓) | ✗ | ✓ | ✓ | ✓ |
| Markdown rendering | ✗ | ✓ | ✓ | ✓ |
| Structured message list | ✗ | ✓ | ✓ | ✓ |
| Message caching | ✗ | ✓ | ✓ | ✓ |
| Session persistence/resume | ✗ | ✓ | ✓ | ✓ |
| Session browser | ✗ | ✓ | ✓ | ✓ |
| File @mention completions | ✗ | ✗ | ✓ | ✗ |
| Multi-line paste detection | ✗ | ✗ | ✗ | ✓ |
| External editor ($EDITOR) | ✗ | ✓ | ✓ | ✓ |
| Alternate buffer mode | ✗ | ✗ | ✗ | ✓ |
| Screen reader layout | ✗ | ✗ | ✗ | ✓ |
| Notifications system | ✗ | ✗ | ✗ | ✓ |
| Shimmer/animation system | ✗ | ✓ | ✓ | ✓ |
| Background shell panes | ✗ | ✗ | ✗ | ✓ |
| Rewind/undo | ✗ | ✗ | ✗ | ✓ |
| Copy mode | ✗ | ✗ | ✗ | ✓ |
| Suggestions display | ✗ | ✗ | ✗ | ✓ |
| Collapsible tool results | ✗ | ✓ | ✓ | ✓ |
| Tool call progress | spinner | ✓ | ✓ | ✓ |
| Multi-agent support | ✗ | ✗ | ✓ | ✗ |
| Proper dialog stack | partial | ✓ | ✓ | ✓ |

---

## Competitive Analysis

### Crush (charmbracelet/crush) — Go/BubbleTea

The most architecturally relevant competitor. It's a Go/BubbleTea codebase, v0.45.1, 2789 commits. Key lessons:

**Component Decomposition:** `internal/ui` is split into 13 subdirectories: `anim/`, `attachments/`, `chat/`, `common/`, `completions/`, `dialog/`, `diffview/`, `image/`, `list/`, `logo/`, `model/`, `styles/`, `util/`. This is the right level of granularity. No file does two things.

**Dialog Interface:** All dialogs implement `Dialog { ID() string; HandleMsg(msg tea.Msg) Action; Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor }`. The `Overlay` struct is a named slice with `OpenDialog`, `CloseDialog(id)`, `BringToFront(id)`. The front dialog is the last element — simple, deterministic.

**List with Lazy Rendering:** `list.List` has dual-offset scroll state (`offsetIdx` + `offsetLine`), `RenderCallback` decoration pattern, `VisibleItemIndices()` for animation optimization. Items are rendered on-demand with cached output.

**Animation System:** `pausedAnimations map[string]struct{}` tracks off-screen animated items. `Animate()` checks `list.VisibleItemIndices()` before propagating animation ticks — only visible items animate. `RestartPausedVisibleAnimations()` called on scroll.

**Semantic Styles:** `Styles` struct has nested semantic sub-structs: `Header`, `Chat.Message`, `Tool`, `LSP`, `Files`, `Section`. Unicode icon constants: `CheckIcon="✓"`, `ToolPending="●"`, `ToolSuccess="✓"`, `ToolError="×"`, `BorderThin="│"`, `BorderThick="▌"`.

**Compact Mode:** `compactModeWidthBreakpoint = 120`, `compactModeHeightBreakpoint = 30`. Falls back to compact layout automatically.

**Prompt History:** `promptHistory struct{ messages []string; index int; draft string }` — preserves current draft when navigating history.

**AGENTS.md Rules:** Never do IO in `Update`. Components are dumb — expose methods, return `tea.Cmd`. Use `github.com/charmbracelet/x/ansi` for all ANSI string manipulation. Use `tea.Batch()` for multiple commands.

### Codex CLI (openai/codex) — Rust/ratatui

56-file TUI. Most architecturally mature event handling:

**ThreadEventStore:** `VecDeque<Event>` with `capacity: 32768`. Deduplicates by event ID. Tracks `user_message_ids: HashSet<String>` to dedup user messages. Separates `EventMsg::SessionConfigured` from the main buffer. This is the pattern for high-volume streaming.

**Dual-Pane Layout:** Committed `HistoryCell` transcript + in-flight `active_cell` for streaming. Transcript overlay toggled via `Ctrl+T`. This is cleaner than our append-to-string approach.

**Per-Widget Files:** `chatwidget.rs`, `bottom_pane.rs`, `diff_render.rs`, `markdown_render.rs`, `pager_overlay.rs`, `shimmer.rs`, `ascii_animation.rs`, `theme_picker.rs`, `terminal_palette.rs`, `file_search.rs`, `slash_command.rs`, `external_editor.rs` — each concern is isolated.

**Style Guide (`styles.md`):** Use terminal semantic colors (`cyan`, `green`, `red`, `magenta`). Avoid custom RGB for primary content — use `dim` for secondary. This is why it looks good on all terminals. Our hardcoded RGB hex colors don't adapt to terminal theme.

**File @Mention System:** `file_search.rs` + `mention_codec.rs` provide full `@filename` autocomplete with fuzzy matching. Pasted content above 10 lines becomes an attachment.

### Gemini CLI (google-gemini/gemini-cli) — TypeScript/React/Ink

The most feature-complete. ~130 TSX components each with a test file. Key lessons for us:

**Semantic Color Interface:** `SemanticColors { text: { primary, secondary, link, accent, response }; background: { primary, message, input, diff: {added, removed} }; border: { default, focused }; ui: { comment, symbol, dark, gradient }; status: { error, success, warning } }`. This is the right abstraction — components reference semantic tokens, themes provide values.

**Layout Architecture:** `DefaultAppLayout` uses flexbox column: `MainContent` (scrollable) → optional `BackgroundShellDisplay` → fixed `{Notifications, Composer, DialogManager, ExitWarning}`. The bottom controls area is always fixed-height; only the content area scrolls.

**Dialog Manager:** `DialogManager` routes to appropriate dialog component based on `uiState`. Dialogs are first-class components, not overlays. This separation of concerns (dialog routing vs. dialog implementation) is cleaner than our overlay stack.

**Streaming Context:** `StreamingContext.Provider` wraps the entire layout tree — streaming state is ambient, not passed as props.

**Screen Reader Layout:** `ScreenReaderAppLayout` is a completely separate rendering path selected by `useIsScreenReaderEnabled()`. Accessibility is a first-class concern.

**Notifications vs. Toasts:** Gemini CLI has a `Notifications` component (persistent informational banners) separate from toasts (ephemeral). We conflate the two.

---

## Design Principles

These principles govern every decision in v2:

### P1: Structured Messages, Not Raw Strings
The entire conversation is a `[]ConversationMessage` slice. Each message has a type, role, content, tool metadata, and a cached rendered string. The cache is invalidated when `m.width` changes. The viewport receives a fully-rendered string built from the visible slice, not a concatenated append buffer.

### P2: One Handler Per Message Type
`Update()` dispatches to named handler functions. It must not exceed ~50 lines. Each handler (e.g., `handleAgentStreamChunk`, `handleToolCallStart`, `handleWindowResize`, `handleKeyMsg`) lives in its own file and is independently testable.

### P3: Unified Overlay Interface
All overlays implement a single Go interface:
```go
type Overlay interface {
    ID() string
    Update(msg tea.Msg) (Overlay, tea.Cmd)
    View(width, height int) string
    ShortHelp() []key.Binding
}
```
The overlay manager holds a `[]Overlay` stack and routes messages to the front. Closing returns `nil`. No callback composition, no dual architectures for settings vs. other overlays.

### P4: Semantic Color Tokens
No hardcoded hex colors in component code. All components reference a `Theme` struct with semantic fields. A `DefaultTheme`, `LightTheme`, and `NoColorTheme` are provided. Theme is passed via a `RenderCtx` struct threaded through all render functions.

### P5: ANSI-Safe String Measurement
`lipgloss.Width()` everywhere. Never `len()` on a string that has been or may have been styled. The custom `wordWrap()` function is replaced with `muesli/reflow` or `charmbracelet/x/ansi`.

### P6: Responsive Layout with Breakpoints
```
Wide mode:   width >= 120
Normal mode: width >= 60
Compact mode: width < 60 OR height < 25
```
In compact mode: no ASCII art header, abbreviated status bar, single-line tool summaries.

### P7: No IO or State Mutations in View/Update Critical Path
Following Crush's AGENTS.md: `View()` is pure. `Update()` only mutates model state, never calls IO. IO always returns a `tea.Cmd`. Heavy work (rendering markdown, syntax highlighting) is done in a `tea.Cmd` and delivered as a `renderedMsg`.

### P8: Input History with Draft Preservation
`inputHistory struct{ entries []string; index int; draft string }`. Up/down arrows navigate. On navigating away from the live input, current text is saved as `draft`. Navigating back to position 0 restores the draft.

### P9: Message Rendering Cache
Each `ConversationMessage` has a `rendered string` and `renderedWidth int`. Render functions only re-execute when `renderedWidth != currentWidth`. This eliminates redundant re-rendering of the full history on every keystroke.

### P10: Test Coverage for All Components
Every overlay, every renderer, every layout calculation must have table-driven tests. We use BubbleTea's `teatest` package for integration-level model tests.

---

## New Architecture

### Package Structure

```
pkg/executor/tui/
├── model.go              # Root model struct (~60 fields → ~25 fields)
├── init.go               # initialModel(), Init()
├── update.go             # Update() dispatch (~50 lines, delegates to handlers)
├── view.go               # View() assembly (~60 lines, delegates to layout)
├── handlers/
│   ├── agent.go          # handleAgentEvent(), handleStreamChunk()
│   ├── input.go          # handleKeyMsg(), handlePaste()
│   ├── window.go         # handleWindowResize()
│   ├── commands.go       # handleSlashCommand(), handleCommandResult()
│   └── toasts.go         # handleToast(), toastTick()
├── layout/
│   ├── layout.go         # LayoutMode enum, calculateLayout()
│   ├── header.go         # buildHeader(), compact + full variants
│   ├── statusbar.go      # buildStatusBar(), buildBottomBar() — ANSI-safe
│   ├── inputbox.go       # buildInputBox()
│   └── viewport.go       # buildViewport(), calculateViewportHeight()
├── messages/
│   ├── types.go          # ConversationMessage, MessageRole, MessageKind
│   ├── renderer.go       # RenderMessage(msg, width, theme) string
│   ├── cache.go          # render cache invalidation logic
│   └── history.go        # InputHistory struct
├── theme/
│   ├── theme.go          # Theme interface + SemanticColors struct
│   ├── default.go        # DefaultTheme (current salmon pink brand)
│   ├── light.go          # LightTheme
│   └── no_color.go       # NoColorTheme (respects NO_COLOR env)
├── overlay/
│   ├── interface.go      # Overlay interface (ID, Update, View, ShortHelp)
│   ├── manager.go        # OverlayManager stack (push/pop/front)
│   ├── approval.go       # Tool approval overlay (refactored)
│   ├── diff.go           # Diff viewer (refactored, no hardcoded heights)
│   ├── command.go        # Command execution overlay (refactored)
│   ├── settings/
│   │   ├── settings.go   # Settings overlay root (decomposed from 39.5KB)
│   │   ├── provider.go   # Provider/model settings section
│   │   ├── display.go    # Display settings section
│   │   └── keybindings.go # Keybindings section
│   ├── help.go           # Help overlay
│   ├── notes.go          # Notes overlay
│   ├── context.go        # Context overlay
│   └── palette.go        # Command palette (with input history)
├── components/
│   ├── spinner.go        # Animated spinner with shimmer
│   ├── toast.go          # Toast notification with auto-dismiss + manual dismiss
│   ├── progress.go       # Tool progress indicator
│   ├── badge.go          # Status badges (tool name, model, tokens)
│   └── markdown.go       # Markdown renderer (via glamour)
└── types/
    ├── events.go         # All TUI-internal event types (unified, no dual system)
    ├── keys.go           # KeyMap struct (all keybindings in one place)
    └── render_ctx.go     # RenderCtx{Theme, Width, Height, LayoutMode}
```

### Simplified Model Struct

The current model has ~40 fields. v2 targets ~20 by grouping related state:

```go
type model struct {
    // Core dimensions
    width, height int
    layout        layout.LayoutMode

    // Input
    textarea     textarea.Model
    inputHistory messages.InputHistory

    // Content
    messages     []messages.ConversationMessage
    streaming    *messages.ConversationMessage // nil when not streaming
    viewport     viewport.Model
    viewportContent string    // cached from messages, invalidated on resize
    
    // Agent state
    agentRunning  bool
    currentTool   string // empty when no tool active
    spinner       spinner.Model

    // Overlays
    overlays overlay.Manager

    // Theme
    theme theme.Theme

    // Toasts
    activeToast *toastState

    // Status
    tokenStats     tokenStats
    summarizing    bool
    workingDir     string
}
```

### Event Flow

```
tea.KeyMsg / tea.WindowSizeMsg / AgentEvent
         ↓
    Update() dispatch
         ↓
  ┌──────────────────────────┐
  │  overlay.Manager.Update() │  (front overlay gets first shot)
  │  returns (overlay, cmd)  │
  └──────────────────────────┘
         ↓ (if not consumed by overlay)
  handler/input.go    → key handling, textarea, history
  handler/agent.go    → streaming, tool calls, completion
  handler/window.go   → resize, layout recalc, cache invalidation
  handler/toasts.go   → toast lifecycle
         ↓
    View() assembly
         ↓
  layout.calculateLayout() → LayoutMode
  buildHeader() | buildCompactHeader()
  buildViewport() ← renderMessages() ← cached renders
  buildInputBox()
  buildStatusBar()
  overlay.Manager.View() overlaid on top
```

### Message Model

```go
type MessageKind int

const (
    KindUserMessage MessageKind = iota
    KindAssistantText
    KindAssistantThinking
    KindToolCall        // tool invocation (collapsible)
    KindToolResult      // tool result (collapsible)
    KindToolError
    KindSystemInfo      // context summaries, warnings, etc.
    KindSeparator
)

type ConversationMessage struct {
    ID        string
    Kind      MessageKind
    Role      string
    Content   string        // raw content
    ToolName  string        // for KindToolCall/KindToolResult
    ToolArgs  string
    Timestamp time.Time
    
    // Render cache
    rendered      string
    renderedWidth int
    collapsed     bool  // for tool calls/results
}
```

`RenderMessage(msg *ConversationMessage, width int, theme theme.Theme) string` checks `msg.renderedWidth == width` and returns `msg.rendered` if valid. Otherwise re-renders and updates the cache. `handleWindowResize` calls `invalidateAllCaches()` which zeroes all `renderedWidth` fields, triggering re-render on next `View()`.

---

## Component Design

### 1. Header / Logo

**Current problem:** ASCII art always 6 lines, hardcoded, no compact variant.

**v2 design:**
- `LayoutModeWide`: Full ASCII "FORGE" logo with version + gradient border (inspired by Gemini CLI's `ThemedGradient`)
- `LayoutModeNormal`: Single-line `⚡ forge v{version}` with current model name
- `LayoutModeCompact`: Nothing — reclaim vertical space entirely
- Header height computed from `lipgloss.Height(rendered)` and stored in model state after each render

### 2. Conversation Viewport

**Current problem:** Raw `*strings.Builder` append, no re-render, no structure.

**v2 design:**
- `[]ConversationMessage` slice with per-message render cache
- Tool calls rendered as collapsible blocks: `▶ read_file(path/to/file)` (collapsed) or expanded to show args/result
- Thinking blocks rendered as collapsed `◌ Thinking...` by default, expandable
- User messages prefixed with `❯` in accent color
- Assistant text rendered with `glamour` markdown (code blocks with syntax highlighting, bold, italic, headers)
- Streaming assistant message has a blinking cursor appended
- `renderMessages(msgs []ConversationMessage, width int, theme Theme) string` returns full viewport content, using cached renders

### 3. Status Bar (Bottom Bar)

**Current problem:** ANSI-unaware `len()` math makes columns misalign.

**v2 design:**
- Left: `[model-name]` badge + `[tokens used/max]` with color gradient (green→yellow→red) based on % used
- Right: `[↑↓ history]` `[/ commands]` `[? help]` key hints — only shown when input is idle
- Uses `lipgloss.Width()` for all width calculations
- In compact mode: single line with just model name + token %

### 4. Input Box

**Current problem:** No input history, no paste detection.

**v2 design:**
- `↑`/`↓` navigates input history (saves draft)
- Multiline paste (>5 lines) shows a confirm banner: `Paste 23 lines as message? [Enter] confirm [Esc] cancel`
- Placeholder text cycles through suggestions when idle and no content
- `Ctrl+E` opens `$EDITOR` (external editor integration) — writes to temp file, watches for close, reads back content
- Max height 8 lines (reduced from 10) — more viewport space
- Dynamic height calculation uses `lipgloss.Width()` for accurate wrap counting

### 5. Overlay System

**Current problem:** Two parallel architectures (BaseOverlay vs. standalone SettingsOverlay), inconsistent close behavior, hardcoded heights.

**v2 design:**

```go
type Overlay interface {
    ID() string
    Update(msg tea.Msg) (Overlay, tea.Cmd)  // nil = close
    View(width, height int) string
    ShortHelp() []key.Binding               // shown in bottom help bar
}
```

`OverlayManager` (replaces `overlayState`):
```go
type Manager struct {
    stack []Overlay
}
func (m *Manager) Push(o Overlay)
func (m *Manager) Pop() 
func (m *Manager) Front() Overlay   // returns top of stack
func (m *Manager) IsActive() bool
func (m *Manager) Update(msg tea.Msg) (tea.Cmd, bool) // bool = consumed
func (m *Manager) View(width, height int) string       // renders front overlay centered
```

All overlay heights are dynamically calculated from content, not hardcoded. The manager centers overlays using `lipgloss.Place()`.

**Settings overlay:** Decomposed from 39.5KB into `settings/` subpackage with separate files per section. Root file delegates to section renderers.

### 6. Command Palette

**Current problem:** 5-item max, 80-char cap, no input history integration.

**v2 design:**
- Dynamically sized: up to 40% of terminal height, minimum 5 items
- Width: min(terminal_width × 0.7, 100)
- Fuzzy matching ranked: exact name prefix > name substring > description match
- Keyboard shortcut hints shown per command: `[/commit]  Commit staged changes  ←Enter`
- Recent commands shown first when no filter text
- Grouped by category: `Agent Commands`, `UI Controls`, `Session`

### 7. Theme System

**Themes provided:**
- `default` — current salmon pink/mint green (brand identity, dark terminal)
- `light` — adapts to light terminal backgrounds
- `no-color` — respects `NO_COLOR` env, all lipgloss, no RGB

**Theme struct:**
```go
type Theme struct {
    // Text
    TextPrimary   lipgloss.Color
    TextSecondary lipgloss.Color  
    TextAccent    lipgloss.Color  // e.g. salmon pink in default
    TextError     lipgloss.Color
    TextSuccess   lipgloss.Color
    TextWarning   lipgloss.Color
    TextMuted     lipgloss.Color

    // Backgrounds
    BgBase        lipgloss.Color
    BgOverlay     lipgloss.Color
    BgInput       lipgloss.Color
    BgMessage     lipgloss.Color

    // Borders
    BorderDefault lipgloss.Color
    BorderFocused lipgloss.Color
    BorderSubtle  lipgloss.Color

    // Semantic
    UserMsgColor    lipgloss.Color
    AssistantColor  lipgloss.Color
    ToolCallColor   lipgloss.Color
    ToolResultColor lipgloss.Color
    ThinkingColor   lipgloss.Color

    // Icons
    IconCheck   string  // "✓"
    IconCross   string  // "✗"  
    IconPending string  // "●"
    IconArrow   string  // "❯"
    IconTool    string  // "⚙"
    IconThink   string  // "◌"
    IconSpinner string  // for fallback
}
```

`RenderCtx{Theme, Width, Height, LayoutMode}` is threaded through all render functions, eliminating global style variables.

### 8. Toast System

**Current problem:** Always 3 seconds, no dismiss, no levels.

**v2 design:**
- `ToastLevel`: `Info`, `Success`, `Warning`, `Error`
- Duration: configurable per toast (default: Info=3s, Success=2s, Warning=5s, Error=8s)
- Dismiss with `Esc` when overlay is not active
- Multiple toasts queue (max 3 visible), stack from bottom
- Positioned above input box, not overlapping content

### 9. Spinner / Loading Indicator

**Current problem:** Static Dot spinner with 50 random messages. No visual hierarchy.

**v2 design:**
- While agent is thinking (no tool active): animated gradient shimmer on a status line `⠿ Thinking...` 
- While tool is executing: `⚙ read_file  path/to/file.go` with spinner on left, tool name + truncated args inline
- Tool completion flash: ✓ in success color for 500ms before clearing
- Implemented as a `components/spinner.go` that takes `SpinnerState{active bool; phase int; toolName string; toolArgs string}`

---

## Implementation Plan

### Phase 1: Foundation — Fix Bugs, Decompose Update (2-3 weeks)

**Goal:** Zero visual bugs, maintainable code. No user-visible feature changes.

1. **Fix all critical bugs:**
   - Replace all `len()` on styled strings with `lipgloss.Width()`
   - Fix `handleWindowResize()` to always update `m.width`/`m.height`
   - Fix `calculateViewportHeight()` to measure actual header height
   - Fix `updateTextAreaHeight()` to use `lipgloss.Width()` for wrap calc
   - Fix `viewport.New(80, 20)` — use `(0, 0)` init

2. **Decompose `Update()`:**
   - Extract each message type to a named handler in `handlers/` package
   - `Update()` becomes a ~50-line dispatch function
   - Remove `//nolint:gocyclo`

3. **Unify message types:**
   - Eliminate dual internal/external `toastMsg`/`operationCompleteMsg` types
   - Single `types/events.go` with all event types

4. **Refactor overlay system:**
   - Define `Overlay` interface in `overlay/interface.go`
   - Implement `OverlayManager` in `overlay/manager.go`  
   - Migrate all overlays to the interface
   - Begin `settings/` subpackage decomposition

5. **Tests:**
   - Table-driven tests for `buildBottomBar()`, `calculateViewportHeight()`, `handleWindowResize()`
   - `teatest`-based integration test for resize → layout consistency

**Exit criteria:** No `//nolint:gocyclo`, all overlay tests pass, no visual layout bugs on terminal resize.

### Phase 2: Structural Upgrade — Message Model + Theme (3-4 weeks)

**Goal:** Structured message list, render caching, semantic theme system.

1. **Implement `messages/` package:**
   - `ConversationMessage` struct with render cache
   - `RenderMessage()` with `glamour` markdown integration
   - `InputHistory` struct with draft preservation
   - Migrate `m.content *strings.Builder` → `m.messages []ConversationMessage`
   - All streaming events update the in-progress message, not appended to a buffer

2. **Implement `theme/` package:**
   - `Theme` struct and `RenderCtx`
   - `DefaultTheme`, `LightTheme`, `NoColorTheme`
   - Thread `RenderCtx` through all `View()` and render functions
   - Remove global style vars from `styles.go` and `types/styles.go`

3. **Responsive layout:**
   - `LayoutMode` enum with breakpoints
   - Compact header variant
   - Compact status bar variant

4. **Input improvements:**
   - `↑`/`↓` input history navigation with draft preservation
   - Multi-line paste detection banner
   - Fix placeholder text

5. **Tests:**
   - Test render cache invalidation on resize
   - Test input history navigation (boundary conditions, draft save/restore)
   - Test theme token application

**Exit criteria:** Markdown renders in conversation, history navigation works, themes switch correctly, resize re-renders all messages correctly.

### Phase 3: World-Class Features (4-5 weeks)

**Goal:** Above-market feature set, polished UX.

1. **Collapsible tool calls:**
   - `▶ tool_name(args)` → `▼ tool_name(args)` toggle with `Enter`/`Space` on selected item
   - Default: collapsed for tool results > 10 lines
   - Thinking blocks: always collapsed by default

2. **Shimmer loading animation:**
   - Gradient shimmer on the spinner line while agent is processing
   - Tool-specific status: shows tool name + truncated args inline

3. **Enhanced command palette:**
   - Dynamic sizing, fuzzy ranking, grouped categories
   - Recent commands history

4. **Settings decomposition:**
   - Split `overlay/settings.go` (39.5KB) into `settings/` subpackage
   - Provider section, display section, keybindings section

5. **External editor (`$EDITOR`):**
   - `Ctrl+E` keybinding in input box
   - Temp file write → watch for close → read back

6. **Toast improvements:**
   - Levels (Info/Success/Warning/Error) with appropriate durations
   - Manual dismiss with `Esc`
   - Toast queue (max 3)

7. **`NO_COLOR` support:**
   - Detect `NO_COLOR` env var
   - Auto-select `NoColorTheme`

8. **In-session theme switcher:**
   - `/theme` slash command opens theme picker overlay
   - Live preview updates conversation render

**Exit criteria:** All Phase 3 features work, settings overlay decomposed, external editor tested, all overlays unit tested.

### Phase 4: Polish + Accessibility (2 weeks)

1. **Scrollbar indicator** in viewport (right edge thin `│` or `▌` showing position)
2. **Session info in header** — current model name, session duration
3. **Keyboard shortcut hints** that adapt to active context (no overlay, command palette, diff viewer, etc.)
4. **Startup flash elimination** — correct initial dimensions before first render
5. **Performance:** Profile render hot path, ensure <5ms frame time for typical session sizes
6. **Documentation:** Update `docs/how-to/` with TUI keyboard shortcuts reference

---

## What We Will NOT Do

These are things competitors have done that we are explicitly **not** building, either because they conflict with Forge's architecture or are out of scope:

1. **Alternate buffer mode** — Gemini CLI has this, but it adds significant complexity around terminal state management. Normal scroll buffer is correct for a coding agent where users want to copy output.

2. **Background shell panes** — Gemini CLI has embedded terminal panes showing background shell processes. Forge manages command execution through its approval flow; embedded shells would bypass that security model.

3. **Session persistence / session browser** — This is a large feature that intersects with the agent's memory and context management. It should be a separate PRD driven by the agent architecture team, not bundled into the TUI redesign.

4. **Rewind / undo** — Requires deep agent integration (rewinding LLM conversation state). Out of scope for TUI layer.

5. **Screen reader layout** — Valuable for accessibility but requires different rendering primitives. This can be addressed in a dedicated accessibility pass after v2 stabilizes.

6. **Multi-agent UI** — Codex has multi-agent thread management. Forge's agent model is single-agent. This would be an agent architecture change, not a TUI concern.

7. **LSP integration** — Crush has LSP-enhanced completions. Out of scope; Forge's tool system handles code intelligence differently.

8. **Ultraviolet canvas rendering** — Crush uses Charm's `ultraviolet` library (canvas/screen primitives) instead of string composition. This is a fundamental rendering model change. We keep lipgloss string composition for v2; ultraviolet can be evaluated for v3.

9. **`charm.land/bubbletea/v2`** — Crush uses Charm's private v2 fork. We stay on the public `charmbracelet/bubbletea` v1.x.

---

## Appendix: File Size Targets

After v2, no single file in `pkg/executor/tui/` should exceed 400 lines. Current worst offenders and their targets:

| File | Current | v2 Target |
|------|---------|-----------|
| `overlay/settings.go` | 39,500 bytes | Split into `settings/` subpackage, 4 files ≤ 300 lines each |
| `slash_commands.go` | 21,800 bytes | Refactor into `handlers/commands.go` + `commands/registry.go` |
| `update.go` | 20,700 bytes | `update.go` (~50 lines dispatch) + `handlers/*.go` |
| `view.go` | ~400 lines | `view.go` (~60 lines) + `layout/*.go` |
| `events.go` | ~350 lines | `handlers/agent.go` + `types/events.go` |

---

*This document is the authoritative design reference for TUI v2. Implementation should track against the phase plan and exit criteria above. ADRs will be written for each major technical decision (message model, theme system, overlay interface) as implementation begins.*
