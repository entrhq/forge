package overlay

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/executor/tui/types"
	pkgtypes "github.com/entrhq/forge/pkg/types"
)

// Command overlay-specific styles that extend the shared overlay styles
var (
	commandStatusStyle = lipgloss.NewStyle().
		Foreground(types.MutedGray).
		Italic(true)
)

// CommandExecutionOverlay displays streaming command output with cancellation support
type CommandExecutionOverlay struct {
	*BaseOverlay
	command       string
	workingDir    string
	executionID   string
	output        *strings.Builder
	status        string
	exitCode      int
	isRunning     bool
	cancelChannel chan<- *pkgtypes.CancellationRequest
}

// NewCommandExecutionOverlay creates a new command execution overlay
func NewCommandExecutionOverlay(command, workingDir, executionID string, cancelChan chan<- *pkgtypes.CancellationRequest) *CommandExecutionOverlay {
	overlay := &CommandExecutionOverlay{
		command:       command,
		workingDir:    workingDir,
		executionID:   executionID,
		output:        &strings.Builder{},
		status:        "Running...",
		isRunning:     true,
		cancelChannel: cancelChan,
	}

	overlayWidth := 80
	overlayHeight := 30

	// Configure base overlay
	baseConfig := BaseOverlayConfig{
		Width:          overlayWidth,
		Height:         overlayHeight,
		ViewportWidth:  76,
		ViewportHeight: 20,
		Content:        "", // Content will be updated via streaming
		OnClose: func(actions types.ActionHandler) tea.Cmd {
			// Don't close if running - cancellation happens via Ctrl+C
			if overlay.isRunning {
				return nil
			}
			// Return nil to signal close - caller will handle ClearOverlay()
			return nil
		},
		RenderHeader:          overlay.renderHeader,
		RenderFooter:          overlay.renderFooter,
		FooterRendersViewport: true, // Footer renders viewport directly
	}

	overlay.BaseOverlay = NewBaseOverlay(baseConfig)
	return overlay
}

// Update handles messages for the command overlay
func (c *CommandExecutionOverlay) Update(msg tea.Msg, state types.StateProvider, actions types.ActionHandler) (types.Overlay, tea.Cmd) {
	// Handle auto-close message
	if autoClose, ok := msg.(autoCloseMsg); ok {
		return c.handleAutoClose(autoClose)
	}

	// Handle command execution events first
	if event, ok := msg.(*pkgtypes.AgentEvent); ok {
		if event.IsCommandExecutionEvent() {
			return c.handleCommandEvent(event)
		}
	}

	// Handle Ctrl+C and Esc specially for cancellation
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			// Send cancellation request if running
			if c.isRunning && c.cancelChannel != nil {
				c.cancelChannel <- &pkgtypes.CancellationRequest{
					ExecutionID: c.executionID,
				}
				c.status = "Canceling..."
				return c, nil
			}
			// If not running, close the overlay by returning nil
			return nil, nil
		}
	}

	// Let BaseOverlay handle standard keys (scrolling, etc.)
	handled, updatedBase, cmd := c.BaseOverlay.Update(msg, actions)
	c.BaseOverlay = updatedBase

	if handled {
		return c, cmd
	}

	return c, nil
}

// handleCommandEvent processes command execution events
func (c *CommandExecutionOverlay) handleCommandEvent(event *pkgtypes.AgentEvent) (types.Overlay, tea.Cmd) {
	if event.CommandExecution == nil {
		return c, nil
	}

	data := event.CommandExecution

	// Only process events for this execution
	if data.ExecutionID != c.executionID {
		return c, nil
	}

	switch event.Type {
	case pkgtypes.EventTypeCommandOutput:
		// Append new output and update viewport
		c.output.WriteString(data.Output)
		c.SetContent(c.output.String())

		// Auto-scroll to bottom if we were already at the bottom
		vp := c.Viewport()
		if vp.AtBottom() {
			vp.GotoBottom()
		}

	case pkgtypes.EventTypeCommandExecutionComplete:
		c.isRunning = false
		c.exitCode = data.ExitCode
		c.status = fmt.Sprintf("Completed in %s (exit code: %d)", data.Duration, data.ExitCode)
		return c, tea.Batch(
			c.maybeAutoClose(),
			c.showExitCodeToast(data.ExitCode, data.Duration),
		)

	case pkgtypes.EventTypeCommandExecutionFailed:
		c.isRunning = false
		c.exitCode = data.ExitCode
		c.status = fmt.Sprintf("Failed in %s (exit code: %d)", data.Duration, data.ExitCode)
		return c, tea.Batch(
			c.maybeAutoClose(),
			c.showExitCodeToast(data.ExitCode, data.Duration),
		)

	case pkgtypes.EventTypeCommandExecutionCanceled:
		c.isRunning = false
		c.status = "Canceled by user"
		// Auto-close overlay on cancellation by returning nil
		return nil, nil
	}

	return c, nil
}

