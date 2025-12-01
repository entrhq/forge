package scratchpad

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/agent/memory/notes"
	"github.com/entrhq/forge/pkg/agent/tools"
)

// SearchNotesTool searches for notes by content and/or tags.
type SearchNotesTool struct {
	manager *notes.Manager
}

// NewSearchNotesTool creates a new SearchNotesTool.
func NewSearchNotesTool(manager *notes.Manager) *SearchNotesTool {
	return &SearchNotesTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *SearchNotesTool) Name() string {
	return "search_notes"
}

// Description returns the tool description.
func (t *SearchNotesTool) Description() string {
	return "Search for notes by content text and/or tags. Returns matching notes with relevance ranking."
}

// Schema returns the JSON schema for the tool's input parameters.
func (t *SearchNotesTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query to match against note content (case-insensitive substring match)",
			},
			"tags": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "Filter by tags (AND logic - all tags must match)",
			},
		},
		[]string{}, // Both query and tags are optional
	)
}

// Execute searches for notes.
func (t *SearchNotesTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var input struct {
		XMLName xml.Name `xml:"arguments"`
		Query   string   `xml:"query"`
		Tags    []string `xml:"tags>tag"`
	}

	if err := xml.Unmarshal(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Search for notes (include scratched notes)
	results := t.manager.Search(notes.SearchOptions{
		Query:            input.Query,
		Tags:             input.Tags,
		IncludeScratched: true,
	})

	// Build result message
	var message strings.Builder
	if len(results) == 0 {
		message.WriteString("No notes found matching the search criteria.")
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
		"result_count": len(results),
		"query":        input.Query,
		"tags":         input.Tags,
	}

	return message.String(), metadata, nil
}

// IsLoopBreaking returns false as this tool doesn't break the agent loop.
func (t *SearchNotesTool) IsLoopBreaking() bool {
	return false
}
