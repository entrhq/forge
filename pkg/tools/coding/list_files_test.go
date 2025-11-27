package coding

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListFilesTool_BasicListing(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create test structure
	writeTestFile(t, filepath.Join(tmpDir, "file1.txt"), "content1")
	writeTestFile(t, filepath.Join(tmpDir, "file2.go"), "content2")
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewListFilesTool(guard)

	xmlInput := `<arguments>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify result contains files
	if !strings.Contains(result, "file1.txt") {
		t.Errorf("Expected result to contain 'file1.txt', got: %s", result)
	}
	if !strings.Contains(result, "file2.go") {
		t.Errorf("Expected result to contain 'file2.go', got: %s", result)
	}
	if !strings.Contains(result, "subdir") {
		t.Errorf("Expected result to contain 'subdir', got: %s", result)
	}

	// Verify metadata
	if metadata["file_count"].(int) != 3 {
		t.Errorf("Expected file_count=3, got %v", metadata["file_count"])
	}
	if metadata["recursive"] != false {
		t.Errorf("Expected recursive=false, got %v", metadata["recursive"])
	}
}

func TestListFilesTool_RecursiveListing(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create nested structure
	writeTestFile(t, filepath.Join(tmpDir, "root.txt"), "root")
	os.MkdirAll(filepath.Join(tmpDir, "dir1", "dir2"), 0755)
	writeTestFile(t, filepath.Join(tmpDir, "dir1", "file1.txt"), "content1")
	writeTestFile(t, filepath.Join(tmpDir, "dir1", "dir2", "file2.txt"), "content2")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewListFilesTool(guard)

	xmlInput := `<arguments>
	<recursive>true</recursive>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify all files are listed
	if !strings.Contains(result, "root.txt") {
		t.Errorf("Expected result to contain 'root.txt', got: %s", result)
	}
	if !strings.Contains(result, "file1.txt") {
		t.Errorf("Expected result to contain 'file1.txt', got: %s", result)
	}
	if !strings.Contains(result, "file2.txt") {
		t.Errorf("Expected result to contain 'file2.txt', got: %s", result)
	}

	// Verify metadata
	if metadata["recursive"] != true {
		t.Errorf("Expected recursive=true, got %v", metadata["recursive"])
	}
	// Should have: root.txt, dir1, file1.txt, dir2, file2.txt = 5 entries
	if metadata["file_count"].(int) != 5 {
		t.Errorf("Expected file_count=5, got %v", metadata["file_count"])
	}
}

