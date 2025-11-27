package coding

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchFilesTool_BasicSearch(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create test files
	writeTestFile(t, filepath.Join(tmpDir, "file1.txt"), "Hello World\nThis is a test\nGoodbye World")
	writeTestFile(t, filepath.Join(tmpDir, "file2.txt"), "Another file\nWith World in it\nEnd of file")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
	<pattern>World</pattern>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify matches found
	if !strings.Contains(result, "Hello World") {
		t.Errorf("Expected result to contain 'Hello World', got: %s", result)
	}
	if !strings.Contains(result, "Goodbye World") {
		t.Errorf("Expected result to contain 'Goodbye World', got: %s", result)
	}
	if !strings.Contains(result, "With World in it") {
		t.Errorf("Expected result to contain 'With World in it', got: %s", result)
	}

	// Verify metadata
	if metadata["match_count"].(int) != 3 {
		t.Errorf("Expected match_count=3, got %v", metadata["match_count"])
	}
	if metadata["files_with_matches"].(int) != 2 {
		t.Errorf("Expected files_with_matches=2, got %v", metadata["files_with_matches"])
	}
	if metadata["pattern"] != "World" {
		t.Errorf("Expected pattern='World', got %v", metadata["pattern"])
	}
}

func TestSearchFilesTool_RegexPattern(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTestFile(t, filepath.Join(tmpDir, "code.go"), "func main() {\n\tfmt.Println(\"test\")\n}\nfunc helper() {}")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
	<pattern>func \w+\(</pattern>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify both function declarations found
	if !strings.Contains(result, "func main()") {
		t.Errorf("Expected result to contain 'func main()', got: %s", result)
	}
	if !strings.Contains(result, "func helper()") {
		t.Errorf("Expected result to contain 'func helper()', got: %s", result)
	}

	// Verify metadata
	if metadata["match_count"].(int) != 2 {
		t.Errorf("Expected match_count=2, got %v", metadata["match_count"])
	}
}

func TestSearchFilesTool_FilePattern(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create files with different extensions
	writeTestFile(t, filepath.Join(tmpDir, "test.go"), "package main\nfunc test() {}")
	writeTestFile(t, filepath.Join(tmpDir, "test.txt"), "package main\nfunc test() {}")
	writeTestFile(t, filepath.Join(tmpDir, "README.md"), "package main")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
	<pattern>package</pattern>
	<file_pattern>*.go</file_pattern>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify only .go file is searched
	if !strings.Contains(result, "test.go") {
		t.Errorf("Expected result to contain 'test.go', got: %s", result)
	}
	if strings.Contains(result, "test.txt") {
		t.Errorf("Expected result NOT to contain 'test.txt', got: %s", result)
	}
	if strings.Contains(result, "README.md") {
		t.Errorf("Expected result NOT to contain 'README.md', got: %s", result)
	}

	// Verify metadata
	if metadata["file_pattern"] != "*.go" {
		t.Errorf("Expected file_pattern='*.go', got %v", metadata["file_pattern"])
	}
	if metadata["files_with_matches"].(int) != 1 {
		t.Errorf("Expected files_with_matches=1, got %v", metadata["files_with_matches"])
	}
}

func TestSearchFilesTool_ContextLines(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	content := "line 1\nline 2\nline 3\nMATCH HERE\nline 5\nline 6\nline 7"
	writeTestFile(t, filepath.Join(tmpDir, "file.txt"), content)

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	tests := []struct {
		name           string
		contextLines   int
		expectedBefore []string
		expectedAfter  []string
	}{
		{
			name:           "default context (2 lines)",
			contextLines:   0, // Will use default of 2
			expectedBefore: []string{"line 2", "line 3"},
			expectedAfter:  []string{"line 5", "line 6"},
		},
		{
			name:           "1 line context",
			contextLines:   1,
			expectedBefore: []string{"line 3"},
			expectedAfter:  []string{"line 5"},
		},
		{
			name:           "3 lines context",
			contextLines:   3,
			expectedBefore: []string{"line 1", "line 2", "line 3"},
			expectedAfter:  []string{"line 5", "line 6", "line 7"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xmlInput := fmt.Sprintf(`<arguments>
	<pattern>MATCH</pattern>
	<context_lines>%d</context_lines>
</arguments>`, tt.contextLines)

			result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			// Verify context lines are present
			for _, line := range tt.expectedBefore {
				if !strings.Contains(result, line) {
					t.Errorf("Expected result to contain context line '%s', got: %s", line, result)
				}
			}
			for _, line := range tt.expectedAfter {
				if !strings.Contains(result, line) {
					t.Errorf("Expected result to contain context line '%s', got: %s", line, result)
				}
			}

			// Verify context_lines in metadata
			expectedContext := tt.contextLines
			if expectedContext == 0 {
				expectedContext = 2 // default
			}
			if metadata["context_lines"].(int) != expectedContext {
				t.Errorf("Expected context_lines=%d, got %v", expectedContext, metadata["context_lines"])
			}
		})
	}
}

func TestSearchFilesTool_NoMatches(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTestFile(t, filepath.Join(tmpDir, "file.txt"), "Nothing to see here\nJust regular text")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
	<pattern>NONEXISTENT</pattern>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify no matches message
	if !strings.Contains(result, "No matches found") {
		t.Errorf("Expected 'No matches found', got: %s", result)
	}

	// Verify metadata
	if metadata["match_count"].(int) != 0 {
		t.Errorf("Expected match_count=0, got %v", metadata["match_count"])
	}
	if metadata["files_with_matches"].(int) != 0 {
		t.Errorf("Expected files_with_matches=0, got %v", metadata["files_with_matches"])
	}
}

func TestSearchFilesTool_MissingPattern(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for missing pattern")
	}
	if !strings.Contains(err.Error(), "missing required parameter: pattern") {
		t.Errorf("Expected 'missing required parameter' error, got: %v", err)
	}
}

