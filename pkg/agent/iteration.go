package agent

import (
	"context"
	"fmt"

	"github.com/entrhq/forge/pkg/agent/core"
	"github.com/entrhq/forge/pkg/agent/memory"
	"github.com/entrhq/forge/pkg/agent/prompts"
	"github.com/entrhq/forge/pkg/types"
)

// promptContext holds the prepared prompt and related metadata
type promptContext struct {
	messages     []*types.Message
	promptTokens int
}

// llmResponse holds the response from the LLM
type llmResponse struct {
	assistantContent string
	toolCallContent  string
	completionTokens int
}

// attemptSummarization tries to summarize the conversation if context manager is available
// Returns true if summarization occurred, false otherwise
func (a *DefaultAgent) attemptSummarization(ctx context.Context, promptTokens int) bool {
	// Early return if no context manager
	if a.contextManager == nil {
		return false
	}

	// Check if memory is the right type
	convMem, ok := a.memory.(*memory.ConversationMemory)
	if !ok {
		agentDebugLog.Printf("Memory is NOT ConversationMemory - type: %T", a.memory)
		return false
	}

	// Attempt summarization
	summarizedCount, err := a.contextManager.EvaluateAndSummarize(ctx, convMem, promptTokens)
	if err != nil {
		agentDebugLog.Printf("Failed to summarize conversation: %v", err)
		return false
	}

	// Check if anything was summarized
	if summarizedCount > 0 {
		agentDebugLog.Printf("Successfully summarized %d messages", summarizedCount)
		return true
	}

	return false
}

// preparePrompt builds the prompt, counts tokens, and handles context summarization
func (a *DefaultAgent) preparePrompt(ctx context.Context, errorContext string) *promptContext {
	systemMessages := a.buildSystemMessages()
	history := a.memory.GetAll()

	// Retrieved memories travel as an ephemeral trailing message rather than
	// as part of the system prompt: they change every turn, and placing them
	// after history keeps the system prompt and history byte-identical across
	// requests so providers can cache that prefix.
	memoryContext := a.retrieveMemories(ctx, history)

	messages := prompts.BuildMessages(systemMessages, history, "", memoryContext, errorContext)

	var promptTokens int
	if a.tokenizer != nil {
		promptTokens = a.tokenizer.CountMessagesTokens(messages)
		agentDebugLog.Printf("Prompt tokens before send: %d", promptTokens)
	}

	if summarized := a.attemptSummarization(ctx, promptTokens); summarized {
		history = a.memory.GetAll()
		messages = prompts.BuildMessages(systemMessages, history, "", memoryContext, errorContext)

		if a.tokenizer != nil {
			promptTokens = a.tokenizer.CountMessagesTokens(messages)
			agentDebugLog.Printf("Tokens after summarization: %d", promptTokens)
		}
	}

	return &promptContext{
		messages:     messages,
		promptTokens: promptTokens,
	}
}

// retrieveMemories returns long-term memories relevant to the current turn,
// or "" when retrieval is disabled or nothing matches.
func (a *DefaultAgent) retrieveMemories(ctx context.Context, history []*types.Message) string {
	if a.retrievalEngine == nil {
		return ""
	}

	a.cancelMu.Lock()
	turnID := a.currentTurnID
	a.cancelMu.Unlock()

	// The most recent user message anchors the HyDE retrieval window.
	var lastUserContent string
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == types.RoleUser {
			lastUserContent = history[i].Content
			break
		}
	}

	return a.retrievalEngine.RetrieveForTurn(ctx, turnID, history, lastUserContent)
}

// callLLM sends the request to the LLM and processes the streaming response
func (a *DefaultAgent) callLLM(ctx context.Context, pctx *promptContext) (*llmResponse, error) {
	// Emit API call start event with context information
	maxTokens := 0
	if a.contextManager != nil {
		maxTokens = a.contextManager.GetMaxTokens()
	}
	a.emitEvent(types.NewAPICallStartEvent("llm", pctx.promptTokens, maxTokens))

	// Get response from LLM
	stream, err := a.provider.StreamCompletion(ctx, pctx.messages)
	if err != nil {
		// Check if this is a context cancellation (user stopped the agent)
		if ctx.Err() != nil {
			return nil, ctx.Err() // Return context error for clean handling
		}
		// Terminal error - LLM/API failures should stop the loop
		a.emitEvent(types.NewErrorEvent(fmt.Errorf("failed to start completion: %w", err)))
		return nil, err
	}

	// Process stream and collect response
	var assistantContent string
	var toolCallContent string
	core.ProcessStream(stream, a.emitEvent, func(content, thinking, toolCall, role string) {
		assistantContent = content
		toolCallContent = toolCall
	})

	// Count completion tokens if tokenizer is available
	var completionTokens int
	if a.tokenizer != nil {
		fullResponse := assistantContent
		if toolCallContent != "" {
			fullResponse += toolCallContent
		}
		completionTokens = a.tokenizer.CountTokens(fullResponse)
	}

	return &llmResponse{
		assistantContent: assistantContent,
		toolCallContent:  toolCallContent,
		completionTokens: completionTokens,
	}, nil
}

// recordResponse handles token usage events and adds the response to memory
func (a *DefaultAgent) recordResponse(pctx *promptContext, resp *llmResponse) {
	// Emit token usage event if we have token counts
	if pctx.promptTokens > 0 || resp.completionTokens > 0 {
		totalTokens := pctx.promptTokens + resp.completionTokens
		a.emitEvent(types.NewTokenUsageEvent(pctx.promptTokens, resp.completionTokens, totalTokens))
	}

	// Add assistant's response to memory
	fullResponse := resp.assistantContent
	if resp.toolCallContent != "" {
		fullResponse += "<tool>" + resp.toolCallContent + "</tool>"
	}
	a.memory.Add(&types.Message{
		Role:    types.RoleAssistant,
		Content: fullResponse,
	})
}
