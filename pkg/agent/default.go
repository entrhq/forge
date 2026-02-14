package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/entrhq/forge/pkg/agent/approval"
	agentcontext "github.com/entrhq/forge/pkg/agent/context"
	"github.com/entrhq/forge/pkg/agent/memory"
	"github.com/entrhq/forge/pkg/agent/memory/notes"
	"github.com/entrhq/forge/pkg/agent/prompts"
	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/llm/tokenizer"
	"github.com/entrhq/forge/pkg/logging"
	"github.com/entrhq/forge/pkg/tools/browser"
	"github.com/entrhq/forge/pkg/types"
)

var agentDebugLog *logging.Logger

func init() {
	var err error
	agentDebugLog, err = logging.NewLogger("agent")
	if err != nil {
		// Logger fell back to stderr due to initialization failure
		agentDebugLog.Warnf("Failed to initialize agent logger, using stderr fallback: %v", err)
	}
}

// DefaultAgent is a basic implementation of the Agent interface.
// It processes user inputs through an LLM provider using an agent loop
// with tools, thinking, and memory management.
type DefaultAgent struct {
	provider           llm.Provider
	channels           *types.AgentChannels
	customInstructions string
	repositoryContext  string
	maxTurns           int
	bufferSize         int
	metadata           map[string]interface{}

	// Agent loop components
	tools         map[string]tools.Tool
	toolsMu       sync.RWMutex
	disabledTools map[string]bool // Tools to exclude from registration
	memory        memory.Memory

	// Approval system
	approvalManager *approval.Manager
	approvalTimeout time.Duration

	// Control channels
	cancelMu     sync.Mutex
	cancelStream context.CancelFunc

	// Command execution tracking
	activeCommands sync.Map // executionID -> context.CancelFunc

	// Running state
	running bool
	runMu   sync.Mutex

	// Error recovery state
	lastErrors [5]string // Ring buffer of last 5 error messages
	errorIndex int       // Current position in ring buffer

	// Token usage tracking
	tokenizer *tokenizer.Tokenizer

	// Context management
	contextManager *agentcontext.Manager

	// Notes management
	notesManager *notes.Manager

	// Browser session management
	browserManager *browser.SessionManager

}

// AgentOption is a function that configures an agent
type AgentOption func(*DefaultAgent)

// WithCustomInstructions sets custom instructions for the agent
// These are user-provided instructions that will be added to the system prompt
func WithCustomInstructions(instructions string) AgentOption {
	return func(a *DefaultAgent) {
		a.customInstructions = instructions
	}
}

// WithRepositoryContext sets repository-specific context from AGENTS.md
// This is separate from custom instructions and represents project-specific information
func WithRepositoryContext(context string) AgentOption {
	return func(a *DefaultAgent) {
		a.repositoryContext = context
	}
}

// WithMaxTurns sets the maximum number of conversation turns
func WithMaxTurns(max int) AgentOption {
	return func(a *DefaultAgent) {
		a.maxTurns = max
	}
}

// WithNotesManager sets a custom notes manager for the agent
// If not provided, a new manager will be created
func WithNotesManager(manager *notes.Manager) AgentOption {
	return func(a *DefaultAgent) {
		a.notesManager = manager
	}
}

// WithBrowserManager sets a browser session manager for the agent
func WithBrowserManager(manager *browser.SessionManager) AgentOption {
	return func(a *DefaultAgent) {
		a.browserManager = manager
	}
}

// WithBufferSize sets the channel buffer size
func WithBufferSize(size int) AgentOption {
	return func(a *DefaultAgent) {
		a.bufferSize = size
	}
}

// WithMetadata sets metadata for the agent
func WithMetadata(metadata map[string]interface{}) AgentOption {
	return func(a *DefaultAgent) {
		a.metadata = metadata
	}
}

// WithApprovalTimeout sets the timeout for approval requests
func WithApprovalTimeout(timeout time.Duration) AgentOption {
	return func(a *DefaultAgent) {
		a.approvalTimeout = timeout
	}
}

