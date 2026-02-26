# TUI Visual Polish & Usability Fixes

## Product Vision

The Forge TUI is the primary interface for developers who want a rich, interactive AI coding session in the terminal. Today it is functional but falls short of the quality bar set by the best terminal tools on the market — it has a visually heavy header that consumes a third of the screen, an input box that refuses pasted text, a viewport that fights the user when they scroll, and no way to copy conversation content. These are not feature gaps — they are friction that erodes trust in the tool on every session.

This PRD covers four targeted improvements that, taken together, lift the Forge TUI to a competitive visual and usability baseline without a full rewrite. Each is independently shippable.

## Key Value Propositions

- **For daily users**: A cleaner, more spacious interface that gets out of the way and lets the conversation take centre stage
- **For new users**: A first impression that signals craft and quality — the header no longer dominates the screen before a single message is sent
- **For all users**: Core interactions that simply work — paste an API key into a settings field, scroll back to re-read output without losing your place, copy a code snippet without leaving the TUI
- **Competitive advantage**: Crush, Codex CLI, and Gemini CLI all ship clean, minimal chrome with open input areas. Forge's current boxed input and ASCII-art header look dated by comparison. These changes close that gap.

## Target Users & Use Cases

### Primary Personas

- **The Daily Driver**: A developer who has Forge open in a terminal pane for hours at a time. They scroll back frequently to re-read agent output, paste API keys when configuring a new project, and want the interface to feel calm and fast — not visually noisy.
- **The New Evaluator**: A developer trying Forge for the first time after seeing it mentioned in a project README. Their first impression is the TUI launch screen. If the header takes up half the terminal and they can't paste their API key into the settings dialog, they leave.
- **The Power User**: A developer who keeps Forge running while the agent executes long multi-step tasks. They need to read earlier context while the agent is still working, and they need to get code snippets out of the conversation buffer without mouse gymnastics.

### Core Use Cases

1. **Long agent runs — read while the agent works**: The agent is writing tests across 12 files. The user wants to scroll up and re-read the plan the agent laid out three minutes ago. Today the viewport snaps back to the bottom on every streamed token. After this feature: scrolling up locks the viewport in place; a subtle indicator shows new content is arriving below.

2. **First-time configuration**: A new user launches Forge, opens Settings, and tries to paste their Anthropic API key. Today the paste is silently discarded — the settings dialog only accepts one character at a time. After this feature: bracketed paste works; the key lands in the field on the first try.

3. **Extracting code from a conversation**: The agent has produced a useful function. The user wants to copy it. Today mouse-capture mode prevents terminal text selection, and there is no keyboard alternative. After this feature: `Ctrl+Y` copies the conversation to the clipboard; `Ctrl+T` suspends the TUI so the user can select text natively.

4. **Starting a session on a small terminal**: A user opens Forge in a 80×24 terminal. Today the ASCII-art header consumes 10 lines, leaving ~10 lines of conversation viewport. After this feature: the compact header uses 3 lines, giving ~17 lines of viewport — 70% more conversation space.

## Product Requirements

### Must Have (P0)

- **Scroll-lock**: When the user scrolls up (PageUp, mouse wheel up), the viewport stops auto-following new agent output. It resumes auto-following when the user presses `G`, scrolls to the bottom, or sends a new message.
- **New-content indicator**: While scroll-locked, a non-intrusive indicator is visible showing that new content is arriving below. It does not scroll or flash.
- **Paste in settings dialogs**: `Ctrl+V` and bracketed paste sequences correctly insert text into all settings input fields. Multi-character paste events must not be silently discarded.
- **Paste in main input**: Bracketed paste must work in the main chat textarea (the bubbles textarea component handles this automatically once bracketed paste is enabled at the program level).

### Should Have (P1)

- **Compact header**: The 6-line ASCII-art header is replaced with a single-line header bar showing the product glyph, current working directory, model name, and version. Total header height drops from 10 lines to 3 lines (1 header + 1 tips + 1 separator).
- **Option B input box**: The enclosed 4-sided rounded input box is replaced with a top-rule-only design — a single hairline rule above the input, with the cursor floating in open space below. No side borders, no bottom border.
- **Copy to clipboard**: `Ctrl+Y` copies the visible conversation buffer to the system clipboard and shows a brief toast confirmation.
- **Bottom bar ANSI fix**: The bottom status bar padding calculation is fixed to use `lipgloss.Width()` instead of `len()`, correcting column misalignment caused by ANSI escape bytes.
- **Contextual tips line**: The dense key-hints text is replaced with a slim line that adapts to current state (idle / agent busy / bash mode).

