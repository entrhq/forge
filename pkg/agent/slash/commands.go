// Package slash provides git operation handlers for slash commands.
// This package handles the execution of /commit and /pr commands after
// they have been approved by the user in the TUI.
//
// Note: The TUI (pkg/executor/tui/slash_commands.go) handles command parsing,
// validation, and user interaction. This package only executes the git operations.
package slash

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/entrhq/forge/pkg/agent/git"
)

type Command struct {
	Name string
	Arg  string
}

type Handler struct {
	workingDir      string
	tracker         *git.ModificationTracker
	commitGenerator *git.CommitMessageGenerator
	prGenerator     *git.PRGenerator
}

func NewHandler(
	workingDir string,
	tracker *git.ModificationTracker,
	commitGen *git.CommitMessageGenerator,
	prGen *git.PRGenerator,
) *Handler {
	return &Handler{
		workingDir:      workingDir,
		tracker:         tracker,
		commitGenerator: commitGen,
		prGenerator:     prGen,
	}
}

func Parse(input string) (*Command, bool) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return nil, false
	}

	parts := strings.SplitN(trimmed[1:], " ", 2)
	cmd := &Command{
		Name: parts[0],
	}

	if len(parts) > 1 {
		cmd.Arg = strings.TrimSpace(parts[1])
	}

	return cmd, true
}

func (h *Handler) Execute(ctx context.Context, cmd *Command) (string, error) {
	switch cmd.Name {
	case "commit":
		return h.handleCommit(ctx, cmd.Arg)
	case "pr":
		return h.handlePR(ctx, cmd.Arg)
	default:
		return "", fmt.Errorf("unknown command: /%s", cmd.Name)
	}
}

func (h *Handler) handleCommit(ctx context.Context, customMessage string) (string, error) {
	// Get modified files from git status instead of tracker
	files, err := git.GetModifiedFiles(h.workingDir)
	if err != nil {
		return "", fmt.Errorf("failed to get modified files: %w", err)
	}

	if len(files) == 0 {
		return "", fmt.Errorf("no files to commit")
	}

	if stageErr := git.StageFiles(h.workingDir, files); stageErr != nil {
		return "", stageErr
	}

	var message string

	if customMessage == "" {
		message, err = h.commitGenerator.Generate(ctx, h.workingDir, files)
		if err != nil {
			return "", err
		}
	} else {
		message = customMessage
	}

	hash, err := git.CreateCommit(h.workingDir, message)
	if err != nil {
		return "", err
	}

	// Clear tracker if it was being used
	if h.tracker != nil {
		h.tracker.Clear()
	}

	return fmt.Sprintf("Commit %s: %s", hash, message), nil
}

func (h *Handler) handlePR(ctx context.Context, customTitle string) (string, error) {
	base, err := git.DetectBaseBranch(h.workingDir)
	if err != nil {
		return "", err
	}

	head, err := h.getCurrentBranch()
	if err != nil {
		return "", err
	}

	commits, err := git.GetCommitsSinceBase(h.workingDir, base, head)
	if err != nil {
		return "", err
	}

	if len(commits) == 0 {
		return "", fmt.Errorf("no commits for PR")
	}

	diffSummary, err := git.GetDiffSummary(h.workingDir, base, head)
	if err != nil {
		return "", err
	}

	prContent, err := h.prGenerator.Generate(ctx, commits, diffSummary, base, head, customTitle)
	if err != nil {
		return "", err
	}

	// Create the PR on GitHub
	prURL, err := git.CreatePR(h.workingDir, prContent.Title, prContent.Description, base, head)
	if err != nil {
		return "", fmt.Errorf("failed to create PR: %w", err)
	}

	var result strings.Builder
	fmt.Fprintf(&result, "✅ PR Created: %s -> %s\n\n", head, base)
	fmt.Fprintf(&result, "Title: %s\n\n", prContent.Title)
	fmt.Fprintf(&result, "URL: %s\n\n", prURL)
	result.WriteString(prContent.Description)

	return result.String(), nil
}

func (h *Handler) getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = h.workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func ShouldIntercept(input string) bool {
	_, ok := Parse(input)
	return ok
}
