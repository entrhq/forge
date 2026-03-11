package workspace

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// defaultIgnorePatterns contains hardcoded patterns that are always ignored.
// These are common directories and files that should be excluded from file operations.
var defaultIgnorePatterns = []string{
	"node_modules/",
	".git/",
	".env",
	".env.*",
	"*.log",
	".DS_Store",
	"vendor/",
	"__pycache__/",
	"*.pyc",
	".vscode/",
	".idea/",
	"dist/",
	"build/",
	"tmp/",
	"temp/",
	"coverage/",
	".next/",
	".nuxt/",
	"target/",
	"*.swp",
	"*.swo",
	"*~",
}

// ignorePattern represents a single ignore pattern with metadata.
type ignorePattern struct {
	pattern  string // Original pattern string
	negation bool   // True if this is a negation pattern (starts with !)
	dirOnly  bool   // True if pattern only matches directories (ends with /)
	isGlob   bool   // True if pattern contains glob characters
	source   string // Source of pattern: "default", "gitignore", "forgeignore"
}

// IgnoreMatcher handles pattern matching for file ignore rules.
// It supports layered patterns from multiple sources with defined precedence.
type IgnoreMatcher struct {
	patterns []ignorePattern
}

// NewIgnoreMatcher creates a new ignore matcher and loads patterns from all sources.
// Pattern loading order (all are merged, last match wins):
// 1. Default hardcoded patterns
// 2. .gitignore patterns (if file exists)
// 3. .forgeignore patterns (if file exists)
func NewIgnoreMatcher(workspaceDir string) (*IgnoreMatcher, error) {
	m := &IgnoreMatcher{
		patterns: make([]ignorePattern, 0),
	}

	// Load default patterns
	m.loadDefaultPatterns()

	// Load .gitignore if it exists
	gitignorePath := filepath.Join(workspaceDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		if err := m.loadPatternsFromFile(gitignorePath, "gitignore"); err != nil {
			// Log warning but continue - don't fail on parse errors
			fmt.Fprintf(os.Stderr, "Warning: failed to parse .gitignore: %v\n", err)
		}
	}

	// Load .forgeignore if it exists
	forgeignorePath := filepath.Join(workspaceDir, ".forgeignore")
	if _, err := os.Stat(forgeignorePath); err == nil {
		if err := m.loadPatternsFromFile(forgeignorePath, "forgeignore"); err != nil {
			// Log warning but continue - don't fail on parse errors
			fmt.Fprintf(os.Stderr, "Warning: failed to parse .forgeignore: %v\n", err)
		}
	}

	return m, nil
}

// loadDefaultPatterns loads the hardcoded default ignore patterns.
func (m *IgnoreMatcher) loadDefaultPatterns() {
	for _, pattern := range defaultIgnorePatterns {
		m.addPattern(pattern, "default")
	}
}

// loadPatternsFromFile loads patterns from a gitignore-style file.
func (m *IgnoreMatcher) loadPatternsFromFile(path, source string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Add pattern with source information
		m.addPattern(line, source)
	}

	return scanner.Err()
}

// addPattern adds a pattern to the matcher with metadata.
func (m *IgnoreMatcher) addPattern(pattern, source string) {
	// Check for negation
	negation := false
	if strings.HasPrefix(pattern, "!") {
		negation = true
		pattern = pattern[1:]
	}

	// Check for directory-only pattern
	dirOnly := strings.HasSuffix(pattern, "/")
	if dirOnly {
		pattern = strings.TrimSuffix(pattern, "/")
	}

	// Check if pattern contains glob characters
	isGlob := strings.ContainsAny(pattern, "*?[]")

	m.patterns = append(m.patterns, ignorePattern{
		pattern:  pattern,
		negation: negation,
		dirOnly:  dirOnly,
		isGlob:   isGlob,
		source:   source,
	})
}

