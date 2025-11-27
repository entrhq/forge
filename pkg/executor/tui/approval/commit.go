package approval

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/agent/slash"
	"github.com/entrhq/forge/pkg/executor/tui/syntax"
	"github.com/entrhq/forge/pkg/executor/tui/types"
)

const commitPreviewTitle = "Commit Preview"

// CommitRequest is a concrete implementation of ApprovalRequest for git commits.
// It encapsulates all data needed to preview and execute a commit operation.
type CommitRequest struct {
	files        []string
	message      string
	diff         string
	args         string
	slashHandler *slash.Handler
}

// NewCommitRequest creates a new commit approval request
func NewCommitRequest(files []string, message, diff, args string, slashHandler *slash.Handler) *CommitRequest {
	return &CommitRequest{
		files:        files,
		message:      message,
		diff:         diff,
		args:         args,
		slashHandler: slashHandler,
	}
}

// Title returns the approval dialog title
func (c *CommitRequest) Title() string {
	return commitPreviewTitle
}

// Content returns the formatted content for the commit preview
func (c *CommitRequest) Content() string {
	var b strings.Builder

	// Show files to commit
	if len(c.files) > 0 {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Files to commit:"))
		b.WriteString("\n")
		for _, file := range c.files {
			b.WriteString("  • " + file + "\n")
		}
		b.WriteString("\n")
	}

	// Show commit message
	if c.message != "" {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Commit Message:"))
		b.WriteString("\n")
		b.WriteString(c.message)
		b.WriteString("\n\n")
	}

	// Show diff with syntax highlighting
	if c.diff != "" {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Changes:"))
		b.WriteString("\n")

		highlightedDiff, err := syntax.HighlightDiff(c.diff, "")
		if err != nil {
			b.WriteString(c.diff)
		} else {
			b.WriteString(highlightedDiff)
		}
	}

	return b.String()
}

// OnApprove returns the command to execute when the user approves the commit
func (c *CommitRequest) OnApprove() tea.Cmd {
	return tea.Batch(
		// First, signal that we're starting commit creation
		func() tea.Msg {
			return types.OperationStartMsg{
				Message: "Creating commit...",
			}
		},
		// Then execute the commit
		func() tea.Msg {
			ctx := context.Background()
			result, err := c.slashHandler.Execute(ctx, &slash.Command{
				Name: "commit",
				Arg:  c.args,
			})
			return types.OperationCompleteMsg{
				Result:       result,
				Err:          err,
				SuccessTitle: "Success",
				SuccessIcon:  "✅",
				ErrorTitle:   "Commit Failed",
				ErrorIcon:    "❌",
			}
		},
	)
}

// OnReject returns the command to execute when the user rejects the commit
func (c *CommitRequest) OnReject() tea.Cmd {
	return func() tea.Msg {
		return types.ToastMsg{
			Message: "Canceled",
			Details: "/commit command canceled",
			Icon:    "ℹ️",
			IsError: false,
		}
	}
}
