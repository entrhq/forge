package tui

import (
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrhq/forge/pkg/executor/tui/overlay"
	tuitypes "github.com/entrhq/forge/pkg/executor/tui/types"
	"github.com/entrhq/forge/pkg/types"
)

var debugLog *log.Logger

func initDebugLog() {
	if debugLog != nil {
		return // Already initialized
	}

	// Create debug log file
	f, err := os.OpenFile("forge-tui-debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Warning: error opening debug log file: %v", err)
		// Create a no-op logger to avoid nil pointer panics
		debugLog = log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lshortfile)
		return
	}
	debugLog = log.New(f, "", log.LstdFlags|log.Lshortfile)
	debugLog.Printf("Debug logging initialized")
}

// Update handles all state updates for the TUI model.
// This is the main event loop handler for Bubble Tea.
//
// Uses pointer receiver to ensure overlay mutations via ActionHandler persist.
// Without pointer receiver, &m passed to overlays points to a local copy,
// causing state changes (SetInput, ShowToast, etc.) to be lost.
//
//nolint:gocyclo
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check if quit was requested by an overlay or component
	if m.shouldQuit {
		return m, tea.Quit
	}

	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	// Handle spinner tick messages
	var spinnerCmd tea.Cmd
	m.spinner, spinnerCmd = m.spinner.Update(msg)

	// Handle command palette keyboard input BEFORE updating textarea
	// This prevents Enter from being processed by textarea when palette is active
	if keyMsg, ok := msg.(tea.KeyMsg); ok && m.commandPalette.IsActive() {
		switch keyMsg.Type {
		case tea.KeyEsc:
			// Cancel command palette
			m.commandPalette.Deactivate()
			m.textarea.Reset()
			return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
		case tea.KeyUp:
			// Navigate up in palette, don't update textarea
			m.commandPalette.SelectPrev()
			return m, spinnerCmd
		case tea.KeyDown:
			// Navigate down in palette, don't update textarea
			m.commandPalette.SelectNext()
			return m, spinnerCmd
		case tea.KeyTab:
			// Autocomplete with selected command and close palette
			selected := m.commandPalette.GetSelected()
			if selected != nil {
				m.textarea.SetValue("/" + selected.Name + " ")
				m.textarea.CursorEnd()
			}
			m.commandPalette.Deactivate()
			return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
		case tea.KeyEnter:
			// Autocomplete with the selected command and close the palette
			selected := m.commandPalette.GetSelected()
			if selected != nil {
				m.textarea.SetValue("/" + selected.Name + " ")
				m.textarea.CursorEnd()
			}
			m.commandPalette.Deactivate()
			return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
		}
		// For other keys, continue to textarea update below
	}

	// Only update textarea if no overlay or result list is active
	// This prevents the textarea from capturing scroll events when an overlay is open
	if !m.overlay.isActive() && !m.resultList.IsActive() {
		// Store old textarea height to detect changes
		oldHeight := m.textarea.Height()
		m.textarea, tiCmd = m.textarea.Update(msg)
		newHeight := m.textarea.Height()

		// If textarea height changed, recalculate viewport height
		if oldHeight != newHeight && m.ready {
			m.recalculateLayout()
		}

		// Check if we should activate/deactivate command palette based on input
		value := m.textarea.Value()

		// Handle command palette activation/deactivation based on input
		switch {
		case value == "/" && !m.commandPalette.IsActive():
			// Only activate palette if input is exactly "/" as first character
			m.commandPalette.Activate()
			m.commandPalette.UpdateFilter("")
		case strings.HasPrefix(value, "/") && m.commandPalette.IsActive():
			// Update filter if palette is already active
			filter := strings.TrimPrefix(value, "/")
			m.commandPalette.UpdateFilter(filter)
		case !strings.HasPrefix(value, "/") && m.commandPalette.IsActive():
			// Deactivate palette if input no longer starts with /
			m.commandPalette.Deactivate()
		}

		// Auto-adjust textarea height based on content after any key press
		m.updateTextAreaHeight()
	}

	switch msg := msg.(type) {
	case tuitypes.ViewResultMsg:
		debugLog.Printf("Received viewResultMsg")
		return m.handleViewResult(msg)

	case tuitypes.ViewNoteMsg:
		debugLog.Printf("Received viewNoteMsg")
		return m.handleViewNote(msg)

	case tea.WindowSizeMsg:
		debugLog.Printf("Received tea.WindowSizeMsg: width=%d, height=%d", msg.Width, msg.Height)
		return m.handleWindowResize(msg)

	case slashCommandCompleteMsg:
		debugLog.Printf("Received slashCommandCompleteMsg")
		return m.handleSlashCommandComplete()

	case operationStartMsg:
		debugLog.Printf("Received operationStartMsg: %s", msg.message)
		return m.handleOperationStart(msg)

	case tuitypes.OperationStartMsg:
		debugLog.Printf("Received types.OperationStartMsg: %s", msg.Message)
		// Convert to internal type
		return m.handleOperationStart(operationStartMsg{message: msg.Message})

	case operationCompleteMsg:
		debugLog.Printf("Received operationCompleteMsg: result=%s, err=%v", msg.result, msg.err)
		return m.handleOperationComplete(msg)

	case tuitypes.OperationCompleteMsg:
		debugLog.Printf("Received types.OperationCompleteMsg: result=%s, err=%v", msg.Result, msg.Err)
		// Convert to internal type
		return m.handleOperationComplete(operationCompleteMsg{
			result:       msg.Result,
			err:          msg.Err,
			successTitle: msg.SuccessTitle,
			successIcon:  msg.SuccessIcon,
			errorTitle:   msg.ErrorTitle,
			errorIcon:    msg.ErrorIcon,
		})

	case toastMsg:
		debugLog.Printf("Received toastMsg: %s", msg.message)
		return m.handleToast(msg)

	case tuitypes.ToastMsg:
		debugLog.Printf("Received types.ToastMsg: %s", msg.Message)
		// Convert to internal type
		return m.handleToast(toastMsg{
			message: msg.Message,
			details: msg.Details,
			icon:    msg.Icon,
			isError: msg.IsError,
		})

	case agentErrMsg:
		debugLog.Printf("Received agentErrMsg: %v", msg.err)
		return m.handleAgentError(msg)

	case bashCommandResultMsg:
		return m.handleBashCommandResult(msg)

	case approvalRequestMsg:
		debugLog.Printf("Received approvalRequestMsg")
		return m.handleApprovalRequest(msg)

	case *types.AgentEvent:
		debugLog.Printf("Received *types.AgentEvent: %s", msg.Type)

		// If overlay is active and it's a command execution event, forward to overlay
		if m.overlay.isActive() && msg.IsCommandExecutionEvent() {
			var overlayCmd tea.Cmd
			m.overlay.overlay, overlayCmd = m.overlay.overlay.Update(msg, m, m)
			// Still handle the event in the main model too
			m.handleAgentEvent(msg)
			return m, tea.Batch(tiCmd, vpCmd, overlayCmd, spinnerCmd)
		}

		// Update viewport BEFORE handling event (important for streaming)
		m.viewport, vpCmd = m.viewport.Update(msg)
		m.handleAgentEvent(msg)
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)

	case tea.MouseMsg:
		debugLog.Printf("Received tea.MouseMsg")
		// Handle mouse events (especially scroll wheel) for viewport
		// If overlay is active, forward mouse events to it
		if m.overlay.isActive() {
			var overlayCmd tea.Cmd
			updatedOverlay, overlayCmd := m.overlay.overlay.Update(msg, m, m)

			// Check if overlay returned nil (signals to close)
			if updatedOverlay == nil {
				m.overlay.deactivate()
				m.viewport.SetContent(m.content.String())
				m.viewport.GotoBottom()
				return m, overlayCmd
			}

			m.overlay.overlay = updatedOverlay
			return m, overlayCmd
		}

		// Route mouse events to viewport for scrolling
		m.viewport, vpCmd = m.viewport.Update(msg)
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)

	case tea.KeyMsg:
		debugLog.Printf("Received tea.KeyMsg: %s", msg.String())
		return m.handleKeyPress(msg, vpCmd, tiCmd, spinnerCmd)

	default:
		debugLog.Printf("Received unknown message type: %T", msg)
	}

	// Update viewport with current message handling
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
}

