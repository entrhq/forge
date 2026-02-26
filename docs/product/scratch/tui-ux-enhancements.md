# TUI UX Enhancements

## Overview

Four targeted improvements to the existing TUI that fix real usability pain points and meaningfully raise the visual quality without a full rewrite. Each can be shipped independently.

---

## 1. Visual Redesign — Compact Header & Cleaner Layout

### Problem

The current header is 6 lines of block-character ASCII art plus a tips line and a working-directory status line. That burns ~10 lines before the conversation even starts (`headerHeight := 10` is literally hardcoded in `calculateViewportHeight()`). On a typical 24-line terminal you have roughly 10 lines of conversation viewport. The tips text is a single wall of dense instructions that's hard to scan.

The bottom bar uses `len()` on strings that contain ANSI escape sequences — because `statusBarStyle` colouring adds invisible bytes, the padding math is wrong, causing the three columns to misalign.

### Proposed Design

**Replace the 6-line ASCII art header with a single-line header bar** that spans the terminal width:

```
  ⬡ forge                      Working directory: ~/projects/myapp           v0.x.x
```

- Left: product glyph + name in brand colour (salmon/coral)
- Centre: cwd (truncated with `…` if too long)
- Right: version string, muted
- Total height: 1 line (was 8–10 lines)

**Separate the tips into a slim contextual line** that only appears when relevant:

- Idle, no input: `  Enter to send  ·  Alt+Enter new line  ·  / commands  ·  Ctrl+C exit`
- Agent busy: `  Ctrl+C to interrupt  ·  Scroll to browse history`
- Bash mode: `  Commands run directly  ·  exit or Ctrl+C to return`

Both lines together are 2 lines (header + tips). Header height drops from 10 → 3 (1 header + 1 tips + 1 blank separator). The viewport gains ~7 lines on every launch.

**Fix the bottom bar padding** by replacing `len(s)` with `lipgloss.Width(s)` on all three columns. This correctly ignores ANSI escape bytes and produces accurate column widths.

**Replace the input box with a top-rule-only design (Option B)** — the current input box uses `RoundedBorder()` with a full enclosing salmon-pink border on all four sides. This is visually heavy and draws too much attention away from the conversation.

Replace the entire box with a single hairline rule above the input field. The cursor floats in open space below the rule. No bottom border, no side borders, no box.

```
  ──────────────────────────────────────────────────────────────────────────────
  ❯ _
```

- Rule colour: [muted dim] — subtle, not branded
- `❯` prompt prefix: [salmon] — the only accent in the input zone
- When the agent is busy and input is disabled, the rule dims further and `❯` disappears
- Hint line sits one blank line below the cursor

This decision is reflected across all 10 layout mockups in `tui-mockups.md`.

### Affected Files

| File | Change |
|---|---|
| `pkg/executor/tui/view.go` | Replace `buildHeader()` with single-line bar; simplify `buildTips()`; fix `buildBottomBar()` padding with `lipgloss.Width()` |
| `pkg/executor/tui/update.go` | Change `calculateViewportHeight()` — `headerHeight := 10` → `headerHeight := 3` |
| `pkg/executor/tui/styles.go` | Add `headerBarStyle`; replace `inputBoxStyle` (4-sided border) with `inputRuleStyle` (top rule only) |
| `pkg/ui/ascii.go` | No change required (ASCII art generation kept, just not used in main header) |

---

## 2. Smart Scroll-Lock — Don't Jump When Reading History

### Problem

Every time the agent streams a new token, `m.viewport.GotoBottom()` is called unconditionally. There are **16+ call sites** across `events.go` and `update.go` that all do this. If a user scrolls up to re-read earlier output while the agent is mid-response, the viewport immediately snaps back to the bottom — making it impossible to read anything while the agent is running.

The pattern to fix this already exists in the codebase — `pkg/executor/tui/overlay/command.go` lines 136–138 uses an `AtBottom()` guard before calling `GotoBottom()`:

```go
if vp.AtBottom() {
    vp.GotoBottom()
}
```

The main viewport just needs the same treatment, extended with user-intent tracking.

### Proposed Behaviour

- When the user presses `PageUp`, `Up`, or scrolls the mouse wheel up: set `followScroll = false`
- When the user presses `G` (jump-to-bottom), `PageDown` past the last line, or sends a new message: set `followScroll = true`
- Replace every unconditional `m.viewport.GotoBottom()` with:
  ```go
  if m.followScroll {
      m.viewport.GotoBottom()
  }
  ```
