# How to Use the TUI Interface

**Last Updated:** January 2025
**Difficulty:** Beginner
**Estimated Time:** 10 minutes

---

## Overview

The Forge Terminal User Interface (TUI) provides an interactive chat-based interface for working with the coding agent. This guide covers all aspects of using the TUI, from basic chat interactions to advanced features like slash commands, overlays, and settings configuration.

---

## Table of Contents

1. [Starting the TUI](#starting-the-tui)
2. [Interface Layout](#interface-layout)
3. [Basic Chat Interface](#basic-chat-interface)
4. [Keyboard Shortcuts](#keyboard-shortcuts)
5. [Smart Scroll-Lock](#smart-scroll-lock)
6. [Clipboard Copy](#clipboard-copy)
7. [Slash Commands](#slash-commands)
8. [Overlays](#overlays)
9. [Agent Thinking Blocks](#agent-thinking-blocks)
10. [Tool Approval Workflow](#tool-approval-workflow)
11. [Settings Configuration](#settings-configuration)
12. [Tips & Best Practices](#tips--best-practices)

---

## Starting the TUI

### Launch Command

```bash
forge tui
```

Or if running from source:

```bash
go run cmd/forge/main.go tui
```

The TUI opens directly into the main interface — there is no splash screen.

---

## Interface Layout

The TUI is composed of five visual zones stacked vertically:

```
⬡ forge          /path/to/workspace          gpt-4o
────────────────────────────────────────────────────
  Enter · send   Alt+Enter · new line   / · commands   Ctrl+Y · copy   Ctrl+C · exit

 ┌ conversation viewport ─────────────────────────┐
 │                                                 │
 │  [agent messages, tool calls, results appear    │
 │   here and scroll as the conversation grows]    │
 │                                                 │
 └─────────────────────────────────────────────────┘

↓  New content below  — press G or PgDn to follow    ← only when scroll-locked

  [loading spinner + message]                         ← only when agent is busy
────────────────────────────────────────────────────
❯ [your input here]
                                    ⸫ Thinking On   ctx ████░░░░ 12k / 128k
```

### Header Bar (2 lines)

- **Left**: `⬡ forge` — brand identifier (salmonPink)
- **Center**: Current workspace directory (truncated if too wide)
- **Right**: Active LLM model name (e.g. `gpt-4o`)
- **Separator**: Full-width `─` rule beneath the bar

### Hints Bar

A single line of contextual keyboard hints that changes based on current state:

| State | Hints shown |
|-------|-------------|
| Idle | `Enter · send   Alt+Enter · new line   / · commands   Ctrl+Y · copy   Ctrl+C · exit` |
| Agent busy | `Ctrl+C · interrupt` (+ `G · follow output` when scroll-locked) |
| Overlay open | `Esc · close   Tab · next field   Enter · confirm` |
| Bash mode | `Enter · run   exit · return to normal   Ctrl+C · cancel` |

### Conversation Viewport

Scrollable area displaying the full conversation history. New content streams in at the bottom. When the viewport is auto-following output, the view scrolls down automatically as the agent responds.

### Scroll-Lock Indicator

When you scroll up while the agent is generating output, a banner appears:

```
↓  New content below  — press G or PgDn to follow
```

Press **G** or **PgDn** to jump back to the bottom and resume auto-following.

### Input Zone

A borderless input area with a `❯` prompt glyph:

```
─────────────────────────────────────────────────
❯ type your message here
```

The textarea grows automatically as you type multi-line content (up to one-third of the screen height).

### Status Bar

The bottom status bar shows:

- **Left**: `bash mode` label (only visible in bash mode, in mintGreen)
- **Right**: Thinking state indicator (`⸫ Thinking On` / `⸫ Thinking Hidden`) and context usage bar

The context bar format: `ctx ████░░░░ 12k / 128k`
- Bar fills proportionally to current context usage
- Color changes from green → orange → red as context fills up

---

## Basic Chat Interface

### Sending Messages

1. **Type your message** in the input box at the bottom
2. **Press Enter** to send the message to the agent
3. The agent will process your request and respond in real time

**Example conversation:**
```
You: Create a new file called hello.go with a simple main function

Agent: I'll create that file for you.

[write_file] hello.go
  + package main...

[✓] File created successfully
```

### Message Types

The chat interface displays different message types with visual indicators:

- **Your messages**: Your input, labeled `You:`
- **Agent messages**: Agent prose responses
- **Thinking blocks**: Extended reasoning (shown/hidden based on the thinking toggle — see [Agent Thinking Blocks](#agent-thinking-blocks))
- **Tool calls**: Actions the agent is taking, shown as the tool name and parameters
- **Tool results**: Outcome of tool executions, summarized with status icons
- **System messages**: Status updates and toast notifications

### Multi-line Input

To add line breaks in your message:
- **Press Alt+Enter** to insert a new line
- Continue typing on the next line
- **Press Enter** (without Alt) to send the complete message

The input area grows automatically to accommodate multiple lines.

### Pasting Text

Standard paste shortcuts work in the input area:
- **Cmd+V** (macOS) or **Ctrl+Shift+V** / **Shift+Insert** (Linux)

In the settings overlay, bracketed paste (ADR-0049) is supported — pasted text is treated as literal input rather than individual keystrokes.

---

## Keyboard Shortcuts

### Global Shortcuts

| Shortcut | Action |
|----------|--------|
| **Enter** | Send message |
| **Alt+Enter** | Insert new line |
| **Ctrl+C** | Exit TUI (or interrupt agent if busy; or exit bash mode) |
| **Esc** | Close active overlay / exit bash mode |
| **Ctrl+Y** | Copy full conversation to clipboard (plain text, ANSI stripped) |

### Viewport Navigation (Scroll-Lock)

| Shortcut | Action |
|----------|--------|
| **PgUp** / **Ctrl+B** | Scroll up (disables auto-follow) |
| **PgDn** | Scroll down (re-enables auto-follow when at bottom) |
| **G** | Jump to bottom and resume auto-follow (when scroll-locked) |

### Tool Result Access

| Shortcut | Action |
|----------|--------|
| **Ctrl+V** | View the last tool result in a full overlay |
| **Ctrl+L** | Open result history — browse all tool results from the session |

### Command Palette

| Shortcut | Action |
|----------|--------|
| **Ctrl+K** / **Ctrl+P** | Toggle command palette |
| **/` (slash)** | Open command palette via input (type `/` as first character) |
| **↑ / ↓** | Navigate commands in palette |
| **Tab** | Autocomplete selected command and close palette |
| **Enter** | Execute selected command immediately |
| **Esc** | Close palette without executing |

---

## Smart Scroll-Lock

The TUI implements smart scroll-lock (ADR-0048) to let you review previous output while the agent is still generating new content.

### How It Works

1. **Auto-follow mode** (default): The viewport scrolls down automatically as new agent output arrives.
2. **Scroll-lock mode**: Activated automatically when you scroll up (PgUp or mouse wheel up). Auto-follow is paused.
3. **New content indicator**: While scroll-locked, if the agent produces new output, a banner appears at the bottom of the viewport:
   ```
   ↓  New content below  — press G or PgDn to follow
   ```
4. **Resume auto-follow**: Press **G** (anywhere in the input) or **PgDn** (when at the bottom) to jump back to the latest content and re-enable auto-follow.

This allows you to review earlier parts of a long response without missing what the agent is currently writing.

---

## Clipboard Copy

Press **Ctrl+Y** at any time to copy the full conversation history to your system clipboard.

- The entire conversation buffer is copied (not just the visible viewport).
- ANSI color codes are automatically stripped — the clipboard receives clean plain text.
- A toast notification confirms success or reports an error (e.g., if no clipboard manager is available).

**Tip:** This is especially useful for sharing agent output in tickets, pull requests, or code reviews.

---

## Slash Commands

Slash commands provide quick access to TUI features and agent actions. Type `/` to open the command palette, or type `/command` directly.

### Available Commands

#### `/help` — Show Help Information

```
/help
```

Displays the help overlay with all keyboard shortcuts and commands.

#### `/stop` — Stop Agent Operation

```
/stop
```

Immediately interrupts the current agent operation. Use this if the agent is stuck or you want to cancel an action.

#### `/commit` — Create Git Commit

```
/commit [message]
```

Creates a git commit with all changes from the current session. If no message is provided, the agent generates one based on the changes.

**Examples:**
```
/commit Add user authentication feature
/commit Fix bug in payment processing
/commit
```

#### `/pr` — Create Pull Request

```
/pr [title]
```

Creates a pull request from the current branch. If no title is provided, the agent generates one.

**Note:** Requires a configured git remote.

#### `/settings` — Open Settings

```
/settings
```

Opens the interactive settings overlay for configuring LLM parameters, auto-approval rules, UI preferences, and more.

#### `/context` — Show Context Information

```
/context
```

Displays detailed information about the current workspace, conversation history, token usage, and memory state.

#### `/bash` — Enter Bash Mode

```
/bash
```

Switches to bash command mode for direct shell command execution. The `❯` prompt turns green. Type `exit` to return to normal mode, or press **Ctrl+C** / **Esc**.

#### `/notes` — Browse Agent Notes

```
/notes
```

Opens the notes viewer overlay to browse scratchpad notes created by the agent during the session.

#### `/snapshot` — Export Context Snapshot

```
/snapshot
```

Dumps the full live conversation payload (as seen by the LLM) to a timestamped JSON file.

- **Output path**: `<workspace>/.forge/context/context-<timestamp>.json`
- **Use case**: Inspecting the exact data sent to the LLM for debugging context management and summarization.

---

## Overlays

Overlays are modal panels that appear on top of the conversation for specific interactions. Press **Esc** to close most overlays.

### Help Overlay (`/help`)

Shows all keyboard shortcuts and available slash commands.

**Controls:**
- **↑ / ↓**: Scroll content
- **Esc**: Close

### Settings Overlay (`/settings`)

Interactive configuration interface organized into collapsible sections.

**Controls:**
- **↑ / ↓**: Navigate sections and items
- **Enter**: Edit the selected item / confirm
- **Space**: Toggle boolean settings
- **Esc**: Close without saving
- **Ctrl+S**: Save and apply changes

### Context Overlay (`/context`)

Displays detailed context information including workspace path, token usage, conversation history length, and active context management strategy.

**Controls:**
- **↑ / ↓**: Scroll content
- **Esc**: Close

### Tool Approval Overlay

Appears when the agent requests to execute an operation that requires explicit approval.

Shows:
- Tool name and description
- All parameters being passed
- Approve / Deny buttons

**Controls:**
- **Tab**: Move between Approve / Deny buttons
- **Enter**: Confirm the selected action
- **a**: Quick-approve
- **d**: Quick-deny
- **Esc**: Deny and close

### Tool Result Overlay (`Ctrl+V`)

Opens the full output of the most recent tool call in a scrollable panel. Useful for reading large file contents or long command output.

**Controls:**
- **↑ / ↓** / **PgUp / PgDn**: Scroll content
- **Esc**: Close

### Result History Overlay (`Ctrl+L`)

Shows a scrollable list of all tool results from the current session. Select any entry to view it in full.

**Controls:**
- **↑ / ↓**: Navigate results
- **Enter**: Open selected result in full overlay
- **Esc**: Close

### Command Palette (`Ctrl+K`, `Ctrl+P`, or `/`)

Quick-access launcher for slash commands.

**Controls:**
- **↑ / ↓**: Navigate commands
- **Tab**: Autocomplete the selected command into the input box
- **Enter**: Execute the selected command immediately
- **Esc**: Close without executing

### Diff Viewer Overlay

Displayed when the agent uses `apply_diff` to modify a file. Shows the unified diff with syntax highlighting.

**Controls:**
- **↑ / ↓**: Scroll through diff
- **PgUp / PgDn**: Page navigation
- **Esc**: Close

### Notes Viewer Overlay (`/notes`)

Browsable list of scratchpad notes created during the session.

**Controls:**
- **↑ / ↓**: Navigate notes
- **Enter**: View full note content
- **Esc**: Close / go back

---

## Agent Thinking Blocks

When using models with extended thinking (e.g., Claude with thinking enabled), the agent's internal reasoning process is shown as "thinking blocks" before its response.

### Show/Hide Thinking

The thinking state is always visible in the status bar:

- `⸫ Thinking On` — thinking blocks are displayed in the conversation
- `⸫ Thinking Hidden` — thinking is happening but not shown

Toggle this in `/settings` under the UI section, or check the current state at a glance in the bottom-right status bar.

### Thinking Block Display

When enabled, thinking blocks appear in the conversation with:
- Italic muted-gray styling to distinguish them from main responses
- Elapsed time indicator showing how long the agent spent reasoning

---

## Tool Approval Workflow

The TUI implements a security-first approval system for potentially impactful operations.

### When Approval is Required

The agent requests approval for:
- **File writes**: Creating or modifying files
- **File deletions**: Removing files
- **Command execution**: Running shell commands
- **Git operations**: Commits, pushes, PR creation
- **Any tool** configured to require approval

### Making a Decision

**To Approve:**
- Press **a** for quick approval
- Or use **Tab** to focus the Approve button, then **Enter**

**To Deny:**
- Press **d** for quick denial
- Or use **Tab** to focus the Deny button, then **Enter**
- Or press **Esc** to deny and close

### Auto-Approval Rules

Configure auto-approval for trusted operations in `/settings` under the Auto-Approval section:

- Auto-approve read-only operations
- Auto-approve writes to specific path patterns
- Auto-approve specific shell commands
- Configure per-tool rules

---

## Settings Configuration

Access settings with `/settings`.

### LLM Section

- **Model**: The model name used for all agent calls
- **Summarization Model**: Optional separate model for context summarization
- **Base URL**: API endpoint (for OpenAI-compatible providers)
- **API Key**: Authentication key

### Auto-Approval Section

Configure which tool calls are automatically approved without prompting.

### UI Section

- **Show Thinking**: Toggle display of extended thinking blocks in the conversation

### Saving Settings

Press **Ctrl+S** inside the settings overlay to save and apply all changes immediately. LLM settings take effect on the next agent call.

---

## Tips & Best Practices

### Effective Communication

1. **Be Specific**: Provide clear, detailed requests
   - ❌ "Make the code better"
   - ✓ "Refactor the authentication function to extract validation logic into a helper"

2. **Provide Context**: Reference existing code or files
   - ✓ "In `pkg/auth/login.go`, add error handling to the `Login` function"

3. **Break Down Complex Tasks**: Split large requests into steps
   - ✓ "First create the database schema. Then implement the repository layer."

### Working with Files

1. **Review Changes**: Always read the diff before approving writes
2. **Use Version Control**: Commit frequently to checkpoint progress
3. **Backup Important Files**: Before major refactoring sessions

### Managing Long Conversations

1. **Check Token Usage**: Watch the context bar in the bottom-right — orange/red means you're near the limit
2. **Context Summarization**: Forge automatically summarizes older messages to free up context when needed — you'll see a "Optimizing context..." toast
3. **Export Context**: Use `/snapshot` to snapshot the full conversation payload for debugging
4. **Start Fresh**: For a completely new topic, restart the TUI

### Scroll & Navigation

1. **Stay in auto-follow**: Let the viewport follow output automatically while the agent works
2. **Scroll back freely**: PgUp to review earlier output without losing auto-follow permanently — just press **G** to jump back
3. **Result history**: Use **Ctrl+L** to browse all tool results without scrolling through the conversation

### Keyboard Efficiency

1. **Use the command palette**: **Ctrl+K** or **Ctrl+P** for instant slash command access
2. **Ctrl+V** for quick result inspection: No need to scroll up to find the last tool output
3. **Ctrl+Y** to share output: Copy the full session to clipboard in one keystroke

### Troubleshooting

**Agent Not Responding:**
- Check network connection (for cloud LLMs)
- Verify API key in `/settings`
- Use `/stop` and try again

**Context Limit Errors:**
- Watch the context bar — switch models or start a new session if at capacity
- Use `/snapshot` to inspect what's in context

**Overlays Not Closing:**
- Press **Esc** (may need to press multiple times if overlays are nested)

**Clipboard Copy Not Working:**
- Linux requires a clipboard manager (`xclip`, `xsel`, or `wl-clipboard`)
- The TUI will show a toast error if no clipboard is available

**Settings Not Saving:**
- Press **Ctrl+S** explicitly inside the settings overlay (changes are not auto-saved on Esc)

---

## Related Documentation

- [Getting Started Guide](../getting-started/quick-start.md)
- [Understanding the Agent Loop](../getting-started/understanding-agent-loop.md)
- [Configure LLM Providers](configure-provider.md)
- [ADR-0048: Smart Scroll-Lock](../adr/0048-tui-smart-scroll-lock.md)
- [ADR-0049: Bracketed Paste Support](../adr/0049-tui-bracketed-paste-support.md)
- [ADR-0050: Clipboard Copy](../adr/0050-tui-clipboard-copy.md)
- [ADR-0051: Visual Redesign](../adr/0051-tui-visual-redesign.md)

---

## Next Steps

Now that you know how to use the TUI, explore:

1. **[Create Custom Tools](create-custom-tool.md)** — Extend agent capabilities
2. **[Configure LLM Providers](configure-provider.md)** — Set up different AI models
3. **[Manage Memory](manage-memory.md)** — Control conversation context
4. **[Handle Errors](handle-errors.md)** — Debug and recover from issues
