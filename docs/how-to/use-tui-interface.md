# How to Use the TUI Interface

**Last Updated:** December 2024  
**Difficulty:** Beginner  
**Estimated Time:** 10 minutes

---

## Overview

The Forge Terminal User Interface (TUI) provides an interactive chat-based interface for working with the coding agent. This guide covers all aspects of using the TUI, from basic chat interactions to advanced features like slash commands, overlays, and settings configuration.

---

## Table of Contents

1. [Starting the TUI](#starting-the-tui)
2. [Basic Chat Interface](#basic-chat-interface)
3. [Keyboard Shortcuts](#keyboard-shortcuts)
4. [Slash Commands](#slash-commands)
5. [Overlays](#overlays)
6. [Tool Approval Workflow](#tool-approval-workflow)
7. [Settings Configuration](#settings-configuration)
8. [Tips & Best Practices](#tips--best-practices)

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

### Welcome Screen

Upon first launch, a welcome screen is displayed with ASCII art branding. This screen confirms that the application has started successfully. Press any key to proceed to the main interface.

### Main Interface

After the welcome screen, the main interface appears. It is composed of several key components:
- **Header**: Shows workspace path and context information.
- **Chat Area**: Displays the conversation history with the agent.
- **Input Box**: Where you type messages (at the bottom of the screen).
- **Status Bar**: Shows the current agent state, token usage, and contextual hints.

---

## Basic Chat Interface

### Sending Messages

1. **Type your message** in the input box at the bottom
2. **Press Enter** to send the message to the agent
3. The agent will process your request and respond

**Example conversation:**
```
You: Create a new file called hello.go with a simple main function

Agent: I'll create that file for you.

[Tool Call: write_file]
- path: hello.go
- content: package main...

✓ File created successfully
```

### Message Types

The chat interface displays different message types with visual indicators:

- **User Messages**: Your input (left-aligned)
- **Agent Messages**: Agent responses (left-aligned, different color)
- **Tool Calls**: Actions the agent wants to take (highlighted)
- **Tool Results**: Outcome of tool executions (with status icons)
- **System Messages**: Status updates and notifications

### Multi-line Input

To add line breaks in your message:
- **Press Alt+Enter** to insert a new line
- Continue typing on the next line
- **Press Enter** (without Alt) to send the complete message

---

## Keyboard Shortcuts

### Essential Shortcuts

| Shortcut | Action |
|----------|--------|
| **Enter** | Send message / Execute command |
| **Alt+Enter** | Insert new line in message |
| **Ctrl+C** | Exit TUI / Cancel operation |
| **Ctrl+D** | Show help overlay |
| **Esc** | Close current overlay / Cancel |

### Navigation Shortcuts

| Shortcut | Action |
|----------|--------|
| **↑ / ↓** | Scroll chat history |
| **PgUp / PgDn** | Page up/down in overlays |
| **Tab** | Navigate between buttons in overlays |
| **Space** | Toggle selection in approval dialogs |

### Command Palette

| Shortcut | Action |
|----------|--------|
| **/** | Open command palette |
| **↑ / ↓** | Navigate commands |
| **Enter** | Execute selected command |
| **Esc** | Close palette |

---

## Slash Commands

Slash commands provide quick access to TUI features and agent actions. Type `/` to open the command palette.

### Available Commands

#### `/help` - Show Help Information
```
/help
```
Displays a help overlay with:
- Available commands
- Keyboard shortcuts
- Usage tips

#### `/stop` - Stop Agent Operation
```
/stop
```
Immediately stops the current agent operation. Use this if the agent is stuck or you want to cancel an action.

#### `/commit` - Create Git Commit
```
/commit [message]
```
Creates a git commit with all changes from the current session.

**Examples:**
```
/commit Add user authentication feature
/commit Fix bug in payment processing
/commit
```

If you don't provide a message, the agent will generate one based on the changes.

**Note:** This command requires approval before execution.

#### `/pr` - Create Pull Request
```
/pr [title]
```
Creates a pull request from the current branch.

**Examples:**
```
/pr Add dark mode support
/pr
```

If you don't provide a title, the agent will generate one.

**Note:** This command requires approval and git remote must be configured.

#### `/settings` - Open Settings
```
/settings
```
Opens the interactive settings overlay where you can configure:
- Auto-approval rules
- LLM provider settings
- Display preferences

#### `/context` - Show Context Information
```
/context
```
Displays detailed information about:
- Current workspace
- Conversation history
- Token usage
- Memory state

#### `/bash` - Enter Bash Mode
```
/bash
```
Switches to bash command mode for direct shell command execution. Type `exit` to return to normal mode.

---

## Overlays

Overlays are modal dialogs that appear on top of the chat interface for specific interactions.

### Types of Overlays

#### 1. Help Overlay (`/help` or Ctrl+D)

Shows comprehensive help information including:
- Command reference
- Keyboard shortcuts
- Usage tips

**Controls:**
- **↑ / ↓**: Scroll content
- **Esc**: Close overlay

#### 2. Settings Overlay (`/settings`)

Interactive configuration interface with tabs:
- **General**: Basic settings
- **LLM**: Provider and model configuration
- **Auto-Approval**: Configure trusted operations
- **Display**: UI preferences

**Controls:**
- **Tab**: Switch between tabs
- **↑ / ↓**: Navigate options
- **Space**: Toggle checkboxes
- **Enter**: Edit text fields
- **Esc**: Close without saving
- **Ctrl+S**: Save changes

#### 3. Context Overlay (`/context`)

Displays detailed context information:
- Workspace path and statistics
- Conversation history summary
- Token usage metrics
- Memory state

**Controls:**
- **↑ / ↓**: Scroll content
- **Esc**: Close overlay

#### 4. Tool Approval Overlay

Appears when the agent requests to execute a tool that requires approval.

Shows:
- Tool name and description
- Parameters being passed
- Approval buttons

**Controls:**
- **Tab**: Navigate between Approve/Deny buttons
- **Enter**: Confirm selection
- **Esc**: Deny and close
- **a**: Quick approve
- **d**: Quick deny

#### 5. Diff Viewer Overlay

Displays code changes with syntax highlighting when the agent modifies files.

Shows:
- File path being modified
- Side-by-side or unified diff view
- Syntax-highlighted code

**Controls:**
- **↑ / ↓**: Scroll through diff
- **PgUp / PgDn**: Page navigation
- **Esc**: Close viewer

#### 6. Command Execution Overlay

Shows real-time output when executing shell commands.

Displays:
- Command being executed
- Stdout output (live)
- Stderr output (live)
- Exit code

**Controls:**
- **↑ / ↓**: Scroll output
- **Ctrl+C**: Terminate command
- **Esc**: Close (after completion)

#### 7. Result List Overlay

Displays multiple tool results in a scrollable list.

Shows:
- Tool name
- Execution status
- Truncated results
- Expandable details

**Controls:**
- **↑ / ↓**: Navigate results
- **Enter**: Expand/collapse result
- **Esc**: Close overlay

---

## Tool Approval Workflow

The TUI implements a security-first approval system for potentially dangerous operations.

### When Approval is Required

The agent will request approval for:
- **File writes**: Creating or modifying files
- **File deletions**: Removing files
- **Command execution**: Running shell commands
- **Git operations**: Commits, pushes, PR creation
- **Any custom tools** marked as requiring approval

### Approval Dialog

When approval is needed, an overlay appears showing:

```
┌─────────────────────────────────────────┐
│ Tool Approval Required                  │
├─────────────────────────────────────────┤
│                                         │
│ Tool: write_file                        │
│                                         │
│ Parameters:                             │
│   path: src/main.go                     │
│   content: package main...              │
│                                         │
│ Description:                            │
│ Write content to a file, creating it    │
│ if it doesn't exist or overwriting if   │
│ it does.                                │
│                                         │
├─────────────────────────────────────────┤
│  [ Approve ]  [ Deny ]                  │
└─────────────────────────────────────────┘
```

### Making a Decision

**To Approve:**
- Press **Tab** to select "Approve" button
- Press **Enter** to confirm
- Or press **a** for quick approval

**To Deny:**
- Press **Tab** to select "Deny" button
- Press **Enter** to confirm
- Or press **d** for quick denial
- Or press **Esc** to cancel

### Auto-Approval Rules

You can configure auto-approval for trusted operations in Settings:

1. Open settings with `/settings`
2. Navigate to the "Auto-Approval" tab
3. Enable rules for:
   - Read-only operations (always safe)
   - Writes to specific paths
   - Specific commands
   - Trusted tools

**Example auto-approval rules:**
- ✓ Auto-approve all read operations
- ✓ Auto-approve writes to `/tmp/*`
- ✓ Auto-approve `git status`, `git diff`
- ✗ Never auto-approve deletions

---

## Settings Configuration

Access settings with the `/settings` command.

### Settings Tabs

#### General Tab
- **Workspace Path**: Current working directory
- **Max Iterations**: Maximum agent loop iterations
- **Enable Toast Notifications**: Show status toasts

#### LLM Tab
- **Provider**: Select LLM provider (OpenAI, Anthropic, etc.)
- **Model**: Choose specific model
- **API Key**: Configure authentication
- **Temperature**: Control response randomness
- **Max Tokens**: Set maximum response length

#### Auto-Approval Tab
- **Read Operations**: Auto-approve file reads
- **Write Operations**: Configure write rules
- **Path Patterns**: Whitelist/blacklist paths
- **Command Patterns**: Trusted commands
- **Tool-Specific Rules**: Per-tool configuration

#### Display Tab
- **Theme**: Color scheme selection
- **Font Size**: Adjust text size
- **Syntax Highlighting**: Enable/disable
- **Show Line Numbers**: In code displays
- **Diff Style**: Unified vs side-by-side

### Saving Settings

Settings are automatically saved when you:
1. Press **Ctrl+S** in the settings overlay
2. Click "Save" button
3. Settings are persisted to `~/.config/forge/settings.json`

### Resetting Settings

To reset to defaults:
1. Open settings overlay
2. Navigate to the "General" tab
3. Click "Reset to Defaults" button
4. Confirm the action

---

## Tips & Best Practices

### Effective Communication

1. **Be Specific**: Provide clear, detailed requests
   - ❌ "Make the code better"
   - ✓ "Refactor the authentication function to use async/await"

2. **Provide Context**: Reference existing code or files
   - ✓ "In src/auth.js, add error handling to the login function"

3. **Break Down Complex Tasks**: Split large requests into steps
   - ✓ "First, create the database schema. Then, implement the models."

### Working with Files

1. **Review Changes**: Always review diffs before approving writes
2. **Use Version Control**: Commit frequently to track changes
3. **Backup Important Files**: Before major refactoring

### Managing Long Conversations

1. **Use `/context`**: Check token usage periodically
2. **Start Fresh**: When context gets too large, start a new session
3. **Summarize**: Ask the agent to summarize work done

### Performance Tips

1. **Close Unused Overlays**: Press Esc to dismiss overlays
2. **Limit Output**: For large command outputs, use grep or head
3. **Batch Operations**: Group related file changes together

### Security Best Practices

1. **Review Tool Calls**: Always read approval dialogs carefully
2. **Verify Paths**: Check that file paths are correct before approving
3. **Audit Commands**: Review shell commands before execution
4. **Use Auto-Approval Carefully**: Only for truly trusted operations

### Keyboard Efficiency

1. **Learn Shortcuts**: Master the essential keyboard shortcuts
2. **Use Command Palette**: Type `/` for quick access to features
3. **Navigate with Keys**: Use arrow keys instead of mouse

### Troubleshooting

**Agent Not Responding:**
- Check network connection (for cloud LLMs)
- Verify API key in settings
- Press `/stop` and try again

**Overlays Not Closing:**
- Press **Esc** multiple times
- Try **Ctrl+C** to force close
- Restart TUI if needed

**Settings Not Saving:**
- Check file permissions in `~/.config/forge/`
- Verify disk space
- Check error messages in logs

**Tool Approval Stuck:**
- Press **Esc** to deny and close
- Use `/stop` to cancel operation
- Check agent logs for errors

---

## Related Documentation

- [Getting Started Guide](../getting-started/quick-start.md)
- [Understanding the Agent Loop](../getting-started/understanding-agent-loop.md)
- [Slash Commands Design](../plans/slash-commands-design.md)
- [Settings Architecture](../plans/settings-architecture.md)
- [Tool Approval System](../plans/auto-approval-and-settings.md)

---

## Next Steps

Now that you know how to use the TUI, explore:

1. **[Create Custom Tools](create-custom-tool.md)** - Extend agent capabilities
2. **[Configure LLM Providers](configure-provider.md)** - Set up different AI models
3. **[Manage Memory](manage-memory.md)** - Control conversation context
4. **[Handle Errors](handle-errors.md)** - Debug and recover from issues

---

**Questions or Issues?**

- Check the [FAQ](../FAQ.md)
- Review [Troubleshooting Guide](../community/troubleshooting.md) *(coming soon)*
- Open an issue on [GitHub](https://github.com/entrhq/forge/issues)
