package workspace

import (
	"fmt"
	"path/filepath"
)

// AddWhitelist adds a directory to the whitelist, allowing file operations
// within that directory even if it's outside the workspace boundaries.
// The directory path is converted to an absolute path and cleaned.
// Symlinks are evaluated to ensure consistent path checking.
//
// This is used for special directories like ~/.forge/tools/ that need
// to be accessible for custom tool management.
func (g *Guard) AddWhitelist(dir string) error {
	if dir == "" {
		return fmt.Errorf("whitelist directory cannot be empty")
	}

	// Convert to absolute path and clean it
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve whitelist directory: %w", err)
	}

	// Evaluate any symlinks to get the real path, handling non-existent paths
	// by resolving parent directories recursively (same approach as Guard.resolveSymlinks)
	evalPath := resolveWhitelistPath(absPath)

	// Check if already whitelisted
	for _, existing := range g.whitelistedDirs {
		if existing == evalPath {
			return nil // Already whitelisted
		}
	}

	g.whitelistedDirs = append(g.whitelistedDirs, evalPath)
	return nil
}

// ClearWhitelist removes all whitelisted directories
func (g *Guard) ClearWhitelist() {
	g.whitelistedDirs = make([]string, 0)
}

// resolveWhitelistPath resolves symlinks in a path, handling non-existent paths
// by recursively resolving parent directories until an existing one is found.
// This ensures consistent path comparison even when whitelisting directories
// that will be created later.
func resolveWhitelistPath(path string) string {
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
			// Reached root without finding existing path, return cleaned original
			return filepath.Clean(path)
		}

		components = append(components, filepath.Base(currentPath))
		currentPath = dir
	}
}

// GetWhitelist returns a copy of the whitelisted directories
func (g *Guard) GetWhitelist() []string {
	whitelist := make([]string, len(g.whitelistedDirs))
	copy(whitelist, g.whitelistedDirs)
	return whitelist
}