// ShouldIgnore checks if a path should be ignored based on loaded patterns.
// The path should be relative to the workspace root.
// Returns true if the path matches an ignore pattern (last match wins).
func (m *IgnoreMatcher) ShouldIgnore(relPath string, isDir bool) bool {
	// Normalize path separators for matching
	relPath = filepath.ToSlash(relPath)

	// Track whether path is ignored (last match wins)
	ignored := false

	// Check each pattern in order
	for _, p := range m.patterns {
		var matches bool

		// For directory-only patterns (ending with /), we need to check if:
		// 1. The path IS that directory (if isDir is true)
		// 2. The path is INSIDE that directory (for both files and dirs)
		if p.dirOnly {
			// Check if path matches the directory name
			dirMatches := m.matchPattern(relPath, p.pattern, p.isGlob)
			// Check if path is inside this directory
			insideDir := strings.HasPrefix(relPath, p.pattern+"/")

			matches = dirMatches || insideDir
		} else {
			matches = m.matchPattern(relPath, p.pattern, p.isGlob)
		}

		if matches {
			// If this is a negation pattern, unignore the path
			// Otherwise, ignore it
			ignored = !p.negation
		}
	}

	return ignored
}

// matchPattern checks if a path matches a pattern, dispatching to helpers.
func (m *IgnoreMatcher) matchPattern(path, pattern string, isGlob bool) bool {
	pattern = filepath.ToSlash(pattern)

	// Check for exact match first, which is the cheapest check.
	if path == pattern {
		return true
	}

	if isGlob {
		return m.matchGlob(path, pattern)
	}

	return m.matchSimple(path, pattern)
}

// matchGlob handles matching for glob patterns.
func (m *IgnoreMatcher) matchGlob(path, pattern string) bool {
	// Try matching against the full path.
	if globMatch(pattern, path) {
		return true
	}

	// Try matching against just the base name.
	if globMatch(pattern, filepath.Base(path)) {
		return true
	}

	// Try matching against each path segment. This handles patterns like "**/foo.go"
	parts := strings.Split(path, "/")
	for i := range parts {
		// Check subpath from root, e.g., "a/b" in "a/b/c.txt"
		if globMatch(pattern, strings.Join(parts[:i+1], "/")) {
			return true
		}
		// Also check just the individual segment, e.g., "b" in "a/b/c.txt"
		if globMatch(pattern, parts[i]) {
			return true
		}
	}

	return false
}

// matchSimple handles matching for simple, non-glob patterns (e.g., directory or exact names).
func (m *IgnoreMatcher) matchSimple(path, pattern string) bool {
	// Check if the pattern is a prefix of the path (e.g., "node_modules/" matching "node_modules/pkg/file.js").
	if strings.HasPrefix(path, pattern+"/") {
		return true
	}

	parts := strings.Split(path, "/")

	// Check if the pattern matches any full path component. This handles "node_modules" in "a/node_modules/b"
	if slices.Contains(parts, pattern) {
		return true
	}

	// Check if pattern matches a subpath from the root. e.g. pattern "a/b" in "a/b/c"
	for i := range parts {
		if strings.Join(parts[:i+1], "/") == pattern {
			return true
		}
	}

	return false
}

// globMatch is a helper to run filepath.Match and handle the error.
func globMatch(pattern, name string) bool {
	// This helper is a standalone function as it doesn't depend on the IgnoreMatcher instance.
	matched, err := filepath.Match(pattern, name)
	return err == nil && matched
}

// PatternCount returns the total number of loaded patterns.
// Useful for debugging and testing.
func (m *IgnoreMatcher) PatternCount() int {
	return len(m.patterns)
}

// Patterns returns a copy of all loaded patterns for debugging.
func (m *IgnoreMatcher) Patterns() []string {
	result := make([]string, len(m.patterns))
	for i, p := range m.patterns {
		prefix := ""
		if p.negation {
			prefix = "!"
		}
		suffix := ""
		if p.dirOnly {
			suffix = "/"
		}
		result[i] = fmt.Sprintf("%s%s%s [%s]", prefix, p.pattern, suffix, p.source)
	}
	return result
}
