package custom

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ToolMetadata represents the YAML metadata for a custom tool
type ToolMetadata struct {
	Name        string      `yaml:"name"`        // Tool identifier (matches directory name)
	Description string      `yaml:"description"` // What the tool does
	Version     string      `yaml:"version"`     // Semantic version (e.g., "1.0.0")
	Entrypoint  string      `yaml:"entrypoint"`  // Compiled binary name (not .go file)
	Usage       string      `yaml:"usage"`       // Multi-line usage instructions for agent
	Parameters  []Parameter `yaml:"parameters"`  // List of parameters
}

// Parameter represents a tool parameter definition
type Parameter struct {
	Name        string `yaml:"name"`        // Parameter identifier
	Type        string `yaml:"type"`        // string | number | boolean
	Required    bool   `yaml:"required"`    // Is this parameter required?
	Description string `yaml:"description"` // What this parameter does
}

// Validate checks if the metadata is valid
func (m *ToolMetadata) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if m.Description == "" {
		return fmt.Errorf("tool description cannot be empty")
	}
	if m.Version == "" {
		return fmt.Errorf("tool version cannot be empty")
	}
	if m.Entrypoint == "" {
		return fmt.Errorf("tool entrypoint cannot be empty")
	}
	
	// Validate parameter types
	for i, param := range m.Parameters {
		if param.Name == "" {
			return fmt.Errorf("parameter %d: name cannot be empty", i)
		}
		if param.Type != "string" && param.Type != "number" && param.Type != "boolean" {
			return fmt.Errorf("parameter %s: type must be string, number, or boolean", param.Name)
		}
		if param.Description == "" {
			return fmt.Errorf("parameter %s: description cannot be empty", param.Name)
		}
	}
	
	return nil
}

// LoadMetadata reads and parses a tool.yaml file
func LoadMetadata(path string) (*ToolMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}
	
	var metadata ToolMetadata
	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	if err := metadata.Validate(); err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}
	
	return &metadata, nil
}

// SaveMetadata writes metadata to a tool.yaml file
func SaveMetadata(path string, metadata *ToolMetadata) error {
	if err := metadata.Validate(); err != nil {
		return fmt.Errorf("invalid metadata: %w", err)
	}
	
	data, err := yaml.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}
	
	return nil
}

// GetToolsDir returns the absolute path to the custom tools directory
func GetToolsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".forge", "tools"), nil
}

// GetToolDir returns the absolute path to a specific tool's directory
func GetToolDir(toolName string) (string, error) {
	toolsDir, err := GetToolsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(toolsDir, toolName), nil
}

// GetToolMetadataPath returns the path to a tool's metadata file
func GetToolMetadataPath(toolName string) (string, error) {
	toolDir, err := GetToolDir(toolName)
	if err != nil {
		return "", err
	}
	return filepath.Join(toolDir, "tool.yaml"), nil
}
