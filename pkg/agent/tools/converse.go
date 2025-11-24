package tools

import (
	"context"
	"encoding/xml"
	"fmt"
)

// ConverseTool is a loop-breaking tool that allows the agent to engage
// in casual conversation or provide information without completing a task.
// This is useful when the user's input is conversational rather than
// task-oriented.
type ConverseTool struct{}

// NewConverseTool creates a new converse tool
func NewConverseTool() *ConverseTool {
	return &ConverseTool{}
}

// Name returns the tool's identifier
func (t *ConverseTool) Name() string {
	return "converse"
}

// Description returns a description of what this tool does
func (t *ConverseTool) Description() string {
	return "Engage in conversation or provide information to the user without completing a task. " +
		"Use this for casual interactions, answering questions, or when the user's input " +
		"doesn't require task completion. The message should be conversational and helpful."
}

// Schema returns the JSON schema for the tool's arguments
func (t *ConverseTool) Schema() map[string]interface{} {
	return BaseToolSchema(
		map[string]interface{}{
			"message": map[string]interface{}{
				"type":        "string",
				"description": "A conversational message to share with the user. Can include information, explanations, or casual responses.",
			},
		},
		[]string{"message"},
	)
}

// Execute runs the tool and returns the conversational message
func (t *ConverseTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var args struct {
		XMLName xml.Name `xml:"arguments"`
		Message string   `xml:"message"`
	}

	if err := UnmarshalXMLWithFallback(argsXML, &args); err != nil {
		return "", nil, fmt.Errorf("invalid arguments for converse: %w", err)
	}

	if args.Message == "" {
		return "", nil, fmt.Errorf("message cannot be empty")
	}

	return args.Message, nil, nil
}

// IsLoopBreaking returns true because this tool terminates the agent loop
func (t *ConverseTool) IsLoopBreaking() bool {
	return true
}