// handleViewResult processes result selection from the result list
func (m *model) handleViewResult(msg tuitypes.ViewResultMsg) (tea.Model, tea.Cmd) {
	if result, ok := m.resultCache.get(msg.ResultID); ok {
		// Close the result list
		m.resultList.Deactivate()
		// Open the result in an overlay
		overlay := overlay.NewToolResultOverlay(result.ToolName, result.Result, m.width, m.height)
		m.overlay.activate(tuitypes.OverlayModeToolResult, overlay)
		return m, nil
	}
	// If result not found, just close the list
	m.resultList.Deactivate()
	return m, nil
}

// handleViewNote processes note selection from the notes list
func (m *model) handleViewNote(msg tuitypes.ViewNoteMsg) (tea.Model, tea.Cmd) {
	if msg.Note != nil {
		// Build note detail content
		content := fmt.Sprintf("Note ID: %s\n", msg.Note.ID)
		content += fmt.Sprintf("Tags: [%s]\n", strings.Join(msg.Note.Tags, ", "))
		content += fmt.Sprintf("Created: %s\n", msg.Note.CreatedAt)
		content += fmt.Sprintf("Updated: %s\n\n", msg.Note.UpdatedAt)
		content += msg.Note.Content

		// Push note detail overlay on top of notes list (allows back navigation)
		overlay := overlay.NewToolResultOverlay("Note Detail", content, m.width, m.height)
		m.overlay.pushOverlay(tuitypes.OverlayModeToolResult, overlay)
		return m, nil
	}
	return m, nil
}

