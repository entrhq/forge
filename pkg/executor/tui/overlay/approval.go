package overlay

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/executor/tui/approval"
	"github.com/entrhq/forge/pkg/executor/tui/types"
)

// Key binding constants for approval overlay
const (
	keyCtrlA = "ctrl+a"
	keyCtrlC = "ctrl+c"
	keyCtrlR = "ctrl+r"
	keyTab   = "tab"
	keyEnter = "enter"
	keyLeft  = "left"
	keyRight = "right"
	keyEsc   = "esc"
)

// GenericApprovalOverlay displays an approval request for any command.
// It is completely agnostic of the specific command being approved.
type GenericApprovalOverlay struct {
	*ApprovalOverlayBase
	request approval.ApprovalRequest
}

// NewGenericApprovalOverlay creates a new generic approval overlay
func NewGenericApprovalOverlay(request approval.ApprovalRequest, width, height int) *GenericApprovalOverlay {
	// Make overlay wide - 90% of screen width
	overlayWidth := max(int(float64(width)*0.9), 80)

	// Fixed viewport height for content
	const maxViewportHeight = 15
	viewportHeight := maxViewportHeight

	// Calculate total overlay height
	// Title (2) + subtitle (1) + spacing (1) + border (2) + buttons (2) + hints (1) = 9 lines
	// Plus viewport height
	overlayHeight := viewportHeight + 9

	overlay := &GenericApprovalOverlay{
		request: request,
	}

	// Configure approval overlay
	approvalConfig := ApprovalOverlayConfig{
		BaseConfig: BaseOverlayConfig{
			Width:                 overlayWidth,
			Height:                overlayHeight,
			ViewportWidth:         overlayWidth - 4,
			ViewportHeight:        viewportHeight,
			Content:               request.Content(),
			RenderHeader:          overlay.renderHeader,
			RenderFooter:          overlay.renderFooter,
			FooterRendersViewport: true, // Footer renders viewport with custom styling
		},
		OnApprove:    request.OnApprove,
		OnReject:     request.OnReject,
		ApproveLabel: " ✓ Accept ",
		RejectLabel:  " ✗ Reject ",
		ShowHints:    true,
	}

	overlay.ApprovalOverlayBase = NewApprovalOverlayBase(approvalConfig)
	return overlay
}

// Update handles messages for the approval overlay
func (a *GenericApprovalOverlay) Update(msg tea.Msg, state types.StateProvider, actions types.ActionHandler) (types.Overlay, tea.Cmd) {
	// Check if this is an approval/rejection key before delegating to base
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case keyCtrlA:
			// Ctrl+A always approves, return nil to close overlay immediately
			return nil, a.request.OnApprove()
		case keyCtrlR, keyCtrlC, keyEsc:
			// These always reject, return nil to close overlay immediately
			return nil, a.request.OnReject()
		case keyEnter:
			// Enter submits the currently selected choice
			if a.Selected() == ApprovalChoiceAccept {
				return nil, a.request.OnApprove()
			}
			return nil, a.request.OnReject()
		}
	}

	// Let base handle other keys (tab, arrows, scrolling, etc.)
	updatedApproval, cmd := a.ApprovalOverlayBase.Update(msg, state, actions)
	a.ApprovalOverlayBase = updatedApproval
	return a, cmd
}

// renderHeader renders the approval overlay header
func (a *GenericApprovalOverlay) renderHeader() string {
	return types.OverlayTitleStyle.Render(a.request.Title())
}

// renderFooter renders the approval overlay footer with buttons and hints
func (a *GenericApprovalOverlay) renderFooter() string {
	contentWidth := a.Width() - 6

	var footer strings.Builder

	// Wrap viewport content in a bordered box
	contentStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(types.SalmonPink).
		Padding(0, 1).
		Width(contentWidth - 4)

	footer.WriteString(contentStyle.Render(a.Viewport().View()))
	footer.WriteString("\n\n")

	// Render buttons
	buttonsRow := a.RenderButtons()
	buttonsLen := lipgloss.Width(buttonsRow)
	buttonsPadding := max(0, (contentWidth-buttonsLen)/2)
	footer.WriteString(strings.Repeat(" ", buttonsPadding) + buttonsRow)
	footer.WriteString("\n")

	// Render hints
	hints := types.OverlayHelpStyle.Render("Ctrl+A: Accept • Ctrl+R: Reject • Tab: Toggle • ↑/↓: Scroll")
	hintsLen := lipgloss.Width(hints)
	hintsPadding := max(0, (contentWidth-hintsLen)/2)
	footer.WriteString(strings.Repeat(" ", hintsPadding) + hints)

	return footer.String()
}

// View renders the approval overlay
func (a *GenericApprovalOverlay) View() string {
	// Delegate to base overlay's View method which handles the rendering
	// The base already calls renderHeader, renderFooter, and wraps in container
	return a.BaseOverlay.View(a.Width())
}
