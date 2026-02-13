package browser

import (
	"context"
	"encoding/xml"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/tools"
)

// ExtractContentTool extracts content from the current page.
type ExtractContentTool struct {
	manager *SessionManager
}

// NewExtractContentTool creates a new extract content tool.
func NewExtractContentTool(manager *SessionManager) *ExtractContentTool {
	return &ExtractContentTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *ExtractContentTool) Name() string {
	return "browser_extract_content"
}

// Description returns the tool description.
func (t *ExtractContentTool) Description() string {
	return "Extract content from the current page in the browser session. Supports multiple formats: markdown (default), plain text, or structured JSON."
}

// Schema returns the tool's JSON schema.
func (t *ExtractContentTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"session": map[string]interface{}{
				"type":        "string",
				"description": "Name of the browser session to extract from",
			},
			"format": map[string]interface{}{
				"type":        "string",
				"description": "Output format: 'markdown' (default), 'text', or 'structured' (JSON)",
			},
			"selector": map[string]interface{}{
				"type":        "string",
				"description": "Optional CSS selector to extract content from specific element (e.g., 'article', '.main-content')",
			},
			"max_length": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum content length in characters. Default: 10000",
			},
		},
		[]string{"session"},
	)
}

// extractContentParams represents the parameters for content extraction.
type extractContentParams struct {
	XMLName   xml.Name `xml:"arguments"`
	Session   string   `xml:"session"`
	Format    string   `xml:"format"`
	Selector  string   `xml:"selector"`
	MaxLength *int     `xml:"max_length"`
}

// Execute extracts content from the page.
func (t *ExtractContentTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Parse parameters
	var input struct {
		XMLName   xml.Name `xml:"arguments"`
		Session   string   `xml:"session"`
		Format    string   `xml:"format"`
		Selector  string   `xml:"selector"`
		MaxLength *int     `xml:"max_length"`
	}
	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate parameters
	if input.Session == "" {
		return "", nil, fmt.Errorf("session name is required")
	}

	// Get session
	session, err := t.manager.GetSession(input.Session)
	if err != nil {
		return "", nil, err
	}

	// Build extraction options
	opts := ExtractOptions{
		Format:    FormatMarkdown, // Default
		Selector:  input.Selector,
		MaxLength: DefaultMaxLength,
	}

	if input.Format != "" {
		switch input.Format {
		case "markdown":
			opts.Format = FormatMarkdown
		case "text":
			opts.Format = FormatText
		case "structured":
			opts.Format = FormatStructured
		default:
			return "", nil, fmt.Errorf("invalid format: %s (must be 'markdown', 'text', or 'structured')", input.Format)
		}
	}

	if input.MaxLength != nil {
		if *input.MaxLength < 100 || *input.MaxLength > 100000 {
			return "", nil, fmt.Errorf("max_length must be between 100 and 100000")
		}
		opts.MaxLength = *input.MaxLength
	}

	// Extract content
	content, err := session.ExtractContent(opts)
	if err != nil {
		return "", nil, err
	}

	// Build result message
	formatDesc := string(opts.Format)
	selectorDesc := "entire page"
	if opts.Selector != "" {
		selectorDesc = fmt.Sprintf("selector: %s", opts.Selector)
	}

	result := fmt.Sprintf(`Content extracted successfully

Extraction Details:
- Session: %s
- URL: %s
- Format: %s
- Source: %s
- Length: %d characters

---

%s`,
		input.Session,
		session.CurrentURL,
		formatDesc,
		selectorDesc,
		len(content),
		content,
	)

	return result, nil, nil
}

// IsLoopBreaking returns whether this tool breaks the agent loop.
func (t *ExtractContentTool) IsLoopBreaking() bool {
	return false
}

// ShouldShow returns whether this tool should be visible.
// Extract content tools are only shown when there are active sessions.
func (t *ExtractContentTool) ShouldShow() bool {
	return t.manager.HasSessions()
}
