# 48. TUI Smart Scroll-Lock

**Status:** Implemented
**Date:** 2025-01-26
**Deciders:** Product Team, Engineering Team
**Technical Story:** [TUI Visual Polish PRD](../product/features/tui-visual-polish.md) — scroll-lock use case

---

## Context

The Forge TUI viewport unconditionally calls `m.viewport.GotoBottom()` every time the agent streams output. There are 16 such call sites across `events.go` and `update.go`. This means if a user scrolls up to re-read earlier output while the agent is still working, the viewport immediately snaps back to the bottom on the next streamed token.

This makes the TUI unusable for reading history during long agent runs — a core workflow for Forge's primary persona (the daily driver running multi-step tasks). Competitors Crush, Codex CLI, and Gemini CLI all implement smart scroll-following where user scroll action locks the viewport in place.

### Background

The pattern for doing this correctly already exists in the codebase. `pkg/executor/tui/overlay/command.go:136-138` guards its own viewport scroll with an `AtBottom()` check:

```go
if vp.AtBottom() {
    vp.GotoBottom()
}
```

The main conversation viewport does not use this guard — every call site is unconditional.

### Problem Statement

Users cannot scroll back to read earlier conversation content while the agent is actively producing output. Any upward scroll is immediately undone by the next streamed token.

### Goals

- When the user explicitly scrolls up, freeze the viewport at the current position regardless of agent activity
- When the user signals intent to return to the bottom (press `G`, scroll to bottom, or send a message), resume auto-following
- Show a non-intrusive indicator while scroll-locked so the user knows new content is arriving below
- Preserve existing auto-follow behaviour exactly when the user has not scrolled up

### Non-Goals

- Animated or pulsing new-content indicator (P2, separate work)
- Per-message scroll anchoring (future)
- Persisting scroll position across sessions

---

## Decision Drivers

* **Usability**: The snap-to-bottom behaviour actively fights the user during long agent runs
* **Consistency**: 16 unconditional call sites are inconsistent — the command overlay already does this correctly
* **Zero regression risk**: The `followScroll = true` default preserves exactly current behaviour for users who never scroll up
* **Competitive parity**: All three measured competitors implement scroll-lock

---

## Considered Options

### Option 1: AtBottom() guard on every call site (no new field)

**Description:** Replace every `m.viewport.GotoBottom()` with `if m.viewport.AtBottom() { m.viewport.GotoBottom() }`. No new model field.

**Pros:**
- No model changes
- Minimal diff

**Cons:**
- `AtBottom()` is true again immediately after a scroll if the content height equals the viewport height — unreliable on short conversations
- No way to show a "new content below" indicator because there is no explicit locked state
- Does not handle the "scrolled up by a small amount so AtBottom is false but user did not consciously scroll" case

### Option 2: followScroll bool field + explicit key handlers (CHOSEN)

**Description:** Add `followScroll bool` and `hasNewContent bool` to the model. Set `followScroll = true` at init. Set `followScroll = false` on PageUp / mouse wheel up. Reset to `true` on: `G` keypress, scroll reaching the bottom, or the user sending a new message. Guard all 16 `GotoBottom()` call sites with `if m.followScroll`. Render a one-line indicator when `!m.followScroll && m.hasNewContent`.

**Pros:**
- Explicit state — reliable regardless of content height
- Enables "new content" indicator
- Default `true` means zero behaviour change for users who never scroll
- Clean, searchable — `followScroll` is easy to grep and understand

**Cons:**
- 16 call sites to update (mechanical but must not miss any)
- Two new model fields

### Option 3: Intercept scroll events in viewport wrapper

**Description:** Wrap `viewport.Model` in a custom struct that overrides scroll methods and tracks user-initiated vs. programmatic scrolling.

**Pros:**
- Centralises the logic

**Cons:**
- bubbles viewport is not designed for wrapping — method set is large, wrapper would need to proxy everything
- More complexity than the problem warrants
- Still needs model state to drive the indicator

---

## Decision

**Chosen Option:** Option 2 — `followScroll bool` field + explicit key handlers

### Rationale

Option 2 is the direct, explicit solution. The state is clear, the default is safe, the indicator is possible, and the 16 call-site updates are mechanical. The existing pattern in `overlay/command.go` validates the approach.

---

## Consequences

### Positive

- Users can read history during long agent runs without viewport fighting them
- Indicator tells users new content is waiting — no confusion about whether the agent is still working
- Zero behaviour change for users who never scroll up
- Establishes correct scroll pattern for future viewport components

### Negative

- 16 call sites must be updated — missing one would leave a scroll-jump
- Two new model fields add minor complexity to the model struct

### Neutral

- `hasNewContent` is set on every streaming event; only rendered when `!m.followScroll`
- The `G` keybinding (jump to bottom) is the standard for terminal pagers (less, vim) — reusing it is consistent

