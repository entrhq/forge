package coding

import (
	"context"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/entrhq/forge/pkg/security/workspace"
)

func TestApplyDiffXMLUnmarshal(t *testing.T) {
	// Test that XML with multiple edits unmarshals correctly
	xmlInput := `<arguments>
	<path>test.go</path>
	<edits>
		<edit>
			<search>old code 1</search>
			<replace>new code 1</replace>
		</edit>
		<edit>
			<search>old code 2</search>
			<replace>new code 2</replace>
		</edit>
	</edits>
</arguments>`

	var input struct {
		XMLName xml.Name `xml:"arguments"`
		Path    string   `xml:"path"`
		Edits   []struct {
			Search  string `xml:"search"`
			Replace string `xml:"replace"`
		} `xml:"edits>edit"`
	}

	err := xml.Unmarshal([]byte(xmlInput), &input)
	if err != nil {
		t.Fatalf("Failed to unmarshal XML: %v", err)
	}

	// Verify path
	if input.Path != "test.go" {
		t.Errorf("Expected path 'test.go', got '%s'", input.Path)
	}

	// Verify we got 2 edits
	if len(input.Edits) != 2 {
		t.Fatalf("Expected 2 edits, got %d", len(input.Edits))
	}

	// Verify first edit
	if input.Edits[0].Search != "old code 1" {
		t.Errorf("Edit 0: Expected search 'old code 1', got '%s'", input.Edits[0].Search)
	}
	if input.Edits[0].Replace != "new code 1" {
		t.Errorf("Edit 0: Expected replace 'new code 1', got '%s'", input.Edits[0].Replace)
	}

	// Verify second edit
	if input.Edits[1].Search != "old code 2" {
		t.Errorf("Edit 1: Expected search 'old code 2', got '%s'", input.Edits[1].Search)
	}
	if input.Edits[1].Replace != "new code 2" {
		t.Errorf("Edit 1: Expected replace 'new code 2', got '%s'", input.Edits[1].Replace)
	}
}

func TestApplyDiffToolWithXML(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "apply_diff_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	originalContent := `package main

func oldFunction() {
	return "old"
}

const oldValue = 42
`
	if writeErr := os.WriteFile(testFile, []byte(originalContent), 0644); writeErr != nil {
		t.Fatalf("Failed to create test file: %v", writeErr)
	}

	// Create workspace guard
	guard, err := workspace.NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create workspace guard: %v", err)
	}

	// Create tool
	tool := NewApplyDiffTool(guard)

	// Create XML input with multiple edits
	xmlInput := `<arguments>
	<path>test.go</path>
	<edits>
		<edit>
			<search>func oldFunction() {
	return "old"
}</search>
			<replace>func newFunction() {
	return "new"
}</replace>
		</edit>
		<edit>
			<search>const oldValue = 42</search>
			<replace>const newValue = 100</replace>
		</edit>
	</edits>
</arguments>`

	// Execute tool
	result, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	// Verify result message
	expectedMsg := "Successfully applied 2 edit(s) to test.go"
	if result != expectedMsg {
		t.Errorf("Expected result '%s', got '%s'", expectedMsg, result)
	}

	// Verify file content was updated
	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	expectedContent := `package main

func newFunction() {
	return "new"
}

const newValue = 100
`
	if string(updatedContent) != expectedContent {
		t.Errorf("File content mismatch.\nExpected:\n%s\nGot:\n%s", expectedContent, string(updatedContent))
	}
}
