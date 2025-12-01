package tui

import (
	"time"

	"github.com/entrhq/forge/pkg/executor/tui/types"
)

// SetOverlay activates an overlay
func (m *model) SetOverlay(mode types.OverlayMode, overlay types.Overlay) {
	m.overlay.activate(mode, overlay)
}

// ClearOverlay closes the current overlay
// If there's an overlay stack, it pops back to the previous overlay
// Otherwise it fully deactivates the overlay
func (m *model) ClearOverlay() {
	// Try to pop to previous overlay first
	hadPrevious := m.overlay.popOverlay()
	
	// Only refocus textarea if we fully closed all overlays
	if !hadPrevious {
		m.textarea.Focus()
	}
}

// ShowToast displays a toast notification
func (m *model) ShowToast(message, details, icon string, isError bool) {
	m.toast = &toastNotification{
		active:    true,
		message:   message,
		details:   details,
		icon:      icon,
		isError:   isError,
		showUntil: time.Now().Add(5 * time.Second),
	}
}

// SetInput sets the textarea content
func (m *model) SetInput(value string) {
	m.textarea.SetValue(value)
	m.updateTextAreaHeight()
}

// SetCursorEnd moves the cursor to the end of input
func (m *model) SetCursorEnd() {
	m.textarea.CursorEnd()
}

// Quit triggers application exit by setting a flag that will be checked in the Update loop.
// This allows overlays and other components to request app termination without directly
// returning tea.Quit (which would break the Bubble Tea command chain).
func (m *model) Quit() {
	m.shouldQuit = true
}
