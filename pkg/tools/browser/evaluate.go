package browser

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/tools"
)

// EvaluateTool executes JavaScript code in the browser session.
type EvaluateTool struct {
	manager *SessionManager
}

// NewEvaluateTool creates a new evaluate tool.
func NewEvaluateTool(manager *SessionManager) *EvaluateTool {
	return &EvaluateTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *EvaluateTool) Name() string {
	return "browser_evaluate"
}

// Description returns the tool description.
func (t *EvaluateTool) Description() string {
	return "Execute JavaScript code in the browser session. Can be used to manipulate the DOM, extract data, or interact with page elements programmatically. Returns the result of the JavaScript expression."
}

// Schema returns the tool's JSON schema.
func (t *EvaluateTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"session": map[string]interface{}{
				"type":        "string",
				"description": "Name of the browser session to use",
			},
			"code": map[string]interface{}{
				"type":        "string",
				"description": "JavaScript code to execute. Can be an expression or a function body. For complex operations, wrap in an IIFE: (function() { /* code */ })();",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "Execution timeout in milliseconds. Default: 30000 (30 seconds)",
			},
		},
		[]string{"session", "code"},
	)
}

// EvaluateInput defines the input parameters.
type EvaluateInput struct {
	XMLName xml.Name `xml:"arguments"`
	Session string   `xml:"session"`
	Code    string   `xml:"code"`
	Timeout *float64 `xml:"timeout"`
}

// Execute executes JavaScript in the browser session.
func (t *EvaluateTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Parse and validate input
	var input EvaluateInput
	if err := xml.Unmarshal(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if input.Session == "" {
		return "", nil, fmt.Errorf("session name is required")
	}

	if input.Code == "" {
		return "", nil, fmt.Errorf("JavaScript code is required")
	}

	// Get session
	session, err := t.manager.GetSession(input.Session)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Set timeout
	timeout := DefaultTimeout
	if input.Timeout != nil && *input.Timeout > 0 {
		timeout = *input.Timeout
	}

	// Execute JavaScript
	result, err := session.Page.Evaluate(input.Code, nil, timeout)
	if err != nil {
		return "", nil, fmt.Errorf("JavaScript execution failed: %w", err)
	}

	// Format result
	var resultStr string
	if result == nil {
		resultStr = "undefined"
	} else {
		// Try to format as JSON for structured data
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			// Fallback to string representation
			resultStr = fmt.Sprintf("%v", result)
		} else {
			resultStr = string(jsonBytes)
		}
	}

	output := fmt.Sprintf(`JavaScript Execution Complete

Session: %s
URL: %s

Result:
%s

The JavaScript code executed successfully in the browser context.`,
		input.Session,
		session.CurrentURL,
		resultStr,
	)

	return output, nil, nil
}

// IsLoopBreaking returns whether this tool breaks the agent loop.
func (t *EvaluateTool) IsLoopBreaking() bool {
	return false
}

// GeneratePreview implements the Previewable interface to show JavaScript evaluation details before execution.
func (t *EvaluateTool) GeneratePreview(ctx context.Context, argsXML []byte) (*tools.ToolPreview, error) {
	var input EvaluateInput
	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.Session == "" {
		return nil, fmt.Errorf("session name is required")
	}

	if input.Code == "" {
		return nil, fmt.Errorf("JavaScript code is required")
	}

	description := fmt.Sprintf("Execute JavaScript in browser session '%s'", input.Session)

	// Show code with reasonable truncation
	codePreview := input.Code
	if len(codePreview) > 200 {
		codePreview = codePreview[:200] + "..."
	}

	content := fmt.Sprintf("Session: %s\n\nJavaScript Code:\n%s", input.Session, codePreview)

	return &tools.ToolPreview{
		Type:        tools.PreviewTypeCommand,
		Title:       "Execute JavaScript",
		Description: description,
		Content:     content,
		Metadata: map[string]interface{}{
			"session": input.Session,
		},
	}, nil
}

// ShouldShow returns whether this tool should be visible.
// JavaScript execution is only shown when there are active sessions.
func (t *EvaluateTool) ShouldShow() bool {
	return t.manager.HasSessions()
}
