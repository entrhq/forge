# PRD: Custom Tools System

## Overview

Enable Forge agents to create, manage, and execute custom tools that persist across conversations. Custom tools are user-defined programs (initially Go-based) that extend the agent's capabilities beyond built-in tools. These tools live in `~/.forge/tools/` and become part of the agent's permanent toolkit.

## Problem Statement

Currently, Forge agents are limited to a fixed set of built-in tools. When agents need to perform repeated complex operations or workflows, they must execute the same sequence of tool calls every time. This is inefficient and doesn't allow agents to learn and improve their capabilities over time.

Users and agents need a way to:
- Encapsulate complex workflows into reusable tools
- Extend Forge's capabilities without modifying core code
- Build domain-specific tools for specialized tasks
- Enable agents to "learn" new capabilities that persist across sessions

## Goals

1. **Enable Tool Creation**: Agents can create new custom tools with user approval
2. **Persistence**: Custom tools persist in `~/.forge/tools/` and are available across all Forge sessions
3. **Simplicity**: Tool creation follows a simple scaffold → edit → compile workflow
4. **Integration**: Custom tools integrate seamlessly with the existing tool system
5. **Security**: Maintain workspace isolation while allowing tool creation outside workspace

## Non-Goals

- Shell script tools (future enhancement)
- Tool marketplace or sharing (future enhancement)
- Automatic tool discovery from other sources
- Version management or dependency resolution
- Sandboxed tool execution

## User Stories

### Agent Perspective

**As an agent, I want to:**
- Create custom tools when I identify repeated workflows
- Use custom tools just like built-in tools
- Iterate on tool implementations to improve them
- Build specialized tools for domain-specific tasks

**Example:**
```
User: "Can you summarize the git commits from last week?"
Agent: *Realizes this is a common request, asks permission to create a tool*
Agent: *Creates git-summarize tool using create_custom_tool*
Agent: *Implements the logic, compiles it*
Agent: *Uses the new tool to complete the request*
Future: Agent can use git-summarize tool immediately in any conversation
```

### User Perspective

**As a user, I want to:**
- Approve tool creation before it happens
- Have custom tools available across all my Forge sessions
- Trust that tools run in my environment securely
- Be able to inspect and modify tools the agent creates

## Success Metrics

- Number of custom tools created per user
- Custom tool reuse rate (how often tools are called after creation)
- Tool creation success rate (compilation success)
- Time saved by reusing custom tools vs. repeated manual workflows

## Technical Design

### Architecture

**Components:**
1. **Tool Registry**: In-memory registry of available custom tools, refreshed each agent turn
2. **Tool Scaffolder**: Generates boilerplate Go code and YAML metadata
3. **Tool Executor**: Wrapper that converts tool calls to execute_command calls
4. **File System**: `~/.forge/tools/` whitelisted for agent writes

**Directory Structure:**
```
~/.forge/tools/
├── git-summarize/
│   ├── tool.yaml          # Tool metadata
│   ├── git-summarize.go   # Source code
│   └── git-summarize      # Compiled binary
├── analyze-logs/
│   ├── tool.yaml
│   ├── analyze-logs.go
│   └── analyze-logs
```

### Tool Metadata Schema (YAML)

```yaml
name: tool-name
description: What the tool does
version: 1.0.0
entrypoint: tool-name  # Points to compiled binary (not .go file)
usage: |
  Detailed usage instructions for the agent.
  Explain parameters, expected inputs, and outputs.
parameters:
  - name: param_name
    type: string  # string, number, boolean
    required: true
    description: What this parameter does
  - name: optional_param
    type: number
    required: false
    description: Optional parameter with default behavior
```

### Go Boilerplate Template

Generated Go code includes:
- Package main with minimal imports
- Flag parsing setup using stdlib `flag` package
- Environment variable reading pattern
- Output helper function for consistent stdout formatting
- Error handling with exit codes
- Main function skeleton

Example structure:
```go
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Flag parsing
	param1 := flag.String("param1", "", "Description")
	flag.Parse()

	// Environment variables
	envVar := os.Getenv("FORGE_WORKSPACE")

	// Main logic (agent implements this)
	// ...

	// Output
	writeOutput("result")
}

func writeOutput(result string) {
	fmt.Println(result)
}
```

