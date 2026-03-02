// Package tui provides result display strategies for tool outputs
package tui

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// DisplayTier represents how a tool result should be displayed in the TUI
type DisplayTier int

const (
	// TierFullInline displays the complete result inline (loop-breaking tools)
	TierFullInline DisplayTier = iota
	// TierSummaryWithPreview displays summary + first few lines
	TierSummaryWithPreview
	// TierSummaryOnly displays only a summary line
	TierSummaryOnly
	// TierOverlayOnly displays nothing inline (handled by overlay)
	TierOverlayOnly
)

// ToolResultClassifier determines how to display a tool result
type ToolResultClassifier struct {
	// Loop-breaking tools (always full inline)
	loopBreakingTools map[string]bool
	// Size threshold for summary vs preview (in lines)
	summaryThreshold int
	// Number of preview lines to show for Tier 2
	previewLines int
}

// NewToolResultClassifier creates a new classifier with default settings
func NewToolResultClassifier() *ToolResultClassifier {
	return &ToolResultClassifier{
		loopBreakingTools: map[string]bool{
			"task_completion": true,
			"ask_question":    true,
			"converse":        true,
		},
		summaryThreshold: 100, // Results >= 100 lines get summary only
		previewLines:     3,   // Show first 3 lines for preview tier
	}
}

// ClassifyToolResult determines the display tier for a tool result
func (c *ToolResultClassifier) ClassifyToolResult(toolName string, result string) DisplayTier {
	// Check if this is a loop-breaking tool (always full inline)
	if c.loopBreakingTools[toolName] {
		return TierFullInline
	}

	// Count lines in result
	lineCount := strings.Count(result, "\n") + 1

	// Large results get summary only
	if lineCount >= c.summaryThreshold {
		return TierSummaryOnly
	}

	// Check for specific tools that should always be summary-only when large
	switch toolName {
	case "read_file", "search_files", "list_files":
		if lineCount >= 50 {
			return TierSummaryOnly
		}
	case "write_file", "apply_diff":
		if lineCount >= 50 {
			return TierSummaryOnly
		}
	}

	// Medium results get summary + preview
	if lineCount >= 20 {
		return TierSummaryWithPreview
	}

	// Small results get full inline
	return TierFullInline
}

// GetPreviewLines extracts the first N lines from a result
func (c *ToolResultClassifier) GetPreviewLines(result string) string {
	// Sanitize output to strip problematic control chars and ANSI
	sanitized := sanitizeOutput(result)
	lines := strings.Split(sanitized, "\n")
	if len(lines) <= c.previewLines {
		return sanitized
	}

	preview := strings.Join(lines[:c.previewLines], "\n")
	remainingLines := len(lines) - c.previewLines
	return fmt.Sprintf("%s\n  ... [%d more lines - Ctrl+V to view full result]", preview, remainingLines)
}

// ToolResultSummarizer generates summaries for tool results
type ToolResultSummarizer struct{}

// NewToolResultSummarizer creates a new summarizer
func NewToolResultSummarizer() *ToolResultSummarizer {
	return &ToolResultSummarizer{}
}

// GenerateSummary creates a one-line summary for a tool result
func (s *ToolResultSummarizer) GenerateSummary(toolName string, result string) string {
	lineCount := strings.Count(result, "\n") + 1
	sizeKB := float64(len(result)) / 1024.0

	switch toolName {
	case "execute_command":
		if exitCode, found := s.extractExitCode(result); found {
			if exitCode == 0 {
				return fmt.Sprintf("Command completed successfully (%d lines, %.1f KB) [Ctrl+V to view]", lineCount, sizeKB)
			}
			return fmt.Sprintf("Command failed with exit code %d (%d lines, %.1f KB) [Ctrl+V to view]", exitCode, lineCount, sizeKB)
		}
		return fmt.Sprintf("Command executed (%d lines, %.1f KB) [Ctrl+V to view]", lineCount, sizeKB)
	case "read_file":
		// Extract filename from result if possible (first line often has it in comments)
		filename := s.extractFilename(result)
		if filename != "" {
			return fmt.Sprintf("Read %d lines from %s (%.1f KB) [Ctrl+V to view]", lineCount, filename, sizeKB)
		}
		return fmt.Sprintf("Read %d lines (%.1f KB) [Ctrl+V to view]", lineCount, sizeKB)

	case "search_files":
		matchCount, fileCount := s.parseSearchResults(result)
		return fmt.Sprintf("Found %d matches in %d files [Ctrl+V to view]", matchCount, fileCount)

	case "list_files":
		fileCount, dirCount := s.parseListResults(result)
		return fmt.Sprintf("Listed %d files and %d directories [Ctrl+V to view]", fileCount, dirCount)

	case "write_file":
		filename := s.extractFilename(result)
		if filename != "" {
			return fmt.Sprintf("Wrote %d lines to %s (%.1f KB)", lineCount, filename, sizeKB)
		}
		return fmt.Sprintf("Wrote %d lines (%.1f KB)", lineCount, sizeKB)

	case "apply_diff":
		editCount := s.parseApplyDiffResults(result)
		filename := s.extractFilename(result)
		if filename != "" && editCount > 0 {
			return fmt.Sprintf("Applied %d edits to %s", editCount, filename)
		}
		return fmt.Sprintf("Applied changes (%.1f KB)", sizeKB)

	default:
		// Generic summary for unknown tools
		return fmt.Sprintf("%s completed (%d lines, %.1f KB) [Ctrl+V to view]", toolName, lineCount, sizeKB)
	}
}

