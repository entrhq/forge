package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrhq/forge/pkg/agent/git"
	"github.com/entrhq/forge/pkg/executor/tui/approval"
	"github.com/entrhq/forge/pkg/executor/tui/overlay"
	tuitypes "github.com/entrhq/forge/pkg/executor/tui/types"
	"github.com/entrhq/forge/pkg/types"
)

// CommandType indicates whether a command is handled by TUI or Agent
type CommandType int

const (
	CommandTypeTUI   CommandType = iota // Handled entirely by TUI
	CommandTypeAgent                    // Sent to agent
)

// CommandHandler processes a slash command and returns either:
// - tea.Cmd for immediate execution
// - ApprovalRequest for commands requiring user approval
// - nil for commands with no side effects
// The model is passed as a pointer and can be modified directly.
type CommandHandler func(m *model, args []string) interface{}

// SlashCommand represents a registered command
type SlashCommand struct {
	Name             string         // Command name (without /)
	Description      string         // Short description for palette
	Type             CommandType    // Where to handle the command
	Handler          CommandHandler // Handler function (for TUI commands)
	RequiresApproval bool           // Whether command requires user approval
	MinArgs          int            // Minimum number of arguments
	MaxArgs          int            // Maximum number of arguments (-1 for unlimited)
}

// commandRegistry holds all registered slash commands
var commandRegistry map[string]*SlashCommand

// init initializes the command registry with built-in commands
func init() {
	commandRegistry = make(map[string]*SlashCommand)

	// Register built-in commands
	registerCommand(&SlashCommand{
		Name:        "help",
		Description: "Show tips and keyboard shortcuts",
		Type:        CommandTypeTUI,
		Handler:     handleHelpCommand,
		MinArgs:     0,
		MaxArgs:     0,
	})

	registerCommand(&SlashCommand{
		Name:        "stop",
		Description: "Stop current agent operation",
		Type:        CommandTypeAgent,
		Handler:     handleStopCommand,
		MinArgs:     0,
		MaxArgs:     0,
	})

	registerCommand(&SlashCommand{
		Name:             "commit",
		Description:      "Create git commit from session changes",
		Type:             CommandTypeAgent,
		Handler:          handleCommitCommand,
		RequiresApproval: true, // Commit requires approval
		MinArgs:          0,
		MaxArgs:          -1, // Unlimited for commit message
	})

	registerCommand(&SlashCommand{
		Name:             "pr",
		Description:      "Create pull request from current branch",
		Type:             CommandTypeAgent,
		Handler:          handlePRCommand,
		RequiresApproval: true, // PR requires approval
		MinArgs:          0,
		MaxArgs:          -1, // Unlimited for PR title
	})

	registerCommand(&SlashCommand{
		Name:        "settings",
		Description: "Open settings configuration",
		Type:        CommandTypeTUI,
		Handler:     handleSettingsCommand,
		MinArgs:     0,
		MaxArgs:     0,
	})

	registerCommand(&SlashCommand{
		Name:        "context",
		Description: "Show detailed context information",
		Type:        CommandTypeTUI,
		Handler:     handleContextCommand,
		MinArgs:     0,
		MaxArgs:     0,
	})

	registerCommand(&SlashCommand{
		Name:        "bash",
		Description: "Enter bash mode for running shell commands",
		Type:        CommandTypeTUI,
		Handler:     handleBashCommand,
		MinArgs:     0,
		MaxArgs:     0,
	})

	registerCommand(&SlashCommand{
		Name:        "notes",
		Description: "View scratchpad notes",
		Type:        CommandTypeTUI,
		Handler:     handleNotesCommand,
		MinArgs:     0,
		MaxArgs:     0,
	})

	registerCommand(&SlashCommand{
		Name:        "snapshot",
		Description: "Export full context snapshot to .forge/context/ for debugging",
		Type:        CommandTypeTUI,
		Handler:     handleSnapshotCommand,
		MinArgs:     0,
		MaxArgs:     0,
	})
}

// registerCommand adds a command to the registry
func registerCommand(cmd *SlashCommand) {
	commandRegistry[cmd.Name] = cmd
}

// getCommand retrieves a command from the registry
func getCommand(name string) (*SlashCommand, bool) {
	cmd, exists := commandRegistry[name]
	return cmd, exists
}

// getAllCommands returns all registered commands
func getAllCommands() []*SlashCommand {
	commands := make([]*SlashCommand, 0, len(commandRegistry))
	for _, cmd := range commandRegistry {
		commands = append(commands, cmd)
	}
	return commands
}

