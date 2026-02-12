package prompts

import (
	"fmt"
	"strings"
)

// ToolMetadata represents metadata for a custom tool
// This is a minimal interface to avoid circular dependencies
type ToolMetadata interface {
	GetName() string
	GetDescription() string
}

// FormatCustomToolsList creates a formatted list of available custom tools
// for inclusion in the system prompt
func FormatCustomToolsList(tools []ToolMetadata) string {
	if len(tools) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## Available Custom Tools\n\n")
	builder.WriteString("The following custom tools are currently available:\n\n")

	for _, tool := range tools {
		builder.WriteString(fmt.Sprintf("- **%s**: %s\n", tool.GetName(), tool.GetDescription()))
	}

	builder.WriteString("\nTo see full parameter details for any custom tool, read its tool.yaml file at ~/.forge/tools/<tool-name>/tool.yaml using the read_file tool.\n")
	builder.WriteString("\nUse the run_custom_tool built-in tool to execute any of these custom tools.")

	return builder.String()
}