// handleWindowResize processes window size change events
// calculateViewportHeight computes the appropriate viewport height based on current model state
func (m *model) calculateViewportHeight() int {
	headerHeight := 10                     // ASCII art (6) + tips (1) + status bar (1) + blank line (1) + spacing (1)
	inputHeight := m.textarea.Height() + 2 // textarea height + border
	statusBarHeight := 1
	loadingHeight := 0
	if m.agentBusy {
		loadingHeight = 1 // Loading indicator is a separate line when visible
	}

	viewportHeight := m.height - headerHeight - inputHeight - statusBarHeight - loadingHeight
	if viewportHeight < 5 {
		viewportHeight = 5
	}
	return viewportHeight
}

func (m *model) handleWindowResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	// Update viewport on window resize
	m.viewport, _ = m.viewport.Update(msg)

	// Also update result list if active
	if m.resultList.IsActive() {
		updated, listCmd := m.resultList.Update(msg, m, m)
		if rl, ok := updated.(*overlay.ResultListModel); ok {
			m.resultList = *rl
		}
		return m, listCmd
	}

	m.width = msg.Width
	m.height = msg.Height

	// Calculate and set viewport dimensions
	m.viewport.Width = m.width - 4
	m.viewport.Height = m.calculateViewportHeight()
	m.textarea.SetWidth(m.width - 8)
	m.ready = true
	m.recalculateLayout()
	return m, nil
}

// handleSlashCommandComplete processes slash command completion
func (m *model) handleSlashCommandComplete() (tea.Model, tea.Cmd) {
	m.agentBusy = false
	m.recalculateLayout()
	return m, nil
}

// handleOperationStart processes operation start events
func (m *model) handleOperationStart(msg operationStartMsg) (tea.Model, tea.Cmd) {
	m.agentBusy = true
	m.currentLoadingMessage = msg.message
	m.recalculateLayout()
	return m, nil
}

// handleOperationComplete processes operation completion events
func (m *model) handleOperationComplete(msg operationCompleteMsg) (tea.Model, tea.Cmd) {
	m.agentBusy = false
	m.recalculateLayout()

	if msg.err != nil {
		m.showToast(msg.errorTitle, fmt.Sprintf("%v", msg.err), msg.errorIcon, true)
	} else {
		m.showToast(msg.successTitle, msg.result, msg.successIcon, false)
	}

	// Close any active overlay after operation completes
	if m.overlay.isActive() {
		m.overlay.deactivate()
		m.viewport.SetContent(m.content.String())
		m.viewport.GotoBottom()
	}

	return m, nil
}

// handleToast processes toast notification messages
func (m *model) handleToast(msg toastMsg) (tea.Model, tea.Cmd) {
	m.showToast(msg.message, msg.details, msg.icon, msg.isError)
	m.agentBusy = false
	m.recalculateLayout()

	// Close any active overlay after showing toast
	// This handles rejection cases where overlay should close
	if m.overlay.isActive() {
		m.overlay.deactivate()
		m.viewport.SetContent(m.content.String())
		m.viewport.GotoBottom()
	}

	return m, nil
}

// handleAgentError processes agent error messages
func (m *model) handleAgentError(msg agentErrMsg) (tea.Model, tea.Cmd) {
	m.content.WriteString(errorStyle.Render(fmt.Sprintf("  ❌ Error: %v", msg.err)))
	m.content.WriteString("\n\n")
	m.viewport.SetContent(m.content.String())
	m.viewport.GotoBottom()
	m.agentBusy = false
	m.recalculateLayout()
	return m, nil
}

