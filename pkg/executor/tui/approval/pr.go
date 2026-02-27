package approval

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/agent/slash"
	"github.com/entrhq/forge/pkg/executor/tui/types"
)

// PRRequest is a concrete implementation of ApprovalRequest for pull requests.
// It encapsulates all data needed to preview and execute a PR operation.
type PRRequest struct {
	branch       string
	prTitle      string
	prDesc       string
	changes      string
	args         string
	slashHandler *slash.Handler
}

// NewPRRequest creates a new PR approval request
func NewPRRequest(branch, prTitle, prDesc, changes, args string, slashHandler *slash.Handler) *PRRequest {
	return &PRRequest{
		branch:       branch,
		prTitle:      prTitle,
		prDesc:       prDesc,
		changes:      changes,
		args:         args,
		slashHandler: slashHandler,
	}
}

// Title returns the approval dialog title
func (p *PRRequest) Title() string {
	// Use the generated PR title if available
	if p.prTitle != "" && p.prTitle != p.args {
		return p.prTitle
	}
	// Fall back to user-provided title if any
	if p.args != "" {
		return p.args
	}
	return "Pull Request Preview"
}

// Content returns the formatted content for the PR preview
func (p *PRRequest) Content() string {
	var b strings.Builder

	// Show branch info at the top
	if p.branch != "" {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Branch:"))
		b.WriteString("\n")
		b.WriteString("  " + p.branch + "\n")
		b.WriteString("\n")
	}

	// Show PR title if it's different from the overlay title
	// (i.e., if we're showing user-provided title in header, show generated title here)
	if p.prTitle != "" && p.args != "" && p.prTitle != p.args {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Generated Title:"))
		b.WriteString("\n")
		b.WriteString(p.prTitle)
		b.WriteString("\n\n")
	}

	// Show PR description prominently if available
	if p.prDesc != "" {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Description:"))
		b.WriteString("\n")
		b.WriteString(p.prDesc)
		b.WriteString("\n\n")
	}

	// Show commits and changes
	if p.changes != "" {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Commits & Changes:"))
		b.WriteString("\n")
		b.WriteString(p.changes)
	}

	return b.String()
}

// OnApprove returns the command to execute when the user approves the PR
func (p *PRRequest) OnApprove() tea.Cmd {
	return tea.Batch(
		// First, signal that we're starting PR creation
		func() tea.Msg {
			return types.OperationStartMsg{
				Message: "Creating pull request on GitHub...",
			}
		},
		// Then execute the PR creation
		func() tea.Msg {
			ctx := context.Background()
			result, err := p.slashHandler.Execute(ctx, &slash.Command{
				Name: "pr",
				Arg:  p.args,
			})
			return types.OperationCompleteMsg{
				Result:       result,
				Err:          err,
				SuccessTitle: "Success",
				SuccessIcon:  "↑",
				ErrorTitle:   "PR Failed",
				ErrorIcon:    "✗",
			}
		},
	)
}

// OnReject returns the command to execute when the user rejects the PR
func (p *PRRequest) OnReject() tea.Cmd {
	return func() tea.Msg {
		return types.ToastMsg{
			Message: "Canceled",
			Details: "/pr command canceled",
			Icon:    "i",
			IsError: false,
		}
	}
}