// renderHeader renders the command execution header
func (c *CommandExecutionOverlay) renderHeader() string {
	var b strings.Builder

	b.WriteString(types.OverlayTitleStyle.Render("Command Execution"))
	b.WriteString("\n\n")

	// Command info
	b.WriteString(fmt.Sprintf("Command: %s", c.command))
	if c.workingDir != "" {
		b.WriteString(fmt.Sprintf("\nWorking Dir: %s", c.workingDir))
	}
	b.WriteString("\n")

	// Status line
	b.WriteString(commandStatusStyle.Render(c.status))

	return b.String()
}

// renderFooter renders the viewport output and help text
func (c *CommandExecutionOverlay) renderFooter() string {
	var b strings.Builder

	// Add blank line after header
	b.WriteString("\n")

	// Render viewport with command output
	b.WriteString(c.BaseOverlay.Viewport().View())
	b.WriteString("\n")

	// Add help text
	if c.isRunning {
		b.WriteString(types.OverlayHelpStyle.Render("Ctrl+C or Esc: Cancel | ↑↓: Scroll | PgUp/PgDn: Page"))
	} else {
		b.WriteString(types.OverlayHelpStyle.Render("Press Esc key to close"))
	}

	return b.String()
}

// View renders the overlay
func (c *CommandExecutionOverlay) View() string {
	return c.BaseOverlay.View(c.Width())
}

// maybeAutoClose checks configuration and returns a command to auto-close the overlay if enabled.
func (c *CommandExecutionOverlay) maybeAutoClose() tea.Cmd {
	// Get UI configuration
	uiConfig := config.GetUI()
	if uiConfig == nil {
		return nil
	}

	// Check if we should auto-close based on exit code
	if !uiConfig.ShouldAutoClose(c.exitCode) {
		return nil
	}

	// Get the configured delay
	_, _, delay := uiConfig.GetAutoCloseSettings()

	// Schedule auto-close after delay
	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return autoCloseMsg{executionID: c.executionID}
	})
}

// autoCloseMsg is sent after the auto-close delay to trigger overlay closure.
type autoCloseMsg struct {
	executionID string
}

// Update handles the auto-close message
func (c *CommandExecutionOverlay) handleAutoClose(msg autoCloseMsg) (types.Overlay, tea.Cmd) {
	// Only auto-close if this is our execution and we're not running
	if msg.executionID == c.executionID && !c.isRunning {
		return nil, nil
	}
	return c, nil
}

// showExitCodeToast returns a command to show a toast notification with the exit code.
func (c *CommandExecutionOverlay) showExitCodeToast(exitCode int, duration string) tea.Cmd {
	// Get UI configuration to check if auto-close is enabled
	uiConfig := config.GetUI()
	if uiConfig == nil {
		return nil
	}

	// Only show toast when the overlay will auto-close for this exit code
	if !uiConfig.ShouldAutoClose(exitCode) {
		return nil
	}

	// Format toast message based on exit code
	var message, icon string
	isError := exitCode != 0

	if exitCode == 0 {
		message = fmt.Sprintf("Command completed (exit code 0) in %s", duration)
		icon = "✓"
	} else {
		message = fmt.Sprintf("Command failed with exit code %d in %s", exitCode, duration)
		icon = "✗"
	}

	// Return a command that sends a ToastMsg
	return func() tea.Msg {
		return types.ToastMsg{
			Message: message,
			Details: fmt.Sprintf("Command: %s", c.command),
			Icon:    icon,
			IsError: isError,
		}
	}
}
