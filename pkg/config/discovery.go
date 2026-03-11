package config

import (
	"fmt"
	"reflect"
)

// DiscoverToolsFromAgent initializes the auto-approval section with tools from the agent
// This should be called after the agent is created and tools are registered
func DiscoverToolsFromAgent(agent any) error {
	if !IsInitialized() {
		return fmt.Errorf("config not initialized")
	}

	// Get tools from agent using the GetTools interface
	type toolGetter interface {
		GetTools() []any
	}

	getter, ok := agent.(toolGetter)
	if !ok {
		return fmt.Errorf("agent does not implement GetTools() method")
	}

	tools := getter.GetTools()
	if len(tools) == 0 {
		return nil // No tools to discover
	}

	// Get the auto-approval section
	section, exists := globalManager.GetSection("auto_approval")
	if !exists {
		return fmt.Errorf("auto-approval section not found")
	}

	autoApproval, ok := section.(*AutoApprovalSection)
	if !ok {
		return fmt.Errorf("auto-approval section has wrong type")
	}

	// Extract tool names for tools that require approval (implement Previewable)
	// Tools that don't require approval (like task_completion, ask_question) are excluded
	type toolNamer interface {
		Name() string
	}

	for _, tool := range tools {
		var toolName string
		if namer, ok := tool.(toolNamer); ok {
			toolName = namer.Name()
		}

		// Use reflection to check if the tool has a GeneratePreview method
		// This is necessary because the tools are returned as []interface{} which loses type information
		toolType := reflect.TypeOf(tool)

		// Check if the type has a GeneratePreview method
		method, hasMethod := toolType.MethodByName("GeneratePreview")
		if !hasMethod {
			continue
		}

		// Verify the method signature matches what we expect
		// GeneratePreview(ctx context.Context, arguments json.RawMessage) (*tools.ToolPreview, error)
		methodType := method.Type
		// NumIn() == 3: receiver + 2 params (context.Context, json.RawMessage)
		// NumOut() == 2: pointer return value + error
		if methodType.NumIn() != 3 || methodType.NumOut() != 2 {
			continue
		}

		// Tool implements Previewable interface, add it to config
		if toolName != "" {
			autoApproval.EnsureToolExists(toolName)
		}
	}

	// Save the updated configuration
	return globalManager.SaveAll()
}
