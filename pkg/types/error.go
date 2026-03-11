package types

import "fmt"

// ErrorCode defines specific error types that can occur in the agent framework.
type ErrorCode string

const (
	ErrorCodeLLMFailure   ErrorCode = "llm_failure"   // ErrorCodeLLMFailure indicates an error occurred while calling the LLM provider.
	ErrorCodeShutdown     ErrorCode = "shutdown"      // ErrorCodeShutdown indicates the agent is shutting down.
	ErrorCodeInvalidInput ErrorCode = "invalid_input" // ErrorCodeInvalidInput indicates the input provided was invalid.
	ErrorCodeTimeout      ErrorCode = "timeout"       // ErrorCodeTimeout indicates an operation timed out.
	ErrorCodeCanceled     ErrorCode = "canceled"      // ErrorCodeCanceled indicates an operation was canceled.
	ErrorCodeInternal     ErrorCode = "internal"      // ErrorCodeInternal indicates an internal error occurred.
)

// AgentError represents a structured error from the agent framework.
type AgentError struct {
	// Metadata holds optional additional context about the error.
	Metadata map[string]any

	// Message provides a human-readable description of the error.
	Message string

	// Cause is the underlying error that caused this error, if any.
	Cause error

	// Code is the specific error code.
	Code ErrorCode
}

// Error implements the error interface.
func (e *AgentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error, implementing errors.Unwrap.
func (e *AgentError) Unwrap() error {
	return e.Cause
}

// WithMetadata adds metadata to the error and returns the error for chaining.
func (e *AgentError) WithMetadata(key string, value any) *AgentError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	return e
}

// NewAgentError creates a new AgentError with the given code and message.
func NewAgentError(code ErrorCode, message string) *AgentError {
	return &AgentError{
		Code:     code,
		Message:  message,
		Metadata: make(map[string]any),
	}
}

// NewAgentErrorWithCause creates a new AgentError with a cause.
func NewAgentErrorWithCause(code ErrorCode, message string, cause error) *AgentError {
	return &AgentError{
		Code:     code,
		Message:  message,
		Cause:    cause,
		Metadata: make(map[string]any),
	}
}

// IsAgentError checks if an error is an AgentError and returns it.
func IsAgentError(err error) (*AgentError, bool) {
	if err == nil {
		return nil, false
	}

	if agentErr, ok := err.(*AgentError); ok {
		return agentErr, true
	}

	return nil, false
}
