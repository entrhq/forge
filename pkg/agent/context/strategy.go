package context

import (
	"context"

	"github.com/entrhq/forge/pkg/agent/memory"
	"github.com/entrhq/forge/pkg/llm"
)

// Strategy defines the interface for context summarization strategies.
// Each strategy implements a specific approach to reducing context size
// while preserving semantic meaning.
type Strategy interface {
	// Name returns the strategy's identifier for logging and debugging.
	Name() string

	// ShouldRun evaluates whether this strategy should execute on this turn.
	// It receives the current conversation memory, current token count, and max allowed tokens.
	// Returns true if the strategy should be applied.
	ShouldRun(conv *memory.ConversationMemory, currentTokens, maxTokens int) bool

	// Summarize performs the actual summarization operation.
	// It receives a context (for cancellation), the conversation memory to summarize,
	// and an LLM provider for generating summaries.
	// Returns the number of messages (or groups) summarized and any error. The conversation is modified in place.
	Summarize(ctx context.Context, conv *memory.ConversationMemory, llm llm.Provider) (int, error)
}
