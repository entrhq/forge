// Package workspace provides security mechanisms for enforcing workspace boundaries
// on file system operations. It prevents path traversal attacks and ensures all
// operations stay within the designated working directory.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Guard enforces workspace boundary restrictions on file paths.
// It validates that all file operations remain within the workspace directory,
// preventing path traversal attacks and unauthorized file access.
type Guard struct {
	workspaceDir    string         // Absolute path to workspace root
	ignoreMatcher   *IgnoreMatcher // Pattern matcher for ignore rules
	whitelistedDirs []string       // Additional allowed directories outside workspace
}

// NewGuard creates a new workspace guard for the given directory.
// The directory path is converted to an absolute path, cleaned, and symlinks are evaluated.
// It also initializes the ignore matcher with patterns from defaults, .gitignore, and .forgeignore.
func NewGuard(workspaceDir string) (*Guard, error) {
	if workspaceDir == "" {
		return nil, fmt.Errorf("workspace directory cannot be empty")
	}

	// Convert to absolute path and clean it
	absPath, err := filepath.Abs(workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace directory: %w", err)
	}

	// Evaluate any symlinks in the workspace path itself
	evalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate workspace directory symlinks: %w", err)
	}

	// Initialize ignore matcher with patterns from all sources
	ignoreMatcher, err := NewIgnoreMatcher(evalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ignore matcher: %w", err)
	}

	return &Guard{
		workspaceDir:    evalPath,
		ignoreMatcher:   ignoreMatcher,
		whitelistedDirs: make([]string, 0),
	}, nil
}

// ValidatePath checks if the given path is within the workspace boundaries.
// It resolves the path to an absolute path and ensures it's a child of the workspace.
//
// Returns an error if:
// - The path is empty
// - The path contains invalid characters or patterns
// - The resolved path is outside the workspace
// - The path attempts directory traversal
func (g *Guard) ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Resolve to absolute path
	resolvedPath, err := g.ResolvePath(path)
	if err != nil {
		return err
	}

	// Check if resolved path is within workspace
	if !g.IsWithinWorkspace(resolvedPath) {
		return fmt.Errorf("path '%s' is outside workspace boundaries", path)
	}

	return nil
}

// ResolvePath converts a relative or absolute path to an absolute path
// within the workspace context. It cleans the path and resolves any
// symbolic links. Supports tilde expansion for paths starting with ~/.
func (g *Guard) ResolvePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Expand tilde to home directory if present
	expandedPath := path
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to expand ~: %w", err)
		}
		expandedPath = filepath.Join(homeDir, path[2:])
	} else if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to expand ~: %w", err)
		}
		expandedPath = homeDir
	}

	// Clean the path to remove any .. or . components
	cleanPath := filepath.Clean(expandedPath)

	// If path is already absolute, use it directly
	// Otherwise, join with workspace directory
	var absPath string
	if filepath.IsAbs(cleanPath) {
		absPath = cleanPath
	} else {
		absPath = filepath.Join(g.workspaceDir, cleanPath)
	}

	// Clean the absolute path
	absPath = filepath.Clean(absPath)

	// Evaluate any symbolic links
	evalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If the file doesn't exist yet, that's okay for write operations
		// Just ensure the parent directory structure would be valid
		parentDir := filepath.Dir(absPath)
		if parentDir != absPath {
			evalParent, parentErr := filepath.EvalSymlinks(parentDir)
			if parentErr == nil {
				// Use the evaluated parent with the original filename
				evalPath = filepath.Join(evalParent, filepath.Base(absPath))
			} else {
				// Parent doesn't exist either, evaluate workspace and use relative from there
				evalWorkspace, wsErr := filepath.EvalSymlinks(g.workspaceDir)
				if wsErr == nil {
					// Get the relative path from workspace
					relPath, relErr := filepath.Rel(g.workspaceDir, absPath)
					if relErr == nil {
						evalPath = filepath.Join(evalWorkspace, relPath)
					} else {
						evalPath = absPath
					}
				} else {
					evalPath = absPath
				}
			}
		} else {
			evalPath = absPath
		}
	}

	return evalPath, nil
}