---

## Implementation

### Phase 1 — Model and init (1 file, ~5 lines)

**`pkg/executor/tui/model.go`** — add two fields to the `model` struct after `shouldQuit`:

```go
followScroll    bool // true = auto-follow agent output; false = user has scrolled up
hasNewContent   bool // true = new content arrived while scroll-locked
```

**`pkg/executor/tui/init.go`** — initialise in model literal:

```go
followScroll:  true,
hasNewContent: false,
```

### Phase 2 — Key handlers (1 file, ~20 lines)

**`pkg/executor/tui/update.go`** — in the `tea.KeyMsg` switch inside `handleKeyPress`:

```go
case "pgup", "ctrl+b":
    m.followScroll = false
    m.viewport.HalfViewUp()

case "g":
    if !m.followScroll {
        m.followScroll = true
        m.hasNewContent = false
        m.viewport.GotoBottom()
    }
```

Mouse wheel up handler (in `tea.MouseMsg` branch):

```go
case tea.MouseWheelUp:
    m.followScroll = false
    m.viewport.LineUp(3)
```

Scroll reaching the bottom (add to mouse wheel down and PageDown handlers):

```go
if m.viewport.AtBottom() {
    m.followScroll = true
    m.hasNewContent = false
}
```

Send message handler — reset follow on message send (already in `handleEnterKey` or similar):

```go
m.followScroll = true
m.hasNewContent = false
m.viewport.GotoBottom()
```

### Phase 3 — Guard all GotoBottom call sites (2 files, 16 edits)

Wrap every `m.viewport.GotoBottom()` with:

```go
if m.followScroll {
    m.viewport.GotoBottom()
} else {
    m.hasNewContent = true
}
```

**`pkg/executor/tui/events.go`** — lines 106, 125, 126, 150, 224, 275, 289, 355 (8 call sites)

**`pkg/executor/tui/update.go`** — lines 390, 407, 418, 432, 599, 639, 656, 675 (8 call sites)

> Note: line numbers are approximate; use `grep -n 'GotoBottom' pkg/executor/tui/events.go pkg/executor/tui/update.go` to confirm all sites before editing.

### Phase 4 — New-content indicator (1 file, ~15 lines)

**`pkg/executor/tui/view.go`** — in `assembleBaseView()`, after the viewport render, prepend an indicator row when locked:

```go
if !m.followScroll && m.hasNewContent {
    indicator := m.styles.mutedStyle.Render("  ↓  New activity below · press G to follow")
    viewportContent = indicator + "\n" + viewportContent
}
```

Or render it as a sticky row between the viewport and the input zone (preferred — does not consume a viewport line):

```go
var newContentBar string
if !m.followScroll && m.hasNewContent {
    newContentBar = m.styles.mutedStyle.Render(
        "  ↓  New activity · G to follow  ",
    ) + "\n"
}
```

Insert `newContentBar` between the viewport and the input section in the final `lipgloss.JoinVertical` call.

### Migration Path

No migration needed. `followScroll: true` at init preserves current behaviour exactly. The change is invisible to users who never scroll up.

### Timeline

- **Phase 1** (model + init): ~15 min
- **Phase 2** (key handlers): ~30 min
- **Phase 3** (guard 16 call sites): ~45 min — mechanical but careful
- **Phase 4** (indicator): ~30 min
- **Total**: ~2 hours, single PR

---

## Validation

### Success Metrics

- Manual: scroll up during a streaming agent response → viewport does not move → indicator appears → press G → viewport jumps to bottom → indicator gone
- Manual: send a new message while scroll-locked → viewport jumps to bottom, lock clears
- Regression: scroll behaviour when user does NOT scroll up is identical to pre-change

### Monitoring

- Watch for reports of viewport not following after the change (would indicate a missed call site)
- Watch for reports of indicator appearing when it should not

---

## Related Decisions

- [ADR-0009](0009-tui-executor-design.md) — TUI Executor Design (original viewport architecture)
- [ADR-0012](0012-enhanced-tui-executor.md) — Enhanced TUI Executor
- [ADR-0051](0051-tui-visual-redesign.md) — TUI Visual Redesign (companion change, same PR or adjacent)

---

## References

- [TUI Visual Polish PRD](../product/features/tui-visual-polish.md)
- [Design exploration](../product/scratch/tui-ux-enhancements.md)
- Existing correct pattern: `pkg/executor/tui/overlay/command.go:136-138`

---

## Notes

The 16 `GotoBottom()` call sites were identified by `grep -n 'GotoBottom' pkg/executor/tui/events.go pkg/executor/tui/update.go`. Confirm the count before implementing — new call sites may have been added since this ADR was written.

**Last Updated:** 2025-01-26
