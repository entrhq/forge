package coding

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/security/workspace"
)

// ReadFileTool reads file contents with optional line range support.
type ReadFileTool struct {
	guard *workspace.Guard
}

// NewReadFileTool creates a new ReadFileTool with workspace security.
func NewReadFileTool(guard *workspace.Guard) *ReadFileTool {
	return &ReadFileTool{
		guard: guard,
	}
}

// Name returns the tool name.
func (t *ReadFileTool) Name() string {
	return "read_file"
}

// Description returns the tool description.
func (t *ReadFileTool) Description() string {
	return "Read the contents of a file with optional line range support. Returns line-numbered content for easy reference."
}

// Schema returns the JSON schema for the tool's input parameters.
func (t *ReadFileTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to read (relative to workspace)",
			},
			"start_line": map[string]interface{}{
				"type":        "integer",
				"description": "Optional starting line number (1-based, inclusive)",
			},
			"end_line": map[string]interface{}{
				"type":        "integer",
				"description": "Optional ending line number (1-based, inclusive)",
			},
		},
		[]string{"path"},
	)
}

// Execute reads the file and returns its contents.
func (t *ReadFileTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var input struct {
		XMLName   xml.Name `xml:"arguments"`
		Path      string   `xml:"path"`
		StartLine int      `xml:"start_line"`
		EndLine   int      `xml:"end_line"`
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

	// Check if file is ignored
	if t.guard.ShouldIgnore(absPath) {
		return "", nil, fmt.Errorf("file '%s' is ignored by .gitignore, .forgeignore, or default patterns", input.Path)
	}

	// Read file
	content, err := t.readFileWithLineNumbers(absPath, input.StartLine, input.EndLine)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Build metadata
	metadata := map[string]interface{}{
		"path": input.Path,
	}
	if input.StartLine > 0 {
		metadata["start_line"] = input.StartLine
	}
	if input.EndLine > 0 {
		metadata["end_line"] = input.EndLine
	}

	// Get file info for additional metadata
	info, err := os.Stat(absPath)
	if err == nil {
		metadata["size_bytes"] = info.Size()
		metadata["modified"] = info.ModTime().Format(time.RFC3339)
	}

	return content, metadata, nil
}

// IsLoopBreaking returns false as this tool doesn't break the agent loop.
func (t *ReadFileTool) IsLoopBreaking() bool {
	return false
}

// readFileWithLineNumbers reads a file and returns its contents with line numbers.
// If startLine and endLine are both 0, reads the entire file.
// Line numbers are 1-based and inclusive.
func (t *ReadFileTool) readFileWithLineNumbers(path string, startLine, endLine int) (string, error) {
	// Validate line range if specified
	if err := t.validateLineRange(startLine, endLine); err != nil {
		return "", err
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return t.scanAndFormatLines(file, startLine, endLine)
}

// validateLineRange validates the start and end line numbers.
func (t *ReadFileTool) validateLineRange(startLine, endLine int) error {
	readAll := startLine == 0 && endLine == 0
	if readAll {
		return nil
	}

	if startLine < 1 {
		return fmt.Errorf("start_line must be >= 1, got %d", startLine)
	}
	if endLine < startLine && endLine != 0 {
		return fmt.Errorf("end_line (%d) must be >= start_line (%d)", endLine, startLine)
	}
	return nil
}

// scanAndFormatLines scans the file and formats lines with line numbers.
func (t *ReadFileTool) scanAndFormatLines(file *os.File, startLine, endLine int) (string, error) {
	scanner := bufio.NewScanner(file)
	var builder strings.Builder
	lineNum := 0
	readAll := startLine == 0 && endLine == 0

	for scanner.Scan() {
		lineNum++

		// Skip lines before start_line
		if !readAll && lineNum < startLine {
			continue
		}

		// Stop reading after end_line
		if !readAll && endLine > 0 && lineNum > endLine {
			break
		}

		// Write line with number
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(fmt.Sprintf("%d | %s", lineNum, scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	// Check if we read any lines
	if builder.Len() == 0 {
		if !readAll && startLine > lineNum {
			return "", fmt.Errorf("start_line %d exceeds file length (%d lines)", startLine, lineNum)
		}
		return "", nil // Empty file
	}

	return builder.String(), nil
}
