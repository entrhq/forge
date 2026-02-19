package browser

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateTool_Name(t *testing.T) {
	manager := NewSessionManager()
	tool := NewEvaluateTool(manager)
	assert.Equal(t, "browser_evaluate", tool.Name())
}

func TestEvaluateTool_Description(t *testing.T) {
	manager := NewSessionManager()
	tool := NewEvaluateTool(manager)
	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "JavaScript")
}

func TestEvaluateTool_Schema(t *testing.T) {
	manager := NewSessionManager()
	tool := NewEvaluateTool(manager)
	schema := tool.Schema()

	assert.NotNil(t, schema)
	assert.Contains(t, schema, "properties")

	props := schema["properties"].(map[string]interface{})
	assert.Contains(t, props, "session")
	assert.Contains(t, props, "code")
	assert.Contains(t, props, "timeout")

	required := schema["required"].([]string)
	assert.Contains(t, required, "session")
	assert.Contains(t, required, "code")
}

func TestEvaluateTool_Execute_ValidationErrors(t *testing.T) {
	manager := NewSessionManager()
	tool := NewEvaluateTool(manager)
	ctx := context.Background()

	tests := []struct {
		name        string
		input       EvaluateInput
		expectError string
	}{
		{
			name: "missing session",
			input: EvaluateInput{
				Code: "console.log('test')",
			},
			expectError: "session name is required",
		},
		{
			name: "missing code",
			input: EvaluateInput{
				Session: "test",
			},
			expectError: "JavaScript code is required",
		},
		{
			name: "session not found",
			input: EvaluateInput{
				Session: "nonexistent",
				Code:    "1 + 1",
			},
			expectError: "failed to get session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsXML, err := xml.Marshal(tt.input)
			require.NoError(t, err)

			_, _, err = tool.Execute(ctx, argsXML)
			assert.Error(t, err)
			if tt.expectError != "" {
				assert.Contains(t, err.Error(), tt.expectError)
			}
		})
	}
}

func TestEvaluateTool_Execute_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	manager := NewSessionManager()
	tool := NewEvaluateTool(manager)
	ctx := context.Background()

	// Initialize Playwright
	err := manager.Initialize()
	require.NoError(t, err)
	defer manager.Shutdown()

	// Create a session
	session, err := manager.StartSession("test", SessionOptions{
		Headless: true,
		Viewport: &Viewport{Width: 1280, Height: 720},
	})
	require.NoError(t, err)
	defer manager.CloseSession("test")

	// Navigate to a page
	_, err = session.Page.Goto("about:blank", playwright.PageGotoOptions{})
	require.NoError(t, err)

	tests := []struct {
		name     string
		code     string
		contains string
	}{
		{
			name:     "simple expression",
			code:     "1 + 1",
			contains: "2",
		},
		{
			name:     "get document title",
			code:     "document.title",
			contains: "JavaScript Execution Complete",
		},
		{
			name:     "return object",
			code:     "({foo: 'bar', num: 42})",
			contains: "foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := EvaluateInput{
				Session: "test",
				Code:    tt.code,
			}

			argsXML, err := xml.Marshal(input)
			require.NoError(t, err)

			result, _, err := tool.Execute(ctx, argsXML)
			require.NoError(t, err)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestEvaluateTool_GeneratePreview(t *testing.T) {
	manager := NewSessionManager()
	tool := NewEvaluateTool(manager)

	tests := []struct {
		name          string
		input         EvaluateInput
		expectedTitle string
		contains      []string
	}{
		{
			name: "simple code",
			input: EvaluateInput{
				Session: "test",
				Code:    "document.title",
			},
			expectedTitle: "Execute JavaScript",
			contains:      []string{"test", "document.title"},
		},
		{
			name: "code with timeout",
			input: EvaluateInput{
				Session: "test",
				Code:    "document.querySelector('h1').textContent",
			},
			expectedTitle: "Execute JavaScript",
			contains:      []string{"test", "document.querySelector"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsXML, err := xml.Marshal(tt.input)
			require.NoError(t, err)

			preview, err := tool.GeneratePreview(context.Background(), argsXML)
			require.NoError(t, err)
			assert.NotNil(t, preview)
			assert.Equal(t, tt.expectedTitle, preview.Title)

			for _, substr := range tt.contains {
				found := strings.Contains(preview.Description, substr) || strings.Contains(preview.Content, substr)
				assert.True(t, found, "expected to find %q in preview", substr)
			}
		})
	}
}
