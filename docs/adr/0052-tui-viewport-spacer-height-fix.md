# 52. TUI Viewport Spacer Height Fix

**Status:** Implemented
**Date:** 2025-01-30
**Deciders:** Engineering Team
**Technical Story:** Fix viewport height calculation to account for visual spacer line in `assembleBaseView`

---

## Context

After the TUI visual redesign (ADR-0051) which replaced the 6-line ASCII art header with a compact single-line header bar, users reported a persistent layout bug: blank space appeared below the TUI viewport, with the amount of blank space proportional to the window resize delta. The bug only manifested when:

1. User resized the terminal window to make it smaller (height reduction)
2. User then pressed space or any key that triggered textarea content changes

The blank space would remain until the next layout recalculation was forced by another resize or viewport scroll event.

### Background

The TUI layout is assembled in `pkg/executor/tui/view.go` via `assembleBaseView()`, which joins multiple sections vertically:

```go
func (m *model) assembleBaseView(header, tips, viewportSection, scrollIndicator, loadingIndicator, inputBox, bottomBar string) string {
    var middle []string
    middle = append(middle, viewportSection)
    if scrollIndicator != "" {
        middle = append(middle, scrollIndicator)
    }
    if m.agentBusy {
        middle = append(middle, loadingIndicator)
    }

    // Visual spacer between header and content
    rows := []string{header, tips, ""}
    rows = append(rows, middle...)
    rows = append(rows, inputBox, bottomBar)

    result := lipgloss.JoinVertical(lipgloss.Left, rows...)
    return result
}
```

Note line 231: `rows := []string{header, tips, ""}` — a blank string `""` is inserted as a visual spacer between the tips line and the viewport content.

The viewport height is calculated in two places in `pkg/executor/tui/update.go`:

1. Inside `Update()` when detecting textarea content changes (inline calculation)
2. In the dedicated `calculateViewportHeight()` function called during layout recalculation

Both calculations used this formula:

```go
headerHeight := 2  // header line + tips line
inputZoneHeight := 1 + strings.Count(m.textarea.Value(), "\n") + 1
statusBarHeight := 1
loadingHeight := 0  // or 1 if agentBusy
scrollIndicatorHeight := 0  // or 1 if scroll lock active

viewportHeight := m.height - headerHeight - inputZoneHeight - statusBarHeight - loadingHeight - scrollIndicatorHeight
```

**The bug:** This formula never accounted for the visual spacer line. The viewport was allocated 1 extra line beyond what the terminal could actually display.

### Problem Statement

The blank spacer row inserted at `view.go:231` consumed 1 line of vertical space but was never subtracted from the viewport height budget in either calculation site. This caused a cumulative off-by-one error during rapid window resize sequences:

1. Each `WindowSizeMsg` event called `handleWindowResize()`, which set `m.height` and `m.width`
2. `calculateViewportHeight()` was called, returning a viewport height that was 1 line too large
3. `GotoBottom()` was called with the incorrect height, setting `YOffset` based on wrong dimensions
4. The next resize event arrived before the layout could stabilize, compounding the error
5. More resize events = more compounding = larger final gap

The bug appeared proportional to resize delta because:
- Small resize (2-3 lines) = 2-3 resize events = 2-3 compounded errors = 2-3 blank lines
- Large resize (5+ lines) = 5+ resize events = 5+ compounded errors = 5+ blank lines

Debug logs confirmed that after resizing from height=20 to height=16 (4 steps), `YOffset=3` — consistent with 3 compounding errors across a 4-event sequence.

### Goals

- Fix the viewport height calculation to account for the visual spacer line
- Apply the fix to both calculation sites (inline in `Update()` and `calculateViewportHeight()`)
- Ensure the layout remains stable during rapid resize sequences
- Remove all temporary debug logging added during investigation

### Non-Goals

- Removing the visual spacer (it serves a legitimate UX purpose)
- Refactoring the dual-calculation-site architecture (separate concern)
- Changing the `assembleBaseView()` row assembly order

---

## Decision Drivers

* **Correctness** — The viewport height formula must match the actual vertical space consumed by all UI elements
* **Maintainability** — The fix must be obvious to future maintainers (both the constant declaration and the comment explaining its origin)
* **Consistency** — Both calculation sites must use the same formula
* **Evidence-based** — The fix must be confirmed via debug logs before being applied

---

## Considered Options

### Option 1: Remove the visual spacer

**Description:** Delete the blank string from the `rows` array in `assembleBaseView()`.

**Pros:**
- Eliminates the accounting problem entirely
- Simpler layout code

**Cons:**
- Degrades UX — the spacer provides visual breathing room between tips and content
- Contradicts the visual design from ADR-0051

### Option 2: Add `spacerHeight = 1` constant

**Description:** Declare `const spacerHeight = 1` in `calculateViewportHeight()` and subtract it from the viewport height budget. Add an inline comment referencing the `assembleBaseView` line number.

**Pros:**
- Minimal change — just adds one constant and one subtraction
- Self-documenting via constant name and comment
- Preserves visual design
- Easy to verify via log analysis

**Cons:**
- Requires applying the fix to two separate calculation sites
- Constant must be kept in sync with `assembleBaseView()` implementation

### Option 3: Dynamically measure spacer height

**Description:** Count the number of empty strings in the `rows` array at runtime and adjust viewport height accordingly.

**Pros:**
- No hardcoded constant to maintain
- Automatically adapts to layout changes

**Cons:**
- Adds runtime overhead to every layout calculation
- Couples viewport height calculation to view assembly internals
- Overly complex for a single blank line

---

## Decision

**Chosen Option:** Option 2 — Add `spacerHeight = 1` constant

### Rationale