// parseSlashCommand parses a slash command input into command name and arguments
// Returns: commandName, args, isCommand
func parseSlashCommand(input string) (string, []string, bool) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return "", nil, false
	}

	// Remove the leading /
	trimmed = trimmed[1:]

	// Split into parts
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", nil, false
	}

	commandName := parts[0]
	args := []string{}
	if len(parts) > 1 {
		args = parts[1:]
	}

	return commandName, args, true
}

// executeSlashCommand executes a slash command
func executeSlashCommand(m *model, commandName string, args []string) (*model, tea.Cmd) {
	cmd, exists := getCommand(commandName)
	if !exists {
		// Unknown command - show error toast
		m.showToast("Unknown command", fmt.Sprintf("Command '/%s' not found. Type /help for available commands.", commandName), "❌", true)
		return m, nil
	}

	// Validate argument count
	if len(args) < cmd.MinArgs {
		m.showToast("Invalid arguments", fmt.Sprintf("Command '/%s' requires at least %d argument(s)", commandName, cmd.MinArgs), "❌", true)
		return m, nil
	}
	if cmd.MaxArgs != -1 && len(args) > cmd.MaxArgs {
		m.showToast("Invalid arguments", fmt.Sprintf("Command '/%s' accepts at most %d argument(s)", commandName, cmd.MaxArgs), "❌", true)
		return m, nil
	}

	// Execute the command handler
	if cmd.Handler != nil {
		result := cmd.Handler(m, args)

		// Handle different return types
		switch v := result.(type) {
		case tea.Cmd:
			// Direct command execution
			return m, v
		case func() tea.Msg:
			// Function that returns a message (also a tea.Cmd, but type switch needs explicit match)
			// For long-running commands (commit, pr), wrap with busy indicator
			if commandName == "commit" || commandName == "pr" {
				m.agentBusy = true
				m.currentLoadingMessage = getRandomLoadingMessage()
				m.recalculateLayout()
				// Wrap the command to clear busy state when done
				return m, func() tea.Msg {
					msg := v()
					// After the command executes, send completion message
					return tea.Batch(
						func() tea.Msg { return msg },
						func() tea.Msg { return slashCommandCompleteMsg{} },
					)()
				}
			}
			return m, tea.Cmd(v)
		case approval.ApprovalRequest:
			// Command requires approval - send approval request message
			return m, func() tea.Msg {
				return approvalRequestMsg{request: v}
			}
		case nil:
			// No-op command
			return m, nil
		default:
			// Unknown return type - show error
			m.showToast("Command Error", fmt.Sprintf("Command '/%s' returned unexpected type", commandName), "❌", true)
			return m, nil
		}
	}

	return m, nil
}

// handleHelpCommand shows help information
func handleHelpCommand(m *model, args []string) interface{} {
	// Build help content
	var helpContent strings.Builder
	helpContent.WriteString("Available Commands:\n\n")

	commands := getAllCommands()
	for _, cmd := range commands {
		helpContent.WriteString(fmt.Sprintf("  /%s\n", cmd.Name))
		helpContent.WriteString(fmt.Sprintf("    %s\n\n", cmd.Description))
	}

	helpContent.WriteString("Keyboard Shortcuts:\n\n")
	helpContent.WriteString("  Enter        Send message\n")
	helpContent.WriteString("  Alt+Enter    New line\n")
	helpContent.WriteString("  Ctrl+C       Exit\n")
	helpContent.WriteString("  Ctrl+D       Show command help\n\n")

	helpContent.WriteString("Tips:\n\n")
	helpContent.WriteString("  • Type / to see available commands\n")
	helpContent.WriteString("  • Use arrow keys to navigate command palette\n")
	helpContent.WriteString("  • Press Escape to cancel command entry\n")

	// Create and activate the help overlay
	helpOverlay := overlay.NewHelpOverlay("Help", helpContent.String())
	m.overlay.activate(tuitypes.OverlayModeHelp, helpOverlay)

	return nil
}

// handleStopCommand stops the current agent operation
func handleStopCommand(m *model, args []string) interface{} {
	if m.channels != nil {
		// Send cancel input to agent
		m.channels.Input <- types.NewCancelInput()
		m.showToast("Stopping", "Sent stop signal to agent", "⏹️", false)
	}
	return nil
}

