# 50. TUI Clipboard Copy

**Status:** Proposed
**Date:** 2025-01-26
**Deciders:** Product Team, Engineering Team
**Technical Story:** [TUI Visual Polish PRD](../product/features/tui-visual-polish.md) — copy conversation content use case

---

## Context

The Forge TUI runs inside an alt-screen session with `tea.WithMouseCellMotion()` active. These two settings together prevent native terminal text selection: the alt-screen hides the scrollback buffer, and mouse capture intercepts drag events before they reach the terminal emulator's selection logic. There is currently no way to extract text from the conversation viewport.

This is a significant friction point when the agent produces useful code, commands, or explanations — the user must switch to a separate file viewer or re-ask the agent to repeat itself.

### Background

The `atotto/clipboard` package is already present in the Forge module and used elsewhere in the codebase. It provides `clipboard.WriteAll(string) error` which writes to the OS clipboard without any UI interaction.

Bubble Tea v1.3.10 provides `tea.Suspend()` and `tea.ResumeMsg` which can temporarily hand control back to the parent terminal, restoring the primary screen buffer and disabling mouse capture. This allows the user to perform native terminal selection, then resume the TUI.

### Problem Statement

Users cannot copy text from the conversation viewport to use in their editor, shell, or other tools. The TUI's alt-screen + mouse-capture configuration makes native terminal selection impossible.

### Goals

- Provide a keyboard shortcut (`Ctrl+Y`) that copies the full visible conversation buffer to the OS clipboard
- Show a brief toast confirmation when copy succeeds
- Handle clipboard unavailability gracefully (headless servers, WSL without clip.exe, etc.)
- P2: provide `Ctrl+T` to suspend the TUI for native terminal selection, then resume on any keypress

### Non-Goals

- Selective copy of a single message (future — requires message-level selection UI)
- Mouse-drag selection within the TUI viewport (requires removing mouse capture — larger change)
- Clipboard read / paste via `Ctrl+V` in the main textarea (handled by ADR-0049)

---

## Decision Drivers

* **User impact**: No copy path exists today — any copy capability is a strict improvement
* **Low risk**: `Ctrl+Y` is additive; nothing existing is modified
* **Dependency already present**: `atotto/clipboard` is already in the module
* **Standard Bubble Tea pattern**: `tea.Suspend()` / `tea.ResumeMsg` is the documented pattern for handing off to external processes

---

## Considered Options

### Option 1: Ctrl+Y copies conversation buffer to clipboard (CHOSEN for P0/P1)

**Description:** On `Ctrl+Y`, extract the content string from `m.viewport`, write it to the OS clipboard via `clipboard.WriteAll()`, and show a 2-second toast.

**Pros:**
- Single keypress, works everywhere the clipboard daemon is available
- No visual disruption — TUI stays active
- Implementation is ~20 lines

**Cons:**
- Does not work on headless servers without a clipboard daemon
- Copies the full buffer, not a selected range

### Option 2: Ctrl+T suspends TUI for native selection (P2)

**Description:** On `Ctrl+T`, call `tea.Suspend()`. The TUI hands control to the parent terminal (primary buffer restored, mouse capture off). A message is printed: "Select text then press any key to resume Forge". On keypress, `tea.Resume()` is called and the TUI reactivates.

**Pros:**
- Works without a clipboard daemon
- Allows selecting any arbitrary span, not just the full buffer
- Native terminal behaviour — familiar to users

**Cons:**
- More complex — must handle `tea.ResumeMsg` in `Update()`
- Screen flicker on suspend/resume depending on terminal emulator
- Not available on all platforms (`tea.Suspend()` sends SIGTSTP — not supported on Windows)

### Option 3: Export to pager (Ctrl+E)

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

**Chosen Option:** Option 1 (`Ctrl+Y` clipboard copy) as the primary implementation. Option 2 (`Ctrl+T` suspend) as a P2 follow-on in the same ADR scope.

### Rationale

`Ctrl+Y` solves the most common case (copy a code block, paste into editor) with minimal complexity. `Ctrl+T` solves the headless / arbitrary-selection case and is worth implementing as a follow-on once `Ctrl+Y` is shipped and tested. Option 3 adds a third mechanism without adding enough value over the first two.

---

## Consequences

### Positive

- Users can extract code and commands from conversations in one keypress
- Toast provides immediate confirmation that copy succeeded
- Clipboard error is handled gracefully — users on headless servers see a clear error toast
- P2 suspend path gives a no-clipboard fallback

### Negative

- `Ctrl+Y` may conflict with a user's terminal keybinding (e.g. some readline configurations use `Ctrl+Y` for yank). This is a known trade-off; the binding can be made configurable in a future settings iteration.
- Copying the full buffer includes ANSI escape codes unless stripped — the clipboard content must be plain text

