package types

// AgentEventType defines the type of event emitted by the agent.
type AgentEventType string

const (
	EventTypeThinkingStart                AgentEventType = "thinking_start"                 // EventTypeThinkingStart indicates the agent is starting to think/reason.
	EventTypeThinkingContent              AgentEventType = "thinking_content"               // EventTypeThinkingContent indicates content from the agent's thinking process.
	EventTypeThinkingEnd                  AgentEventType = "thinking_end"                   // EventTypeThinkingEnd indicates the agent has finished thinking.
	EventTypeToolCallStart                AgentEventType = "tool_call_start"                // EventTypeToolCallStart indicates the agent is starting to format a tool call.
	EventTypeToolCallContent              AgentEventType = "tool_call_content"              // EventTypeToolCallContent indicates content from the tool call XML.
	EventTypeToolCallEnd                  AgentEventType = "tool_call_end"                  // EventTypeToolCallEnd indicates the agent has finished the tool call XML.
	EventTypeMessageStart                 AgentEventType = "message_start"                  // EventTypeMessageStart indicates the agent is starting to compose a message.
	EventTypeMessageContent               AgentEventType = "message_content"                // EventTypeMessageContent indicates content from the agent's message.
	EventTypeMessageEnd                   AgentEventType = "message_end"                    // EventTypeMessageEnd indicates the agent has finished composing the message.
	EventTypeToolCall                     AgentEventType = "tool_call"                      // EventTypeToolCall indicates the agent is calling a tool.
	EventTypeToolResult                   AgentEventType = "tool_result"                    // EventTypeToolResult indicates a successful tool call result.
	EventTypeToolResultError              AgentEventType = "tool_result_error"              // EventTypeToolResultError indicates a tool call resulted in an error.
	EventTypeNoToolCall                   AgentEventType = "no_tool_call"                   // EventTypeNoToolCall indicates the agent decided not to call any tools.
	EventTypeAPICallStart                 AgentEventType = "api_call_start"                 // EventTypeAPICallStart indicates the agent is making an API call.
	EventTypeAPICallEnd                   AgentEventType = "api_call_end"                   // EventTypeAPICallEnd indicates an API call has completed.
	EventTypeToolsUpdate                  AgentEventType = "tools_update"                   // EventTypeToolsUpdate indicates the agent's available tools have been updated.
	EventTypeUpdateBusy                   AgentEventType = "update_busy"                    // EventTypeUpdateBusy indicates a change in the agent's busy status.
	EventTypeTurnEnd                      AgentEventType = "turn_end"                       // EventTypeTurnEnd indicates the agent has finished processing the current turn.
	EventTypeError                        AgentEventType = "error"                          // EventTypeError indicates an error occurred during agent processing.
	EventTypeToolApprovalRequest          AgentEventType = "tool_approval_request"          // EventTypeToolApprovalRequest indicates the agent is requesting approval for a tool execution.
	EventTypeToolApprovalTimeout          AgentEventType = "tool_approval_timeout"          // EventTypeToolApprovalTimeout indicates an approval request has timed out.
	EventTypeToolApprovalGranted          AgentEventType = "tool_approval_granted"          // EventTypeToolApprovalGranted indicates the user approved the tool execution.
	EventTypeToolApprovalRejected         AgentEventType = "tool_approval_rejected"         // EventTypeToolApprovalRejected indicates the user rejected the tool execution.
	EventTypeTokenUsage                   AgentEventType = "token_usage"                    // EventTypeTokenUsage indicates token usage information from an LLM completion.
	EventTypeCommandExecutionStart        AgentEventType = "command_execution_start"        // EventTypeCommandExecutionStart indicates a command has started executing.
	EventTypeCommandOutput                AgentEventType = "command_output"                 // EventTypeCommandOutput indicates output from a running command.
	EventTypeCommandExecutionComplete     AgentEventType = "command_execution_complete"     // EventTypeCommandExecutionComplete indicates a command finished successfully.
	EventTypeCommandExecutionFailed       AgentEventType = "command_execution_failed"       // EventTypeCommandExecutionFailed indicates a command failed with an error.
	EventTypeCommandExecutionCanceled     AgentEventType = "command_execution_canceled"     // EventTypeCommandExecutionCanceled indicates a command was canceled by the user.
	EventTypeContextSummarizationStart    AgentEventType = "context_summarization_start"    // EventTypeContextSummarizationStart indicates context summarization has started.
	EventTypeContextSummarizationProgress AgentEventType = "context_summarization_progress" // EventTypeContextSummarizationProgress indicates progress during context summarization.
	EventTypeContextSummarizationComplete AgentEventType = "context_summarization_complete" // EventTypeContextSummarizationComplete indicates context summarization finished successfully.
	EventTypeContextSummarizationError    AgentEventType = "context_summarization_error"    // EventTypeContextSummarizationError indicates an error occurred during context summarization.
	EventTypeNotesData                    AgentEventType = "notes_data"                     // EventTypeNotesData indicates notes data response from agent.
)

