package agent

import (
	"github.com/entrhq/forge/pkg/agent/memory/notes"
	"github.com/entrhq/forge/pkg/types"
)

// handleNotesRequest processes a notes request and emits notes data event
func (a *DefaultAgent) handleNotesRequest(input *types.Input) {
	// Extract params from metadata
	paramsInterface, ok := input.Metadata["params"]
	if !ok {
		// No params provided, use defaults
		paramsInterface = types.NotesRequestParams{Limit: 10}
	}

	params, ok := paramsInterface.(types.NotesRequestParams)
	if !ok {
		agentDebugLog.Printf("Invalid notes request params type")
		return
	}

	// Build list options from request parameters
	opts := notes.ListOptions{
		Tag:              params.Tag,
		IncludeScratched: params.IncludeScratched,
		Limit:            params.Limit,
	}

	// Get notes from manager
	notesList := a.notesManager.List(opts)

	// Convert to NotesData format
	notesData := make([]types.NoteData, len(notesList))
	for i, note := range notesList {
		notesData[i] = types.NoteData{
			ID:        note.ID,
			Content:   note.Content,
			Tags:      note.Tags,
			Scratched: note.Scratched,
			CreatedAt: note.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: note.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	// Emit notes data event
	a.emitEvent(types.NewNotesDataEvent(notesData, ""))
}
