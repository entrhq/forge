package browser

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/tools"
)

// FillTool fills form inputs in the browser session.
type FillTool struct {
	manager *SessionManager
}

// NewFillTool creates a new fill tool.
func NewFillTool(manager *SessionManager) *FillTool {
	return &FillTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *FillTool) Name() string {
	return "browser_fill_form"
}

// Description returns the tool description.
func (t *FillTool) Description() string {
	return "Fill a form input field in the browser session. Works with text inputs, textareas, and other fillable elements."
}

// Schema returns the tool's JSON schema.
func (t *FillTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"session": map[string]interface{}{
				"type":        "string",
				"description": "Name of the browser session to use",
			},
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS selector for the input element to fill (e.g., 'input[name=\"email\"]', '#password', 'textarea.comment')",
			},
			"value": map[string]interface{}{
				"type":        "string",
				"description": "Text value to fill into the input field",
			},
		},
		[]string{"session", "selector", "value"},
	)
}

// fillParams represents the parameters for filling.
type fillParams struct {
	XMLName  xml.Name `xml:"arguments"`
	Session  string   `xml:"session"`
	Selector string   `xml:"selector"`
	Value    string   `xml:"value"`
}

// Execute fills a form input.
func (t *FillTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Parse parameters
	var input struct {
		XMLName  xml.Name `xml:"arguments"`
		Session  string   `xml:"session"`
		Selector string   `xml:"selector"`
		Value    string   `xml:"value"`
	}
	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate parameters
	if input.Session == "" {
		return "", nil, fmt.Errorf("session name is required")
	}
	if input.Selector == "" {
		return "", nil, fmt.Errorf("selector is required")
	}
	// Note: value can be empty string (e.g., clearing a field)

	// Get session
	session, err := t.manager.GetSession(input.Session)
	if err != nil {
		return "", nil, err
	}

	// Build fill options
	opts := FillOptions{
		Selector: input.Selector,
		Value:    input.Value,
	}

	// Fill the input
	if err := session.Fill(opts); err != nil {
		return "", nil, err
	}

	// Build result message
	valueDesc := fmt.Sprintf("%d characters", len(input.Value))
	if input.Value == "" {
		valueDesc = "empty (cleared field)"
	}

	result := fmt.Sprintf(`Form field filled successfully

Fill Details:
- Session: %s
- Selector: %s
- Value: %s
- Current URL: %s

The form field has been filled with the specified value. You can now click submit buttons or fill additional fields.`,
		input.Session,
		input.Selector,
		valueDesc,
		session.CurrentURL,
	)

	return result, nil, nil
}

// IsLoopBreaking returns whether this tool breaks the agent loop.
func (t *FillTool) IsLoopBreaking() bool {
	return false
}

// ShouldShow returns whether this tool should be visible.
// Fill tools are only shown when there are active sessions.
func (t *FillTool) ShouldShow() bool {
	return t.manager.HasSessions()
}
