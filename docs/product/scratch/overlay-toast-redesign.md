# Overlay & Toast Redesign — Mockups

**Context:** The main TUI chrome was redesigned (ADR-0051) to be compact and flat —
a single-line header bar, `─` rule separator, and a `❯` prompt glyph with no rounded
border on the input box. Overlays and toasts still use the old rounded-border style
which now clashes.

**Goal:** Bring overlays and toasts into the same design language, and fix the existing
problem where overlays and toasts use hardcoded dimensions that break on small terminals
or get stranded in the corner on large ones.

---

## Design Principles (carried from ADR-0051)

- No rounded corners / `RoundedBorder()` — use straight box-drawing or flat rules
- No emoji
- Consistent color palette: salmonPink accent, mintGreen success, mutedGray secondary, brightWhite text
- Keyboard hints in `mutedGray italic`, kept short
- Labels flat-bold, not button-box padded unless selection state requires it
- **All overlays and toasts must respond to window resize events** — no hardcoded pixel/cell counts

---

## Responsive Sizing Model

### The Core Problem with the Current Code

The existing overlays use hardcoded constants:
```go
const overlayWidth  = 80   // help.go
const overlayHeight = 25
const viewportWidth  = 76
const viewportHeight = 20
```

On a 60-column terminal the overlay is wider than the screen and gets clipped.
On a 200-column terminal it floats as a tiny fixed box in the centre.
Viewport heights are fixed too — content overflows into the border on short terminals.

### Width: Percentage-Based with Min/Max Clamps

Every overlay computes its display width from the live terminal width at render time:

```
overlayWidth = clamp(terminalWidth × widthFactor, minWidth, maxWidth)
```

| Overlay type        | widthFactor | minWidth | maxWidth |
|---------------------|-------------|----------|----------|
| Approval / Diff     | 90%         | 60       | 140      |
| Help / Context      | 80%         | 56       | 100      |
| Settings            | 80%         | 56       | 100      |
| Notes               | 80%         | 56       | 100      |
| Toast (simple)      | 70%         | 40       | 90       |
| Toast (with detail) | 70%         | 40       | 90       |

The inner content width is always `overlayWidth - 4` (2 chars border + 2 chars padding each side).

### Height: Dynamic from Available Space

The total visible terminal height is known from the last `tea.WindowSizeMsg`. Overlays
must not exceed it:

```
maxOverlayHeight = terminalHeight - 4   // 2 line top margin, 2 line bottom margin
viewportHeight   = maxOverlayHeight - overlayChrome
```

`overlayChrome` = the fixed rows the overlay needs around the viewport:
- 1 row: top border (with title)
- 1 row: subtitle / separator
- 1 row: blank padding
- (for approval) 1 row: buttons, 1 row: hints
- 1 row: bottom border
Typical chrome is 5–7 rows. On a 24-line terminal a diff viewer would get ~17 lines of
viewport instead of a hardcoded 10.

If the terminal is so short that even the chrome doesn't fit (< 12 lines), the overlay
falls back to a compact one-line toast indicating the overlay can't be shown at this size.

### Content: Wrap, Don't Clip

All text inside overlays must be word-wrapped to `contentWidth` before rendering.
Long file paths, tool argument strings, API key values, etc. must be truncated with `…`
at the right edge, never allowed to overflow the box border.

Diff content is already line-based and can be scrolled — no truncation needed there.

### Resize Events: Overlays Must Respond

Every overlay type must implement a resize handler that recomputes its width, height,
and internal viewport dimensions when a `tea.WindowSizeMsg` arrives. Currently most
overlays ignore resize events while open.

Pattern for each overlay's `Update()`:
```go
case tea.WindowSizeMsg:
    o.width, o.height = computeOverlayDimensions(msg.Width, msg.Height)
    o.viewport.Width  = o.width - 4
    o.viewport.Height = computeViewportHeight(o.height, o.chromeRows)
```

---

## Terminal Size Breakpoints

| Terminal width | Mode     | Behaviour                                               |
|----------------|----------|---------------------------------------------------------|
| < 50 cols      | Micro    | Overlays suppressed; toast shown as 1-line status bar   |
| 50–79 cols     | Compact  | Overlays use 95% width; single-column button layout     |
| 80–119 cols    | Normal   | Standard layout as per mockups below                    |
| ≥ 120 cols     | Wide     | Overlays capped at maxWidth; extra space stays as margin|