- When `followScroll` is `false` and new content arrives, render a small indicator at the bottom of the viewport:
  ```
  ↓  New activity — press G to jump to bottom
  ```
  This is drawn as a `renderToastOverlay` row, not inside the viewport content, so it doesn't pollute the conversation buffer.

### Implementation Notes

**Model change** — add one field to `model` struct in `model.go`:
```go
followScroll bool  // true = auto-follow new content; false = user scrolled up
```
Initialise to `true` in `init.go`.

**Detect user scroll-up** — in `Update()`, catch `tea.KeyMsg` for `pgup`/`up` and `tea.MouseMsg` for `MouseWheelUp` before the viewport update, set `m.followScroll = false`.

**Detect return-to-bottom** — in `handleKeyPress()`, add `g` / `G` / `end` cases that set `m.followScroll = true` then call `m.viewport.GotoBottom()`. Also reset in `handleAgentMessage()` send path when `m.agentBusy` goes false.

**Guard all GotoBottom calls** — there are 16 call sites to update:

| File | Lines | Context |
|---|---|---|
| `events.go` | 106, 125, 126, 150, 224, 275, 289, 355 | Streaming content handlers |
| `update.go` | 390, 407, 418, 432, 599, 639, 656, 675 | Key handlers + `recalculateLayout()` |

**New indicator** — add `m.hasNewContentWhileScrolled bool` field, set it true when `!m.followScroll && newContent`, reset when `followScroll` becomes true. `applyOverlays()` renders the indicator if this flag is set.

### Affected Files

| File | Change |
|---|---|
| `pkg/executor/tui/model.go` | Add `followScroll bool`, `hasNewContentWhileScrolled bool` |
| `pkg/executor/tui/init.go` | Set `followScroll: true` in `initialModel()` |
| `pkg/executor/tui/update.go` | Detect scroll-up keys/mouse; add `G` handler; guard `recalculateLayout()` call; reset on send |
| `pkg/executor/tui/events.go` | Guard all 8 `GotoBottom()` calls |
| `pkg/executor/tui/view.go` | Render "↓ New activity" indicator in `applyOverlays()` |

---

## 3. Paste Into Settings Dialogs

### Problem

The settings overlay uses a key-by-key character input handler (`handleDialogCharInput` in `overlay/settings.go` line 1273). It only accepts input when `len(keyMsg.String()) == 1` — single characters. When you paste from the clipboard, the terminal sends a burst of characters that arrives either as a bracketed paste sequence or as rapid key events. Both are silently discarded because the length check fails for anything more than one character.

There are two fixes needed:

1. **Enable bracketed paste mode** so the terminal wraps paste content in `ESC[?2004h` brackets and Bubble Tea delivers it as a structured `tea.PasteMsg` (rather than raw key spam).
2. **Handle `tea.PasteMsg` in the settings dialog** by inserting the pasted string into the active input field's `value`.

### Implementation

**Enable bracketed paste** — add `tea.WithBracketedPaste()` as a program option in `executor.go`:
```go
e.program = tea.NewProgram(
    &m,
    tea.WithAltScreen(),
    tea.WithMouseCellMotion(),
    tea.WithBracketedPaste(),
)
```

Bubble Tea v1.3.10 (the current version) supports `tea.WithBracketedPaste()` and delivers `tea.PasteMsg` events.

**Handle PasteMsg in the overlay dispatch** — in `update.go` the early overlay forwarding block already routes all messages to `m.overlay.overlay.Update(msg)`. The settings overlay's `Update()` method then calls `handleDialogInput()`.

Add a `tea.PasteMsg` case to `handleDialogInput()` in `overlay/settings.go` (around line 1144 where the `tea.KeyMsg` switch lives):
```go
case tea.PasteMsg:
    if m.activeDialog != nil {
        m.handleDialogPaste(string(msg))
        return m, nil
    }
```

Add `handleDialogPaste(text string)` that appends the pasted string to the active input field value with the same validation as `handleDialogCharInput` (max length, allowed characters per field type).

**Also fix the main input textarea** — the main `textarea` component in `bubbles` already supports bracketed paste natively once `tea.WithBracketedPaste()` is enabled. No additional work needed for the main chat input.

### Affected Files

