# 0054. TUI Markdown Rendering for Agent Responses

**Status:** Proposed
**Date:** 2025-07-14
**Deciders:** Justin
**Technical Story:** Agent responses contain markdown syntax (headings, code blocks, bullet lists, emphasis) that currently renders as raw text in the Forge TUI, making output hard to read. This ADR defines the technical approach for rendering markdown in the TUI.

---

## Context

### Background

Forge's agent loop produces responses formatted in markdown — a standard output format for LLMs. The Forge TUI today takes that markdown and displays it as plain text via `formatEntry()` → `wordWrap()` in `pkg/executor/tui/helpers.go`. Users see raw `##`, `**`, ` ``` `, and `- ` symbols rather than rendered structure.

The existing codebase already uses:
- `github.com/charmbracelet/lipgloss` for terminal styling
- `github.com/alecthomas/chroma/v2` for syntax highlighting (in `pkg/executor/tui/syntax/syntax.go`)
- `github.com/charmbracelet/bubbletea` and `github.com/charmbracelet/bubbles` for the TUI framework
- A `DisplayMessage.RenderFn func(width int) string` closure pattern for width-aware, re-flowable message rendering

The `charmbracelet/glamour` library (`charm.land/glamour/v2`) is the canonical markdown renderer for the charmbracelet ecosystem. It is used by `charmbracelet/crush` (the reference AI coding TUI), the GitHub CLI, GitLab CLI, and Glow. It builds on goldmark (GFM-compatible parser) and chroma (code highlighting) — both already present in our dependency tree.

### Problem Statement

Agent responses in the TUI render as raw markdown text, degrading readability and making Forge look unpolished compared to every competing AI coding tool (Crush, Codex CLI, Gemini CLI, Aider).

### Goals

- Render finalized agent prose messages with full markdown formatting (headings, bullets, code blocks, emphasis, inline code, links)
- Integrate with the existing `RenderFn` pattern so rendered messages reflow correctly on terminal resize
- Add `charm.land/glamour/v2` as a dependency and wire it into the agent message commit path
- Support dark and light terminal themes; allow `GLAMOUR_STYLE` env var override
- Introduce zero regressions to the streaming preview, tool result rendering, or headless mode

### Non-Goals

- Rendering user messages in markdown (P2, future work)
- Rendering thinking/scratchpad blocks in markdown (P2, future work)
- A config-file `markdown_style` key (P2, future work)
- Modifying the `ToolResultClassifier` / `DisplayTier` pipeline — tool results are out of scope
- Custom glamour style JSON file support (future work)

---

## Decision Drivers

* **Ecosystem fit**: glamour is by charmbracelet, the same org as bubbletea/lipgloss/bubbles — zero integration friction
* **Zero new heavy dependencies**: glamour depends on goldmark and chroma, both already in `go.mod`
* **Width-aware reflow**: glamour's `WithWordWrap(width)` option maps directly onto the existing `RenderFn func(width int) string` pattern
* **Production proven**: used by GitHub CLI, crush, Glow — not an experimental library
* **Correctness on resize**: the `RenderFn` closure approach (already used for thinking blocks) re-renders at the correct width on every `WindowSizeMsg` — glamour fits this pattern naturally
* **Streaming safety**: glamour requires a complete markdown document; the existing message buffering naturally provides complete text at `handleMessageEnd`

---

## Considered Options

### Option 1: glamour (`charm.land/glamour/v2`)

**Description:** Add glamour as a dependency. Create `pkg/executor/tui/markdown/markdown.go` with a `Renderer` type wrapping `glamour.TermRenderer`. Wire it into `handleMessageEnd()` by replacing the `newEntryMsg("", msgText, lipgloss.NewStyle(), "\n\n")` call with one that uses a width-aware `RenderFn` closure calling the glamour renderer.

