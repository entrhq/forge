package overlay

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/executor/tui/types"
)

// SettingsOverlay provides a full interactive settings editor
type SettingsOverlay struct {
	width   int
	height  int
	focused bool

	// Navigation state
	selectedSection int
	selectedItem    int
	sections        []settingsSection

	// Edit state
	hasChanges bool
	editMode   bool

	// Scroll state
	scrollOffset int

	// Dialog state
	activeDialog  *inputDialog
	confirmDialog *confirmDialog
}

// settingsSection represents a section with its items
type settingsSection struct {
	id          string
	title       string
	description string
	items       []settingsItem
}

// settingsItem represents an editable item
type settingsItem struct {
	key         string
	displayName string
	value       interface{}
	itemType    itemType
	modified    bool
}

// itemType defines the type of setting item
type itemType int

const (
	itemTypeToggle itemType = iota
	itemTypeText
	itemTypeList
)

// inputDialog represents a modal dialog for text input
type inputDialog struct {
	title         string
	fields        []inputField
	selectedField int
	onConfirm     func(values map[string]string) error
	onCancel      func()
}

// inputField represents a single input field in a dialog
type inputField struct {
	label     string
	key       string
	value     string
	fieldType fieldType
	options   []string // For radio buttons
	maxLength int
	validator func(string) error
	errorMsg  string
}

// fieldType defines the type of input field
type fieldType int

const (
	fieldTypeText fieldType = iota
	fieldTypeRadio
)

// Section ID constants
const (
	sectionCommandWhitelist = "command_whitelist"
)

// confirmDialog represents a confirmation dialog
type confirmDialog struct {
	title   string
	message string
	details []string
	onYes   func()
	onNo    func()
}

// NewSettingsOverlay creates a new interactive settings overlay
func NewSettingsOverlay(width, height int) *SettingsOverlay {
	overlay := &SettingsOverlay{
		width:           width,
		height:          height,
		focused:         true,
		selectedSection: 0,
		selectedItem:    0,
		hasChanges:      false,
		editMode:        false,
		scrollOffset:    0,
	}

	overlay.loadSettings()
	return overlay
}

