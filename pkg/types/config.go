package types

import "time"

// ModelInfo contains information about an LLM model.
type ModelInfo struct {
	// Metadata holds optional additional information about the model.
	Metadata map[string]any

	// Name is the model identifier (e.g., "gpt-4", "claude-3-opus").
	Name string

	// Provider is the name of the LLM provider (e.g., "openai", "anthropic").
	Provider string

	// MaxTokens is the maximum context window size for this model.
	MaxTokens int

	// SupportsStreaming indicates if the model supports streaming responses.
	SupportsStreaming bool
}

// AgentConfig holds configuration for an agent instance.
type AgentConfig struct {
	// Metadata holds optional additional configuration.
	Metadata map[string]any

	// SystemPrompt is the initial system message that sets the agent's behavior.
	SystemPrompt string

	// Timeout is the maximum duration for the agent to run. 0 means no timeout.
	Timeout time.Duration

	// MaxTurns limits the number of conversation turns. 0 means unlimited.
	MaxTurns int

	// BufferSize sets the size of internal channels for message passing.
	// Defaults to 10 if not set.
	BufferSize int

	// EnableStreaming enables streaming responses from the LLM.
	EnableStreaming bool
}

// NewAgentConfig creates a new AgentConfig with sensible defaults.
func NewAgentConfig() *AgentConfig {
	return &AgentConfig{
		MaxTurns:        0, // unlimited
		Timeout:         0, // no timeout
		EnableStreaming: true,
		BufferSize:      10,
		Metadata:        make(map[string]any),
	}
}

// WithSystemPrompt sets the system prompt and returns the config for chaining.
func (c *AgentConfig) WithSystemPrompt(prompt string) *AgentConfig {
	c.SystemPrompt = prompt
	return c
}

// WithMaxTurns sets the maximum turns and returns the config for chaining.
func (c *AgentConfig) WithMaxTurns(maxTurns int) *AgentConfig {
	c.MaxTurns = maxTurns
	return c
}

// WithTimeout sets the timeout and returns the config for chaining.
func (c *AgentConfig) WithTimeout(timeout time.Duration) *AgentConfig {
	c.Timeout = timeout
	return c
}

// WithStreaming sets streaming enabled/disabled and returns the config for chaining.
func (c *AgentConfig) WithStreaming(enabled bool) *AgentConfig {
	c.EnableStreaming = enabled
	return c
}

// WithBufferSize sets the buffer size and returns the config for chaining.
func (c *AgentConfig) WithBufferSize(size int) *AgentConfig {
	c.BufferSize = size
	return c
}

// WithMetadata adds metadata to the config and returns the config for chaining.
func (c *AgentConfig) WithMetadata(key string, value any) *AgentConfig {
	if c.Metadata == nil {
		c.Metadata = make(map[string]any)
	}
	c.Metadata[key] = value
	return c
}
