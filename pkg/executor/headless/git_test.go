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

	// Create two files
	testFile1 := filepath.Join(testDir, "test1.txt")
	if err := os.WriteFile(testFile1, []byte("content 1"), 0644); err != nil {
		t.Fatalf("failed to write test file 1: %v", err)
	}

	testFile2 := filepath.Join(testDir, "test2.txt")
	if err := os.WriteFile(testFile2, []byte("content 2"), 0644); err != nil {
		t.Fatalf("failed to write test file 2: %v", err)
	}

	// Get changed files
	files, err := gm.GetChangedFiles(ctx)
	if err != nil {
		t.Fatalf("failed to get changed files: %v", err)
	}

	// Should have both files
	if len(files) != 2 {
		t.Errorf("expected 2 changed files, got %d", len(files))
	}
}

func TestGitManager_hasChangesToCommit(t *testing.T) {
	testDir := setupTestRepo(t)

	config := GitConfig{
		AutoCommit:  true,
		AuthorName:  "Forge Bot",
		AuthorEmail: "forge@example.com",
	}
	gm := NewGitManager(testDir, config, "")

	ctx := context.Background()

	// No changes initially
	hasChanges, err := gm.hasChangesToCommit(ctx)
	if err != nil {
		t.Fatalf("failed to check for changes: %v", err)
	}
	if hasChanges {
		t.Error("should have no changes initially")
	}

	// Create and stage a file
	testFile := filepath.Join(testDir, "test.txt")
	if writeErr := os.WriteFile(testFile, []byte("test content"), 0644); writeErr != nil {
		t.Fatalf("failed to write test file: %v", writeErr)
	}

	if _, addErr := gm.execGit(ctx, "add", "test.txt"); addErr != nil {
		t.Fatalf("failed to stage file: %v", addErr)
	}

	// Should have changes now
	hasChanges, err = gm.hasChangesToCommit(ctx)
	if err != nil {
		t.Fatalf("failed to check for changes: %v", err)
	}
	if !hasChanges {
		t.Error("should have changes after staging file")
	}
}

func TestGitManager_CommitWithNoChanges(t *testing.T) {
	testDir := setupTestRepo(t)

	// Create a config file (which will be excluded)
	configFile := filepath.Join(testDir, "forge-config.yaml")
	if err := os.WriteFile(configFile, []byte("task: test\nmode: write\n"), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config := GitConfig{
		AutoCommit:  true,
		AuthorName:  "Forge Bot",
		AuthorEmail: "forge@example.com",
	}
	gm := NewGitManager(testDir, config, configFile)

	ctx := context.Background()

	// Try to commit - should succeed but create no commit since only config file exists
	if err := gm.Commit(ctx, "Test commit"); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Verify no new commit was created by checking log count
	output, err := gm.execGit(ctx, "rev-list", "--count", "HEAD")
	if err != nil {
		t.Fatalf("failed to count commits: %v", err)
	}

	// Should still have only 1 commit (the initial one)
	if strings.TrimSpace(output) != "1" {
		t.Errorf("expected 1 commit, got %s", strings.TrimSpace(output))
	}
}