// loadSettings loads settings from config into editable sections
func (s *SettingsOverlay) loadSettings() {
	if !config.IsInitialized() {
		return
	}

	manager := config.Global()
	configSections := manager.GetSections()

	s.sections = make([]settingsSection, 0, len(configSections))

	for _, sec := range configSections {
		section := settingsSection{
			id:          sec.ID(),
			title:       sec.Title(),
			description: sec.Description(),
			items:       make([]settingsItem, 0),
		}

		data := sec.Data()

		switch sec.ID() {
		case "auto_approval":
			// Create toggle items for each tool
			keys := make([]string, 0, len(data))
			for k := range data {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, key := range keys {
				value := data[key]
				item := settingsItem{
					key:         key,
					displayName: key,
					value:       value,
					itemType:    itemTypeToggle,
					modified:    false,
				}
				section.items = append(section.items, item)
			}

		case sectionCommandWhitelist:
			// Show patterns as list items
			if patterns, ok := data["patterns"].([]interface{}); ok {
				for i, p := range patterns {
					if patternMap, ok := p.(map[string]interface{}); ok {
						pattern := patternMap["pattern"]
						desc := patternMap["description"]
						displayName := fmt.Sprintf("%v", pattern)
						if desc != nil && desc != "" {
							displayName = fmt.Sprintf("%v - %v", pattern, desc)
						}

						item := settingsItem{
							key:         fmt.Sprintf("pattern_%d", i),
							displayName: displayName,
							value:       patternMap,
							itemType:    itemTypeList,
							modified:    false,
						}
						section.items = append(section.items, item)
					}
				}
			}
		}

		s.sections = append(s.sections, section)
	}
}

// Update handles messages for the interactive settings overlay
func (s *SettingsOverlay) Update(msg tea.Msg, state types.StateProvider, actions types.ActionHandler) (types.Overlay, tea.Cmd) {
	// Handle active dialog input first
	if s.activeDialog != nil {
		return s.handleDialogInput(msg)
	}

	// Handle confirmation dialog
	if s.confirmDialog != nil {
		return s.handleConfirmInput(msg)
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}

	return s.handleKeyPress(keyMsg, actions)
}

// handleKeyPress processes keyboard input for the settings overlay
func (s *SettingsOverlay) handleKeyPress(keyMsg tea.KeyMsg, actions types.ActionHandler) (types.Overlay, tea.Cmd) {
	switch keyMsg.String() {
	case keyEsc, "q":
		return s.handleEscape(actions)
	case "ctrl+s":
		return s.handleSave()
	case "up", "k":
		s.navigateUp()
	case "down", "j":
		s.navigateDown()
	case "left", "h":
		s.navigateLeft()
	case "right", "l":
		s.navigateRight()
	case " ", keyEnter:
		s.toggleCurrent()
	case keyTab:
		s.nextSection()
	case "shift+tab":
		s.prevSection()
	case "a":
		s.handleAddPattern()
	case "e":
		s.handleEditPattern()
	case "d":
		s.handleDeletePattern()
	}
	return s, nil
}

// handleEscape handles the escape key press
func (s *SettingsOverlay) handleEscape(actions types.ActionHandler) (types.Overlay, tea.Cmd) {
	if s.hasChanges {
		s.showUnsavedChangesDialog(actions)
		return s, nil
	}
	if actions != nil {
		actions.ClearOverlay()
		return nil, nil
	}
	return nil, nil
}

// handleSave handles the save command
func (s *SettingsOverlay) handleSave() (types.Overlay, tea.Cmd) {
	if s.hasChanges {
		if err := s.saveSettings(); err == nil {
			s.hasChanges = false
		}
	}
	return s, nil
}

// handleAddPattern handles adding a new pattern
func (s *SettingsOverlay) handleAddPattern() {
	if s.isInWhitelistSection() {
		s.showAddPatternDialog()
	}
}

// handleEditPattern handles editing a selected pattern
func (s *SettingsOverlay) handleEditPattern() {
	if s.isInWhitelistSection() && s.isPatternSelected() {
		s.showEditPatternDialog()
	}
}

// handleDeletePattern handles deleting a selected pattern
func (s *SettingsOverlay) handleDeletePattern() {
	if s.isInWhitelistSection() && s.isPatternSelected() {
		s.showDeleteConfirmation()
	}
}

// navigateUp moves selection up
func (s *SettingsOverlay) navigateUp() {
	if len(s.sections) == 0 {
		return
	}

	if s.selectedItem > 0 {
		s.selectedItem--
	} else if s.selectedSection > 0 {
		s.selectedSection--
		if len(s.sections[s.selectedSection].items) > 0 {
			s.selectedItem = len(s.sections[s.selectedSection].items) - 1
		}
	}
}

// navigateDown moves selection down
func (s *SettingsOverlay) navigateDown() {
	if len(s.sections) == 0 {
		return
	}

	currentSection := s.sections[s.selectedSection]
	if s.selectedItem < len(currentSection.items)-1 {
		s.selectedItem++
	} else if s.selectedSection < len(s.sections)-1 {
		s.selectedSection++
		s.selectedItem = 0
	}
}

// navigateLeft moves to previous section
func (s *SettingsOverlay) navigateLeft() {
	s.prevSection()
}

// navigateRight moves to next section
func (s *SettingsOverlay) navigateRight() {
	s.nextSection()
}

// nextSection moves to the next section
func (s *SettingsOverlay) nextSection() {
	if s.selectedSection < len(s.sections)-1 {
		s.selectedSection++
		s.selectedItem = 0
	}
}

// prevSection moves to the previous section
func (s *SettingsOverlay) prevSection() {
	if s.selectedSection > 0 {
		s.selectedSection--
		s.selectedItem = 0
	}
}

// toggleCurrent toggles the current item
func (s *SettingsOverlay) toggleCurrent() {
	if len(s.sections) == 0 {
		return
	}

	section := &s.sections[s.selectedSection]
	if s.selectedItem >= len(section.items) {
		return
	}

	item := &section.items[s.selectedItem]
	if item.itemType == itemTypeToggle {
		if boolVal, ok := item.value.(bool); ok {
			item.value = !boolVal
			item.modified = true
			s.hasChanges = true
		}
	}
}

// saveSettings saves changes back to config
func (s *SettingsOverlay) saveSettings() error {
	if !config.IsInitialized() {
		return fmt.Errorf("config not initialized")
	}

	manager := config.Global()

	for _, section := range s.sections {
		configSection, exists := manager.GetSection(section.id)
		if !exists {
			continue
		}

		// Build updated data map
		data := make(map[string]interface{})

		switch section.id {
		case "auto_approval":
			for _, item := range section.items {
				data[item.key] = item.value
			}

		case sectionCommandWhitelist:
			// Reconstruct patterns array
			patterns := make([]interface{}, 0)
			for _, item := range section.items {
				if item.itemType == itemTypeList {
					patterns = append(patterns, item.value)
				}
			}
			data["patterns"] = patterns
		}

		// Update section
		if err := configSection.SetData(data); err != nil {
			return fmt.Errorf("failed to update section %s: %w", section.id, err)
		}
	}

	// Save all changes
	return manager.SaveAll()
}

// View renders the interactive settings overlay
func (s *SettingsOverlay) View() string {
	if !config.IsInitialized() {
		return s.renderError("Configuration not initialized")
	}

	// If dialog is active, render it on top
	if s.activeDialog != nil {
		return s.renderWithDialog()
	}

	// If confirmation dialog is active, render it on top
	if s.confirmDialog != nil {
		return s.renderWithConfirmation()
	}

	var content strings.Builder

	// Title
	title := types.OverlayTitleStyle.Render("Settings")
	content.WriteString(title)
	content.WriteString("\n\n")

	// Help text
	helpText := s.buildHelpText()
	content.WriteString(types.OverlaySubtitleStyle.Render(helpText))
	content.WriteString("\n\n")

	// Render sections
	for i, section := range s.sections {
		if i > 0 {
			content.WriteString("\n")
		}
		content.WriteString(s.renderSection(section, i == s.selectedSection))
	}

	// Status bar
	if s.hasChanges {
		content.WriteString("\n\n")
		saveHint := lipgloss.NewStyle().
			Foreground(types.SalmonPink).
			Bold(true).
			Render("● Unsaved changes - Press Ctrl+S to save")
		content.WriteString(saveHint)
	}

	// Create bordered box
	// CreateOverlayContainerStyle adds border (2) + padding (4) = 6 total width
	boxStyle := types.CreateOverlayContainerStyle(s.width - 6).Height(s.height - 4)

	return lipgloss.Place(
		s.width,
		s.height,
		lipgloss.Center,
		lipgloss.Center,
		boxStyle.Render(content.String()),
	)
}

// buildHelpText creates the help text based on current state
func (s *SettingsOverlay) buildHelpText() string {
	shortcuts := []string{
		"↑↓/jk: Navigate",
		"Tab/←→/hl: Switch section",
		"Space/Enter: Toggle",
	}

	// Add whitelist-specific shortcuts if in that section
	if s.isInWhitelistSection() {
		shortcuts = append(shortcuts, "a: Add", "e: Edit", "d: Delete")
	}

	shortcuts = append(shortcuts, "Ctrl+S: Save", "Esc/q: Close")
	return strings.Join(shortcuts, " • ")
}

// renderSection renders a settings section
func (s *SettingsOverlay) renderSection(section settingsSection, isSelected bool) string {
	var out strings.Builder

	// Section title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(types.MintGreen)

	if isSelected {
		titleStyle = titleStyle.Foreground(types.SalmonPink)
	}

	out.WriteString(titleStyle.Render("▸ " + section.title))
	out.WriteString("\n")

	// Section description
	if section.description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(types.MutedGray).
			Italic(true)
		out.WriteString("  ")
		out.WriteString(descStyle.Render(section.description))
		out.WriteString("\n")
	}

	// Render items
	for i, item := range section.items {
		isItemFocused := isSelected && i == s.selectedItem
		out.WriteString(s.renderItem(item, isItemFocused))
	}

	return out.String()
}

