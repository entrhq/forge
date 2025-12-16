package overlay

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrhq/forge/pkg/executor/tui/types"
)

// HelpOverlay displays help information in a modal dialog
type HelpOverlay struct {
	*BaseOverlay
	title string
}

// NewHelpOverlay creates a new help overlay
func NewHelpOverlay(title, content string) *HelpOverlay {
	const (
		viewportWidth  = 76
		viewportHeight = 20
		overlayWidth   = 80
		overlayHeight  = 25
	)

	overlay := &HelpOverlay{
		title: title,
	}

	// Configure base overlay with custom key handler
	baseConfig := BaseOverlayConfig{
		Width:          overlayWidth,
		Height:         overlayHeight,
		ViewportWidth:  viewportWidth,
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
					return true, overlay.close(actions)
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
		return h, cmd
	}

	return h, nil
}

// renderHeader renders the help overlay header
func (h *HelpOverlay) renderHeader() string {
	return types.OverlayTitleStyle.Render(h.title)
}

// renderFooter renders the help overlay footer
func (h *HelpOverlay) renderFooter() string {
	return types.OverlayHelpStyle.Render("Press ESC or Enter to close")
}

// View renders the help overlay
func (h *HelpOverlay) View() string {
	return h.BaseOverlay.View(h.BaseOverlay.Viewport().Width)
}
