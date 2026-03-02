package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrhq/forge/pkg/types"
)

// handleEnter handles Enter key press — dispatches to the appropriate input handler.
func (m *model) handleEnter(tiCmd, vpCmd, spinnerCmd tea.Cmd) (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textarea.Value())

	if input == "" {
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
	}

	switch {
	case m.bashMode:
		return m.handleBashModeInput(input, tiCmd, vpCmd, spinnerCmd)
	case strings.HasPrefix(input, "/"):
		return m.handleSlashCommand(input, tiCmd, vpCmd, spinnerCmd)
	case strings.HasPrefix(input, "!"):
		return m.handleSingleShotBash(input, tiCmd, vpCmd, spinnerCmd)
	default:
		return m.handleAgentMessage(input, tiCmd, vpCmd, spinnerCmd)
	}
}

// handleBashModeInput processes input while the TUI is in persistent bash mode.
func (m *model) handleBashModeInput(input string, tiCmd, vpCmd, spinnerCmd tea.Cmd) (tea.Model, tea.Cmd) {
	if input == "exit" {
		m.bashMode = false
		m.textarea.Reset()
		m.recalculateLayout()
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
	}

	m.appendMsg(newRawMsg(bashPromptStyle.Render(fmt.Sprintf("$ %s", input)), "\n"))
	m.textarea.Reset()
	m.recalculateLayout()

	return m, tea.Batch(tiCmd, vpCmd, spinnerCmd, m.executeBashCommand(input))
}

// handleSlashCommand processes slash commands entered in the input box.
// Slash commands are not displayed in the chat history — they execute silently.
func (m *model) handleSlashCommand(input string, tiCmd, vpCmd, spinnerCmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.textarea.Reset()

	commandName, args, ok := parseSlashCommand(input)
	if !ok {
		m.showToast("Invalid command", "Could not parse slash command", "✗", true)
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
	}

	updatedModel, cmd := executeSlashCommand(m, commandName, args)
	return updatedModel, tea.Batch(tiCmd, vpCmd, spinnerCmd, cmd)
}

// handleSingleShotBash processes a one-off bash command prefixed with '!'.
func (m *model) handleSingleShotBash(input string, tiCmd, vpCmd, spinnerCmd tea.Cmd) (tea.Model, tea.Cmd) {
	bashCmd := strings.TrimSpace(input[1:])
	if bashCmd == "" {
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
	}

	m.appendMsg(newRawMsg(bashPromptStyle.Render(fmt.Sprintf("$ %s", bashCmd)), "\n"))
	m.textarea.Reset()
	m.recalculateLayout()

	return m, tea.Batch(tiCmd, vpCmd, spinnerCmd, m.executeBashCommand(bashCmd))
}

// handleAgentMessage sends a regular user message to the agent.
func (m *model) handleAgentMessage(input string, tiCmd, vpCmd, spinnerCmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.appendMsg(newEntryMsg("❯ ", input, userStyle, "\n\n"))
	m.textarea.Reset()

	m.agentBusy = true
	m.currentLoadingMessage = getRandomLoadingMessage()
	m.recalculateLayout()

	userInput := types.NewUserInput(input)
	m.channels.Input <- userInput

	return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
}
