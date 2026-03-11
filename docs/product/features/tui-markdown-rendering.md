# TUI Markdown Rendering

## Product Vision

Forge's agent responds in markdown. In the TUI today, that markdown arrives as raw text — asterisks, backticks, hash symbols and all — making responses harder to read than a plain-text email. Developers using Forge for real coding work need to read long, structured responses: explanations with headings, code blocks with syntax highlighting, bullet-point plans, and inline emphasis. Right now, all of that structure is noise.

This feature adds first-class markdown rendering to every agent response in the Forge TUI. When the agent completes a message, its content is rendered through a terminal-aware markdown renderer — headers become bold section titles, code blocks get syntax highlighting, bullets become clean lists, and emphasis renders as expected. The terminal becomes as readable as a rendered README.

## Key Value Propositions

- **For daily users**: Agent responses become dramatically easier to read — structure is visible at a glance, code blocks are highlighted, headings separate ideas cleanly
- **For new users**: The TUI's first-impression quality jumps immediately; the output looks polished and intentional rather than broken
- **For all users**: Code snippets inside agent responses get syntax highlighting using the same Chroma-based pipeline already used for diffs — consistency throughout the UI
- **Competitive advantage**: Crush (charmbracelet's own reference AI coding TUI) ships markdown rendering as a first-class feature. Codex CLI and Gemini CLI both render markdown. Forge is behind the baseline — this change closes the gap.

## Target Users & Use Cases

### Primary Personas

- **The Daily Driver**: A developer running Forge in a tmux pane for hours. They read agent explanations, plans, and code review feedback. Today they must mentally parse raw markdown. After this feature: they can read at a glance.
- **The New Evaluator**: A developer trying Forge for the first time. The first agent response contains `**bold**` and ` ```go ``` ` blocks rendered as raw text — it looks broken. After this feature: the response looks as polished as any modern AI chat UI.
- **The Code Reviewer**: A developer asking the agent to review a function. The agent responds with a numbered list of issues, inline code references, and a code block showing the fix. After this feature: that structure is immediately readable in the terminal.

### Core Use Cases

1. **Reading a structured explanation**: The agent explains a refactoring approach with a heading, three bullet points, and a code example. Today: raw asterisks and backticks. After: clean formatted output with a bold title, indented bullets, and a syntax-highlighted block.

2. **Code generation response**: The agent returns a new function implementation inside a fenced code block. Today: plain monospace with no highlighting. After: language-aware syntax highlighting via Chroma, consistent with how diffs are shown elsewhere in the TUI.

3. **Multi-section agent plan**: The agent breaks a task into phases with `## Phase 1`, `## Phase 2` headers and nested bullet points. Today: the structure is invisible — it looks like a wall of text with `##` noise. After: each section is clearly delineated and scannable.

4. **Inline emphasis and inline code**: The agent references a `filepath.Join()` call or says "this is **critical**". Today: backticks and asterisks appear literally. After: inline code is visually distinct; emphasis is bold or italic.

## Product Requirements

### Must Have (P0)

- **Finalized message rendering**: When an agent message is complete (`handleMessageEnd`), its content is rendered through a markdown renderer before being committed to the viewport. Raw markdown is never displayed to the user in finished messages.
- **Code block syntax highlighting**: Fenced code blocks in agent responses are syntax-highlighted using Chroma, consistent with the existing diff/code highlighting infrastructure.
- **Width-aware rendering**: Markdown is rendered at the current terminal width and re-rendered when the terminal is resized, so line wrapping and layout always match the available space.
- **Dark-terminal style**: The default markdown style is tuned for dark terminals (the dominant environment for terminal-based coding tools), with appropriate foreground colors for headings, code, and emphasis.
- **Graceful fallback**: If the markdown renderer encounters an error, the raw message text is displayed unchanged — the TUI never shows a blank or corrupted message.

### Should Have (P1)

- **Streaming preview**: During streaming (before the message is finalized), content is shown as plain text. When the message completes, the viewport content is replaced with the fully-rendered version. No mid-stream flickering of partial markdown.
- **Consistent code block style**: The code block background color and font treatment inside rendered markdown matches the tool-result code display already in the TUI, creating a unified look.
- **Light-terminal support**: When a light terminal background is detected (via `COLORFGBG` or `TERM_BACKGROUND` env var), a light-mode glamour style is applied. Defaults to dark if detection is ambiguous.
- **GLAMOUR_STYLE override**: Users can set `GLAMOUR_STYLE=dracula` (or any built-in style name) to override the default style, matching the standard glamour convention used across the charmbracelet ecosystem.

### Could Have (P2)

- **User message markdown**: Render the user's own messages with the same markdown pipeline, so their inline code, bullets, and emphasis are also formatted.
- **Thinking block rendering**: Apply a stripped-down "plain" markdown renderer to the agent's thinking/scratchpad blocks (same approach as crush's `PlainMarkdownRenderer`) — structure without color.
- **Per-style theme config**: Expose a `markdown_style` key in Forge's config file so users can permanently set their preferred glamour style without an env var.
- **Custom style file**: Allow users to point `GLAMOUR_STYLE` at a local JSON file path containing a custom `ansi.StyleConfig`, enabling full color/layout customization.