// renderItem renders a single setting item
func (s *SettingsOverlay) renderItem(item settingsItem, isFocused bool) string {
	var out strings.Builder

	prefix := "  "
	if isFocused {
		prefix = "➜ "
	}

	labelStyle := lipgloss.NewStyle().
		Foreground(types.MutedGray)

	if isFocused {
		labelStyle = labelStyle.Foreground(types.BrightWhite).Bold(true)
	} else if item.itemType == itemTypeToggle {
		if enabled, ok := item.value.(bool); ok && enabled {
			// Enabled toggles are brighter even when not focused
			labelStyle = labelStyle.Foreground(types.BrightWhite)
		}
	}

	out.WriteString(prefix)

	// Type-specific rendering
	switch item.itemType {
	case itemTypeToggle:
		enabled, ok := item.value.(bool)
		if !ok {
			enabled = false
		}
		check := "[ ]"
		if enabled {
			check = "[x]"
			check = lipgloss.NewStyle().Foreground(types.MintGreen).Render(check)
		}
		out.WriteString(fmt.Sprintf("%s %s", check, labelStyle.Render(item.displayName)))
	case itemTypeList:
		out.WriteString(fmt.Sprintf("• %s", labelStyle.Render(item.displayName)))
	case itemTypeText:
		out.WriteString(fmt.Sprintf("%s: %v", labelStyle.Render(item.displayName), item.value))
	}

	if item.modified {
		out.WriteString(lipgloss.NewStyle().Foreground(types.SalmonPink).Render(" *"))
	}

	out.WriteString("\n")
	return out.String()
}