// WithContextManager sets a context manager for the agent to handle context summarization
func WithContextManager(manager *agentcontext.Manager) AgentOption {
	return func(a *DefaultAgent) {
		a.contextManager = manager
	}
}

// WithDisabledTools returns an option to disable specific built-in tools
// This is useful for headless mode where interactive tools should be disabled
func WithDisabledTools(toolNames ...string) AgentOption {
	return func(a *DefaultAgent) {
		if a.disabledTools == nil {
			a.disabledTools = make(map[string]bool)
		}
		for _, name := range toolNames {
			a.disabledTools[name] = true
		}
	}
}

// NewDefaultAgent creates a new DefaultAgent with the given provider and options.
func NewDefaultAgent(provider llm.Provider, opts ...AgentOption) *DefaultAgent {
	// Create tokenizer for client-side token counting
	tok, err := tokenizer.New()
	if err != nil {
		// Fall back to nil tokenizer if initialization fails
		tok = nil
	}

	a := &DefaultAgent{
		provider:   provider,
		bufferSize: 10, // default buffer size
		tools:      make(map[string]tools.Tool),
		memory:     memory.NewConversationMemory(),
		tokenizer:  tok,
	}

	// Register built-in tools
	a.RegisterDefaultTools()

	// Apply options
	for _, opt := range opts {
		opt(a)
	}

	// Initialize notes manager if not provided via option
	if a.notesManager == nil {
		a.notesManager = notes.NewManager()
	}

	// Create channels with configured buffer size
	a.channels = types.NewAgentChannels(a.bufferSize)

	// Initialize approval manager with default timeout
	a.approvalTimeout = 5 * time.Minute
	a.approvalManager = approval.NewManager(a.approvalTimeout, a.emitEvent)

	// If context manager was provided, set its event channel now that channels exist
	if a.contextManager != nil {
		a.contextManager.SetEventChannel(a.channels.Event)
	}

	return a
}

func (a *DefaultAgent) RegisterDefaultTools() {
	// Initialize built-in tools (respecting disabled tools configuration)
	if !a.disabledTools["task_completion"] {
		a.tools["task_completion"] = tools.NewTaskCompletionTool()
	}
	if !a.disabledTools["ask_question"] {
		a.tools["ask_question"] = tools.NewAskQuestionTool()
	}
	if !a.disabledTools["converse"] {
		a.tools["converse"] = tools.NewConverseTool()
	}
}

// Start begins the agent's event loop in a goroutine.
func (a *DefaultAgent) Start(ctx context.Context) error {
	a.runMu.Lock()
	if a.running {
		a.runMu.Unlock()
		return fmt.Errorf("agent is already running")
	}
	a.running = true
	a.runMu.Unlock()

	// Start event loop
	go a.eventLoop(ctx)

	return nil
}