// IsWithinWorkspace checks if an absolute path is within the workspace boundaries
// or within any whitelisted directory. This is the core security check - it ensures
// a path is either the workspace itself, a child directory of the workspace, or
// within an explicitly whitelisted directory.
func (g *Guard) IsWithinWorkspace(absPath string) bool {
	// Evaluate symlinks to ensure consistent path comparison
	// This is important on systems like macOS where /var -> /private/var
	evalPath := g.resolveSymlinks(absPath)

	// Check if path is exactly the workspace or a child of it
	if evalPath == g.workspaceDir || strings.HasPrefix(evalPath+string(filepath.Separator), g.workspaceDir+string(filepath.Separator)) {
		return true
	}

	// Check whitelisted directories
	for _, whitelisted := range g.whitelistedDirs {
		if evalPath == whitelisted || strings.HasPrefix(evalPath+string(filepath.Separator), whitelisted+string(filepath.Separator)) {
			return true
		}
	}

	return false
}

// resolveSymlinks resolves symlinks in a path, handling non-existent paths
// by recursively resolving parent directories until an existing one is found.
func (g *Guard) resolveSymlinks(path string) string {
	// Try direct resolution first
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}

	// For non-existent paths, collect path components and resolve from root
	var components []string
	currentPath := path

	// Walk up the directory tree collecting components until we find something that exists
	for {
		if resolved, err := filepath.EvalSymlinks(currentPath); err == nil {
			// Found an existing path, now reconstruct with collected components
			result := resolved
			for i := len(components) - 1; i >= 0; i-- {
				result = filepath.Join(result, components[i])
			}
			return result
		}

		// Path doesn't exist, move up one level
		dir := filepath.Dir(currentPath)
		if dir == currentPath || dir == "." || dir == "/" {
			// Reached root without finding existing path, return original
			return path
		}

		components = append(components, filepath.Base(currentPath))
		currentPath = dir
	}
}

// WorkspaceDir returns the absolute path of the workspace directory.
func (g *Guard) WorkspaceDir() string {
	return g.workspaceDir
}

// MakeRelative converts an absolute path to a path relative to the workspace.
// Returns an error if the path is not within the workspace.
func (g *Guard) MakeRelative(absPath string) (string, error) {
	if !g.IsWithinWorkspace(absPath) {
		return "", fmt.Errorf("path '%s' is not within workspace", absPath)
	}

	relPath, err := filepath.Rel(g.workspaceDir, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to make path relative: %w", err)
	}

	return relPath, nil
}

// ShouldIgnore checks if a path should be ignored based on loaded ignore patterns.
// The path can be either absolute or relative - it will be converted to relative for matching.
// Returns true if the path matches any ignore pattern (considering precedence and negation).
// Whitelisted paths are never ignored, regardless of ignore patterns.
func (g *Guard) ShouldIgnore(path string) bool {
	// Get absolute path for whitelist checking
	var absPath string
	if filepath.IsAbs(path) {
		absPath = path
	} else {
		absPath = filepath.Join(g.workspaceDir, path)
	}

	// Resolve symlinks for consistent comparison
	evalPath := g.resolveSymlinks(absPath)

	// Check if path is in a whitelisted directory - never ignore whitelisted paths
	for _, whitelisted := range g.whitelistedDirs {
		if evalPath == whitelisted || strings.HasPrefix(evalPath+string(filepath.Separator), whitelisted+string(filepath.Separator)) {
			return false
		}
	}

	// Convert to relative path for pattern matching
	var relPath string
	if filepath.IsAbs(path) {
		var err error
		relPath, err = g.MakeRelative(path)
		if err != nil {
			// If we can't make it relative, it's outside workspace, so don't ignore
			// (workspace boundary check will catch this elsewhere)
			return false
		}
	} else {
		relPath = path
	}

	// Check if path is a directory by attempting to stat it
	isDir := false
	if info, err := os.Lstat(absPath); err == nil {
		isDir = info.IsDir()
	}

	return g.ignoreMatcher.ShouldIgnore(relPath, isDir)
}
