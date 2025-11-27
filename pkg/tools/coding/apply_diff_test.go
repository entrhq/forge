package coding

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyDiffTool_SingleEdit(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.go")
	originalContent := "package main\n\nfunc old() {\n\treturn\n}"
	writeTestFile(t, testFile, originalContent)

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	xmlInput := `<arguments>
	<path>test.go</path>
	<edits>
		<edit>
			<search>func old() {</search>
			<replace>func newFunc() {</replace>
		</edit>
	</edits>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify result message
	if !strings.Contains(result, "Successfully applied 1 edit") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify metadata
	if metadata["edits_applied"].(int) != 1 {
		t.Errorf("Expected edits_applied=1, got %v", metadata["edits_applied"])
	}

	// Verify file was modified
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if !strings.Contains(string(content), "func newFunc()") {
		t.Errorf("Expected file to contain 'func newFunc()', got: %s", string(content))
	}
	if strings.Contains(string(content), "func old()") {
		t.Errorf("Expected file NOT to contain 'func old()', got: %s", string(content))
	}
}

func TestApplyDiffTool_MultipleEdits(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.go")
	originalContent := "const oldName = \"value\"\nvar oldVar = 42\nfunc oldFunc() {}"
	writeTestFile(t, testFile, originalContent)

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	xmlInput := `<arguments>
	<path>test.go</path>
	<edits>
		<edit>
			<search>const oldName</search>
			<replace>const newName</replace>
		</edit>
		<edit>
			<search>var oldVar</search>
			<replace>var newVar</replace>
		</edit>
		<edit>
			<search>func oldFunc</search>
			<replace>func newFunc</replace>
		</edit>
	</edits>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify metadata
	if metadata["edits_applied"].(int) != 3 {
		t.Errorf("Expected edits_applied=3, got %v", metadata["edits_applied"])
	}

	// Verify all replacements
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	expectedReplacements := []string{"const newName", "var newVar", "func newFunc"}
	for _, expected := range expectedReplacements {
		if !strings.Contains(string(content), expected) {
			t.Errorf("Expected file to contain '%s', got: %s", expected, string(content))
		}
	}

	// Verify result
	if !strings.Contains(result, "Successfully applied 3 edit") {
		t.Errorf("Expected success message with 3 edits, got: %s", result)
	}
}

func TestApplyDiffTool_LineChangesTracking(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.go")
	originalContent := "line1\nline2\nline3"
	writeTestFile(t, testFile, originalContent)

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	tests := []struct {
		name            string
		search          string
		replace         string
		expectedAdded   int
		expectedRemoved int
	}{
		{
			name:            "add lines",
			search:          "line2",
			replace:         "line2\nnewline1\nnewline2",
			expectedAdded:   2,
			expectedRemoved: 0,
		},
		{
			name:            "remove lines",
			search:          "line1\nline2\nline3",
			replace:         "singleline",
			expectedAdded:   0,
			expectedRemoved: 2,
		},
		{
			name:            "no line change",
			search:          "line2",
			replace:         "modified",
			expectedAdded:   0,
			expectedRemoved: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset file content
			writeTestFile(t, testFile, originalContent)

			xmlInput := `<arguments>
	<path>test.go</path>
	<edits>
		<edit>
			<search>` + tt.search + `</search>
			<replace>` + tt.replace + `</replace>
		</edit>
	</edits>
</arguments>`

			_, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			if metadata["lines_added"].(int) != tt.expectedAdded {
				t.Errorf("Expected lines_added=%d, got %v", tt.expectedAdded, metadata["lines_added"])
			}
			if metadata["lines_removed"].(int) != tt.expectedRemoved {
				t.Errorf("Expected lines_removed=%d, got %v", tt.expectedRemoved, metadata["lines_removed"])
			}
		})
	}
}

func TestApplyDiffTool_MissingPath(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	xmlInput := `<arguments>
	<edits>
		<edit>
			<search>test</search>
			<replace>replace</replace>
		</edit>
	</edits>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for missing path")
	}
	if !strings.Contains(err.Error(), "path is required") {
		t.Errorf("Expected 'path is required' error, got: %v", err)
	}
}

func TestApplyDiffTool_MissingEdits(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	xmlInput := `<arguments>
	<path>test.go</path>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for missing edits")
	}
	if !strings.Contains(err.Error(), "at least one edit is required") {
		t.Errorf("Expected 'at least one edit is required' error, got: %v", err)
	}
}

func TestApplyDiffTool_EmptySearchText(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.go")
	writeTestFile(t, testFile, "content")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	xmlInput := `<arguments>
	<path>test.go</path>
	<edits>
		<edit>
			<search></search>
			<replace>replace</replace>
		</edit>
	</edits>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for empty search text")
	}
	if !strings.Contains(err.Error(), "search text cannot be empty") {
		t.Errorf("Expected 'search text cannot be empty' error, got: %v", err)
	}
}

func TestApplyDiffTool_SearchTextNotFound(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.go")
	writeTestFile(t, testFile, "actual content")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	xmlInput := `<arguments>
	<path>test.go</path>
	<edits>
		<edit>
			<search>nonexistent text</search>
			<replace>replace</replace>
		</edit>
	</edits>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for search text not found")
	}
	if !strings.Contains(err.Error(), "search text not found") {
		t.Errorf("Expected 'search text not found' error, got: %v", err)
	}
	// Verify helpful error message with recovery steps
	if !strings.Contains(err.Error(), "Recovery steps") {
		t.Errorf("Expected recovery steps in error message, got: %v", err)
	}
}