In compact mode (50–79 cols), buttons stack vertically instead of side by side:
```
  ┌─ Tool Approval Required ──────────────────┐
  │  write_file · src/main.go                 │
  │  ─────────────────────────────────────    │
  │  @@ -12,7 +12,9 @@                        │
  │  -   old line                             │
  │  +   new line                             │
  │  ─────────────────────────────────────    │
  │  [ ✓ Accept ]                             │
  │  [ ✗ Reject ]                             │
  │  Ctrl+A · Ctrl+R · Tab · ↑↓              │
  └───────────────────────────────────────────┘
```

In micro mode (< 50 cols), the overlay is suppressed and a toast replaces it:
```
  ! Overlay requires ≥ 50 cols to display
```

---

## Overlay Container — Selected Design: Option C

Straight corners (`┌ ┐ └ ┘`), **mutedGray border**, title text embedded in the top
rule in **salmonPink**. Dimmed background via `lipgloss.Place`. No nested rounded boxes.

```
  ┌─ Title Here ──────────────────────────────────────────────┐
  │  Subtitle or secondary info                               │
  │  ─────────────────────────────────────────────────────    │
  │  [scrollable content area]                               │
  │                                                           │
  │  ─────────────────────────────────────────────────────    │
  │  [ ✓ Accept ]   [ ✗ Reject ]                             │
  │  Ctrl+A accept · Ctrl+R reject · Tab toggle · ↑↓ scroll   │
  └───────────────────────────────────────────────────────────┘
```

Width and height are computed dynamically (see Responsive Sizing Model above).

---

## Overlay Type Mockups

### 1. Diff Viewer / Tool Approval

**Normal terminal (≥ 80 cols):**
```
  ┌─ Tool Approval Required ──────────────────────────────────┐
  │  write_file · src/main.go                                 │
  │  ─────────────────────────────────────────────────────    │
  │  @@ -12,7 +12,9 @@ func main() {                          │
  │      ctx := context.Background()                          │
  │  -   client := NewClient()                                │
  │  +   client := NewClient(cfg)                             │
  │  +   defer client.Close()                                 │
  │      return client.Run(ctx)                               │
  │  ─────────────────────────────────────────────────────    │
  │  [ ✓ Accept ]   [ ✗ Reject ]                             │
  │  Ctrl+A accept · Ctrl+R reject · Tab toggle · ↑↓ scroll   │
  └───────────────────────────────────────────────────────────┘
```

**Compact terminal (50–79 cols):**
```
  ┌─ Tool Approval Required ──────────────┐
  │  write_file · src/main.go             │
  │  ─────────────────────────────────    │
  │  @@ -12,7 +12,9 @@ func main() {      │
  │  -   client := NewClient()            │
  │  +   client := NewClient(cfg)         │
  │  ─────────────────────────────────    │
  │  [ ✓ Accept ]                         │
  │  [ ✗ Reject ]                         │
  │  Ctrl+A · Ctrl+R · Tab · ↑↓          │
  └───────────────────────────────────────┘
```

Key changes from current:
- Straight corners, mutedGray border (not salmonPink rounded border)
- Title embedded in top rule in salmonPink text
- Inner content uses plain `─` separator rules — no nested box-in-box
- Buttons in compact mode stack vertically
- Viewport height computed dynamically from terminal height

---

### 2. Help Overlay

**Normal terminal (≥ 80 cols):**
```
  ┌─ Keyboard Shortcuts ──────────────────────────────────────┐
  │                                                           │
  │  Navigation                                               │
  │  ──────────                                               │
  │  Enter            send message                            │
  │  Alt+Enter        new line                                │
  │  ↑ / ↓            scroll viewport                         │
  │  PgUp / PgDn      scroll by page                          │
  │  G                jump to bottom                          │
  │                                                           │
  │  Commands                                                 │
  │  ──────────                                               │
  │  /help            show this overlay                       │
  │  /settings        open settings                           │
  │  /notes           view scratchpad                         │
  │  /context         view token usage                        │
  │  /bash            toggle bash mode                        │
  │                                                           │
  │  Esc or Enter to close                                    │
  └───────────────────────────────────────────────────────────┘
```