// handleBashCommandResult processes bash command execution results
func (m *model) handleBashCommandResult(msg bashCommandResultMsg) (tea.Model, tea.Cmd) {
	// Display the result in the viewport
	m.content.WriteString(fmt.Sprintf("[%s] Command completed:\n", msg.timestamp))
	m.content.WriteString(msg.result)
	m.content.WriteString("\n\n")

	m.viewport.SetContent(m.content.String())
	m.viewport.GotoBottom()

	return m, nil
}

// handleApprovalRequest processes generic approval requests (for slash commands, etc.)
func (m *model) handleApprovalRequest(msg approvalRequestMsg) (tea.Model, tea.Cmd) {
	// Create and activate generic approval overlay
	overlay := overlay.NewGenericApprovalOverlay(msg.request, m.width, m.height)
	m.overlay.activate(tuitypes.OverlayModeApproval, overlay)
	return m, nil
}

// handleKeyPress processes keyboard input
func (m *model) handleKeyPress(msg tea.KeyMsg, vpCmd, tiCmd, spinnerCmd tea.Cmd) (tea.Model, tea.Cmd) {
	// Command palette handling is now done earlier in Update() before textarea update
	// This prevents the duplicate handling issue

	// If an overlay is active, pass keys to the overlay
	if m.overlay.isActive() {
		if m.overlay.overlay != nil {
			updated, cmd := m.overlay.overlay.Update(msg, m, m)
			// If overlay returns nil, it wants to close
			if updated == nil {
				m.ClearOverlay()
			} else {
				// updated is already an Overlay interface, no need for type assertion
				m.overlay.overlay = updated
			}
			return m, tea.Batch(cmd, spinnerCmd)
		}
		// If overlay is marked active but nil, deactivate it
		m.overlay.deactivate()
	}

	// If result list is active, pass keys to the result list
	if m.resultList.IsActive() {
		updated, cmd := m.resultList.Update(msg, m, m)
		if rl, ok := updated.(*overlay.ResultListModel); ok {
			m.resultList = *rl
		}
		return m, tea.Batch(cmd, spinnerCmd)
	}

	// Handle key presses based on type
	switch msg.Type {
	case tea.KeyEsc:
		// Escape exits bash mode if active
		if m.bashMode {
			m.bashMode = false
			m.textarea.Reset()
			m.updatePrompt()
			m.recalculateLayout()
			return m, nil
		}

	case tea.KeyCtrlC:
		return m.handleCtrlC()

	case tea.KeyCtrlV:
		return m.handleCtrlV()

	case tea.KeyCtrlL:
		return m.handleCtrlL()

	case tea.KeyCtrlK:
		return m.handleCtrlK()

	case tea.KeyCtrlP:
		return m.handleCtrlP()

	case tea.KeyEnter:
		// Check if Alt is held down
		if msg.Alt {
			// Insert a newline character
			m.textarea.InsertString("\n")
			m.updateTextAreaHeight()
			return m, nil
		}
		return m.handleEnter(tiCmd, vpCmd, spinnerCmd)
	}

	return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
}

// handleCtrlC handles Ctrl+C key press (exit or exit bash mode)
func (m *model) handleCtrlC() (tea.Model, tea.Cmd) {
	if m.bashMode {
		m.bashMode = false
		m.textarea.Reset()
		m.updatePrompt()
		m.recalculateLayout()
		return m, nil
	}
	return m, tea.Quit
}

// handleCtrlV handles Ctrl+V key press (view last tool result)
func (m *model) handleCtrlV() (tea.Model, tea.Cmd) {
	if m.lastToolCallID != "" {
		if result, ok := m.resultCache.get(m.lastToolCallID); ok {
			overlay := overlay.NewToolResultOverlay(result.ToolName, result.Result, m.width, m.height)
			m.overlay.activate(tuitypes.OverlayModeToolResult, overlay)
		}
	}
	return m, nil
}

// handleCtrlL handles Ctrl+L key press (show result history)
func (m *model) handleCtrlL() (tea.Model, tea.Cmd) {
	results := m.resultCache.getAll()
	m.resultList.Activate(results, m.width, m.height)
	return m, nil
}

// handleCtrlK handles Ctrl+K key press (toggle command palette)
func (m *model) handleCtrlK() (tea.Model, tea.Cmd) {
	if m.commandPalette.IsActive() {
		m.commandPalette.Deactivate()
	} else {
		m.commandPalette.Activate()
	}
	return m, nil
}

