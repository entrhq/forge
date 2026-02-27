package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/entrhq/forge/pkg/executor/tui/types"
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
	loadingIndicator := m.buildLoadingIndicator()
	inputBox := m.buildInputBox()
	bottomBar := m.buildBottomBar()

	// Build viewport section
	viewportSection := m.viewport.View()

	// ADR-0048: scroll-lock indicator (shown only when user has scrolled up and new content arrived)
	scrollIndicator := m.buildScrollLockIndicator()

	// Assemble the base UI
	baseView := m.assembleBaseView(header, tips, viewportSection, scrollIndicator, loadingIndicator, inputBox, bottomBar)

	// Layer overlays
	return m.applyOverlays(baseView)
}

// buildHeader renders the compact single-line header bar.
// Total height: 2 lines (bar + separator).
func (m *model) buildHeader() string {
	modelName := ""
	if m.provider != nil {
		modelName = m.provider.GetModel()
	}

	cwd := m.workspaceDir
	if abs, err := filepath.Abs(cwd); err == nil {
		cwd = abs
	}
	if m.width > 0 && lipgloss.Width(cwd) > m.width/2 {
		cwd = "…" + cwd[len(cwd)-(m.width/2):]
	}

	left := headerStyle.Render("⬡ forge")
	mid := tipsStyle.Render(cwd)
	right := tipsStyle.Render(modelName)

	totalUsed := lipgloss.Width(left) + lipgloss.Width(mid) + lipgloss.Width(right)
	gap := (m.width - totalUsed) / 2
	if gap < 1 {
		gap = 1
	}
	pad := strings.Repeat(" ", gap)

	bar := left + pad + mid + pad + right
	separator := tipsStyle.Render(strings.Repeat("─", m.width))
	return bar + "\n" + separator
}

// buildTips returns a single-line hints string adapted to the current TUI state.
func (m *model) buildTips() string {
	switch {
	case m.overlay.isActive():
		return tipsStyle.Render("  Esc · close   Tab · next field   Enter · confirm")

	case m.agentBusy:
		hints := "  Ctrl+C · interrupt"
		if !m.followScroll {
			hints += "   G · follow output"
		}
		return tipsStyle.Render(hints)

	case m.bashMode:
		return tipsStyle.Render("  Enter · run   exit · return to normal   Ctrl+C · cancel")

	default:
		return tipsStyle.Render("  Enter · send   Alt+Enter · new line   / · commands   Ctrl+Y · copy   Ctrl+C · exit")
	}
}

