package scratchpad

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/memory/notes"
	"github.com/entrhq/forge/pkg/agent/tools"
)

// DeleteNoteTool permanently deletes notes.
type DeleteNoteTool struct {
	manager *notes.Manager
}

// NewDeleteNoteTool creates a new DeleteNoteTool.
func NewDeleteNoteTool(manager *notes.Manager) *DeleteNoteTool {
	return &DeleteNoteTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *DeleteNoteTool) Name() string {
	return "delete_note"
}

// Description returns the tool description.
func (t *DeleteNoteTool) Description() string {
	return "Permanently delete a note from the scratchpad. Consider using scratch_note instead to preserve context."
}

// Schema returns the JSON schema for the tool's input parameters.
func (t *DeleteNoteTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the note to delete",
			},
		},
		[]string{"id"},
	)
}

// Execute deletes a note.
func (t *DeleteNoteTool) Execute(_ context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var input struct {
		XMLName xml.Name `xml:"arguments"`
		ID      string   `xml:"id"`
	}

	if err := xml.Unmarshal(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.ID == "" {
		return "", nil, fmt.Errorf("missing required parameter: id")
	}

	// Delete the note
	if err := t.manager.Delete(input.ID); err != nil {
		return "", nil, err
	}

	// Build result message
	message := fmt.Sprintf("Note %s deleted successfully", input.ID)

	// Build metadata
	metadata := map[string]interface{}{
		"note_id":     input.ID,
		"total_notes": t.manager.Count(),
	}

	return message, metadata, nil
}

// IsLoopBreaking returns false as this tool doesn't break the agent loop.
func (t *DeleteNoteTool) IsLoopBreaking() bool {
	return false
}
