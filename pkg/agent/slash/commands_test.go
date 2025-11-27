package slash

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/entrhq/forge/pkg/agent/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock LLM client for testing
type mockLLMClient struct {
	response string
	err      error
}

func (m *mockLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

// TestParse tests the command parsing logic
func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantCommand *Command
		wantOK      bool
	}{
		{
			name:  "commit without message",
			input: "/commit",
			wantCommand: &Command{
				Name: "commit",
				Arg:  "",
			},
			wantOK: true,
		},
		{
			name:  "commit with message",
			input: "/commit fix: resolve bug",
			wantCommand: &Command{
				Name: "commit",
				Arg:  "fix: resolve bug",
			},
			wantOK: true,
		},
		{
			name:  "pr without title",
			input: "/pr",
			wantCommand: &Command{
				Name: "pr",
				Arg:  "",
			},
			wantOK: true,
		},
		{
			name:  "pr with title",
			input: "/pr Add new feature",
			wantCommand: &Command{
				Name: "pr",
				Arg:  "Add new feature",
			},
			wantOK: true,
		},
		{
			name:  "with leading whitespace",
			input: "  /commit test",
			wantCommand: &Command{
				Name: "commit",
				Arg:  "test",
			},
			wantOK: true,
		},
		{
			name:  "with trailing whitespace",
			input: "/commit test  ",
			wantCommand: &Command{
				Name: "commit",
				Arg:  "test",
			},
			wantOK: true,
		},
		{
			name:  "with multiple spaces in argument",
			input: "/commit fix:  multiple   spaces",
			wantCommand: &Command{
				Name: "commit",
				Arg:  "fix:  multiple   spaces",
			},
			wantOK: true,
		},
		{
			name:        "not a slash command",
			input:       "regular message",
			wantCommand: nil,
			wantOK:      false,
		},
		{
			name:        "empty string",
			input:       "",
			wantCommand: nil,
			wantOK:      false,
		},
		{
			name:        "just slash",
			input:       "/",
			wantCommand: &Command{Name: "", Arg: ""},
			wantOK:      true,
		},
		{
			name:        "slash in middle",
			input:       "not /commit",
			wantCommand: nil,
			wantOK:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := Parse(tt.input)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				require.NotNil(t, cmd)
				assert.Equal(t, tt.wantCommand.Name, cmd.Name)
				assert.Equal(t, tt.wantCommand.Arg, cmd.Arg)
			} else {
				assert.Nil(t, cmd)
			}
		})
	}
}