### Could Have (P2)

- **Suspend for terminal selection**: `Ctrl+T` suspends the TUI (restoring the primary screen buffer and normal mouse mode) so the user can select text natively in the terminal. Any key press resumes the TUI.
- **Export to pager**: `Ctrl+E` pipes the conversation to `$PAGER` (defaulting to `less`) using `tea.ExecProcess()`.
- **Animated new-content indicator**: The "↓ new activity" indicator pulses or has a subtle animation to draw the eye without being distracting.

## User Experience Flow

### Entry Points

- Scroll-lock activates automatically on any upward scroll — no configuration, no discovery needed.
- Paste support is passive — it works when the user does what they already expect to work.
- Copy (`Ctrl+Y`) and suspend (`Ctrl+T`) are discoverable via the key hints line at the bottom of the screen.
- The visual redesign is visible on every launch with no user action required.

### Core User Journey — Scroll While Agent is Running

```
Agent is streaming output → Viewport auto-follows (followScroll = true)
     ↓
User presses PageUp or scrolls mouse wheel up
     ↓
followScroll = false → Viewport freezes at current position
     ↓
Agent continues streaming → Viewport does not move
"↓ New activity below — press G to follow" indicator appears
     ↓
  [User reads history]          [User presses G or scrolls to bottom]
        ↓                                      ↓
  Indicator stays visible           followScroll = true
  Agent output accumulates          Viewport jumps to bottom
  below out of view                 Indicator disappears
```

### Core User Journey — Paste API Key in Settings

```
User opens Settings overlay (Ctrl+S or / → settings)
     ↓
User focuses the API Key field
     ↓
User pastes from clipboard (Cmd+V / middle-click / Shift+Insert)
     ↓
Terminal sends bracketed paste sequence (requires terminal bracketed paste support)
[Note: Ctrl+V delivers a raw key event on terminals without bracketed paste —
 see Known Limitations. tea.PasteMsg is only delivered when the terminal
 supports the bracketed paste protocol.]
     ↓
Bubble Tea delivers tea.PasteMsg to Update()
     ↓
handleDialogInput() handles tea.PasteMsg → handleDialogPaste()
     ↓
API key appears in the field, cursor at end
     ↓
User presses Enter to confirm → setting saved
```

### Core User Journey — Copy Conversation Content

```
Agent has produced useful code in the conversation
     ↓
User presses Ctrl+Y
     ↓
  [Path A — clipboard]                [Path B — suspend, P2]
        ↓                                      ↓
  clipboard.WriteAll(content)         tea.Suspend() called
  Toast: "✓ Copied to clipboard"      TUI suspends, primary buffer restores
  Disappears after 2 seconds          Message printed: "Select text, then press any key"
                                      User selects and copies natively
                                      User presses any key → tea.Resume()
                                      TUI resumes, alt-screen restored
```

### Success States

- User scrolls up mid-agent-run, reads earlier output, never loses their place
- User pastes a 51-character API key into settings on the first try
- User copies a code block with one keypress and pastes it into their editor
- On a 24-line terminal, the conversation viewport shows 17+ lines from launch

### Error / Edge States

- **Clipboard unavailable** (headless server, no display): `clipboard.WriteAll` returns an error; show toast "Clipboard unavailable — use Ctrl+T to select" (or suppress if Ctrl+T is also unavailable)
- **Paste into read-only field**: `handleDialogPaste` checks field editability before inserting; pastes into non-editable fields are silently ignored (same as single-char input today)
- **Paste contains newlines into single-line field**: Strip newlines before inserting, or insert only up to the first newline
- **Terminal does not support bracketed paste**: The `ESC[?2004h` sequence is sent but the terminal ignores it — paste events arrive as rapid key events instead. These still fail (same as today) but the bracketed paste path now works for the majority of modern terminals. No regression.
- **Suspend not supported** (`tea.Suspend()` on Windows or some terminal emulators): Detect and fall back to clipboard-only path; do not expose the `Ctrl+T` hint in those environments

## User Interface & Interaction Design

### Key Interactions

