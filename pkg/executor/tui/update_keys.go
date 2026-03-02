package tui

import (
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrhq/forge/pkg/executor/tui/overlay"
	tuitypes "github.com/entrhq/forge/pkg/executor/tui/types"
)

// handleKeyPress processes keyboard input from the main Update loop.
func (m *model) handleKeyPress(msg tea.KeyMsg, vpCmd, tiCmd, spinnerCmd tea.Cmd) (tea.Model, tea.Cmd) {
	// If result list is active, pass keys to it.
	if m.resultList.IsActive() {
		updated, cmd := m.resultList.Update(msg, m, m)
		if rl, ok := updated.(*overlay.ResultListModel); ok {
			m.resultList = *rl
		}
		return m, tea.Batch(cmd, spinnerCmd)
	}

	// ADR-0048: scroll-lock keys are handled in a dedicated helper.
	if handled, mdl, cmd := m.handleScrollKey(msg, vpCmd, tiCmd, spinnerCmd); handled {
		return mdl, cmd
	}

	switch msg.Type {
	case tea.KeyEsc:
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

	case tea.KeyCtrlY:
		return m.handleCopyToClipboard()

	case tea.KeyEnter:
		if msg.Alt {
			m.textarea.InsertString("\n")
			m.updateTextAreaHeight()
			return m, nil
		}
		return m.handleEnter(tiCmd, vpCmd, spinnerCmd)
	}

	return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
}

// handleScrollKey handles ADR-0048 scroll-lock key events (PgUp, PgDn, g).
// Returns (handled, model, cmd) — if handled is false the caller should
// continue with normal key dispatch.
func (m *model) handleScrollKey(msg tea.KeyMsg, vpCmd, tiCmd, spinnerCmd tea.Cmd) (bool, tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyPgUp, tea.KeyCtrlB:
		m.followScroll = false
		m.viewport.HalfPageUp()
		return true, m, tea.Batch(tiCmd, vpCmd, spinnerCmd)

	case tea.KeyPgDown:
		m.viewport.HalfPageDown()
		if m.viewport.AtBottom() {
			m.resumeFollowScroll()
		}
		return true, m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
	}

	// "g" or "G": jump to bottom and resume auto-follow (mirrors vim / less).
	key := msg.String()
	if (key == "g" || key == "G") && !m.followScroll {
		m.resumeFollowScroll()
		m.viewport.GotoBottom()
		return true, m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
	}

	return false, m, nil
}

// handleCtrlC exits the TUI or exits bash mode if active.
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

// handleCtrlV opens the last tool result in an overlay.
func (m *model) handleCtrlV() (tea.Model, tea.Cmd) {
	if m.lastToolCallID != "" {
		if result, ok := m.resultCache.get(m.lastToolCallID); ok {
			ol := overlay.NewToolResultOverlay(result.ToolName, result.Result, m.width, m.height)
			m.overlay.activate(tuitypes.OverlayModeToolResult, ol)
		}
	}
	return m, nil
}

// handleCtrlL opens the full tool result history list.
func (m *model) handleCtrlL() (tea.Model, tea.Cmd) {
	results := m.resultCache.getAll()
	m.resultList.Activate(results, m.width, m.height)
	return m, nil
}

// handleCtrlK toggles the command palette.
func (m *model) handleCtrlK() (tea.Model, tea.Cmd) {
	if m.commandPalette.IsActive() {
		m.commandPalette.Deactivate()
	} else {
		m.commandPalette.Activate()
	}
	return m, nil
}

// handleCtrlP toggles the command palette (alternate binding).
func (m *model) handleCtrlP() (tea.Model, tea.Cmd) {
	if m.commandPalette.IsActive() {
		m.commandPalette.Deactivate()
	} else {
		m.commandPalette.Activate()
	}
	return m, nil
}

// handleCopyToClipboard copies the full conversation history to the OS clipboard
// and shows a brief toast confirmation (ADR-0050).
// Re-renders all messages at the viewport width so line wrapping matches what
// the user sees. ANSI escape codes are stripped before writing to clipboard.
func (m *model) handleCopyToClipboard() (tea.Model, tea.Cmd) {
	content := stripANSI(m.renderMessages(m.viewport.Width))
	if err := clipboard.WriteAll(content); err != nil {
		m.showToast("Clipboard unavailable", "No clipboard manager detected", "!", true)
		return m, nil
	}
	m.showToast("Copied to clipboard", "", "✓", false)
	return m, nil
}