// TestShouldIntercept tests the input filtering logic
func TestShouldIntercept(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "slash command",
			input: "/commit",
			want:  true,
		},
		{
			name:  "slash command with args",
			input: "/pr Add feature",
			want:  true,
		},
		{
			name:  "regular message",
			input: "Please update the code",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "slash in middle",
			input: "not /command",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldIntercept(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestExecute_UnknownCommand tests error handling for unknown commands
func TestExecute_UnknownCommand(t *testing.T) {
	handler := NewHandler("/tmp", nil, nil, nil)
	cmd := &Command{Name: "unknown"}

	result, err := handler.Execute(context.Background(), cmd)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
	assert.Empty(t, result)
}

// TestHandleCommit tests commit command execution
func TestHandleCommit(t *testing.T) {
	// Create a temporary git repository for testing
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	tests := []struct {
		name          string
		customMessage string
		setupRepo     func(t *testing.T)
		llmResponse   string
		llmErr        error
		wantErr       bool
		wantContains  string
	}{
		{
			name:          "commit with custom message",
			customMessage: "fix: custom commit message",
			setupRepo: func(t *testing.T) {
				createTestFile(t, tmpDir, "test.txt", "content")
			},
			llmResponse:  "generated message", // Should not be used
			wantErr:      false,
			wantContains: "fix: custom commit message",
		},
		{
			name:          "commit with generated message",
			customMessage: "",
			setupRepo: func(t *testing.T) {
				createTestFile(t, tmpDir, "test2.txt", "content")
			},
			llmResponse:  "feat: generated commit",
			wantErr:      false,
			wantContains: "feat: generated commit",
		},
		{
			name:          "no files to commit",
			customMessage: "",
			setupRepo:     func(t *testing.T) {}, // No changes
			llmResponse:   "",
			wantErr:       true,
			wantContains:  "no files to commit",
		},
		{
			name:          "generator error",
			customMessage: "",
			setupRepo: func(t *testing.T) {
				createTestFile(t, tmpDir, "test3.txt", "content")
			},
			llmErr:       errors.New("generator failed"),
			wantErr:      true,
			wantContains: "generator failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup repository state
			tt.setupRepo(t)

			// Create mock LLM client and generators
			mockLLM := &mockLLMClient{
				response: tt.llmResponse,
				err:      tt.llmErr,
			}
			commitGen := git.NewCommitMessageGenerator(mockLLM)

			handler := NewHandler(tmpDir, git.NewModificationTracker(), commitGen, nil)
			result, err := handler.handleCommit(context.Background(), tt.customMessage)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantContains != "" {
					assert.Contains(t, err.Error(), tt.wantContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, result, tt.wantContains)
				assert.Contains(t, result, "Commit")
			}
		})
	}
}

// TestNewHandler tests handler construction
func TestNewHandler(t *testing.T) {
	workingDir := "/test/dir"
	tracker := git.NewModificationTracker()
	mockLLM := &mockLLMClient{}
	commitGen := git.NewCommitMessageGenerator(mockLLM)
	prGen := git.NewPRGenerator(mockLLM)

	handler := NewHandler(workingDir, tracker, commitGen, prGen)

	assert.NotNil(t, handler)
	assert.Equal(t, workingDir, handler.workingDir)
	assert.Equal(t, tracker, handler.tracker)
	assert.Equal(t, commitGen, handler.commitGenerator)
	assert.Equal(t, prGen, handler.prGenerator)
}

// TestHandlePR tests PR command execution
func TestHandlePR(t *testing.T) {
	tests := []struct {
		name         string
		customTitle  string
		setupRepo    func(t *testing.T, tmpDir string)
		llmResponse  string
		llmErr       error
		wantErr      bool
		wantContains string
	}{
		{
			name:        "no commits for PR",
			customTitle: "",
			setupRepo: func(t *testing.T, tmpDir string) {
				// Create a branch but with no new commits
				err := runGitCommand(tmpDir, "checkout", "-b", "empty-branch")
				require.NoError(t, err)
			},
			llmResponse:  "",
			wantErr:      true,
			wantContains: "no commits for PR",
		},
		{
			name:        "llm generation error",
			customTitle: "",
			setupRepo: func(t *testing.T, tmpDir string) {
				// Create a feature branch with commits
				err := runGitCommand(tmpDir, "checkout", "-b", "feature-branch")
				require.NoError(t, err)
				createTestFile(t, tmpDir, "feature.txt", "new feature")
				err = runGitCommand(tmpDir, "add", "feature.txt")
				require.NoError(t, err)
				err = runGitCommand(tmpDir, "commit", "-m", "feat: add new feature")
				require.NoError(t, err)
			},
			llmErr:       errors.New("LLM service unavailable"),
			wantErr:      true,
			wantContains: "LLM service unavailable",
		},
		{
			name:        "successful PR generation (fails at GitHub push)",
			customTitle: "Custom PR Title",
			setupRepo: func(t *testing.T, tmpDir string) {
				// Create a feature branch with commits
				err := runGitCommand(tmpDir, "checkout", "-b", "test-feature")
				require.NoError(t, err)
				createTestFile(t, tmpDir, "test.txt", "test content")
				err = runGitCommand(tmpDir, "add", "test.txt")
				require.NoError(t, err)
				err = runGitCommand(tmpDir, "commit", "-m", "test: add test file")
				require.NoError(t, err)
			},
			llmResponse: `{
				"title": "Add test feature",
				"description": "## Summary\n\nAdded test feature\n\n## Changes\n\n- Added test.txt"
			}`,
			wantErr:      true,
			wantContains: "failed to create PR", // Will fail at GitHub push step
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary git repository for each test
			tmpDir := t.TempDir()
			initGitRepo(t, tmpDir)

			// Setup repository state
			tt.setupRepo(t, tmpDir)

			// Create mock LLM client and generators
			mockLLM := &mockLLMClient{
				response: tt.llmResponse,
				err:      tt.llmErr,
			}
			prGen := git.NewPRGenerator(mockLLM)

			handler := NewHandler(tmpDir, git.NewModificationTracker(), nil, prGen)
			result, err := handler.handlePR(context.Background(), tt.customTitle)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantContains != "" {
					assert.Contains(t, err.Error(), tt.wantContains)
				}
			} else {
				assert.NoError(t, err)
				if tt.wantContains != "" {
					assert.Contains(t, result, tt.wantContains)
				}
			}
		})
	}
}

