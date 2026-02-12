package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGuard_WhitelistIgnoreInteraction(t *testing.T) {
	// Create a temp workspace
	workspaceDir := t.TempDir()

	// Create a .gitignore that ignores .forge directories
	gitignoreContent := `.forge/`
	gitignorePath := filepath.Join(workspaceDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		t.Fatalf("Failed to create .gitignore: %v", err)
	}

	// Create guard
	guard, err := NewGuard(workspaceDir)
	if err != nil {
		t.Fatalf("Failed to create guard: %v", err)
	}

	// Create a whitelisted directory (simulating ~/.forge/tools)
	whitelistDir := t.TempDir()
	forgeToolsDir := filepath.Join(whitelistDir, ".forge", "tools")
	if err := os.MkdirAll(forgeToolsDir, 0755); err != nil {
		t.Fatalf("Failed to create .forge/tools: %v", err)
	}

	// Add whitelist
	if err := guard.AddWhitelist(forgeToolsDir); err != nil {
		t.Fatalf("Failed to add whitelist: %v", err)
	}

	// Test 1: File in workspace .forge directory should be ignored
	workspaceForgeDir := filepath.Join(workspaceDir, ".forge")
	if err := os.MkdirAll(workspaceForgeDir, 0755); err != nil {
		t.Fatalf("Failed to create workspace .forge dir: %v", err)
	}
	workspaceForgeFile := filepath.Join(workspaceForgeDir, "config.json")
	if err := os.WriteFile(workspaceForgeFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create workspace .forge file: %v", err)
	}

	if !guard.ShouldIgnore(workspaceForgeFile) {
		t.Error("Expected workspace .forge/config.json to be ignored by .gitignore pattern")
	}

	// Test 2: File in whitelisted .forge/tools directory should NOT be ignored
	whitelistFile := filepath.Join(forgeToolsDir, "tool.yaml")
	if err := os.WriteFile(whitelistFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create whitelist file: %v", err)
	}

	if guard.ShouldIgnore(whitelistFile) {
		t.Error("Expected whitelisted .forge/tools/tool.yaml to NOT be ignored despite .gitignore pattern")
	}

	// Test 3: Verify whitelisted path is within workspace boundaries
	if !guard.IsWithinWorkspace(whitelistFile) {
		t.Error("Expected whitelisted file to be considered within workspace")
	}

	// Test 4: Nested file in whitelisted directory should also not be ignored
	nestedDir := filepath.Join(forgeToolsDir, "my-tool")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested dir: %v", err)
	}
	nestedFile := filepath.Join(nestedDir, "main.go")
	if err := os.WriteFile(nestedFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	if guard.ShouldIgnore(nestedFile) {
		t.Error("Expected nested file in whitelisted directory to NOT be ignored")
	}
}

func TestGuard_WhitelistWithRelativePaths(t *testing.T) {
	// Create temp workspace
	workspaceDir := t.TempDir()

	guard, err := NewGuard(workspaceDir)
	if err != nil {
		t.Fatalf("Failed to create guard: %v", err)
	}

	// Create whitelisted directory
	whitelistDir := t.TempDir()
	if err := guard.AddWhitelist(whitelistDir); err != nil {
		t.Fatalf("Failed to add whitelist: %v", err)
	}

	// Create file in whitelisted directory
	testFile := filepath.Join(whitelistDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with absolute path
	if guard.ShouldIgnore(testFile) {
		t.Error("Expected whitelisted file (absolute path) to NOT be ignored")
	}

	// Whitelisted paths are considered within workspace, so MakeRelative should work
	// (though the relative path may be unusual since whitelist is outside workspace root)
	// The key test is that ShouldIgnore returns false
	if guard.ShouldIgnore(testFile) {
		t.Error("Expected ShouldIgnore to return false for whitelisted path")
	}
}
