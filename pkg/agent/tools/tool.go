package tools

import (
	"context"
	"encoding/xml"
)

// Tool represents a capability that an agent can use during execution.
// Tools are invoked by the LLM through XML-formatted tool calls and can
// perform actions like task completion, asking questions, or custom operations.
//
// Example tool call format from LLM:
//
//	<tool>
//	<server_name>local</server_name>
//	<tool_name>task_completion</tool_name>
//	<arguments>
//	  <result>Task completed successfully</result>
//	</arguments>
//	</tool>
type Tool interface {
	// Name returns the unique identifier for this tool (e.g., "task_completion")
	Name() string

	// Description returns a human-readable description of what this tool does
	Description() string

	// Schema returns the JSON schema for this tool's input parameters
	// The schema should be a valid JSON Schema object defining the structure
	// of the arguments that this tool accepts
	Schema() map[string]interface{}

	// Execute runs the tool with the given XML arguments and returns a result string
	// The arguments should be unmarshaled from XML into the tool's argument struct
	// Returns: (result string, metadata map, error)
	// Metadata is optional and can be nil - it will be included in tool result events
	Execute(ctx context.Context, argumentsXML []byte) (string, map[string]interface{}, error)

	// IsLoopBreaking indicates whether this tool should terminate the agent loop
	// Loop-breaking tools (like task_completion, ask_question, converse) will
	// cause the agent to stop iterating and return control to the user
	IsLoopBreaking() bool
}

// ToolCall represents a parsed tool invocation from the LLM's response
type ToolCall struct {
	XMLName    xml.Name       `xml:"tool"`
	ServerName string         `xml:"server_name"`
	ToolName   string         `xml:"tool_name"`
	Arguments  ArgumentsBlock `xml:"arguments"`
}

// ArgumentsBlock holds the raw XML of the arguments element
type ArgumentsBlock struct {
	InnerXML []byte `xml:",innerxml"`
}

// GetArgumentsXML returns the arguments wrapped in <arguments> tags for unmarshaling.
// Uses efficient byte slice operations to avoid multiple string allocations.
func (tc *ToolCall) GetArgumentsXML() []byte {
	const prefix = "<arguments>"
	const suffix = "</arguments>"

	// Pre-allocate exact size needed
	result := make([]byte, 0, len(prefix)+len(tc.Arguments.InnerXML)+len(suffix))
	result = append(result, []byte(prefix)...)
	result = append(result, tc.Arguments.InnerXML...)
	result = append(result, []byte(suffix)...)
	return result
}

// Previewable is an optional interface that tools can implement to provide
// a preview of their changes before execution. This enables the approval flow
// where users can review and approve/reject tool actions.
type Previewable interface {
	// GeneratePreview creates a preview of what this tool will do with the given arguments.
	// Returns a ToolPreview containing the preview data and metadata.
	GeneratePreview(ctx context.Context, argumentsXML []byte) (*ToolPreview, error)
}

// ToolPreview represents a preview of what a tool will do.
// It contains enough information to show the user what changes will be made.
type ToolPreview struct {
	// Type indicates the kind of preview (diff, command, file_write, etc.)
	Type PreviewType

	// Title is a short description of the action
	Title string

	// Description provides additional context about the action
	Description string

	// Content contains the preview data (diff text, command to run, etc.)
	Content string

	// Metadata holds additional preview information (file path, language, etc.)
	Metadata map[string]interface{}
}

// PreviewType indicates the kind of preview being shown
type PreviewType string

const (
	// PreviewTypeDiff represents a file diff preview
	PreviewTypeDiff PreviewType = "diff"

	// PreviewTypeCommand represents a command execution preview
	PreviewTypeCommand PreviewType = "command"

	// PreviewTypeFileWrite represents a file write/creation preview
	PreviewTypeFileWrite PreviewType = "file_write"
)

// BaseToolSchema creates a common JSON schema structure for a tool
// with the given properties and required fields
func BaseToolSchema(properties map[string]interface{}, required []string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
