package custom

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Registry manages the discovery and loading of custom tools
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*ToolMetadata // tool name -> metadata
}

// NewRegistry creates a new custom tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*ToolMetadata),
	}
}

// Refresh scans the tools directory and reloads all tool metadata
// This should be called at the start of each agent turn
func (r *Registry) Refresh() error {
	toolsDir, err := GetToolsDir()
	if err != nil {
		return fmt.Errorf("failed to get tools directory: %w", err)
	}

	// Create tools directory if it doesn't exist
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tools directory: %w", err)
	}

	// Read all subdirectories
	entries, err := os.ReadDir(toolsDir)
	if err != nil {
		return fmt.Errorf("failed to read tools directory: %w", err)
	}

	// Load metadata for each tool directory
	newTools := make(map[string]*ToolMetadata)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		toolName := entry.Name()
		metadataPath := filepath.Join(toolsDir, toolName, "tool.yaml")

		// Check if tool.yaml exists
		if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
			continue
		}

		// Load and validate metadata
		metadata, err := LoadMetadata(metadataPath)
		if err != nil {
			// Log error but continue with other tools
			continue
		}

		// Verify binary exists and is executable
		binaryPath := filepath.Join(toolsDir, toolName, metadata.Entrypoint)
		info, err := os.Stat(binaryPath)
		if err != nil {
			// Binary doesn't exist, skip this tool
			continue
		}

		// Check if file is executable (on Unix-like systems)
		if info.Mode()&0111 == 0 {
			// Not executable, skip this tool
			continue
		}

		newTools[toolName] = metadata
	}

	// Update registry atomically
	r.mu.Lock()
	r.tools = newTools
	r.mu.Unlock()

	return nil
}

// Get retrieves metadata for a specific tool
func (r *Registry) Get(toolName string) (*ToolMetadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	metadata, ok := r.tools[toolName]
	return metadata, ok
}

// List returns all available custom tools
func (r *Registry) List() []*ToolMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tools := make([]*ToolMetadata, 0, len(r.tools))
	for _, metadata := range r.tools {
		tools = append(tools, metadata)
	}
	return tools
}

// Count returns the number of registered custom tools
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// Has checks if a tool is registered
func (r *Registry) Has(toolName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.tools[toolName]
	return ok
}

// GetBinaryPath returns the absolute path to a tool's binary
func (r *Registry) GetBinaryPath(toolName string) (string, error) {
	metadata, ok := r.Get(toolName)
	if !ok {
		return "", fmt.Errorf("tool %s not found", toolName)
	}

	toolsDir, err := GetToolsDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(toolsDir, toolName, metadata.Entrypoint), nil
}
