package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrhq/forge/pkg/executor/tui/overlay"
	tuitypes "github.com/entrhq/forge/pkg/executor/tui/types"
)

// handleViewResult processes result selection from the result list.
func (m *model) handleViewResult(msg tuitypes.ViewResultMsg) {
	if result, ok := m.resultCache.get(msg.ResultID); ok {
		m.resultList.Deactivate()
		ol := overlay.NewToolResultOverlay(result.ToolName, result.Result, m.width, m.height)
		m.overlay.activate(tuitypes.OverlayModeToolResult, ol)
		return
	}
	m.resultList.Deactivate()
}

// handleViewNote processes note selection from the notes list.
func (m *model) handleViewNote(msg tuitypes.ViewNoteMsg) {
	if msg.Note != nil {
		content := fmt.Sprintf("Note ID: %s\n", msg.Note.ID)
		content += fmt.Sprintf("Tags: [%s]\n", strings.Join(msg.Note.Tags, ", "))
		content += fmt.Sprintf("Created: %s\n", msg.Note.CreatedAt)
		content += fmt.Sprintf("Updated: %s\n\n", msg.Note.UpdatedAt)
		content += msg.Note.Content
		ol := overlay.NewToolResultOverlay("Note Detail", content, m.width, m.height)
		m.overlay.pushOverlay(tuitypes.OverlayModeToolResult, ol)
	}
}

// handleSlashCommandComplete processes slash command completion.
func (m *model) handleSlashCommandComplete() (tea.Model, tea.Cmd) {
	m.agentBusy = false
	m.recalculateLayout()
	return m, nil
}

// handleOperationStart processes operation start events.
func (m *model) handleOperationStart(msg operationStartMsg) (tea.Model, tea.Cmd) {
	m.agentBusy = true
	m.currentLoadingMessage = msg.message
	m.recalculateLayout()
	return m, nil
}

// handleOperationComplete processes operation completion events.
func (m *model) handleOperationComplete(msg operationCompleteMsg) (tea.Model, tea.Cmd) {
	m.agentBusy = false
	m.recalculateLayout()

	if msg.err != nil {
		m.showToast(msg.errorTitle, fmt.Sprintf("%v", msg.err), msg.errorIcon, true)
	} else {
		m.showToast(msg.successTitle, msg.result, msg.successIcon, false)
	}

	if m.overlay.isActive() {
		m.overlay.deactivate()
		m.recalculateLayout()
	}

	return m, nil
}

// handleToast processes toast notification messages.
func (m *model) handleToast(msg toastMsg) (tea.Model, tea.Cmd) {
	m.showToast(msg.message, msg.details, msg.icon, msg.isError)
	m.agentBusy = false
	m.recalculateLayout()

	// Close any active overlay after showing toast (handles rejection cases).
	if m.overlay.isActive() {
		m.overlay.deactivate()
		m.recalculateLayout()
	}

	return m, nil
}

// handleAgentError processes agent error messages.
func (m *model) handleAgentError(msg agentErrMsg) (tea.Model, tea.Cmd) {
	errMsg := truncateLines(fmt.Sprintf("%v", msg.err), 2)
	rendered := warningStyle.Render(fmt.Sprintf("  ⚠ %s", sanitizeOutput(errMsg)))
	m.appendMsg(newRawMsg(rendered, "\n\n"))
	m.agentBusy = false
	m.recalculateLayout()
	return m, nil
}

// handleBashCommandResult processes bash command execution results.
func (m *model) handleBashCommandResult(msg bashCommandResultMsg) (tea.Model, tea.Cmd) {
	header := fmt.Sprintf("[%s] Command completed:\n", msg.timestamp)
	m.appendMsg(newRawMsg(header+msg.result, "\n\n"))
	m.recalculateLayout()
	return m, nil
}

// handleApprovalRequest processes generic approval requests (slash commands, etc.).
func (m *model) handleApprovalRequest(msg approvalRequestMsg) (tea.Model, tea.Cmd) {
	// Use pushOverlay to handle concurrent approval requests by stacking them.
	ol := overlay.NewGenericApprovalOverlay(msg.request, m.width, m.height)
	m.overlay.pushOverlay(tuitypes.OverlayModeApproval, ol)
	return m, nil
}
