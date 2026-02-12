# 37. Custom Tools System

**Status:** Proposed
**Date:** 2024-01-09
**Deciders:** Engineering Team
**Technical Story:** Enable agents to create, manage, and execute persistent custom tools that extend Forge capabilities beyond built-in tools.

---

## Context

Forge agents currently operate with a fixed set of built-in tools defined at compile time. When agents need to perform complex or repeated operations, they must execute the same sequence of tool calls every time. This creates several limitations:

1. **No Learning**: Agents cannot learn new capabilities that persist across sessions
2. **Inefficiency**: Repeated workflows require the same tool call sequences each time
3. **Limited Extensibility**: Adding new tools requires modifying core Forge code
4. **Domain Specificity**: Users with specialized needs cannot add domain-specific tools

### Background

The tool system (ADR-0011, ADR-0019) provides a clean abstraction for tool execution with XML-based calling conventions. The workspace guard system ensures security by restricting file operations to the workspace directory. However, this security model needs extension to support tool creation outside the workspace while maintaining safety.

### Problem Statement

How can we enable agents to create persistent, executable tools that extend their capabilities while maintaining security, simplicity, and integration with the existing tool system?

### Goals

- Enable agents to create custom Go-based tools with user approval
- Tools persist in `~/.forge/tools/` and are available across all sessions
- Simple scaffold → edit → compile workflow
- Seamless integration with existing tool execution system
- Maintain workspace isolation with controlled exceptions

### Non-Goals

- Shell script tools (future enhancement)
- Tool marketplace or distribution system
- Automatic dependency resolution
- Sandboxed execution environment
- Version management in first iteration

---

## Decision Drivers

* **Simplicity**: Tool creation must be straightforward for agents
* **Security**: Cannot compromise workspace isolation security model
* **Performance**: Tool discovery must not add significant overhead
* **Integration**: Custom tools should work like built-in tools from agent perspective
* **Persistence**: Tools must survive across sessions and Forge versions
* **User Control**: Users must approve and be able to inspect/modify tools

---

## Considered Options

### Option 1: Go Plugins with Dynamic Loading

**Description:** Use Go's plugin system to compile tools as .so files that are loaded dynamically at runtime.

**Pros:**
- Native Go integration
- Can share types with Forge
- Potentially better performance

**Cons:**
- Plugin system is fragile and platform-specific
- Versioning issues with Go toolchain
- Requires exact Go version match
- Complex debugging and error handling
- Limited cross-platform support

### Option 2: Executable Scripts with YAML Metadata

**Description:** Custom tools are standalone executables (Go binaries or shell scripts) with YAML metadata defining their interface.

**Pros:**
- Simple and portable
- Works across platforms
- Easy to inspect and debug
- Clear separation from Forge binary
- No versioning issues
- Can start with Go, add shell later

**Cons:**
- Requires spawning processes
- Slightly higher execution overhead
- Need argument marshaling layer

### Option 3: Declarative YAML Tools

**Description:** Tools defined purely in YAML with limited scripting capabilities.

**Pros:**
- No compilation required
- Very simple security model
- Easy to validate

**Cons:**
- Extremely limited capabilities
- Would need custom scripting language
- Not suitable for complex logic
- Doesn't meet extensibility goals

---

## Decision

**Chosen Option:** Option 2 - Executable Scripts with YAML Metadata

We will implement custom tools as standalone executables with YAML metadata files, starting with Go-based tools and leaving room for shell script support later.

### Rationale

This approach provides the best balance of simplicity, security, and extensibility:

1. **Simplicity**: Clear workflow - scaffold, edit, compile, execute
2. **Portability**: Standard executables work everywhere Go works
3. **Debuggability**: Tools can be tested independently of Forge
4. **Security**: Clean separation with controlled filesystem access
5. **Flexibility**: Can support multiple languages in the future
6. **No Versioning Issues**: Tools are independent binaries

The overhead of process spawning is acceptable given the benefits, and we can optimize execution patterns if needed.

---

## Consequences

### Positive

- Agents can learn and expand capabilities over time
- Users can create domain-specific tools without forking Forge
- Clear separation between Forge core and custom extensions
- Tools are easily inspectable and modifiable by users
- Simple to debug and test tools independently
- Future-proof for additional language support

### Negative

- Process spawning overhead for each tool execution
- Need to manage tool lifecycle (creation, compilation, updates)
- Additional complexity in workspace guard for whitelisting
- Potential security risk if users don't review generated tools
- No type safety between Forge and custom tools

### Neutral

- Tools live in `~/.forge/tools/` rather than in workspace
- Registry refresh on each agent turn adds small overhead
- Need new tools: create_custom_tool, run_custom_tool

---

## Implementation

### Architecture

**Components:**

1. **Tool Registry** (`pkg/tools/registry/`)
   - In-memory map of custom tools
   - Refreshed each agent turn via filesystem scan
   - Validates YAML schema and binary existence

2. **Tool Scaffolder** (`pkg/tools/custom/scaffold.go`)
   - Generates boilerplate Go code
   - Creates YAML metadata template
   - Handles directory structure creation

3. **Tool Executor** (`pkg/tools/custom/executor.go`)
   - Wraps execute_command for tool execution
   - Converts arguments to CLI flags
   - Handles timeout and error reporting

4. **Workspace Guard Extension** (`pkg/security/guard.go`)
   - Whitelist `~/.forge/tools/` for writes
   - Maintain security for all other paths

**Directory Structure:**
```
~/.forge/tools/
├── git-summarize/
│   ├── tool.yaml
│   ├── git-summarize.go
│   └── git-summarize          # compiled binary
├── analyze-logs/
│   ├── tool.yaml
│   ├── analyze-logs.go
│   └── analyze-logs
```

