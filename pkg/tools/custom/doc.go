// Package custom provides the custom tools system, enabling agents to create,
// manage, and execute persistent Go-based tools that extend Forge capabilities.
//
// Custom tools are standalone executable programs with YAML metadata files that
// define their interface. They live in ~/.forge/tools/ and persist across sessions.
//
// Architecture:
//   - Registry: Scans ~/.forge/tools/ and loads tool metadata on each agent turn
//   - Scaffolder: Generates boilerplate Go code and YAML templates
//   - Executor: Wraps execute_command to run tools with argument conversion
//
// Example workflow:
//  1. Agent calls create_custom_tool to scaffold a new tool
//  2. Agent edits the generated .go file to implement logic
//  3. Agent compiles the tool with go build
//  4. Tool becomes available immediately via run_custom_tool
//
// Security:
//   - ~/.forge/tools/ is whitelisted in workspace guard
//   - Tools run with user's environment permissions
//   - User approval required before tool creation
package custom
