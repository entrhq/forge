package coding

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/security/workspace"
)

// WriteFileTool creates or overwrites files with workspace validation.
type WriteFileTool struct {
	guard *workspace.Guard
}

// NewWriteFileTool creates a new WriteFileTool with workspace security.
func NewWriteFileTool(guard *workspace.Guard) *WriteFileTool {
	return &WriteFileTool{
		guard: guard,
	}
}

// Name returns the tool name.
func (t *WriteFileTool) Name() string {
	return "write_file"
}

// Description returns the tool description.
func (t *WriteFileTool) Description() string {
	return "Write content to a file, creating it if it doesn't exist or overwriting if it does. Automatically creates parent directories as needed."
}

// Schema returns the JSON schema for the tool's input parameters.
func (t *WriteFileTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to write (relative to workspace)",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		[]string{"path", "content"},
	)
}

// Execute writes content to the specified file.
func (t *WriteFileTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var input struct {
		XMLName xml.Name `xml:"arguments"`
		Path    string   `xml:"path"`
		Content string   `xml:"content"`
	}

	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.Path == "" {
		return "", nil, fmt.Errorf("missing required parameter: path")
	}

	// Validate path with workspace guard
	if err := t.guard.ValidatePath(input.Path); err != nil {
		return "", nil, fmt.Errorf("invalid path: %w", err)
	}

	// Resolve to absolute path
	absPath, err := t.guard.ResolvePath(input.Path)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(absPath)
	if mkdirErr := os.MkdirAll(dir, 0755); mkdirErr != nil {
		return "", nil, fmt.Errorf("failed to create directories: %w", mkdirErr)
	}

	// Check if file exists
	fileExists := false
	if _, statErr := os.Stat(absPath); statErr == nil {
		fileExists = true
	}

	// Write file atomically using a temporary file
	tmpPath := absPath + ".tmp"
	if writeErr := os.WriteFile(tmpPath, []byte(input.Content), 0600); writeErr != nil {
		return "", nil, fmt.Errorf("failed to write temporary file: %w", writeErr)
	}

	// Rename temporary file to target file (atomic operation)
	if renameErr := os.Rename(tmpPath, absPath); renameErr != nil {
		// Clean up temporary file on error
		os.Remove(tmpPath)
		return "", nil, fmt.Errorf("failed to rename temporary file: %w", renameErr)
	}

	// Calculate line changes
	var lineChanges LineChanges
	if fileExists {
		// Read the original content to calculate diff
		originalContent, readErr := os.ReadFile(absPath)
		if readErr == nil {
			lineChanges = CalculateLineChanges(string(originalContent), input.Content)
		}
	} else {
		// New file - all lines are added
		lineChanges = CalculateLineChanges("", input.Content)
	}

	// Get file info for metadata
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Get relative path for output message
	relPath, err := t.guard.MakeRelative(absPath)
	if err != nil {
		relPath = input.Path // Fallback to original path
	}

	var message string
	if fileExists {
		message = fmt.Sprintf("File '%s' overwritten successfully (+%d/-%d lines)",
			relPath, lineChanges.LinesAdded, lineChanges.LinesRemoved)
	} else {
		message = fmt.Sprintf("File '%s' created successfully (+%d lines)",
			relPath, lineChanges.LinesAdded)
	}

	// Build metadata
	metadata := map[string]interface{}{
		"file_path":     input.Path,
		"file_exists":   fileExists,
		"lines_added":   lineChanges.LinesAdded,
		"lines_removed": lineChanges.LinesRemoved,
		"size_bytes":    fileInfo.Size(),
	}

	return message, metadata, nil
}

// IsLoopBreaking returns false as this tool doesn't break the agent loop.
func (t *WriteFileTool) IsLoopBreaking() bool {
	return false
}

// GeneratePreview implements the Previewable interface to show what will be written.
func (t *WriteFileTool) GeneratePreview(ctx context.Context, argsXML []byte) (*tools.ToolPreview, error) {
	var input struct {
		XMLName xml.Name `xml:"arguments"`
		Path    string   `xml:"path"`
		Content string   `xml:"content"`
	}

	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.Path == "" {
		return nil, fmt.Errorf("missing required parameter: path")
	}

	// Validate path
	if err := t.guard.ValidatePath(input.Path); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Resolve to absolute path
	absPath, err := t.guard.ResolvePath(input.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if file exists
	var previewContent string
	var title, description string
	var previewType tools.PreviewType

	if _, statErr := os.Stat(absPath); statErr == nil {
		// File exists - show diff
		originalContent, readErr := os.ReadFile(absPath)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read existing file: %w", readErr)
		}

		relPath, relErr := t.guard.MakeRelative(absPath)
		if relErr != nil || relPath == "" {
			relPath = input.Path
		}

		previewContent = GenerateUnifiedDiff(string(originalContent), input.Content, relPath)
		previewType = tools.PreviewTypeDiff
		title = fmt.Sprintf("Overwrite %s", relPath)
		description = fmt.Sprintf("This will overwrite the existing file %s", relPath)
	} else {
		// File doesn't exist - show new content
		relPath, relErr := t.guard.MakeRelative(absPath)
		if relErr != nil || relPath == "" {
			relPath = input.Path
		}

		previewContent = input.Content
		previewType = tools.PreviewTypeFileWrite
		title = fmt.Sprintf("Create new file %s", relPath)
		description = fmt.Sprintf("This will create a new file at %s", relPath)
	}

	relPath, relErr := t.guard.MakeRelative(absPath)
	if relErr != nil || relPath == "" {
		relPath = input.Path
	}

	language := detectLanguage(relPath)

	return &tools.ToolPreview{
		Type:        previewType,
		Title:       title,
		Description: description,
		Content:     previewContent,
		Metadata: map[string]interface{}{
			"file_path": relPath,
			"language":  language,
			"size":      len(input.Content),
		},
	}, nil
}
