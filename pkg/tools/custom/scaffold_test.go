package custom

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScaffold(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	opts := ScaffoldOptions{
		Name:        "test-tool",
		Description: "A test tool",
		Version:     "1.0.0",
		ToolsDir:    tmpDir,
	}

	// Scaffold the tool
	if err := Scaffold(opts); err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	// Verify directory structure
	toolDir := filepath.Join(tmpDir, "test-tool")
	if _, err := os.Stat(toolDir); os.IsNotExist(err) {
		t.Error("Tool directory was not created")
	}

	// Verify tool.yaml exists
	metadataPath := filepath.Join(toolDir, "tool.yaml")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Error("tool.yaml was not created")
	}

	// Verify Go source file exists
	goFilePath := filepath.Join(toolDir, "test-tool.go")
	if _, err := os.Stat(goFilePath); os.IsNotExist(err) {
		t.Error("Go source file was not created")
	}

	// Load and validate metadata
	metadata, err := LoadMetadata(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if metadata.Name != opts.Name {
		t.Errorf("Name = %v, want %v", metadata.Name, opts.Name)
	}
	if metadata.Description != opts.Description {
		t.Errorf("Description = %v, want %v", metadata.Description, opts.Description)
	}
	if metadata.Version != opts.Version {
		t.Errorf("Version = %v, want %v", metadata.Version, opts.Version)
	}
	if metadata.Entrypoint != "test-tool" {
		t.Errorf("Entrypoint = %v, want test-tool", metadata.Entrypoint)
	}

	// Verify Go file contains expected content
	goContent, err := os.ReadFile(goFilePath)
	if err != nil {
		t.Fatalf("Failed to read Go file: %v", err)
	}

	expectedStrings := []string{
		"package main",
		"import (",
		"flag.Parse()",
		"func main()",
		"func writeOutput",
		"func writeError",
	}

	for _, expected := range expectedStrings {
		if !containsString(string(goContent), expected) {
			t.Errorf("Go file missing expected content: %s", expected)
		}
	}
}

func TestScaffold_MissingName(t *testing.T) {
	opts := ScaffoldOptions{
		Description: "A test tool",
	}

	err := Scaffold(opts)
	if err == nil {
		t.Error("Scaffold() expected error for missing name")
	}
}

func TestScaffold_MissingDescription(t *testing.T) {
	opts := ScaffoldOptions{
		Name: "test-tool",
	}

	err := Scaffold(opts)
	if err == nil {
		t.Error("Scaffold() expected error for missing description")
	}
}

func TestScaffold_DefaultVersion(t *testing.T) {
	tmpDir := t.TempDir()

	opts := ScaffoldOptions{
		Name:        "test-tool",
		Description: "A test tool",
		ToolsDir:    tmpDir,
		// Version not specified
	}

	if err := Scaffold(opts); err != nil {
		t.Fatalf("Scaffold() error = %v", err)
	}

	metadataPath := filepath.Join(tmpDir, "test-tool", "tool.yaml")
	metadata, err := LoadMetadata(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if metadata.Version != "1.0.0" {
		t.Errorf("Version = %v, want 1.0.0 (default)", metadata.Version)
	}
}

func TestScaffold_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()

	opts := ScaffoldOptions{
		Name:        "test-tool",
		Description: "A test tool",
		ToolsDir:    tmpDir,
	}

	// First scaffold should succeed
	if err := Scaffold(opts); err != nil {
		t.Fatalf("First Scaffold() error = %v", err)
	}

	// Second scaffold should fail
	err := Scaffold(opts)
	if err == nil {
		t.Error("Scaffold() expected error for existing tool")
	}
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
