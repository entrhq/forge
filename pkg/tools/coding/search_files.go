package coding

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/security/workspace"
)

// SearchFilesTool searches for patterns in files using regular expressions.
type SearchFilesTool struct {
	guard *workspace.Guard
}

// NewSearchFilesTool creates a new SearchFilesTool with workspace security.
func NewSearchFilesTool(guard *workspace.Guard) *SearchFilesTool {
	return &SearchFilesTool{
		guard: guard,
	}
}

// Name returns the tool name.
func (t *SearchFilesTool) Name() string {
	return "search_files"
}

// Description returns the tool description.
func (t *SearchFilesTool) Description() string {
	return "Search for patterns in files using regular expressions. Returns matches with surrounding context lines."
}

// Schema returns the JSON schema for the tool's input parameters.
func (t *SearchFilesTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to search in (relative to workspace, defaults to workspace root)",
			},
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Regular expression pattern to search for",
			},
			"file_pattern": map[string]interface{}{
				"type":        "string",
				"description": "Optional glob pattern to filter files (e.g., '*.go', '*.py')",
			},
			"context_lines": map[string]interface{}{
				"type":        "integer",
				"description": "Number of context lines to show before and after match (default: 2)",
			},
		},
		[]string{"pattern"}, // pattern is required
	)
}