// Helper methods for dialogs and state management

func (s *SettingsOverlay) isInWhitelistSection() bool {
	if s.selectedSection >= len(s.sections) {
		return false
	}
	return s.sections[s.selectedSection].id == sectionCommandWhitelist
}

func (s *SettingsOverlay) isPatternSelected() bool {
	if !s.isInWhitelistSection() {
		return false
	}
	section := s.sections[s.selectedSection]
	return len(section.items) > 0 && s.selectedItem < len(section.items)
}

func (s *SettingsOverlay) showAddPatternDialog() {
	s.activeDialog = &inputDialog{
		title: "Add Whitelist Pattern",
		fields: []inputField{
			{
				label:     "Pattern",
				key:       "pattern",
				fieldType: fieldTypeText,
				validator: func(v string) error {
					if strings.TrimSpace(v) == "" {
						return fmt.Errorf("pattern cannot be empty")
					}
					return nil
				},
			},
			{
				label:     "Description",
				key:       "description",
				fieldType: fieldTypeText,
			},
			{
				label:     "Match Type",
				key:       "type",
				fieldType: fieldTypeRadio,
				value:     "prefix",
				options:   []string{"prefix", "exact"},
			},
		},
		selectedField: 0,
		onConfirm: func(values map[string]string) error {
			s.addPattern(values["pattern"], values["description"], values["type"])
			return nil
		},
		onCancel: func() {
			s.activeDialog = nil
		},
	}
}

func (s *SettingsOverlay) showEditPatternDialog() {
	section := s.sections[s.selectedSection]
	item := section.items[s.selectedItem]
	data, ok := item.value.(map[string]interface{})
	if !ok {
		return
	}

	// Get the type value, default to "prefix" if not present
	typeValue := "prefix"
	if t, ok := data["type"].(string); ok {
		typeValue = t
	}

	s.activeDialog = &inputDialog{
		title: "Edit Whitelist Pattern",
		fields: []inputField{
			{
				label:     "Pattern",
				key:       "pattern",
				value:     fmt.Sprintf("%v", data["pattern"]),
				fieldType: fieldTypeText,
				validator: func(v string) error {
					if strings.TrimSpace(v) == "" {
						return fmt.Errorf("pattern cannot be empty")
					}
					return nil
				},
			},
			{
				label:     "Description",
				key:       "description",
				value:     fmt.Sprintf("%v", data["description"]),
				fieldType: fieldTypeText,
			},
			{
				label:     "Match Type",
				key:       "type",
				fieldType: fieldTypeRadio,
				value:     typeValue,
				options:   []string{"prefix", "exact"},
			},
		},
		selectedField: 0,
		onConfirm: func(values map[string]string) error {
			s.updatePattern(s.selectedItem, values["pattern"], values["description"], values["type"])
			return nil
		},
		onCancel: func() {
			s.activeDialog = nil
		},
	}
}

func (s *SettingsOverlay) showDeleteConfirmation() {
	s.confirmDialog = &confirmDialog{
		title:   "Delete Pattern",
		message: "Are you sure you want to delete this pattern?",
		onYes: func() {
			s.deletePattern(s.selectedItem)
			s.confirmDialog = nil
		},
		onNo: func() {
			s.confirmDialog = nil
		},
	}
}

func (s *SettingsOverlay) showUnsavedChangesDialog(actions types.ActionHandler) {
	s.confirmDialog = &confirmDialog{
		title:   "Unsaved Changes",
		message: "You have unsaved changes. Save before closing?",
		onYes: func() {
			// Ignore error from saveSettings - we're closing anyway
			_ = s.saveSettings() //nolint:errcheck
			s.hasChanges = false
			s.confirmDialog = nil
			if actions != nil {
				actions.ClearOverlay()
			}
		},
		onNo: func() {
			s.hasChanges = false
			s.confirmDialog = nil
			if actions != nil {
				actions.ClearOverlay()
			}
		},
	}
}

