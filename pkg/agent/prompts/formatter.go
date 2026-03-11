package prompts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/agent/tools"
)

// FormatToolSchema converts a tool's schema into a human-readable description
// for inclusion in the system prompt.
func FormatToolSchema(tool tools.Tool) string {
	var builder strings.Builder

	// Tool name and description
	fmt.Fprintf(&builder, "## %s\n\n", tool.Name())
	fmt.Fprintf(&builder, "%s\n\n", tool.Description())

	// Schema details
	schema := tool.Schema()

	// Extract properties if they exist
	properties, ok := schema["properties"].(map[string]any)
	if ok && len(properties) > 0 {
		builder.WriteString("**Parameters:**\n\n")

		// Get required fields if specified
		requiredFields := make(map[string]bool)
		if req, ok := schema["required"].([]string); ok {
			for _, field := range req {
				requiredFields[field] = true
			}
		}

		// Format each property
		for propName, propValue := range properties {
			propMap, ok := propValue.(map[string]any)
			if !ok {
				continue
			}

			// Mark if required
			required := ""
			if requiredFields[propName] {
				required = " (required)"
			}

			// Get type and description
			propType := "any"
			if t, ok := propMap["type"].(string); ok {
				propType = t
			}

			propDesc := ""
			if d, ok := propMap["description"].(string); ok {
				propDesc = d
			}

			fmt.Fprintf(&builder, "- `%s` (%s)%s: %s\n",
				propName, propType, required, propDesc)
		}
		builder.WriteString("\n")
	}

	// Loop-breaking indicator
	if tool.IsLoopBreaking() {
		builder.WriteString("*This is a loop-breaking tool - using it will end the current turn.*\n\n")
	}

	// Example usage with XML format
	builder.WriteString("**Example:**\n```xml\n")

	// Check if tool provides custom XML example
	if provider, ok := tool.(XMLExampleProvider); ok {
		builder.WriteString(provider.XMLExample())
	} else {
		// Auto-generate from schema
		builder.WriteString(GenerateXMLExample(schema, tool.Name()))
	}

	builder.WriteString("\n```\n\n")

	return builder.String()
}

// FormatToolSchemas formats multiple tools into a comprehensive tools section
func FormatToolSchemas(toolsList []tools.Tool) string {
	if len(toolsList) == 0 {
		return "No tools available."
	}

	var builder strings.Builder
	builder.WriteString("# AVAILABLE TOOLS\n\n")

	for i, tool := range toolsList {
		builder.WriteString(FormatToolSchema(tool))
		// Add separator between tools (except for the last one)
		if i < len(toolsList)-1 {
			builder.WriteString("---\n\n")
		}
	}

	return builder.String()
}

// FormatToolForLLM creates a JSON schema representation suitable for LLM providers
// that support native tool calling (future enhancement)
func FormatToolForLLM(tool tools.Tool) map[string]any {
	return map[string]any{
		"name":        tool.Name(),
		"description": tool.Description(),
		"parameters":  tool.Schema(),
	}
}

// SchemaToJSON converts a schema map to a pretty-printed JSON string
func SchemaToJSON(schema map[string]any) (string, error) {
	jsonBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}
	return string(jsonBytes), nil
}
