package coding

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/entrhq/forge/pkg/security/workspace"
)

func TestReadFileTool_BasicRead(t *testing.T) {
	// Create temp directory and test file
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5"
	writeTestFile(t, testFile, content)

	// Create tool
	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewReadFileTool(guard)

	// Test reading entire file
	xmlInput := `<arguments>
	<path>test.txt</path>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify result contains line numbers
	expectedLines := []string{
		"1 | line 1",
		"2 | line 2",
		"3 | line 3",
		"4 | line 4",
		"5 | line 5",
	}
	for _, expected := range expectedLines {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected result to contain '%s', got:\n%s", expected, result)
		}
	}

	// Verify metadata
	if metadata["path"] != "test.txt" {
		t.Errorf("Expected path 'test.txt', got '%v'", metadata["path"])
	}
	if _, ok := metadata["size_bytes"]; !ok {
		t.Error("Expected size_bytes in metadata")
	}
	if _, ok := metadata["modified"]; !ok {
		t.Error("Expected modified timestamp in metadata")
	}
}

func TestReadFileTool_LineRange(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "range.txt")
	content := "line 1\nline 2\nline 3\nline 4\nline 5\nline 6\nline 7"
	writeTestFile(t, testFile, content)

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewReadFileTool(guard)

	tests := []struct {
		name          string
		startLine     int
		endLine       int
		expectedLines []string
	}{
		{
			name:          "read lines 2-4",
			startLine:     2,
			endLine:       4,
			expectedLines: []string{"2 | line 2", "3 | line 3", "4 | line 4"},
		},
		{
			name:          "read from line 5 to end",
			startLine:     5,
			endLine:       0,
			expectedLines: []string{"5 | line 5", "6 | line 6", "7 | line 7"},
		},
		{
			name:          "read single line",
			startLine:     3,
			endLine:       3,
			expectedLines: []string{"3 | line 3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xmlInput := generateReadFileXML("range.txt", tt.startLine, tt.endLine)
			result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			// Verify expected lines are present
			for _, expected := range tt.expectedLines {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain '%s', got:\n%s", expected, result)
				}
			}

			// Verify metadata includes line range
			if tt.startLine > 0 {
				if metadata["start_line"] != tt.startLine {
					t.Errorf("Expected start_line %d, got %v", tt.startLine, metadata["start_line"])
				}
			}
			if tt.endLine > 0 {
				if metadata["end_line"] != tt.endLine {
					t.Errorf("Expected end_line %d, got %v", tt.endLine, metadata["end_line"])
				}
			}
		})
	}
}

func TestReadFileTool_EmptyFile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "empty.txt")
	writeTestFile(t, testFile, "")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewReadFileTool(guard)

	xmlInput := `<arguments>
	<path>empty.txt</path>
</arguments>`

	result, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result != "" {
		t.Errorf("Expected empty result for empty file, got: %s", result)
	}
}

func TestReadFileTool_InvalidPath(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewReadFileTool(guard)

	xmlInput := `<arguments>
	<path>../outside.txt</path>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for path outside workspace")
	}
	if !strings.Contains(err.Error(), "invalid path") {
		t.Errorf("Expected 'invalid path' error, got: %v", err)
	}
}

func TestReadFileTool_NonExistentFile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewReadFileTool(guard)

	xmlInput := `<arguments>
	<path>does-not-exist.txt</path>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestReadFileTool_InvalidLineRange(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.txt")
	writeTestFile(t, testFile, "line 1\nline 2\nline 3")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewReadFileTool(guard)

	tests := []struct {
		name      string
		startLine int
		endLine   int
		wantError string
	}{
		{
			name:      "start line is 0 with end line",
			startLine: 0,
			endLine:   5,
			wantError: "start_line must be >= 1",
		},
		{
			name:      "end line before start line",
			startLine: 5,
			endLine:   2,
			wantError: "end_line (2) must be >= start_line (5)",
		},
		{
			name:      "start line exceeds file length",
			startLine: 100,
			endLine:   0,
			wantError: "start_line 100 exceeds file length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xmlInput := generateReadFileXML("test.txt", tt.startLine, tt.endLine)
			_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
			if err == nil {
				t.Error("Expected error for invalid line range")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Expected error containing '%s', got: %v", tt.wantError, err)
			}
		})
	}
}

func TestReadFileTool_MissingPath(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewReadFileTool(guard)

	xmlInput := `<arguments>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for missing path")
	}
	if !strings.Contains(err.Error(), "missing required parameter: path") {
		t.Errorf("Expected 'missing required parameter' error, got: %v", err)
	}
}

func TestReadFileTool_IgnoredFile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create .gitignore
	gitignore := filepath.Join(tmpDir, ".gitignore")
	writeTestFile(t, gitignore, "ignored.txt")

	// Create ignored file
	ignoredFile := filepath.Join(tmpDir, "ignored.txt")
	writeTestFile(t, ignoredFile, "should not be readable")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewReadFileTool(guard)

	xmlInput := `<arguments>
	<path>ignored.txt</path>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for ignored file")
	}
	if !strings.Contains(err.Error(), "is ignored") {
		t.Errorf("Expected 'is ignored' error, got: %v", err)
	}
}

func TestReadFileTool_Metadata(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.txt")
	writeTestFile(t, testFile, "test content")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewReadFileTool(guard)

	// Test basic read
	result, metadata, err := tool.Execute(context.Background(), []byte(`<arguments><path>test.txt</path></arguments>`))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify name and description
	if tool.Name() != "read_file" {
		t.Errorf("Expected name 'read_file', got '%s'", tool.Name())
	}

	desc := tool.Description()
	if !strings.Contains(desc, "Read the contents") {
		t.Errorf("Expected description to mention reading contents, got: %s", desc)
	}

	// Verify schema
	schema := tool.Schema()
	if schema == nil {
		t.Fatal("Expected non-nil schema")
	}

	// Verify loop breaking status
	if tool.IsLoopBreaking() {
		t.Error("ReadFileTool should not be loop-breaking")
	}

	// Verify basic metadata fields
	if metadata["path"] != "test.txt" {
		t.Errorf("Expected path in metadata")
	}

	// Verify result is not empty
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

// Helper functions

func setupTestDir(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "read_file_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return tmpDir, func() { os.RemoveAll(tmpDir) }
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
}

func createWorkspaceGuard(t *testing.T, dir string) *workspace.Guard {
	t.Helper()
	guard, err := workspace.NewGuard(dir)
	if err != nil {
		t.Fatalf("Failed to create workspace guard: %v", err)
	}
	return guard
}

func generateReadFileXML(path string, startLine, endLine int) string {
	xml := "<arguments>\n\t<path>" + path + "</path>\n"
	if startLine > 0 {
		xml += fmt.Sprintf("\t<start_line>%d</start_line>\n", startLine)
	}
	if endLine > 0 {
		xml += fmt.Sprintf("\t<end_line>%d</end_line>\n", endLine)
	}
	xml += "</arguments>"
	return xml
}