### Tool Creation Workflow

1. **Scaffold**: Agent calls `create_custom_tool` with name and description
   - Creates `~/.forge/tools/{name}/` directory
   - Generates `tool.yaml` with basic metadata
   - Generates `{name}.go` with boilerplate code

2. **Implement**: Agent edits the generated files
   - Uses `write_file` or `apply_diff` to modify `.go` file
   - Updates `tool.yaml` parameters array

3. **Compile**: Agent compiles the Go source
   - Runs `go build -o {name} {name}.go` via `execute_command`
   - Handles compilation errors by fixing code

4. **Update Metadata**: Agent updates YAML entrypoint
   - Changes `entrypoint: {name}.go` to `entrypoint: {name}`

5. **Execute**: Tool is now available
   - Registry refreshes on next agent turn
   - Agent can call tool via `run_custom_tool`

### Tool Execution

**run_custom_tool** is a wrapper around execute_command:

1. Lookup tool in registry by name
2. Convert arguments object to CLI flags
   - `{"since": "2024-01-01", "limit": 10}` → `--since=2024-01-01 --limit=10`
3. Construct command with binary path from YAML
4. Call `execute_command` with command, working_dir, timeout
5. Return stdout, stderr, exit code to agent

**Parameters:**
- `tool_name` (string, required): Name of custom tool
- `arguments` (object): Key-value pairs for tool parameters
- `timeout` (number): Execution timeout in seconds (default: 30)

### Security Model

**Workspace Isolation:**
- `~/.forge/tools/` is whitelisted for agent writes (exception to workspace guard)
- Custom tool binaries are validated to be within whitelisted directories before execution
- Tool execution working directory is validated before running commands
- Agent must request user approval before creating or running tools

**Safety Mechanisms:**
- Tool creation requires explicit user approval (via Previewable interface)
- Tool execution requires explicit user approval (via Previewable interface)
- Tools are visible and inspectable in `~/.forge/tools/`
- Users can delete or modify tools manually
- Path traversal protection on tool names and entrypoints
- Binary path validation before execution
- Compilation errors prevent broken tools from being available

### Tool Discovery and Loading

**On Each Agent Turn:**
1. Scan `~/.forge/tools/` for subdirectories
2. Read `tool.yaml` from each subdirectory
3. Validate YAML schema and verify binary exists
4. Register tool in in-memory map: `name → ToolMetadata`

**Performance:**
- File system scan is lightweight (only reading YAML files)
- No need for explicit reload command
- Newly created tools available immediately on next turn

## Implementation Plan

### Phase 1: Core Infrastructure
1. Implement tool registry with YAML parsing
2. Whitelist `~/.forge/tools/` in workspace guard
3. Create `create_custom_tool` tool with scaffolding logic
4. Define Go boilerplate template

### Phase 2: Execution
1. Implement `run_custom_tool` wrapper
2. Add argument-to-flag conversion logic
3. Integrate with execute_command

### Phase 3: Agent Integration
1. Add tool building guidance to system prompt
2. Test tool creation workflow end-to-end
3. Document patterns and best practices

### Phase 4: Polish
1. Error handling improvements
2. Validation and helpful error messages
3. Tool management utilities (list, delete)

## Future Enhancements

- **Shell Script Support**: Allow `.sh` tools with similar metadata
- **Argument Validation**: Validate arguments against YAML schema before execution
- **Tool Versioning**: Support multiple versions of the same tool
- **Tool Discovery**: Find and suggest tools from community repositories
- **Dependency Management**: Handle tool dependencies and build requirements
- **Tool Testing**: Framework for testing custom tools
- **Tool Documentation**: Auto-generate docs from YAML and code

## Open Questions

1. Should we support tools that depend on external packages (`go get` dependencies)?
2. How should we handle tool updates/versioning?
3. Should there be a limit on number of custom tools?
4. What about tools that require long-running processes or background tasks?

## References

- ADR 0019: XML Tool Call Format
- ADR 0026: Headless Mode
- Custom Tools System Scratch Document: `docs/product/scratch/custom-tools-system.md`