func TestListFilesTool_PatternFilter(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create files with different extensions
	writeTestFile(t, filepath.Join(tmpDir, "file1.go"), "go code")
	writeTestFile(t, filepath.Join(tmpDir, "file2.go"), "go code")
	writeTestFile(t, filepath.Join(tmpDir, "file3.txt"), "text")
	writeTestFile(t, filepath.Join(tmpDir, "README.md"), "readme")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewListFilesTool(guard)

	tests := []struct {
		name            string
		pattern         string
		expectedFiles   []string
		unexpectedFiles []string
	}{
		{
			name:            "go files",
			pattern:         "*.go",
			expectedFiles:   []string{"file1.go", "file2.go"},
			unexpectedFiles: []string{"file3.txt", "README.md"},
		},
		{
			name:            "txt files",
			pattern:         "*.txt",
			expectedFiles:   []string{"file3.txt"},
			unexpectedFiles: []string{"file1.go", "file2.go", "README.md"},
		},
		{
			name:            "markdown files",
			pattern:         "*.md",
			expectedFiles:   []string{"README.md"},
			unexpectedFiles: []string{"file1.go", "file3.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xmlInput := fmt.Sprintf(`<arguments>
	<pattern>%s</pattern>
</arguments>`, tt.pattern)

			result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			// Verify expected files are present
			for _, file := range tt.expectedFiles {
				if !strings.Contains(result, file) {
					t.Errorf("Expected result to contain '%s', got: %s", file, result)
				}
			}

			// Verify unexpected files are absent
			for _, file := range tt.unexpectedFiles {
				if strings.Contains(result, file) {
					t.Errorf("Expected result NOT to contain '%s', got: %s", file, result)
				}
			}

			// Verify metadata includes pattern
			if metadata["pattern"] != tt.pattern {
				t.Errorf("Expected pattern='%s', got %v", tt.pattern, metadata["pattern"])
			}
		})
	}
}

func TestListFilesTool_SpecificPath(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create directory structure
	subDir := filepath.Join(tmpDir, "mydir")
	os.Mkdir(subDir, 0755)
	writeTestFile(t, filepath.Join(subDir, "file1.txt"), "content1")
	writeTestFile(t, filepath.Join(subDir, "file2.txt"), "content2")
	writeTestFile(t, filepath.Join(tmpDir, "root.txt"), "root")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewListFilesTool(guard)

	xmlInput := `<arguments>
	<path>mydir</path>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify only files in mydir are listed
	if !strings.Contains(result, "file1.txt") {
		t.Errorf("Expected result to contain 'file1.txt', got: %s", result)
	}
	if !strings.Contains(result, "file2.txt") {
		t.Errorf("Expected result to contain 'file2.txt', got: %s", result)
	}
	if strings.Contains(result, "root.txt") {
		t.Errorf("Expected result NOT to contain 'root.txt', got: %s", result)
	}

	// Verify metadata
	if metadata["path"] != "mydir" {
		t.Errorf("Expected path='mydir', got %v", metadata["path"])
	}
	if metadata["file_count"].(int) != 2 {
		t.Errorf("Expected file_count=2, got %v", metadata["file_count"])
	}
}

func TestListFilesTool_EmptyDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create empty subdirectory
	emptyDir := filepath.Join(tmpDir, "empty")
	os.Mkdir(emptyDir, 0755)

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewListFilesTool(guard)

	xmlInput := `<arguments>
	<path>empty</path>
</arguments>`

	result, metadata, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify result indicates no files
	if !strings.Contains(result, "No files found") {
		t.Errorf("Expected 'No files found', got: %s", result)
	}

	// Verify metadata
	if metadata["file_count"].(int) != 0 {
		t.Errorf("Expected file_count=0, got %v", metadata["file_count"])
	}
}

func TestListFilesTool_NonExistentPath(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewListFilesTool(guard)

	xmlInput := `<arguments>
	<path>nonexistent</path>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for non-existent path")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Expected 'does not exist' error, got: %v", err)
	}
}

func TestListFilesTool_FileNotDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create a file
	testFile := filepath.Join(tmpDir, "file.txt")
	writeTestFile(t, testFile, "content")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewListFilesTool(guard)

	xmlInput := `<arguments>
	<path>file.txt</path>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error when path is a file")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("Expected 'not a directory' error, got: %v", err)
	}
}

func TestListFilesTool_InvalidPath(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewListFilesTool(guard)

	xmlInput := `<arguments>
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

func TestListFilesTool_IgnoredFiles(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create .gitignore
	gitignore := filepath.Join(tmpDir, ".gitignore")
	writeTestFile(t, gitignore, "*.ignore\nignored_dir/\n")

	// Create files to be ignored
	writeTestFile(t, filepath.Join(tmpDir, "file.ignore"), "ignored")
	writeTestFile(t, filepath.Join(tmpDir, "file.txt"), "not ignored")
	os.Mkdir(filepath.Join(tmpDir, "ignored_dir"), 0755)
	writeTestFile(t, filepath.Join(tmpDir, "ignored_dir", "file.txt"), "ignored")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewListFilesTool(guard)

	xmlInput := `<arguments>
	<recursive>true</recursive>
</arguments>`

	result, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify ignored files are not listed
	if strings.Contains(result, "file.ignore") {
		t.Errorf("Expected ignored file NOT to be listed, got: %s", result)
	}
	if strings.Contains(result, "ignored_dir") {
		t.Errorf("Expected ignored directory NOT to be listed, got: %s", result)
	}

	// Verify non-ignored files are listed
	if !strings.Contains(result, "file.txt") {
		t.Errorf("Expected 'file.txt' to be listed, got: %s", result)
	}
}

func TestListFilesTool_InvalidPattern(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	writeTestFile(t, filepath.Join(tmpDir, "file.txt"), "content")

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewListFilesTool(guard)

	// Invalid glob pattern with unmatched bracket
	xmlInput := `<arguments>
	<pattern>[invalid</pattern>
</arguments>`

	_, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err == nil {
		t.Error("Expected error for invalid pattern")
	}
	if !strings.Contains(err.Error(), "invalid pattern") {
		t.Errorf("Expected 'invalid pattern' error, got: %v", err)
	}
}

func TestListFilesTool_FileSizeFormatting(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create files of various sizes
	writeTestFile(t, filepath.Join(tmpDir, "small.txt"), "x")                          // 1 byte
	writeTestFile(t, filepath.Join(tmpDir, "medium.txt"), strings.Repeat("x", 2048))   // 2 KB
	writeTestFile(t, filepath.Join(tmpDir, "large.txt"), strings.Repeat("x", 1048576)) // 1 MB

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewListFilesTool(guard)

	xmlInput := `<arguments>
</arguments>`

	result, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify size formatting (exact format may vary)
	if !strings.Contains(result, "B") {
		t.Errorf("Expected result to contain byte sizes, got: %s", result)
	}

	// Should show file size for each file
	lines := strings.Split(result, "\n")
	fileLineCount := 0
	for _, line := range lines {
		if strings.Contains(line, ".txt") {
			fileLineCount++
			if !strings.Contains(line, "(") || !strings.Contains(line, ")") {
				t.Errorf("Expected file line to contain size in parentheses, got: %s", line)
			}
		}
	}
	if fileLineCount != 3 {
		t.Errorf("Expected 3 file lines, got %d", fileLineCount)
	}
}

func TestListFilesTool_Metadata(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewListFilesTool(guard)

	// Verify name
	if tool.Name() != "list_files" {
		t.Errorf("Expected name 'list_files', got '%s'", tool.Name())
	}

	// Verify description
	desc := tool.Description()
	if !strings.Contains(desc, "List files") {
		t.Errorf("Expected description to mention listing files, got: %s", desc)
	}

	// Verify schema
	schema := tool.Schema()
	if schema == nil {
		t.Fatal("Expected non-nil schema")
	}

	// Verify loop breaking status
	if tool.IsLoopBreaking() {
		t.Error("ListFilesTool should not be loop-breaking")
	}
}

func TestListFilesTool_SortingOrder(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create mixed files and directories
	writeTestFile(t, filepath.Join(tmpDir, "z_file.txt"), "last file")
	writeTestFile(t, filepath.Join(tmpDir, "a_file.txt"), "first file")
	os.Mkdir(filepath.Join(tmpDir, "z_dir"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "a_dir"), 0755)

	guard := createWorkspaceGuard(t, tmpDir)
	tool := NewListFilesTool(guard)

	xmlInput := `<arguments>
</arguments>`

	result, _, err := tool.Execute(context.Background(), []byte(xmlInput))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	lines := strings.Split(result, "\n")

	// Find indices of directories and files
	var dirIndices, fileIndices []int
	for i, line := range lines {
		if strings.Contains(line, "📁") {
			dirIndices = append(dirIndices, i)
		} else if strings.Contains(line, "📄") {
			fileIndices = append(fileIndices, i)
		}
	}

	// Verify directories come before files
	if len(dirIndices) > 0 && len(fileIndices) > 0 {
		lastDirIndex := dirIndices[len(dirIndices)-1]
		firstFileIndex := fileIndices[0]
		if lastDirIndex > firstFileIndex {
			t.Error("Expected directories to be listed before files")
		}
	}
}