**Compact terminal (50–79 cols):**
```
  ┌─ Keyboard Shortcuts ──────────────┐
  │                                   │
  │  Navigation                       │
  │  ──────────                       │
  │  Enter      send message          │
  │  Alt+Enter  new line              │
  │  ↑ / ↓      scroll                │
  │  G          jump to bottom        │
  │                                   │
  │  Commands                         │
  │  ──────────                       │
  │  /help      this overlay          │
  │  /settings  settings              │
  │  /notes     scratchpad            │
  │                                   │
  │  Esc to close                     │
  └───────────────────────────────────┘
```

Notes:
- Content is a scrollable viewport — if terminal is short, user scrolls to see more
- No truncation of keybindings; descriptions may be shortened on narrow terminals
- Width computed as `min(terminalWidth × 0.80, 100)`, min 56 cols

---

### 3. Context Information Overlay

**Normal terminal (≥ 80 cols):**
```
  ┌─ Context Information ─────────────────────────────────────┐
  │                                                           │
  │  Token Usage                                              │
  │  ctx ████████░░  92k / 128k   72%                        │
  │                                                           │
  │  Breakdown                                                │
  │  ─────────────────────────────────────────────────────    │
  │  System prompt            8,241 tokens                    │
  │  Tool definitions         4,102 tokens   32 tools         │
  │  Conversation history    79,312 tokens   48 messages      │
  │    raw messages          61,100 tokens   44 msgs          │
  │    summaries             18,212 tokens    4 blocks        │
  │                                                           │
  │  Cumulative                                               │
  │  ─────────────────────────────────────────────────────    │
  │  Input tokens            142,000                          │
  │  Output tokens            28,500                          │
  │  Total tokens            170,500                          │
  │                                                           │
  │  Esc to close                                             │
  └───────────────────────────────────────────────────────────┘
```

Notes:
- Progress bar is the same `█░` style as the main status bar (reuses same render function)
- Numbers are right-aligned within their column — column widths computed from content
- Scrollable viewport if terminal is short

---

### 4. Notes Overlay

**Normal terminal (≥ 80 cols):**
```
  ┌─ Scratchpad Notes (3) ────────────────────────────────────┐
  │                                                           │
  │  > [decision, auth]                                       │
  │    Use JWT with refresh tokens for auth scaling           │
  │                                                           │
  │    [pattern, test]                                        │
  │    Test suite requires DB migration before running        │
  │                                                           │
  │    [bug, api]                                             │
  │    Payment service 500s when user context missing         │
  │                                                           │
  │  ─────────────────────────────────────────────────────    │
  │  ↑↓ select · Enter view · Esc close                       │
  └───────────────────────────────────────────────────────────┘
```

**Compact terminal (50–79 cols):**
```
  ┌─ Scratchpad Notes (3) ──────────┐
  │                                 │
  │  > [decision, auth]             │
  │    Use JWT with refresh toke…   │
  │                                 │
  │    [pattern, test]              │
  │    Test suite requires DB mi…   │
  │                                 │
  │  ─────────────────────────────  │
  │  ↑↓ · Enter · Esc               │
  └─────────────────────────────────┘
```

Notes:
- Selected item shown with `>` glyph in salmonPink
- Content preview truncated to `contentWidth - 4` with `…` suffix (never clips into border)
- Tags in mutedGray, content in brightWhite
- Width from `min(terminalWidth × 0.80, 100)`, min 56 cols

---

### 5. Settings Overlay (abbreviated)

**Normal terminal (≥ 80 cols):**
```
  ┌─ Settings ────────────────────────────────────────────────┐
  │                                                           │
  │  Provider                                                 │
  │  ─────────────────────────────────────────────────────    │
  │  API Key         ••••••••••••••••••••[sk-ant-...]         │
  │  Model           claude-opus-4-5                          │
  │  Max Tokens      8192                                     │
  │                                                           │
  │  Behaviour                                                │
  │  ─────────────────────────────────────────────────────    │
  │  Auto-compact    on                                       │
  │  Max turns       50                                       │
  │                                                           │
  │  Tab next · Shift+Tab prev · Enter edit · Esc close       │
  └───────────────────────────────────────────────────────────┘
```

**Editing a field (inline, not a pop-up dialog):**
```
  ┌─ Settings ────────────────────────────────────────────────┐
  │  ...                                                      │
  │  > Model     claude-opus-4-5_                             │
  │  ...                                                      │
  │  Enter confirm · Esc cancel                               │
  └───────────────────────────────────────────────────────────┘
```

