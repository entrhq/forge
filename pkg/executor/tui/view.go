package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// View renders the entire TUI interface.
// This is called by Bubble Tea whenever the UI needs to be redrawn.
func (m *model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Build header and status sections
	header := m.buildHeader()
	tips := m.buildTips()
	topStatus := m.buildTopStatus()
	loadingIndicator := m.buildLoadingIndicator()
	inputBox := m.buildInputBox()
	bottomBar := m.buildBottomBar()

	// Build viewport section
	viewportSection := m.viewport.View()

	// Assemble the base UI
	baseView := m.assembleBaseView(header, tips, topStatus, viewportSection, loadingIndicator, inputBox, bottomBar)

	// Layer overlays
	return m.applyOverlays(baseView)
}

// buildHeader renders the Forge ASCII art header
func (m *model) buildHeader() string {
	return headerStyle.Render(`
	███████╗ ██████╗ ██████╗  ██████╗ ███████╗
	██╔════╝██╔═══██╗██╔══██╗██╔════╝ ██╔════╝
	█████╗  ██║   ██║██████╔╝██║  ███╗█████╗
	██╔══╝  ██║   ██║██╔══██╗██║   ██║██╔══╝
	██║     ╚██████╔╝██║  ██║╚██████╔╝███████╗
	╚═╝      ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚══════╝`)
}

// buildTips renders context-sensitive usage tips
func (m *model) buildTips() string {
	if m.bashMode {
		return tipsStyle.Render(`  Bash Mode: Commands execute directly • Type 'exit' or Ctrl+C to return • Enter to run`)
	}
	return tipsStyle.Render(`  Tips: Ask questions • Alt+Enter for new line • Enter to send • !cmd for bash • /bash for mode • Ctrl+V to view last tool result • Ctrl+L for result history • Ctrl+C to exit`)
}

// buildTopStatus renders the working directory status bar
func (m *model) buildTopStatus() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "~"
	}
	return statusBarStyle.Render(fmt.Sprintf(" Working directory: %s", cwd))
}

// buildLoadingIndicator renders the loading spinner when agent is busy
func (m *model) buildLoadingIndicator() string {
	if !m.agentBusy {
		return ""
	}
	loadingMsg := fmt.Sprintf("%s %s", m.spinner.View(), m.currentLoadingMessage)
	loadingStyle := lipgloss.NewStyle().
		Foreground(salmonPink).
		Width(m.width-4).
		Padding(0, 2)
	return loadingStyle.Render(loadingMsg)
}

// buildInputBox renders the text input area
func (m *model) buildInputBox() string {
	return inputBoxStyle.Width(m.width - 4).Render(m.textarea.View())
}

// buildBottomBar renders the bottom status bar with token usage
func (m *model) buildBottomBar() string {
	bottomLeft := "~/forge"
	bottomCenter := "Enter to send • Alt+Enter for new line"
	if m.bashMode {
		bottomCenter = "🔧 BASH MODE • Enter to run • 'exit' to return"
	}
	bottomRight := m.buildTokenDisplay()

	totalUsed := len(bottomLeft) + len(bottomCenter) + len(bottomRight)
	leftPadding := (m.width - totalUsed) / 3
	rightPadding := m.width - totalUsed - leftPadding*2
	if leftPadding < 2 {
		leftPadding = 2
	}
	if rightPadding < 2 {
		rightPadding = 2
	}

	return statusBarStyle.Width(m.width).Render(
		bottomLeft +
			strings.Repeat(" ", leftPadding) +
			bottomCenter +
			strings.Repeat(" ", rightPadding) +
			bottomRight,
	)
}

// buildTokenDisplay renders the token usage statistics
func (m *model) buildTokenDisplay() string {
	if m.totalTokens == 0 {
		return "Forge Agent"
	}

	contextStr := formatTokenCount(m.currentContextTokens)
	if m.maxContextTokens > 0 {
		contextStr = fmt.Sprintf("%s/%s", contextStr, formatTokenCount(m.maxContextTokens))
		percentage := float64(m.currentContextTokens) / float64(m.maxContextTokens) * 100
		if percentage >= 80 {
			contextStr = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render(contextStr)
		}
	}

	return fmt.Sprintf("◆ Context: %s | Input: %s | Output: %s | Total: %s",
		contextStr,
		formatTokenCount(m.totalPromptTokens),
		formatTokenCount(m.totalCompletionTokens),
		formatTokenCount(m.totalTokens))
}

