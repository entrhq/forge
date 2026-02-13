package browser

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/tools"
)

// WaitTool waits for elements or conditions in the browser session.
type WaitTool struct {
	manager *SessionManager
}

// NewWaitTool creates a new wait tool.
func NewWaitTool(manager *SessionManager) *WaitTool {
	return &WaitTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *WaitTool) Name() string {
	return "browser_wait"
}

// Description returns the tool description.
func (t *WaitTool) Description() string {
	return "Wait for an element to reach a specific state in the browser session. Useful for waiting for dynamic content, loading indicators, or elements to appear/disappear."
}

// Schema returns the tool's JSON schema.
func (t *WaitTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"session": map[string]interface{}{
				"type":        "string",
				"description": "Name of the browser session to use",
			},
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "CSS selector for the element to wait for (e.g., '.loading-spinner', '#content')",
			},
			"state": map[string]interface{}{
				"type":        "string",
				"description": "State to wait for: 'attached' (in DOM), 'detached' (removed from DOM), 'visible' (default), or 'hidden'",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "Maximum wait time in milliseconds. Default: 30000 (30 seconds)",
			},
		},
		[]string{"session", "selector"},
	)
}

// waitParams represents the parameters for waiting.
type waitParams struct {
	XMLName  xml.Name `xml:"arguments"`
	Session  string   `xml:"session"`
	Selector string   `xml:"selector"`
	State    string   `xml:"state"`
	Timeout  *float64 `xml:"timeout"`
}

// Execute waits for an element.
func (t *WaitTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Parse parameters
	var input struct {
		XMLName  xml.Name `xml:"arguments"`
		Session  string   `xml:"session"`
		Selector string   `xml:"selector"`
		State    string   `xml:"state"`
		Timeout  *float64 `xml:"timeout"`
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

	// Build wait options
	opts := WaitOptions{
		Selector: input.Selector,
		State:    input.State,
	}

	if opts.State == "" {
		opts.State = "visible" // Default state
	}

	// Validate state value
	validStates := map[string]bool{
		"attached": true,
		"detached": true,
		"visible":  true,
		"hidden":   true,
	}
	if !validStates[opts.State] {
		return "", nil, fmt.Errorf("invalid state: %s (must be 'attached', 'detached', 'visible', or 'hidden')", opts.State)
	}

	if input.Timeout != nil {
		if *input.Timeout < 0 || *input.Timeout > 300000 {
			return "", nil, fmt.Errorf("timeout must be between 0 and 300000 milliseconds (5 minutes)")
		}
		opts.Timeout = *input.Timeout
	}

	// Wait for the element
	if err := session.Wait(opts); err != nil {
		return "", nil, fmt.Errorf("wait failed: %w", err)
	}

	// Build result message
	timeoutDesc := "30 seconds (default)"
	if input.Timeout != nil {
		timeoutDesc = fmt.Sprintf("%.0f milliseconds", *input.Timeout)
	}

	result := fmt.Sprintf(`Wait completed successfully

Wait Details:
- Session: %s
- Selector: %s
- State: %s
- Timeout: %s
- Current URL: %s

The element reached the desired state. You can now proceed with extraction, clicking, or other interactions.`,
		input.Session,
		input.Selector,
		opts.State,
		timeoutDesc,
		session.CurrentURL,
	)

	return result, nil, nil
}

// IsLoopBreaking returns whether this tool breaks the agent loop.
func (t *WaitTool) IsLoopBreaking() bool {
	return false
}

// ShouldShow returns whether this tool should be visible.
// Wait tools are only shown when there are active sessions.
func (t *WaitTool) ShouldShow() bool {
	return t.manager.HasSessions()
}


