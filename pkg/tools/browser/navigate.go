package browser

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/tools"
)

// NavigateTool navigates to a URL in a browser session.
type NavigateTool struct {
	manager *SessionManager
}

// NewNavigateTool creates a new navigate tool.
func NewNavigateTool(manager *SessionManager) *NavigateTool {
	return &NavigateTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *NavigateTool) Name() string {
	return "browser_navigate"
}

// Description returns the tool description.
func (t *NavigateTool) Description() string {
	return "Navigate to a URL in an active browser session. The browser will load the page and wait for it to be ready."
}

// Schema returns the tool's JSON schema.
func (t *NavigateTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"session": map[string]interface{}{
				"type":        "string",
				"description": "Name of the browser session to use",
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "URL to navigate to (must include protocol, e.g., https://example.com)",
			},
			"wait_until": map[string]interface{}{
				"type":        "string",
				"description": "When to consider navigation complete: 'load' (default), 'domcontentloaded', or 'networkidle'",
			},
		},
		[]string{"session", "url"},
	)
}

// NavigateInput represents the parameters for navigation.
type NavigateInput struct {
	XMLName   xml.Name `xml:"arguments"`
	Session   string   `xml:"session"`
	URL       string   `xml:"url"`
	WaitUntil string   `xml:"wait_until"`
}

// Execute navigates to a URL.
func (t *NavigateTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Parse parameters
	var input NavigateInput
	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate parameters
	if input.Session == "" {
		return "", nil, fmt.Errorf("session name is required")
	}
	if input.URL == "" {
		return "", nil, fmt.Errorf("URL is required")
	}

	// Get session
	session, err := t.manager.GetSession(input.Session)
	if err != nil {
		return "", nil, err
	}

	// Build navigation options
	opts := NavigateOptions{
		WaitUntil: input.WaitUntil,
	}
	if opts.WaitUntil == "" {
		opts.WaitUntil = "load"
	}

	// Validate wait_until value
	validWaitStates := map[string]bool{
		"load":             true,
		"domcontentloaded": true,
		"networkidle":      true,
	}
	if !validWaitStates[opts.WaitUntil] {
		return "", nil, fmt.Errorf("invalid wait_until value: %s (must be 'load', 'domcontentloaded', or 'networkidle')", opts.WaitUntil)
	}

	// Navigate
	if navErr := session.Navigate(input.URL, opts); navErr != nil {
		return "", nil, navErr
	}

	// Get page title
	title, err := session.Page.Title()
	if err != nil {
		title = "Unknown"
	}

	result := fmt.Sprintf(`Navigation successful

Page Details:
- URL: %s
- Title: %s
- Session: %s

The page has loaded and is ready for interaction. You can now use extract_content, click, fill, and other browser tools to interact with the page.`,
		session.CurrentURL,
		title,
		input.Session,
	)

	return result, nil, nil
}

// IsLoopBreaking returns whether this tool breaks the agent loop.
func (t *NavigateTool) IsLoopBreaking() bool {
	return false
}

// ShouldShow returns whether this tool should be visible.
// Navigation tools are only shown when there are active sessions.
func (t *NavigateTool) ShouldShow() bool {
	return t.manager.HasSessions()
}
