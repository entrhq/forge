# 50. TUI Clipboard Copy

**Status:** Implemented
**Date:** 2025-01-26
**Deciders:** Product Team, Engineering Team
**Technical Story:** [TUI Visual Polish PRD](../product/features/tui-visual-polish.md) — copy conversation content use case

---

## Context

The Forge TUI runs inside an alt-screen session with `tea.WithMouseCellMotion()` active. These two settings together prevent native terminal text selection: the alt-screen hides the scrollback buffer, and mouse capture intercepts drag events before they reach the terminal emulator's selection logic. There is currently no way to extract text from the conversation viewport.

This is a significant friction point when the agent produces useful code, commands, or explanations — the user must switch to a separate file viewer or re-ask the agent to repeat itself.

### Background

The `atotto/clipboard` package is already present in the Forge module and used elsewhere in the codebase. It provides `clipboard.WriteAll(string) error` which writes to the OS clipboard without any UI interaction.

### Problem Statement

Users cannot copy text from the conversation viewport to use in their editor, shell, or other tools. The TUI's alt-screen + mouse-capture configuration makes native terminal selection impossible.

### Goals

- Provide a keyboard shortcut (`Ctrl+Y`) that copies the full visible conversation buffer to the OS clipboard
- Show a brief toast confirmation when copy succeeds
- Handle clipboard unavailability gracefully (headless servers, WSL without clip.exe, etc.)

### Non-Goals

- Selective copy of a single message (future — requires message-level selection UI)
- Mouse-drag selection within the TUI viewport (requires removing mouse capture — larger change)
- Clipboard read / paste via `Ctrl+V` in the main textarea (handled by ADR-0049)
- Suspend/resume for native terminal selection (`tea.Suspend` sends SIGTSTP which causes terminal emulators to close the window entirely rather than suspend the process — not viable)

---

## Decision Drivers

* **User impact**: No copy path exists today — any copy capability is a strict improvement
* **Low risk**: `Ctrl+Y` is additive; nothing existing is modified
* **Dependency already present**: `atotto/clipboard` is already in the module

---

## Considered Options

### Option 1: Ctrl+Y copies conversation buffer to clipboard (CHOSEN)

**Description:** On `Ctrl+Y`, extract the full content string from `m.content`, strip ANSI codes, write it to the OS clipboard via `clipboard.WriteAll()`, and show a 2-second toast.

**Pros:**
- Single keypress, works everywhere the clipboard daemon is available
- No visual disruption — TUI stays active
- Implementation is ~20 lines
- Copies full conversation history regardless of scroll position

**Cons:**
- Does not work on headless servers without a clipboard daemon
- Copies the full buffer, not a selected range

### Option 2: Export to pager (Ctrl+E)

**Description:** Pipe conversation content to `$PAGER` (default `less`) using `tea.ExecProcess()`.

**Pros:**
- Full scrollback + search in a familiar tool
- Works without clipboard

**Cons:**
- Leaves the TUI context entirely
- `less` does not copy to clipboard — user still needs an extra step
- More complex than `Ctrl+Y`

---

## Decision

**Chosen Option:** Option 1 (`Ctrl+Y` clipboard copy).

### Rationale

`Ctrl+Y` solves the most common case (copy a code block, paste into editor) with minimal complexity. The pager option adds complexity without enabling clipboard copy directly.

---

## Consequences

### Positive

- Users can extract code and commands from conversations in one keypress
- Toast provides immediate confirmation that copy succeeded
- Clipboard error is handled gracefully — users on headless servers see a clear error toast

### Negative

- `Ctrl+Y` may conflict with a user's terminal keybinding (e.g. some readline configurations use `Ctrl+Y` for yank). This is a known trade-off; the binding can be made configurable in a future settings iteration.
- Does not work on headless servers without a clipboard daemon — users see an error toast but have no in-TUI fallback
- Copying the full buffer includes ANSI escape codes unless stripped — the clipboard content must be plain text

### Neutral

- Toast system is already implemented (`m.toast` in model, `showToast()` in `view.go`) — no new infrastructure needed

---

## Implementation

### Implementation (~20 lines, 2 files)

**`pkg/executor/tui/update.go`** — add to the `tea.KeyMsg` switch in `handleKeyPress()`:

```go
case tea.KeyCtrlY:
    return m.handleCopyToClipboard()
```

**`pkg/executor/tui/update.go`** — new method:

```go
// handleCopyToClipboard copies the full conversation buffer to the OS clipboard
// and shows a brief toast confirmation (ADR-0050).
// It uses m.content (the raw conversation string builder) rather than
// m.viewport.View() so the user always gets the complete history, not just the
// visible window. ANSI escape codes are stripped so the clipboard contains
// plain text.
func (m *model) handleCopyToClipboard() (tea.Model, tea.Cmd) {
    content := stripANSI(m.content.String())
    if err := clipboard.WriteAll(content); err != nil {
        m.showToast("Clipboard unavailable", "No clipboard manager detected", "⚠", true)
        return m, nil
    }
    m.showToast("Copied to clipboard", "", "✓", false)
    return m, nil
}
```

`clipboard` is `github.com/atotto/clipboard` — already in the module. Add import to `update.go`.

**ANSI stripping** — a small helper to remove escape sequences from the content string so the clipboard receives plain text:

```go
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
    return ansiEscape.ReplaceAllString(s, "")
}
```

Place `stripANSI` in `pkg/executor/tui/helpers.go` (file already exists).

**`pkg/executor/tui/view.go`** — add `Ctrl+Y · copy` to the hints line in `buildTips()`:

```
Enter · send   Alt+Enter · new line   / · commands   Ctrl+Y · copy   Ctrl+C · exit
```

### Migration Path

No migration needed. The binding is additive — no existing behaviour is changed.

### Timeline

- ~45 min, single PR

---

## Validation

### Success Metrics

- Manual: press `Ctrl+Y` → toast appears → paste into editor → plain text, no ANSI escapes
- Manual: press `Ctrl+Y` on a headless server without clipboard → error toast appears, no crash
- Regression: `Ctrl+C` (exit) and all other keybindings continue to work

### Monitoring

- Watch for reports of ANSI escapes in clipboard content (indicates `stripANSI` missed a sequence)

---

## Related Decisions

- [ADR-0012](0012-enhanced-tui-executor.md) — Enhanced TUI Executor (alt-screen + mouse capture, root cause)
- [ADR-0049](0049-tui-bracketed-paste-support.md) — TUI Bracketed Paste (companion feature)
- [ADR-0051](0051-tui-visual-redesign.md) — TUI Visual Redesign (hints line update)

---

## References

- [TUI Visual Polish PRD](../product/features/tui-visual-polish.md)
- [Design exploration](../product/scratch/tui-ux-enhancements.md)
- `atotto/clipboard`: https://github.com/atotto/clipboard

---

## Notes

The implementation uses `m.content.String()` (the full conversation `strings.Builder`) rather than `m.viewport.View()` (visible window only), so the clipboard always receives the complete history regardless of scroll position. ANSI codes are stripped via `stripANSI()` before writing.

**Last Updated:** 2025-01-26