func (s *SettingsOverlay) addPattern(pattern, description, matchType string) {
	// Implementation of adding pattern to the local state
	// This updates s.sections directly
	for i, section := range s.sections {
		if section.id == sectionCommandWhitelist {
			newItem := settingsItem{
				key:         fmt.Sprintf("pattern_new_%d", len(section.items)),
				displayName: fmt.Sprintf("%s - %s", pattern, description),
				value: map[string]interface{}{
					"pattern":     pattern,
					"description": description,
					"type":        matchType,
				},
				itemType: itemTypeList,
				modified: true,
			}
			s.sections[i].items = append(s.sections[i].items, newItem)
			s.hasChanges = true
			break
		}
	}
}

func (s *SettingsOverlay) updatePattern(index int, pattern, description, matchType string) {
	for i, section := range s.sections {
		if section.id == sectionCommandWhitelist {
			if index < len(section.items) {
				s.sections[i].items[index].value = map[string]interface{}{
					"pattern":     pattern,
					"description": description,
					"type":        matchType,
				}
				s.sections[i].items[index].displayName = fmt.Sprintf("%s - %s", pattern, description)
				s.sections[i].items[index].modified = true
				s.hasChanges = true
			}
			break
		}
	}
}

func (s *SettingsOverlay) deletePattern(index int) {
	for i, section := range s.sections {
		if section.id == sectionCommandWhitelist {
			if index < len(section.items) {
				// Remove item at index
				section.items = append(section.items[:index], section.items[index+1:]...)
				s.sections[i].items = section.items
				s.hasChanges = true

				// Adjust selection if needed
				if s.selectedItem >= len(section.items) && s.selectedItem > 0 {
					s.selectedItem--
				}
			}
			break
		}
	}
}

// handleDialogInput handles keyboard input for the active dialog
func (s *SettingsOverlay) handleDialogInput(msg tea.Msg) (types.Overlay, tea.Cmd) {
	if s.activeDialog == nil {
		return s, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return s, nil
	}

	switch keyMsg.String() {
	case keyEsc:
		return s.handleDialogCancel()
	case keyEnter:
		return s.handleDialogConfirm()
	case "tab", "down":
		s.moveToNextDialogField()
	case "shift+tab", "up":
		s.moveToPrevDialogField()
	case " ":
		s.handleDialogSpace()
	case "backspace":
		s.handleDialogBackspace()
	default:
		s.handleDialogCharInput(keyMsg)
	}

	return s, nil
}

// handleDialogCancel handles canceling the dialog
func (s *SettingsOverlay) handleDialogCancel() (types.Overlay, tea.Cmd) {
	if s.activeDialog.onCancel != nil {
		s.activeDialog.onCancel()
	}
	return s, nil
}

// handleDialogConfirm validates and confirms the dialog input
func (s *SettingsOverlay) handleDialogConfirm() (types.Overlay, tea.Cmd) {
	values := make(map[string]string)

	// Validate all fields
	for i := range s.activeDialog.fields {
		field := &s.activeDialog.fields[i]
		values[field.key] = field.value

		if field.validator != nil {
			if err := field.validator(field.value); err != nil {
				field.errorMsg = err.Error()
				return s, nil
			}
		}
	}

	// Call onConfirm callback
	if s.activeDialog.onConfirm != nil {
		if err := s.activeDialog.onConfirm(values); err != nil {
			return s, nil
		}
	}

	// Clear the dialog after successful confirmation
	s.activeDialog = nil
	return s, nil
}

// moveToNextDialogField moves to the next field in the dialog
func (s *SettingsOverlay) moveToNextDialogField() {
	s.activeDialog.selectedField++
	if s.activeDialog.selectedField >= len(s.activeDialog.fields) {
		s.activeDialog.selectedField = 0
	}
}

// moveToPrevDialogField moves to the previous field in the dialog
func (s *SettingsOverlay) moveToPrevDialogField() {
	s.activeDialog.selectedField--
	if s.activeDialog.selectedField < 0 {
		s.activeDialog.selectedField = len(s.activeDialog.fields) - 1
	}
}