The `>` glyph (salmonPink) marks the actively-edited field.
The trailing `_` represents the text cursor position.

Notes:
- Field name column width fixed at `max(longest field name + 2, 18)`
- Value column gets the remaining `contentWidth - fieldColWidth`
- Long values (API keys) truncated with `…` at right edge; full value visible during edit
- Settings scrolls as a viewport if there are more sections than terminal height allows

---

## Toast Mockups

Toasts are transient overlays above the input box (~3 second lifetime).
They use `renderToastOverlay()` which overlays lines above the input.

### Simple Toast (success/info — one line, no border)

```
  ✓  Settings saved successfully
```

- No border — flat styled line
- Icon + message on one line
- salmonPink icon for neutral/success; ProgressRed for error
- Left-padded 2 chars from left edge
- Width: message is truncated at `terminalWidth - 6` with `…` if too long

### Error Toast (one line)

```
  ✗  Failed to save: permission denied
```

- Same flat one-line style; icon and text in ProgressRed

### Toast with Detail (two lines, thin straight box)

When a toast has both a message and a detail string:

```
  ┌──────────────────────────────────────────────────────┐
  │ ✓  Git commit created                                │
  │    abc1234 · feat: add new overlay design            │
  └──────────────────────────────────────────────────────┘
```

- Straight corners, mutedGray border
- Width: `min(terminalWidth - 8, 90)`, min 40
- Both lines truncated at `contentWidth - 2` with `…`

### Summarization In-Progress (persistent, not a toast)

Shown while context summarization is running — stays until complete:

```
  ┌──────────────────────────────────────────────────────┐
  │ ◆  Optimizing context  [selective-summary]           │
  │    ████████████░░░░░░░░  12/20 items  (60%)          │
  │    Summarizing turn 12 / tool result                 │
  └──────────────────────────────────────────────────────┘
```

- Straight corners, mutedGray border
- `◆` glyph (not emoji)
- Progress bar reuses `█░` style (same render function as status bar and context overlay)
- Width: `min(terminalWidth - 8, 90)`, min 40
- On narrow terminals the progress bar shrinks to fit: `barWidth = contentWidth - 20`

---

## Responsive Rules — Implementation Checklist

These are the concrete constraints every overlay and toast implementation must satisfy:

### Width
- [ ] No hardcoded `overlayWidth = 80` constants — compute from `terminalWidth`
- [ ] `contentWidth = overlayWidth - 4` (border + padding)
- [ ] All text truncated at `contentWidth` with `…` before rendering
- [ ] `lipgloss.Width()` used for all width measurements, never `len()`

### Height
- [ ] No hardcoded `viewportHeight = 20` constants
- [ ] `viewportHeight = terminalHeight - chromeRows - 4` (top/bottom margin)
- [ ] `chromeRows` is a const per overlay type (title row + separator + footer rows)
- [ ] If `viewportHeight < 3`, show compact "terminal too small" notice instead

### Resize
- [ ] Every overlay handles `tea.WindowSizeMsg` and recomputes its dimensions
- [ ] `BaseOverlay.Update()` routes `tea.WindowSizeMsg` to resize handler
- [ ] Viewport height is updated: `o.viewport.Height = newViewportHeight`

### Compact mode (50–79 cols)
- [ ] Approval buttons stack vertically (one per line)
- [ ] Hint text shortened: `"Ctrl+A · Ctrl+R · ↑↓"` instead of full sentence
- [ ] Column layout in context/settings collapses to label-then-value on next line

### Micro mode (< 50 cols)
- [ ] Modal overlays suppressed
- [ ] A single-line toast shown instead: `! Terminal too narrow for overlay`

### Toasts
- [ ] Width: `min(terminalWidth - 8, 90)`, min 40
- [ ] Message truncated at `toastWidth - 6` with `…`
- [ ] Positioned above input: `startLine = len(baseLines) - 5 - toastHeight`

---

## Open Questions (resolved)

1. **Overlay container style**: Option C (straight corners, mutedGray border, title in rule) — confirmed.
2. **Toast style**: flat one-line for simple; compact straight-corner box when detail line present — confirmed.
3. **Notes selected item**: `>` glyph in salmonPink (replace bubble list full-line highlight).
4. **Settings edit state**: inline `>` glyph + cursor in field row (not a pop-up dialog).
5. **Responsive**: all dimensions computed dynamically from terminal size; hardcoded constants removed.