func TestSearchFilesTool_InvalidRegex(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
	<pattern>[invalid(regex</pattern>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for invalid regex")
	}
	if !strings.Contains(err.Error(), "invalid regex pattern") {
		t.Errorf("Expected 'invalid regex pattern' error, got: %v", err)
	}
}

func TestSearchFilesTool_InvalidPath(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
	<pattern>test</pattern>
	<path>../outside</path>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for path outside workspace")
	}
	if !strings.Contains(err.Error(), "invalid path") {
		t.Errorf("Expected 'invalid path' error, got: %v", err)
	}
}

func TestSearchFilesTool_RecursiveSearch(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create nested directory structure
	os.MkdirAll(filepath.Join(tmpDir, "dir1", "dir2"), 0755)
	writeTestFile(t, filepath.Join(tmpDir, "root.txt"), "FINDME at root")
	writeTestFile(t, filepath.Join(tmpDir, "dir1", "level1.txt"), "FINDME at level 1")
	writeTestFile(t, filepath.Join(tmpDir, "dir1", "dir2", "level2.txt"), "FINDME at level 2")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
	<pattern>FINDME</pattern>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify all nested files are searched
	if !strings.Contains(result, "root.txt") {
		t.Errorf("Expected result to contain 'root.txt', got: %s", result)
	}
	if !strings.Contains(result, "level1.txt") {
		t.Errorf("Expected result to contain 'level1.txt', got: %s", result)
	}
	if !strings.Contains(result, "level2.txt") {
		t.Errorf("Expected result to contain 'level2.txt', got: %s", result)
	}

	// Verify metadata
	if metadata["match_count"].(int) != 3 {
		t.Errorf("Expected match_count=3, got %v", metadata["match_count"])
	}
	if metadata["files_with_matches"].(int) != 3 {
		t.Errorf("Expected files_with_matches=3, got %v", metadata["files_with_matches"])
	}
}

func TestSearchFilesTool_IgnoredFiles(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create .gitignore
	writeTestFile(t, filepath.Join(tmpDir, ".gitignore"), "*.ignore\nignored_dir/")

	// Create files
	writeTestFile(t, filepath.Join(tmpDir, "included.txt"), "MATCH in included")
	writeTestFile(t, filepath.Join(tmpDir, "excluded.ignore"), "MATCH in excluded")
	os.Mkdir(filepath.Join(tmpDir, "ignored_dir"), 0755)
	writeTestFile(t, filepath.Join(tmpDir, "ignored_dir", "file.txt"), "MATCH in ignored dir")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
	<pattern>MATCH</pattern>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify only non-ignored file is searched
	if !strings.Contains(result, "included.txt") {
		t.Errorf("Expected result to contain 'included.txt', got: %s", result)
	}
	if strings.Contains(result, "excluded.ignore") {
		t.Errorf("Expected result NOT to contain 'excluded.ignore', got: %s", result)
	}
	if strings.Contains(result, "ignored_dir") {
		t.Errorf("Expected result NOT to contain 'ignored_dir', got: %s", result)
	}

	// Verify metadata
	if metadata["match_count"].(int) != 1 {
		t.Errorf("Expected match_count=1, got %v", metadata["match_count"])
	}
}