// handleDialogSpace handles space key in dialog (toggle radio or add space to text)
func (s *SettingsOverlay) handleDialogSpace() {
	field := &s.activeDialog.fields[s.activeDialog.selectedField]

	if field.fieldType == fieldTypeRadio && len(field.options) > 0 {
		s.toggleRadioButton(field)
	} else if field.fieldType == fieldTypeText {
		s.addSpaceToTextField(field)
	}
}

// toggleRadioButton toggles the radio button to next option
func (s *SettingsOverlay) toggleRadioButton(field *inputField) {
	currentIdx := 0
	for i, opt := range field.options {
		if opt == field.value {
			currentIdx = i
			break
		}
	}
	nextIdx := (currentIdx + 1) % len(field.options)
	field.value = field.options[nextIdx]
}

// addSpaceToTextField adds a space character to the text field
func (s *SettingsOverlay) addSpaceToTextField(field *inputField) {
	if field.maxLength == 0 || len(field.value) < field.maxLength {
		field.value += " "
		field.errorMsg = ""
	}
}

// handleDialogBackspace handles backspace key in dialog
func (s *SettingsOverlay) handleDialogBackspace() {
	field := &s.activeDialog.fields[s.activeDialog.selectedField]
	if field.fieldType == fieldTypeText && len(field.value) > 0 {
		field.value = field.value[:len(field.value)-1]
		field.errorMsg = ""
	}
}

// handleDialogCharInput handles character input in dialog
func (s *SettingsOverlay) handleDialogCharInput(keyMsg tea.KeyMsg) {
	field := &s.activeDialog.fields[s.activeDialog.selectedField]
	if field.fieldType == fieldTypeText {
		if len(keyMsg.String()) == 1 && (field.maxLength == 0 || len(field.value) < field.maxLength) {
			field.value += keyMsg.String()
			field.errorMsg = ""
		}
	}
}

// handleConfirmInput handles keyboard input for confirmation dialogs
func (s *SettingsOverlay) handleConfirmInput(msg tea.Msg) (types.Overlay, tea.Cmd) {
	if s.confirmDialog == nil {
		return s, nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "y", "Y":
			if s.confirmDialog.onYes != nil {
				s.confirmDialog.onYes()
			}
			// Check if we should close the overlay after saving
			shouldClose := s.confirmDialog == nil && !s.hasChanges
			if shouldClose {
				return nil, nil
			}
			return s, nil

		case "n", "N":
			if s.confirmDialog.onNo != nil {
				s.confirmDialog.onNo()
			}
			// Check if we should close the overlay after discarding
			shouldClose := s.confirmDialog == nil && !s.hasChanges
			if shouldClose {
				return nil, nil
			}
			return s, nil

		case keyEsc:
			// Cancel - just close dialog and return to editing
			s.confirmDialog = nil
			return s, nil
		}
	}

	return s, nil
}

// renderWithDialog renders the settings view with an input dialog overlay
func (s *SettingsOverlay) renderWithDialog() string {
	// Render dialog on top
	dialogView := s.renderInputDialog()

	// Layer dialog over base view
	return s.layerDialogOver(dialogView)
}

// renderWithConfirmation renders the settings view with a confirmation dialog overlay
func (s *SettingsOverlay) renderWithConfirmation() string {
	// Render confirmation dialog on top
	dialogView := s.renderConfirmDialog()

	// Layer dialog over base view
	return s.layerDialogOver(dialogView)
}

