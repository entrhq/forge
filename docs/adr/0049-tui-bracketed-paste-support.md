# 49. TUI Bracketed Paste Support

**Status:** Proposed
**Date:** 2025-01-26
**Deciders:** Product Team, Engineering Team
**Technical Story:** [TUI Visual Polish PRD](../product/features/tui-visual-polish.md) — paste in settings use case

---

## Context

Paste does not work in the Forge TUI settings dialogs. When a user copies a 51-character API key from their password manager and pastes it into a settings input field, the paste is silently discarded. The user is forced to type the key character by character.

There are two independent root causes:

1. **Program level**: `tea.WithBracketedPaste()` is not passed to `tea.NewProgram()` in `pkg/executor/tui/executor.go`. Without this option, Bubble Tea never sends bracketed paste sequences to the `Update()` function as `tea.PasteMsg` — they arrive as a rapid burst of individual `tea.KeyMsg` events instead, which the textarea may or may not handle correctly.

2. **Settings overlay level**: `handleDialogCharInput()` in `pkg/executor/tui/overlay/settings.go:1273` only accepts input where `len(keyMsg.String()) == 1`. Multi-character events (including any paste that did reach the function) are silently dropped. Additionally, there is no `tea.PasteMsg` case in the `handleDialogInput()` switch at `settings.go:1144`.

The main chat textarea is a `charmbracelet/bubbles/textarea` component which handles `tea.PasteMsg` natively — once bracketed paste is enabled at the program level, the main input benefits automatically. Only the settings dialog needs a bespoke handler.

### Background

Bracketed paste is a terminal feature (xterm extension, widely supported) where the terminal wraps pasted text with `ESC[200~` ... `ESC[201~`. When Bubble Tea receives these sequences with `tea.WithBracketedPaste()` enabled, it delivers a single `tea.PasteMsg` containing the full pasted string, rather than individual key events for each character.

Current Bubble Tea version: `v1.3.10`. `tea.WithBracketedPaste()` and `tea.PasteMsg` are available and stable in this version.

### Problem Statement

Users cannot paste text into settings dialog fields (API keys, model names, base URLs). This is a critical friction point for new users configuring Forge for the first time and for existing users rotating API keys.

### Goals

- OS-native paste (Cmd+V / middle-click / Shift+Insert) must insert text into all editable settings fields on terminals that support bracketed paste
- Multi-character paste must not be truncated or silently dropped
- Paste in the main chat textarea must continue to work (it already does via the bubbles textarea component)
- Terminals that do not support bracketed paste must not regress (fall back to current single-char behaviour)

### Non-Goals

- Paste into read-only settings fields (silently ignore, same as today)
- Clipboard read via `Ctrl+V` when bracketed paste is unavailable (requires `atotto/clipboard` read path — separate work)
- Paste history or undo in settings fields

---

## Decision Drivers

* **User impact**: First-time configuration is broken for anyone using a password manager
* **Implementation size**: Two files, ~30 lines total — very low risk
* **Library support**: `tea.WithBracketedPaste()` is already available in the current version
* **Zero regression**: Terminals without bracketed paste fall back to existing behaviour

---

## Considered Options

### Option 1: Enable bracketed paste at program level only

**Description:** Add `tea.WithBracketedPaste()` to `tea.NewProgram()`. Rely on the bubbles textarea to handle `tea.PasteMsg` in the main input. Do nothing about the settings overlay.

**Pros:**
- One-line change
- Fixes main textarea paste

**Cons:**
- Settings dialogs still cannot accept paste — the original problem is unsolved
- `tea.PasteMsg` delivered to `Update()` would propagate to the settings overlay's `Update()` but have no handler, effectively discarding it

### Option 2: Handle paste in settings overlay without enabling bracketed paste (REJECTED)

**Description:** Handle `Ctrl+V` in the settings overlay by reading from the OS clipboard directly via `atotto/clipboard`.

**Pros:**
- Works even when the terminal does not support bracketed paste

**Cons:**
- Requires clipboard read permission / daemon (not available in all environments)
- Does not fix middle-click or shift+insert paste (which use bracketed paste, not Ctrl+V)
- Adds a clipboard dependency to the settings overlay
- Inconsistent with how every other Bubble Tea application handles paste

### Option 3: Enable bracketed paste at program level AND add PasteMsg handler in settings overlay (CHOSEN)

**Description:** Add `tea.WithBracketedPaste()` to `tea.NewProgram()`. Add a `tea.PasteMsg` case to `handleDialogInput()` in the settings overlay that calls a new `handleDialogPaste()` method. Also relax the `len(keyMsg.String()) == 1` guard in `handleDialogCharInput()` to allow multi-rune key strings.

**Pros:**
- Fixes all paste mechanisms (Cmd+V, Ctrl+V, middle-click, shift+insert) for both settings and main input
- Consistent with Bubble Tea idiom
- Small, focused change
- Main textarea benefits for free

**Cons:**
- Two files to touch instead of one
- `handleDialogCharInput` guard relaxation must be careful not to pass control sequences through

---

## Decision

**Chosen Option:** Option 3 — Enable bracketed paste at program level AND add `PasteMsg` handler in settings overlay

### Rationale

Option 1 does not solve the problem. Option 2 is a workaround for a problem that the standard library mechanism solves correctly. Option 3 is the idiomatic Bubble Tea solution and touches only two files with minimal risk.

---

## Consequences

### Positive

- API keys and other multi-character values can be pasted into settings fields on first try
- Main chat textarea paste continues to work (no change needed)
- Standard OS paste shortcuts all work: Cmd+V, Ctrl+V, middle-click, shift+insert
- Terminals without bracketed paste support fall back to current behaviour (no regression)