// handleCommitCommand creates a git commit with preview
func handleCommitCommand(m *model, args []string) interface{} {
	if m.slashHandler == nil {
		m.showToast("Error", "Git operations not available", "❌", true)
		return nil
	}

	commitMessage := strings.Join(args, " ")

	// Gather preview data and return approval request
	return func() tea.Msg {
		ctx := context.Background()

		// Get modified files
		files, err := git.GetModifiedFiles(m.workspaceDir)
		if err != nil {
			return toastMsg{
				message: "Commit Failed",
				details: fmt.Sprintf("Failed to get modified files: %v", err),
				icon:    "❌",
				isError: true,
			}
		}

		if len(files) == 0 {
			return toastMsg{
				message: "Nothing to Commit",
				details: "No modified files found",
				icon:    "ℹ️",
				isError: false,
			}
		}

		// Get diff for preview
		diff := getDiffForFiles(m.workspaceDir, files)

		// Generate commit message if not provided
		message := commitMessage
		if message == "" {
			generatedMsg, err := m.commitGen.Generate(ctx, m.workspaceDir, files)
			if err != nil {
				return toastMsg{
					message: "Commit Failed",
					details: fmt.Sprintf("Failed to generate commit message: %v", err),
					icon:    "❌",
					isError: true,
				}
			}
			message = generatedMsg
		}

		// Return approval request instead of command-specific message
		return approvalRequestMsg{
			request: approval.NewCommitRequest(files, message, diff, commitMessage, m.slashHandler),
		}
	}
}

