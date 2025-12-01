package overlay

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/executor/tui/types"
)

// ContextOverlay displays detailed context information in a modal dialog
type ContextOverlay struct {
	*BaseOverlay
	title string
}

// ContextInfo contains all context statistics to display
type ContextInfo struct {
	// System prompt
	SystemPromptTokens int
	CustomInstructions bool

	// Tool system
	ToolCount          int
	ToolTokens         int
	ToolNames          []string
	CurrentToolCall    string
	HasPendingToolCall bool

	// Message history
	MessageCount       int
	ConversationTurns  int
	ConversationTokens int

	// Token usage - current context
	CurrentContextTokens int
	MaxContextTokens     int
	FreeTokens           int
	UsagePercent         float64

	// Token usage - cumulative across all API calls
	TotalPromptTokens     int
	TotalCompletionTokens int
	TotalTokens           int
}

// NewContextOverlay creates a new context information overlay
func NewContextOverlay(info *ContextInfo, width, height int) *ContextOverlay {
	content := buildContextContent(info)

	overlay := &ContextOverlay{
		title: "Context Information",
	}

	// Use fixed width like help overlay for consistent centered appearance
	overlayWidth := 80
	overlayHeight := 24

	// Configure base overlay
	baseConfig := BaseOverlayConfig{
		Width:          overlayWidth,
		Height:         overlayHeight,
		ViewportWidth:  76,
		ViewportHeight: 20,
		Content:        content,
		OnClose: func(actions types.ActionHandler) tea.Cmd {
			// Return nil to signal close - caller will handle ClearOverlay()
			return nil
		},
		RenderHeader: overlay.renderHeader,
		RenderFooter: overlay.renderFooter,
	}

	overlay.BaseOverlay = NewBaseOverlay(baseConfig)
	return overlay
}

// formatTokenCount formats a token count with K/M suffixes for readability
func formatTokenCount(count int) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
	if count >= 1000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%d", count)
}

// buildContextContent formats the context information for display
func buildContextContent(info *ContextInfo) string {
	var b strings.Builder

	// System section
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("System"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  System Prompt:      %s tokens\n", formatTokenCount(info.SystemPromptTokens)))
	if info.CustomInstructions {
		b.WriteString("  Custom Instructions: Yes\n")
	} else {
		b.WriteString("  Custom Instructions: No\n")
	}
	b.WriteString("\n")

	// Tool System section
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Tool System"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Available Tools:    %d (%s tokens)\n", info.ToolCount, formatTokenCount(info.ToolTokens)))
	if info.HasPendingToolCall {
		b.WriteString(fmt.Sprintf("  Current Tool Call:  %s\n", info.CurrentToolCall))
	}
	b.WriteString("\n")

	// History section
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Message History"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Messages:           %d\n", info.MessageCount))
	b.WriteString(fmt.Sprintf("  Conversation Turns: %d\n", info.ConversationTurns))
	b.WriteString(fmt.Sprintf("  Conversation:       %s tokens\n", formatTokenCount(info.ConversationTokens)))
	b.WriteString("\n")

	// Current Context section
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Current Context"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Used:               %s / %s tokens (%.1f%%)\n",
		formatTokenCount(info.CurrentContextTokens),
		formatTokenCount(info.MaxContextTokens),
		info.UsagePercent))
	b.WriteString(fmt.Sprintf("  Free Space:         %s tokens\n", formatTokenCount(info.FreeTokens)))

	// Add a progress bar
	barWidth := 40
	filledWidth := int(float64(barWidth) * info.UsagePercent / 100.0)
	emptyWidth := barWidth - filledWidth

	var barColor lipgloss.Color
	switch {
	case info.UsagePercent < 70:
		barColor = types.ProgressGreen // Green
	case info.UsagePercent < 90:
		barColor = types.ProgressYellow // Yellow
	default:
		barColor = types.ProgressRed // Red
	}

	filled := lipgloss.NewStyle().Foreground(barColor).Render(strings.Repeat("█", filledWidth))
	empty := lipgloss.NewStyle().Foreground(types.ProgressEmpty).Render(strings.Repeat("░", emptyWidth))
	b.WriteString(fmt.Sprintf("  [%s%s]\n", filled, empty))
	b.WriteString("\n")

	// Cumulative Token Usage section
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Cumulative Usage (All API Calls)"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Input Tokens:       %s\n", formatTokenCount(info.TotalPromptTokens)))
	b.WriteString(fmt.Sprintf("  Output Tokens:      %s\n", formatTokenCount(info.TotalCompletionTokens)))
	b.WriteString(fmt.Sprintf("  Total:              %s\n", formatTokenCount(info.TotalTokens)))

	return b.String()
}

// Update handles messages for the context overlay
func (c *ContextOverlay) Update(msg tea.Msg, state types.StateProvider, actions types.ActionHandler) (types.Overlay, tea.Cmd) {
	handled, updatedBase, cmd := c.BaseOverlay.Update(msg, actions)
	c.BaseOverlay = updatedBase

	if handled {
		return c, cmd
	}

	// Handle Enter key to close (in addition to Esc)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter {
			return c, c.BaseOverlay.close(actions)
		}
	}

	return c, nil
}

// renderHeader renders the context overlay header
func (c *ContextOverlay) renderHeader() string {
	return types.OverlayTitleStyle.Render(c.title)
}

// renderFooter renders the context overlay footer
func (c *ContextOverlay) renderFooter() string {
	return types.OverlayHelpStyle.Render("Press ESC or Enter to close • ↑/↓ to scroll")
}

// View renders the overlay
func (c *ContextOverlay) View() string {
	return c.BaseOverlay.View(c.Width())
}
