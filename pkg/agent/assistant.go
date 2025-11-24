package agent

import (
	"context"

	"github.com/entrhq/forge/pkg/types"
)

// runAgentLoop executes the agent loop with tools and thinking
// The loop continues until a loop-breaking tool is used or circuit breaker triggers
func (a *DefaultAgent) runAgentLoop(ctx context.Context) {
	var errorContext string

	for {
		// Check if context was canceled (e.g., via /stop command)
		select {
		case <-ctx.Done():
			// Context canceled - stop the agent loop
			// Emit a user-friendly message about the cancellation
			a.memory.Add(types.NewUserMessage("Operation stopped by user."))
			return
		default:
			// Continue with iteration
		}

		// Execute one iteration with optional error context from previous iteration
		shouldContinue, nextErrorContext := a.executeIteration(ctx, errorContext)
		if !shouldContinue {
			// Loop-breaking tool was used or circuit breaker triggered
			return
		}

		// Update error context for next iteration
		errorContext = nextErrorContext
	}
}

// executeIteration performs a single iteration of the agent loop
// Returns (shouldContinue, errorContext) where:
//   - shouldContinue: false means loop should break (loop-breaking tool used or circuit breaker)
//   - errorContext: message to inject as user context for error recovery (empty if no error)
func (a *DefaultAgent) executeIteration(ctx context.Context, errorContext string) (bool, string) {
	// Step 1: Prepare prompt with summarization if needed
	pctx := a.preparePrompt(ctx, errorContext)

	// Step 2: Call LLM and get streaming response
	resp, err := a.callLLM(ctx, pctx)
	if err != nil {
		// Context cancellation - stop silently
		if ctx.Err() != nil {
			return false, ""
		}
		// LLM error already emitted in callLLM
		return false, ""
	}

	// Step 3: Record response (emit tokens, add to memory)
	a.recordResponse(pctx, resp)

	// Step 4: Process the tool call (parse, validate, execute)
	return a.processToolCall(ctx, resp.toolCallContent)
}

// emitEvent sends an event on the event channel.
// This is a blocking send to ensure critical events like TurnEnd are not dropped.
// It safely handles the case where the event channel may be closed during shutdown.
func (a *DefaultAgent) emitEvent(event *types.AgentEvent) {
	defer func() {
		if r := recover(); r != nil {
			// Event channel was closed during shutdown - this is expected
			// Log at debug level only as this is a normal shutdown scenario
		}
	}()
	a.channels.Event <- event
}
