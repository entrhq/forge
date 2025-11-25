package tools

import (
	"context"
	"encoding/xml"
	"fmt"
)

const taskCompletionToolName = "task_completion"

// TaskCompletionTool is a loop-breaking tool that allows the agent to signal
// that it has completed the user's task. This tool should be used when the
// agent has finished all work and wants to present the final result to the user.
type TaskCompletionTool struct{}

// NewTaskCompletionTool creates a new task completion tool
func NewTaskCompletionTool() *TaskCompletionTool {
	return &TaskCompletionTool{}
}

// Name returns the tool's identifier
func (t *TaskCompletionTool) Name() string {
	return taskCompletionToolName
}

// Description returns a description of what this tool does
func (t *TaskCompletionTool) Description() string {
	return "Signal that the task is complete and present the final result to the user. " +
		"Use this when you have finished all work and want to show the outcome. " +
		"The result should be comprehensive and not require further input from the user."
}

// Schema returns the JSON schema for the tool's arguments
func (t *TaskCompletionTool) Schema() map[string]interface{} {
	return BaseToolSchema(
		map[string]interface{}{
			"result": map[string]interface{}{
				"type":        "string",
				"description": "The final result of the task. Should be clear, complete, and not end with questions or offers for further assistance.",
			},
		},
		[]string{"result"},
	)
}

// Execute runs the tool and returns the result
func (t *TaskCompletionTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var args struct {
		XMLName xml.Name `xml:"arguments"`
		Result  string   `xml:"result"`
	}

	if err := UnmarshalXMLWithFallback(argsXML, &args); err != nil {
		return "", nil, fmt.Errorf("invalid arguments for %s: %w", taskCompletionToolName, err)
	}

	if args.Result == "" {
		return "", nil, fmt.Errorf("result cannot be empty")
	}

	// Return the result - this will be presented to the user
	return args.Result, nil, nil
}

// IsLoopBreaking returns true because this tool terminates the agent loop
func (t *TaskCompletionTool) IsLoopBreaking() bool {
	return true
}
