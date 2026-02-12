# Custom Tools System - Design Discussion

## Feature Overview

Build a tool building system that allows Forge users to create custom tools that persist in `~/.forge/tools/` directory. These tools are loaded at runtime and available to the agent, making each user's Forge instance progressively smarter and more capable over time.

## Core Principles

- **Code-first approach**: Tools solve problems through reusable actions/functions
- **User-specific**: Each user builds their own tool library
- **Persistent**: Tools are saved and loaded across sessions
- **Extensible**: Foundation for future plugin ecosystem (GitHub imports, etc.)

## Creation Workflow

### Trigger Mechanisms
1. **User-initiated**: User explicitly asks "create a tool for X"
2. **Agent-suggested**: Agent recognizes a pattern and asks user if it should create a tool
3. **Manual creation**: Users can write tools directly into `~/.forge/tools/` directory

### Tool Author
- Agent writes the tool code (Go or shell scripts)
- Humans can also manually create tools
- Future: Import tools from GitHub repos/plugin ecosystem

## Technical Implementation

### Supported Languages
- **Go**: Primary language for complex tools
- **Shell scripts**: For simpler automation tasks
- **Avoid**: TypeScript (OS-specific), WebAssembly (not needed)

### Directory Structure
```
~/.forge/tools/
  └── git-summarize/
      ├── tool.yaml          # Metadata and interface definition
      ├── git-summarize.go   # Source code (if Go-based)
      └── git-summarize      # Compiled binary (optional, cached)
```

**Folder-per-tool** approach for organization and clarity.

### Tool Metadata (tool.yaml)

```yaml
name: git-summarize
description: Summarizes git commit history for a given time period
version: 1.0.0
entrypoint: git-summarize.go  # or git-summarize (binary) or git-summarize.sh
usage: |
  Pass --since and --until as CLI args for simple usage.
  For complex filtering, create a config.json and pass --config=/path/to/config.json
parameters:
  - name: since
    type: string
    required: false
    description: Start date for commit range
  - name: until
    type: string
    required: false
    description: End date for commit range
  - name: config
    type: string
    required: false
    description: Path to JSON config file for complex filtering
```

### Input/Output Model

**Inputs:**
- **CLI arguments**: For simple parameters (flags like `--since`, `--until`)
- **Environment variables**: For configuration/context
- **JSON/YAML files**: For complex structured inputs
  - Agent writes a config file, passes path as CLI arg
  - Tool parses the file internally
  - Usage explained in YAML `usage` field

**Outputs:**
- Unstructured text result (stdout)
- Agent interprets the output
- No strict schema required

### Execution Model

**Priority: Compiled binaries**
- **Go tools**: Compile on first use, cache binary
  - Agent writes `.go` source code
  - Forge compiles with `go build` on first load
  - Updates `entrypoint` in YAML to point to binary (or maintains separate binary)
  - Subsequent uses run the compiled binary (faster)
  
**Fallback: Source execution**
- Support `go run` for development/debugging
- Shell scripts execute directly

**Entrypoint Detection:**
- YAML `entrypoint` field specifies the file to execute
- Could be: `tool-name.go`, `tool-name` (binary), `tool-name.sh`
- Forge determines execution method based on file extension/type

### Tool Interface

**Not required to follow Go Tool interface**, but must provide:
- Name
- Description  
- Input parameters (defined in YAML)
- Result output (text to stdout)

Tools are executed via wrapper that runs the command/binary with provided inputs.

## Open Questions

1. **Compilation strategy**: Always compile? Compile on first use? Optional build flag?
2. **Binary caching**: Where to store compiled binaries? Update entrypoint or keep separate?
3. **Dependencies**: How to handle Go tool dependencies (go.mod, external packages)?
4. **Versioning**: Should tools be versioned? How to handle updates?
5. **Security**: Sandboxing for user-created tools? Approval workflow?
6. **Error handling**: How to surface compilation errors? Runtime errors?

## Future Enhancements

- Tool plugin ecosystem (import from GitHub)
- Tool marketplace/sharing
- Automatic tool suggestions based on usage patterns
- Tool composition (tools that call other tools)
- Tool testing framework
