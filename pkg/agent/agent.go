// Package agent provides the core agent interface and DefaultAgent implementation
// for the Forge agent framework.
//
// The DefaultAgent is available directly from this package for simple usage:
//
//	import "github.com/entrhq/forge/pkg/agent"
//	ag := agent.NewDefaultAgent(provider, agent.WithSystemPrompt("..."))
//
// The package is organized with subpackages for specialized functionality:
//   - core: Internal stream processing utilities
//   - memory: Conversation history and context management (planned)
//   - tools: Tool/function calling system (planned)
//   - middleware: Event hooks and cross-cutting concerns (planned)
//   - orchestration: Multi-agent coordination and workflows (planned)
package agent

import (
	"context"

	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/types"
)

// Agent interface defines the core capabilities of a Forge agent.
// Agents are async event-driven components that process messages through
// an LLM provider and communicate via channels.
type Agent interface {
	// Start begins the agent's event loop in a goroutine.
	// The agent will listen for messages on its input channel and process them
	// asynchronously, sending responses to the output channel.
	//
	// The agent runs until:
	// - The context is canceled
	// - The shutdown channel is closed
	// - An unrecoverable error occurs
	//
	// Returns an error if the agent fails to start, otherwise returns nil
	// and continues running asynchronously.
	Start(ctx context.Context) error

	// Shutdown gracefully stops the agent.
	// This method signals the agent to stop processing new messages and
	// complete any in-flight operations before shutting down.
	//
	// Returns when the agent has fully stopped or the context is canceled.
	Shutdown(ctx context.Context) error

	// GetChannels returns the communication channels for this agent.
	// The executor uses these channels to send input and receive output.
	GetChannels() *types.AgentChannels

	// GetTool retrieves a specific tool by name from the agent's tool registry.
	// Returns nil if the tool is not found.
	GetTool(name string) interface{}

	// GetTools returns a list of all available tools registered with the agent.
	// This includes both built-in tools and any custom tools that have been registered.
	GetTools() []interface{}

	// GetContextInfo returns detailed context information for debugging and display.
	// This includes system prompt length, tool count, message history, and token usage.
	GetContextInfo() *ContextInfo

	// SetProvider updates the LLM provider used by the agent.
	// This allows hot-reloading of provider configuration without restarting the agent.
	// The update is thread-safe and will take effect on the next agent iteration.
	SetProvider(provider llm.Provider) error
}

// ContextInfo contains detailed agent context statistics
type ContextInfo struct {
	// System prompt
	SystemPromptTokens      int
	CustomInstructions      bool
	RepositoryContextTokens int

	// Tool system
	ToolCount  int
	ToolTokens int
	ToolNames  []string

	// Message history
	MessageCount       int
	ConversationTurns  int
	ConversationTokens int

	// History composition breakdown
	RawMessageCount      int // Regular (unsummarized) user/assistant/tool messages
	RawMessageTokens     int
	SummaryBlockCount    int // [SUMMARIZED] all non-goal-batch summarized blocks
	SummaryBlockTokens   int
	GoalBatchBlockCount  int // [GOAL BATCH] blocks from goal-batch compaction
	GoalBatchBlockTokens int

	// Token usage - current context
	CurrentContextTokens int
	MaxContextTokens     int
	FreeTokens           int
	UsagePercent         float64

	// Token usage - cumulative across all API calls
	TotalPromptTokens     int
	TotalCompletionTokens int
	TotalTokens           int
}
