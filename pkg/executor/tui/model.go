package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/entrhq/forge/pkg/agent"
	"github.com/entrhq/forge/pkg/agent/git"
	"github.com/entrhq/forge/pkg/agent/slash"
	"github.com/entrhq/forge/pkg/executor/tui/approval"
	"github.com/entrhq/forge/pkg/executor/tui/overlay"
	"github.com/entrhq/forge/pkg/types"
)

// model represents the state of the TUI application.
// It contains all components needed for the interactive terminal interface.
type model struct {
	// Bubble Tea components
	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	// Agent integration
	agent    agent.Agent
	channels *types.AgentChannels

	// Git and slash command support
	slashHandler *slash.Handler
	workspaceDir string
	commitGen    *git.CommitMessageGenerator
	prGen        *git.PRGenerator

	// Customization
	header string // Custom ASCII art header (empty means use default)

	// Content buffers
	content        *strings.Builder
	thinkingBuffer *strings.Builder
	messageBuffer  *strings.Builder

	// UI state
	overlay        *overlayState
	commandPalette *overlay.CommandPalette
	summarization  *summarizationStatus
	toast          *toastNotification

	// Agent state
	isThinking            bool
	agentBusy             bool
	bashMode              bool // Track if in bash mode
	currentLoadingMessage string
	toolNameDisplayed     bool // Track if we've already displayed the tool name
	pendingNotesRequest   bool // Track if we're waiting for notes data

	// Window dimensions
	width  int
	height int
	ready  bool

	// Message state
	hasMessageContentStarted bool

	// Token usage tracking
	totalPromptTokens     int // Cumulative input tokens across all API calls
	totalCompletionTokens int // Cumulative output tokens across all API calls
	totalTokens           int // Cumulative total tokens (input + output)
	currentContextTokens  int // Current conversation context size
	maxContextTokens      int // Maximum allowed context size

	// Tool result display
	resultClassifier *ToolResultClassifier
	resultSummarizer *ToolResultSummarizer
	resultCache      *resultCache
	resultList       overlay.ResultListModel // Result history list overlay
	lastToolCallID   string                  // Track the last tool call for 'v' shortcut
	lastToolName     string                  // Track the last tool name

	// Application state
	shouldQuit bool // Flag to trigger application exit
}

// agentErrMsg represents an error from agent operations
type agentErrMsg struct{ err error }

// approvalRequestMsg signals that a tool approval request is needed
type approvalRequestMsg struct {
	request approval.ApprovalRequest
}

// slashCommandCompleteMsg signals that a slash command has completed
type slashCommandCompleteMsg struct{}

// notesDataMsg contains notes data received from the agent
type notesDataMsg struct {
	notes []types.NoteData
}

// operationStartMsg signals that a long-running operation has started
type operationStartMsg struct {
	message string // Loading message to display
}

// operationCompleteMsg signals that a long-running operation has completed
type operationCompleteMsg struct {
	result       string
	err          error
	successTitle string
	successIcon  string
	errorTitle   string
	errorIcon    string
}

// toastMsg triggers a toast notification
type toastMsg struct {
	message string
	details string
	icon    string
	isError bool
}

// summarizationStatus tracks an active context summarization operation
type summarizationStatus struct {
	active          bool
	strategy        string
	currentTokens   int
	maxTokens       int
	itemsProcessed  int
	totalItems      int
	currentItem     string
	progressPercent float64
	startTime       time.Time
}

// toastNotification represents a temporary notification message
type toastNotification struct {
	active    bool
	message   string
	details   string
	icon      string
	isError   bool
	showUntil time.Time
}