// getDiffForFiles gets the git diff for the specified files
func getDiffForFiles(workingDir string, files []string) string {
	// Try to get diff against HEAD first (for modified tracked files)
	args := append([]string{"diff", "HEAD", "--"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		// If that fails (new files not in HEAD), try without HEAD
		// This will show working directory changes
		args = append([]string{"diff", "--"}, files...)
		cmd = exec.Command("git", args...)
		cmd.Dir = workingDir

		output, err = cmd.Output()
		if err != nil {
			// If both fail, return a helpful message
			return "(Unable to generate diff preview - files may be untracked)"
		}
	}

	// If output is empty, the files might be new/untracked
	// Try to show them as additions
	if len(output) == 0 {
		// Get diff of what would be staged if we add these files
		args = append([]string{"diff", "--no-index", "/dev/null", "--"}, files...)
		cmd = exec.Command("git", args...)
		cmd.Dir = workingDir

		output, err = cmd.Output()
		if err != nil {
			return "(New/untracked files - run commit to see full content)"
		}
	}

	return string(output)
}

// handlePRCommand creates a pull request with preview
func handlePRCommand(m *model, args []string) interface{} {
	if m.slashHandler == nil {
		m.showToast("Error", "Git operations not available", "❌", true)
		return nil
	}

	prTitle := strings.Join(args, " ")

	return func() tea.Msg {
		ctx := context.Background()

		// Get base branch
		base, err := git.DetectBaseBranch(m.workspaceDir)
		if err != nil {
			return toastMsg{
				message: "PR Failed",
				details: fmt.Sprintf("Failed to detect base branch: %v", err),
				icon:    "❌",
				isError: true,
			}
		}

		// Get current branch
		head, err := getCurrentBranch(m.workspaceDir)
		if err != nil {
			return toastMsg{
				message: "PR Failed",
				details: fmt.Sprintf("Failed to get current branch: %v", err),
				icon:    "❌",
				isError: true,
			}
		}

		// Get commits since base
		commits, err := git.GetCommitsSinceBase(m.workspaceDir, base, head)
		if err != nil {
			return toastMsg{
				message: "PR Failed",
				details: fmt.Sprintf("Failed to get commits: %v", err),
				icon:    "❌",
				isError: true,
			}
		}

		if len(commits) == 0 {
			return toastMsg{
				message: "Nothing to PR",
				details: "No commits found for pull request",
				icon:    "ℹ️",
				isError: false,
			}
		}

		// Get diff summary
		diffSummary, err := git.GetDiffSummary(m.workspaceDir, base, head)
		if err != nil {
			return toastMsg{
				message: "PR Failed",
				details: fmt.Sprintf("Failed to get diff summary: %v", err),
				icon:    "❌",
				isError: true,
			}
		}

		// Generate PR content
		prContent, err := m.prGen.Generate(ctx, commits, diffSummary, base, head, prTitle)
		if err != nil {
			return toastMsg{
				message: "PR Failed",
				details: fmt.Sprintf("Failed to generate PR content: %v", err),
				icon:    "❌",
				isError: true,
			}
		}

		// Build commits and changes preview
		var changesContent strings.Builder
		changesContent.WriteString(fmt.Sprintf("Commits (%d):\n", len(commits)))
		for _, commit := range commits {
			changesContent.WriteString(fmt.Sprintf("  • %s\n", commit.Message))
		}
		changesContent.WriteString("\n")
		changesContent.WriteString(diffSummary)

		// Return approval request instead of command-specific message
		branchInfo := fmt.Sprintf("%s → %s", head, base)
		return approvalRequestMsg{
			request: approval.NewPRRequest(branchInfo, prContent.Title, prContent.Description, changesContent.String(), prTitle, m.slashHandler),
		}
	}
}

// getCurrentBranch gets the current git branch name
func getCurrentBranch(workingDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// handleSettingsCommand shows the settings configuration
func handleSettingsCommand(m *model, args []string) interface{} {
	// Create settings overlay with callback for LLM settings changes
	onLLMSettingsChange := func() error {
		return m.reloadLLMProvider()
	}

	settingsOverlay := overlay.NewSettingsOverlayWithCallback(m.width, m.height, onLLMSettingsChange, m.provider)
	m.overlay.activateAndClearStack(tuitypes.OverlayModeSettings, settingsOverlay)

	return nil
}

// handleContextCommand shows detailed context information
func handleContextCommand(m *model, args []string) interface{} {
	// Get context info from agent
	if m.agent == nil {
		m.showToast("Error", "Agent not available", "❌", true)
		return nil
	}

	contextInfo := m.agent.GetContextInfo()

	// Add token usage from TUI tracking
	freeTokens := 0
	usagePercent := 0.0
	if contextInfo.MaxContextTokens > 0 {
		freeTokens = contextInfo.MaxContextTokens - contextInfo.CurrentContextTokens
		usagePercent = float64(contextInfo.CurrentContextTokens) / float64(contextInfo.MaxContextTokens) * 100.0
	}

	// Build overlay info structure
	overlayInfo := &overlay.ContextInfo{
		SystemPromptTokens:      contextInfo.SystemPromptTokens,
		CustomInstructions:      contextInfo.CustomInstructions,
		RepositoryContextTokens: contextInfo.RepositoryContextTokens,
		ToolCount:               contextInfo.ToolCount,
		ToolTokens:              contextInfo.ToolTokens,
		ToolNames:               contextInfo.ToolNames,
		MessageCount:            contextInfo.MessageCount,
		ConversationTurns:       contextInfo.ConversationTurns,
		ConversationTokens:      contextInfo.ConversationTokens,
		RawMessageCount:         contextInfo.RawMessageCount,
		RawMessageTokens:        contextInfo.RawMessageTokens,
		SummaryBlockCount:       contextInfo.SummaryBlockCount,
		SummaryBlockTokens:      contextInfo.SummaryBlockTokens,
		GoalBatchBlockCount:     contextInfo.GoalBatchBlockCount,
		GoalBatchBlockTokens:    contextInfo.GoalBatchBlockTokens,
		CurrentContextTokens:    contextInfo.CurrentContextTokens,
		MaxContextTokens:        contextInfo.MaxContextTokens,
		FreeTokens:              freeTokens,
		UsagePercent:            usagePercent,
		TotalPromptTokens:       m.totalPromptTokens,
		TotalCompletionTokens:   m.totalCompletionTokens,
		TotalTokens:             m.totalTokens,
	}

	// Create and activate context overlay
	contextOverlay := overlay.NewContextOverlay(overlayInfo, m.width, m.height)
	m.overlay.activate(tuitypes.OverlayModeContext, contextOverlay)

	return nil
}

// handleBashCommand enters bash mode for running shell commands
func handleBashCommand(m *model, args []string) interface{} {
	m.bashMode = true
	m.updatePrompt()
	m.showToast("Bash Mode", "Entered bash mode. Commands will be executed directly. Type 'exit' or press Ctrl+C to return.", "🔧", false)
	return nil
}

// handleNotesCommand requests notes data from the agent and shows notes viewer
func handleNotesCommand(m *model, args []string) interface{} {
	// Send notes request to agent
	return m.requestNotes()
}

// contextSnapshotTokens holds aggregate token statistics for the exported snapshot.
type contextSnapshotTokens struct {
	CurrentContext  int     `json:"current_context"`
	MaxContext      int     `json:"max_context"`
	UsagePercent    float64 `json:"usage_percent"`
	SystemPrompt    int     `json:"system_prompt"`
	Conversation    int     `json:"conversation"`
	RawMessages     int     `json:"raw_messages"`
	SummaryBlocks   int     `json:"summary_blocks"`
	GoalBatchBlocks int     `json:"goal_batch_blocks"`
}

// contextSnapshotMessage is one message entry in the exported snapshot.
type contextSnapshotMessage struct {
	Index         int    `json:"index"`
	Role          string `json:"role"`
	Content       string `json:"content"`
	Tokens        int    `json:"tokens"`
	IsSummarized  bool   `json:"is_summarized"`
	SummaryType   string `json:"summary_type,omitempty"`
	SummaryCount  int    `json:"summary_count,omitempty"`
	SummaryMethod string `json:"summary_method,omitempty"`
}

// contextSnapshot is the top-level structure written to the JSON file.
type contextSnapshot struct {
	ExportedAt   string                   `json:"exported_at"`
	Workspace    string                   `json:"workspace"`
	TokenSummary contextSnapshotTokens    `json:"token_summary"`
	Messages     []contextSnapshotMessage `json:"messages"`
}

// handleSnapshotCommand exports the full conversation payload to a timestamped
// JSON file in <workspace>/.forge/context/ and shows a toast with the output path.
func handleSnapshotCommand(m *model, args []string) interface{} {
	if m.agent == nil {
		m.showToast("Error", "Agent not available", "❌", true)
		return nil
	}

	snapshot := buildContextSnapshot(m)

	// Ensure the output directory exists.
	outDir := filepath.Join(m.workspaceDir, ".forge", "context")
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		m.showToast("Export failed", fmt.Sprintf("Could not create directory: %v", err), "❌", true)
		return nil
	}

	// Write the snapshot with a timestamp-based filename so exports accumulate.
	filename := time.Now().Format("20060102-150405") + "-context.json"
	outPath := filepath.Join(outDir, filename)

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		m.showToast("Export failed", fmt.Sprintf("Could not marshal JSON: %v", err), "❌", true)
		return nil
	}

	if err := os.WriteFile(outPath, data, 0o600); err != nil {
		m.showToast("Export failed", fmt.Sprintf("Could not write file: %v", err), "❌", true)
		return nil
	}

	m.showToast("Context exported", outPath, "📄", false)
	return nil
}

