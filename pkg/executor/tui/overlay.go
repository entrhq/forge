package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/executor/tui/types"
)

// overlayState tracks the active overlay and its state
type overlayState struct {
	mode    types.OverlayMode
	overlay types.Overlay
	stack   []overlayStackEntry // Stack of previous overlays for navigation
}

// overlayStackEntry represents a saved overlay state
type overlayStackEntry struct {
	mode    types.OverlayMode
	overlay types.Overlay
}

// newOverlayState creates a new overlay state
func newOverlayState() *overlayState {
	return &overlayState{
		mode: types.OverlayModeNone,
	}
}

// activate activates an overlay without affecting the stack
func (o *overlayState) activate(mode types.OverlayMode, overlay types.Overlay) {
	o.mode = mode
	o.overlay = overlay
}

// activateAndClearStack activates an overlay and clears the navigation stack
// Use this for top-level commands that should replace any existing overlay hierarchy
func (o *overlayState) activateAndClearStack(mode types.OverlayMode, overlay types.Overlay) {
	o.stack = nil
	o.mode = mode
	o.overlay = overlay
}

// pushOverlay saves current overlay and activates a new one
func (o *overlayState) pushOverlay(mode types.OverlayMode, overlay types.Overlay) {
	// Save current overlay to stack if one is active
	if o.mode != types.OverlayModeNone && o.overlay != nil {
		o.stack = append(o.stack, overlayStackEntry{
			mode:    o.mode,
			overlay: o.overlay,
		})
	}
	// Activate new overlay
	o.mode = mode
	o.overlay = overlay
}

// popOverlay returns to the previous overlay in the stack
// Returns true if there was a previous overlay, false if stack was empty
func (o *overlayState) popOverlay() bool {
	if len(o.stack) == 0 {
		// No previous overlay, fully deactivate
		o.deactivate()
		return false
	}
	
	// Pop the last overlay from the stack
	lastIdx := len(o.stack) - 1
	prev := o.stack[lastIdx]
	o.stack = o.stack[:lastIdx]
	
	// Restore it
	o.mode = prev.mode
	o.overlay = prev.overlay
	return true
}

// deactivate closes the current overlay
func (o *overlayState) deactivate() {
	o.mode = types.OverlayModeNone
	o.overlay = nil
}

// isActive returns whether any overlay is currently active
func (o *overlayState) isActive() bool {
	// Check mode first, then verify overlay is not nil
	if o.mode == types.OverlayModeNone {
		return false
	}
	// If mode is set but overlay is nil, this is an inconsistent state
	// We should deactivate to prevent panics
	if o.overlay == nil {
		o.mode = types.OverlayModeNone
		return false
	}
	return true
}

// renderOverlay renders an overlay centered on a clean background
// This creates a modal appearance by not showing the base view underneath
func renderOverlay(baseView string, overlay types.Overlay, width, height int) string {
	if overlay == nil {
		return baseView
	}

	// Get the overlay content
	overlayView := overlay.View()

	// Position the overlay centered on a clean background
	// The lipgloss.Place function will fill the remaining space with whitespace
	// creating a clean modal appearance
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		overlayView,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// renderToastOverlay renders a toast-style overlay at the bottom of the screen
// without affecting the base view's layout
func renderToastOverlay(baseView string, toastContent string) string {
	if toastContent == "" {
		return baseView
	}

	// Split base view into lines
	baseLines := strings.Split(baseView, "\n")

	// Calculate where to position the toast (bottom of screen, above input area)
	// We want to overlay it on top of the existing content
	toastLines := strings.Split(strings.TrimRight(toastContent, "\n"), "\n")
	toastHeight := len(toastLines)

	// Position toast starting from a few lines above the bottom
	// This puts it just above the input box
	startLine := len(baseLines) - 5 - toastHeight
	if startLine < 0 {
		startLine = 0
	}

	// Build result with toast overlaid
	var result strings.Builder
	for i, line := range baseLines {
		toastLineIdx := i - startLine
		if toastLineIdx >= 0 && toastLineIdx < len(toastLines) {
			// Overlay the toast line, left-aligned with small padding
			toastLine := toastLines[toastLineIdx]
			padding := 2 // Left padding for spacing from edge
			// Write toast with left padding
			result.WriteString(strings.Repeat(" ", padding))
			result.WriteString(toastLine)
		} else {
			result.WriteString(line)
		}
		if i < len(baseLines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
