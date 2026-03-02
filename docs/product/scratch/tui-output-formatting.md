# TUI Output Formatting & Polish

## Problem Statement

The Forge TUI currently struggles with rendering command outputs, tool results, and errors cleanly in the main conversation viewport. Specifically, there are three primary issues:

1. **Terminal Rendering Breakage (ANSI & Control Chars)**: Raw output from commands often contains ANSI escape codes, carriage returns (`\r`), backspaces (`\b`), and other non-printable control characters. When written to the main chat viewport (`m.content`), these characters break `lipgloss` text measuring and wrapping logic, causing the viewport to miscalculate heights or corrupt previous lines.
2. **Command Output Spew**: Unlike other tools which show a concise 1-line summary and a 3-line preview in the chat log, `execute_command` streams its *entire* unedited output directly into the main conversation log. This completely overwhelms the chat history when running commands like `npm install` or `go test`.
3. **Verbose, Alarming Errors**: When a tool fails or an agent loop encounters an error (via `pkgtypes.EventTypeError`), the TUI currently dumps the entire raw error payload into the chat log using a loud, bright `errorStyle`. Since Forge is an autonomous agent, these are often just intermediate bumps that the agent can recover from (like a mismatched filename), not catastrophic application failures. Showing massive red error logs overwhelms the user.

## Proposed Solution & Implementation Plan

### 1. Robust Output Sanitization
We need a thorough sanitization function to guarantee terminal safety.
- **Action**: Enhance the existing `helpers.go:stripANSI` function. Create a new `sanitizeOutput(s string) string` function that applies the ANSI regex AND strips/replaces problematic control characters (e.g., stripping characters `< 0x20` while preserving `\n` and `\t`).
- **Application**: Ensure `sanitizeOutput` is universally applied to any arbitrary text right before it is appended to `m.content.WriteString()` in `events.go`.

### 2. Standardize Command Tool Results
We should bring `execute_command` into the same fold as the rest of the tools, showing a neat summary in the chat log while keeping the live-action streaming isolated to the center overlay.
- **Action 1 (Stop Spewing)**: In `pkg/executor/tui/events.go` inside `handleCommandExecutionOutput`, remove the line where it appends directly to `m.content`. (The overlay handles streaming on its own just fine via `overlay/command.go`).
- **Action 2 (Re-classify Execute Command)**: In `pkg/executor/tui/result_display.go`, remove the special-case `if toolName == "execute_command" { return TierOverlayOnly }`. This will allow command completions to fall through to standard line-count classification (`TierSummaryWithPreview`).
- **Result**: `execute_command` will correctly print a neat `✓ Executed command (X lines)` item with a 3-line truncated preview in the chat log, hiding the rest in the viewable overlay.

### 3. Error Softening & Truncation
Errors should look like "agent stumbling blocks" rather than fatal app crashes.
- **Action 1 (Truncation)**: Create a `truncateLines(s string, maxLines int)` helper.
- **Action 2 (Softer Styles)**: In `pkg/executor/tui/styles.go`, implement a `warningStyle` (using `mutedGray` or another subtle, non-red tone) to replace `errorStyle` for standard agent errors.
- **Action 3 (Update Handlers)**: In `events.go:handleError`, truncate the error text to max 2 lines, change the icon to something like `⚠`, and style it softly: `  ⚠ Agent encountered an issue: [truncated error]`.
