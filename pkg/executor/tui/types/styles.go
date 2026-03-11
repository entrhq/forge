package types

import "github.com/charmbracelet/lipgloss"

// Color Palette
// This is the single source of truth for all TUI colors.
// Use these constants throughout the TUI to ensure visual consistency.
var (
	// Primary Colors - Core brand colors
	SalmonPink  = lipgloss.Color("#FFB3BA") // Soft pastel salmon pink - primary accent
	CoralPink   = lipgloss.Color("#FFCCCB") // Lighter coral accent - secondary
	MintGreen   = lipgloss.Color("#A8E6CF") // Soft mint green - success/accept states
	MutedGray   = lipgloss.Color("#6B7280") // Muted gray - secondary text
	BrightWhite = lipgloss.Color("#F9FAFB") // Bright white - primary text
	DarkBg      = lipgloss.Color("#111827") // Dark background - container backgrounds

	// Semantic Colors - For specific UI states
	Black = lipgloss.Color("#000000") // Black - high contrast text on colored backgrounds

	// Diff Colors - For code diffs and syntax highlighting
	DiffAddColor      = lipgloss.Color("#90EE90") // Green for additions
	DiffDeleteColor   = lipgloss.Color("#FFB3BA") // Red for deletions (matches SalmonPink)
	DiffHunkColor     = lipgloss.Color("#87CEEB") // Cyan for hunk headers
	DiffHeaderColor   = lipgloss.Color("#FFA07A") // Orange for file headers
	DiffAddBgColor    = lipgloss.Color("#2d4a2b") // Dark green background for added lines
	DiffDeleteBgColor = lipgloss.Color("#4a2d2d") // Dark red background for deleted lines

	// UI Element Colors - For specific UI components
	PaletteBg      = lipgloss.Color("#2d2d2d") // Dark gray background for command palette
	ProgressGreen  = lipgloss.Color("#98C379") // Green for healthy progress bars
	ProgressYellow = lipgloss.Color("#E5C07B") // Yellow for warning progress bars
	ProgressRed    = lipgloss.Color("#E06C75") // Red for critical progress bars
	ProgressEmpty  = lipgloss.Color("#3E4451") // Dark gray for empty progress bars
)

// Common Styles
// These are pre-configured styles for common UI elements.
// Use these as base styles and customize as needed.
var (
	// Text Styles
	HeaderStyle = lipgloss.NewStyle().
			Foreground(SalmonPink).
			Bold(true)

	TipsStyle = lipgloss.NewStyle().
			Foreground(MutedGray)

	UserStyle = lipgloss.NewStyle().
			Foreground(CoralPink).
			Bold(true)

	ThinkingStyle = lipgloss.NewStyle().
			Foreground(MutedGray).
			Italic(true)

	ToolStyle = lipgloss.NewStyle().
			Foreground(MintGreen)

	ToolResultStyle = lipgloss.NewStyle().
			Foreground(BrightWhite)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(SalmonPink)

	BashPromptStyle = lipgloss.NewStyle().
			Foreground(MintGreen).
			Bold(true)

	// Container Styles
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(MutedGray).
			Padding(0, 1)

	InputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(SalmonPink).
			Padding(0, 1)
)

// Button Styles
// CreateButtonStyle creates a button style with the given foreground and background colors.
// Use this helper to create consistent button styling across the TUI.
func CreateButtonStyle(fg, bg lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Padding(0, 2).
		Foreground(fg).
		Background(bg)
}

// GetAcceptButtonStyle returns the style for an accept button based on selection state.
func GetAcceptButtonStyle(selected bool) lipgloss.Style {
	if selected {
		return CreateButtonStyle(Black, MintGreen)
	}
	return lipgloss.NewStyle().
		Bold(true).
		Padding(0, 2).
		Foreground(MutedGray)
}

// GetRejectButtonStyle returns the style for a reject button based on selection state.
func GetRejectButtonStyle(selected bool) lipgloss.Style {
	if selected {
		return CreateButtonStyle(Black, SalmonPink)
	}
	return lipgloss.NewStyle().
		Bold(true).
		Padding(0, 2).
		Foreground(MutedGray)
}

// CreateStyledSpacer creates a spacer.
// Use this to create gaps between UI elements.
func CreateStyledSpacer(width int) string {
	spacerStyle := lipgloss.NewStyle()
	return spacerStyle.Render(lipgloss.PlaceHorizontal(width, lipgloss.Left, ""))
}

// Overlay Container Styles
// CreateOverlayContainerStyle creates a standardized container style for all overlays.
// Uses straight corners and mutedGray border to match the flat design language (ADR-0051).
// Note: Only sets width, not height, to allow content to determine the container height naturally.
func CreateOverlayContainerStyle(width int) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(MutedGray).
		Padding(0, 1).
		Width(width)
}

// ComputeOverlayWidth returns a responsive overlay width clamped between minW and maxW.
// factor is a fraction of the terminal width (e.g. 0.80 for 80%).
func ComputeOverlayWidth(terminalWidth int, factor float64, minW, maxW int) int {
	w := max(int(float64(terminalWidth)*factor), minW)
	if w > maxW {
		w = maxW
	}
	return w
}

// ComputeViewportHeight returns the usable viewport height inside an overlay given the
// terminal height and the number of fixed "chrome" rows the overlay needs outside the viewport
// (title row, separator, footer rows, border rows, top/bottom margin).
func ComputeViewportHeight(terminalHeight, chromeRows int) int {
	h := max(
		// 2-line top margin + 2-line bottom margin
		terminalHeight-chromeRows-4, 3)
	return h
}

// Shared text styles for overlay content
var (
	// OverlayTitleStyle is used for main overlay titles
	OverlayTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(SalmonPink)

	// OverlaySubtitleStyle is used for overlay subtitles and secondary text
	OverlaySubtitleStyle = lipgloss.NewStyle().
				Foreground(MutedGray)

	// OverlayHelpStyle is used for help text and hints
	OverlayHelpStyle = lipgloss.NewStyle().
				Foreground(MutedGray).
				Italic(true)
)