| Keybinding | Action | Context |
|---|---|---|
| `PageUp` / Mouse wheel up | Lock scroll position | Any time |
| `G` / scroll to bottom | Unlock scroll, jump to bottom | While scroll-locked |
| `Enter` (send message) | Unlock scroll, jump to bottom | While scroll-locked |
| Bracketed paste (Cmd+V / Ctrl+V / Shift+Insert) | Paste clipboard text into settings field | Settings dialog active — **requires terminal with bracketed paste support** |
| `Ctrl+Y` | Copy conversation to clipboard | Any time |
| `Ctrl+T` | Suspend TUI for native selection | Any time (P2) |

### Input Box Design (Option B — Top Rule Only)

```
  ──────────────────────────────────────────────────────────────────────────────
  ❯ _

  Enter · send   Alt+Enter · new line   / · commands   Ctrl+Y · copy   Ctrl+C
```

- Rule: `─` repeated to full terminal width, [muted dim] colour
- Prompt glyph `❯`: [salmon] — the only accent element in the input zone
- When agent is busy / input disabled: rule dims further, `❯` is replaced with a spinner or hidden
- Multiline input: text wraps below the `❯`; the rule stays fixed above; the hint line stays fixed below
- The rule adapts width automatically to any terminal resize

### Header Design (Compact Single-Line)

```
⬡ forge  ·  ~/projects/myapp  ·  claude-3-5-sonnet                     v0.4.2
────────────────────────────────────────────────────────────────────────────────
```

- Left: `⬡ forge` in [salmon bold]
- Centre: current working directory, truncated with `…` if terminal is narrow
- Right: model name and version, [muted]
- Separator: full-width `─` in [dim]
- Total height: 2 lines (was 8–10 lines)

### New-Content Indicator

```
  ↓  New activity  ·  G to follow
```

- Rendered as a toast overlay row at the bottom of the viewport — it does not alter the conversation buffer
- Colour: [muted] — visible but not alarming
- Disappears immediately when `followScroll` becomes true

### Information Architecture

- **Header zone** (2 lines): identity, context, version — static
- **Conversation viewport** (dynamic): all messages, tool calls, agent output — scrollable
- **Input zone** (3 lines: rule + input + blank): user composition — always present
- **Hint line** (1 line): contextual key hints — adapts to state

### Progressive Disclosure

- Default state: hint line shows the four most common keys (send, new line, commands, exit)
- Agent busy state: hint line changes to interrupt + scroll hint
- Scroll-locked state: new-content indicator appears in viewport
- Advanced keys (`Ctrl+Y`, `Ctrl+T`): shown in hint line when relevant, not always visible

## Feature Metrics & Success Criteria

### Key Performance Indicators

- **Paste success rate**: % of settings dialog sessions where a value is entered without the user having to type character-by-character (proxy: session recordings / user reports)
- **Scroll-lock usage**: % of sessions where the user scrolls up while the agent is active (indicates the feature is exercised)
- **Viewport efficiency**: Average conversation lines visible on launch across terminal sizes (target: 15+ on 24-line terminal)
- **User satisfaction**: Qualitative feedback — does the TUI feel "clean" and "polished"?

### Success Thresholds

- Zero reports of "I can't paste my API key" after the paste fix ships
- Zero reports of "the viewport keeps jumping" after the scroll-lock ships
- No regression in existing TUI functionality (verify via existing test suite + manual smoke test)

## User Enablement

### Discoverability

- Scroll-lock is automatic — no discovery needed; the indicator is the discoverability mechanism
- Paste works with standard OS shortcuts — no discovery needed
- `Ctrl+Y` appears in the hints line when the conversation has content
- `Ctrl+T` (P2) appears in the hints line as "Ctrl+T · select"

### Onboarding

No onboarding flow needed. All four improvements work transparently on first use. The visual redesign requires no learning.

### Mastery Path

- Basic: paste in settings, read history while agent works
- Intermediate: use `Ctrl+Y` to extract code from conversations
- Advanced: use `Ctrl+T` (P2) to select arbitrary spans of text from the conversation

## Risk & Mitigation

### User Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Scroll-lock confuses users who don't realise they're locked | Low | New-content indicator clearly signals the locked state and shows how to unlock |
| Bracketed paste breaks on unusual terminals | Low | Bracketed paste is widely supported; terminals that ignore it fall back to current behaviour (no regression) |
| Compact header removes information users relied on | Low | All information is preserved — just reorganised onto one line |
| `Ctrl+Y` conflicts with user's terminal keybinding | Low | Document the binding; make it rebindable in a future settings iteration |

