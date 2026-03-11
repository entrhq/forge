package prompts

import (
	"strings"
	"testing"
)

func TestGenerateXMLExample(t *testing.T) {
	t.Run("SimpleStringParameter", func(t *testing.T) {
		schema := map[string]any{
			"properties": map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "A simple message",
				},
			},
			"required": []string{"message"},
		}

		result := GenerateXMLExample(schema, "test_tool")

		if !strings.Contains(result, "<tool_name>test_tool</tool_name>") {
			t.Error("Expected tool_name in result")
		}
		if !strings.Contains(result, "<message>") {
			t.Error("Expected message parameter in result")
		}
	})

	t.Run("EntityEscapingForContentFields", func(t *testing.T) {
		schema := map[string]any{
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "File content",
				},
			},
			"required": []string{"content"},
		}

		result := GenerateXMLExample(schema, "write_file")

		// Per ADR-0024, we prefer XML entity escaping over CDATA
		if !strings.Contains(result, "&amp;") {
			t.Error("Expected entity escaping for content field (per ADR-0024)")
		}
	})

	t.Run("NestedArrayOfObjects", func(t *testing.T) {
		schema := map[string]any{
			"properties": map[string]any{
				"edits": map[string]any{
					"type":        "array",
					"description": "List of edits",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"search": map[string]any{
								"type":        "string",
								"description": "Search text",
							},
							"replace": map[string]any{
								"type":        "string",
								"description": "Replace text",
							},
						},
					},
				},
			},
			"required": []string{"edits"},
		}

		result := GenerateXMLExample(schema, "apply_diff")

		// Should have nested structure
		if !strings.Contains(result, "<edits>") {
			t.Error("Expected edits array")
		}
		if !strings.Contains(result, "<edit>") {
			t.Error("Expected edit element (singular)")
		}
		if !strings.Contains(result, "<search>") {
			t.Error("Expected search element")
		}
		if !strings.Contains(result, "<replace>") {
			t.Error("Expected replace element")
		}

		// Should NOT have CDATA wrapping the entire edits structure
		if strings.Contains(result, "<edits><![CDATA[") {
			t.Error("Should NOT wrap array structure in CDATA")
		}
	})

	t.Run("BooleanParameter", func(t *testing.T) {
		schema := map[string]any{
			"properties": map[string]any{
				"recursive": map[string]any{
					"type":        "boolean",
					"description": "Whether to recurse",
				},
			},
			"required": []string{"recursive"},
		}

		result := GenerateXMLExample(schema, "list_files")

		if !strings.Contains(result, "<recursive>true</recursive>") {
			t.Error("Expected boolean true value")
		}
	})

	t.Run("NumericParameters", func(t *testing.T) {
		schema := map[string]any{
			"properties": map[string]any{
				"count": map[string]any{
					"type":        "integer",
					"description": "Count value",
				},
				"ratio": map[string]any{
					"type":        "number",
					"description": "Ratio value",
				},
			},
			"required": []string{"count", "ratio"},
		}

		result := GenerateXMLExample(schema, "test_tool")

		if !strings.Contains(result, "<count>42</count>") {
			t.Error("Expected integer example")
		}
		if !strings.Contains(result, "<ratio>3.14</ratio>") {
			t.Error("Expected float example")
		}
	})
}
