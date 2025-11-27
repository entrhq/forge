package coding

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileTool_CreateNewFile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewWriteFileTool(guard)

	xmlInput := `<arguments>
	<path>new.txt</path>
	<content>Hello, World!</content>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify result message
	if !strings.Contains(result, "created successfully") {
		t.Errorf("Expected 'created successfully' in result, got: %s", result)
	}
	if !strings.Contains(result, "+1 lines") {
		t.Errorf("Expected '+1 lines' in result, got: %s", result)
	}

	// Verify file was created
	filePath := filepath.Join(tmpDir, "new.txt")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}
	if string(content) != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got: %s", string(content))
	}

	// Verify metadata
	if metadata["file_exists"] != false {
		t.Errorf("Expected file_exists=false for new file")
	}
	if metadata["lines_added"] != 1 {
		t.Errorf("Expected lines_added=1, got %v", metadata["lines_added"])
	}
	if metadata["lines_removed"] != 0 {
		t.Errorf("Expected lines_removed=0, got %v", metadata["lines_removed"])
	}
}

func TestWriteFileTool_OverwriteExistingFile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create initial file
	testFile := filepath.Join(tmpDir, "existing.txt")
	writeTestFile(t, testFile, "line 1\nline 2\nline 3")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewWriteFileTool(guard)

	xmlInput := `<arguments>
	<path>existing.txt</path>
	<content>new line 1
new line 2</content>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify result message
	if !strings.Contains(result, "overwritten successfully") {
		t.Errorf("Expected 'overwritten successfully' in result, got: %s", result)
	}

	// Verify file content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read overwritten file: %v", err)
	}
	expected := "new line 1\nnew line 2"
	if string(content) != expected {
		t.Errorf("Expected '%s', got: %s", expected, string(content))
	}

	// Verify metadata
	if metadata["file_exists"] != true {
		t.Errorf("Expected file_exists=true for existing file")
	}
	if metadata["lines_added"].(int) < 1 {
		t.Errorf("Expected lines_added > 0, got %v", metadata["lines_added"])
	}
}

func TestWriteFileTool_CreateNestedDirectories(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewWriteFileTool(guard)

	xmlInput := `<arguments>
	<path>deep/nested/path/file.txt</path>
	<content>nested content</content>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify file was created with nested directories
	filePath := filepath.Join(tmpDir, "deep", "nested", "path", "file.txt")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read nested file: %v", err)
	}
	if string(content) != "nested content" {
		t.Errorf("Expected 'nested content', got: %s", string(content))
	}

	// Verify directories were created
	dirPath := filepath.Join(tmpDir, "deep", "nested", "path")
	info, err := os.Stat(dirPath)
	if err != nil {
		t.Fatalf("Nested directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Expected path to be a directory")
	}
}

func TestWriteFileTool_InvalidPath(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewWriteFileTool(guard)

	xmlInput := `<arguments>
	<path>../outside.txt</path>
	<content>should not be written</content>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for path outside workspace")
	}
	if !strings.Contains(err.Error(), "invalid path") {
		t.Errorf("Expected 'invalid path' error, got: %v", err)
	}
}

func TestWriteFileTool_MissingPath(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewWriteFileTool(guard)

	xmlInput := `<arguments>
	<content>content without path</content>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for missing path")
	}
	if !strings.Contains(err.Error(), "missing required parameter: path") {
		t.Errorf("Expected 'missing required parameter' error, got: %v", err)
	}
}

func TestWriteFileTool_EmptyContent(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewWriteFileTool(guard)

	xmlInput := `<arguments>
	<path>empty.txt</path>
	<content></content>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify file was created
	filePath := filepath.Join(tmpDir, "empty.txt")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read empty file: %v", err)
	}
	if len(content) != 0 {
		t.Errorf("Expected empty file, got: %s", string(content))
	}

	// Verify result message
	if !strings.Contains(result, "created successfully") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify metadata
	if metadata["size_bytes"].(int64) != 0 {
		t.Errorf("Expected size_bytes=0, got %v", metadata["size_bytes"])
	}
}

