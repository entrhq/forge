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
// StartSessionInput defines the input parameters for starting a browser session.
type StartSessionInput struct {
	XMLName  xml.Name `xml:"arguments"`
	Name     string   `xml:"name"`
	Headless *bool    `xml:"headless"`
	Width    *int     `xml:"width"`
	Height   *int     `xml:"height"`
}

// Execute starts a new browser session.
func (t *StartSessionTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Parse and validate input
	input, err := t.parseInput(argsXML)
	if err != nil {
		return "", nil, err
	}

	// Build session options
	opts := t.buildSessionOptions(input)

	// Validate viewport dimensions
	if validateErr := t.validateViewport(opts.Viewport); validateErr != nil {
		return "", nil, validateErr
	}

	// Ensure manager is initialized
	if initErr := t.manager.Initialize(); initErr != nil {
		return "", nil, fmt.Errorf("failed to initialize browser: %w", initErr)
	}

	// Start session
	session, err := t.manager.StartSession(input.Name, opts)
	if err != nil {
		return "", nil, fmt.Errorf("failed to start session: %w", err)
	}

	// Build and return success message
	result := t.buildSuccessMessage(session, opts)
	return result, nil, nil
}

// parseInput parses and validates the XML input parameters.
func (t *StartSessionTool) parseInput(argsXML []byte) (*StartSessionInput, error) {
	var input StartSessionInput
	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if input.Name == "" {
		return nil, fmt.Errorf("session name is required")
	}

	return &input, nil
}

// buildSessionOptions constructs SessionOptions from input and config defaults.
func (t *StartSessionTool) buildSessionOptions(input *StartSessionInput) SessionOptions {
	// Get headless default from config
	headlessDefault := false
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

	// Apply user-provided overrides
	if input.Headless != nil {
		opts.Headless = *input.Headless
	}
	if input.Width != nil {
		opts.Viewport.Width = *input.Width
	}
	if input.Height != nil {
		opts.Viewport.Height = *input.Height
	}

	return opts
}

// validateViewport validates viewport dimensions are within acceptable range.
func (t *StartSessionTool) validateViewport(vp *Viewport) error {
	if vp.Width < 100 || vp.Width > 5000 {
		return fmt.Errorf("viewport width must be between 100 and 5000 pixels")
	}
	if vp.Height < 100 || vp.Height > 5000 {
		return fmt.Errorf("viewport height must be between 100 and 5000 pixels")
	}
	return nil
}

// buildSuccessMessage creates the success message for session creation.
func (t *StartSessionTool) buildSuccessMessage(session *Session, opts SessionOptions) string {
	mode := "headed"
	if session.Headless {
		mode = "headless"
	}

	visibilityNote := "You should see a browser window open. The agent will interact with this window."
	if session.Headless {
		visibilityNote = "Browser is running in the background (headless mode)."
	}

	return fmt.Sprintf(`Browser session created successfully

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
		visibilityNote,
	)
}

// GeneratePreview generates a concise one-line preview of the start session operation.
func (t *StartSessionTool) GeneratePreview(argsXML []byte) (string, error) {
	input, err := t.parseInput(argsXML)
	if err != nil {
		return "", err
	}

	mode := "headed"
	if input.Headless != nil && *input.Headless {
		mode = "headless"
	}

	width := DefaultViewportWidth
	if input.Width != nil {
		width = *input.Width
	}

	height := DefaultViewportHeight
	if input.Height != nil {
		height = *input.Height
	}

	return fmt.Sprintf("Start browser session '%s' (%s, %dx%d)", input.Name, mode, width, height), nil
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
