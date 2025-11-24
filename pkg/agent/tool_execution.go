package agent

import (
	"context"
	"fmt"
	"maps"

	"github.com/entrhq/forge/pkg/agent/prompts"
	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/tools/coding"
	"github.com/entrhq/forge/pkg/types"
)

// executeToolCall emits events, executes the tool, and handles execution errors
// Returns (result, metadata, shouldContinue, errorContext)
func (a *DefaultAgent) executeToolCall(ctx context.Context, tool tools.Tool, toolCall tools.ToolCall) (string, map[string]interface{}, bool, string) {
	// Emit tool call event - parse arguments to map for event emission
	argsMap, err := tools.XMLToMap(toolCall.GetArgumentsXML())
	if err != nil {
		// If parsing fails, emit empty map - the actual tool execution will handle the raw XML
		argsMap = make(map[string]interface{})
	}
	a.emitEvent(types.NewToolCallEvent(toolCall.ToolName, argsMap))

	// Inject event emitter and command registry into context for tools that support streaming events
	ctxWithEmitter := context.WithValue(ctx, coding.EventEmitterKey, coding.EventEmitter(a.emitEvent))
	ctxWithRegistry := context.WithValue(ctxWithEmitter, coding.CommandRegistryKey, &a.activeCommands)

	// Execute the tool
	result, metadata, toolErr := tool.Execute(ctxWithRegistry, toolCall.GetArgumentsXML())

	if toolErr != nil {
		a.emitEvent(types.NewToolResultErrorEvent(toolCall.ToolName, toolErr))
		errMsg := prompts.BuildErrorRecoveryMessage(prompts.ErrorRecoveryContext{
			Type:     prompts.ErrorTypeToolExecution,
			ToolName: toolCall.ToolName,
			Error:    toolErr,
		})

		// Track error and check circuit breaker
		if a.trackError(errMsg) {
			a.emitEvent(types.NewErrorEvent(fmt.Errorf("circuit breaker triggered: 5 consecutive tool execution errors")))
			return "", nil, false, ""
		}

		a.emitEvent(types.NewErrorEvent(fmt.Errorf("tool execution failed: %w", toolErr)))
		return "", nil, true, errMsg
	}

	return result, metadata, true, ""
}

// processToolResult handles successful tool execution results
// Returns (shouldContinue, errorContext)
func (a *DefaultAgent) processToolResult(tool tools.Tool, toolCall tools.ToolCall, result string, metadata map[string]interface{}) (bool, string) {
	event := types.NewToolResultEvent(toolCall.ToolName, result)
	// Add metadata to the event if present
	if len(metadata) > 0 {
		maps.Copy(event.Metadata, metadata)
	}
	a.emitEvent(event)

	// Success! Reset error tracking
	a.resetErrorTracking()

	// Check if this is a loop-breaking tool
	if tool.IsLoopBreaking() {
		return false, ""
	}

	// For non-breaking tools, add result to memory and continue loop
	a.memory.Add(types.NewUserMessage(fmt.Sprintf("Tool '%s' result:\n%s", toolCall.ToolName, result)))
	return true, ""
}

// handleToolApproval checks if tool requires approval and handles the approval flow
// Returns shouldExecute - false if approval was rejected/timed out, true otherwise
func (a *DefaultAgent) handleToolApproval(ctx context.Context, tool tools.Tool, toolCall tools.ToolCall) bool {
	// Check if tool requires approval
	previewable, ok := tool.(tools.Previewable)
	if !ok {
		// No approval needed - proceed with execution
		return true
	}

	// Generate preview
	preview, err := previewable.GeneratePreview(ctx, toolCall.GetArgumentsXML())
	if err != nil {
		// If preview generation fails, log error but continue with execution
		// (degraded mode - execute without approval)
		a.emitEvent(types.NewErrorEvent(fmt.Errorf("failed to generate preview for %s: %w", toolCall.ToolName, err)))
		return true
	}

	// Request approval from user
	approved, timedOut := a.requestApproval(ctx, toolCall, preview)

	if timedOut {
		// Timeout - treat as rejection and continue loop without executing
		errMsg := fmt.Sprintf("Tool approval request timed out after %v. The tool was not executed.", a.approvalTimeout)
		a.memory.Add(types.NewUserMessage(errMsg))
		return false
	}

	if !approved {
		// User rejected - continue loop without executing
		errMsg := fmt.Sprintf("Tool '%s' execution was rejected by user.", toolCall.ToolName)
		a.memory.Add(types.NewUserMessage(errMsg))
		return false
	}

	// User approved - continue with execution
	return true
}

// lookupTool retrieves a tool by name and handles lookup errors
// Returns (tool, shouldContinue, errorContext)
func (a *DefaultAgent) lookupTool(toolName string) (tools.Tool, bool, string) {
	tool, exists := a.getTool(toolName)
	if !exists {
		errMsg := prompts.BuildErrorRecoveryMessage(prompts.ErrorRecoveryContext{
			Type:           prompts.ErrorTypeUnknownTool,
			ToolName:       toolName,
			AvailableTools: a.getToolsList(),
		})

		// Track error and check circuit breaker
		if a.trackError(errMsg) {
			a.emitEvent(types.NewErrorEvent(fmt.Errorf("circuit breaker triggered: 5 consecutive unknown tool errors")))
			return nil, false, ""
		}

		a.emitEvent(types.NewErrorEvent(fmt.Errorf("unknown tool: %s", toolName)))
		return nil, true, errMsg
	}

	return tool, true, ""
}

// executeTool handles tool lookup, execution, and result processing
// Returns (shouldContinue, errorContext) following the same pattern as executeIteration
func (a *DefaultAgent) executeTool(ctx context.Context, toolCall tools.ToolCall) (bool, string) {
	// Look up the tool
	tool, shouldContinue, errCtx := a.lookupTool(toolCall.ToolName)
	if !shouldContinue || errCtx != "" {
		return shouldContinue, errCtx
	}

	// Handle tool approval if needed
	if !a.handleToolApproval(ctx, tool, toolCall) {
		// Tool approval was rejected or timed out - continue loop without executing
		return true, ""
	}

	// Execute the tool call
	result, metadata, shouldContinue, errCtx := a.executeToolCall(ctx, tool, toolCall)
	if !shouldContinue || errCtx != "" {
		return shouldContinue, errCtx
	}

	// Process the successful result
	return a.processToolResult(tool, toolCall, result, metadata)
}