// Shutdown gracefully stops the agent.
func (a *DefaultAgent) Shutdown(ctx context.Context) error {
	// Signal shutdown
	close(a.channels.Shutdown)

	// Wait for completion or context cancellation
	select {
	case <-a.channels.Done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetChannels returns the communication channels for this agent.
func (a *DefaultAgent) GetChannels() *types.AgentChannels {
	return a.channels
}

// eventLoop is the main processing loop for the agent.
func (a *DefaultAgent) eventLoop(ctx context.Context) {
	defer a.channels.Close()
	defer func() {
		a.runMu.Lock()
		a.running = false
		a.runMu.Unlock()
	}()

	// Start a separate goroutine to handle cancellation requests
	// This ensures cancellations are processed even when the main loop is blocked
	cancelCtx, cancelStop := context.WithCancel(ctx)
	defer cancelStop()

	go func() {
		for {
			select {
			case <-cancelCtx.Done():
				return
			case cancelReq := <-a.channels.Cancel:
				if cancelReq == nil {
					return
				}
				// Handle command cancellation request immediately
				a.handleCommandCancellation(cancelReq)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			// Context canceled
			a.emitEvent(types.NewErrorEvent(ctx.Err()))
			return

		case <-a.channels.Shutdown:
			// Shutdown requested
			return

		case input := <-a.channels.Input:
			if input == nil {
				// Channel closed
				return
			}

			// Handle cancellation immediately (synchronously) so it can interrupt ongoing processing
			if input.IsCancel() {
				a.processInput(ctx, input)
				continue
			}

			// Process other inputs asynchronously so eventLoop can continue handling cancel requests
			go a.processInput(ctx, input)

		case approval := <-a.channels.Approval:
			if approval == nil {
				// Channel closed
				return
			}

			// Handle approval response
			a.handleApprovalResponse(approval)
		}
	}
}

// processInput handles a single input from the user.
func (a *DefaultAgent) processInput(ctx context.Context, input *types.Input) {
	// Handle cancellation
	if input.IsCancel() {
		a.cancelMu.Lock()
		if a.cancelStream != nil {
			a.cancelStream()
			a.cancelStream = nil
		}
		a.cancelMu.Unlock()
		return
	}

	// Handle user input
	if input.IsUserInput() {
		a.processUserInput(ctx, input.Content)
		return
	}

	// Handle form input (not yet implemented)
	if input.IsFormInput() {
		a.emitEvent(types.NewErrorEvent(fmt.Errorf("form input not yet supported")))
		a.emitEvent(types.NewTurnEndEvent())
		return
	}

	// Handle notes request
	if input.IsNotesRequest() {
		a.handleNotesRequest(input)
		return
	}
}

// processUserInput processes a user text input using the agent loop.
func (a *DefaultAgent) processUserInput(ctx context.Context, content string) {
	// Add user message to memory
	userMsg := types.NewUserMessage(content)
	a.memory.Add(userMsg)

	// Create cancellable context for this turn
	turnCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	a.cancelMu.Lock()
	a.cancelStream = cancel
	a.cancelMu.Unlock()

	defer func() {
		a.cancelMu.Lock()
		a.cancelStream = nil
		a.cancelMu.Unlock()
	}()

	// Emit busy status
	a.emitEvent(types.NewUpdateBusyEvent(true))
	defer a.emitEvent(types.NewUpdateBusyEvent(false))

	// Run agent loop (now in assistant.go)
	a.runAgentLoop(turnCtx)

	// Emit turn end
	a.emitEvent(types.NewTurnEndEvent())
}

// RegisterTool adds a custom tool to the agent's tool registry.
// Built-in tools (task_completion, ask_question, converse) are always available
// and cannot be overridden.
func (a *DefaultAgent) RegisterTool(tool tools.Tool) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	// Prevent overriding built-in tools
	builtIns := map[string]bool{
		"task_completion": true,
		"ask_question":    true,
		"converse":        true,
	}
	if builtIns[name] {
		return fmt.Errorf("cannot override built-in tool: %s", name)
	}

	a.toolsMu.Lock()
	defer a.toolsMu.Unlock()

	a.tools[name] = tool
	return nil
}

// GetTool retrieves a specific tool by name from the agent's tool registry.
// Returns nil if the tool is not found.
func (a *DefaultAgent) GetTool(name string) interface{} {
	a.toolsMu.RLock()
	defer a.toolsMu.RUnlock()

	return a.tools[name]
}

// GetTools returns a list of all available tools (built-in + custom)
// This is used internally for prompt building and memory
func (a *DefaultAgent) GetTools() []interface{} {
	a.toolsMu.RLock()
	defer a.toolsMu.RUnlock()

	toolsList := make([]interface{}, 0, len(a.tools))
	for _, tool := range a.tools {
		toolsList = append(toolsList, tool)
	}
	return toolsList
}

// GetContextInfo returns detailed context information for debugging and display
func (a *DefaultAgent) GetContextInfo() *ContextInfo {
	a.toolsMu.RLock()
	defer a.toolsMu.RUnlock()

	// Build system prompt without tools and without repository context to calculate base system tokens
	baseSystemPrompt := prompts.NewPromptBuilder().
		WithCustomInstructions(a.customInstructions).
		Build()

	// Build just the repository context section to calculate repository tokens
	repositorySection := ""
	if a.repositoryContext != "" {
		repositorySection = "<repository_context>\n" + a.repositoryContext + "\n</repository_context>\n\n"
	}

	// Build just the tools section to calculate tool tokens
	toolsSection := ""
	if len(a.tools) > 0 {
		toolsSection = "<available_tools>\n" +
			prompts.FormatToolSchemas(a.getToolsList()) +
			"</available_tools>\n\n"
	}

	// Calculate token counts for each section
	systemPromptTokens := 0
	repositoryTokens := 0
	toolTokens := 0
	if a.tokenizer != nil {
		systemPromptTokens = a.tokenizer.CountTokens(baseSystemPrompt)
		repositoryTokens = a.tokenizer.CountTokens(repositorySection)
		toolTokens = a.tokenizer.CountTokens(toolsSection)
	}

	// Build full system prompt for current context calculation
	builder := prompts.NewPromptBuilder().
		WithTools(a.getToolsList()).
		WithCustomInstructions(a.customInstructions)

	if a.repositoryContext != "" {
		builder = builder.WithRepositoryContext(a.repositoryContext)
	}

	fullSystemPrompt := builder.Build()

	// Get tool names
	toolNames := make([]string, 0, len(a.tools))
	for name := range a.tools {
		toolNames = append(toolNames, name)
	}

	// Get message history stats
	messages := a.memory.GetAll()
	messageCount := len(messages)

	// Count conversation turns (user messages)
	conversationTurns := 0
	for _, msg := range messages {
		if msg.Role == types.RoleUser {
			conversationTurns++
		}
	}

	// Calculate token counts
	conversationTokens := 0
	currentTokens := 0
	if a.tokenizer != nil {
		conversationTokens = a.tokenizer.CountMessagesTokens(messages)
		// Calculate current context tokens
		currentTokens = conversationTokens + a.tokenizer.CountTokens(fullSystemPrompt)
	} else {
		// Fallback: approximate token counting when tokenizer is unavailable
		// Use rough estimate of 1 token ≈ 4 characters
		for _, msg := range messages {
			conversationTokens += (len(msg.Content) + len(string(msg.Role)) + 12) / 4 // +12 for message overhead
		}
		currentTokens = conversationTokens + len(fullSystemPrompt)/4
	}

	// Get max tokens from context manager
	maxTokens := 0
	if a.contextManager != nil {
		maxTokens = a.contextManager.GetMaxTokens()
	}

	// Calculate free tokens and usage percentage
	freeTokens := 0
	usagePercent := 0.0
	if maxTokens > 0 {
		freeTokens = maxTokens - currentTokens
		if freeTokens < 0 {
			freeTokens = 0
		}
		usagePercent = float64(currentTokens) / float64(maxTokens) * 100.0
	}

	return &ContextInfo{
		SystemPromptTokens:      systemPromptTokens,
		CustomInstructions:      a.customInstructions != "",
		RepositoryContextTokens: repositoryTokens,
		ToolCount:               len(a.tools),
		ToolTokens:              toolTokens,
		ToolNames:               toolNames,
		MessageCount:            messageCount,
		ConversationTurns:       conversationTurns,
		ConversationTokens:      conversationTokens,
		CurrentContextTokens:    currentTokens,
		MaxContextTokens:        maxTokens,
		FreeTokens:              freeTokens,
		UsagePercent:            usagePercent,
		TotalPromptTokens:       0, // These will be filled by the executor from its tracking
		TotalCompletionTokens:   0,
		TotalTokens:             0,
	}
}

// GetProvider returns the LLM provider used by this agent
func (a *DefaultAgent) GetProvider() llm.Provider {
	return a.provider
}

// SetProvider updates the LLM provider used by this agent.
// This allows hot-reloading of provider configuration without restarting the agent.
// The update is thread-safe and will take effect on the next agent iteration.
func (a *DefaultAgent) SetProvider(provider llm.Provider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	// Update the agent's provider
	a.provider = provider

	// Also update the context manager's provider if it exists
	if a.contextManager != nil {
		a.contextManager.SetProvider(provider)
	}

	return nil
}
