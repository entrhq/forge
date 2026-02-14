package browser

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/tools"
)

// ClickTool clicks an element in the browser session.
type ClickTool struct {
	manager *SessionManager
}

// NewClickTool creates a new click tool.
func NewClickTool(manager *SessionManager) *ClickTool {
	return &ClickTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *ClickTool) Name() string {
	return "browser_click"
}

// Description returns the tool description.
func (t *ClickTool) Description() string {
	return "Click an element in the browser session using a CSS selector. Supports single and double clicks, and different mouse buttons."
}

// Schema returns the tool's JSON schema.
func (t *ClickTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"session": map[string]interface{}{
				"type":        "string",
				"description": "Name of the browser session to use",
			},
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS selector for the element to click (e.g., 'button.submit', '#login-btn', 'a[href=\"/about\"]')",
			},
			"button": map[string]interface{}{
				"type":        "string",
				"description": "Mouse button to use: 'left' (default), 'right', or 'middle'",
			},
			"click_count": map[string]interface{}{
				"type":        "integer",
				"description": "Number of clicks: 1 (default) for single click, 2 for double click",
			},
		},
		[]string{"session", "selector"},
	)
}

// clickParams represents the parameters for clicking.
type clickParams struct {
	XMLName    xml.Name `xml:"arguments"`
	Session    string   `xml:"session"`
	Selector   string   `xml:"selector"`
	Button     string   `xml:"button"`
	ClickCount *int     `xml:"click_count"`
}

// Execute clicks an element.
func (t *ClickTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Parse parameters
	var input struct {
		XMLName    xml.Name `xml:"arguments"`
		Session    string   `xml:"session"`
		Selector   string   `xml:"selector"`
		Button     string   `xml:"button"`
		ClickCount *int     `xml:"click_count"`
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

	// Get session
	session, err := t.manager.GetSession(input.Session)
	if err != nil {
		return "", nil, err
	}

	// Build click options
	opts := ClickOptions{
		Selector:   input.Selector,
		Button:     input.Button,
		ClickCount: 1, // Default
	}

	if input.ClickCount != nil {
		if *input.ClickCount < 1 || *input.ClickCount > 3 {
			return "", nil, fmt.Errorf("click_count must be between 1 and 3")
		}
		opts.ClickCount = *input.ClickCount
	}

	// Validate button value
	if opts.Button != "" {
		validButtons := map[string]bool{
			"left":   true,
			"right":  true,
			"middle": true,
		}
		if !validButtons[opts.Button] {
			return "", nil, fmt.Errorf("invalid button: %s (must be 'left', 'right', or 'middle')", opts.Button)
		}
	}

	// Click the element
	if err := session.Click(opts); err != nil {
		return "", nil, err
	}

	// Build result message
	clickType := "single click"
	switch opts.ClickCount {
	case 2:
		clickType = "double click"
	case 3:
		clickType = "triple click"
	}

	buttonDesc := "left button"
	switch opts.Button {
	case "right":
		buttonDesc = "right button"
	case "middle":
		buttonDesc = "middle button"
	}

	result := fmt.Sprintf(`Click executed successfully

Click Details:
- Session: %s
- Selector: %s
- Action: %s with %s
- Current URL: %s

The element has been clicked. If this caused navigation or page changes, you may want to extract content or verify the new page state.`,
		input.Session,
		input.Selector,
		clickType,
		buttonDesc,
		session.CurrentURL,
	)

	return result, nil, nil
}

// IsLoopBreaking returns whether this tool breaks the agent loop.
func (t *ClickTool) IsLoopBreaking() bool {
	return false
}

// ShouldShow returns whether this tool should be visible.
// Click tools are only shown when there are active sessions.
func (t *ClickTool) ShouldShow() bool {
	return t.manager.HasSessions()
}