// buildContextSnapshot assembles a contextSnapshot from live agent state.
// It uses GetContextInfo() for token statistics, GetSystemPrompt() for the full
// system prompt (which is never stored in conversation memory), and GetMessages()
// for the conversation payload. The resulting Messages slice mirrors exactly what
// the agent passes to the LLM: [system, ...history].
func buildContextSnapshot(m *model) *contextSnapshot {
	info := m.agent.GetContextInfo()
	systemPrompt := m.agent.GetSystemPrompt()
	messages := m.agent.GetMessages()

	// Reserve capacity for the system prompt entry + all conversation messages.
	msgEntries := make([]contextSnapshotMessage, 0, 1+len(messages))

	// Index 0 is always the system prompt, synthesized fresh (not in memory).
	sysTokens := (len(systemPrompt) + len("system") + 12) / 4
	msgEntries = append(msgEntries, contextSnapshotMessage{
		Index:   0,
		Role:    "system",
		Content: systemPrompt,
		Tokens:  sysTokens,
	})

	// Append conversation history starting at index 1.
	for i, msg := range messages {
		isSummarized, _ := msg.Metadata["summarized"].(bool)
		summaryType, _ := msg.Metadata["summary_type"].(string)
		summaryCount, _ := msg.Metadata["summary_count"].(int)
		summaryMethod, _ := msg.Metadata["summary_method"].(string)

		// Approximate per-message token count (content chars / 4 + role overhead).
		// Accurate counting would require the tokenizer, which is not exposed here.
		approxTokens := (len(msg.Content) + len(string(msg.Role)) + 12) / 4

		msgEntries = append(msgEntries, contextSnapshotMessage{
			Index:         i + 1,
			Role:          string(msg.Role),
			Content:       msg.Content,
			Tokens:        approxTokens,
			IsSummarized:  isSummarized,
			SummaryType:   summaryType,
			SummaryCount:  summaryCount,
			SummaryMethod: summaryMethod,
		})
	}

	usagePct := 0.0
	if info.MaxContextTokens > 0 {
		usagePct = float64(info.CurrentContextTokens) / float64(info.MaxContextTokens) * 100.0
	}

	return &contextSnapshot{
		ExportedAt: time.Now().Format(time.RFC3339),
		Workspace:  m.workspaceDir,
		TokenSummary: contextSnapshotTokens{
			CurrentContext:  info.CurrentContextTokens,
			MaxContext:      info.MaxContextTokens,
			UsagePercent:    usagePct,
			SystemPrompt:    info.SystemPromptTokens,
			Conversation:    info.ConversationTokens,
			RawMessages:     info.RawMessageTokens,
			SummaryBlocks:   info.SummaryBlockTokens,
			GoalBatchBlocks: info.GoalBatchBlockTokens,
		},
		Messages: msgEntries,
	}
}
