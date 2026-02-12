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

	// Evaluate any symlinks to get the real path
	evalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// If path doesn't exist yet, use the cleaned absolute path
		// This allows whitelisting directories that will be created later
		evalPath = filepath.Clean(absPath)
	}

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

// GetWhitelist returns a copy of the whitelisted directories
func (g *Guard) GetWhitelist() []string {
	whitelist := make([]string, len(g.whitelistedDirs))
	copy(whitelist, g.whitelistedDirs)
	return whitelist
}
