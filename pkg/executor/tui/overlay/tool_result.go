package overlay

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/executor/tui/types"
)

// ToolResultOverlay displays the full result of a tool call
type ToolResultOverlay struct {
	*BaseOverlay
	toolName string
}

// NewToolResultOverlay creates a new tool result overlay
func NewToolResultOverlay(toolName, result string, width, height int) *ToolResultOverlay {
	// Calculate overlay dimensions (80% of screen)
	overlayWidth := int(float64(width) * 0.8)
	overlayHeight := int(float64(height) * 0.8)

	if overlayWidth < 60 {
		overlayWidth = 60
	}
	if overlayHeight < 20 {
		overlayHeight = 20
	}

	overlay := &ToolResultOverlay{
		toolName: toolName,
	}

	// Configure base overlay
	baseConfig := BaseOverlayConfig{
		Width:          overlayWidth,
		Height:         overlayHeight,
		ViewportWidth:  overlayWidth - 4,
		ViewportHeight: overlayHeight - 6,
		Content:        result,
		OnClose: func(actions types.ActionHandler) tea.Cmd {
			// Return nil to signal close - caller will handle ClearOverlay()
			return nil
		},
		OnCustomKey: func(msg tea.KeyMsg, actions types.ActionHandler) (bool, tea.Cmd) {
			// Allow 'q' and 'v' to close the overlay
			if msg.String() == "q" || msg.String() == "v" {
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

// Update handles messages
func (o *ToolResultOverlay) Update(msg tea.Msg, state types.StateProvider, actions types.ActionHandler) (types.Overlay, tea.Cmd) {
	handled, updatedBase, cmd := o.BaseOverlay.Update(msg, actions)
	o.BaseOverlay = updatedBase

	if handled {
		// Check if this is a close signal (Esc key pressed)
		// BaseOverlay.close() returns nil cmd, which signals the overlay wants to close
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "esc" || keyMsg.String() == "ctrl+c" {
				// Return nil to signal close - caller will handle ClearOverlay()
				return nil, cmd
			}
		}
		return o, cmd
	}

	return o, nil
}

// renderHeader renders the tool result header
func (o *ToolResultOverlay) renderHeader() string {
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(types.DiffHunkColor). // Cyan/Sky blue
		Render(fmt.Sprintf("Tool Result: %s", o.toolName))

	return header
}

// renderFooter renders the tool result footer
func (o *ToolResultOverlay) renderFooter() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("↑/↓: scroll • q/esc/v: close")
}

// View renders the overlay
func (o *ToolResultOverlay) View() string {
	return o.BaseOverlay.View(o.Width())
}
