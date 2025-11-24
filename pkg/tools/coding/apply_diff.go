package coding

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/security/workspace"
)

// ApplyDiffTool applies search/replace operations to files for precise code editing.
type ApplyDiffTool struct {
	guard *workspace.Guard
}

// NewApplyDiffTool creates a new ApplyDiffTool with workspace security.
func NewApplyDiffTool(guard *workspace.Guard) *ApplyDiffTool {
	return &ApplyDiffTool{
		guard: guard,
	}
}

// Name returns the tool name.
func (t *ApplyDiffTool) Name() string {
	return "apply_diff"
}

// Description returns the tool description.
func (t *ApplyDiffTool) Description() string {
	return "Apply precise search/replace operations to files. Supports multiple edits in a single operation for surgical code changes."
}

// Schema returns the JSON schema for the tool's input parameters.
func (t *ApplyDiffTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to edit (relative to workspace)",
			},
			"edits": map[string]interface{}{
				"type":        "array",
				"description": "List of search/replace operations to apply",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"search": map[string]interface{}{
							"type":        "string",
							"description": "Exact text to search for (must match exactly including whitespace)",
						},
						"replace": map[string]interface{}{
							"type":        "string",
							"description": "Text to replace the search text with",
						},
					},
					"required": []string{"search", "replace"},
				},
			},
		},
		[]string{"path", "edits"},
	)
}

// Execute performs the search/replace operations and returns metadata about the changes.
//
//nolint:gocyclo // TODO: refactor to reduce complexity
func (t *ApplyDiffTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var input struct {
		XMLName xml.Name `xml:"arguments"`
		Path    string   `xml:"path"`
		Edits   []struct {
			Search  string `xml:"search"`
			Replace string `xml:"replace"`
		} `xml:"edits>edit"`
	}

	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.Path == "" {
		return "", nil, fmt.Errorf("path is required")
	}

	if len(input.Edits) == 0 {
		return "", nil, fmt.Errorf("at least one edit is required")
	}

	// Resolve path to absolute path
	absPath, err := t.guard.ResolvePath(input.Path)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Validate path is within workspace
	if validateErr := t.guard.ValidatePath(input.Path); validateErr != nil {
		return "", nil, fmt.Errorf("invalid path: %w", validateErr)
	}

	// Read current file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read file: %w", err)
	}

	fileContent := string(content)
	originalContent := fileContent

	// Apply each edit in sequence
	appliedEdits := 0
	totalLinesAdded := 0
	totalLinesRemoved := 0

	for i, edit := range input.Edits {
		if edit.Search == "" {
			return "", nil, fmt.Errorf("edit %d: search text cannot be empty", i+1)
		}

		// Check if search text exists
		if !strings.Contains(fileContent, edit.Search) {
			return "", nil, fmt.Errorf("edit %d: search text not found in file:\n%s", i+1, edit.Search)
		}

		// Count occurrences to warn about multiple matches
		count := strings.Count(fileContent, edit.Search)
		if count > 1 {
			return "", nil, fmt.Errorf("edit %d: search text appears %d times in file, must be unique", i+1, count)
		}

		// Track line changes for this edit
		searchLines := strings.Count(edit.Search, "\n") + 1
		replaceLines := strings.Count(edit.Replace, "\n") + 1

		if replaceLines > searchLines {
			totalLinesAdded += replaceLines - searchLines
		} else if searchLines > replaceLines {
			totalLinesRemoved += searchLines - replaceLines
		}

		// Apply the replacement
		fileContent = strings.Replace(fileContent, edit.Search, edit.Replace, 1)
		appliedEdits++
	}

	// Only write if changes were made
	if fileContent == originalContent {
		return "No changes made to file", nil, nil
	}

	// Write the modified content atomically
	tmpPath := absPath + ".tmp"
	if writeErr := os.WriteFile(tmpPath, []byte(fileContent), 0600); writeErr != nil {
		return "", nil, fmt.Errorf("failed to write temporary file: %w", writeErr)
	}

	if renameErr := os.Rename(tmpPath, absPath); renameErr != nil {
		os.Remove(tmpPath)
		return "", nil, fmt.Errorf("failed to rename temporary file: %w", renameErr)
	}

	// Get relative path for response
	relPath, err := t.guard.MakeRelative(absPath)
	if err != nil {
		relPath = input.Path
	}

	// Build metadata about the changes
	metadata := map[string]interface{}{
		"edits_applied": appliedEdits,
		"lines_added":   totalLinesAdded,
		"lines_removed": totalLinesRemoved,
		"file_path":     relPath,
	}

	return fmt.Sprintf("Successfully applied %d edit(s) to %s", appliedEdits, relPath), metadata, nil
}

