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

	// Calculate palette width (70% standard)
	paletteWidth := types.ComputeOverlayWidth(width, 0.70, 40, 90)
	
	// innerWidth accounts for border (2) and padding (2)
	innerWidth := paletteWidth - 4
	if innerWidth < 0 {
		innerWidth = 0
	}

	var lines []string

	// Header: Title + Divider
	title := types.OverlayTitleStyle.Render("Slash Commands")
	lines = append(lines, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, title))
	lines = append(lines, lipgloss.NewStyle().Foreground(types.MutedGray).Render(strings.Repeat("─", innerWidth)))

	// Show up to 5 commands
	maxVisible := 5
	if len(cp.filteredCommands) < maxVisible {
		maxVisible = len(cp.filteredCommands)
	}

	for i := 0; i < maxVisible; i++ {
		cmd := cp.filteredCommands[i]
		isSelected := i == cp.selectedIndex

		prefixStr := "  "
		if isSelected {
			prefixStr = "❯ "
		}

		// Style the prefix chevron in salmon pink when selected
		prefixStyle := lipgloss.NewStyle().Foreground(types.MutedGray)
		if isSelected {
			prefixStyle = lipgloss.NewStyle().Foreground(types.SalmonPink).Bold(true)
		}

		cmdNameStyle := lipgloss.NewStyle().
			Foreground(types.SalmonPink).
			Bold(isSelected)

		// Make description bold when selected instead of using a background highlight
		descStyle := lipgloss.NewStyle().
			Foreground(types.MutedGray).
			Bold(isSelected)

		renderedPrefix := prefixStyle.Render(prefixStr)
		renderedName := cmdNameStyle.Render("/" + cmd.Name)
		renderedDesc := descStyle.Render(cmd.Description)

		line := renderedPrefix + renderedName + "  " + renderedDesc
		lines = append(lines, line)
	}

	// Footer hint + Divider
	if len(cp.filteredCommands) > maxVisible {
		lines = append(lines, lipgloss.NewStyle().Foreground(types.MutedGray).Render(strings.Repeat("─", innerWidth)))
		
		footerHint := "... and more. Keep typing to filter."
		lines = append(lines, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, types.OverlayHelpStyle.Render(footerHint)))
	}

	// Wrap in container style
	containerStyle := types.CreateOverlayContainerStyle(paletteWidth)
	
	// Join lines to explicitly control newlines and prevent lipgloss padded blank lines
	content := strings.Join(lines, "\n")
	
	return containerStyle.Render(content)
}

// IsActive returns whether the palette is active
func (cp *CommandPalette) IsActive() bool {
	return cp.active
}
