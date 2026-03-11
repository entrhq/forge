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
	SystemPromptTokens      int
	CustomInstructions      bool
	RepositoryContextTokens int

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

	// History composition breakdown
	RawMessageCount      int
	RawMessageTokens     int
	SummaryBlockCount    int
	SummaryBlockTokens   int
	GoalBatchBlockCount  int
	GoalBatchBlockTokens int

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
	overlayWidth := types.ComputeOverlayWidth(width, 0.80, 56, 100)
	viewportHeight := types.ComputeViewportHeight(height, 5)
	overlayHeight := viewportHeight + 5

	content := buildContextContent(info)

	overlay := &ContextOverlay{
		title: "Context Information",
	}

	// Configure base overlay
	baseConfig := BaseOverlayConfig{
		Width:          overlayWidth,
		Height:         overlayHeight,
		ViewportWidth:  overlayWidth - 4,
		ViewportHeight: viewportHeight,
		Content:        content,
		OnClose: func(actions types.ActionHandler) tea.Cmd {
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
	fmt.Fprintf(&b, "  System Prompt:      %s tokens\n", formatTokenCount(info.SystemPromptTokens))
	if info.CustomInstructions {
		b.WriteString("  Custom Instructions: Yes\n")
	} else {
		b.WriteString("  Custom Instructions: No\n")
	}
	if info.RepositoryContextTokens > 0 {
		fmt.Fprintf(&b, "  Repository Context (AGENTS.md): %s tokens\n", formatTokenCount(info.RepositoryContextTokens))
	}
	b.WriteString("\n")

	// Tool System section
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Tool System"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "  Available Tools:    %d (%s tokens)\n", info.ToolCount, formatTokenCount(info.ToolTokens))
	if info.HasPendingToolCall {
		fmt.Fprintf(&b, "  Current Tool Call:  %s\n", info.CurrentToolCall)
	}
	b.WriteString("\n")

	// History section
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Message History"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "  Messages:           %d\n", info.MessageCount)
	fmt.Fprintf(&b, "  Conversation Turns: %d\n", info.ConversationTurns)
	fmt.Fprintf(&b, "  Conversation:       %s tokens\n", formatTokenCount(info.ConversationTokens))
	b.WriteString("\n")

	// History composition section
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("History Composition"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "  Raw Messages:       %d (%s tokens)\n",
		info.RawMessageCount, formatTokenCount(info.RawMessageTokens))
	fmt.Fprintf(&b, "  Summarized Blocks:  %d (%s tokens)\n",
		info.SummaryBlockCount, formatTokenCount(info.SummaryBlockTokens))
	fmt.Fprintf(&b, "  Goal Batch Blocks:  %d (%s tokens)\n",
		info.GoalBatchBlockCount, formatTokenCount(info.GoalBatchBlockTokens))
	// Show compression ratio if any summarization has occurred
	if info.SummaryBlockCount > 0 || info.GoalBatchBlockCount > 0 {
		totalCompressed := info.SummaryBlockTokens + info.GoalBatchBlockTokens
		totalConv := info.ConversationTokens
		if totalConv > 0 {
			compressedPct := float64(totalCompressed) / float64(totalConv) * 100.0
			fmt.Fprintf(&b, "  Compressed Content: %.1f%% of conversation\n", compressedPct)
		}
	}
	b.WriteString("\n")

	// Current Context section
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Current Context"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "  Used:               %s / %s tokens (%.1f%%)\n",
		formatTokenCount(info.CurrentContextTokens),
		formatTokenCount(info.MaxContextTokens),
		info.UsagePercent)
	fmt.Fprintf(&b, "  Free Space:         %s tokens\n", formatTokenCount(info.FreeTokens))

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
	fmt.Fprintf(&b, "  [%s%s]\n", filled, empty)
	b.WriteString("\n")

	// Cumulative Token Usage section
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(types.SalmonPink).Render("Cumulative Usage (All API Calls)"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "  Input Tokens:       %s\n", formatTokenCount(info.TotalPromptTokens))
	fmt.Fprintf(&b, "  Output Tokens:      %s\n", formatTokenCount(info.TotalCompletionTokens))
	fmt.Fprintf(&b, "  Total:              %s\n", formatTokenCount(info.TotalTokens))

	return b.String()
}

// Update handles messages for the context overlay
func (c *ContextOverlay) Update(msg tea.Msg, state types.StateProvider, actions types.ActionHandler) (types.Overlay, tea.Cmd) {
	handled, updatedBase, cmd := c.BaseOverlay.Update(msg, actions)
	c.BaseOverlay = updatedBase

	if handled {
		// Check if this is a close key (ESC, Ctrl+C)
		// When BaseOverlay handles a close key, we should return nil to signal overlay close
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == keyEsc || keyMsg.String() == keyCtrlC {
				return nil, cmd
			}
		}
		return c, cmd
	}

	// Handle Enter key to close (in addition to Esc)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter {
			return nil, c.close(actions)
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" || msg.String() == "ctrl+c" {
			return nil, nil // signal close
		}
	case tea.WindowSizeMsg:
		newOverlayWidth := types.ComputeOverlayWidth(msg.Width, 0.80, 56, 100)
		vpHeight := types.ComputeViewportHeight(msg.Height, 5)

		c.SetDimensions(newOverlayWidth, vpHeight+5)

		vp := c.Viewport()
		vp.Width = newOverlayWidth - 4
		vp.Height = vpHeight
		// Viewport will re-wrap content automatically when dimensions change
	}

	return c, nil
}

// renderHeader renders the context overlay header
func (c *ContextOverlay) renderHeader() string {
	contentWidth := c.BaseOverlay.Viewport().Width

	titleLen := len(c.title)
	titlePadding := max(0, (contentWidth-titleLen)/2)

	var titleStr strings.Builder
	for range titlePadding {
		titleStr.WriteString(" ")
	}
	titleStr.WriteString(types.OverlayTitleStyle.Render(c.title))

	separator := lipgloss.NewStyle().Foreground(types.MutedGray).Render(strings.Repeat(sepChar, contentWidth))

	return titleStr.String() + "\n" + separator + "\n"
}

// renderFooter renders the context overlay footer
func (c *ContextOverlay) renderFooter() string {
	contentWidth := c.BaseOverlay.Viewport().Width

	hint := "ESC or Enter to close • ↑/↓ to scroll"
	hintLen := lipgloss.Width(hint)
	hintPadding := max(0, (contentWidth-hintLen)/2)
	var padStr strings.Builder
	for range hintPadding {
		padStr.WriteString(" ")
	}

	return "\n" + padStr.String() + types.OverlayHelpStyle.Render(hint)
}

// View renders the overlay
func (c *ContextOverlay) View() string {
	return c.BaseOverlay.View(c.Width())
}
