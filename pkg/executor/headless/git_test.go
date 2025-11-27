package headless

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a temporary directory with an initialized git repository
func setupTestRepo(t *testing.T) string {
	t.Helper()

	testDir := t.TempDir()

	// Initialize git repo
	cmd := []string{"git", "init"}
	if err := execCommand(testDir, cmd...); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user
	if err := execCommand(testDir, "git", "config", "user.email", "test@example.com"); err != nil {
		t.Fatalf("failed to set git email: %v", err)
	}
	if err := execCommand(testDir, "git", "config", "user.name", "Test User"); err != nil {
		t.Fatalf("failed to set git name: %v", err)
	}

	// Create initial commit
	readmePath := filepath.Join(testDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repository\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}

	if err := execCommand(testDir, "git", "add", "README.md"); err != nil {
		t.Fatalf("failed to add README: %v", err)
	}

	if err := execCommand(testDir, "git", "commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	return testDir
}

// execCommand runs a git command in the given directory
func execCommand(dir string, args ...string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func TestGitManager_CheckWorkspaceClean(t *testing.T) {
	testDir := setupTestRepo(t)

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{}, "")

	// Test clean workspace
	if err := gm.CheckWorkspaceClean(ctx); err != nil {
		t.Error("workspace should be clean after initial commit")
	}

	// Create a new file
	testFile := filepath.Join(testDir, "test.txt")
	if writeErr := os.WriteFile(testFile, []byte("test content"), 0644); writeErr != nil {
		t.Fatalf("failed to write test file: %v", writeErr)
	}

	// Workspace should now be dirty
	if err := gm.CheckWorkspaceClean(ctx); err == nil {
		t.Error("workspace should be dirty after creating new file")
	}
}

func TestGitManager_GetCurrentBranch(t *testing.T) {
	testDir := setupTestRepo(t)

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{}, "")

	branch, err := gm.GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}

	// Default branch should be main or master
	if branch != "main" && branch != "master" {
		t.Errorf("unexpected default branch: %s", branch)
	}
}

func TestGitManager_CreateBranch(t *testing.T) {
	testDir := setupTestRepo(t)

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{}, "")

	// Create new branch
	newBranch := "feature/test-branch"
	if err := gm.CreateBranch(ctx, newBranch); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Verify we're on the new branch
	currentBranch, err := gm.GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	if currentBranch != newBranch {
		t.Errorf("expected branch %s, got %s", newBranch, currentBranch)
	}
}