// buildScrollLockIndicator renders a "↓ New content below" hint when the user
// has scrolled up and new agent output has arrived (ADR-0048).
func (m *model) buildScrollLockIndicator() string {
	if m.followScroll || !m.hasNewContent {
		return ""
	}
	return scrollLockIndicatorStyle.
		Width(m.width - 4).
		Align(lipgloss.Center).
		Render("↓  New content below  — press G or PgDn to follow")
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

// buildInputBox renders the Option B input zone:
//
//	──────────────────────── (rule, full width)
//	❯ <textarea>
func (m *model) buildInputBox() string {
	rule := tipsStyle.Render(strings.Repeat("─", m.width))
	var prompt string
	if m.bashMode {
		prompt = bashPromptStyle.Render("❯")
	} else {
		prompt = inputPromptStyle.Render("❯")
	}
	input := m.textarea.View()
	return rule + "\n" + prompt + " " + input
}

// buildBottomBar renders the bottom status bar: mode indicator (left) + token usage (right).
func (m *model) buildBottomBar() string {
	var left string
	if m.bashMode {
		left = lipgloss.NewStyle().Foreground(mintGreen).Bold(true).Render("bash mode")
	}

	right := m.buildTokenDisplay()

	// Subtract 2 for the Padding(0, 1) on statusBarStyle (1 char each side)
	gap := m.width - 2 - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return statusBarStyle.Width(m.width).Render(
		left + strings.Repeat(" ", gap) + right,
	)
}

// buildTokenDisplay renders the token usage statistics with a context progress bar.
// Format: ctx ████░░░░ 12k / 128k
func (m *model) buildTokenDisplay() string {
	if m.totalTokens == 0 {
		return ""
	}

	var ctxPart string
	if m.maxContextTokens > 0 {
		pct := float64(m.currentContextTokens) / float64(m.maxContextTokens)

		// 8-cell progress bar
		const barWidth = 8
		filled := int(pct * barWidth)
		if filled > barWidth {
			filled = barWidth
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		// Color bar by usage level
		var barColor lipgloss.Color
		switch {
		case pct >= 0.95:
			barColor = lipgloss.Color("203") // red
		case pct >= 0.80:
			barColor = lipgloss.Color("214") // orange
		default:
			barColor = lipgloss.Color("#A8E6CF") // mintGreen
		}

		barStr := lipgloss.NewStyle().Foreground(barColor).Render(bar)
		ctxPart = fmt.Sprintf("ctx %s %s / %s",
			barStr,
			formatTokenCount(m.currentContextTokens),
			formatTokenCount(m.maxContextTokens))
	} else {
		ctxPart = fmt.Sprintf("ctx %s", formatTokenCount(m.currentContextTokens))
	}

	return ctxPart
}

// assembleBaseView combines all UI components into the base view.
// scrollIndicator is the ADR-0048 "new content" banner (empty string when not needed).
func (m *model) assembleBaseView(header, tips, viewportSection, scrollIndicator, loadingIndicator, inputBox, bottomBar string) string {
	// Collect the rows that always appear between the viewport and the input box
	var middle []string
	middle = append(middle, viewportSection)
	if scrollIndicator != "" {
		middle = append(middle, scrollIndicator)
	}
	if m.agentBusy {
		middle = append(middle, loadingIndicator)
	}

	rows := []string{header, tips, ""}
	rows = append(rows, middle...)
	rows = append(rows, inputBox, bottomBar)
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
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
		paletteContent := m.commandPalette.Render(m.width, m.height)
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

	// Calculate width matching redesign
	boxWidth := types.ComputeOverlayWidth(m.width, 0.70, 40, 90)

	var content strings.Builder

	// Header - Redesign to match mockups: salmon title + muted gray separator line
	headerStyle := lipgloss.NewStyle().Foreground(salmonPink).Bold(true)
	title := fmt.Sprintf("Optimizing context... [%s]", m.summarization.strategy)
	content.WriteString(headerStyle.Render(title))
	content.WriteString("\n")

	sepStr := ""
	for i := 0; i < boxWidth; i++ {
		sepStr += "─"
	}
	content.WriteString(lipgloss.NewStyle().Foreground(mutedGray).Render(sepStr))
	content.WriteString("\n\n")

	// Progress bar
	barWidth := boxWidth - 10
	if barWidth < 20 {
		barWidth = 20
	}

	filledWidth := int(float64(barWidth) * m.summarization.progressPercent / 100.0)
	if filledWidth > barWidth {
		filledWidth = barWidth
	}
	emptyWidth := barWidth - filledWidth

	// Use solid block for filled, sparse block for empty
	progressStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A8E6CF")) // Mint green
	emptyStyle := lipgloss.NewStyle().Foreground(mutedGray)

	barFilled := ""
	for i := 0; i < filledWidth; i++ {
		barFilled += "█"
	}
	for i := 0; i < emptyWidth; i++ {
		barFilled += "░"
	}

	bar := progressStyle.Render(barFilled[:filledWidth*3]) + emptyStyle.Render(barFilled[filledWidth*3:])

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
		content.WriteString(lipgloss.NewStyle().Foreground(mutedGray).Render(m.summarization.currentItem))
	} else if m.summarization.totalItems > 0 {
		content.WriteString(lipgloss.NewStyle().Foreground(mutedGray).Render(fmt.Sprintf("Processing item %d of %d...",
			m.summarization.itemsProcessed, m.summarization.totalItems)))
	}

	// Create styled box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(mutedGray).
		Padding(0, 1).
		Width(boxWidth)

	return "\n" + boxStyle.Render(content.String()) + "\n"
}

// renderToast renders a toast notification
func (m *model) renderToast() string {
	if !m.toast.active || time.Now().After(m.toast.showUntil) {
		return ""
	}

	// Calculate width matching redesign
	boxWidth := types.ComputeOverlayWidth(m.width, 0.70, 40, 90)

	var content strings.Builder

	// Determine base style per type
	baseColor := mutedGray
	if m.toast.isError {
		baseColor = lipgloss.Color("203") // Red color for errors
	}

	// Icon and message
	header := fmt.Sprintf("%s %s", m.toast.icon, m.toast.message)

	// If no details, render as flat boxless string (Option B)
	if m.toast.details == "" {
		flatStyle := lipgloss.NewStyle().
			Foreground(baseColor).
			Padding(0, 1).
			Width(boxWidth)
		return "\n" + flatStyle.Render(header) + "\n"
	}

	content.WriteString(header)
	content.WriteString("\n")

	// Details (Option C: Normal border)
	detailStyle := lipgloss.NewStyle().Foreground(mutedGray)
	content.WriteString(detailStyle.Render(m.toast.details))

	// Create styled box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(baseColor).
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
