package headless

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GitManager handles git operations for headless mode
type GitManager struct {
	workspaceDir string
	config       GitConfig
}

// NewGitManager creates a new git manager
func NewGitManager(workspaceDir string, config GitConfig) *GitManager {
	return &GitManager{
		workspaceDir: workspaceDir,
		config:       config,
	}
}

// CheckWorkspaceClean checks if the git workspace is clean
func (g *GitManager) CheckWorkspaceClean(ctx context.Context) error {
	output, err := g.execGit(ctx, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if strings.TrimSpace(output) != "" {
		return fmt.Errorf("workspace has uncommitted changes")
	}

	return nil
}

// GetCurrentBranch returns the current git branch
func (g *GitManager) GetCurrentBranch(ctx context.Context) (string, error) {
	output, err := g.execGit(ctx, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(output), nil
}

// CreateBranch creates a new git branch and switches to it
// If the branch already exists, it just switches to it
func (g *GitManager) CreateBranch(ctx context.Context, branchName string) error {
	// Check if branch exists first
	_, err := g.execGit(ctx, "show-ref", "--verify", "--quiet", fmt.Sprintf("refs/heads/%s", branchName))
	if err == nil {
		// Branch exists, just checkout
		_, err = g.execGit(ctx, "checkout", branchName)
		if err != nil {
			return fmt.Errorf("failed to checkout existing branch '%s': %w", branchName, err)
		}
		return nil
	}

	// Branch doesn't exist, create it
	_, err = g.execGit(ctx, "checkout", "-b", branchName)
	if err != nil {
		return fmt.Errorf("failed to create branch '%s': %w", branchName, err)
	}

	return nil
}

// Commit creates a git commit with the configured author
func (g *GitManager) Commit(ctx context.Context, message string) error {
	// Stage all changes
	_, err := g.execGit(ctx, "add", "-A")
	if err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Create commit with configured author
	args := []string{
		"commit",
		"-m", message,
	}

	if g.config.AuthorName != "" && g.config.AuthorEmail != "" {
		args = append(args,
			"--author", fmt.Sprintf("%s <%s>", g.config.AuthorName, g.config.AuthorEmail),
		)
	}

	_, err = g.execGit(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	// Auto-push if configured
	if g.config.AutoPush {
		if err := g.Push(ctx); err != nil {
			return fmt.Errorf("failed to auto-push: %w", err)
		}
	}

	return nil
}

// GetChangedFiles returns a list of files that have been modified or are untracked
func (g *GitManager) GetChangedFiles(ctx context.Context) ([]string, error) {
	// Use git status --porcelain to get both modified and untracked files
	output, err := g.execGit(ctx, "status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Parse git status format: "XY filename"
		// X shows the status of the index, Y shows the status of the work tree
		// We want files with any status (modified, added, untracked, etc.)
		if len(line) > 3 {
			filename := strings.TrimSpace(line[3:])
			files = append(files, filename)
		}
	}

	return files, nil
}

// GetDiffStat returns git diff statistics
func (g *GitManager) GetDiffStat(ctx context.Context) (string, error) {
	output, err := g.execGit(ctx, "diff", "--stat", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get diff stat: %w", err)
	}

	return output, nil
}

// Push pushes the current branch to the remote
func (g *GitManager) Push(ctx context.Context) error {
	// Get current branch
	branch, err := g.GetCurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Push to remote
	_, err = g.execGit(ctx, "push", "origin", branch)
	if err != nil {
		return fmt.Errorf("failed to push branch '%s': %w", branch, err)
	}

	return nil
}

// Rollback rolls back all uncommitted changes
func (g *GitManager) Rollback(ctx context.Context) error {
	// Reset all changes
	_, err := g.execGit(ctx, "reset", "--hard", "HEAD")
	if err != nil {
		return fmt.Errorf("failed to rollback changes: %w", err)
	}

	// Clean untracked files
	_, err = g.execGit(ctx, "clean", "-fd")
	if err != nil {
		return fmt.Errorf("failed to clean untracked files: %w", err)
	}

	return nil
}

// execGit executes a git command and returns its output
func (g *GitManager) execGit(ctx context.Context, args ...string) (string, error) {
	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(execCtx, "git", args...)
	cmd.Dir = g.workspaceDir

	// Execute and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// GenerateCommitMessage generates a commit message based on changes
func (g *GitManager) GenerateCommitMessage(ctx context.Context, taskDescription string) string {
	if g.config.CommitMessage != "" {
		return g.config.CommitMessage
	}

	// Default format
	return fmt.Sprintf("chore: %s\n\nAutomated changes via Forge headless mode", taskDescription)
}

// GenerateBranchName generates a branch name for the headless run
func GenerateBranchName(prefix string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s", prefix, timestamp)
}