// IsLoopBreaking returns whether this tool should break the agent loop.
func (t *ApplyDiffTool) IsLoopBreaking() bool {
	return false
}

// XMLExample provides a concrete XML usage example for this tool.
func (t *ApplyDiffTool) XMLExample() string {
	return `<tool>
<server_name>local</server_name>
<tool_name>apply_diff</tool_name>
<arguments>
  <path>src/main.go</path>
  <edits>
    <edit>
      <search><![CDATA[func oldFunction() {
	return "old"
}]]></search>
      <replace><![CDATA[func newFunction() {
	return "new"
}]]></replace>
    </edit>
    <edit>
      <search><![CDATA[const oldValue = 42]]></search>
      <replace><![CDATA[const newValue = 100]]></replace>
    </edit>
  </edits>
</arguments>
</tool>`
}

// GeneratePreview implements the Previewable interface to show a diff preview.
func (t *ApplyDiffTool) GeneratePreview(ctx context.Context, argsXML []byte) (*tools.ToolPreview, error) {
	var input struct {
		XMLName xml.Name `xml:"arguments"`
		Path    string   `xml:"path"`
		Edits   []struct {
			Search  string `xml:"search"`
			Replace string `xml:"replace"`
		} `xml:"edits>edit"`
	}

	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	if len(input.Edits) == 0 {
		return nil, fmt.Errorf("at least one edit is required")
	}

	// Resolve and validate path
	absPath, err := t.guard.ResolvePath(input.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	if validateErr := t.guard.ValidatePath(input.Path); validateErr != nil {
		return nil, fmt.Errorf("invalid path: %w", validateErr)
	}

	// Read current file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	originalContent := string(content)
	modifiedContent := originalContent

	// Apply edits to generate modified version
	for i, edit := range input.Edits {
		if edit.Search == "" {
			return nil, fmt.Errorf("edit %d: search text cannot be empty", i+1)
		}

		if !strings.Contains(modifiedContent, edit.Search) {
			return nil, fmt.Errorf("edit %d: search text not found in file", i+1)
		}

		count := strings.Count(modifiedContent, edit.Search)
		if count > 1 {
			return nil, fmt.Errorf("edit %d: search text appears %d times in file, must be unique", i+1, count)
		}

		modifiedContent = strings.Replace(modifiedContent, edit.Search, edit.Replace, 1)
	}

	// Generate diff
	relPath, err := t.guard.MakeRelative(absPath)
	if err != nil || relPath == "" {
		relPath = input.Path
	}

	diffContent := GenerateUnifiedDiff(originalContent, modifiedContent, relPath)

	// Detect file language from extension for syntax highlighting metadata
	language := detectLanguage(relPath)

	return &tools.ToolPreview{
		Type:        tools.PreviewTypeDiff,
		Title:       fmt.Sprintf("Apply %d edit(s) to %s", len(input.Edits), relPath),
		Description: fmt.Sprintf("This will modify %s with %d search/replace operation(s)", relPath, len(input.Edits)),
		Content:     diffContent,
		Metadata: map[string]interface{}{
			"file_path":  relPath,
			"language":   language,
			"edit_count": len(input.Edits),
		},
	}, nil
}

// detectLanguage returns a language identifier based on file extension
func detectLanguage(filename string) string {
	// Map of file extensions to language names
	langMap := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "javascript",
		".java": "java",
		".rs":   "rust",
		".c":    "c",
		".h":    "c",
		".cpp":  "cpp",
		".hpp":  "cpp",
		".rb":   "ruby",
		".php":  "php",
		".html": "html",
		".css":  "css",
		".json": "json",
		".yaml": "yaml",
		".yml":  "yaml",
		".md":   "markdown",
	}

	ext := strings.ToLower(filename)
	for suffix, lang := range langMap {
		if strings.HasSuffix(ext, suffix) {
			return lang
		}
	}
	return "text"
}
