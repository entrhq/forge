package custom

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/tools"
)

// CreateCustomToolTool scaffolds a new custom tool with boilerplate code.
type CreateCustomToolTool struct {
	toolsDir string // Optional override for testing
}

// NewCreateCustomToolTool creates a new CreateCustomToolTool.
func NewCreateCustomToolTool() *CreateCustomToolTool {
	return &CreateCustomToolTool{}
}

// NewCreateCustomToolToolWithDir creates a new CreateCustomToolTool with a custom tools directory (for testing).
func NewCreateCustomToolToolWithDir(toolsDir string) *CreateCustomToolTool {
	return &CreateCustomToolTool{
		toolsDir: toolsDir,
	}
}

// Name returns the tool name.
func (t *CreateCustomToolTool) Name() string {
	return "create_custom_tool"
}

// Description returns the tool description.
func (t *CreateCustomToolTool) Description() string {
	return "Create a new custom tool with Go boilerplate code in ~/.forge/tools/. Generates tool.yaml metadata and a Go source file template that the agent can then edit and compile."
}

// Schema returns the JSON schema for the tool's input parameters.
func (t *CreateCustomToolTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Tool name (will be the directory and binary name)",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Tool description (what the tool does)",
			},
			"version": map[string]interface{}{
				"type":        "string",
				"description": "Tool version (optional, defaults to 1.0.0)",
			},
		},
		[]string{"name", "description"},
	)
}

// Execute creates a new custom tool scaffold.
func (t *CreateCustomToolTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var input struct {
		XMLName     xml.Name `xml:"arguments"`
		Name        string   `xml:"name"`
		Description string   `xml:"description"`
		Version     string   `xml:"version"`
	}

	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.Name == "" {
		return "", nil, fmt.Errorf("missing required parameter: name")
	}

	if input.Description == "" {
		return "", nil, fmt.Errorf("missing required parameter: description")
	}

	// Use default version if not provided
	if input.Version == "" {
		input.Version = "1.0.0"
	}

	// Create scaffold options
	opts := ScaffoldOptions{
		Name:        input.Name,
		Description: input.Description,
		Version:     input.Version,
		ToolsDir:    t.toolsDir, // Use override if set (for testing)
	}

	// Execute scaffold
	if err := Scaffold(opts); err != nil {
		return "", nil, fmt.Errorf("failed to scaffold tool: %w", err)
	}

	// Get tool directory for metadata
	toolDir, err := GetToolDir(input.Name)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get tool directory: %w", err)
	}

	message := buildToolCreationGuide(input.Name, toolDir)

	metadata := map[string]interface{}{
		"tool_name":   input.Name,
		"tool_dir":    toolDir,
		"version":     input.Version,
		"source_file": input.Name + ".go",
	}

	return message, metadata, nil
}

// IsLoopBreaking returns false as this tool doesn't break the agent loop.
func (t *CreateCustomToolTool) IsLoopBreaking() bool {
	return false
}

// buildToolCreationGuide returns the complete workflow guide for creating a custom tool.
func buildToolCreationGuide(toolName, toolDir string) string {
	return fmt.Sprintf(`Custom tool '%s' created successfully at %s

## Tool Creation Workflow

### 1. Implement the Tool Logic
Edit %s/%s.go to add your implementation:
- Add required imports to the Go source
- Define command-line flags using the flag package
- Read environment variables as needed with os.Getenv()
- Implement core logic in main()
- Write results to stdout, errors to stderr
- Use appropriate exit codes (0 for success, non-zero for errors)

### 2. Update Tool Metadata
Edit %s/tool.yaml to match your implementation:
- Update parameters to reflect actual CLI flags or inputs
- Add descriptions for each parameter
- Document any security considerations

### 3. Compile the Tool
Run: cd %s && go build -o %s %s.go

If compilation fails:
- Review error messages
- Fix code issues
- Recompile

### 4. Update Entrypoint
After successful compilation, update tool.yaml:
- Change entrypoint from "%s.go" to "%s" (the compiled binary)

### 5. Verify Auto-Discovery
The tool is automatically discovered and immediately available after compilation.
Test it by calling it directly.

## Input Handling Patterns

**Simple inputs:** Use CLI flags
  flag.String("input", "", "description")
  flag.Parse()

**Complex inputs:** Write JSON/YAML file, pass path as flag
  data := map[string]interface{}{"key": "value"}
  json.Marshal(data)
  ioutil.WriteFile("/tmp/input.json", data, 0644)

**Context:** Use environment variables
  apiKey := os.Getenv("MY_API_KEY")

## Best Practices

- Validate inputs to prevent injection attacks
- Use clear, descriptive parameter names
- Provide helpful error messages
- Document security considerations in tool.yaml
- Keep tools focused on a single responsibility`,
		toolName, toolDir,
		toolDir, toolName,
		toolDir,
		toolDir, toolName, toolName,
		toolName, toolName)
}