// handleCtrlP handles Ctrl+P key press (toggle command palette - alternate)
func (m *model) handleCtrlP() (tea.Model, tea.Cmd) {
	if m.commandPalette.IsActive() {
		m.commandPalette.Deactivate()
	} else {
		m.commandPalette.Activate()
	}
	return m, nil
}

// handleEnter handles Enter key press (send message or execute bash command)
func (m *model) handleEnter(tiCmd, vpCmd, spinnerCmd tea.Cmd) (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textarea.Value())

	if input == "" {
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
	}

	// Handle bash mode
	if m.bashMode {
		return m.handleBashModeInput(input, tiCmd, vpCmd, spinnerCmd)
	}

	// Handle slash commands
	if strings.HasPrefix(input, "/") {
		return m.handleSlashCommand(input, tiCmd, vpCmd, spinnerCmd)
	}

	// Handle single-shot bash commands
	if strings.HasPrefix(input, "!") {
		return m.handleSingleShotBash(input, tiCmd, vpCmd, spinnerCmd)
	}

	// Handle regular agent message
	return m.handleAgentMessage(input, tiCmd, vpCmd, spinnerCmd)
}

// handleBashModeInput processes bash mode input
func (m *model) handleBashModeInput(input string, tiCmd, vpCmd, spinnerCmd tea.Cmd) (tea.Model, tea.Cmd) {
	// Check for exit command
	if input == "exit" {
		m.bashMode = false
		m.textarea.Reset()
		m.recalculateLayout()
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
	}

	// Execute the bash command
	m.content.WriteString(bashPromptStyle.Render(fmt.Sprintf("$ %s", input)))
	m.content.WriteString("\n")

	// Clear the input area
	m.textarea.Reset()
	m.viewport.SetContent(m.content.String())
	m.viewport.GotoBottom()

	// Execute command and return to event loop
	return m, tea.Batch(tiCmd, vpCmd, spinnerCmd, m.executeBashCommand(input))
}

// handleSlashCommand processes slash commands
func (m *model) handleSlashCommand(input string, tiCmd, vpCmd, spinnerCmd tea.Cmd) (tea.Model, tea.Cmd) {
	// Do NOT display slash commands in chat history - they are executed silently

	// Clear the input area
	m.textarea.Reset()

	// Parse slash command
	commandName, args, ok := parseSlashCommand(input)
	if !ok {
		m.showToast("Invalid command", "Could not parse slash command", "❌", true)
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
	}

	// Execute slash command
	updatedModel, cmd := executeSlashCommand(m, commandName, args)
	return updatedModel, tea.Batch(tiCmd, vpCmd, spinnerCmd, cmd)
}

// handleSingleShotBash processes single-shot bash commands
func (m *model) handleSingleShotBash(input string, tiCmd, vpCmd, spinnerCmd tea.Cmd) (tea.Model, tea.Cmd) {
	// Remove the leading '!' and get the command
	bashCmd := strings.TrimSpace(input[1:])
	if bashCmd == "" {
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
	}

	// Display the command
	m.content.WriteString(bashPromptStyle.Render(fmt.Sprintf("$ %s", bashCmd)))
	m.content.WriteString("\n")

	// Clear the input area
	m.textarea.Reset()
	m.viewport.SetContent(m.content.String())
	m.viewport.GotoBottom()

	// Execute command
	return m, tea.Batch(tiCmd, vpCmd, spinnerCmd, m.executeBashCommand(bashCmd))
}

// handleAgentMessage processes regular agent messages
func (m *model) handleAgentMessage(input string, tiCmd, vpCmd, spinnerCmd tea.Cmd) (tea.Model, tea.Cmd) {
	// Display user message
	formatted := formatEntry("You: ", input, userStyle, m.width, true)
	// Strip any trailing newlines before adding our spacing
	formatted = strings.TrimRight(formatted, "\n")
	m.content.WriteString(formatted + "\n\n")

	// Clear input
	m.textarea.Reset()
	m.viewport.SetContent(m.content.String())
	m.viewport.GotoBottom()

	// Set agent busy
	m.agentBusy = true
	m.currentLoadingMessage = getRandomLoadingMessage()
	m.recalculateLayout()

	// Send message to agent
	userInput := types.NewUserInput(input)
	debugLog.Printf("Sending user input to agent: %+v", userInput)
	m.channels.Input <- userInput

	return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
}

// recalculateLayout updates viewport content and scrolls to bottom
func (m *model) recalculateLayout() {
	// Update viewport height based on current state (including loading indicator)
	m.viewport.Height = m.calculateViewportHeight()
	m.viewport.SetContent(m.content.String())
	m.viewport.GotoBottom()
}
