# TUI v2 Implementation Review

I've reviewed the current TUI implementation against the `docs/product/scratch/tui-v2.md` design spec. Here is the gap analysis showing what has been completed and what outstanding recommendations we still need to take on:

### ✅ Completed Items
- **Bug 1 (ANSI-unsafe width):** `buildBottomBar` has been fixed to use `lipgloss.Width()` rather than `len()`.
- **Bug 2 (Hardcoded Header Height):** `calculateViewportHeight` now respects the compact ADR-0051 4-line height (`headerHeight = 4`), removing the hardcoded `10`.

### 🔴 Outstanding Critical Bugs
- **Bug 3 (Resize Short-circuit):** In `update.go#L366`, `handleWindowResize` still skips updating `m.width` and `m.height` if `m.resultList.IsActive()` is true. This breaks the background layout size if the window is resized while an overlay is active.
- **Bug 4 (Unicode-unsafe layout):** Both `updateTextAreaHeight` and `wordWrap` in `helpers.go` still use byte-length counting (`len(string)`), which means multi-byte Unicode characters/emojis will break line heights and visual word wrapping.
- **Bug 5 (Unbounded Memory Growth):** `m.content`, `m.thinkingBuffer`, and `m.messageBuffer` in `model.go` remain raw `*strings.Builder` pointers. They haven't been replaced with the recommended `[]ConversationMessage` array, meaning the string will grow indefinitely during long sessions.
- **Bug 6 (Hardcoded Init Dimensions):** `init.go` still forces `viewport.New(80, 20)` synchronously at boot.
- **Bug 7 (Weak Cache ID):** `events.go` still generates `lastToolCallID` using `time.Now().UnixNano()` (wall clock), which is fragile.

### 🟡 Outstanding Architectural & UX Debt
- **Monolithic Message Loop:** `update.go` is still heavily centralized (over 650+ lines). The `tui-v2.md` mandate to break out a `handlers/` and `components/` sub-package architecture is pending.
- **Dual Type Definitions:** We still have an internal `toastMsg` struct living right beside the external `tuitypes.ToastMsg`.
- **No Semantic Theme System:** Colors are still scattered globally via `types.SalmonPink` or local `styles.go` mappings. The recommendation to build a holistic `theme/` package wasn't implemented yet.
- **Missing Copy/Paste Functionality:** `tea.EnableBracketedPaste` still has not been added to `Init()`, nor has `tea.PasteMsg` been integrated to support rich multi-line terminal pasting.
- **No Input History:** Up/Down arrow key command history cycling for the input prompt is missing from the state model.