// AgentEvent represents an event emitted by the agent during execution.
type AgentEvent struct {
	// Metadata holds optional additional information about the event.
	Metadata map[string]interface{}

	// ToolInput is the input being sent to the tool (for tool call events).
	ToolInput map[string]interface{}

	// ToolOutput is the result from the tool (for tool result events).
	ToolOutput interface{}

	// Error contains error information for error events.
	Error error

	// Content holds text content for content-type events (thinking, message, etc.).
	Content string

	// ToolName is the name of the tool being called (for tool events).
	ToolName string

	// Type indicates the kind of event.
	Type AgentEventType

	// IsBusy indicates if the agent is busy (for busy status events).
	IsBusy bool

	// ApprovalID is a unique identifier for approval requests/responses.
	ApprovalID string

	// Preview holds preview data for approval requests.
	Preview interface{}

	// TokenUsage contains token usage information (for token usage events).
	// Fields: PromptTokens, CompletionTokens, TotalTokens
	TokenUsage *TokenUsage

	// CommandExecution contains command execution information (for command execution events).
	CommandExecution *CommandExecution

	// ContextSummarization contains context summarization information (for context summarization events).
	ContextSummarization *ContextSummarization

	// APICallInfo contains API call information (for API call events).
	APICallInfo *APICallInfo

	// NotesData contains notes data (for notes data events).
	NotesData *NotesData
}

// TokenUsage contains token usage statistics from an LLM API call.
type TokenUsage struct {
	// PromptTokens is the number of tokens in the input/prompt.
	PromptTokens int

	// CompletionTokens is the number of tokens in the generated completion/response.
	CompletionTokens int

	// TotalTokens is the total number of tokens used (prompt + completion).
	TotalTokens int
}

// ContextSummarization contains information about context summarization.
type ContextSummarization struct {
	// Strategy is the name of the summarization strategy being executed.
	Strategy string

	// CurrentTokens is the current token count before summarization.
	CurrentTokens int

	// MaxTokens is the maximum allowed tokens.
	MaxTokens int

	// TokensSaved is the number of tokens saved by summarization.
	TokensSaved int

	// NewTokenCount is the token count after summarization.
	NewTokenCount int

	// ItemsProcessed is the number of items that have been summarized.
	ItemsProcessed int

	// TotalItems is the total number of items to summarize.
	TotalItems int

	// Duration is how long the summarization took.
	Duration string

	// Error contains error information if summarization failed.
	ErrorMessage string
}

// CommandExecution contains information about command execution.
type CommandExecution struct {
	// Command is the shell command being executed.
	Command string

	// WorkingDir is the working directory for the command.
	WorkingDir string

	// Output is the buffered output chunk (for CommandOutput events).
	Output string

	// StreamType indicates whether output is from stdout or stderr.
	StreamType string // "stdout" or "stderr"

	// ExitCode is the command's exit code (for completion/failed events).
	ExitCode int

	// Duration is how long the command took to execute.
	Duration string

	// ExecutionID is a unique identifier for this command execution.
	ExecutionID string
}

// APICallInfo contains information about an API call.
type APICallInfo struct {
	// ContextTokens is the current conversation context size in tokens.
	ContextTokens int

	// MaxContextTokens is the configured maximum context limit in tokens.
	MaxContextTokens int
}

// NewThinkingStartEvent creates a thinking start event.
func NewThinkingStartEvent() *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeThinkingStart,
		Metadata: make(map[string]interface{}),
	}
}

// NewThinkingContentEvent creates a thinking content event.
func NewThinkingContentEvent(content string) *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeThinkingContent,
		Content:  content,
		Metadata: make(map[string]interface{}),
	}
}

// NewThinkingEndEvent creates a thinking end event.
func NewThinkingEndEvent() *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeThinkingEnd,
		Metadata: make(map[string]interface{}),
	}
}