// extractExitCode attempts to determine if a command result string embeds "Exit code: X"
func (s *ToolResultSummarizer) extractExitCode(result string) (int, bool) {
	// Our execute_command tool generally prefixes or appends exit codes in its structured output.
	// But as a fallback, we just look for common patterns to give a cleaner UI summary.
	if strings.Contains(result, "Exit code: ") {
		var code int
		// Let's grab the last line usually
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			if _, err := fmt.Sscanf(line, "Exit code: %d", &code); err == nil {
				return code, true
			}
		}
	}
	return 0, false
}

// extractFilename attempts to extract a filename from tool result using
// multiple heuristics. It looks for file paths in the first few lines and
// returns the most likely filename found.
func (s *ToolResultSummarizer) extractFilename(result string) string {
	lines := strings.Split(result, "\n")
	if len(lines) == 0 {
		return ""
	}

	// Try multiple extraction strategies in order of confidence
	extractors := []filenameExtractor{
		s.extractFromQuotedPath,
		s.extractFromPathPattern,
		s.extractFromExtensionPattern,
	}

	// Search first few lines for filename
	const maxLinesToSearch = 5
	for i := 0; i < min(maxLinesToSearch, len(lines)); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		for _, extractor := range extractors {
			if filename := extractor(line); filename != "" {
				return filename
			}
		}
	}

	return ""
}

// filenameExtractor is a function that attempts to extract a filename from a line
type filenameExtractor func(string) string

// extractFromQuotedPath extracts filenames from quoted paths like "path/to/file.go"
func (s *ToolResultSummarizer) extractFromQuotedPath(line string) string {
	// Match quoted strings that look like file paths
	patterns := []string{
		`"([^"]+\.[a-zA-Z0-9]{1,4})"`,  // "file.ext"
		`'([^']+\.[a-zA-Z0-9]{1,4})'`,  // 'file.ext'
		"`([^`]+\\.[a-zA-Z0-9]{1,4})`", // `file.ext`
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			return filepath.Base(matches[1])
		}
	}

	return ""
}

// extractFromPathPattern extracts filenames from path-like patterns
func (s *ToolResultSummarizer) extractFromPathPattern(line string) string {
	// Match file paths with directory separators
	// Examples: pkg/foo/bar.go, ./internal/test.py, /absolute/path/file.js
	// More flexible pattern that handles ./ prefix and absolute paths
	pathPattern := regexp.MustCompile(`(?:^|\s)((?:\./|/)?(?:[\w-]+/)+[\w-]+\.[a-zA-Z0-9]{1,4})(?:\s|$|:)`)
	if matches := pathPattern.FindStringSubmatch(line); len(matches) > 1 {
		return filepath.Base(matches[1])
	}

	return ""
}

// extractFromExtensionPattern extracts filenames based on common file extensions
func (s *ToolResultSummarizer) extractFromExtensionPattern(line string) string {
	// Common file extensions we want to detect
	extensions := []string{
		// Programming languages
		"go", "py", "js", "ts", "jsx", "tsx", "java", "cpp", "c", "h", "hpp",
		"rs", "rb", "php", "swift", "kt", "scala", "cs", "fs",
		// Config and data
		"json", "yaml", "yml", "toml", "xml", "ini", "conf", "cfg",
		// Documentation
		"md", "txt", "rst", "adoc",
		// Web
		"html", "css", "scss", "sass", "less",
		// Other
		"sql", "sh", "bash", "zsh", "fish",
	}

	// Build a pattern that matches word characters followed by any of these extensions
	extensionPattern := strings.Join(extensions, "|")
	pattern := fmt.Sprintf(`\b([\w-]+\.(?:%s))\b`, extensionPattern)
	re := regexp.MustCompile(pattern)

	if matches := re.FindStringSubmatch(line); len(matches) > 1 {
		return matches[1]
	}

	// Special case: files without extensions (Dockerfile, Makefile, etc.)
	noExtPattern := regexp.MustCompile(`\b(Dockerfile|Makefile|Rakefile|Gemfile|Procfile)\b`)
	if matches := noExtPattern.FindStringSubmatch(line); len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// parseSearchResults extracts match and file counts from search results
func (s *ToolResultSummarizer) parseSearchResults(result string) (matchCount, fileCount int) {
	lines := strings.Split(result, "\n")
	files := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Count matches (non-empty lines are typically matches)
		matchCount++

		// Extract filename before colon (format: "file.go:123: match")
		if colonIdx := strings.Index(line, ":"); colonIdx > 0 {
			filename := line[:colonIdx]
			files[filename] = true
		}
	}

	fileCount = len(files)
	return matchCount, fileCount
}

// parseListResults extracts file and directory counts from list results
func (s *ToolResultSummarizer) parseListResults(result string) (fileCount, dirCount int) {
	lines := strings.Split(result, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "▸") && strings.HasSuffix(line, "/") {
			dirCount++
		} else if strings.HasPrefix(line, "·") {
			fileCount++
		}
	}

	return fileCount, dirCount
}

// parseApplyDiffResults extracts edit count from apply_diff results
func (s *ToolResultSummarizer) parseApplyDiffResults(result string) int {
	// Look for patterns like "Applied 5 edits" in the result
	if strings.Contains(result, "Applied") && strings.Contains(result, "edit") {
		// Simple count of how many "edit" or "change" mentions
		return strings.Count(result, "edit")
	}
	// Fallback: count non-empty lines as edits
	lines := strings.Split(result, "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
