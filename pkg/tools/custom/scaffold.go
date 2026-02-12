package custom

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"text/template"
)

// ScaffoldOptions contains options for scaffolding a new custom tool
type ScaffoldOptions struct {
	Name        string // Tool name (will be directory and binary name)
	Description string // Tool description
	Version     string // Tool version (default: "1.0.0")
	ToolsDir    string // Optional: Custom tools directory (for testing)
}

// Scaffold creates a new custom tool directory structure with boilerplate code
func Scaffold(opts ScaffoldOptions) error {
	// Validate options
	if opts.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	// Validate tool name to prevent path traversal
	if err := validateToolName(opts.Name); err != nil {
		return err
	}

	if opts.Description == "" {
		return fmt.Errorf("tool description cannot be empty")
	}
	if opts.Version == "" {
		opts.Version = "1.0.0"
	}

	// Get tool directory
	var toolDir string
	if opts.ToolsDir != "" {
		toolDir = filepath.Join(opts.ToolsDir, opts.Name)
	} else {
		var err error
		toolDir, err = GetToolDir(opts.Name)
		if err != nil {
			return fmt.Errorf("failed to get tool directory: %w", err)
		}
	}

	// Check if tool already exists
	if _, err := os.Stat(toolDir); err == nil {
		return fmt.Errorf("tool %s already exists at %s", opts.Name, toolDir)
	}

	// Create tool directory
	if err := os.MkdirAll(toolDir, 0750); err != nil {
		return fmt.Errorf("failed to create tool directory: %w", err)
	}

	// Create tool.yaml metadata
	metadata := &ToolMetadata{
		Name:        opts.Name,
		Description: opts.Description,
		Version:     opts.Version,
		Entrypoint:  opts.Name, // Binary name; Go source is opts.Name + ".go"
		Usage:       fmt.Sprintf("Usage instructions for %s", opts.Name),
		Parameters:  []Parameter{}, // Empty initially, agent will add parameters
	}

	metadataPath := filepath.Join(toolDir, "tool.yaml")
	if err := SaveMetadata(metadataPath, metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}

	// Create Go source file with boilerplate
	goFilePath := filepath.Join(toolDir, opts.Name+".go")
	if err := writeGoBoilerplate(goFilePath, opts.Name); err != nil {
		return fmt.Errorf("failed to write Go boilerplate: %w", err)
	}

	return nil
}

// validateToolName ensures the tool name is safe and cannot be used for path traversal
func validateToolName(name string) error {
	// Check for empty or just whitespace
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	// Reject . and .. explicitly
	if name == "." || name == ".." {
		return fmt.Errorf("tool name cannot be '.' or '..'")
	}

	// Ensure the name doesn't contain path separators
	if filepath.Base(name) != name {
		return fmt.Errorf("tool name cannot contain path separators")
	}

	// Enforce alphanumeric, underscore, and hyphen only
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("tool name must contain only alphanumeric characters, underscores, and hyphens")
	}

	return nil
}

// writeGoBoilerplate writes the Go boilerplate template to a file
func writeGoBoilerplate(path, toolName string) error {
	tmpl, err := template.New("boilerplate").Parse(goBoilerplateTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	data := struct {
		ToolName string
	}{
		ToolName: toolName,
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// goBoilerplateTemplate is the template for new custom tool Go source files
const goBoilerplateTemplate = `package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define flags for your tool parameters
	// TODO: Add your parameters here
	// Example:
	// exampleParam := flag.String("example", "", "Example parameter")
	// requiredParam := flag.String("required", "", "Required parameter")
	
	flag.Parse()

	// Access environment variables if needed (optional - remove if not used)
	// Example:
	// apiKey := os.Getenv("MY_API_KEY")
	// if apiKey == "" {
	// 	writeError(fmt.Errorf("MY_API_KEY environment variable not set"))
	// }

	// TODO: Implement your tool logic here
	// Use flag values to access parameters
	// Example: if *requiredParam == "" { ... }
	
	// Output results to stdout
	// The agent will see this output
	result := "TODO: Implement tool logic"
	writeOutput(result)
}

// writeOutput writes the tool result to stdout
// This is what the agent will see as the tool execution result
func writeOutput(result string) {
	fmt.Println(result)
}

// writeError writes an error message to stderr and exits with code 1
func writeError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}
`