func TestApplyDiffTool_MultipleMatchesError(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.go")
	writeTestFile(t, testFile, "return err\nif err != nil {\n\treturn err\n}")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	xmlInput := `<arguments>
	<path>test.go</path>
	<edits>
		<edit>
			<search>return err</search>
			<replace>return nil</replace>
		</edit>
	</edits>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for multiple matches")
	}
	if !strings.Contains(err.Error(), "appears 2 times") {
		t.Errorf("Expected 'appears 2 times' error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "must be unique") {
		t.Errorf("Expected 'must be unique' error, got: %v", err)
	}
	// Verify helpful error message
	if !strings.Contains(err.Error(), "Recovery steps") {
		t.Errorf("Expected recovery steps in error message, got: %v", err)
	}
}

func TestApplyDiffTool_InvalidPath(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	xmlInput := `<arguments>
	<path>../outside/file.go</path>
	<edits>
		<edit>
			<search>test</search>
			<replace>replace</replace>
		</edit>
	</edits>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for path outside workspace")
	}
	if !strings.Contains(err.Error(), "invalid path") {
		t.Errorf("Expected 'invalid path' error, got: %v", err)
	}
}

func TestApplyDiffTool_NonexistentFile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	xmlInput := `<arguments>
	<path>nonexistent.go</path>
	<edits>
		<edit>
			<search>test</search>
			<replace>replace</replace>
		</edit>
	</edits>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("Expected 'failed to read file' error, got: %v", err)
	}
}

func TestApplyDiffTool_NoChanges(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.go")
	originalContent := "unchanged content"
	writeTestFile(t, testFile, originalContent)

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	xmlInput := `<arguments>
	<path>test.go</path>
	<edits>
		<edit>
			<search>unchanged content</search>
			<replace>unchanged content</replace>
		</edit>
	</edits>
</arguments>`

	result, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify no changes message
	if !strings.Contains(result, "No changes made") {
		t.Errorf("Expected 'No changes made' message, got: %s", result)
	}

	// Verify file content unchanged
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != originalContent {
		t.Errorf("Expected file content unchanged, got: %s", string(content))
	}
}

func TestApplyDiffTool_AtomicWrite(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.go")
	originalContent := "original content"
	writeTestFile(t, testFile, originalContent)

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	xmlInput := `<arguments>
	<path>test.go</path>
	<edits>
		<edit>
			<search>original content</search>
			<replace>modified content</replace>
		</edit>
	</edits>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify no temporary file left behind
	tmpPath := testFile + ".tmp"
	if _, statErr := os.Stat(tmpPath); statErr == nil {
		t.Error("Expected temporary file to be removed")
	}

	// Verify file was modified
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != "modified content" {
		t.Errorf("Expected 'modified content', got: %s", string(content))
	}
}

func TestApplyDiffTool_WhitespacePreservation(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.go")
	originalContent := "func test() {\n\t// comment\n\treturn nil\n}"
	writeTestFile(t, testFile, originalContent)

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	xmlInput := `<arguments>
	<path>test.go</path>
	<edits>
		<edit>
			<search>	// comment</search>
			<replace>	// updated comment</replace>
		</edit>
	</edits>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify whitespace preserved
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if !strings.Contains(string(content), "\t// updated comment") {
		t.Errorf("Expected tab to be preserved, got: %s", string(content))
	}
}

func TestApplyDiffTool_SequentialEdits(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	testFile := filepath.Join(tmpDir, "test.go")
	originalContent := "func old1() {}\nfunc old2() {}\nfunc old3() {}"
	writeTestFile(t, testFile, originalContent)

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	// Apply edits sequentially (each edit depends on previous)
	xmlInput := `<arguments>
	<path>test.go</path>
	<edits>
		<edit>
			<search>func old1() {}</search>
			<replace>func new1() {}</replace>
		</edit>
		<edit>
			<search>func old2() {}</search>
			<replace>func new2() {}</replace>
		</edit>
		<edit>
			<search>func old3() {}</search>
			<replace>func new3() {}</replace>
		</edit>
	</edits>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify all edits applied
	if metadata["edits_applied"].(int) != 3 {
		t.Errorf("Expected edits_applied=3, got %v", metadata["edits_applied"])
	}

	// Verify final content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	expected := "func new1() {}\nfunc new2() {}\nfunc new3() {}"
	if string(content) != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, string(content))
	}

	// Verify result message
	if !strings.Contains(result, "Successfully applied 3 edit") {
		t.Errorf("Expected success message with 3 edits, got: %s", result)
	}
}

func TestApplyDiffTool_Metadata(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewApplyDiffTool(guard)

	// Verify name
	if tool.Name() != "apply_diff" {
		t.Errorf("Expected name 'apply_diff', got '%s'", tool.Name())
	}

	// Verify description
	desc := tool.Description()
	if !strings.Contains(desc, "search/replace") {
		t.Errorf("Expected description to mention search/replace, got: %s", desc)
	}

	// Verify schema
	schema := tool.Schema()
	if schema == nil {
		t.Fatal("Expected non-nil schema")
	}

	// Verify loop breaking status
	if tool.IsLoopBreaking() {
		t.Error("ApplyDiffTool should not be loop-breaking")
	}
}
