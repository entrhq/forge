package browser

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/config"
)

// StartSessionTool creates a new browser session.
type StartSessionTool struct {
	manager *SessionManager
}

// NewStartSessionTool creates a new start session tool.
func NewStartSessionTool(manager *SessionManager) *StartSessionTool {
	return &StartSessionTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *StartSessionTool) Name() string {
	return "start_browser_session"
}

// Description returns the tool description.
func (t *StartSessionTool) Description() string {
	return "Create a new browser session for web automation. Sessions persist across agent loop iterations and can be reused for multiple operations."
}

// Schema returns the tool's JSON schema.
func (t *StartSessionTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Unique name for the browser session (e.g., 'research', 'app_test')",
			},
			"headless": map[string]interface{}{
				"type":        "boolean",
				"description": "Run browser in headless mode (no visible window). Default: false (headed mode for transparency)",
			},
			"width": map[string]interface{}{
				"type":        "integer",
				"description": "Browser viewport width in pixels. Default: 1280",
			},
			"height": map[string]interface{}{
				"type":        "integer",
				"description": "Browser viewport height in pixels. Default: 720",
			},
		},
		[]string{"name"},
	)
}

// startSessionParams represents the parameters for starting a session.
type startSessionParams struct {
	XMLName  xml.Name `xml:"arguments"`
	Name     string   `xml:"name"`
	Headless *bool    `xml:"headless"`
	Width    *int     `xml:"width"`
	Height   *int     `xml:"height"`
}

// Execute starts a new browser session.
func (t *StartSessionTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Parse parameters
	var input struct {
		XMLName  xml.Name `xml:"arguments"`
		Name     string   `xml:"name"`
		Headless *bool    `xml:"headless"`
		Width    *int     `xml:"width"`
		Height   *int     `xml:"height"`
	}
	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate session name
	if input.Name == "" {
		return "", nil, fmt.Errorf("session name is required")
	}

	// Build session options with config defaults
	headlessDefault := false // Fallback if config not available
	if config.IsInitialized() {
		if ui := config.GetUI(); ui != nil {
			headlessDefault = ui.IsBrowserHeadless()
		}
	}

	opts := SessionOptions{
		Headless: headlessDefault,
		Viewport: &Viewport{
			Width:  DefaultViewportWidth,
			Height: DefaultViewportHeight,
		},
		Timeout: DefaultTimeout,
	}

	if input.Headless != nil {
		opts.Headless = *input.Headless
	}
	if input.Width != nil {
		opts.Viewport.Width = *input.Width
	}
	if input.Height != nil {
		opts.Viewport.Height = *input.Height
	}

	// Validate viewport dimensions
	if opts.Viewport.Width < 100 || opts.Viewport.Width > 5000 {
		return "", nil, fmt.Errorf("viewport width must be between 100 and 5000 pixels")
	}
	if opts.Viewport.Height < 100 || opts.Viewport.Height > 5000 {
		return "", nil, fmt.Errorf("viewport height must be between 100 and 5000 pixels")
	}

	// Ensure manager is initialized
	if err := t.manager.Initialize(); err != nil {
		return "", nil, fmt.Errorf("failed to initialize browser: %w", err)
	}

	// Start session
	session, err := t.manager.StartSession(input.Name, opts)
	if err != nil {
		return "", nil, fmt.Errorf("failed to start session: %w", err)
	}

	// Build success message
	mode := "headed"
	if session.Headless {
		mode = "headless"
	}

	result := fmt.Sprintf(`Browser session created successfully

Session Details:
- Name: %s
- Mode: %s
- Viewport: %dx%d pixels
- Status: Ready

The session is now active and browser tools are available. Use navigate, extract_content, click, fill, and other browser tools to interact with web pages.

%s`,
		session.Name,
		mode,
		opts.Viewport.Width,
		opts.Viewport.Height,
		func() string {
			if session.Headless {
				return "Browser is running in the background (headless mode)."
			}
			return "You should see a browser window open. The agent will interact with this window."
		}(),
	)

	return result, nil, nil
}

// IsLoopBreaking returns whether this tool breaks the agent loop.
func (t *StartSessionTool) IsLoopBreaking() bool {
	return false
}

// ShouldShow returns whether this tool should be visible.
// Session management tools are only shown when browser automation is enabled in settings.
func (t *StartSessionTool) ShouldShow() bool {
	if !config.IsInitialized() {
		return false
	}
	ui := config.GetUI()
	if ui == nil {
		return false
	}
	return ui.IsBrowserEnabled()
}

// GeneratePreview generates a preview of the browser session start.
func (t *StartSessionTool) GeneratePreview(ctx context.Context, argsXML []byte) (*tools.ToolPreview, error) {
	var params startSessionParams
	if err := tools.UnmarshalXMLWithFallback(argsXML, &params); err != nil {
		return nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	preview := &tools.ToolPreview{
		Type:        tools.PreviewTypeCommand,
		Title:       "Start Browser Session",
		Description: fmt.Sprintf("Opening a new browser session named '%s'", params.Name),
		Metadata: map[string]interface{}{
			"session_name": params.Name,
			"headless":     params.Headless,
		},
	}

	return preview, nil
}