**Pros:**
- Charmbracelet-native, zero friction with existing lipgloss/bubbletea stack
- Full GFM support (tables, task lists, strikethrough, autolinks)
- Code block syntax highlighting via chroma (consistent with existing diff highlighting)
- GLAMOUR_STYLE env var support out of the box
- Used by crush as the direct reference implementation — we can follow their exact pattern
- `WithWordWrap(width)` maps perfectly onto our `RenderFn func(width int) string` pattern

**Cons:**
- New module dependency (`charm.land/glamour/v2` — different module path from the old `github.com/charmbracelet/glamour`)
- glamour renders a complete document; cannot stream partial markdown (acceptable — we only render at message-end)
- Some glamour styles add left-margin padding that may conflict with existing message indentation

### Option 2: Custom lipgloss-based markdown renderer

**Description:** Write a bespoke markdown renderer using goldmark (already in `go.mod` as a glamour transitive dep) for parsing and lipgloss for styling, without taking glamour as a dep.

**Pros:**
- Full control over every visual detail
- No new direct dependency

**Cons:**
- Significant engineering effort — goldmark + full ANSI style mapper is weeks of work
- Unlikely to match glamour's quality for tables, nested lists, and edge cases
- We would essentially be reimplementing glamour from scratch
- Maintenance burden: every new markdown feature must be hand-coded

### Option 3: go-term-markdown (`github.com/MichaelMure/go-term-markdown`)

**Description:** Use MichaelMure's terminal markdown library as an alternative to glamour.

**Pros:**
- Not dependent on the charmbracelet module path conventions
- Simpler API for one-shot rendering

**Cons:**
- Less actively maintained (last release 2022, fewer stars than glamour)
- Not part of the charmbracelet ecosystem — style integration with lipgloss requires a custom bridge
- Does not use chroma for code highlighting by default
- No precedent in our reference tools (crush, GitHub CLI use glamour)

### Option 4: Render markdown only for code blocks (partial approach)

**Description:** Continue showing prose as plain text; only apply syntax highlighting to detected fenced code blocks.

**Pros:**
- Minimal scope; no new dependencies

**Cons:**
- Leaves headings, bullets, emphasis, and inline code as raw text — most of the problem unsolved
- Inconsistent: code is pretty but prose is still noisy
- Still a regression compared to competitors

---

## Decision

**Chosen Option:** Option 1 — glamour (`charm.land/glamour/v2`)

### Rationale

1. **Direct charmbracelet ecosystem fit**: glamour is designed to work with lipgloss and bubbletea. Our `RenderFn func(width int) string` pattern maps directly onto `glamour.WithWordWrap(width)`. No bridging layer needed.

2. **Zero re-implementation risk**: glamour is production-hardened and used by GitHub CLI and crush. Writing our own renderer (Option 2) would take weeks and produce an inferior result.

3. **Transitive deps already present**: goldmark and chroma are already in our module graph (as glamour and chroma are both used). Adding glamour makes these direct rather than transitive.

4. **Reference implementation available**: crush's `internal/ui/common/markdown.go` is a complete, minimal reference (< 30 lines) that we can follow exactly:
   ```go
   glamour.NewTermRenderer(
       glamour.WithStyles(styleConfig),
       glamour.WithWordWrap(width),
   )
   ```

5. **GLAMOUR_STYLE**: The env var support is already built into glamour and is a standard convention across the charmbracelet ecosystem. Users get customization for free.

---

## Consequences

### Positive

- Agent responses render with full markdown formatting — headings, bullets, code blocks, emphasis
- Code blocks inside responses get Chroma syntax highlighting, consistent with tool result display
- Terminal resize correctly reflows all rendered messages via the existing `RenderFn` pattern
- Users can override style via `GLAMOUR_STYLE` without any config file changes
- The implementation is small (~100 lines across two new files + a few-line change to `events.go`)

### Negative

- New direct Go module dependency: `charm.land/glamour/v2`
- glamour renders the full document at message-end — if an agent response is extremely long (>500KB), there may be a brief render pause (benchmarks show this is not a concern for typical responses)
- glamour's default left-margin indent may need to be zeroed out to match Forge's existing message padding conventions