// NewToolCallStartEvent creates a tool call start event.
func NewToolCallStartEvent() *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeToolCallStart,
		Metadata: make(map[string]interface{}),
	}
}

// NewToolCallContentEvent creates a tool call content event.
func NewToolCallContentEvent(content string) *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeToolCallContent,
		Content:  content,
		Metadata: make(map[string]interface{}),
	}
}

// NewToolCallEndEvent creates a tool call end event.
func NewToolCallEndEvent() *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeToolCallEnd,
		Metadata: make(map[string]interface{}),
	}
}

// NewMessageStartEvent creates a message start event.
func NewMessageStartEvent() *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeMessageStart,
		Metadata: make(map[string]interface{}),
	}
}

// NewMessageContentEvent creates a message content event.
func NewMessageContentEvent(content string) *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeMessageContent,
		Content:  content,
		Metadata: make(map[string]interface{}),
	}
}

// NewMessageEndEvent creates a message end event.
func NewMessageEndEvent() *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeMessageEnd,
		Metadata: make(map[string]interface{}),
	}
}

// NewToolCallEvent creates a tool call event.
func NewToolCallEvent(toolName string, toolInput map[string]interface{}) *AgentEvent {
	return &AgentEvent{
		Type:      EventTypeToolCall,
		ToolName:  toolName,
		ToolInput: toolInput,
		Metadata:  make(map[string]interface{}),
	}
}

// NewToolResultEvent creates a tool result event.
func NewToolResultEvent(toolName string, output interface{}) *AgentEvent {
	return &AgentEvent{
		Type:       EventTypeToolResult,
		ToolName:   toolName,
		ToolOutput: output,
		Metadata:   make(map[string]interface{}),
	}
}

// NewToolResultErrorEvent creates a tool result error event.
func NewToolResultErrorEvent(toolName string, err error) *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeToolResultError,
		ToolName: toolName,
		Error:    err,
		Metadata: make(map[string]interface{}),
	}
}

// NewNoToolCallEvent creates a no tool call event.
func NewNoToolCallEvent() *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeNoToolCall,
		Metadata: make(map[string]interface{}),
	}
}

// NewAPICallStartEvent creates an API call start event with context token information.
func NewAPICallStartEvent(apiName string, contextTokens, maxContextTokens int) *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeAPICallStart,
		Metadata: map[string]interface{}{"api_name": apiName},
		APICallInfo: &APICallInfo{
			ContextTokens:    contextTokens,
			MaxContextTokens: maxContextTokens,
		},
	}
}

// NewAPICallEndEvent creates an API call end event.
func NewAPICallEndEvent(apiName string) *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeAPICallEnd,
		Metadata: map[string]interface{}{"api_name": apiName},
	}
}

// NewToolsUpdateEvent creates a tools update event.
func NewToolsUpdateEvent(tools []string) *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeToolsUpdate,
		Metadata: map[string]interface{}{"tools": tools},
	}
}

// NewUpdateBusyEvent creates a busy status update event.
func NewUpdateBusyEvent(isBusy bool) *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeUpdateBusy,
		IsBusy:   isBusy,
		Metadata: make(map[string]interface{}),
	}
}

// NewTurnEndEvent creates a turn end event.
func NewTurnEndEvent() *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeTurnEnd,
		Metadata: make(map[string]interface{}),
	}
}

// NewErrorEvent creates an error event.
func NewErrorEvent(err error) *AgentEvent {
	return &AgentEvent{
		Type:     EventTypeError,
		Error:    err,
		Metadata: make(map[string]interface{}),
	}
}

// NewToolApprovalRequestEvent creates a tool approval request event.
func NewToolApprovalRequestEvent(approvalID, toolName string, toolInput map[string]interface{}, preview interface{}) *AgentEvent {
	return &AgentEvent{
		Type:       EventTypeToolApprovalRequest,
		ApprovalID: approvalID,
		ToolName:   toolName,
		ToolInput:  toolInput,
		Preview:    preview,
		Metadata:   make(map[string]interface{}),
	}
}

// NewToolApprovalTimeoutEvent creates a tool approval timeout event.
func NewToolApprovalTimeoutEvent(approvalID, toolName string) *AgentEvent {
	return &AgentEvent{
		Type:       EventTypeToolApprovalTimeout,
		ApprovalID: approvalID,
		ToolName:   toolName,
		Metadata:   make(map[string]interface{}),
	}
}