func TestSearchFilesTool_BinaryFileSkipping(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create text file
	writeTestFile(t, filepath.Join(tmpDir, "text.txt"), "MATCH in text")

	// Create binary file (with null bytes)
	binaryFile := filepath.Join(tmpDir, "binary.bin")
	binaryContent := []byte{'M', 'A', 'T', 'C', 'H', 0, 0, 0, 'b', 'i', 'n', 'a', 'r', 'y'}
	if err := os.WriteFile(binaryFile, binaryContent, 0600); err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
	<pattern>MATCH</pattern>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify text file is searched but binary is skipped
	if !strings.Contains(result, "text.txt") {
		t.Errorf("Expected result to contain 'text.txt', got: %s", result)
	}
	if strings.Contains(result, "binary.bin") {
		t.Errorf("Expected result NOT to contain 'binary.bin', got: %s", result)
	}

	// Should only find match in text file
	if metadata["match_count"].(int) != 1 {
		t.Errorf("Expected match_count=1, got %v", metadata["match_count"])
	}
}

func TestSearchFilesTool_InvalidFilePattern(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTestFile(t, filepath.Join(tmpDir, "file.txt"), "content")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
	<pattern>content</pattern>
	<file_pattern>[invalid</file_pattern>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for invalid file pattern")
	}
	if !strings.Contains(err.Error(), "invalid file pattern") {
		t.Errorf("Expected 'invalid file pattern' error, got: %v", err)
	}
}

func TestSearchFilesTool_MultipleMatchesInFile(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	content := "First MATCH here\nSome text\nSecond MATCH here\nMore text\nThird MATCH here"
	writeTestFile(t, filepath.Join(tmpDir, "file.txt"), content)

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
	<pattern>MATCH</pattern>
	<context_lines>1</context_lines>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify all matches found
	if !strings.Contains(result, "First MATCH") {
		t.Errorf("Expected result to contain 'First MATCH', got: %s", result)
	}
	if !strings.Contains(result, "Second MATCH") {
		t.Errorf("Expected result to contain 'Second MATCH', got: %s", result)
	}
	if !strings.Contains(result, "Third MATCH") {
		t.Errorf("Expected result to contain 'Third MATCH', got: %s", result)
	}

	// Verify metadata
	if metadata["match_count"].(int) != 3 {
		t.Errorf("Expected match_count=3, got %v", metadata["match_count"])
	}
	if metadata["files_with_matches"].(int) != 1 {
		t.Errorf("Expected files_with_matches=1, got %v", metadata["files_with_matches"])
	}
}

func TestSearchFilesTool_CaseSensitiveSearch(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTestFile(t, filepath.Join(tmpDir, "file.txt"), "UPPERCASE\nlowercase\nMiXeDcAsE")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
	<pattern>UPPER</pattern>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify case-sensitive matching
	if !strings.Contains(result, "UPPERCASE") {
		t.Errorf("Expected result to contain 'UPPERCASE', got: %s", result)
	}

	// Should only match uppercase
	if metadata["match_count"].(int) != 1 {
		t.Errorf("Expected match_count=1, got %v", metadata["match_count"])
	}
}

func TestSearchFilesTool_Metadata(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	// Verify name
	if tool.Name() != "search_files" {
		t.Errorf("Expected name 'search_files', got '%s'", tool.Name())
	}

	// Verify description
	desc := tool.Description()
	if !strings.Contains(desc, "Search for patterns") {
		t.Errorf("Expected description to mention searching, got: %s", desc)
	}

	// Verify schema
	schema := tool.Schema()
	if schema == nil {
		t.Fatal("Expected non-nil schema")
	}

	// Verify loop breaking status
	if tool.IsLoopBreaking() {
		t.Error("SearchFilesTool should not be loop-breaking")
	}
}

func TestSearchFilesTool_SpecificDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create directory structure
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)
	writeTestFile(t, filepath.Join(tmpDir, "root.txt"), "MATCH at root")
	writeTestFile(t, filepath.Join(subDir, "sub.txt"), "MATCH in subdir")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewSearchFilesTool(guard)

	xmlInput := `<arguments>
	<pattern>MATCH</pattern>
	<path>subdir</path>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify only subdir is searched
	if !strings.Contains(result, "sub.txt") {
		t.Errorf("Expected result to contain 'sub.txt', got: %s", result)
	}
	if strings.Contains(result, "root.txt") {
		t.Errorf("Expected result NOT to contain 'root.txt', got: %s", result)
	}

	// Verify metadata
	if metadata["path"] != "subdir" {
		t.Errorf("Expected path='subdir', got %v", metadata["path"])
	}
	if metadata["match_count"].(int) != 1 {
		t.Errorf("Expected match_count=1, got %v", metadata["match_count"])
	}
}
