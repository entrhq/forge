package overlay

import (
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrhq/forge/pkg/executor/tui/types"
)

// HelpOverlay displays help information in a modal dialog
type HelpOverlay struct {
	*BaseOverlay
	title string
}

// NewHelpOverlay creates a new help overlay
func NewHelpOverlay(title, content string, termWidth, termHeight int) *HelpOverlay {
	overlayWidth := types.ComputeOverlayWidth(termWidth, 0.80, 56, 100)
	
	// chromeRows = title+sep (2) + blank (1) + footer (1) + blank (1) = 5
	viewportHeight := types.ComputeViewportHeight(termHeight, 5)
	overlayHeight := viewportHeight + 5

	overlay := &HelpOverlay{
		title: title,
	}

	// Configure base overlay with custom key handler
	baseConfig := BaseOverlayConfig{
		Width:          overlayWidth,
		Height:         overlayHeight,
		ViewportWidth:  overlayWidth - 4,
		ViewportHeight: viewportHeight,
		Content:        content,
		OnClose: func(actions types.ActionHandler) tea.Cmd {
			// Return nil to signal close - caller will handle ClearOverlay()
			return nil
		},
		OnCustomKey: func(msg tea.KeyMsg, actions types.ActionHandler) (bool, tea.Cmd) {
			// Allow Enter to close help overlay
			if msg.Type == tea.KeyEnter {
				if overlay.BaseOverlay != nil {
					// We can just return an unhandled message and let it fall out naturally,
					// but since this is OnCustomKey, we want it to close it.
					// We'll let Update handle this.
					return false, nil
				}
			}
			return false, nil
		},
		RenderHeader: overlay.renderHeader,
		RenderFooter: overlay.renderFooter,
	}

	overlay.BaseOverlay = NewBaseOverlay(baseConfig)
	return overlay
}

// Update handles messages for the help overlay
func (h *HelpOverlay) Update(msg tea.Msg, state types.StateProvider, actions types.ActionHandler) (types.Overlay, tea.Cmd) {
	handled, updatedBase, cmd := h.BaseOverlay.Update(msg, actions)
	h.BaseOverlay = updatedBase

	if handled {
		// Check if this is a close key (ESC, Ctrl+C)
		// When BaseOverlay handles a close key, we should return nil to signal overlay close
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
				return nil, cmd
			}
		}
		return h, cmd
	}
	
	// Handle additional keys/events
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" || msg.String() == "ctrl+c" || msg.Type == tea.KeyEnter {
			return nil, nil // signal close
		}
	case tea.WindowSizeMsg:
		newOverlayWidth := types.ComputeOverlayWidth(msg.Width, 0.80, 56, 100)
		vpHeight := types.ComputeViewportHeight(msg.Height, 5)
		
		h.SetDimensions(newOverlayWidth, vpHeight + 5)
		
		vp := h.BaseOverlay.Viewport()
		vp.Width = newOverlayWidth - 4
		vp.Height = vpHeight
		
		// Re-initialize content when viewport resizes so it wraps correctly over new width
		h.BaseOverlay.SetContent(h.BaseOverlay.Viewport().View())
	}

	return h, nil
}

// renderHeader renders the help overlay header
func (h *HelpOverlay) renderHeader() string {
	contentWidth := h.BaseOverlay.Viewport().Width

	// Title
	titleLen := len(h.title)
	titlePadding := max(0, (contentWidth-titleLen)/2)
	titleStr := ""
	for i:=0; i<titlePadding; i++ { titleStr+=" " }
	titleStr += types.OverlayTitleStyle.Render(h.title)

	// Separator
	sepStr := ""
	for i:=0; i<contentWidth; i++ { sepStr+="─" }
	separator := lipgloss.NewStyle().Foreground(types.MutedGray).Render(sepStr)

	return titleStr + "\n" + separator + "\n"
}

// renderFooter renders the help overlay footer
func (h *HelpOverlay) renderFooter() string {
	contentWidth := h.BaseOverlay.Viewport().Width
	hint := "Esc or Enter to close"
	
	// For padding, we need width of string inside
	hintLen := lipgloss.Width(hint)
	hintPadding := max(0, (contentWidth-hintLen)/2)
	padStr := ""
	for i:=0; i<hintPadding; i++ { padStr+=" " }

	return "\n\n" + padStr + types.OverlayHelpStyle.Render(hint)
}

// View renders the help overlay
func (h *HelpOverlay) View() string {
	return h.BaseOverlay.View(h.BaseOverlay.Width())
}
