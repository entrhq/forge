package tools

import "context"

// ToolResult represents the result of a tool execution with optional metadata.
type ToolResult struct {
	Output   string                 // The main output/result message
	Metadata map[string]interface{} // Optional metadata about the execution
}

// MetadataProvider is an optional interface that tools can implement to return
// structured metadata along with their execution result. This metadata can be
// used for tracking, analytics, or other purposes.
//
// For example, file-modifying tools can return line change information:
//
//	return &ToolResult{
//	    Output: "File modified successfully",
//	    Metadata: map[string]interface{}{
//	        "lines_added": 10,
//	        "lines_removed": 5,
//	    },
//	}
type MetadataProvider interface {
	Tool
	// ExecuteWithMetadata runs the tool and returns both output and metadata
	ExecuteWithMetadata(ctx context.Context, argumentsXML []byte) (*ToolResult, error)
}
