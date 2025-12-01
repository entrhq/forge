package scratchpad

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/agent/memory/notes"
	"github.com/entrhq/forge/pkg/agent/tools"
)

// ListNotesTool lists all notes with optional filtering.
type ListNotesTool struct {
	manager *notes.Manager
}

// NewListNotesTool creates a new ListNotesTool.
func NewListNotesTool(manager *notes.Manager) *ListNotesTool {
	return &ListNotesTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *ListNotesTool) Name() string {
	return "list_notes"
}

// Description returns the tool description.
func (t *ListNotesTool) Description() string {
	return "List all notes with optional filtering by tag and scratched status. Returns notes sorted by creation time (newest first)."
}

// Schema returns the JSON schema for the tool's input parameters.
func (t *ListNotesTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"tag": map[string]interface{}{
				"type":        "string",
				"description": "Optional tag to filter notes by",
			},
			"include_scratched": map[string]interface{}{
				"type":        "boolean",
				"description": "Include scratched notes in the results (default: false)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of notes to return (default: 10)",
			},
		},
		[]string{}, // All parameters are optional
	)
}

// Execute lists notes.
func (t *ListNotesTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var input struct {
		XMLName          xml.Name `xml:"arguments"`
		Tag              string   `xml:"tag"`
		IncludeScratched bool     `xml:"include_scratched"`
		Limit            int      `xml:"limit"`
	}

	if err := xml.Unmarshal(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Default limit
	if input.Limit == 0 {
		input.Limit = 10
	}

	// List notes
	results := t.manager.List(notes.ListOptions{
		Tag:              input.Tag,
		IncludeScratched: input.IncludeScratched,
		Limit:            input.Limit,
	})

	// Build result message
	var message strings.Builder
	if len(results) == 0 {
		message.WriteString("No notes found.")
	} else {
		message.WriteString(fmt.Sprintf("Found %d note(s):\n\n", len(results)))
		for i, note := range results {
			message.WriteString(fmt.Sprintf("%d. [%s] (tags: %s)\n",
				i+1, note.ID, strings.Join(note.Tags, ", ")))
			message.WriteString(fmt.Sprintf("   %s\n", note.Content))
			if note.Scratched {
				message.WriteString("   [SCRATCHED]\n")
			}
			message.WriteString("\n")
		}
	}

	// Build metadata
	metadata := map[string]interface{}{
		"note_count":        len(results),
		"tag_filter":        input.Tag,
		"include_scratched": input.IncludeScratched,
		"limit":             input.Limit,
	}

	return message.String(), metadata, nil
}

// IsLoopBreaking returns false as this tool doesn't break the agent loop.
func (t *ListNotesTool) IsLoopBreaking() bool {
	return false
}
