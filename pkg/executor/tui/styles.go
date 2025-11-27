package tui

import "github.com/charmbracelet/lipgloss"

// Color Palette
// This is the single source of truth for all TUI colors.
// Use these constants throughout the TUI to ensure visual consistency.
var (
	// Primary Colors - Core brand colors
	salmonPink  = lipgloss.Color("#FFB3BA") // Soft pastel salmon pink - primary accent
	coralPink   = lipgloss.Color("#FFCCCB") // Lighter coral accent - secondary
	mintGreen   = lipgloss.Color("#A8E6CF") // Soft mint green - success/accept states
	mutedGray   = lipgloss.Color("#6B7280") // Muted gray - secondary text
	brightWhite = lipgloss.Color("#F9FAFB") // Bright white - primary text
)

// Common Styles
// These are pre-configured styles for common UI elements.
// Use these as base styles and customize as needed.
var (
	// Text Styles
	headerStyle = lipgloss.NewStyle().
			Foreground(salmonPink).
			Bold(true)

	tipsStyle = lipgloss.NewStyle().
			Foreground(mutedGray)

	userStyle = lipgloss.NewStyle().
			Foreground(coralPink).
			Bold(true)

	thinkingStyle = lipgloss.NewStyle().
			Foreground(mutedGray).
			Italic(true)

	toolStyle = lipgloss.NewStyle().
			Foreground(mintGreen)

	toolResultStyle = lipgloss.NewStyle().
			Foreground(brightWhite)

	errorStyle = lipgloss.NewStyle().
			Foreground(salmonPink)

	bashPromptStyle = lipgloss.NewStyle().
			Foreground(mintGreen).
			Bold(true)

	// Container Styles
	statusBarStyle = lipgloss.NewStyle().
			Foreground(mutedGray).
			Padding(0, 1)

	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(salmonPink).
			Padding(0, 1)

	// OverlayTitleStyle is used for main overlay titles
	OverlayTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(salmonPink)

	// OverlaySubtitleStyle is used for overlay subtitles and secondary text
	OverlaySubtitleStyle = lipgloss.NewStyle().
				Foreground(mutedGray)

	// OverlayHelpStyle is used for help text and hints
	OverlayHelpStyle = lipgloss.NewStyle().
				Foreground(mutedGray).
				Italic(true)
)