### Neutral

- Streaming preview continues to show plain text during token delivery — no change to that code path
- Headless mode is unaffected (glamour degrades to `notty` style when terminal detection fails)
- The existing `wordWrap()` function in `helpers.go` is preserved; it continues to be used for non-markdown content (tool results, system messages)

---

## Implementation

### Phase 1: Add dependency and create renderer package (~1 hour)

**1.1 Add glamour to go.mod:**

```bash
go get charm.land/glamour/v2
```

**1.2 Create `pkg/executor/tui/markdown/markdown.go`:**

```go
// Package markdown provides a terminal markdown renderer for the Forge TUI.
// It wraps charmbracelet/glamour to render agent responses with full GFM
// support including headings, code blocks (syntax-highlighted via Chroma),
// bullet lists, and emphasis.
package markdown

import (
    "os"
    "strings"

    "charm.land/glamour/v2"
    "charm.land/glamour/v2/ansi"
)

// defaultStyle is used when GLAMOUR_STYLE is not set and the terminal
// background is not detected as light.
const defaultStyle = "dark"

// lightStyle is used when a light terminal background is detected.
const lightStyle = "light"

// Renderer wraps a glamour TermRenderer with width-awareness.
// A new underlying renderer is created on each Render call so that
// the word-wrap width always matches the current terminal width.
type Renderer struct {
    style string
}

// New returns a Renderer configured with the appropriate style.
// It respects the GLAMOUR_STYLE environment variable; if unset,
// it attempts to detect the terminal background and falls back to dark.
func New() *Renderer {
    style := os.Getenv("GLAMOUR_STYLE")
    if style == "" {
        style = detectStyle()
    }
    return &Renderer{style: style}
}

// Render renders markdown content at the given width.
// Returns the raw content unchanged if rendering fails.
func (r *Renderer) Render(content string, width int) string {
    if width < 20 {
        return content
    }
    renderer, err := glamour.NewTermRenderer(
        glamour.WithStandardStyle(r.style),
        glamour.WithWordWrap(width),
    )
    if err != nil {
        return content
    }
    out, err := renderer.Render(content)
    if err != nil {
        return content
    }
    // glamour appends a trailing newline; trim it so callers control spacing.
    return strings.TrimRight(out, "\n")
}

// RenderFn returns a closure suitable for use as DisplayMessage.RenderFn.
// The closure captures the content and re-renders at the provided width
// on each call, enabling correct reflow on terminal resize.
func (r *Renderer) RenderFn(content string) func(width int) string {
    return func(width int) string {
        return r.Render(content, width)
    }
}

// detectStyle returns "light" if the terminal appears to have a light
// background, "dark" otherwise.
func detectStyle() string {
    // COLORFGBG is set by some terminals: "foreground;background"
    // where background=15 typically means white/light.
    colorfgbg := os.Getenv("COLORFGBG")
    if strings.HasSuffix(colorfgbg, ";15") || strings.HasSuffix(colorfgbg, ";7") {
        return lightStyle
    }
    return defaultStyle
}

// StyleConfig returns the glamour ansi.StyleConfig for the active style,
// useful for passing to glamour.WithStyles() when custom overrides are needed.
func StyleConfig(style string) ansi.StyleConfig {
    // glamour's built-in styles are accessed via WithStandardStyle;
    // this is a placeholder for future custom style support.
    _ = style
    return ansi.StyleConfig{}
}
```

**1.3 Create `pkg/executor/tui/markdown/markdown_test.go`:**

```go
package markdown_test

import (
    "strings"
    "testing"

    "github.com/entrhq/forge/pkg/executor/tui/markdown"
)

func TestRenderer_Render_Heading(t *testing.T) {
    r := markdown.New()
    out := r.Render("## Hello World", 80)
    if strings.Contains(out, "##") {
        t.Errorf("expected heading markers to be stripped, got: %q", out)
    }
    if !strings.Contains(out, "Hello World") {
        t.Errorf("expected heading text to be present, got: %q", out)
    }
}

func TestRenderer_Render_CodeBlock(t *testing.T) {
    r := markdown.New()
    input := "```go\nfunc main() {}\n```"
    out := r.Render(input, 80)
    if strings.Contains(out, "```") {
        t.Errorf("expected code fence markers to be stripped, got: %q", out)
    }
    if !strings.Contains(out, "func main()") {
        t.Errorf("expected code content to be present, got: %q", out)
    }
}

