package prompts

import (
	"strings"
	"testing"
)

// mockToolMetadata is a test implementation of ToolMetadata interface
type mockToolMetadata struct {
	name        string
	description string
}

func (m *mockToolMetadata) GetName() string {
	return m.name
}

func (m *mockToolMetadata) GetDescription() string {
	return m.description
}

func TestFormatCustomToolsList(t *testing.T) {
	tests := []struct {
		name     string
		tools    []ToolMetadata
		expected []string // substrings that should appear in output
		empty    bool     // should return empty string
	}{
		{
			name:  "NoTools",
			tools: []ToolMetadata{},
			empty: true,
		},
		{
			name: "SingleTool",
			tools: []ToolMetadata{
				&mockToolMetadata{name: "web_search", description: "Search the web for information"},
			},
			expected: []string{
				"## Available Custom Tools",
				"**web_search**",
				"Search the web for information",
				"tool.yaml",
				"run_custom_tool",
			},
		},
		{
			name: "MultipleTools",
			tools: []ToolMetadata{
				&mockToolMetadata{name: "web_search", description: "Search the web"},
				&mockToolMetadata{name: "calculator", description: "Perform calculations"},
				&mockToolMetadata{name: "translator", description: "Translate text"},
			},
			expected: []string{
				"## Available Custom Tools",
				"**web_search**",
				"Search the web",
				"**calculator**",
				"Perform calculations",
				"**translator**",
				"Translate text",
				"tool.yaml",
				"run_custom_tool",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCustomToolsList(tt.tools)

			if tt.empty {
				if result != "" {
					t.Errorf("Expected empty string, got: %s", result)
				}
				return
			}

			for _, expected := range tt.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nGot: %s", expected, result)
				}
			}
		})
	}
}
