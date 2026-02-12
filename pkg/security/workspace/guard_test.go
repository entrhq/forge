package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewGuard(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "newguard-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name         string
		workspaceDir string
		wantErr      bool
	}{
		{
			name:         "valid existing directory",
			workspaceDir: tmpDir,
			wantErr:      false,
		},
		{
			name:         "current directory",
			workspaceDir: ".",
			wantErr:      false,
		},
		{
			name:         "empty directory",
			workspaceDir: "",
			wantErr:      true,
		},
		{
			name:         "non-existent directory",
			workspaceDir: "/tmp/does-not-exist-12345",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard, err := NewGuard(tt.workspaceDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGuard() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && guard == nil {
				t.Error("NewGuard() returned nil guard without error")
			}
			if !tt.wantErr && guard.workspaceDir == "" {
				t.Error("NewGuard() created guard with empty workspace directory")
			}
		})
	}
}

func TestGuard_ValidatePath(t *testing.T) {
	// Create temporary workspace for testing
	tmpDir, err := os.MkdirTemp("", "workspace-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create guard: %v", err)
	}

	// Create a subdirectory for testing
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid file in workspace",
			path:    "file.txt",
			wantErr: false,
		},
		{
			name:    "valid file in subdirectory",
			path:    "subdir/file.txt",
			wantErr: false,
		},
		{
			name:    "workspace root",
			path:    ".",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "parent directory traversal",
			path:    "../outside.txt",
			wantErr: true,
		},
		{
			name:    "multiple parent traversals",
			path:    "../../outside.txt",
			wantErr: true,
		},
		{
			name:    "absolute path outside workspace",
			path:    "/etc/passwd",
			wantErr: true,
		},
		{
			name:    "hidden traversal",
			path:    "subdir/../../outside.txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := guard.ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGuard_ResolvePath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workspace-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create guard: %v", err)
	}

	// Guard's workspace dir is now symlink-resolved, so use that for comparisons
	workspaceDir := guard.WorkspaceDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
		check   func(string) bool
	}{
		{
			name:    "relative path",
			path:    "file.txt",
			wantErr: false,
			check: func(resolved string) bool {
				expected := filepath.Join(workspaceDir, "file.txt")
				return resolved == expected
			},
		},
		{
			name:    "path with dots",
			path:    "./subdir/../file.txt",
			wantErr: false,
			check: func(resolved string) bool {
				expected := filepath.Join(workspaceDir, "file.txt")
				return resolved == expected
			},
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "tilde expansion with trailing path",
			path:    "~/.forge/tools/test.yaml",
			wantErr: false,
			check: func(resolved string) bool {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return false
				}
				expected := filepath.Join(homeDir, ".forge/tools/test.yaml")
				return resolved == expected
			},
		},
		{
			name:    "tilde expansion alone",
			path:    "~",
			wantErr: false,
			check: func(resolved string) bool {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return false
				}
				return resolved == homeDir
			},
		},
		{
			name:    "tilde with slash only",
			path:    "~/",
			wantErr: false,
			check: func(resolved string) bool {
				homeDir, err := os.UserHomeDir()
				if err != nil {
					return false
				}
				return resolved == homeDir
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := guard.ResolvePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolvePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(resolved) {
				t.Errorf("ResolvePath() = %v, check failed", resolved)
			}
		})
	}
}

func TestGuard_IsWithinWorkspace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workspace-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create guard: %v", err)
	}

	// Use the guard's resolved workspace directory for all tests
	workspaceDir := guard.WorkspaceDir()

	tests := []struct {
		name    string
		absPath string
		want    bool
	}{
		{
			name:    "workspace root",
			absPath: workspaceDir,
			want:    true,
		},
		{
			name:    "file in workspace",
			absPath: filepath.Join(workspaceDir, "file.txt"),
			want:    true,
		},
		{
			name:    "subdirectory in workspace",
			absPath: filepath.Join(workspaceDir, "subdir", "file.txt"),
			want:    true,
		},
		{
			name:    "parent directory",
			absPath: filepath.Dir(workspaceDir),
			want:    false,
		},
		{
			name:    "sibling directory",
			absPath: filepath.Join(filepath.Dir(workspaceDir), "other"),
			want:    false,
		},
		{
			name:    "root directory",
			absPath: "/",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := guard.IsWithinWorkspace(tt.absPath); got != tt.want {
				t.Errorf("IsWithinWorkspace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGuard_MakeRelative(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workspace-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create guard: %v", err)
	}

	// Use the guard's resolved workspace directory
	workspaceDir := guard.WorkspaceDir()

	tests := []struct {
		name    string
		absPath string
		want    string
		wantErr bool
	}{
		{
			name:    "file in workspace root",
			absPath: filepath.Join(workspaceDir, "file.txt"),
			want:    "file.txt",
			wantErr: false,
		},
		{
			name:    "file in subdirectory",
			absPath: filepath.Join(workspaceDir, "subdir", "file.txt"),
			want:    filepath.Join("subdir", "file.txt"),
			wantErr: false,
		},
		{
			name:    "workspace root",
			absPath: workspaceDir,
			want:    ".",
			wantErr: false,
		},
		{
			name:    "path outside workspace",
			absPath: "/etc/passwd",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := guard.MakeRelative(tt.absPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("MakeRelative() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MakeRelative() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGuard_WorkspaceDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "workspace-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Resolve the tmpDir to account for symlinks (like /tmp -> /private/tmp on macOS)
	resolvedTmpDir, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("Failed to resolve tmpDir symlinks: %v", err)
	}

	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create guard: %v", err)
	}

	if got := guard.WorkspaceDir(); got != resolvedTmpDir {
		t.Errorf("WorkspaceDir() = %v, want %v", got, resolvedTmpDir)
	}
}

// TestGuard_SymlinkSecurity tests that symbolic links are properly evaluated
func TestGuard_SymlinkSecurity(t *testing.T) {
	// Create temporary directories
	tmpDir, err := os.MkdirTemp("", "workspace-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outsideDir, err := os.MkdirTemp("", "outside-test-*")
	if err != nil {
		t.Fatalf("Failed to create outside dir: %v", err)
	}
	defer os.RemoveAll(outsideDir)

	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create guard: %v", err)
	}

	// Create a symlink pointing outside the workspace
	symlinkPath := filepath.Join(tmpDir, "link-to-outside")
	err = os.Symlink(outsideDir, symlinkPath)
	if err != nil {
		t.Skipf("Cannot create symlink (may need permissions): %v", err)
	}

	// Attempting to access through the symlink should fail
	err = guard.ValidatePath("link-to-outside/file.txt")
	if err == nil {
		t.Error("ValidatePath() should reject symlink pointing outside workspace")
	}
}