func TestGitManager_Commit(t *testing.T) {
	testDir := setupTestRepo(t)

	config := GitConfig{
		AutoCommit:  true,
		AuthorName:  "Forge Bot",
		AuthorEmail: "forge@example.com",
	}
	gm := NewGitManager(testDir, config, "")

	// Create a file to commit
	testFile := filepath.Join(testDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ctx := context.Background()

	// Commit the changes
	if err := gm.Commit(ctx, "Add test file"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Workspace should be clean after commit
	if err := gm.CheckWorkspaceClean(ctx); err != nil {
		t.Error("workspace should be clean after commit")
	}
}

func TestGitManager_CommitExcludesConfigFile(t *testing.T) {
	testDir := setupTestRepo(t)

	// Create a config file
	configFile := filepath.Join(testDir, "forge-config.yaml")
	if err := os.WriteFile(configFile, []byte("task: test\nmode: write\n"), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create another file
	testFile := filepath.Join(testDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	config := GitConfig{
		AutoCommit:  true,
		AuthorName:  "Forge Bot",
		AuthorEmail: "forge@example.com",
	}
	gm := NewGitManager(testDir, config, configFile)

	ctx := context.Background()

	// Commit the changes
	if err := gm.Commit(ctx, "Add test file"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Check git status - config file should still be untracked
	output, err := gm.execGit(ctx, "status", "--porcelain")
	if err != nil {
		t.Fatalf("failed to check git status: %v", err)
	}

	// Config file should appear as untracked
	if !strings.Contains(output, "forge-config.yaml") {
		t.Error("config file should remain untracked after commit")
	}

	// Test file should not appear (it was committed)
	if strings.Contains(output, "test.txt") {
		t.Error("test file should be committed and not appear in status")
	}
}

func TestGitManager_GetChangedFiles(t *testing.T) {
	testDir := setupTestRepo(t)

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{}, "")

	// Initially no changes
	files, err := gm.GetChangedFiles(ctx)
	if err != nil {
		t.Fatalf("failed to get changed files: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no changed files, got %d", len(files))
	}

	// Create a new file
	testFile := filepath.Join(testDir, "test.txt")
	if writeErr := os.WriteFile(testFile, []byte("test content"), 0644); writeErr != nil {
		t.Fatalf("failed to write test file: %v", writeErr)
	}

	// Should detect new file
	files, err = gm.GetChangedFiles(ctx)
	if err != nil {
		t.Fatalf("failed to get changed files: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 changed file, got %d", len(files))
	}
	if len(files) > 0 && files[0] != "test.txt" {
		t.Errorf("expected test.txt, got %s", files[0])
	}
}

func TestGitManager_Rollback(t *testing.T) {
	testDir := setupTestRepo(t)

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{}, "")

	// Modify existing file
	readmePath := filepath.Join(testDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Modified\n"), 0644); err != nil {
		t.Fatalf("failed to modify README: %v", err)
	}

	// Create a new file
	testFile := filepath.Join(testDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Rollback changes
	if err := gm.Rollback(ctx); err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}

	// Workspace should be clean
	if err := gm.CheckWorkspaceClean(ctx); err != nil {
		t.Error("workspace should be clean after rollback")
	}

	// Test file should be removed
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("test file should be removed after rollback")
	}

	// README should be restored to original content
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README: %v", err)
	}
	if string(content) != "# Test Repository\n" {
		t.Errorf("README content not restored, got: %s", string(content))
	}
}

func TestGenerateBranchName(t *testing.T) {
	branchName := GenerateBranchName("forge/test")

	// Branch name should contain the prefix
	if !strings.HasPrefix(branchName, "forge/test-") {
		t.Errorf("branch name should start with 'forge/test-', got: %s", branchName)
	}

	// Should contain timestamp
	if len(branchName) < len("forge/test-20060102-150405") {
		t.Errorf("branch name should contain timestamp, got: %s", branchName)
	}
}

func TestGitManager_GenerateCommitMessage(t *testing.T) {
	testDir := setupTestRepo(t)

	ctx := context.Background()

	tests := []struct {
		name   string
		config GitConfig
		task   string
		want   string
	}{
		{
			name: "custom message",
			config: GitConfig{
				CommitMessage: "Custom commit message",
			},
			task: "Any task",
			want: "Custom commit message",
		},
		{
			name:   "generated message",
			config: GitConfig{},
			task:   "Fix bug in authentication",
			want:   "chore: Fix bug in authentication\n\nAutomated changes via Forge headless mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := NewGitManager(testDir, tt.config, "")
			msg := gm.GenerateCommitMessage(ctx, tt.task)
			if msg != tt.want {
				t.Errorf("GenerateCommitMessage() = %v, want %v", msg, tt.want)
			}
		})
	}
}

func TestGitManager_WorkspaceStateValidation(t *testing.T) {
	testDir := setupTestRepo(t)

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{}, "")

	t.Run("clean workspace", func(t *testing.T) {
		if err := gm.CheckWorkspaceClean(ctx); err != nil {
			t.Errorf("CheckWorkspaceClean() failed on clean workspace: %v", err)
		}
	})

	t.Run("branch workflow", func(t *testing.T) {
		// Create and checkout new branch
		if err := gm.CreateBranch(ctx, "feature/test"); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		// Validation should pass on new branch
		if err := gm.CheckWorkspaceClean(ctx); err != nil {
			t.Errorf("CheckWorkspaceClean() failed on new branch: %v", err)
		}

		// Create a file
		testFile := filepath.Join(testDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		// Validation should fail with uncommitted changes
		if err := gm.CheckWorkspaceClean(ctx); err == nil {
			t.Error("CheckWorkspaceClean() should fail with uncommitted changes")
		}

		// Commit the changes
		config := GitConfig{
			AuthorName:  "Forge Bot",
			AuthorEmail: "forge@example.com",
		}
		gm = NewGitManager(testDir, config, "")
		if err := gm.Commit(ctx, "Add test file"); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Validation should pass again
		if err := gm.CheckWorkspaceClean(ctx); err != nil {
			t.Errorf("CheckWorkspaceClean() failed after commit: %v", err)
		}
	})
}

func TestGitManager_DetachedHeadDetection(t *testing.T) {
	testDir := setupTestRepo(t)

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{}, "")

	// Get the commit hash
	output, err := gm.execGit(ctx, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("failed to get commit hash: %v", err)
	}
	commitHash := strings.TrimSpace(output)

	// Checkout the commit directly (creates detached HEAD)
	if checkoutErr := execCommand(testDir, "git", "checkout", commitHash); checkoutErr != nil {
		t.Fatalf("failed to checkout commit: %v", checkoutErr)
	}

	// Get current branch should return empty in detached HEAD
	branch, err := gm.GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	if branch != "" {
		t.Errorf("expected empty branch in detached HEAD, got: %s", branch)
	}
}