### Adoption Risks

- The visual changes are visible on every launch — if the new header feels wrong, users will notice immediately. Mitigation: the mockups (`tui-mockups.md`) have been reviewed before implementation; the design is conservative and information-complete.
- Scroll-lock changes behaviour users may be accidentally relying on. Mitigation: the default (`followScroll = true`) preserves existing behaviour exactly; the change only activates when the user explicitly scrolls up.

## Dependencies & Integration Points

### Feature Dependencies

- Bubble Tea v1.3.10 (already in use): provides `tea.WithBracketedPaste()`, `tea.PasteMsg`, `tea.Suspend()`, `tea.Resume()`
- `atotto/clipboard` (already in use): provides `clipboard.WriteAll()`
- `charmbracelet/lipgloss` v1.1.0 (already in use): provides `lipgloss.Width()` for ANSI-safe string measurement

### System Integration

- All changes are confined to `pkg/executor/tui/` and `pkg/executor/tui/overlay/`
- No changes to the agent loop, tool system, or LLM providers
- No changes to the CLI executor or headless mode
- The `pkg/ui/ascii.go` ASCII art generator is retained but no longer called from the main header path

### External Dependencies

None. All dependencies are already vendored.

## Constraints & Trade-offs

### Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Input box style | Top rule only (Option B) | 11 alternatives evaluated; enclosing box adds visual weight without functional benefit; open rule keeps focus on conversation content |
| Header height | Single line (compact) | 10-line header is the largest single cause of viewport cramping; single line recovers 7+ lines of conversation space |
| Scroll-lock default | Auto-follow (same as today) | Zero behaviour change for users who never scroll up; opt-in to lock via natural scroll gesture |
| Copy mechanism | Clipboard first, suspend second (P2) | Clipboard works without any terminal changes; suspend is more powerful but adds complexity |
| Bracketed paste | Enable at program level | Fixes both settings dialogs and main textarea in one change; no per-field handling needed |

### Known Limitations

- **Ctrl+V on non-bracketed-paste terminals**: Bracketed paste support is the mechanism by which paste events reach the TUI as `tea.PasteMsg`. On terminals that do not implement the bracketed paste protocol (rare, but includes some minimal/embedded terminals), pressing Ctrl+V delivers a raw key event rather than a `tea.PasteMsg` — paste will still fail on those terminals, same as today. This is a terminal capability limitation, not a Forge bug. Modern terminals (iTerm2, Alacritty, kitty, Windows Terminal, GNOME Terminal, tmux ≥ 2.6) all support bracketed paste. The fix benefits the vast majority of users.
- **Alt-screen drag selection**: `tea.WithAltScreen()` and `tea.WithMouseCellMotion()` remain active. Drag-to-select with the mouse in the conversation viewport is not fixed by this PRD — it requires removing or toggling mouse capture, which is a larger change. `Ctrl+Y` and `Ctrl+T` (P2) are the provided alternatives.
- **Windows clipboard**: `atotto/clipboard` on Windows requires `clip.exe` which may not be available in all environments. Error handling covers this case.
- **Settings dialog field validation**: `handleDialogPaste` applies the same per-field validation as single-char input. Pasting invalid characters (e.g. spaces into an API key field) will silently truncate or reject, consistent with current single-char behaviour.

### Future Considerations

- Full TUI package decomposition (tracked in `docs/product/scratch/tui-v2.md`)
- Theme system / multiple colour schemes
- Rebindable keybindings
- Mouse-based text selection inside alt-screen (terminal-dependent, requires vendor cooperation)

## Competitive Analysis

| Product | Input style | Header | Scroll behaviour | Copy support |
|---|---|---|---|---|
| **Crush** (charmbracelet) | Open, rule-based | Compact 1-line | Smart follow | Clipboard shortcut |
| **Codex CLI** (OpenAI) | Open prompt | Minimal 1-line | Smart follow | Native terminal (no mouse capture in viewport) |
| **Gemini CLI** (Google) | Open prompt | No header | Smart follow | Native terminal |
| **Forge (current)** | Enclosed box | 6-line ASCII art | Always jumps to bottom | Not possible |
| **Forge (this PRD)** | Top rule only | 1-line compact | Smart scroll-lock | Ctrl+Y + Ctrl+T |

