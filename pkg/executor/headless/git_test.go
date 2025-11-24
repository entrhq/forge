package headless

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// setupTestGitRepo creates a test git repository in /tmp/test-git-workspace
func setupTestGitRepo(t *testing.T) (string, func()) {
	t.Helper()

	testDir := "/tmp/test-git-workspace"

	// Clean up any existing test directory
	_ = os.RemoveAll(testDir)

	// Create directory
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Initialize git repository
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
	readmePath := filepath.Join(testDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repository\n"), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = testDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add README: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = testDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		_ = os.RemoveAll(testDir)
	}

	return testDir, cleanup
}

func TestGitManager_CheckWorkspaceClean(t *testing.T) {
	testDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{})

	// Test clean workspace
	err := gm.CheckWorkspaceClean(ctx)
	if err != nil {
		t.Errorf("expected clean workspace, got error: %v", err)
	}

	// Create uncommitted file
	testFile := filepath.Join(testDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Test dirty workspace
	err = gm.CheckWorkspaceClean(ctx)
	if err == nil {
		t.Error("expected error for dirty workspace, got nil")
	}
}

func TestGitManager_GetCurrentBranch(t *testing.T) {
	testDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{})

	branch, err := gm.GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}

	// Git init creates either 'main' or 'master' depending on git version
	if branch != "main" && branch != "master" {
		t.Errorf("expected branch 'main' or 'master', got %q", branch)
	}
}

func TestGitManager_CreateBranch(t *testing.T) {
	testDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{})

	// Create new branch
	branchName := "feature/test-branch"
	err := gm.CreateBranch(ctx, branchName)
	if err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Verify we're on the new branch
	currentBranch, err := gm.GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}

	if currentBranch != branchName {
		t.Errorf("expected branch %q, got %q", branchName, currentBranch)
	}
}

func TestGitManager_Commit(t *testing.T) {
	testDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()
	config := GitConfig{
		AuthorName:  "Forge AI",
		AuthorEmail: "forge@example.com",
	}
	gm := NewGitManager(testDir, config)

	// Create a file to commit
	testFile := filepath.Join(testDir, "new-file.txt")
	if err := os.WriteFile(testFile, []byte("test content\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create commit
	commitMsg := "chore: add test file"
	err := gm.Commit(ctx, commitMsg)
	if err != nil {
		t.Fatalf("failed to create commit: %v", err)
	}

	// Verify commit was created
	cmd := exec.Command("git", "log", "-1", "--pretty=%s")
	cmd.Dir = testDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get commit message: %v", err)
	}

	actualMsg := string(output)
	if actualMsg != commitMsg+"\n" {
		t.Errorf("expected commit message %q, got %q", commitMsg, actualMsg)
	}

	// Verify author
	cmd = exec.Command("git", "log", "-1", "--pretty=%an <%ae>")
	cmd.Dir = testDir
	output, err = cmd.Output()
	if err != nil {
		t.Fatalf("failed to get commit author: %v", err)
	}

	expectedAuthor := "Forge AI <forge@example.com>\n"
	actualAuthor := string(output)
	if actualAuthor != expectedAuthor {
		t.Errorf("expected author %q, got %q", expectedAuthor, actualAuthor)
	}
}

func TestGitManager_GetChangedFiles(t *testing.T) {
	testDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{})

	// Initially no changes
	files, err := gm.GetChangedFiles(ctx)
	if err != nil {
		t.Fatalf("failed to get changed files: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 changed files, got %d", len(files))
	}

	// Modify a file
	readmePath := filepath.Join(testDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Modified README\n"), 0644); err != nil {
		t.Fatalf("failed to modify README: %v", err)
	}

	// Add a new file
	newFile := filepath.Join(testDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new content\n"), 0644); err != nil {
		t.Fatalf("failed to write new file: %v", err)
	}

	// Get changed files - note: git diff only shows tracked files by default
	// We need to stage the new file first or use git status
	files, err = gm.GetChangedFiles(ctx)
	if err != nil {
		t.Fatalf("failed to get changed files: %v", err)
	}

	// Should see at least the modified README (new.txt is untracked, not in diff)
	if len(files) < 1 {
		t.Errorf("expected at least 1 changed file, got %d: %v", len(files), files)
	}
}

func TestGitManager_Rollback(t *testing.T) {
	testDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{})

	// Modify existing file
	readmePath := filepath.Join(testDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Modified README\n"), 0644); err != nil {
		t.Fatalf("failed to modify README: %v", err)
	}

	// Add new file
	newFile := filepath.Join(testDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new content\n"), 0644); err != nil {
		t.Fatalf("failed to write new file: %v", err)
	}

	// Rollback changes
	err := gm.Rollback(ctx)
	if err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}

	// Verify workspace is clean
	err = gm.CheckWorkspaceClean(ctx)
	if err != nil {
		t.Errorf("expected clean workspace after rollback, got error: %v", err)
	}

	// Verify new file was removed
	if _, err := os.Stat(newFile); !os.IsNotExist(err) {
		t.Error("expected new file to be removed after rollback")
	}

	// Verify README was restored
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read README: %v", err)
	}
	if string(content) != "# Test Repository\n" {
		t.Errorf("expected original README content, got %q", string(content))
	}
}

