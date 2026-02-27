# 49. TUI Bracketed Paste Support

**Status:** Implemented
**Date:** 2025-01-26
**Deciders:** Product Team, Engineering Team
**Technical Story:** [TUI Visual Polish PRD](../product/features/tui-visual-polish.md) — paste in settings use case

---

## Context

Paste does not work in the Forge TUI settings dialogs. When a user copies a 51-character API key from their password manager and pastes it into a settings input field, the paste is silently discarded. The user is forced to type the key character by character.

There is one root cause:

1. **Settings overlay level**: `handleDialogCharInput()` in `pkg/executor/tui/overlay/settings.go` only accepts input where `len(keyMsg.String()) == 1`. Multi-character paste events are silently dropped. Additionally, there is no `keyMsg.Paste` branch in `handleDialogInput()`, so pasted content falls through to the normal key-switch and is discarded.

The main chat textarea is a `charmbracelet/bubbles/textarea` component which handles pasted `KeyMsg` events natively — no change needed there.

### Background

Bracketed paste is a terminal feature (xterm extension, widely supported) where the terminal wraps pasted text with `ESC[200~` ... `ESC[201~`. When Bubble Tea v1.3.10 receives these sequences, it sets `Paste: true` on the `tea.KeyMsg` and populates `keyMsg.Runes` with the full pasted content — delivering it as a single event rather than a rapid burst of individual key events.

**Important API note for Bubble Tea v1.3.10:** Bracketed paste is **enabled by default**. There is no exported `tea.PasteMsg` type and no `tea.WithBracketedPaste()` option. The only available option is `tea.WithoutBracketedPaste()` to disable it. Pasted text arrives exclusively as `tea.KeyMsg{Paste: true, Type: KeyRunes}` with pasted runes in `keyMsg.Runes`.

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
* **Implementation size**: One file, ~30 lines total — very low risk
* **Library support**: Bracketed paste is on by default in Bubble Tea v1.3.10; paste arrives as `tea.KeyMsg{Paste: true}` with no additional program setup required
* **Zero regression**: Terminals without bracketed paste fall back to existing behaviour

---

## Considered Options

### Option 1: Enable bracketed paste at program level only

**Description:** Add a program-level option to ensure bracketed paste is active. Rely on the bubbles textarea to handle paste in the main input. Do nothing about the settings overlay.

**Pros:**
- Minimal change
- Fixes main textarea paste (already works without any change in v1.3.10)

**Cons:**
- Settings dialogs still cannot accept paste — the original problem is unsolved
- Pasted `KeyMsg{Paste: true}` delivered to the settings overlay `Update()` would have no handler, effectively discarding it

### Option 2: Handle paste in settings overlay via OS clipboard read (REJECTED)

**Description:** Handle `Ctrl+V` in the settings overlay by reading from the OS clipboard directly via `atotto/clipboard`.

**Pros:**
- Works even when the terminal does not support bracketed paste

**Cons:**
- Requires clipboard read permission / daemon (not available in all environments)
- Does not fix middle-click or shift+insert paste (which use bracketed paste, not Ctrl+V)
- Adds a clipboard dependency to the settings overlay
- Inconsistent with how every other Bubble Tea application handles paste

### Option 3: Check `keyMsg.Paste` in settings overlay AND relax the char-input guard (CHOSEN)

**Description:** In `handleDialogInput()`, check `keyMsg.Paste` on the incoming `tea.KeyMsg` before the key-switch and route to a new `handleDialogPaste()` method. Also relax the `len(keyMsg.String()) == 1` guard in `handleDialogCharInput()` to use `isPrintableInput()` so multi-rune inputs are not silently dropped.

**Pros:**
- Fixes all paste mechanisms (Cmd+V, middle-click, shift+insert) for settings input
- Consistent with Bubble Tea v1 idiom (`KeyMsg.Paste` field)
- Small, focused change — one file
- Main textarea benefits for free (no change needed)

**Cons:**
- `handleDialogCharInput` guard relaxation must be careful not to pass control sequences through

---

## Decision

**Chosen Option:** Option 3 — Check `keyMsg.Paste` in settings overlay AND relax the char-input guard

### Rationale

Option 1 does not solve the problem. Option 2 is a workaround for a problem that the standard library mechanism solves correctly. Option 3 is the idiomatic Bubble Tea v1 solution and touches only one file with minimal risk.

---

## Consequences

### Positive

- API keys and other multi-character values can be pasted into settings fields on first try
- Main chat textarea paste continues to work (no change needed)
- Standard OS paste shortcuts all work: Cmd+V, middle-click, shift+insert
- Terminals without bracketed paste support fall back to current behaviour (no regression)

