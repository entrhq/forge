package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrhq/forge/pkg/executor/tui/overlay"
	"github.com/entrhq/forge/pkg/executor/tui/types"
)

// mockMsg is a custom message type that only the mock overlay handles
type mockMsg struct{}

// mockOverlay is a mock implementation of types.Overlay
type mockOverlay struct {
	shouldClose bool
	cmdToReturn tea.Cmd
}

func (m *mockOverlay) Update(msg tea.Msg, state types.StateProvider, actions types.ActionHandler) (types.Overlay, tea.Cmd) {
	// Only respond to mockMsg or always respond if desired.
	// We want to trigger the closing logic.
	if _, ok := msg.(mockMsg); ok && m.shouldClose {
		return nil, m.cmdToReturn
	}
	return m, nil
}

func (m *mockOverlay) View() string           { return "" }
func (m *mockOverlay) Width() int             { return 0 }
func (m *mockOverlay) Height() int            { return 0 }
func (m *mockOverlay) SetDimensions(w, h int) {}
func (m *mockOverlay) Focused() bool          { return true }
func (m *mockOverlay) SetFocused(f bool)      {}

// TestUpdate_OverlayClosingWithCommand verifies that when an overlay closes and returns a command,
// that command is not dropped by the main Update loop.
func TestUpdate_OverlayClosingWithCommand(t *testing.T) {
	// Setup mock command
	expectedMsg := "mock-command-executed"
	mockCmd := func() tea.Msg { return expectedMsg }

	// Initialize model with minimal components
	m := &model{
		overlay:        newOverlayState(),
		spinner:        spinner.New(),
		textarea:       textarea.New(),
		viewport:       viewport.New(80, 24),
		commandPalette: overlay.NewCommandPalette(nil),
		content:        &strings.Builder{},
		// We need to initialize other fields to avoid nil panics in Update
	}
	// Initialize logging which Update uses
	initDebugLog()

	// Activate overlay
	activeOverlay := &mockOverlay{
		shouldClose: true,
		cmdToReturn: mockCmd,
	}
	m.overlay.activate(types.OverlayModeApproval, activeOverlay)

	// Verify overlay is active initially
	if !m.overlay.isActive() {
		t.Fatal("Overlay should be active initially")
	}

	// Call Update with a mockMsg to trigger the overlay update
	// textarea and other components don't listen for mockMsg, so they should return nil cmd.
	// Only our fix should ensure that mockCmd makes it into the final batch.
	_, cmd := m.Update(mockMsg{})

	// Verify overlay is closed
	if m.overlay.isActive() {
		t.Error("Overlay should be closed after Update")
	}
	if m.overlay.overlay != nil {
		t.Error("Overlay reference should be nil after Update")
	}

	// Verify the returned command is not nil
	if cmd == nil {
		t.Fatal("Returned command should not be nil")
	}

	t.Log("Test finished - verified that overlay closed and a command was returned.")
}