## User Experience Flow

### Before (Current State)

```
╭─ assistant ──────────────────────────────────────────────╮
│ ## Analysis                                               │
│                                                           │
│ Here's what I found:                                      │
│                                                           │
│ - **Issue 1**: The `filepath.Join` call on line 42 will   │
│   panic if `parts` is empty.                              │
│ - **Issue 2**: Error is silently swallowed at line 87.    │
│                                                           │
│ ```go                                                      │
│ func safePath(parts ...string) string {                    │
│     if len(parts) == 0 { return "" }                      │
│     return filepath.Join(parts...)                        │
│ }                                                          │
│ ```                                                        │
╰──────────────────────────────────────────────────────────╯
```
Raw `##`, `**`, backticks visible. Code block indistinguishable from prose.

### After (Target State)

```
  Analysis
  ─────────────────────────────────────────────

  Here's what I found:

  • Issue 1: The filepath.Join call on line 42 will
    panic if parts is empty.
  • Issue 2: Error is silently swallowed at line 87.

  ┌──────────────────────────────────────────────┐
  │ func safePath(parts ...string) string {       │  ← syntax highlighted
  │     if len(parts) == 0 { return "" }          │
  │     return filepath.Join(parts...)            │
  │ }                                             │
  └──────────────────────────────────────────────┘
```
Heading renders as bold section title. Bullets are clean. Code block has background and syntax highlighting. Inline code is visually distinct.

### Interaction Model

- **No user action required**: Markdown rendering is always-on. There is no toggle. The TUI silently improves.
- **Resize reflow**: On terminal resize, all rendered messages in the viewport are re-rendered at the new width. The user sees no layout artifacts.
- **Style override**: Power users can set `GLAMOUR_STYLE=dracula` before launching Forge to get their preferred color scheme.
- **Fallback transparency**: If a message fails to render, the raw text appears unchanged with no error surfaced to the user (errors are logged at debug level).

## UI & Interaction Design

### Renderer Configuration

The markdown renderer is configured per-session with:

- **Style**: `dark` by default; `light` if terminal background detection suggests a light theme; overridden by `GLAMOUR_STYLE`
- **Word wrap**: Set to `currentWidth - 4` to leave a small visual margin and account for any message padding
- **Chroma formatter**: `terminal256` (consistent with existing syntax highlighting in `pkg/executor/tui/syntax/syntax.go`)

### Rendering Timing

| Phase | Behavior |
|---|---|
| Streaming (tokens arriving) | Plain text, no markdown processing |
| Message finalized (`handleMessageEnd`) | Full markdown render applied, viewport updated |
| Terminal resize | All existing messages re-rendered at new width via `RenderFn` closure |

### Style Mapping

Built-in glamour styles available to users via `GLAMOUR_STYLE`:
`dark` (default), `light`, `dracula`, `tokyo-night`, `pink`, `ascii`, `notty`, `auto`

## Feature Metrics

### Leading Indicators (measurable at launch)

- All agent messages in the TUI render without raw markdown symbols visible
- Zero regressions: non-markdown agent messages (plain prose) display correctly
- Resize test: rendered messages reflow correctly at all terminal widths ≥ 40 columns
- `GLAMOUR_STYLE` env var correctly overrides style

### Lagging Indicators (qualitative, post-launch)

- Reduction in user reports of "messy output" or "unreadable responses"
- Increase in positive impressions in user feedback referencing TUI quality
- Competitors' markdown rendering no longer cited as a differentiator in head-to-head comparisons

## User Enablement

- **Zero configuration**: Markdown rendering is on by default with no user action
- **Style customization**: Document `GLAMOUR_STYLE` env var in `--help` output and in the README TUI section
- **Changelog entry**: Feature noted in release notes with a before/after screenshot

## Risk & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| glamour adds noticeable render latency on long messages | Low | Medium | Rendering happens at message-end (not during streaming); benchmarks show glamour renders ~100KB in <10ms |
| Rendered output looks wrong on some terminal emulators | Medium | Medium | Test on iTerm2, Kitty, tmux, WezTerm, and basic xterm; implement graceful fallback to raw text |
| Light-terminal detection is unreliable | Medium | Low | Default to dark; document `GLAMOUR_STYLE=light` as the override |
| glamour v2 (`charm.land/glamour/v2`) module path differs from v1 | Low | Low | Use v2 path consistently; pin in go.mod |
| Markdown in user messages looks different from agent messages | Low | Low | User messages are plain text for now; addressed in P2 |
| New dependency introduces security surface | Low | Low | glamour has no network access, no exec, pure text transformation |