### Negative

- `handleDialogCharInput` guard change must be tested against all control key sequences to confirm none slip through

### Neutral

- `tea.WithBracketedPaste()` causes the terminal to receive `ESC[?2004h` on TUI launch and `ESC[?2004l` on exit — invisible to users, standard terminal behaviour

---

## Implementation

### Step 1 — Enable bracketed paste at program level (1 line)

**`pkg/executor/tui/executor.go`** — add `tea.WithBracketedPaste()` to the `tea.NewProgram()` options:

```go
p := tea.NewProgram(
    m,
    tea.WithAltScreen(),
    tea.WithMouseCellMotion(),
    tea.WithBracketedPaste(),   // add this line
)
```

This is the only change needed for the main chat textarea — the bubbles `textarea.Model` already handles `tea.PasteMsg` internally.

### Step 2 — Add PasteMsg case to settings overlay Update (1 new case, ~5 lines)

**`pkg/executor/tui/overlay/settings.go`** — in `handleDialogInput()` starting at line 1144, add a `tea.PasteMsg` case before the `tea.KeyMsg` case:

```go
case tea.PasteMsg:
    return m.handleDialogPaste(string(msg))
```

### Step 3 — Implement handleDialogPaste (~25 lines)

**`pkg/executor/tui/overlay/settings.go`** — new method on `SettingsOverlay`:

```go
// handleDialogPaste inserts pasted text into the currently focused settings field.
// Newlines are stripped — settings fields are single-line.
func (m *SettingsOverlay) handleDialogPaste(text string) (tea.Model, tea.Cmd) {
    // Strip newlines, carriage returns, and tabs — all settings fields are
    // single-line values (API keys, model names, base URLs). Password managers
    // sometimes include trailing tabs or newlines in copied secrets.
    text = strings.Map(func(r rune) rune {
        if r == '\n' || r == '\r' || r == '\t' {
            return -1
        }
        return r
    }, text)

    if text == "" {
        return m, nil
    }

    // Delegate to the same field-insertion logic used by single-char input
    // by calling handleDialogCharInput once per rune, OR by appending directly
    // to the active field value. Direct append is simpler and avoids per-rune
    // overhead for long pastes.
    return m.insertTextAtCursor(text)
}
```

`insertTextAtCursor` either already exists (under a different name) or is extracted from the single-char path in `handleDialogCharInput`. It appends `text` to the current field value at the cursor position and advances the cursor.

### Step 4 — Relax single-char guard in handleDialogCharInput

**`pkg/executor/tui/overlay/settings.go:1273`** — current guard:

```go
if len(keyMsg.String()) == 1 {
    // insert character
}
```

This guard exists to avoid passing control sequences (e.g. `ctrl+c`, `alt+x`) to the field. The correct guard is to check that all runes in the string are printable, not that the length is exactly 1:

```go
if isPrintableInput(keyMsg.String()) {
    // insert character(s)
}
```

New helper function:

```go
// isPrintableInput returns true if s contains only printable runes and should
// be inserted into the active text field.
func isPrintableInput(s string) bool {
    if s == "" {
        return false
    }
    for _, r := range s {
        if !unicode.IsPrint(r) {
            return false
        }
    }
    return true
}
```

> This relaxation is actually not required for `tea.PasteMsg` (which is handled in its own case before `tea.KeyMsg`), but it is a correctness fix in its own right and prevents future confusion.

### Migration Path

No migration. `tea.WithBracketedPaste()` is additive — terminals that do not support it ignore the `ESC[?2004h` sequence and paste arrives as individual key events (same as today).

### Timeline

- **Step 1** (program level): ~5 min
- **Steps 2–3** (PasteMsg handler): ~30 min
- **Step 4** (guard relaxation): ~15 min
- **Total**: ~1 hour, single PR

---

## Validation

### Success Metrics

- Manual: open Settings, focus the API Key field, paste a 40+ character key → full key appears in the field
- Manual: paste a value containing a newline → newline is stripped, rest of value inserted
- Manual: paste a value containing a tab character (e.g. from a password manager with trailing tab) → tab is stripped, rest of value inserted
- Manual: paste into a read-only or non-editable field → no change, no crash
- Regression: existing single-character input in all settings fields continues to work
- Regression: main chat textarea paste continues to work

### Monitoring

- Watch for reports of settings fields accepting unexpected input after the guard relaxation
- Watch for reports of paste not working on specific terminal emulators (indicates no bracketed paste support — document as known limitation)

---

## Related Decisions

- [ADR-0017](0017-auto-approval-and-settings-system.md) — Settings System (settings overlay architecture)
- [ADR-0012](0012-enhanced-tui-executor.md) — Enhanced TUI Executor (program init location)
- [ADR-0050](0050-tui-clipboard-copy.md) — TUI Clipboard Copy (companion feature, same release)

---

## References

- [TUI Visual Polish PRD](../product/features/tui-visual-polish.md)
- [Design exploration](../product/scratch/tui-ux-enhancements.md)
- Bubble Tea `WithBracketedPaste`: https://pkg.go.dev/github.com/charmbracelet/bubbletea#WithBracketedPaste
- xterm bracketed paste spec: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Bracketed-Paste-Mode

---

## Notes

The `handleDialogInput` switch in `settings.go` dispatches on message type. The `tea.PasteMsg` case must appear **before** the `tea.KeyMsg` case to ensure paste events are handled before the key fallthrough logic runs.

`strings` and `unicode` packages are already imported in `settings.go` — no new imports needed.

**Last Updated:** 2025-01-26