// NewToolApprovalGrantedEvent creates a tool approval granted event.
func NewToolApprovalGrantedEvent(approvalID, toolName string) *AgentEvent {
	return &AgentEvent{
		Type:       EventTypeToolApprovalGranted,
		ApprovalID: approvalID,
		ToolName:   toolName,
		Metadata:   make(map[string]interface{}),
	}
}

// NewToolApprovalRejectedEvent creates a tool approval rejected event.
func NewToolApprovalRejectedEvent(approvalID, toolName string) *AgentEvent {
	return &AgentEvent{
		Type:       EventTypeToolApprovalRejected,
		ApprovalID: approvalID,
		ToolName:   toolName,
		Metadata:   make(map[string]interface{}),
	}
}

// NewTokenUsageEvent creates a token usage event.
func NewTokenUsageEvent(promptTokens, completionTokens, totalTokens int) *AgentEvent {
	return &AgentEvent{
		Type: EventTypeTokenUsage,
		TokenUsage: &TokenUsage{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		},
		Metadata: make(map[string]interface{}),
	}
}

// NewCommandExecutionStartEvent creates a command execution start event.
func NewCommandExecutionStartEvent(executionID, command, workingDir string) *AgentEvent {
	return &AgentEvent{
		Type: EventTypeCommandExecutionStart,
		CommandExecution: &CommandExecution{
			ExecutionID: executionID,
			Command:     command,
			WorkingDir:  workingDir,
		},
		Metadata: make(map[string]interface{}),
	}
}

// NewCommandOutputEvent creates a command output event.
func NewCommandOutputEvent(executionID, output, streamType string) *AgentEvent {
	return &AgentEvent{
		Type: EventTypeCommandOutput,
		CommandExecution: &CommandExecution{
			ExecutionID: executionID,
			Output:      output,
			StreamType:  streamType,
		},
		Metadata: make(map[string]interface{}),
	}
}

// NewCommandExecutionCompleteEvent creates a command execution complete event.
func NewCommandExecutionCompleteEvent(executionID string, exitCode int, duration string) *AgentEvent {
	return &AgentEvent{
		Type: EventTypeCommandExecutionComplete,
		CommandExecution: &CommandExecution{
			ExecutionID: executionID,
			ExitCode:    exitCode,
			Duration:    duration,
		},
		Metadata: make(map[string]interface{}),
	}
}

// NewCommandExecutionFailedEvent creates a command execution failed event.
func NewCommandExecutionFailedEvent(executionID string, exitCode int, duration string, err error) *AgentEvent {
	return &AgentEvent{
		Type:  EventTypeCommandExecutionFailed,
		Error: err,
		CommandExecution: &CommandExecution{
			ExecutionID: executionID,
			ExitCode:    exitCode,
			Duration:    duration,
		},
		Metadata: make(map[string]interface{}),
	}
}

// NewCommandExecutionCanceledEvent creates a command execution canceled event.
func NewCommandExecutionCanceledEvent(executionID string, duration string) *AgentEvent {
	return &AgentEvent{
		Type: EventTypeCommandExecutionCanceled,
		CommandExecution: &CommandExecution{
			ExecutionID: executionID,
			Duration:    duration,
			ExitCode:    -1, // Indicate cancellation
		},
		Metadata: make(map[string]interface{}),
	}
}