### Negative

- `handleDialogCharInput` guard change was validated against control key sequences via `isPrintableInput()` unit tests to confirm none slip through

### Neutral

- Bubble Tea v1.3.10 enables bracketed paste by default; the terminal receives `ESC[?2004h` on TUI launch and `ESC[?2004l` on exit — invisible to users, standard terminal behaviour

---

## Implementation

### Step 1 — Bracketed paste at program level

In **Bubble Tea v1.3.10**, bracketed paste is **enabled by default**. There is no `tea.WithBracketedPaste()` option — the only available option is `tea.WithoutBracketedPaste()` to disable it. Paste events arrive as `tea.KeyMsg{Paste: true, Type: KeyRunes}` with the pasted runes in `keyMsg.Runes`.

**No change required in `pkg/executor/tui/executor.go`.**

The main chat textarea (`charmbracelet/bubbles/textarea`) receives pasted content via the `KeyMsg.Paste` path and handles it natively — no change needed there either.

### Step 2 — Check `keyMsg.Paste` in settings overlay (~5 lines)

**`pkg/executor/tui/overlay/settings.go`** — in `handleDialogInput()`, add a `keyMsg.Paste` check **before** the key-switch so pasted content is never routed to the normal key dispatch:

```go
// In Bubble Tea v1, bracketed paste arrives as a KeyMsg with Paste: true
// and Type: KeyRunes. Handle it before the key-switch so pasted content
// is never routed to the normal key dispatch.
if keyMsg.Paste {
    return s.handleDialogPaste(string(keyMsg.Runes))
}
```

### Step 3 — Implement handleDialogPaste (~25 lines)

**`pkg/executor/tui/overlay/settings.go`** — new method on `SettingsOverlay`:

```go
// handleDialogPaste inserts pasted text into the currently focused settings field.
// Newlines, carriage returns and tabs are stripped — all settings fields are
// single-line values (API keys, model names, base URLs). Password managers
// sometimes append a trailing newline or tab to copied secrets.
func (s *SettingsOverlay) handleDialogPaste(text string) (types.Overlay, tea.Cmd) {
    // Strip control characters: \n, \r, \t — all settings fields are single-line.
    text = strings.Map(func(r rune) rune {
        if r == '\n' || r == '\r' || r == '\t' {
            return -1
        }
        return r
    }, text)

    if text == "" {
        return s, nil
    }

    field := &s.activeDialog.fields[s.activeDialog.selectedField]
    if field.fieldType == fieldTypeText || field.fieldType == fieldTypePassword {
        if field.maxLength == 0 {
            field.value += text
        } else {
            remaining := field.maxLength - len(field.value)
            if remaining > 0 {
                runes := []rune(text)
                if len(runes) > remaining {
                    runes = runes[:remaining]
                }
                field.value += string(runes)
            }
        }
        field.errorMsg = ""
    }

    return s, nil
}
```

### Step 4 — Relax single-char guard in handleDialogCharInput

**`pkg/executor/tui/overlay/settings.go`** — replace:

```go
if len(keyMsg.String()) == 1 {
```

with:

```go
if isPrintableInput(keyMsg.String()) {
```

New helper function:

```go
// isPrintableInput returns true if s contains only printable runes and should
// be inserted into the active text field. Rejects empty strings and any
// control sequences (ctrl+c, alt+x, etc.).
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

> This relaxation is not strictly required for the paste path (which is handled before the key-switch in Step 2), but it is a correctness fix in its own right and prevents future confusion.

### Migration Path

No migration. Bracketed paste was already enabled by default in Bubble Tea v1.3.10; this change only adds correct handling of the `KeyMsg.Paste` path in the settings overlay.

### Timeline

- **Steps 2–3** (paste handler): ~30 min
- **Step 4** (guard relaxation + `isPrintableInput`): ~15 min
- **Total**: ~45 min, single PR

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
- Bubble Tea `KeyMsg.Paste` field: https://pkg.go.dev/github.com/charmbracelet/bubbletea#KeyMsg
- xterm bracketed paste spec: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Bracketed-Paste-Mode

---

## Notes

In Bubble Tea v1.3.10, there is no exported `tea.PasteMsg` type. The `keyMsg.Paste` check in `handleDialogInput` must appear **before** the key-switch so pasted content is never dispatched to the normal key handlers.

`strings` and `unicode` packages are already imported in `settings.go` — no new imports needed.

**Last Updated:** 2025-06-09