func TestWriteFileTool_LargeFile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewWriteFileTool(guard)

	// Create large content (10,000 lines)
	var builder strings.Builder
	for i := 1; i <= 10000; i++ {
		builder.WriteString(fmt.Sprintf("line %d\n", i))
	}
	largeContent := builder.String()

	xmlInput := fmt.Sprintf(`<arguments>
	<path>large.txt</path>
	<content>%s</content>
</arguments>`, largeContent)

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify file was created
	filePath := filepath.Join(tmpDir, "large.txt")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read large file: %v", err)
	}
	if string(content) != largeContent {
		t.Error("Large file content mismatch")
	}

	// Verify metadata
	if metadata["lines_added"] != 10000 {
		t.Errorf("Expected lines_added=10000, got %v", metadata["lines_added"])
	}

	// Verify result message
	if !strings.Contains(result, "+10000 lines") {
		t.Errorf("Expected '+10000 lines' in result, got: %s", result)
	}
}

func TestWriteFileTool_SpecialCharacters(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewWriteFileTool(guard)

	// Note: XML entities are already decoded by the XML parser
	specialContent := "Special chars: <>&\"'\nUnicode: 你好 🚀"
	xmlInput := fmt.Sprintf(`<arguments>
	<path>special.txt</path>
	<content><![CDATA[%s]]></content>
</arguments>`, specialContent)

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify file content
	filePath := filepath.Join(tmpDir, "special.txt")
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read special chars file: %v", err)
	}
	if string(content) != specialContent {
		t.Errorf("Expected '%s', got: %s", specialContent, string(content))
	}
}

func TestWriteFileTool_AtomicWrite(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create initial file
	testFile := filepath.Join(tmpDir, "atomic.txt")
	writeTestFile(t, testFile, "original content")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewWriteFileTool(guard)

	xmlInput := `<arguments>
	<path>atomic.txt</path>
	<content>new content</content>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify no temporary files left behind
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Errorf("Found temporary file after write: %s", entry.Name())
		}
	}

	// Verify final content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != "new content" {
		t.Errorf("Expected 'new content', got: %s", string(content))
	}
}

func TestWriteFileTool_Metadata(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewWriteFileTool(guard)

	// Verify name and description
	if tool.Name() != "write_file" {
		t.Errorf("Expected name 'write_file', got '%s'", tool.Name())
	}

	desc := tool.Description()
	if !strings.Contains(desc, "Write content") {
		t.Errorf("Expected description to mention writing content, got: %s", desc)
	}

	// Verify schema
	schema := tool.Schema()
	if schema == nil {
		t.Fatal("Expected non-nil schema")
	}

	// Verify loop breaking status
	if tool.IsLoopBreaking() {
		t.Error("WriteFileTool should not be loop-breaking")
	}
}

func TestWriteFileTool_LineChanges(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create initial file with 3 lines
	testFile := filepath.Join(tmpDir, "changes.txt")
	writeTestFile(t, testFile, "line 1\nline 2\nline 3")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewWriteFileTool(guard)

	tests := []struct {
		name            string
		content         string
		expectedAdded   int
		expectedRemoved int
	}{
		{
			name:            "add lines",
			content:         "line 1\nline 2\nline 3\nline 4\nline 5",
			expectedAdded:   5, // Total lines in new content
			expectedRemoved: 3, // Total lines in old content
		},
		{
			name:            "remove lines",
			content:         "line 1",
			expectedAdded:   1, // Total lines in new content
			expectedRemoved: 3, // Total lines in old content
		},
		{
			name:            "mixed changes",
			content:         "new line 1\nnew line 2\nline 3\nline 4",
			expectedAdded:   4, // Total lines in new content
			expectedRemoved: 3, // Total lines in old content
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset file to original state
			writeTestFile(t, testFile, "line 1\nline 2\nline 3")

			xmlInput := fmt.Sprintf(`<arguments>
	<path>changes.txt</path>
	<content>%s</content>
</arguments>`, tt.content)

			result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			// Verify metadata
			if metadata["lines_added"] != tt.expectedAdded {
				t.Errorf("Expected lines_added=%d, got %v", tt.expectedAdded, metadata["lines_added"])
			}
			if metadata["lines_removed"] != tt.expectedRemoved {
				t.Errorf("Expected lines_removed=%d, got %v", tt.expectedRemoved, metadata["lines_removed"])
			}

			// Verify result message includes line changes
			if !strings.Contains(result, fmt.Sprintf("+%d", tt.expectedAdded)) {
				t.Errorf("Expected result to contain '+%d', got: %s", tt.expectedAdded, result)
			}
			if !strings.Contains(result, fmt.Sprintf("-%d", tt.expectedRemoved)) {
				t.Errorf("Expected result to contain '-%d', got: %s", tt.expectedRemoved, result)
			}
		})
	}
}
