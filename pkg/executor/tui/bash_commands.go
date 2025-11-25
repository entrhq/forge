package tui

import (
	"context"
	"fmt"
	"html"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/tools/coding"
	"github.com/entrhq/forge/pkg/types"
)

// executeBashCommand executes a shell command directly, bypassing the agent
func (m *model) executeBashCommand(command string) tea.Cmd {
	return func() tea.Msg {
		// Get the tool from agent's tool registry
		toolIface := m.agent.GetTool("execute_command")
		if toolIface == nil {
			return toastMsg{
				message: "Error",
				details: "execute_command tool not available",
				icon:    "❌",
				isError: true,
			}
		}

		// Type assert to tools.Tool
		tool, ok := toolIface.(tools.Tool)
		if !ok {
			return toastMsg{
				message: "Error",
				details: "execute_command tool has invalid type",
				icon:    "❌",
				isError: true,
			}
		}

		// Prepare XML arguments for the tool
		argsXML := fmt.Sprintf("<arguments><command>%s</command></arguments>", html.EscapeString(command))

		// Create context with event emitter for streaming support
		// Note: The execute_command tool will send its own CommandExecutionStart event
		ctx := context.WithValue(context.Background(), coding.EventEmitterKey, coding.EventEmitter(func(event *types.AgentEvent) {
			m.channels.Event <- event
		}))

		// Execute the tool
		result, _, err := tool.Execute(ctx, []byte(argsXML))

		// Send completion or error event
		if err != nil {
			return toastMsg{
				message: "Command Failed",
				details: fmt.Sprintf("Error: %v", err),
				icon:    "❌",
				isError: true,
			}
		}

		// Return result to display in viewport
		timestamp := time.Now().Format("15:04:05")
		return bashCommandResultMsg{
			timestamp: timestamp,
			command:   command,
			result:    fmt.Sprintf("%v", result),
		}
	}
}

// bashCommandResultMsg is sent when a bash command completes
type bashCommandResultMsg struct {
	timestamp string
	command   string
	result    string
}

// updatePrompt changes the textarea prompt based on current mode
func (m *model) updatePrompt() {
	if m.bashMode {
		m.textarea.Prompt = "bash> "
		m.textarea.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(mintGreen)
	} else {
		m.textarea.Prompt = "> "
		m.textarea.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(salmonPink)
	}
}
