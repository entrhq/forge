package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrhq/forge/pkg/types"
)

// requestNotes sends a notes list request to the agent
func (m *model) requestNotes() tea.Cmd {
	// Mark that we're waiting for notes data
	m.pendingNotesRequest = true

	return func() tea.Msg {
		// Create notes request input
		notesInput := types.NewNotesRequestInput(types.NotesRequestParams{
			Limit: 100, // Request up to 100 notes
		})

		// Send to agent
		m.channels.Input <- notesInput

		// Return operation start message to show loading state
		return operationStartMsg{
			message: "Loading notes...",
		}
	}
}