func TestRenderer_Render_FallbackOnNarrowWidth(t *testing.T) {
    r := markdown.New()
    input := "## Hello"
    out := r.Render(input, 10) // below 20-col minimum
    if out != input {
        t.Errorf("expected raw fallback for narrow width, got: %q", out)
    }
}

func TestRenderer_RenderFn_Reflows(t *testing.T) {
    r := markdown.New()
    fn := r.RenderFn("## Title\n\nSome content.")
    out80 := fn(80)
    out40 := fn(40)
    // Both should render (no raw ## visible) but may differ in wrapping
    if strings.Contains(out80, "##") {
        t.Errorf("expected rendered output at width 80, got: %q", out80)
    }
    if strings.Contains(out40, "##") {
        t.Errorf("expected rendered output at width 40, got: %q", out40)
    }
}
```

### Phase 2: Wire into the message commit path (~1 hour)

The key integration point is `handleMessageEnd()` in `pkg/executor/tui/events.go`. Currently:

```go
// Current code (events.go ~line 277)
func (m *model) handleMessageEnd() (model, tea.Cmd) {
    msgText := m.messageBuffer.String()
    m.messageBuffer.Reset()
    entry := newEntryMsg("", msgText, lipgloss.NewStyle(), "\n\n")
    // ...
}
```

**2.1 Update `pkg/executor/tui/events.go`:**

Add the renderer field to the `model` struct and initialize it at startup, then replace the `newEntryMsg` call in `handleMessageEnd`:

```go
// In model struct (model.go or wherever the struct is defined):
type model struct {
    // ... existing fields ...
    mdRenderer *markdown.Renderer
}

// In the model initializer (New() or similar):
mdRenderer: markdown.New(),
```

Replace the `newEntryMsg` call in `handleMessageEnd`:

```go
// Updated handleMessageEnd (events.go)
func (m *model) handleMessageEnd() (model, tea.Cmd) {
    msgText := m.messageBuffer.String()
    m.messageBuffer.Reset()

    // Build a width-aware RenderFn so the message reflows on terminal resize.
    renderFn := m.mdRenderer.RenderFn(msgText)
    entry := newEntryMsgWithRenderFn("", renderFn, "\n\n")
    // ... rest of existing logic unchanged ...
}
```

**2.2 Add `newEntryMsgWithRenderFn` helper to `helpers.go`:**

```go
// newEntryMsgWithRenderFn creates a DisplayMessage using a width-aware render
// function. The renderFn is called on every View() pass, enabling correct
// reflow when the terminal is resized.
func newEntryMsgWithRenderFn(icon string, renderFn func(int) string, suffix string) DisplayMessage {
    return DisplayMessage{
        Icon:     icon,
        RenderFn: renderFn,
        Suffix:   suffix,
    }
}
```

### Phase 3: Model field and initialization (~30 minutes)

**3.1 Locate where `model` is constructed** (likely `pkg/executor/tui/tui.go` or `model.go`). Add `mdRenderer` field initialization:

```go
import "github.com/entrhq/forge/pkg/executor/tui/markdown"