// assembleBaseView combines all UI components into the base view
func (m *model) assembleBaseView(header, tips, topStatus, viewportSection, loadingIndicator, inputBox, bottomBar string) string {
	if m.agentBusy {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			tips,
			topStatus,
			"",
			viewportSection,
			loadingIndicator,
			inputBox,
			bottomBar,
		)
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		tips,
		topStatus,
		"",
		viewportSection,
		inputBox,
		bottomBar,
	)
}

// applyOverlays layers all active overlays on top of the base view
func (m *model) applyOverlays(baseView string) string {
	if m.overlay.isActive() {
		baseView = renderOverlay(baseView, m.overlay.overlay, m.width, m.height)
	}

	if m.resultList.IsActive() {
		baseView = renderOverlay(baseView, &m.resultList, m.width, m.height)
	}

	if m.commandPalette.IsActive() {
		paletteContent := m.commandPalette.Render(m.width)
		baseView = renderToastOverlay(baseView, paletteContent)
	}

	if m.summarization.active {
		summarizationContent := m.renderSummarizationStatus()
		baseView = renderToastOverlay(baseView, summarizationContent)
	}

	// Add toast notification as overlay if active and not expired
	if m.toast.active && time.Now().Before(m.toast.showUntil) {
		toastContent := m.renderToast()
		baseView = renderToastOverlay(baseView, toastContent)
	}

	return baseView
}

// renderSummarizationStatus renders the context summarization status overlay
func (m *model) renderSummarizationStatus() string {
	if !m.summarization.active {
		return ""
	}

	// Create box with border
	boxWidth := m.width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}

	var content strings.Builder

	// Header line with brain icon and message
	header := fmt.Sprintf("🧠 Optimizing context... [%s]", m.summarization.strategy)
	content.WriteString(header)
	content.WriteString("\n")

	// Progress bar
	barWidth := boxWidth - 10 // Leave room for percentage
	if barWidth < 20 {
		barWidth = 20
	}

	filledWidth := int(float64(barWidth) * m.summarization.progressPercent / 100.0)
	if filledWidth > barWidth {
		filledWidth = barWidth
	}

	bar := strings.Repeat("━", filledWidth) + strings.Repeat("━", barWidth-filledWidth)
	// Show both item count and percentage
	if m.summarization.totalItems > 0 {
		progressLine := fmt.Sprintf("%s %d/%d items (%.0f%%)",
			bar, m.summarization.itemsProcessed, m.summarization.totalItems, m.summarization.progressPercent)
		content.WriteString(progressLine)
	} else {
		progressLine := fmt.Sprintf("%s %.0f%%", bar, m.summarization.progressPercent)
		content.WriteString(progressLine)
	}
	content.WriteString("\n")

	// Current item description
	if m.summarization.currentItem != "" {
		content.WriteString(m.summarization.currentItem)
	} else if m.summarization.totalItems > 0 {
		content.WriteString(fmt.Sprintf("Processing item %d of %d...",
			m.summarization.itemsProcessed, m.summarization.totalItems))
	}

	// Create styled box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(salmonPink).
		Padding(0, 1).
		Width(boxWidth)

	return "\n" + boxStyle.Render(content.String()) + "\n"
}

// renderToast renders a toast notification
func (m *model) renderToast() string {
	if !m.toast.active || time.Now().After(m.toast.showUntil) {
		return ""
	}

	// Create box with border
	boxWidth := m.width - 4
	if boxWidth < 40 {
		boxWidth = 40
	}

	var content strings.Builder

	// Icon and message
	header := fmt.Sprintf("%s %s", m.toast.icon, m.toast.message)
	content.WriteString(header)
	content.WriteString("\n")

	// Details
	if m.toast.details != "" {
		content.WriteString(m.toast.details)
	}

	// Create styled box
	borderColor := salmonPink
	if m.toast.isError {
		borderColor = lipgloss.Color("203") // Red color for errors
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(boxWidth)

	return "\n" + boxStyle.Render(content.String()) + "\n"
}

// showToast displays a toast notification to the user
func (m *model) showToast(message, details, icon string, isError bool) {
	m.toast.active = true
	m.toast.message = message
	m.toast.details = details
	m.toast.icon = icon
	m.toast.isError = isError
	m.toast.showUntil = time.Now().Add(3 * time.Second)
}
