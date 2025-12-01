package scratchpad

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/agent/memory/notes"
	"github.com/entrhq/forge/pkg/agent/tools"
)

// ListTagsTool lists all unique tags currently in use across active notes.
type ListTagsTool struct {
	manager *notes.Manager
}

// NewListTagsTool creates a new ListTagsTool.
func NewListTagsTool(manager *notes.Manager) *ListTagsTool {
	return &ListTagsTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *ListTagsTool) Name() string {
	return "list_tags"
}

// Description returns the tool description.
func (t *ListTagsTool) Description() string {
	return "List all unique tags currently in use across active (non-scratched) notes. Helps discover what topics and categories are being tracked."
}

// Schema returns the JSON schema for the tool's input parameters.
func (t *ListTagsTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{},
		[]string{},
	)
}

// Execute lists all tags.
func (t *ListTagsTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var input struct {
		XMLName xml.Name `xml:"arguments"`
	}

	if err := xml.Unmarshal(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Get all tags
	tags := t.manager.ListTags()

	// Build result message
	var message string
	if len(tags) == 0 {
		message = "No tags found in active notes."
	} else {
		message = fmt.Sprintf("Found %d unique tag(s) in active notes:\n\n%s",
			len(tags),
			strings.Join(tags, "\n"))
	}

	// Build metadata
	metadata := map[string]interface{}{
		"tags":         tags,
		"tag_count":    len(tags),
		"active_notes": t.manager.CountActive(),
		"total_notes":  t.manager.Count(),
	}

	return message, metadata, nil
}

// IsLoopBreaking returns false as this tool doesn't break the agent loop.
func (t *ListTagsTool) IsLoopBreaking() bool {
	return false
}
