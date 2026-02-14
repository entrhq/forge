# Built-in Tools Reference

Complete reference for all built-in tools available in the Forge coding agent framework.

## Table of Contents

- [Overview](#overview)
- [File Operations](#file-operations)
  - [read_file](#read_file)
  - [write_file](#write_file)
  - [list_files](#list_files)
  - [search_files](#search_files)
  - [apply_diff](#apply_diff)
- [Command Execution](#command-execution)
  - [execute_command](#execute_command)
- [Browser Automation](#browser-automation)
  - [start_session](#start_session)
  - [close_session](#close_session)
  - [list_sessions](#list_sessions)
  - [navigate](#navigate)
  - [click](#click)
  - [fill](#fill)
  - [search](#search)
  - [extract_content](#extract_content)
  - [analyze_page](#analyze_page)
  - [wait](#wait)
- [Agent Control](#agent-control)
  - [task_completion](#task_completion)
  - [ask_question](#ask_question)
  - [converse](#converse)
- [Security & Best Practices](#security--best-practices)

---

## Overview

Forge provides a comprehensive set of built-in tools that enable the coding agent to interact with the filesystem, execute commands, and control the conversation flow. All tools are designed with:

- **Workspace Security**: All file operations are restricted to the workspace directory
- **Validation**: Input validation and error handling
- **Streaming Support**: Real-time output for long-running operations
- **Preview Generation**: Pre-execution previews for destructive operations

---

## File Operations

### read_file

Read the contents of a file with optional line range support.

**Server Name**: `local`

**Parameters**:
- `path` (string, required): Path to the file to read (relative to workspace)
- `start_line` (integer, optional): Starting line number (1-based, inclusive)
- `end_line` (integer, optional): Ending line number (1-based, inclusive)

**Returns**: Line-numbered file content

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>read_file</tool_name>
<arguments>
  <path>src/main.go</path>
  <start_line>1</start_line>
  <end_line>50</end_line>
</arguments>
</tool>
```

**Features**:
- Returns content with line numbers for easy reference
- Supports reading specific line ranges for large files
- Respects `.gitignore` and `.forgeignore` patterns
- Validates all paths are within workspace

**Implementation**: `pkg/tools/coding/read_file.go`

---

### write_file

Write content to a file, creating it if it doesn't exist or overwriting if it does.

**Server Name**: `local`

**Parameters**:
- `path` (string, required): Path to the file to write (relative to workspace)
- `content` (string, required): Content to write to the file

**Returns**: Success message indicating file created or overwritten

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>write_file</tool_name>
<arguments>
  <path>src/hello.go</path>
  <content><![CDATA[package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
]]></content>
</arguments>
</tool>
```

**Features**:
- Automatically creates parent directories as needed
- Atomic write operation using temporary files
- Generates diff previews for existing files
- Sets appropriate file permissions (0600)

**Implementation**: `pkg/tools/coding/write_file.go`

---

### list_files

List files and directories in a specified path with optional filtering.

**Server Name**: `local`

**Parameters**:
- `path` (string, optional): Directory path to list (relative to workspace, defaults to workspace root)
- `recursive` (boolean, optional): Whether to list files recursively (default: false)
- `pattern` (string, optional): Glob pattern to filter files (e.g., '*.go', 'test_*.py')

**Returns**: Formatted list of files and directories with sizes

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>list_files</tool_name>
<arguments>
  <path>pkg</path>
  <recursive>true</recursive>
  <pattern>*.go</pattern>
</arguments>
</tool>
```

**Features**:
- Supports recursive directory traversal
- Glob pattern filtering for file selection
- Respects `.gitignore` and `.forgeignore` patterns
- Human-readable file sizes (KB, MB, GB)
- Sorted output (directories first, then alphabetically)

**Implementation**: `pkg/tools/coding/list_files.go`

---

### search_files

Search for patterns in files using regular expressions.

**Server Name**: `local`

**Parameters**:
- `pattern` (string, required): Regular expression pattern to search for
- `path` (string, optional): Directory path to search in (relative to workspace, defaults to workspace root)
- `file_pattern` (string, optional): Glob pattern to filter files (e.g., '*.go', '*.py')
- `context_lines` (integer, optional): Number of context lines to show before and after match (default: 2)

**Returns**: Matches with surrounding context lines

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>search_files</tool_name>
<arguments>
  <pattern>func\s+\w+\(.*\)\s+error</pattern>
  <file_pattern>*.go</file_pattern>
  <context_lines>3</context_lines>
</arguments>
</tool>
```

**Features**:
- Full regular expression support
- Configurable context lines around matches
- File pattern filtering for targeted searches
- Automatically skips binary files
- Respects `.gitignore` and `.forgeignore` patterns
- Line-numbered output for easy reference

**Implementation**: `pkg/tools/coding/search_files.go`

---

### apply_diff

Apply precise search/replace operations to files for surgical code changes.

**Server Name**: `local`

**Parameters**:
- `path` (string, required): Path to the file to edit (relative to workspace)
- `edits` (array, required): List of search/replace operations to apply
  - Each edit contains:
    - `search` (string, required): Exact text to search for (must match exactly including whitespace)
    - `replace` (string, required): Text to replace the search text with

**Returns**: Success message with number of edits applied

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>apply_diff</tool_name>
<arguments>
  <path>src/main.go</path>
  <edits>
    <edit>
      <search><![CDATA[func oldFunction() {
	return "old"
}]]></search>
      <replace><![CDATA[func newFunction() {
	return "new"
}]]></replace>
    </edit>
    <edit>
      <search><![CDATA[const oldValue = 42]]></search>
      <replace><![CDATA[const newValue = 100]]></replace>
    </edit>
  </edits>
</arguments>
</tool>
```

**Features**:
- Multiple edits in a single operation
- Exact string matching (including whitespace)
- Validates search text exists and is unique
- Atomic file updates using temporary files
- Generates unified diff previews
- Fails fast if search text not found or appears multiple times

**Best Practices**:
- Use `read_file` first to see exact content
- Include enough context to make search unique
- Use CDATA sections for multi-line content
- Test edits are unique before applying

**Implementation**: `pkg/tools/coding/apply_diff.go`

---

## Command Execution

### execute_command

Execute a shell command in the workspace directory with timeout and streaming support.

**Server Name**: `local`

**Parameters**:
- `command` (string, required): The shell command to execute
- `timeout` (number, optional): Command timeout in seconds (default: 30)
- `working_dir` (string, optional): Working directory relative to workspace (default: workspace root)

**Returns**: Command output (stdout and stderr) with exit code

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>execute_command</tool_name>
<arguments>
  <command>go test ./...</command>
  <timeout>60</timeout>
</arguments>
</tool>
```

**Features**:
- Configurable timeout (default 30 seconds)
- Real-time output streaming for TUI
- Captures both stdout and stderr
- Returns exit code for error handling
- Validates working directory is within workspace
- Command execution tracking with unique IDs
- Cancellation support through context

**Security Considerations**:
- Commands run in workspace directory only
- No automatic approval - requires user confirmation in TUI
- Timeout prevents hanging commands
- Shell injection protection through context cancellation

**Implementation**: `pkg/tools/coding/execute_command.go`

---

## Browser Automation

Tools for controlling a headless browser to perform web automation tasks. Powered by Playwright.

### start_session

Start a new browser session.

**Server Name**: `local`

**Parameters**:
- `session_id` (string, optional): A unique ID for the session. If not provided, one will be generated.
- `headless` (boolean, optional): Whether to run the browser in headless mode (default: true).

**Returns**: The session ID of the new browser session.

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>start_session</tool_name>
<arguments>
  <session_id>my-web-session</session_id>
  <headless>true</headless>
</arguments>
</tool>
```

**Features**:
- Creates an isolated browser context.
- Supports multiple concurrent sessions.
- Automatic cleanup of timed-out sessions.

**Implementation**: `pkg/tools/browser/start_session.go`

---
### close_session

Close an existing browser session.

**Server Name**: `local`

**Parameters**:
- `session_id` (string, required): The ID of the browser session to close.

**Returns**: A success message.

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>close_session</tool_name>
<arguments>
  <session_id>my-web-session</session_id>
</arguments>
</tool>
```

**Implementation**: `pkg/tools/browser/close_session.go`

---

### list_sessions

List all active browser sessions.

**Server Name**: `local`

**Parameters**: None

**Returns**: A list of active session IDs and their statuses.

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>list_sessions</tool_name>
<arguments/>
</tool>
```

**Implementation**: `pkg/tools/browser/list_sessions.go`

---

### navigate

Navigate the browser to a specific URL.

**Server Name**: `local`

**Parameters**:
- `session_id` (string, required): The ID of the browser session.
- `url` (string, required): The URL to navigate to.

**Returns**: A success message with the final URL.

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>navigate</tool_name>
<arguments>
  <session_id>my-web-session</session_id>
  <url>https://www.google.com</url>
</arguments>
</tool>
```

**Implementation**: `pkg/tools/browser/navigate.go`

---

### click

Click on an element on the page.

**Server Name**: `local`

**Parameters**:
- `session_id` (string, required): The ID of the browser session.
- `selector` (string, required): The CSS selector of the element to click.

**Returns**: A success message.

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>click</tool_name>
<arguments>
  <session_id>my-web-session</session_id>
  <selector>#submit-button</selector>
</arguments>
</tool>
```

**Implementation**: `pkg/tools/browser/click.go`

---

### fill

Fill an input field on the page.

**Server Name**: `local`

**Parameters**:
- `session_id` (string, required): The ID of the browser session.
- `selector` (string, required): The CSS selector of the input field.
- `text` (string, required): The text to fill into the input field.

**Returns**: A success message.

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>fill</tool_name>
<arguments>
  <session_id>my-web-session</session_id>
  <selector>#username</selector>
  <text>my-user</text>
</arguments>
</tool>
```

**Implementation**: `pkg/tools/browser/fill.go`

---

### search

Perform a search on the page, typically within a search bar.

**Server Name**: `local`

**Parameters**:
- `session_id` (string, required): The ID of the browser session.
- `selector` (string, optional): The CSS selector for the search input. Defaults to common search selectors.
- `query` (string, required): The search query.

**Returns**: A success message.

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>search</tool_name>
<arguments>
  <session_id>my-web-session</session_id>
  <query>AI agents</query>
</arguments>
</tool>
```

**Implementation**:
`pkg/tools/browser/search.go`

---

### extract_content

Extract content from the page, optionally filtered by a selector.

**Server Name**: `local`

**Parameters**:
- `session_id` (string, required): The ID of the browser session.
- `selector` (string, optional): The CSS selector to extract content from. If not provided, extracts from the entire page.

**Returns**: The extracted content.

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>extract_content</tool_name>
<arguments>
  <session_id>my-web-session</session_id>
  <selector>.main-content</selector>
</arguments>
</tool>
```

**Implementation**: `pkg/tools/browser/extract_content.go`

---

### analyze_page

Use an AI model to analyze the current page content and identify key components.

**Server Name**: `local`

**Parameters**:
- `session_id` (string, required): The ID of the browser session.
- `objective` (string, required): The objective for the analysis (e.g., "Find the login form").

**Returns**: A structured analysis of the page, including identified components and their selectors.

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>analyze_page</tool_name>
<arguments>
  <session_id>my-web-session</session_id>
  <objective>Find the main navigation bar</objective>
</arguments>
</tool>
```

**Implementation**: `pkg/tools/browser/analyze_page.go`

---

### wait

Wait for a specific amount of time or for an element to appear.

**Server Name**: `local`

**Parameters**:
- `session_id` (string, required): The ID of the browser session.
- `selector` (string, optional): The CSS selector of an element to wait for.
- `timeout` (integer, optional): The maximum time to wait in seconds.

**Returns**: A success message.

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>wait</tool_name>
<arguments>
  <session_id>my-web-session</session_id>
  <selector>#dynamic-content</selector>
  <timeout>10</timeout>
</arguments>
</tool>
```

**Implementation**: `pkg/tools/browser/wait.go`

---

## Agent Control

These tools control the agent's conversation flow and are "loop-breaking" - they end the current agent turn.

### task_completion

Signal that the task is complete and present the final result to the user.

**Server Name**: `local`

**Parameters**:
- `result` (string, required): The final result of the task. Should be clear, complete, and not end with questions or offers for further assistance.

**Returns**: The final result (presented to user)

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>task_completion</tool_name>
<arguments>
  <result>I've successfully created the new user authentication module. The implementation includes password hashing with bcrypt, session management, and input validation. All tests are passing.</result>
</arguments>
</tool>
```

**When to Use**:
- Task is fully completed
- All requested work is done
- No further user input needed
- Ready to present final deliverables

**Loop Breaking**: ✅ Yes - Ends the agent loop and waits for new user input

**Implementation**: `pkg/agent/tools/task_completion.go`

---

### ask_question

Ask the user a clarifying question when additional information is needed.

**Server Name**: `local`

**Parameters**:
- `question` (string, required): A clear, specific question asking for the information needed to proceed with the task
- `suggestions` (array, optional): List of 2-4 suggested answers to help the user respond quickly

**Returns**: The question (presented to user)

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>ask_question</tool_name>
<arguments>
  <question>Which database would you like to use for this project?</question>
  <suggestions>
    <suggestion>PostgreSQL</suggestion>
    <suggestion>MySQL</suggestion>
    <suggestion>SQLite</suggestion>
  </suggestions>
</arguments>
</tool>
```

**When to Use**:
- Ambiguous requirements
- Multiple valid approaches exist
- Missing critical information
- User preference needed

**Best Practices**:
- Ask specific, actionable questions
- Provide 2-4 suggestions when possible
- Don't ask questions you can reasonably infer
- Ask early rather than making wrong assumptions

**Loop Breaking**: ✅ Yes - Ends the agent loop and waits for user response

**Implementation**: `pkg/agent/tools/ask_question.go`

---

### converse

Engage in conversation or provide information without completing a task.

**Server Name**: `local`

**Parameters**:
- `message` (string, required): A conversational message to share with the user. Can include information, explanations, or casual responses.

**Returns**: The message (presented to user)

**Example**:
```xml
<tool>
<server_name>local</server_name>
<tool_name>converse</tool_name>
<arguments>
  <message>That's a great question! The main difference between interface{} and any in Go is purely semantic - they're identical at runtime. The 'any' alias was introduced in Go 1.18 to make generic code more readable.</message>
</arguments>
</tool>
```

**When to Use**:
- Answering general questions
- Providing explanations
- Casual conversation
- No action required

**When NOT to Use**:
- Task completion (use `task_completion`)
- Asking questions (use `ask_question`)
- During task execution (just continue working)

**Loop Breaking**: ✅ Yes - Ends the agent loop and waits for user response

**Implementation**: `pkg/agent/tools/converse.go`

---

## Security & Best Practices

### Workspace Security

All file operations are protected by the **WorkspaceGuard**:

1. **Path Validation**: All paths must be within workspace
2. **Ignore Patterns**: Respects `.gitignore` and `.forgeignore`
3. **Traversal Protection**: Prevents `../` attacks
4. **Absolute Path Resolution**: Validates final resolved paths

**Example Protections**:
```go
// ❌ Rejected - outside workspace
read_file(path: "/etc/passwd")

// ❌ Rejected - traversal attempt  
read_file(path: "../../secrets.txt")

// ✅ Allowed - within workspace
read_file(path: "src/main.go")
```

### Best Practices

**File Operations**:
- Use `read_file` with line ranges for large files
- Prefer `apply_diff` over `write_file` for edits
- Always check file existence before operations
- Use relative paths from workspace root

**Search Operations**:
- Use specific file patterns to narrow search
- Start with simple patterns, refine as needed
- Consider context lines for understanding matches

**Command Execution**:
- Set appropriate timeouts for long operations
- Validate commands before execution
- Check exit codes for error handling
- Stream output for user feedback

**Agent Control**:
- Use `task_completion` only when truly done
- Ask questions early to avoid rework
- Provide helpful suggestions with questions
- Use `converse` for information-only responses

### Error Handling

All tools return descriptive errors:

```go
// Path validation errors
"invalid path: path traverses outside workspace"
"file 'src/main.go' is ignored by .gitignore"

// File operation errors
"failed to read file: file not found"
"failed to write file: permission denied"

// Search/diff errors
"edit 1: search text not found in file"
"edit 2: search text appears 3 times in file, must be unique"

// Command execution errors
"command timed out after 30s"
"command exited with code 1"
```

### Performance Tips

**Large Files**:
- Use `start_line` and `end_line` with `read_file`
- Avoid reading entire large files unnecessarily
- Use `search_files` instead of reading all files

**Batch Operations**:
- Use multiple `edits` in single `apply_diff` call
- Combine related file operations when possible
- Consider streaming for long-running commands

**Search Optimization**:
- Use specific file patterns to reduce search scope
- Use precise regex patterns to minimize false matches
- Limit context lines to reduce output size

---

## Related Documentation

- [Tool Schema Reference](tool-schema.md) - JSON Schema details for tool parameters
- [Architecture: Tool System](../architecture/tool-system.md) - Tool system design
- [ADR-003: Tool Calling](../adrs/003-tool-calling.md) - Tool calling protocol
- [Testing Guide](../guides/testing.md) - Testing tool implementations
