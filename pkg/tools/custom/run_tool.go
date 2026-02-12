package custom

import (
	"context"
	"encoding/xml"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/security/workspace"
)

// RunCustomToolTool executes custom tools from ~/.forge/tools/
type RunCustomToolTool struct {
	registry *Registry
	guard    *workspace.Guard
}

// GetRegistry returns the custom tools registry for prompt building
func (t *RunCustomToolTool) GetRegistry() *Registry {
	return t.registry
}

// NewRunCustomToolTool creates a new run_custom_tool tool instance
func NewRunCustomToolTool(guard *workspace.Guard) *RunCustomToolTool {
	registry := NewRegistry()
	// Do initial refresh to populate the registry
	_ = registry.Refresh()
	return &RunCustomToolTool{
		registry: registry,
		guard:    guard,
	}
}

func (t *RunCustomToolTool) Name() string {
	return "run_custom_tool"
}

func (t *RunCustomToolTool) Description() string {
	return "Execute a custom tool from ~/.forge/tools/. Discovers available tools, converts arguments to CLI flags, and executes the tool binary."
}

func (t *RunCustomToolTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"tool_name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the custom tool to execute (must exist in ~/.forge/tools/)",
			},
			"arguments": map[string]interface{}{
				"type":        "object",
				"description": "Tool-specific arguments as direct XML elements (not nested). Each becomes a CLI flag (e.g., <count>20</count> → --count=20)",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "Execution timeout in seconds (default: 30)",
			},
		},
		[]string{"tool_name"},
	)
}

func (t *RunCustomToolTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var input runCustomToolInput
	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.ToolName == "" {
		return "", nil, fmt.Errorf("tool_name is required and must be a non-empty string")
	}

	// Get timeout with default
	timeout := 30.0
	if input.Timeout > 0 {
		timeout = input.Timeout
	}

	// Check if tool exists in registry
	_, exists := t.registry.Get(input.ToolName)
	if !exists {
		available := t.registry.List()
		if len(available) == 0 {
			return "", nil, fmt.Errorf("custom tool '%s' not found. No custom tools are currently available. Use create_custom_tool to create one", input.ToolName)
		}
		// Build list of available tool names
		var toolNames []string
		for _, tool := range available {
			toolNames = append(toolNames, tool.Name)
		}
		return "", nil, fmt.Errorf("custom tool '%s' not found. Available tools: %s", input.ToolName, strings.Join(toolNames, ", "))
	}

	// Get binary path
	binaryPath, err := t.registry.GetBinaryPath(input.ToolName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get tool binary path: %w", err)
	}

	// Convert arguments to CLI flags
	var flags []string
	for key, value := range input.Arguments {
		flags = append(flags, fmt.Sprintf("--%s=%v", key, value))
	}

	// Execute tool with timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, binaryPath, flags...)

	// Set working directory to workspace (so tools can access workspace files)
	cmd.Dir = t.guard.WorkspaceDir()

	// Capture output
	output, err := cmd.CombinedOutput()

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return "", nil, fmt.Errorf("tool execution timed out after %.0f seconds", timeout)
		}
		return "", nil, fmt.Errorf("tool execution failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil, nil
}

func (t *RunCustomToolTool) IsLoopBreaking() bool {
	return false
}

// XMLExample provides a concrete XML usage example for this tool.
func (t *RunCustomToolTool) XMLExample() string {
	return `<tool>
<server_name>local</server_name>
<tool_name>run_custom_tool</tool_name>
<arguments>
  <tool_name>recent-commits</tool_name>
  <count>20</count>
  <format>oneline</format>
</arguments>
</tool>`
}

// Refresh reloads the tool registry from disk
// This can be called periodically or after creating new tools
func (t *RunCustomToolTool) Refresh() error {
	return t.registry.Refresh()
}

// runCustomToolInput represents the XML input structure
type runCustomToolInput struct {
	XMLName   xml.Name               `xml:"arguments"`
	ToolName  string                 `xml:"tool_name"`
	Arguments map[string]interface{} `xml:"arguments,omitempty"`
	Timeout   float64                `xml:"timeout,omitempty"`
}
