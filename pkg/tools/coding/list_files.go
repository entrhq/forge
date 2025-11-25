package coding

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/security/workspace"
)

// ListFilesTool lists files and directories with optional recursion and filtering.
type ListFilesTool struct {
	guard *workspace.Guard
}

// NewListFilesTool creates a new ListFilesTool with workspace security.
func NewListFilesTool(guard *workspace.Guard) *ListFilesTool {
	return &ListFilesTool{
		guard: guard,
	}
}

// Name returns the tool name.
func (t *ListFilesTool) Name() string {
	return "list_files"
}

// Description returns the tool description.
func (t *ListFilesTool) Description() string {
	return "List files and directories in a specified path. Supports recursive listing and glob pattern filtering."
}

// Schema returns the JSON schema for the tool's input parameters.
func (t *ListFilesTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to list (relative to workspace, defaults to workspace root)",
			},
			"recursive": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to list files recursively (default: false)",
			},
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Optional glob pattern to filter files (e.g., '*.go', 'test_*.py')",
			},
		},
		[]string{}, // No required fields - all optional
	)
}

// Execute lists files in the specified directory.
func (t *ListFilesTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Parse arguments
	var input struct {
		XMLName   xml.Name `xml:"arguments"`
		Path      string   `xml:"path"`
		Recursive bool     `xml:"recursive"`
		Pattern   string   `xml:"pattern"`
	}

	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Default to workspace root if no path provided
	if input.Path == "" {
		input.Path = "."
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

	// Check if path exists and is a directory
	info, err := os.Stat(absPath)
	if err != nil {
		return "", nil, fmt.Errorf("path does not exist: %w", err)
	}
	if !info.IsDir() {
		return "", nil, fmt.Errorf("path is not a directory: %s", input.Path)
	}

	// List files
	var entries []fileEntry
	if input.Recursive {
		entries, err = t.listRecursive(absPath, input.Pattern)
	} else {
		entries, err = t.listDirectory(absPath, input.Pattern)
	}
	if err != nil {
		return "", nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Format output
	result := t.formatEntries(entries)

	// Build metadata
	metadata := map[string]interface{}{
		"path":       input.Path,
		"recursive":  input.Recursive,
		"file_count": len(entries),
	}
	if input.Pattern != "" {
		metadata["pattern"] = input.Pattern
	}

	return result, metadata, nil
}

// IsLoopBreaking returns false as this tool doesn't break the agent loop.
func (t *ListFilesTool) IsLoopBreaking() bool {
	return false
}

// fileEntry represents a file or directory entry.
type fileEntry struct {
	Path  string
	IsDir bool
	Size  int64
}

// listDirectory lists files in a single directory (non-recursive).
func (t *ListFilesTool) listDirectory(dirPath string, pattern string) ([]fileEntry, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var result []fileEntry
	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())

		// Skip ignored paths
		if t.guard.ShouldIgnore(fullPath) {
			continue
		}

		// Apply pattern filter if specified
		if pattern != "" {
			matched, err := filepath.Match(pattern, entry.Name())
			if err != nil {
				return nil, fmt.Errorf("invalid pattern: %w", err)
			}
			if !matched {
				continue
			}
		}

		info, err := entry.Info()
		if err != nil {
			continue // Skip entries we can't stat
		}

		result = append(result, fileEntry{
			Path:  fullPath,
			IsDir: entry.IsDir(),
			Size:  info.Size(),
		})
	}

	return result, nil
}

// listRecursive lists files recursively.
func (t *ListFilesTool) listRecursive(rootPath string, pattern string) ([]fileEntry, error) {
	var result []fileEntry

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip entries with errors
		}

		// Skip the root directory itself
		if path == rootPath {
			return nil
		}

		// Check if path is within workspace (security check)
		if !t.guard.IsWithinWorkspace(path) {
			return filepath.SkipDir
		}

		// Skip ignored paths
		if t.guard.ShouldIgnore(path) {
			if info.IsDir() {
				return filepath.SkipDir // Skip entire directory
			}
			return nil // Skip file
		}

		// Apply pattern filter if specified (only to files)
		if pattern != "" && !info.IsDir() {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err != nil {
				return fmt.Errorf("invalid pattern: %w", err)
			}
			if !matched {
				return nil // Skip non-matching files
			}
		}

		result = append(result, fileEntry{
			Path:  path,
			IsDir: info.IsDir(),
			Size:  info.Size(),
		})

		return nil
	})

	return result, err
}

// formatEntries formats file entries into a readable string.
func (t *ListFilesTool) formatEntries(entries []fileEntry) string {
	if len(entries) == 0 {
		return "No files found"
	}

	// Sort entries: directories first, then by name
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir // Directories first
		}
		return entries[i].Path < entries[j].Path
	})

	var builder strings.Builder
	var totalFiles, totalDirs int

	for _, entry := range entries {
		// Get relative path for display
		relPath, err := t.guard.MakeRelative(entry.Path)
		if err != nil {
			relPath = filepath.Base(entry.Path) // Fallback to just filename
		}

		if entry.IsDir {
			builder.WriteString(fmt.Sprintf("📁 %s/\n", relPath))
			totalDirs++
		} else {
			// Format file size
			sizeStr := formatFileSize(entry.Size)
			builder.WriteString(fmt.Sprintf("📄 %s (%s)\n", relPath, sizeStr))
			totalFiles++
		}
	}

	// Add summary
	builder.WriteString(fmt.Sprintf("\nTotal: %d files, %d directories", totalFiles, totalDirs))

	return builder.String()
}

// formatFileSize formats a file size in bytes to a human-readable string.
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
