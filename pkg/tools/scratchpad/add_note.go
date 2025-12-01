package scratchpad

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/memory/notes"
	"github.com/entrhq/forge/pkg/agent/tools"
)

// AddNoteTool creates new notes in the scratchpad.
type AddNoteTool struct {
	manager *notes.Manager
}

// NewAddNoteTool creates a new AddNoteTool.
func NewAddNoteTool(manager *notes.Manager) *AddNoteTool {
	return &AddNoteTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *AddNoteTool) Name() string {
	return "add_note"
}

// Description returns the tool description.
func (t *AddNoteTool) Description() string {
	return "Create a new note in the scratchpad with content and tags for organizing information during task execution."
}

// Schema returns the JSON schema for the tool's input parameters.
func (t *AddNoteTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Note content (max 800 characters)",
			},
			"tags": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "List of 1-5 tags for organizing the note",
				"minItems":    1,
				"maxItems":    5,
			},
		},
		[]string{"content", "tags"},
	)
}

// Execute creates a new note.
func (t *AddNoteTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var input struct {
		XMLName xml.Name `xml:"arguments"`
		Content string   `xml:"content"`
		Tags    []string `xml:"tags>tag"`
	}

	if err := xml.Unmarshal(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.Content == "" {
		return "", nil, fmt.Errorf("missing required parameter: content")
	}

	if len(input.Tags) == 0 {
		return "", nil, fmt.Errorf("missing required parameter: tags (at least 1 tag required)")
	}

	// Add the note
	note, err := t.manager.Add(input.Content, input.Tags)
	if err != nil {
		return "", nil, err
	}

	// Build result message
	message := fmt.Sprintf("Note created successfully with ID: %s", note.ID)

	// Build metadata
	metadata := map[string]interface{}{
		"note_id":     note.ID,
		"total_notes": t.manager.Count(),
		"tags":        note.Tags,
	}

	return message, metadata, nil
}

// IsLoopBreaking returns false as this tool doesn't break the agent loop.
func (t *AddNoteTool) IsLoopBreaking() bool {
	return false
}