### YAML Schema

```yaml
name: string              # Tool identifier (matches directory name)
description: string       # What the tool does
version: string           # Semantic version (e.g., "1.0.0")
entrypoint: string        # Compiled binary name (not .go file)
usage: string             # Multi-line usage instructions for agent
parameters:
  - name: string          # Parameter identifier
    type: string          # string | number | boolean
    required: boolean     # Is this parameter required?
    description: string   # What this parameter does
```

### Go Boilerplate Template

```go
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define flags
	// TODO: Add your parameters here
	exampleParam := flag.String("example", "", "Example parameter")
	flag.Parse()

	// Access environment variables if needed
	workspace := os.Getenv("FORGE_WORKSPACE")
	_ = workspace

	// TODO: Implement your tool logic here
	// Use *exampleParam to access flag values
	
	// Output results
	writeOutput(fmt.Sprintf("Result: %s", *exampleParam))
}

// writeOutput writes the tool result to stdout
func writeOutput(result string) {
	fmt.Println(result)
}
```

### Tool Creation Workflow

1. **Scaffold**: `create_custom_tool` generates structure
   ```
   - Creates ~/.forge/tools/{name}/
   - Writes tool.yaml with name, description, version
   - Writes {name}.go with boilerplate
   ```

2. **Implement**: Agent edits files
   ```
   - Uses write_file/apply_diff on .go source
   - Updates tool.yaml parameters array
   ```

3. **Compile**: Agent builds binary
   ```
   - execute_command: go build -o {name} {name}.go
   - Working directory: ~/.forge/tools/{name}/
   ```

4. **Update Metadata**: Agent fixes entrypoint
   ```
   - apply_diff on tool.yaml
   - Changes entrypoint from {name}.go to {name}
   ```

5. **Execute**: Tool immediately available
   ```
   - Registry refreshes on next turn
   - run_custom_tool executes via wrapper
   ```

### Tool Execution

`run_custom_tool` parameters:
- `tool_name` (string, required): Name of tool to execute
- `arguments` (object): Key-value pairs for tool parameters
- `timeout` (number): Execution timeout in seconds (default: 30)

Execution flow:
1. Lookup tool metadata in registry
2. Convert arguments to CLI flags: `{"since": "2024-01-01"}` → `--since=2024-01-01`
3. Construct command: `~/.forge/tools/{name}/{entrypoint} {flags}`
4. Call `execute_command` with constructed command, working_dir, timeout
5. Return stdout, stderr, exit code to agent

### Security Model

**Workspace Isolation:**
- `~/.forge/tools/` is whitelisted as exception to workspace guard
- All other filesystem restrictions remain in place
- Tools run with same permissions as Forge (user's environment)

**Safety Mechanisms:**
- Agent must request user approval before creating tools
- Tools visible at `~/.forge/tools/` for user inspection
- Compilation errors prevent broken tools from being available
- Users can manually delete or modify tools

**User Approval Flow:**
```
Agent: "I'd like to create a custom tool called 'git-summarize' to help 
       summarize git commits. This will create files in ~/.forge/tools/. 
       May I proceed?"
User: [Approves or denies]
```

### Tool Discovery

**On Each Agent Turn:**
1. Scan `~/.forge/tools/` for subdirectories
2. For each directory, read `tool.yaml`
3. Validate YAML schema
4. Verify binary at `entrypoint` path exists and is executable
5. Register in map: `toolName → CustomToolMetadata`

**Performance Considerations:**
- Scanning is lightweight (directory listing + YAML reads)
- Cached in memory for the turn
- No need for explicit reload mechanism
- Typical expected scale: 10-50 tools maximum

### Migration Path

No migration required - this is a new feature. Tools directory will be created on first use.

### Timeline

- Phase 1 (Week 1): Core infrastructure, registry, scaffolder
- Phase 2 (Week 2): Execution wrapper, integration testing
- Phase 3 (Week 3): Agent prompt updates, documentation, polish

---

## Validation

### Success Metrics

- Tool creation success rate (compilation succeeds >90% of time)
- Tool reuse rate (created tools are called multiple times)
- Number of custom tools per user (indicates adoption)
- Time saved by tool reuse vs. manual sequences

### Monitoring

- Log tool creation events and outcomes
- Track tool execution counts and success rates
- Monitor agent turn overhead from registry refresh
- Collect user feedback on tool creation experience

---

## Related Decisions

- [ADR-0011](0011-coding-tools-architecture.md) - Coding Tools Architecture
- [ADR-0019](0019-xml-cdata-tool-call-format.md) - XML/CDATA Tool Call Format
- [ADR-0027](0027-safety-constraint-system.md) - Safety Constraint System

---

## References

- [Custom Tools System PRD](../product/features/custom-tools-system.md)
- [Custom Tools Scratch Document](../product/scratch/custom-tools-system.md)
- [Tool Building System Prompt](../product/scratch/tool-building-system-prompt.md)

---

## Notes

**Future Enhancements:**
- Shell script support (add `.sh` detection)
- Argument validation against YAML schema
- Tool versioning and updates
- Tool marketplace/sharing
- Dependency management for Go packages
- Tool testing framework

**Open Questions:**
1. Should we support `go get` dependencies in tools? (Deferred to future)
2. How to handle tool name conflicts with built-ins? (Custom tools win, log warning)
3. Limit on number of tools? (No hard limit initially, monitor performance)
4. Long-running or background tools? (Not in first iteration)

**Last Updated:** 2025-02-12