The visual spacer serves a legitimate UX purpose and was explicitly introduced in ADR-0051. Removing it (Option 1) would degrade the visual design. Dynamic measurement (Option 3) adds unnecessary complexity for a single static line.

Option 2 is the minimal fix that preserves the visual design while being self-documenting. The constant name `spacerHeight` clearly communicates its purpose, and the inline comment linking to `assembleBaseView` line 231 ensures future maintainers understand the dependency.

The fix was verified via debug logs before being applied:
- Debug logging confirmed `YOffset` incremented by 1 on each resize-smaller event
- After applying the fix, build succeeded and user confirmed the bug was resolved
- All debug logging was then removed

---

## Consequences

### Positive

- Viewport layout is now stable during rapid resize sequences
- No blank space appears below the TUI after resize + keypress
- The fix is self-documenting via constant name and comment
- Visual design from ADR-0051 is preserved

### Negative

- The `spacerHeight` constant must be manually updated if `assembleBaseView()` changes its spacer implementation
- The fix exists in two separate calculation sites that must be kept in sync

### Neutral

- One additional constant declaration in `calculateViewportHeight()`
- One additional subtraction in the viewport height formula at both sites

---

## Implementation

The fix was applied to `pkg/executor/tui/update.go` at two sites:

### Site 1: Inline calculation in `Update()` (around line 140)

```go
headerHeight := 2
inputZoneHeight := 1 + strings.Count(m.textarea.Value(), "\n") + 1
statusBarHeight := 1
loadingHeight := 0
if m.agentBusy {
    loadingHeight = 1
}
scrollIndicatorHeight := 0
if !m.followScroll && m.hasNewContent {
    scrollIndicatorHeight = 1
}
// Visual spacer line between header and viewport (assembleBaseView line 231 adds "")
const spacerHeight = 1

newVpHeight := m.height - headerHeight - spacerHeight - inputZoneHeight - statusBarHeight - loadingHeight - scrollIndicatorHeight
```

### Site 2: `calculateViewportHeight()` function (around line 289)

```go
func (m *model) calculateViewportHeight() int {
    headerHeight := 2
    inputZoneHeight := 1 + strings.Count(m.textarea.Value(), "\n") + 1
    statusBarHeight := 1

    loadingHeight := 0
    if m.agentBusy {
        loadingHeight = 1
    }

    scrollIndicatorHeight := 0
    if !m.followScroll && m.hasNewContent {
        scrollIndicatorHeight = 1
    }

    // Visual spacer line between header and viewport (assembleBaseView line 231 adds "")
    const spacerHeight = 1

    viewportHeight := m.height - headerHeight - spacerHeight - inputZoneHeight - statusBarHeight - loadingHeight - scrollIndicatorHeight
    if viewportHeight < 1 {
        viewportHeight = 1
    }
    return viewportHeight
}
```

Both sites now include:
1. `const spacerHeight = 1` declaration
2. Inline comment referencing `assembleBaseView` line 231
3. Subtraction of `spacerHeight` in the formula

### Migration Path

No migration required — this is a runtime layout calculation fix. Existing TUI sessions will automatically pick up the corrected layout on the next resize or recalculation event.

### Cleanup

All temporary debug logging added during investigation was removed:
- `pkg/executor/tui/update.go`: Removed 5 `debugLog.Debugf` calls (UPDATE-START, KEY-DEBUG, TEXTAREA-UPDATE, UPDATE-END, RESIZE)
- `pkg/executor/tui/events.go`: Removed 7 debug logs from `scrollToBottomOrMark()`
- `pkg/executor/tui/view.go`: Removed 4 debug blocks (VIEWPORT-VIEW, FINAL-VIEW, INPUT-BOX, VIEW assembleBaseView)

---

## Validation

### Success Metrics

1. No blank space appears below the TUI after window resize + keypress
2. Layout remains stable during rapid resize sequences (20+ events/second)
3. `make build` succeeds with no compilation errors
4. No debug logging remains in production code

### Verification

User confirmed via interactive testing:
- Resized terminal window from 25 lines to 15 lines (10-line delta)
- Pressed space key after resize
- No blank space appeared below the viewport
- Layout remained stable during subsequent interaction

Build verification:
```bash
make build
# Output: ✓ forge built successfully at .bin/forge
```

Code search verification:
```bash
rg "debugLog\." pkg/executor/tui/
# Output: No matches found
```

---

## Related Decisions

- [ADR-0051](0051-tui-visual-redesign.md) — TUI Visual Redesign (introduced the compact header and visual spacer)
- [ADR-0025](0025-tui-package-reorganization.md) — TUI Package Reorganization (established view/update separation)

---

## References

- `pkg/executor/tui/view.go:231` — Visual spacer insertion in `assembleBaseView()`
- `pkg/executor/tui/update.go:302` — `calculateViewportHeight()` implementation
- `pkg/executor/tui/update.go:~140` — Inline viewport height calculation in `Update()`

---

## Notes

**Why the bug appeared proportional to resize delta:**

The bug appeared proportional because of cascading errors during rapid resize sequences. Each `WindowSizeMsg` event:
1. Called `calculateViewportHeight()` which returned a height 1 line too large
2. Called `GotoBottom()` with incorrect height, setting `YOffset` based on wrong dimensions
3. The next event arrived before layout could stabilize, compounding the error

More resize events = more compounding = larger final gap. This is why a 2-line resize produced 2 blank lines, and a 5-line resize produced 5 blank lines.

**Why `m.textarea.Height()` was replaced:**

The original inline calculation used `m.textarea.Height()` which returns the previous frame's allocated component height, not the live content line count. This caused a one-frame lag artifact. The fix uses `strings.Count(m.textarea.Value(), "\n") + 1` for same-tick accuracy.

**Last Updated:** 2025-01-30
