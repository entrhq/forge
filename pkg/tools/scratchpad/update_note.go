package scratchpad

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/memory/notes"
	"github.com/entrhq/forge/pkg/agent/tools"
)

// UpdateNoteTool updates existing notes.
type UpdateNoteTool struct {
	manager *notes.Manager
}

// NewUpdateNoteTool creates a new UpdateNoteTool.
func NewUpdateNoteTool(manager *notes.Manager) *UpdateNoteTool {
	return &UpdateNoteTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *UpdateNoteTool) Name() string {
	return "update_note"
}

// Description returns the tool description.
func (t *UpdateNoteTool) Description() string {
	return "Update an existing note's content and/or tags. At least one of content or tags must be provided."
}

// Schema returns the JSON schema for the tool's input parameters.
func (t *UpdateNoteTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the note to update",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "New content for the note (replaces existing if provided)",
			},
			"tags": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "New tags for the note (replaces existing if provided, must be 1-5 tags)",
				"minItems":    1,
				"maxItems":    5,
			},
		},
		[]string{"id"}, // Only ID is required
	)
}

// Execute updates a note.
func (t *UpdateNoteTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var input struct {
		XMLName xml.Name `xml:"arguments"`
		ID      string   `xml:"id"`
		Content string   `xml:"content"`
		Tags    []string `xml:"tags>tag"`
	}

	if err := xml.Unmarshal(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.ID == "" {
		return "", nil, fmt.Errorf("missing required parameter: id")
	}

	// At least one of content or tags must be provided
	if input.Content == "" && len(input.Tags) == 0 {
		return "", nil, fmt.Errorf("at least one of content or tags must be provided")
	}

	// Update the note
	var contentPtr *string
	if input.Content != "" {
		contentPtr = &input.Content
	}
	note, err := t.manager.Update(input.ID, contentPtr, input.Tags)
	if err != nil {
		return "", nil, err
	}

	// Build result message
	var message string
	if input.Content != "" && len(input.Tags) > 0 {
		message = fmt.Sprintf("Note %s updated successfully (content and tags modified)", note.ID)
	} else if input.Content != "" {
		message = fmt.Sprintf("Note %s updated successfully (content modified)", note.ID)
	} else {
		message = fmt.Sprintf("Note %s updated successfully (tags modified)", note.ID)
	}

	// Build metadata
	metadata := map[string]interface{}{
		"note_id":     note.ID,
		"total_notes": t.manager.Count(),
	}

	return message, metadata, nil
}

// IsLoopBreaking returns false as this tool doesn't break the agent loop.
func (t *UpdateNoteTool) IsLoopBreaking() bool {
	return false
}