// In New() or initialModel():
mdRenderer: markdown.New(),
```

**3.2 Verify `DisplayMessage.RenderFn` signature matches the existing usage** in `pkg/executor/tui/messages.go`. The closure `func(width int) string` must match exactly — this is already the established pattern used for thinking blocks.

### Phase 4: Testing and verification (~1 hour)

- Run `make test` — verify all existing TUI tests pass
- Run `go test ./pkg/executor/tui/markdown/...` — verify new unit tests pass
- Run `make run` — manually verify:
  - Agent responses with headings render as bold titles
  - Fenced code blocks render with syntax highlighting
  - Bullet lists render as clean `•` bullets
  - Inline code renders with visual distinction
  - Terminal resize reflows rendered messages correctly
  - `GLAMOUR_STYLE=dracula make run` applies the dracula theme

---

## Validation

### Manual Test Cases

1. **Heading rendering**: Ask the agent "explain what a mutex is with a heading and example code". Verify `## Mutex` renders as a bold title without `##` visible.

2. **Code block rendering**: Ask the agent to write a short Go function. Verify the fenced code block renders with syntax highlighting and no backtick fences visible.

3. **Bullet list rendering**: Ask the agent to list three things. Verify bullets render as `•` (or `–`) without leading `-` or `*` visible.

4. **Resize reflow**: Resize the terminal while viewing a rendered message. Verify the message reflows to the new width without artifacts.

5. **GLAMOUR_STYLE override**: Launch with `GLAMOUR_STYLE=dracula`. Verify the dracula color scheme is applied to agent responses.

6. **Plain text passthrough**: Send a message that produces a response with no markdown. Verify it renders correctly as plain prose with no extra whitespace or artifacts.

7. **Streaming preview**: Watch a streaming response. Verify it shows as plain text during streaming, then switches to the rendered version when the message completes.

8. **Headless mode**: Run `forge --headless "say hello"`. Verify headless output is unaffected by the markdown renderer.

### Success Metrics

- All 8 manual test cases pass on iTerm2, tmux, and standard macOS Terminal
- `make test` passes with ≥ 80% coverage on `pkg/executor/tui/markdown`
- No new linter violations (`make lint` clean)
- Zero regressions in existing TUI tests

---

## Related Decisions

- [ADR-0051](0051-tui-visual-redesign.md) — TUI visual redesign (parent feature set)
- [ADR-0035](0035-auto-close-command-overlay.md) — TUI overlay system (reference for ADR style)

---

## References

- [charm.land/glamour/v2](https://github.com/charmbracelet/glamour) — glamour library source
- [charmbracelet/crush — markdown.go](https://github.com/charmbracelet/crush/blob/main/internal/ui/common/markdown.go) — reference implementation
- [charmbracelet/crush — assistant.go](https://github.com/charmbracelet/crush/blob/main/internal/ui/chat/assistant.go) — usage context
- [glamour README](https://github.com/charmbracelet/glamour/blob/master/README.md) — full API documentation
- [PRD: TUI Markdown Rendering](../product/features/tui-markdown-rendering.md) — product requirements
- `pkg/executor/tui/events.go` — `handleMessageEnd()` integration point
- `pkg/executor/tui/syntax/syntax.go` — existing Chroma highlighting infrastructure
- `pkg/executor/tui/messages.go` — `DisplayMessage.RenderFn` pattern

---

## Notes

**On the `charm.land/glamour/v2` module path**: glamour v2 moved from `github.com/charmbracelet/glamour` to `charm.land/glamour/v2` as part of the charmbracelet v2 module migration. The v1 path still exists and is widely used, but v2 is the current release and aligns with the v2 versions of bubbletea, lipgloss, and bubbles that may be adopted in future TUI updates. We use v2 to stay consistent with crush's `go.mod` which already uses `charm.land/glamour/v2 v2.0.0`.

**On streaming**: glamour does not support incremental/streaming rendering — goldmark parses the full AST before emitting output. This is a non-issue for our use case: we already buffer the full message in `m.messageBuffer` before calling `handleMessageEnd()`. The streaming preview path (`handleMessageContent`) is untouched.

**On left-margin padding**: glamour's default `dark` style adds 2-column left margin to body text. This may need to be adjusted (via a custom `ansi.StyleConfig`) if it conflicts with existing message padding. The Implementation phase should verify this visually and zero out the margin if needed.

**Last Updated:** 2025-07-14