## Dependencies

- **New Go dependency**: `charm.land/glamour/v2` — charmbracelet's terminal markdown renderer
- **Existing**: `github.com/alecthomas/chroma/v2` already in `go.mod` (glamour uses it internally too)
- **Existing**: `github.com/charmbracelet/lipgloss` already in `go.mod`
- **No infrastructure changes**: Pure in-process text transformation, no external services

## Constraints

- Must not break the streaming preview experience — raw text during streaming is acceptable and intentional
- Must not change the existing tool-result rendering pipeline (`DisplayTier` / `ToolResultClassifier`) — only agent prose messages are in scope for this feature
- Must work in headless/CI mode (glamour degrades gracefully when `TERM` is unset or `notty` is detected)
- Terminal width must be ≥ 20 columns for rendering to engage; below that, raw text fallback applies

## Competitive Analysis

| Tool | Markdown Rendering | Notes |
|---|---|---|
| **Crush** (charmbracelet) | ✅ Full glamour rendering | Uses `MarkdownRenderer` + `PlainMarkdownRenderer` from `internal/ui/common`; `WithStyles(sty.Markdown)` for theme integration |
| **Gemini CLI** | ✅ React Ink + markdown | Custom markdown component with semantic color abstraction |
| **Codex CLI** | ✅ ratatui markdown | Renders formatted output using ratatui's built-in text styling |
| **Aider** | ✅ Rich library | Python's Rich library for terminal markdown |
| **Forge (current)** | ❌ Raw text | Plain text with `wordWrap()` only |

Forge is the only major AI coding TUI that does not render markdown. This is a clear quality gap.

## Evolution & Roadmap

**This release (v1):**
- P0 items: glamour rendering for all finalized agent messages
- P1 items: consistent code block style, light-terminal support, GLAMOUR_STYLE override

**Next release:**
- P2: User message markdown rendering
- P2: Thinking block plain-markdown rendering
- P2: `markdown_style` config key

**Future:**
- Per-conversation style switching
- Custom glamour style JSON config file support

## Technical References

- ADR-0054: TUI Markdown Rendering — technical design and implementation plan
- `pkg/executor/tui/events.go` — `handleMessageEnd()`, `handleMessageContent()` — integration points
- `pkg/executor/tui/helpers.go` — `formatEntry()`, `wordWrap()` — current text rendering path
- `pkg/executor/tui/messages.go` — `DisplayMessage` with `RenderFn func(width int) string` — reflow pattern
- `pkg/executor/tui/syntax/syntax.go` — existing Chroma-based syntax highlighting
- `charm.land/glamour/v2` — markdown renderer library
- `github.com/charmbracelet/crush` — reference implementation using glamour

## Appendix

### Research & Validation

**Crush reference implementation** (`charmbracelet/crush`):
- `internal/ui/common/markdown.go` — two functions: `MarkdownRenderer(sty, width)` and `PlainMarkdownRenderer(sty, width)`
- Both call `glamour.NewTermRenderer(glamour.WithStyles(...), glamour.WithWordWrap(width))`
- `internal/ui/chat/assistant.go` — calls `renderMarkdown(content, width)` in `renderMessageContent()`
- Render result is trimmed of trailing `\n`; raw content used as fallback on error
- Renderer is re-created on each render call (not cached per-message) — width changes handled naturally

**glamour v2 API surface:**
- `glamour.Render(in, style string) (string, error)` — one-shot with named style
- `glamour.RenderWithEnvironmentConfig(in string) (string, error)` — reads `GLAMOUR_STYLE`
- `glamour.NewTermRenderer(...TermRendererOption) (*TermRenderer, error)` — configurable renderer
- `TermRenderer.Render(in string) (string, error)` — render a complete markdown string
- `glamour.WithStyles(ansi.StyleConfig)` — custom style config
- `glamour.WithStandardStyle(string)` — named built-in style (`dark`, `light`, `dracula`, etc.)
- `glamour.WithWordWrap(int)` — wrap at N columns
- `glamour.WithEnvironmentConfig()` — read `GLAMOUR_STYLE` env var
- `TermRenderer` implements `io.Writer` + `io.Reader` + `Close()` — supports streaming write/read pattern
- glamour uses goldmark (GFM extensions) for parsing + chroma for code highlighting

### Design Artifacts

Before/after terminal screenshots to be added post-implementation.