// WithMetadata adds metadata to the event and returns the event for chaining.
func (e *AgentEvent) WithMetadata(key string, value interface{}) *AgentEvent {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// IsThinkingEvent returns true if this is any thinking-related event.
func (e *AgentEvent) IsThinkingEvent() bool {
	return e.Type == EventTypeThinkingStart ||
		e.Type == EventTypeThinkingContent ||
		e.Type == EventTypeThinkingEnd
}

// IsMessageEvent returns true if this is any message-related event.
func (e *AgentEvent) IsMessageEvent() bool {
	return e.Type == EventTypeMessageStart ||
		e.Type == EventTypeMessageContent ||
		e.Type == EventTypeMessageEnd
}

// IsToolEvent returns true if this is any tool-related event.
func (e *AgentEvent) IsToolEvent() bool {
	return e.Type == EventTypeToolCall ||
		e.Type == EventTypeToolResult ||
		e.Type == EventTypeToolResultError ||
		e.Type == EventTypeNoToolCall
}

// IsAPIEvent returns true if this is any API-related event.
func (e *AgentEvent) IsAPIEvent() bool {
	return e.Type == EventTypeAPICallStart ||
		e.Type == EventTypeAPICallEnd
}

// IsContentEvent returns true if this event contains text content.
func (e *AgentEvent) IsContentEvent() bool {
	return e.Type == EventTypeThinkingContent ||
		e.Type == EventTypeMessageContent
}

// IsErrorEvent returns true if this is an error event.
func (e *AgentEvent) IsErrorEvent() bool {
	return e.Type == EventTypeError
}

// IsApprovalEvent returns true if this is any approval-related event.
func (e *AgentEvent) IsApprovalEvent() bool {
	return e.Type == EventTypeToolApprovalRequest ||
		e.Type == EventTypeToolApprovalTimeout ||
		e.Type == EventTypeToolApprovalGranted ||
		e.Type == EventTypeToolApprovalRejected
}

// IsCommandExecutionEvent returns true if this is any command execution-related event.
func (e *AgentEvent) IsCommandExecutionEvent() bool {
	return e.Type == EventTypeCommandExecutionStart ||
		e.Type == EventTypeCommandOutput ||
		e.Type == EventTypeCommandExecutionComplete ||
		e.Type == EventTypeCommandExecutionFailed ||
		e.Type == EventTypeCommandExecutionCanceled
}

// NewContextSummarizationStartEvent creates a context summarization start event.
func NewContextSummarizationStartEvent(strategy string, currentTokens, maxTokens int) *AgentEvent {
	return &AgentEvent{
		Type: EventTypeContextSummarizationStart,
		ContextSummarization: &ContextSummarization{
			Strategy:      strategy,
			CurrentTokens: currentTokens,
			MaxTokens:     maxTokens,
		},
		Metadata: make(map[string]interface{}),
	}
}

// NewContextSummarizationProgressEvent creates a context summarization progress event.
func NewContextSummarizationProgressEvent(strategy string, itemsProcessed, totalItems, tokensSaved int) *AgentEvent {
	return &AgentEvent{
		Type: EventTypeContextSummarizationProgress,
		ContextSummarization: &ContextSummarization{
			Strategy:       strategy,
			ItemsProcessed: itemsProcessed,
			TotalItems:     totalItems,
			TokensSaved:    tokensSaved,
		},
		Metadata: make(map[string]interface{}),
	}
}

// NewContextSummarizationCompleteEvent creates a context summarization complete event.
func NewContextSummarizationCompleteEvent(strategy string, tokensSaved, newTokenCount, itemsProcessed int, duration string) *AgentEvent {
	return &AgentEvent{
		Type: EventTypeContextSummarizationComplete,
		ContextSummarization: &ContextSummarization{
			Strategy:       strategy,
			TokensSaved:    tokensSaved,
			NewTokenCount:  newTokenCount,
			ItemsProcessed: itemsProcessed,
			Duration:       duration,
		},
		Metadata: make(map[string]interface{}),
	}
}

// NewContextSummarizationErrorEvent creates a context summarization error event.
func NewContextSummarizationErrorEvent(strategy string, err error) *AgentEvent {
	return &AgentEvent{
		Type:  EventTypeContextSummarizationError,
		Error: err,
		ContextSummarization: &ContextSummarization{
			Strategy:     strategy,
			ErrorMessage: err.Error(),
		},
		Metadata: make(map[string]interface{}),
	}
}

// IsContextSummarizationEvent returns true if this is any context summarization-related event.
func (e *AgentEvent) IsContextSummarizationEvent() bool {
	return e.Type == EventTypeContextSummarizationStart ||
		e.Type == EventTypeContextSummarizationProgress ||
		e.Type == EventTypeContextSummarizationComplete ||
		e.Type == EventTypeContextSummarizationError
}

// NoteData represents a single note for display.
type NoteData struct {
	ID        string
	Content   string
	Tags      []string
	CreatedAt string
	UpdatedAt string
	Scratched bool
}

// NotesData contains the notes response data.
type NotesData struct {
	Notes []NoteData
	Tag   string // Echo the requested tag filter
}

// NewNotesDataEvent creates a new notes data event.
func NewNotesDataEvent(notes []NoteData, tag string) *AgentEvent {
	return &AgentEvent{
		Type: EventTypeNotesData,
		NotesData: &NotesData{
			Notes: notes,
			Tag:   tag,
		},
	}
}
