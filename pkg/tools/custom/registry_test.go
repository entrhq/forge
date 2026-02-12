package custom

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistry_RefreshAndGet(t *testing.T) {
	// Create temporary tools directory
	tmpDir := t.TempDir()

	// Create a test tool
	toolDir := filepath.Join(tmpDir, "test-tool")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatalf("Failed to create tool directory: %v", err)
	}

	// Write metadata
	metadata := &ToolMetadata{
		Name:        "test-tool",
		Description: "A test tool",
		Version:     "1.0.0",
		Entrypoint:  "test-tool",
		Usage:       "Usage instructions",
		Parameters: []Parameter{
			{
				Name:        "input",
				Type:        "string",
				Required:    true,
				Description: "Input parameter",
			},
		},
	}
	metadataPath := filepath.Join(toolDir, "tool.yaml")
	if err := SaveMetadata(metadataPath, metadata); err != nil {
		t.Fatalf("Failed to save metadata: %v", err)
	}

	// Create executable binary (just a dummy file with exec permissions)
	binaryPath := filepath.Join(toolDir, "test-tool")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create binary: %v", err)
	}

	// Test registry with custom directory
	registry := NewRegistryWithDir(tmpDir)

	// Initially empty
	if registry.Count() != 0 {
		t.Errorf("Count() = %v, want 0", registry.Count())
	}

	// Refresh should load the tool
	if err := registry.Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	// Should now have one tool
	if registry.Count() != 1 {
		t.Errorf("Count() = %v, want 1", registry.Count())
	}

	// Get the tool
	loaded, ok := registry.Get("test-tool")
	if !ok {
		t.Fatal("Get() returned false, want true")
	}

	if loaded.Name != metadata.Name {
		t.Errorf("Name = %v, want %v", loaded.Name, metadata.Name)
	}
	if loaded.Description != metadata.Description {
		t.Errorf("Description = %v, want %v", loaded.Description, metadata.Description)
	}
}

func TestRegistry_List(t *testing.T) {
	// Create temporary tools directory
	tmpDir := t.TempDir()

	// Create multiple test tools
	for i := 1; i <= 3; i++ {
		toolName := fmt.Sprintf("tool-%d", i)
		toolDir := filepath.Join(tmpDir, toolName)
		if err := os.MkdirAll(toolDir, 0755); err != nil {
			t.Fatalf("Failed to create tool directory: %v", err)
		}

		metadata := &ToolMetadata{
			Name:        toolName,
			Description: fmt.Sprintf("Tool %d", i),
			Version:     "1.0.0",
			Entrypoint:  toolName,
		}
		
		metadataPath := filepath.Join(toolDir, "tool.yaml")
		if err := SaveMetadata(metadataPath, metadata); err != nil {
			t.Fatalf("Failed to save metadata: %v", err)
		}

		binaryPath := filepath.Join(toolDir, toolName)
		if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
			t.Fatalf("Failed to create binary: %v", err)
		}
	}

	registry := NewRegistryWithDir(tmpDir)
	if err := registry.Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	tools := registry.List()
	if len(tools) != 3 {
		t.Errorf("List() returned %d tools, want 3", len(tools))
	}
}

func TestRegistry_Has(t *testing.T) {
	registry := NewRegistry()

	if registry.Has("nonexistent") {
		t.Error("Has() returned true for nonexistent tool")
	}

	// Add a tool manually for testing
	registry.tools["test-tool"] = &ToolMetadata{
		Name: "test-tool",
	}

	if !registry.Has("test-tool") {
		t.Error("Has() returned false for existing tool")
	}
}

func TestRegistry_GetBinaryPath(t *testing.T) {
	// Create temporary tools directory
	tmpDir := t.TempDir()

	registry := NewRegistryWithDir(tmpDir)

	// Add a tool manually
	registry.tools["test-tool"] = &ToolMetadata{
		Name:       "test-tool",
		Entrypoint: "test-binary",
	}

	path, err := registry.GetBinaryPath("test-tool")
	if err != nil {
		t.Fatalf("GetBinaryPath() error = %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "test-tool", "test-binary")
	if path != expectedPath {
		t.Errorf("GetBinaryPath() = %v, want %v", path, expectedPath)
	}

	// Test nonexistent tool
	_, err = registry.GetBinaryPath("nonexistent")
	if err == nil {
		t.Error("GetBinaryPath() expected error for nonexistent tool")
	}
}

func TestRegistry_RefreshSkipsInvalidTools(t *testing.T) {
	tmpDir := t.TempDir()

	// Create tool with missing binary
	toolDir := filepath.Join(tmpDir, "no-binary")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatalf("Failed to create tool directory: %v", err)
	}

	metadata := &ToolMetadata{
		Name:        "no-binary",
		Description: "Tool without binary",
		Version:     "1.0.0",
		Entrypoint:  "missing",
	}
	metadataPath := filepath.Join(toolDir, "tool.yaml")
	if err := SaveMetadata(metadataPath, metadata); err != nil {
		t.Fatalf("Failed to save metadata: %v", err)
	}

	registry := NewRegistry()
	if err := registry.Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	// Should not load tool without binary
	if registry.Has("no-binary") {
		t.Error("Registry loaded tool without binary")
	}
}
