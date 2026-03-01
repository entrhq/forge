# TUI v2 Implementation Review

I've reviewed the current TUI implementation against the `docs/product/scratch/tui-v2.md` design spec and ADR-0051. Here is the gap analysis showing what has been completed and what outstanding recommendations we still need to take on:

### ✅ Completed Items
- **Bug 1 (ANSI-unsafe width):** `buildBottomBar` uses `lipgloss.Width()` for gap calculation (not `len()`).
- **Bug 2 (Hardcoded Header Height):** `calculateViewportHeight` uses `const headerHeight = 4`, reflecting the actual 4-line chrome (bar + separator + tips + blank spacer). The old hardcoded `10` is gone.
- **Bug 3 (Resize Short-circuit):** `handleWindowResize` (`update.go`) now always sets `m.width` and `m.height` unconditionally (lines 373–374) before the `resultList.IsActive()` branch. The early-return bug is fixed.
- **Bug 4 (Unicode-unsafe layout):** `wordWrap()` and `updateTextAreaHeight()` in `helpers.go` both use `lipgloss.Width()` for all visual-width measurements. No `len()` calls remain on string content.
- **Bug 6 (Hardcoded Init Dimensions):** `init.go` now calls `viewport.New(0, 0)`, deferring actual sizing until the first `tea.WindowSizeMsg`.
- **Bug 7 (Weak Cache ID):** `events.go` now preferentially uses `event.ToolCallID` when the upstream event carries one; the `time.Now().UnixNano()` fallback only fires for synthetic/out-of-band tool events and is documented as such.
- **Compact header (ADR-0051 Gap 1):** `buildHeader()` renders a 2-line compact bar (brand · cwd · model+version, then a `dimSep` separator rule). ASCII art is gone. `pkg/version` package created.
- **Option B input box (ADR-0051 Gap 2):** `buildInputBox()` renders top-rule-only design (rule + `❯` prompt glyph + textarea). `inputRuleStyle` and `dimSep` (#374151) added to `styles.go`.
- **Contextual tips (ADR-0051 Gap 4):** `buildTips()` is state-aware: overlay-open / agent-busy / bash-mode / idle branches.
- **Dynamic viewport height (ADR-0051 Gap 5):** `calculateViewportHeight()` uses `strings.Count(m.textarea.Value(), "\n") + 1` for live line count. Textarea update path detects line-count changes and calls `recalculateLayout()`.

### 🔴 Outstanding Critical Bugs
- **Bug 5 (Unbounded Memory Growth):** `m.content`, `m.thinkingBuffer`, and `m.messageBuffer` in `model.go` remain raw `*strings.Builder` pointers (`init.go` lines 52–54). They have not been replaced with a `[]ConversationMessage` structured slice. Long sessions will grow these buffers indefinitely.

### 🟡 Outstanding Architectural & UX Debt
- **Monolithic Message Loop:** `update.go` is still a single large file (650+ lines). The `tui-v2.md` mandate to decompose into `handlers/` and `components/` sub-packages is pending.
- **Dual Type Definitions:** Internal `toastMsg` struct coexists with external `tuitypes.ToastMsg`, requiring conversion boilerplate.
- **No Semantic Theme System:** Colors are declared as raw `lipgloss.Color` vars in `styles.go`. A holistic `theme/` package with semantic tokens (Primary, Secondary, FgMuted, Border, etc.) was not implemented.
- **Missing Bracketed Paste in Main TUI:** `tea.EnableBracketedPaste` is not returned from `Init()` in the main model, and `tea.PasteMsg` is not handled in `Update()`. (The settings overlay has bracketed paste tests, but the main input path does not support it.)
- **No Input History:** Up/Down arrow key command history cycling for the input prompt is missing from the state model and key handler.
