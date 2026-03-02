# TUI Reskin Implementation Gaps

This document tracks gaps between the ADR/PRD specification and the current implementation in `pkg/executor/tui/`.

---

## Gap 1 — Compact Header Not Implemented

**Spec:** ADR-0051 Step 1, PRD §Header Design
**Files changed:** `pkg/executor/tui/view.go` (`buildHeader()`), `pkg/executor/tui/update.go` (`headerHeight`), `pkg/version/version.go` (new package)

**Status: ✅ Resolved**

`buildHeader()` renders the compact 2-line header bar (brand left · cwd centre · model+version right, followed by a `inputRuleStyle` separator). The ASCII art call is gone. `headerHeight = 4` in `calculateViewportHeight()` accounts for: bar (1) + separator (1) + tips (1) + blank spacer (1).

The `version.Version` constant lives in the new `pkg/version` package (`pkg/version/version.go`). `view.go` imports it.

---

## Gap 2 — Input Box Still Uses 4-Sided Border

**Spec:** ADR-0051 Step 2, PRD §Input Design (Option B)
**Files changed:** `pkg/executor/tui/view.go` (`buildInputBox()`), `pkg/executor/tui/styles.go`

**Status: ✅ Resolved**

`buildInputBox()` renders a top-rule-only design:

```
────────────────────────────────────────────────────────────────────────────────
❯ <textarea>
```

The rule uses `inputRuleStyle` (colored with `dimSep = #374151`). `bashMode` switches the prompt glyph to `bashPromptStyle` (mint green).

---

## Gap 3 — Bottom Bar Padding Uses `len()` Instead of `lipgloss.Width()`

**Spec:** ADR-0051 Step 3, PRD §Bottom Bar
**Files changed:** `pkg/executor/tui/view.go` (`buildBottomBar()`)

**Status: ✅ Resolved**

`buildBottomBar()` uses `lipgloss.Width(left)` and `lipgloss.Width(right)` for gap calculation (line 162). No `len()` calls on styled strings remain in either `buildBottomBar()` or `buildTokenDisplay()`.

---

## Gap 4 — Static Tips Line

**Spec:** ADR-0051 Step 4
**Files changed:** `pkg/executor/tui/view.go` (`buildTips()`)

**Status: ✅ Resolved**

`buildTips()` is state-aware with four branches:

- `m.overlay.isActive()` → `Esc · close   Tab · next field   Enter · confirm`
- `m.agentBusy` → `Ctrl+C · interrupt` (+ `G · follow output` when scroll is locked)
- `m.bashMode` → `Enter · run   exit · return to normal   Ctrl+C · cancel`
- default (idle) → `Enter · send   Alt+Enter · new line   / · commands   Ctrl+Y · copy   Ctrl+C · exit`

---

## Gap 5 — Viewport Height Not Dynamic for Multiline Input

**Spec:** ADR-0051 Step 2b
**Files changed:** `pkg/executor/tui/update.go` (`calculateViewportHeight()`, textarea update branch)

**Status: ✅ Resolved**

`calculateViewportHeight()` uses `strings.Count(m.textarea.Value(), "\n") + 1` for `liveLines`, not `m.textarea.Height()`. The textarea update branch compares `oldLines`/`newLines` (via `strings.Count`) and calls `m.recalculateLayout()` on change.

---

## Gap 6 — Style Constants Missing / Unverified

**Spec:** ADR-0051 Step 5
**Files changed:** `pkg/executor/tui/styles.go`

**Status: ✅ Resolved**

`dimSep = lipgloss.Color("#374151")` is declared in `styles.go`. `inputRuleStyle` uses `dimSep`. `inputPromptStyle` uses `salmonPink` with `Bold(true)`. All ADR-referenced style constants are present.

> **Note:** The color constant is `dimSep` in code, not `dimSeparator` as the ADR originally stated. The ADR has been corrected.

---

## Gap 7 — `workingDir` Field Missing from Model

**Spec:** ADR-0051 Step 1 (`buildHeader()` uses `m.workingDir`)
**Files checked:** `pkg/executor/tui/model.go`, `pkg/executor/tui/view.go`

**Status: ✅ Resolved (no change needed)**

The model field is `workspaceDir` (not `workingDir`). `buildHeader()` correctly uses `m.workspaceDir`. The ADR code snippet has been corrected to use `m.workspaceDir`.

---

## Gap 8 — `modelName` / `version` Fields Missing or Unverified

**Spec:** ADR-0051 Step 1 (`buildHeader()` uses `m.modelName` and `version.Version`)
**Files changed:** `pkg/version/version.go` (new), `pkg/executor/tui/view.go` (import added)

**Status: ✅ Resolved**

There is no `m.modelName` model field — `buildHeader()` computes the model name locally via `m.provider.GetModel()`. `version.Version` is imported from `pkg/version`. The ADR code snippet has been corrected to show the actual implementation pattern.

---

## Gap 9 — `m.styles` Not Used in Some Render Functions

**Spec:** ADR-0051 Step 1 references `m.styles.brandStyle`, `m.styles.mutedStyle`, `m.styles.dimStyle`
**Files checked:** `pkg/executor/tui/model.go`, `pkg/executor/tui/styles.go`

**Status: ✅ Resolved (no change needed)**

There is no `m.styles` struct. All styles are package-level vars in `styles.go`. `buildHeader()` uses `headerStyle` (package-level) instead of `m.styles.brandStyle`. The ADR code snippet has been corrected to use package-level vars.

---

## Gap 10 — `followScroll` Field Name May Differ

**Spec:** ADR-0051 Step 4 references `m.followScroll`
**Files checked:** `pkg/executor/tui/model.go`

**Status: ✅ Resolved (no change needed)**

`m.followScroll` is the correct field name (model.go line 86). The ADR field reference is correct.

---

## Gap 11 — `recalculateLayout()` May Not Exist

**Spec:** ADR-0051 Step 2b calls `m.recalculateLayout()`
**Files checked:** `pkg/executor/tui/update.go`

**Status: ✅ Resolved (no change needed)**

`recalculateLayout()` exists at `update.go:743`. It calls `calculateViewportHeight()`, updates `m.viewport.Height`, and syncs viewport content.

---

## Summary Table

| Gap | File(s) | Status |
|-----|---------|--------|
| 1 — Compact header | view.go, update.go, pkg/version/version.go | ✅ Resolved |
| 2 — Option B input box | view.go, styles.go | ✅ Resolved |
| 3 — Bottom bar ANSI fix | view.go | ✅ Resolved |
| 4 — Contextual tips line | view.go | ✅ Resolved |
| 5 — Dynamic viewport height | update.go | ✅ Resolved |
| 6 — Style constants | styles.go | ✅ Resolved |
| 7 — workingDir field | model.go (no change) | ✅ Resolved |
| 8 — modelName / version fields | pkg/version/version.go, view.go | ✅ Resolved |
| 9 — m.styles struct | model.go (no change) | ✅ Resolved |
| 10 — followScroll field name | model.go (no change) | ✅ Resolved |
| 11 — recalculateLayout() | update.go (no change) | ✅ Resolved |