| File | Change |
|---|---|
| `pkg/executor/tui/executor.go` | Add `tea.WithBracketedPaste()` to `tea.NewProgram` options |
| `pkg/executor/tui/overlay/settings.go` | Add `tea.PasteMsg` case to `handleDialogInput()`; add `handleDialogPaste()` method |

---

## 4. Copy Content From the Terminal

### Problem

Two things prevent terminal-native copy/paste from working:

1. **`tea.WithAltScreen()`** puts the TUI in the terminal's alternate buffer. The alternate buffer is separate from the scrollback buffer — when you exit the TUI, the session disappears and there's nothing to select from. While the TUI is running, the OS can technically let you select text, but most terminals don't support alt-screen selection well.

2. **`tea.WithMouseCellMotion()`** enables Bubble Tea's mouse event capture using `DECSET 1002`. This tells the terminal to send all mouse button presses and drags as escape sequences to the application. When mouse capture is active, most terminals stop doing native text selection on click-drag because the application is consuming the events instead.

So any drag-to-select action either doesn't start (mouse captured), or selects the right text but immediately clears the selection when you release (some terminals reset on focus change in alt-screen).

### Proposed Solutions

Two complementary mechanisms that solve different use cases:

#### 4a. Ctrl+Y — Copy Viewport to Clipboard

`Ctrl+Y` copies the entire visible conversation (the `m.content` buffer) to the system clipboard. This is the quickest path to getting content out of the TUI without needing mouse selection to work.

The `atotto/clipboard` package is already used elsewhere in the codebase. Implementation:
- Catch `ctrl+y` in `handleKeyPress()` in `update.go`
- Call `clipboard.WriteAll(m.content.String())`
- Show a 2-second toast: "✓ Conversation copied to clipboard"

This works without any changes to alt-screen or mouse mode.

#### 4b. Ctrl+Y with Suspend — Temporary Mouse Release for Terminal Selection

A more powerful variant: `Ctrl+Y` calls `tea.Suspend()` to temporarily hand control back to the terminal (suspends the TUI, restores normal terminal mode). The user can then use the terminal's native selection and copy. When they press any key, `tea.Resume()` is called and the TUI resumes.

Show a message in the suspended state (printed to stdout before suspend):
```
  TUI suspended — select and copy text normally, then press any key to resume
```

This solves the selection problem completely because it exits mouse-capture mode and restores the primary screen buffer.

**Note:** `tea.Suspend()` / `tea.Resume()` are available in Bubble Tea v1.x. This requires adding a `tea.KeyCtrlY` handler that returns `tea.Suspend()` as a command, and handling `tea.ResumeMsg` to restore state.

#### 4c. Export to Pager — Ctrl+E

`Ctrl+E` pipes `m.content.String()` to `$PAGER` (defaulting to `less`). This opens the full conversation in the system pager where the user can search, scroll, and select freely. The TUI is suspended while the pager runs, then resumes.

Implementation uses `tea.ExecProcess()` which is the Bubble Tea-idiomatic way to shell out to an external process and resume afterwards.

### Recommendation

Implement **4a** (clipboard copy) as the fast path and **4b** (suspend for terminal selection) as the power-user path. Defer 4c unless there's demand.

### Affected Files

| File | Change |
|---|---|
| `pkg/executor/tui/update.go` | Add `ctrl+y` handler in `handleKeyPress()`; handle `tea.ResumeMsg` |
| `pkg/executor/tui/view.go` | Update tips/bottom bar to show Ctrl+Y hint |

---

## Priority & Effort Estimate

| Feature | User Impact | Effort | Priority |
|---|---|---|---|
| 2. Smart scroll-lock | High — affects every multi-turn session | Medium (16 call sites, 2 new fields) | **P0** |
| 3. Paste into settings | High — critical for API key entry | Low (2 files, ~30 lines) | **P0** |
| 4a. Copy to clipboard | Medium — common need | Low (~15 lines) | **P1** |
| 1. Visual redesign | Medium — first impression quality | Medium (3 files, new styles) | **P1** |
| 4b. Suspend for selection | Low — power users | Low (~20 lines) | **P2** |

---

## Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Input box style | **Option B — top rule only** | Enclosing box draws too much attention; open rule keeps focus on conversation |

---

## Non-Goals

- Full TUI rewrite / package decomposition (tracked separately in `tui-v2.md`)
- Theme system / multiple colour schemes
- Mouse-based text selection inside the alt-screen (terminal-dependent, not reliably implementable)
- Settings UI redesign beyond the paste fix
