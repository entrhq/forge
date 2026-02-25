package overlay

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/executor/tui/types"
)

// CommandItem represents a command in the palette
type CommandItem struct {
	Name        string
	Description string
}

// CommandPalette manages command suggestions and selection
type CommandPalette struct {
	commands         []CommandItem
	filteredCommands []CommandItem
	selectedIndex    int
	filter           string
	active           bool
}

// NewCommandPalette creates a new command palette
func NewCommandPalette(commands []CommandItem) *CommandPalette {
	return &CommandPalette{
		commands:         commands,
		filteredCommands: commands,
		selectedIndex:    0,
		active:           false,
	}
}

// Activate shows the command palette
func (cp *CommandPalette) Activate() {
	cp.active = true
	cp.filter = ""
	cp.selectedIndex = 0
	cp.updateFiltered()
}

// Deactivate hides the command palette
func (cp *CommandPalette) Deactivate() {
	cp.active = false
	cp.filter = ""
	cp.selectedIndex = 0
}

// UpdateFilter updates the filter string and refreshes filtered commands
func (cp *CommandPalette) UpdateFilter(filter string) {
	newFilter := strings.ToLower(strings.TrimSpace(filter))
	// Only reset selection if the filter actually changed
	if newFilter != cp.filter {
		cp.filter = newFilter
		cp.selectedIndex = 0
		cp.updateFiltered()
	}
}

// updateFiltered updates the list of filtered commands based on current filter.
// Name matches are ranked before description-only matches so that typing a
// command prefix always surfaces the intended command at the top.
func (cp *CommandPalette) updateFiltered() {
	if cp.filter == "" {
		cp.filteredCommands = cp.commands
		return
	}

	var nameMatches, descMatches []CommandItem
	for _, cmd := range cp.commands {
		nameMatch := strings.Contains(strings.ToLower(cmd.Name), cp.filter)
		descMatch := strings.Contains(strings.ToLower(cmd.Description), cp.filter)
		switch {
		case nameMatch:
			nameMatches = append(nameMatches, cmd)
		case descMatch:
			descMatches = append(descMatches, cmd)
		}
	}
	nameMatches = append(nameMatches, descMatches...)
	cp.filteredCommands = nameMatches

	// Ensure selected index is valid after filtering
	switch {
	case len(cp.filteredCommands) == 0:
		cp.selectedIndex = 0
	case cp.selectedIndex >= len(cp.filteredCommands):
		cp.selectedIndex = len(cp.filteredCommands) - 1
	case cp.selectedIndex < 0:
		cp.selectedIndex = 0
	}
}

// SelectNext moves selection down
func (cp *CommandPalette) SelectNext() {
	if len(cp.filteredCommands) == 0 {
		return
	}
	cp.selectedIndex = (cp.selectedIndex + 1) % len(cp.filteredCommands)
}

// SelectPrev moves selection up
func (cp *CommandPalette) SelectPrev() {
	if len(cp.filteredCommands) == 0 {
		return
	}
	cp.selectedIndex--
	if cp.selectedIndex < 0 {
		cp.selectedIndex = len(cp.filteredCommands) - 1
	}
}

// GetSelected returns the currently selected command
func (cp *CommandPalette) GetSelected() *CommandItem {
	if len(cp.filteredCommands) == 0 ||
		cp.selectedIndex < 0 ||
		cp.selectedIndex >= len(cp.filteredCommands) {
		return nil
	}
	return &cp.filteredCommands[cp.selectedIndex]
}

// Render renders the command palette
func (cp *CommandPalette) Render(width int) string {
	if !cp.active || len(cp.filteredCommands) == 0 {
		return ""
	}

	var sb strings.Builder

	// Calculate palette width (80% of screen or max 80 chars)
	paletteWidth := width * 80 / 100
	if paletteWidth > 80 {
		paletteWidth = 80
	}
	if paletteWidth < 40 {
		paletteWidth = 40
	}

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(types.SalmonPink).
		Bold(true).
		PaddingLeft(1)

	sb.WriteString(headerStyle.Render("Available Commands:"))
	sb.WriteString("\n")

	// Show up to 5 commands
	maxVisible := 5
	if len(cp.filteredCommands) < maxVisible {
		maxVisible = len(cp.filteredCommands)
	}

	for i := 0; i < maxVisible; i++ {
		cmd := cp.filteredCommands[i]
		prefix := "  "
		if i == cp.selectedIndex {
			prefix = "> "
		}

		// Command name in salmon pink, description in soft gray
		cmdNameStyle := lipgloss.NewStyle().
			Foreground(types.SalmonPink).
			Bold(i == cp.selectedIndex)

		descStyle := lipgloss.NewStyle().
			Foreground(types.MutedGray)

		if i == cp.selectedIndex {
			// Highlighted background for selected item
			lineStyle := lipgloss.NewStyle().
				Background(types.PaletteBg).
				Width(paletteWidth - 2).
				PaddingLeft(1)

			line := prefix + cmdNameStyle.Render("/"+cmd.Name) + "  " + descStyle.Render(cmd.Description)
			sb.WriteString(lineStyle.Render(line))
		} else {
			line := prefix + cmdNameStyle.Render("/"+cmd.Name) + "  " + descStyle.Render(cmd.Description)
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	// Footer hint
	if len(cp.filteredCommands) > maxVisible {
		footerStyle := lipgloss.NewStyle().
			Foreground(types.MutedGray).
			Italic(true).
			PaddingLeft(1)
		sb.WriteString(footerStyle.Render("... and more. Keep typing to filter."))
		sb.WriteString("\n")
	}

	// Wrap in border
	paletteStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(types.SalmonPink).
		Width(paletteWidth).
		Padding(0, 1)

	return paletteStyle.Render(sb.String())
}

// IsActive returns whether the palette is active
func (cp *CommandPalette) IsActive() bool {
	return cp.active
}
