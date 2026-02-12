package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGuard_AddWhitelist(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}

	// Create a directory outside workspace
	outsideDir := t.TempDir()

	// Initially should not be within workspace
	if guard.IsWithinWorkspace(outsideDir) {
		t.Error("Directory should not be within workspace initially")
	}

	// Add to whitelist
	if err := guard.AddWhitelist(outsideDir); err != nil {
		t.Fatalf("AddWhitelist() error = %v", err)
	}

	// Should now be considered within workspace
	if !guard.IsWithinWorkspace(outsideDir) {
		t.Errorf("Directory should be within workspace after whitelisting")
	}

	// Child of whitelisted directory should also be allowed
	childPath := filepath.Join(outsideDir, "subdir", "file.txt")
	if !guard.IsWithinWorkspace(childPath) {
		t.Errorf("Child of whitelisted directory should be within workspace")
	}
}

func TestGuard_AddWhitelistNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}

	// Add non-existent directory to whitelist (should work)
	nonExistent := filepath.Join(tmpDir, "..", "nonexistent")
	if err := guard.AddWhitelist(nonExistent); err != nil {
		t.Fatalf("AddWhitelist() error = %v", err)
	}

	// Should be in whitelist
	whitelist := guard.GetWhitelist()
	if len(whitelist) != 1 {
		t.Errorf("GetWhitelist() returned %d items, want 1", len(whitelist))
	}
}

func TestGuard_AddWhitelistDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}

	outsideDir := t.TempDir()

	// Add twice
	if err := guard.AddWhitelist(outsideDir); err != nil {
		t.Fatalf("AddWhitelist() error = %v", err)
	}
	if err := guard.AddWhitelist(outsideDir); err != nil {
		t.Fatalf("AddWhitelist() second call error = %v", err)
	}

	// Should only have one entry
	whitelist := guard.GetWhitelist()
	if len(whitelist) != 1 {
		t.Errorf("GetWhitelist() returned %d items, want 1", len(whitelist))
	}
}

func TestGuard_AddWhitelistEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}

	err = guard.AddWhitelist("")
	if err == nil {
		t.Error("AddWhitelist() expected error for empty path")
	}
}

func TestGuard_ClearWhitelist(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}

	outsideDir := t.TempDir()

	// Add to whitelist
	if err := guard.AddWhitelist(outsideDir); err != nil {
		t.Fatalf("AddWhitelist() error = %v", err)
	}

	// Verify it's whitelisted
	if !guard.IsWithinWorkspace(outsideDir) {
		t.Error("Directory should be within workspace after whitelisting")
	}

	// Clear whitelist
	guard.ClearWhitelist()

	// Should no longer be whitelisted
	if guard.IsWithinWorkspace(outsideDir) {
		t.Error("Directory should not be within workspace after clearing whitelist")
	}

	// Whitelist should be empty
	whitelist := guard.GetWhitelist()
	if len(whitelist) != 0 {
		t.Errorf("GetWhitelist() returned %d items, want 0", len(whitelist))
	}
}

func TestGuard_GetWhitelist(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}

	// Initially empty
	whitelist := guard.GetWhitelist()
	if len(whitelist) != 0 {
		t.Errorf("GetWhitelist() initially returned %d items, want 0", len(whitelist))
	}

	// Add multiple directories
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	if err := guard.AddWhitelist(dir1); err != nil {
		t.Fatalf("AddWhitelist() error = %v", err)
	}
	if err := guard.AddWhitelist(dir2); err != nil {
		t.Fatalf("AddWhitelist() error = %v", err)
	}

	// Should have both
	whitelist = guard.GetWhitelist()
	if len(whitelist) != 2 {
		t.Errorf("GetWhitelist() returned %d items, want 2", len(whitelist))
	}

	// Verify it's a copy (modifying returned slice shouldn't affect guard)
	whitelist[0] = "/modified/path"
	newWhitelist := guard.GetWhitelist()
	if newWhitelist[0] == "/modified/path" {
		t.Error("GetWhitelist() should return a copy, not the original slice")
	}
}

func TestGuard_ValidatePathWithWhitelist(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("NewGuard() error = %v", err)
	}

	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "test.txt")

	// Create the file
	if err := os.WriteFile(outsideFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Should fail validation initially
	if err := guard.ValidatePath(outsideFile); err == nil {
		t.Error("ValidatePath() expected error for path outside workspace")
	}

	// Add to whitelist
	if err := guard.AddWhitelist(outsideDir); err != nil {
		t.Fatalf("AddWhitelist() error = %v", err)
	}

	// Should now pass validation
	if err := guard.ValidatePath(outsideFile); err != nil {
		t.Errorf("ValidatePath() error = %v, want nil after whitelisting", err)
	}
}