// TestGetCurrentBranch tests the getCurrentBranch helper
func TestGetCurrentBranch(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	handler := NewHandler(tmpDir, nil, nil, nil)

	// Test on main/master branch
	branch, err := handler.getCurrentBranch()
	assert.NoError(t, err)
	// Could be "main" or "master" depending on git version
	assert.True(t, branch == "main" || branch == "master" || branch == "HEAD", "expected main, master, or HEAD, got %s", branch)

	// Test on feature branch
	err = runGitCommand(tmpDir, "checkout", "-b", "test-branch")
	require.NoError(t, err)

	branch, err = handler.getCurrentBranch()
	assert.NoError(t, err)
	assert.Equal(t, "test-branch", branch)
}

// TestExecute_CommitCommand tests the commit command path
func TestExecute_CommitCommand(t *testing.T) {
	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create a file to commit
	createTestFile(t, tmpDir, "test.txt", "content")

	mockLLM := &mockLLMClient{response: "test: add test file"}
	commitGen := git.NewCommitMessageGenerator(mockLLM)
	handler := NewHandler(tmpDir, git.NewModificationTracker(), commitGen, nil)

	cmd := &Command{Name: "commit", Arg: "fix: custom message"}
	result, err := handler.Execute(context.Background(), cmd)

	assert.NoError(t, err)
	assert.Contains(t, result, "Commit")
	assert.Contains(t, result, "fix: custom message")
}

// TestExecute_PRCommand tests the PR command path
func TestExecute_PRCommand(t *testing.T) {
	t.Skip("requires gh CLI setup and GitHub authentication")

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Create a feature branch with commits
	err := runGitCommand(tmpDir, "checkout", "-b", "feature")
	require.NoError(t, err)
	createTestFile(t, tmpDir, "feature.txt", "content")
	err = runGitCommand(tmpDir, "add", "feature.txt")
	require.NoError(t, err)
	err = runGitCommand(tmpDir, "commit", "-m", "feat: add feature")
	require.NoError(t, err)

	mockLLM := &mockLLMClient{
		response: `{"title": "Add feature", "description": "Feature description"}`,
	}
	prGen := git.NewPRGenerator(mockLLM)
	handler := NewHandler(tmpDir, nil, nil, prGen)

	cmd := &Command{Name: "pr", Arg: ""}
	result, err := handler.Execute(context.Background(), cmd)

	// This will fail without gh CLI, but tests the execution path
	assert.Error(t, err)
	assert.Empty(t, result)
}

// Helper functions

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	
	// Initialize git repo
	err := runGitCommand(dir, "init")
	require.NoError(t, err, "failed to init git repo")
	
	// Configure git user
	err = runGitCommand(dir, "config", "user.email", "test@example.com")
	require.NoError(t, err, "failed to configure git user email")
	
	err = runGitCommand(dir, "config", "user.name", "Test User")
	require.NoError(t, err, "failed to configure git user name")
	
	// Create initial commit
	readmePath := filepath.Join(dir, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test Repo"), 0644)
	require.NoError(t, err, "failed to create README")
	
	err = runGitCommand(dir, "add", "README.md")
	require.NoError(t, err, "failed to add README")
	
	err = runGitCommand(dir, "commit", "-m", "Initial commit")
	require.NoError(t, err, "failed to create initial commit")
}

func createTestFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	path := filepath.Join(dir, filename)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err, "failed to create test file")
}

func runGitCommand(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Run()
}
