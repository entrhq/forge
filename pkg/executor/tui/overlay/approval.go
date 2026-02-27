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
	overlayWidth := types.ComputeOverlayWidth(width, 0.90, 60, 140)
	viewportHeight := types.ComputeViewportHeight(height, 8)
	overlayHeight := viewportHeight + 8

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
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		newOverlayWidth := types.ComputeOverlayWidth(sizeMsg.Width, 0.90, 60, 140)
		vpHeight := types.ComputeViewportHeight(sizeMsg.Height, 8)
		
		a.SetDimensions(newOverlayWidth, vpHeight+8)
		
		vp := a.BaseOverlay.Viewport()
		vp.Width = newOverlayWidth - 4
		vp.Height = vpHeight
		a.BaseOverlay.SetContent(vp.View())
	}

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
	contentWidth := a.Viewport().Width
	title := a.request.Title()

	titleLen := lipgloss.Width(title)
	titlePadding := max(0, (contentWidth-titleLen)/2)

	var header strings.Builder
	p1 := ""
	for i:=0; i<titlePadding; i++ { p1 += " " }
	header.WriteString(p1 + types.OverlayTitleStyle.Render(title))

	return header.String()
}

// renderFooter renders the approval overlay footer with buttons and hints
func (a *GenericApprovalOverlay) renderFooter() string {
	contentWidth := a.Viewport().Width

	var footer strings.Builder
	
	sepStr := ""
	for i:=0; i<contentWidth; i++ { sepStr += "─" }
	separator := lipgloss.NewStyle().Foreground(types.MutedGray).Render(sepStr)

	// Since we are nested inside the footer call, we prepend our own rendered diff
	footer.WriteString(a.Viewport().View())
	footer.WriteString("\n" + separator + "\n")

	// Render buttons
	buttonsRow := a.RenderButtons()
	buttonsLen := lipgloss.Width(buttonsRow)
	buttonsPadding := max(0, (contentWidth-buttonsLen)/2)
	pad1 := ""
	for i:=0; i<buttonsPadding; i++ { pad1 += " " }
	footer.WriteString(pad1 + buttonsRow)
	footer.WriteString("\n")

	// Render hints
	hints := types.OverlayHelpStyle.Render("Ctrl+A: Accept • Ctrl+R: Reject • Tab: Toggle • ↑/↓: Scroll")
	hintsLen := lipgloss.Width(hints)
	hintsPadding := max(0, (contentWidth-hintsLen)/2)
	pad2 := ""
	for i:=0; i<hintsPadding; i++ { pad2 += " " }
	footer.WriteString(pad2 + hints)

	return footer.String()
}

// View renders the approval overlay
func (a *GenericApprovalOverlay) View() string {
	// Delegate to base overlay's View method which handles the rendering
	// The base already calls renderHeader, renderFooter, and wraps in container
	return a.BaseOverlay.View(a.Width())
}