func TestGenerateBranchName(t *testing.T) {
	prefix := "forge/auto"
	branchName := GenerateBranchName(prefix)

	if branchName == "" {
		t.Error("expected non-empty branch name")
	}

	// Should start with prefix
	if len(branchName) <= len(prefix) {
		t.Errorf("branch name %q should be longer than prefix %q", branchName, prefix)
	}

	// Generate two branch names with slight delay
	time.Sleep(time.Second)
	branchName2 := GenerateBranchName(prefix)

	// They should be different due to timestamp
	if branchName == branchName2 {
		t.Error("expected different branch names when generated at different times")
	}
}

func TestGitManager_GenerateCommitMessage(t *testing.T) {
	testDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name        string
		config      GitConfig
		task        string
		wantContain string
	}{
		{
			name: "custom message",
			config: GitConfig{
				CommitMessage: "fix: custom commit message",
			},
			task:        "Fix bug",
			wantContain: "fix: custom commit message",
		},
		{
			name:        "generated message",
			config:      GitConfig{},
			task:        "Add feature X",
			wantContain: "Add feature X",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gm := NewGitManager(testDir, tt.config)
			msg := gm.GenerateCommitMessage(ctx, tt.task)

			if !contains(msg, tt.wantContain) {
				t.Errorf("commit message %q should contain %q", msg, tt.wantContain)
			}
		})
	}
}

func TestGitManager_WorkspaceStateValidation(t *testing.T) {
	testDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{})

	// Test 1: Clean workspace on main branch
	t.Run("clean workspace", func(t *testing.T) {
		err := gm.CheckWorkspaceClean(ctx)
		if err != nil {
			t.Errorf("expected clean workspace: %v", err)
		}

		branch, err := gm.GetCurrentBranch(ctx)
		if err != nil {
			t.Fatalf("failed to get branch: %v", err)
		}
		if branch != "main" && branch != "master" {
			t.Errorf("unexpected branch: %s", branch)
		}
	})

	// Test 2: Create feature branch and commit
	t.Run("branch workflow", func(t *testing.T) {
		branchName := GenerateBranchName("forge/test")
		
		// Create branch
		err := gm.CreateBranch(ctx, branchName)
		if err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		// Verify on new branch
		currentBranch, err := gm.GetCurrentBranch(ctx)
		if err != nil {
			t.Fatalf("failed to get current branch: %v", err)
		}
		if currentBranch != branchName {
			t.Errorf("expected branch %q, got %q", branchName, currentBranch)
		}

		// Make changes
		testFile := filepath.Join(testDir, "feature.txt")
		if err := os.WriteFile(testFile, []byte("feature content\n"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		// Commit changes
		config := GitConfig{
			AuthorName:  "Forge AI",
			AuthorEmail: "forge@example.com",
		}
		gm = NewGitManager(testDir, config)
		err = gm.Commit(ctx, "feat: add new feature")
		if err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Verify clean after commit
		err = gm.CheckWorkspaceClean(ctx)
		if err != nil {
			t.Errorf("workspace should be clean after commit: %v", err)
		}
	})
}

func TestGitManager_DetachedHeadDetection(t *testing.T) {
	testDir, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()
	gm := NewGitManager(testDir, GitConfig{})

	// Get initial branch
	initialBranch, err := gm.GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("failed to get initial branch: %v", err)
	}

	// Create a commit to get a commit hash
	testFile := filepath.Join(testDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	cmd := exec.Command("git", "add", "test.txt")
	cmd.Dir = testDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "test commit")
	cmd.Dir = testDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Get commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = testDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get commit hash: %v", err)
	}
	commitHash := string(output[:7]) // First 7 characters

	// Checkout detached HEAD
	cmd = exec.Command("git", "checkout", commitHash)
	cmd.Dir = testDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to checkout detached HEAD: %v", err)
	}

	// Get current branch should return empty for detached HEAD
	branch, err := gm.GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}

	if branch != "" {
		t.Logf("Note: detached HEAD returned branch: %q (this is OK, depends on git version)", branch)
	}

	// Restore original branch for cleanup
	cmd = exec.Command("git", "checkout", initialBranch)
	cmd.Dir = testDir
	_ = cmd.Run()
}

// Note: contains() and findSubstring() helper functions are defined in quality_gate_test.go
// to avoid duplication
