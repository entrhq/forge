package headless

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupGitRepo creates a temporary git repository for testing
func setupGitRepo(t *testing.T) string {
	t.Helper()

	testDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = testDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = testDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = testDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git name: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(testDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = testDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to stage files: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = testDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	return testDir
}

// TestGitManager_BranchCreationWorkflow tests the branch creation workflow
func TestGitManager_BranchCreationWorkflow(t *testing.T) {
	testDir := setupGitRepo(t)
	ctx := context.Background()

	config := GitConfig{
		AutoCommit: true,
		Branch:     "feature/test-branch",
	}

	gm := NewGitManager(testDir, config)

	// Get initial branch
	initialBranch, err := gm.GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("failed to get initial branch: %v", err)
	}
	t.Logf("Initial branch: %s", initialBranch)

	// Verify we're not on the target branch
	if initialBranch == config.Branch {
		t.Fatalf("test setup error: already on target branch")
	}

	// Create and checkout the branch
	err = gm.CreateBranch(ctx, config.Branch)
	if err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Check that we're now on the configured branch
	currentBranch, err := gm.GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}

	if currentBranch != config.Branch {
		t.Errorf("expected branch %q, got %q", config.Branch, currentBranch)
	}
}

// TestGitManager_BranchAlreadyExists tests switching to an existing branch
func TestGitManager_BranchAlreadyExists(t *testing.T) {
	testDir := setupGitRepo(t)
	ctx := context.Background()

	config := GitConfig{
		AutoCommit: true,
		Branch:     "feature/existing-branch",
	}

	gm := NewGitManager(testDir, config)

	// Create the branch
	err := gm.CreateBranch(ctx, config.Branch)
	if err != nil {
		t.Fatalf("failed to create initial branch: %v", err)
	}

	// Switch back to main/master
	initialBranch := "main"
	cmd := exec.Command("git", "rev-parse", "--verify", "main")
	cmd.Dir = testDir
	if cmdErr := cmd.Run(); cmdErr != nil {
		initialBranch = "master"
	}

	cmd = exec.Command("git", "checkout", initialBranch)
	cmd.Dir = testDir
	if cmdErr := cmd.Run(); cmdErr != nil {
		t.Fatalf("failed to checkout main: %v", cmdErr)
	}

	// Verify we're on the initial branch
	currentBranch, err := gm.GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	if currentBranch != initialBranch {
		t.Fatalf("test setup error: not on initial branch")
	}

	// Call CreateBranch again - should switch to existing branch
	err = gm.CreateBranch(ctx, config.Branch)
	if err != nil {
		t.Fatalf("failed to switch to existing branch: %v", err)
	}

	// Check that we're on the configured branch
	currentBranch, err = gm.GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}

	if currentBranch != config.Branch {
		t.Errorf("expected branch %q, got %q", config.Branch, currentBranch)
	}
}