// renderInputDialog renders an input dialog
func (s *SettingsOverlay) renderInputDialog() string {
	if s.activeDialog == nil {
		return ""
	}

	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(types.SalmonPink).
		Bold(true)
	content.WriteString(titleStyle.Render(s.activeDialog.title))
	content.WriteString("\n\n")

	// Fields
	for i, field := range s.activeDialog.fields {
		isSelected := i == s.activeDialog.selectedField

		// Label
		labelStyle := lipgloss.NewStyle().Foreground(types.BrightWhite)
		content.WriteString(labelStyle.Render(field.label))
		content.WriteString("\n")

		// Field content based on type
		switch field.fieldType {
		case fieldTypeText:
			// Text input field
			fieldStyle := lipgloss.NewStyle().
				Foreground(types.BrightWhite).
				Background(types.DarkBg).
				Padding(0, 1)

			if isSelected {
				fieldStyle = fieldStyle.Border(lipgloss.RoundedBorder()).
					BorderForeground(types.SalmonPink)
			}

			value := field.value
			if isSelected {
				value += "▸" // Cursor
			}

			content.WriteString(fieldStyle.Render(value))
			content.WriteString("\n")

			// Character count or error
			if field.errorMsg != "" {
				errMsgStyle := lipgloss.NewStyle().Foreground(types.SalmonPink)
				content.WriteString(errMsgStyle.Render(field.errorMsg))
				content.WriteString("\n")
			} else if field.maxLength > 0 {
				countStyle := lipgloss.NewStyle().Foreground(types.MutedGray)
				count := fmt.Sprintf("[%d/%d]", len(field.value), field.maxLength)
				content.WriteString(countStyle.Render(count))
				content.WriteString("\n")
			}

		case fieldTypeRadio:
			// Radio buttons
			for j, option := range field.options {
				radioStyle := lipgloss.NewStyle().Foreground(types.BrightWhite)
				if isSelected {
					radioStyle = radioStyle.Bold(true)
				}

				bullet := "○"
				if option == field.value {
					bullet = "●"
					radioStyle = radioStyle.Foreground(types.MintGreen)
				}

				optionText := fmt.Sprintf("%s %s", bullet, option)
				if j > 0 {
					content.WriteString("    ") // Indent
				}
				content.WriteString(radioStyle.Render(optionText))
				if j < len(field.options)-1 {
					content.WriteString("    ")
				}
			}
			content.WriteString("\n")
		}

		content.WriteString("\n")
	}

	// Buttons
	buttonRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		"[Enter to Add] [Esc to Cancel]",
	)
	buttonStyle := lipgloss.NewStyle().Foreground(types.MutedGray)
	content.WriteString(buttonStyle.Render(buttonRow))

	// Create dialog box
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(types.SalmonPink).
		Background(types.DarkBg).
		Padding(1, 2).
		Width(60)

	return dialogStyle.Render(content.String())
}

// renderConfirmDialog renders a confirmation dialog
func (s *SettingsOverlay) renderConfirmDialog() string {
	if s.confirmDialog == nil {
		return ""
	}

	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(types.SalmonPink).
		Bold(true)
	content.WriteString(titleStyle.Render(s.confirmDialog.title))
	content.WriteString("\n\n")

	// Message
	messageStyle := lipgloss.NewStyle().Foreground(types.BrightWhite)
	content.WriteString(messageStyle.Render(s.confirmDialog.message))
	content.WriteString("\n\n")

	// Details
	detailStyle := lipgloss.NewStyle().Foreground(types.MutedGray)
	for _, detail := range s.confirmDialog.details {
		content.WriteString(detailStyle.Render(detail))
		content.WriteString("\n")
	}

	content.WriteString("\n")

	// Buttons
	buttonRow := "[y] Yes, delete    [n] No, cancel"
	buttonStyle := lipgloss.NewStyle().Foreground(types.MutedGray)
	content.WriteString(buttonStyle.Render(buttonRow))

	// Create dialog box
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(types.SalmonPink).
		Background(types.DarkBg).
		Padding(1, 2).
		Width(60)

	return dialogStyle.Render(content.String())
}

// layerDialogOver layers a dialog over the base view
func (s *SettingsOverlay) layerDialogOver(dialogView string) string {
	// Place dialog in center
	return lipgloss.Place(
		s.width,
		s.height,
		lipgloss.Center,
		lipgloss.Center,
		dialogView,
		lipgloss.WithWhitespaceChars(""),
		lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
	)
}

// renderError renders an error message
func (s *SettingsOverlay) renderError(message string) string {
	errMsgStyle := lipgloss.NewStyle().
		Foreground(types.SalmonPink).
		Bold(true)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(types.SalmonPink).
		Background(types.DarkBg).
		Padding(1, 2).
		Width(s.width - 4).
		Height(s.height - 4)

	content := errMsgStyle.Render("Error: ") + message

	return lipgloss.Place(
		s.width,
		s.height,
		lipgloss.Center,
		lipgloss.Center,
		boxStyle.Render(content),
	)
}

// Focused returns whether this overlay should handle input
func (s *SettingsOverlay) Focused() bool {
	return s.focused
}

// Width returns the overlay width
func (s *SettingsOverlay) Width() int {
	return s.width
}

// Height returns the overlay height
func (s *SettingsOverlay) Height() int {
	return s.height
}
