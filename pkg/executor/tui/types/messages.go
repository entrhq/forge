package types

import (
	"time"
)

// AgentErrMsg represents an error from agent operations
type AgentErrMsg struct{ Err error }

// SlashCommandCompleteMsg signals that a slash command has completed
type SlashCommandCompleteMsg struct{}

// OperationStartMsg signals that a long-running operation has started
type OperationStartMsg struct {
	Message string // Loading message to display
}

// OperationCompleteMsg signals that a long-running operation has completed
type OperationCompleteMsg struct {
	Result       string
	Err          error
	SuccessTitle string
	SuccessIcon  string
	ErrorTitle   string
	ErrorIcon    string
}

// SummarizationStatus tracks an active context summarization operation
type SummarizationStatus struct {
	Active          bool
	Strategy        string
	CurrentTokens   int
	MaxTokens       int
	ItemsProcessed  int
	TotalItems      int
	CurrentItem     string
	ProgressPercent float64
	StartTime       time.Time
}

// ToastNotification represents a temporary notification message
type ToastNotification struct {
	Active    bool
	Message   string
	Details   string
	Icon      string
	IsError   bool
	ShowUntil time.Time
}

// ToastMsg is a message type for showing toast notifications
type ToastMsg struct {
	Message string
	Details string
	Icon    string
	IsError bool
}

// ViewResultMsg is sent when a result is selected from the list
type ViewResultMsg struct {
	ResultID string
}

// ViewNoteMsg is sent when a note is selected from the notes list
type ViewNoteMsg struct {
	Note *NoteData
}

// NoteData represents a single note for display (mirrored from pkg/types)
type NoteData struct {
	ID        string
	Content   string
	Tags      []string
	CreatedAt string
	UpdatedAt string
	Scratched bool
}
