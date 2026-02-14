package browser

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/config"
)

// CloseSessionTool closes a browser session.
type CloseSessionTool struct {
	manager *SessionManager
}

// NewCloseSessionTool creates a new close session tool.
func NewCloseSessionTool(manager *SessionManager) *CloseSessionTool {
	return &CloseSessionTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *CloseSessionTool) Name() string {
	return "close_browser_session"
}

// Description returns the tool description.
func (t *CloseSessionTool) Description() string {
	return "Close a browser session and clean up resources. The browser window will close and the session will no longer be available."
}

// Schema returns the tool's JSON schema.
func (t *CloseSessionTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"session": map[string]interface{}{
				"type":        "string",
				"description": "Name of the browser session to close",
			},
		},
		[]string{"session"},
	)
}

// CloseSessionInput represents the parameters for closing a session.
type CloseSessionInput struct {
	XMLName xml.Name `xml:"arguments"`
	Session string   `xml:"session"`
}

// Execute closes a browser session.
func (t *CloseSessionTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Parse parameters
	var input CloseSessionInput
	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate parameters
	if input.Session == "" {
		return "", nil, fmt.Errorf("session name is required")
	}

	// Close session
	err := t.manager.CloseSession(input.Session)
	if err != nil {
		return "", nil, fmt.Errorf("failed to close session: %w", err)
	}

	result := fmt.Sprintf(`Session closed successfully

Session: %s

The browser window has been closed and all resources have been cleaned up. Browser tools will remain available if other sessions are still active.`,
		input.Session,
	)

	return result, nil, nil
}

// IsLoopBreaking returns whether this tool breaks the agent loop.
func (t *CloseSessionTool) IsLoopBreaking() bool {
	return false
}

// ShouldShow returns whether this tool should be visible.
// Session management tools are only shown when browser automation is enabled in settings.
func (t *CloseSessionTool) ShouldShow() bool {
	if !config.IsInitialized() {
		return false
	}
	ui := config.GetUI()
	if ui == nil {
		return false
	}
	return ui.IsBrowserEnabled()
}

// GeneratePreview implements the Previewable interface to show close session details before execution.
func (t *CloseSessionTool) GeneratePreview(ctx context.Context, argsXML []byte) (*tools.ToolPreview, error) {
	var input struct {
		XMLName xml.Name `xml:"arguments"`
		Session string   `xml:"session"`
	}

	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.Session == "" {
		return nil, fmt.Errorf("session name is required")
	}

	return &tools.ToolPreview{
		Type:        tools.PreviewTypeCommand,
		Title:       "Close Browser Session",
		Description: fmt.Sprintf("This will close browser session '%s'", input.Session),
		Content:     fmt.Sprintf("Session: %s\n\nThe browser window will be closed and all resources will be cleaned up.", input.Session),
		Metadata: map[string]interface{}{
			"session": input.Session,
		},
	}, nil
}