// Execute searches for the pattern in files.
func (t *SearchFilesTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Parse arguments
	var input struct {
		XMLName      xml.Name `xml:"arguments"`
		Path         string   `xml:"path"`
		Pattern      string   `xml:"pattern"`
		FilePattern  string   `xml:"file_pattern"`
		ContextLines int      `xml:"context_lines"`
	}

	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.Pattern == "" {
		return "", nil, fmt.Errorf("missing required parameter: pattern")
	}

	// Default to workspace root if no path provided
	if input.Path == "" {
		input.Path = "."
	}

	// Default context lines
	if input.ContextLines == 0 {
		input.ContextLines = 2
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

	// Compile regex pattern
	regex, err := regexp.Compile(input.Pattern)
	if err != nil {
		return "", nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Search files
	matches, err := t.searchDirectory(absPath, regex, input.FilePattern, input.ContextLines)
	if err != nil {
		return "", nil, fmt.Errorf("search failed: %w", err)
	}

	// Format output
	result := t.formatMatches(matches)

	// Build metadata
	metadata := map[string]interface{}{
		"path":          input.Path,
		"pattern":       input.Pattern,
		"match_count":   len(matches),
		"context_lines": input.ContextLines,
	}
	if input.FilePattern != "" {
		metadata["file_pattern"] = input.FilePattern
	}

	// Count unique files
	fileSet := make(map[string]bool)
	for _, match := range matches {
		fileSet[match.FilePath] = true
	}
	metadata["files_with_matches"] = len(fileSet)

	return result, metadata, nil
}

// IsLoopBreaking returns false as this tool doesn't break the agent loop.
func (t *SearchFilesTool) IsLoopBreaking() bool {
	return false
}

// searchMatch represents a single match in a file.
type searchMatch struct {
	FilePath    string
	LineNumber  int
	Line        string
	Context     []string // Lines before and after
	ContextFrom int      // Starting line number of context
}

// searchDirectory searches all files in a directory recursively.
func (t *SearchFilesTool) searchDirectory(dirPath string, regex *regexp.Regexp, filePattern string, contextLines int) ([]searchMatch, error) {
	var matches []searchMatch

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip entries with errors
		}

		// Skip directories
		if info.IsDir() {
			// Check if directory is within workspace
			if !t.guard.IsWithinWorkspace(path) {
				return filepath.SkipDir
			}
			// Skip ignored directories
			if t.guard.ShouldIgnore(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if path is within workspace
		if !t.guard.IsWithinWorkspace(path) {
			return nil
		}

		// Skip ignored files
		if t.guard.ShouldIgnore(path) {
			return nil
		}

		// Apply file pattern filter if specified
		if filePattern != "" {
			matched, matchErr := filepath.Match(filePattern, filepath.Base(path))
			if matchErr != nil {
				return fmt.Errorf("invalid file pattern: %w", matchErr)
			}
			if !matched {
				return nil
			}
		}

		// Skip binary files (simple heuristic)
		if isBinaryFile(path) {
			return nil
		}

		// Search file
		fileMatches, err := t.searchFile(path, regex, contextLines)
		if err != nil {
			return nil // Skip files we can't read
		}

		matches = append(matches, fileMatches...)
		return nil
	})

	return matches, err
}

// searchFile searches for pattern in a single file.
func (t *SearchFilesTool) searchFile(filePath string, regex *regexp.Regexp, contextLines int) ([]searchMatch, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []searchMatch
	var lines []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Read all lines first (for context)
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		lines = append(lines, line)

		// Check if line matches
		if regex.MatchString(line) {
			// Calculate context range
			contextFrom := lineNum - contextLines
			if contextFrom < 1 {
				contextFrom = 1
			}
			contextTo := lineNum + contextLines
			if contextTo > len(lines) {
				contextTo = len(lines)
			}
			_ = contextTo // Will be used when adding context lines later

			// Collect context lines (we'll update this after reading all lines)
			match := searchMatch{
				FilePath:    filePath,
				LineNumber:  lineNum,
				Line:        line,
				ContextFrom: contextFrom,
			}
			matches = append(matches, match)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Now add context to matches
	for i := range matches {
		contextFrom := matches[i].ContextFrom
		contextTo := matches[i].LineNumber + contextLines
		if contextTo > len(lines) {
			contextTo = len(lines)
		}

		// Extract context lines (excluding the match line itself)
		for j := contextFrom; j <= contextTo; j++ {
			if j != matches[i].LineNumber {
				matches[i].Context = append(matches[i].Context, lines[j-1])
			}
		}
	}

	return matches, nil
}

// formatMatches formats search matches into a readable string.
func (t *SearchFilesTool) formatMatches(matches []searchMatch) string {
	if len(matches) == 0 {
		return "No matches found"
	}

	var builder strings.Builder
	currentFile := ""

	for _, match := range matches {
		// Get relative path for display
		relPath, err := t.guard.MakeRelative(match.FilePath)
		if err != nil {
			relPath = match.FilePath
		}

		// Print file header if changed
		if match.FilePath != currentFile {
			if currentFile != "" {
				builder.WriteString("\n")
			}
			builder.WriteString(fmt.Sprintf("📄 %s\n", relPath))
			builder.WriteString(strings.Repeat("-", 60) + "\n")
			currentFile = match.FilePath
		}

		// Print context before match
		contextLineNum := match.ContextFrom
		for _, line := range match.Context {
			if contextLineNum < match.LineNumber {
				builder.WriteString(fmt.Sprintf("  %d | %s\n", contextLineNum, line))
				contextLineNum++
			} else {
				// This is context after the match
				break
			}
		}

		// Print the matching line (highlighted)
		builder.WriteString(fmt.Sprintf("▶ %d | %s\n", match.LineNumber, match.Line))

		// Print context after match
		for i := match.LineNumber - match.ContextFrom; i < len(match.Context); i++ {
			builder.WriteString(fmt.Sprintf("  %d | %s\n", contextLineNum, match.Context[i]))
			contextLineNum++
		}

		builder.WriteString("\n")
	}

	// Add summary
	builder.WriteString(fmt.Sprintf("Found %d matches", len(matches)))

	return builder.String()
}

// isBinaryFile performs a simple check to determine if a file is binary.
// This is a heuristic and may not be 100% accurate.
func isBinaryFile(path string) bool {
	// Check file extension first (common binary extensions)
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".bin": true, ".dat": true, ".db": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
		".o": true, ".a": true, ".pyc": true,
	}
	if binaryExts[ext] {
		return true
	}

	// Read first few bytes to check for binary content
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return false
	}

	// Check for null bytes (common in binary files)
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}

	return false
}