All three leading competitors use open, unboxed input areas and compact or absent headers. Forge's current aesthetic is a significant outlier. This PRD closes the gap on the highest-impact visual and usability dimensions.

## Go-to-Market Considerations

### Positioning

These are bug fixes and polish improvements, not a new feature. Position them as quality-of-life improvements in release notes: "The TUI is now cleaner, faster to read, and behaves the way you expect."

### Documentation Needs

- Update the TUI section of the README with new screenshots reflecting the compact header and open input
- Update keybindings reference to include `Ctrl+Y` and `Ctrl+T`
- No new docs required — the changes are self-evident

### Support Requirements

- Support team should know: "paste not working in settings" is fixed in this release — close any open issues on this topic
- Known remaining limitation: drag-to-select in the TUI viewport is still not possible (alt-screen + mouse capture); redirect to `Ctrl+Y`

## Evolution & Roadmap

### Version History

- **v1.0 (this PRD)**: Scroll-lock, paste in settings, copy to clipboard, compact header, Option B input box

### Future Vision

- The full TUI v2 redesign (`docs/product/scratch/tui-v2.md`) builds on this foundation: package decomposition, theme system, collapsible message blocks, and a unified overlay interface. The visual language established here (open input, compact header) becomes the baseline for the v2 design.

### Deprecation Strategy

- The 6-line ASCII art header in `pkg/ui/ascii.go` is retained in code but no longer called from the TUI main path. It can be removed in a follow-up cleanup PR once the compact header is confirmed stable.

## Technical References

- **Architecture**: See ADR (to be written) for scroll-lock implementation pattern and bracketed paste integration
- **Design exploration**: `docs/product/scratch/tui-ux-enhancements.md` — detailed per-feature implementation notes with exact file/line references
- **Visual mockups**: `docs/product/scratch/tui-mockups.md` — 10 full-terminal layout mockups (all using Option B input)
- **Input alternatives**: `docs/product/scratch/tui-input-alternatives.md` — 11 input box styles evaluated before selecting Option B
- **Full redesign vision**: `docs/product/scratch/tui-v2.md` — longer-term architectural redesign, independent of this PRD

## Appendix

### Affected Files Summary

| File | Changes |
|---|---|
| `pkg/executor/tui/executor.go` | Add `tea.WithBracketedPaste()` to `tea.NewProgram` options |
| `pkg/executor/tui/model.go` | Add `followScroll bool`, `hasNewContent bool` fields |
| `pkg/executor/tui/init.go` | Initialise `followScroll: true` |
| `pkg/executor/tui/update.go` | Scroll-up detection; `G` handler; guard all `GotoBottom()` calls; `Ctrl+Y` handler; `tea.ResumeMsg` handler; fix `headerHeight` constant |
| `pkg/executor/tui/events.go` | Guard all 8 `GotoBottom()` calls with `followScroll` check |
| `pkg/executor/tui/view.go` | Replace `buildHeader()` with compact bar; replace `buildInputBox()` with top-rule design; fix `buildBottomBar()` padding; render new-content indicator; add `Ctrl+Y` hint |
| `pkg/executor/tui/styles.go` | Add `headerBarStyle`, `inputRuleStyle`; remove 4-sided `inputBoxStyle` |
| `pkg/executor/tui/overlay/settings.go` | Add `tea.PasteMsg` case to `handleDialogInput()`; add `handleDialogPaste()` method |

### GotoBottom Call Sites to Guard

| File | Lines | Context |
|---|---|---|
| `events.go` | 106, 125, 126, 150, 224, 275, 289, 355 | Streaming content handlers |
| `update.go` | 390, 407, 418, 432, 599, 639, 656, 675 | Key handlers + `recalculateLayout()` |

### Research & Validation

- Competitive TUI research conducted on Crush (charmbracelet/crush), Codex CLI (openai/codex, Rust/ratatui), and Gemini CLI (google-gemini/gemini-cli)
- 11 input box styles mocked up and evaluated; Option B selected for minimum chrome and maximum focus on conversation content
- 10 full-layout mockups produced covering the spectrum from zero-chrome to dense dashboard; all use Option B input
- Exact bug locations verified against current source: `update.go:319` (`headerHeight := 10`), `settings.go:1273` (`len(keyMsg.String()) == 1`), `events.go:106` (first unconditional `GotoBottom`)