### Neutral

- Toast system is already implemented (`m.toast` in model, `showToast()` in `view.go`) — no new infrastructure needed

---

## Implementation

### Phase 1 — Ctrl+Y clipboard copy (~20 lines, 2 files)

**`pkg/executor/tui/update.go`** — add to the `tea.KeyMsg` switch in `handleKeyPress()`:

```go
case "ctrl+y":
    return m.handleCopyToClipboard()
```

**`pkg/executor/tui/update.go`** — new method:

```go
// handleCopyToClipboard copies the conversation viewport content to the OS
// clipboard and shows a toast confirmation.
func (m model) handleCopyToClipboard() (model, tea.Cmd) {
    content := stripANSI(m.viewport.View())
    if err := clipboard.WriteAll(content); err != nil {
        m.showToast("⚠  Clipboard unavailable — use Ctrl+T to select text")
        return m, nil
    }
    m.showToast("✓  Copied to clipboard")
    return m, nil
}
```

`clipboard` is `github.com/atotto/clipboard` — already in the module. Add import to `update.go`.

**ANSI stripping** — a small helper to remove escape sequences from the viewport string so the clipboard receives plain text:

```go
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
    return ansiEscape.ReplaceAllString(s, "")
}
```

Place `stripANSI` in `pkg/executor/tui/helpers.go` (file already exists).

**`pkg/executor/tui/view.go`** — add `Ctrl+Y · copy` to the hints line in `buildTips()` (or the contextual hints equivalent from ADR-0051):

```
Enter · send   Alt+Enter · new line   / · commands   Ctrl+Y · copy   Ctrl+C · exit
```

### Phase 2 — Ctrl+T suspend (P2, ~25 lines, 1 file)

**`pkg/executor/tui/update.go`** — add to the `tea.KeyMsg` switch:

```go
case "ctrl+t":
    return m.handleSuspendForSelection()
```

**`pkg/executor/tui/update.go`** — new method:

```go
func (m model) handleSuspendForSelection() (model, tea.Cmd) {
    // Print instructions to stdout before suspending — they will be visible
    // in the primary screen buffer while the TUI is suspended.
    fmt.Fprintln(os.Stderr, "\nForge suspended. Select text, then press any key to resume.")
    return m, tea.Suspend
}
```

**`pkg/executor/tui/update.go`** — handle `tea.ResumeMsg` in the main `Update()` switch:

```go
case tea.ResumeMsg:
    // TUI is resuming after suspension — force a full re-render
    return m, tea.ClearScreen
```

**Platform guard** — `tea.Suspend` sends SIGTSTP and is not available on Windows. Add a build tag or a runtime check:

```go
case "ctrl+t":
    if runtime.GOOS == "windows" {
        m.showToast("⚠  Suspend not available on Windows — use Ctrl+Y to copy")
        return m, nil
    }
    return m.handleSuspendForSelection()
```

### Migration Path

No migration needed. Both bindings are additive — no existing behaviour is changed.

### Timeline

- **Phase 1** (`Ctrl+Y` clipboard): ~45 min, single PR
- **Phase 2** (`Ctrl+T` suspend): ~30 min, follow-on PR or included in Phase 1 PR if time allows

---

## Validation

### Success Metrics

- Manual: press `Ctrl+Y` → toast appears → paste into editor → plain text, no ANSI escapes
- Manual: press `Ctrl+Y` on a headless server without clipboard → error toast appears, no crash
- Manual (P2): press `Ctrl+T` → TUI suspends, instructions visible → select text → press key → TUI resumes cleanly
- Manual (P2): press `Ctrl+T` on Windows → error toast, no crash
- Regression: `Ctrl+C` (exit) and all other keybindings continue to work

### Monitoring

- Watch for reports of ANSI escapes in clipboard content (indicates `stripANSI` missed a sequence)
- Watch for reports of `Ctrl+T` leaving the terminal in a bad state (indicates `ResumeMsg` handling is incomplete)

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
- Bubble Tea Suspend: https://pkg.go.dev/github.com/charmbracelet/bubbletea#Suspend

---

## Notes

`m.viewport.View()` returns the rendered string of the currently visible viewport lines including ANSI colour codes. For the full conversation buffer (not just the visible window), use the underlying content string stored in the model — typically `m.viewport.SetContent()` was called with a pre-rendered string. If a reference to the raw content is not stored separately, copy from `m.viewport.View()` and strip ANSI; the result will be the visible window only. Consider storing the raw markdown/plain-text conversation separately in a future iteration to enable full-buffer copy.

**Last Updated:** 2025-01-26
