package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type CommitInfo struct {
	Hash    string
	Message string
}

type CommitMessageGenerator struct {
	llmClient LLMClient
}

type LLMClient interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

func NewCommitMessageGenerator(llmClient LLMClient) *CommitMessageGenerator {
	return &CommitMessageGenerator{
		llmClient: llmClient,
	}
}

func (g *CommitMessageGenerator) Generate(ctx context.Context, workingDir string, files []string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files to commit")
	}

	diff, err := getDiff(workingDir, files)
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	prompt := buildCommitPrompt(diff, files)
	message, err := g.llmClient.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate commit message: %w", err)
	}

	return strings.TrimSpace(message), nil
}

func getDiff(workingDir string, files []string) (string, error) {
	// Try git diff HEAD first (for modified tracked files)
	args := append([]string{"diff", "HEAD", "--"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// If that fails, try without HEAD (unstaged changes)
		stdout.Reset()
		stderr.Reset()

		args = append([]string{"diff", "--"}, files...)
		cmd = exec.Command("git", args...)
		cmd.Dir = workingDir
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			// If both fail, try --cached in case files were already staged
			stdout.Reset()
			stderr.Reset()

			args = append([]string{"diff", "--cached", "--"}, files...)
			cmd = exec.Command("git", args...)
			cmd.Dir = workingDir
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("git diff failed: %w, stderr: %s", err, stderr.String())
			}
		}
	}

	return stdout.String(), nil
}

func buildCommitPrompt(diff string, files []string) string {
	var sb strings.Builder

	sb.WriteString("Generate a conventional commit message for these changes.\\n\\n")
	sb.WriteString("Format: <type>(<scope>): <description>\\n")
	sb.WriteString("Types: feat, fix, docs, style, refactor, test, chore\\n\\n")

	sb.WriteString("Files changed:\\n")
	for _, file := range files {
		sb.WriteString(fmt.Sprintf("- %s\\n", file))
	}

	sb.WriteString("\\nDiff:\\n")
	sb.WriteString(truncateDiff(diff, 3000))

	sb.WriteString("\\n\\nGenerate ONLY the commit message (one line), nothing else.")

	return sb.String()
}

func truncateDiff(diff string, maxChars int) string {
	if len(diff) <= maxChars {
		return diff
	}
	return diff[:maxChars] + "\\n... (diff truncated)"
}

// GetModifiedFiles returns a list of modified files from git status
func GetModifiedFiles(workingDir string) ([]string, error) {
	// Get all modified, new, and deleted files
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git status failed: %w, stderr: %s", err, stderr.String())
	}

	var files []string
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// git status --porcelain format: XY filename
		// Where X and Y are status codes (2 chars total)
		// There's a space after the status, so filename starts at position 3
		if len(line) > 2 {
			// Check if file is deleted (status code D in either position)
			// First char is index status, second char is worktree status
			status := line[:2]
			if status[0] == 'D' || status[1] == 'D' {
				// Skip deleted files - they can't be staged with git add
				continue
			}

			// Split on whitespace and take the last part (handles renamed files too)
			parts := strings.Fields(line)
			if len(parts) > 0 {
				// For renamed files, format is "R old -> new", we want the new name
				filename := parts[len(parts)-1]
				files = append(files, filename)
			}
		}
	}

	return files, nil
}

func StageFiles(workingDir string, files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files to stage")
	}

	args := append([]string{"add"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = workingDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func CreateCommit(workingDir, message string) (string, error) {
	// Get current user's git config for co-author
	userNameCmd := exec.Command("git", "config", "user.name")
	userNameCmd.Dir = workingDir
	userNameOutput, err := userNameCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git user.name: %w", err)
	}
	currentUserName := strings.TrimSpace(string(userNameOutput))

	userEmailCmd := exec.Command("git", "config", "user.email")
	userEmailCmd.Dir = workingDir
	userEmailOutput, err := userEmailCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git user.email: %w", err)
	}
	currentUserEmail := strings.TrimSpace(string(userEmailOutput))

	// Set commit author to anvxl
	configNameCmd := exec.Command("git", "config", "user.name", "anvxl")
	configNameCmd.Dir = workingDir
	if err := configNameCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to set git user.name: %w", err)
	}

	configEmailCmd := exec.Command("git", "config", "user.email", "anvxl@entr.net.au")
	configEmailCmd.Dir = workingDir
	if err := configEmailCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to set git user.email: %w", err)
	}

	// Add co-author trailer to commit message
	messageWithCoAuthor := message + "\n\nCo-authored-by: " + currentUserName + " <" + currentUserEmail + ">"

	cmd := exec.Command("git", "commit", "-m", messageWithCoAuthor)
	cmd.Dir = workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Restore original git config on error
		exec.Command("git", "config", "user.name", currentUserName).Run()
		exec.Command("git", "config", "user.email", currentUserEmail).Run()
		return "", fmt.Errorf("git commit failed: %w, stderr: %s", err, stderr.String())
	}

	// Restore original git config after commit
	restoreNameCmd := exec.Command("git", "config", "user.name", currentUserName)
	restoreNameCmd.Dir = workingDir
	restoreNameCmd.Run()

	restoreEmailCmd := exec.Command("git", "config", "user.email", currentUserEmail)
	restoreEmailCmd.Dir = workingDir
	restoreEmailCmd.Run()

	hash, err := getLatestCommitHash(workingDir)
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}

	return hash, nil
}

func getLatestCommitHash(workingDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w, stderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
