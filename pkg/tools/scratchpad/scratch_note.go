package scratchpad

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/memory/notes"
	"github.com/entrhq/forge/pkg/agent/tools"
)

// ScratchNoteTool marks notes as scratched/addressed.
type ScratchNoteTool struct {
	manager *notes.Manager
}

// NewScratchNoteTool creates a new ScratchNoteTool.
func NewScratchNoteTool(manager *notes.Manager) *ScratchNoteTool {
	return &ScratchNoteTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *ScratchNoteTool) Name() string {
	return "scratch_note"
}

// Description returns the tool description.
func (t *ScratchNoteTool) Description() string {
	return "Mark a note as scratched/addressed without deleting it. Preserves context while indicating the note is no longer active."
}

// Schema returns the JSON schema for the tool's input parameters.
func (t *ScratchNoteTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "ID of the note to mark as scratched",
			},
		},
		[]string{"id"},
	)
}

// Execute marks a note as scratched.
func (t *ScratchNoteTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
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

	// Scratch the note
	_, err := t.manager.Scratch(input.ID)
	if err != nil {
		return "", nil, err
	}

	// Build result message
	message := fmt.Sprintf("Note %s marked as scratched", input.ID)

	// Build metadata
	metadata := map[string]interface{}{
		"note_id":     input.ID,
		"total_notes": t.manager.Count(),
	}

	return message, metadata, nil
}

// IsLoopBreaking returns false as this tool doesn't break the agent loop.
func (t *ScratchNoteTool) IsLoopBreaking() bool {
	return false
}
