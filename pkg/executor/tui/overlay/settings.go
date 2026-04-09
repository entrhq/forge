package overlay

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/executor/tui/types"
	"github.com/entrhq/forge/pkg/llm"
)

const (
	// Section and field names
	llmSection                = "llm"
	modelField                = "model"
	summarizationModelField   = "summarization_model"
	browserAnalysisModelField = "browser_analysis_model"
	baseURLField              = "base_url"
	apiKeyField               = "api_key"
)

// SettingsOverlay provides a full interactive settings editor
type SettingsOverlay struct {
	termWidth  int
	termHeight int
	width      int
	height     int
	focused    bool

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

	// Runtime provider for displaying actual values
	provider llm.Provider

	// Callback for when LLM settings change
	onLLMSettingsChange func() error

	// Callback for when UI settings change (e.g. show_thinking toggled via overlay)
	onUISettingsChange func() error

	// Cursor blink state
	cursorBlink bool
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
	value       any
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
	fieldTypePassword
	fieldTypeRadio
)

// Section ID constants
const (
	sectionCommandWhitelist = "command_whitelist"
)

// confirmDialog represents a confirmation dialog
type confirmDialog struct {
	title       string
	message     string
	details     []string
	yesLabel    string
	noLabel     string
	cancelLabel string
	onYes       func()
	onNo        func()
}

// NewSettingsOverlay creates a new interactive settings overlay
func NewSettingsOverlay(width, height int) *SettingsOverlay {
	return NewSettingsOverlayWithCallback(width, height, nil, nil)
}

// NewSettingsOverlayWithCallback creates a new settings overlay with an optional callback
// that is invoked when LLM settings are saved.
func NewSettingsOverlayWithCallback(width, height int, onLLMSettingsChange func() error, provider llm.Provider) *SettingsOverlay {
	overlayWidth := types.ComputeOverlayWidth(width, 0.90, 60, 140)
	overlayHeight := types.ComputeViewportHeight(height, 4)

	overlay := &SettingsOverlay{
		termWidth:           width,
		termHeight:          height,
		width:               overlayWidth,
		height:              overlayHeight,
		focused:             true,
		selectedSection:     0,
		selectedItem:        0,
		hasChanges:          false,
		provider:            provider,
		onLLMSettingsChange: onLLMSettingsChange,
		editMode:            false,
		scrollOffset:        0,
		cursorBlink:         true,
	}

	overlay.loadSettings()
	return overlay
}

// SetOnUISettingsChange registers a callback invoked after UI settings are saved.
// Use this to sync runtime state (e.g. showThinking on the TUI model) after
// the user saves changes in the settings overlay.
func (s *SettingsOverlay) SetOnUISettingsChange(fn func() error) {
	s.onUISettingsChange = fn
}

// loadSettings loads settings from config into editable sections
//
//nolint:gocyclo // UI settings loading has inherent complexity
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
			if patterns, ok := data["patterns"].([]any); ok {
				for i, p := range patterns {
					if patternMap, ok := p.(map[string]any); ok {
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

		case llmSection:
			// Create text items for LLM configuration
			// Order matters for a better UX
			llmFields := []struct {
				key         string
				displayName string
			}{
				{modelField, "Model"},
				{summarizationModelField, "Summarization Model"},
				{browserAnalysisModelField, "Browser Analysis Model"},
				{baseURLField, "Base URL"},
				{apiKeyField, "API Key"},
			}

			for _, field := range llmFields {
				value := ""

				// Try to get value from runtime provider first (actual running config)
				if s.provider != nil {
					switch field.key {
					case modelField:
						if model := s.provider.GetModel(); model != "" {
							value = model
						}
					case baseURLField:
						if baseURL := s.provider.GetBaseURL(); baseURL != "" {
							value = baseURL
						}
					case apiKeyField:
						if apiKey := s.provider.GetAPIKey(); apiKey != "" {
							// Store actual API key - renderItem will handle masking for display
							value = apiKey
						}
					}
				}

				// Fall back to config file value if provider didn't have it
				if value == "" {
					if v, ok := data[field.key]; ok && v != nil {
						value = fmt.Sprintf("%v", v)
					}
				}

				// For summarization_model: default to the main model if not explicitly set.
				if value == "" && field.key == summarizationModelField && s.provider != nil {
					value = s.provider.GetModel()
				}

				// For browser_analysis_model: default to the main model if not explicitly set.
				if value == "" && field.key == browserAnalysisModelField && s.provider != nil {
					value = s.provider.GetModel()
				}

				item := settingsItem{
					key:         field.key,
					displayName: field.displayName,
					value:       value,
					itemType:    itemTypeText,
					modified:    false,
				}
				section.items = append(section.items, item)
			}

		case "ui":
			// Create toggle and text items for UI configuration
			uiFields := []struct {
				key         string
				displayName string
				itemType    itemType
			}{
				{"show_thinking", "Show Thinking Blocks", itemTypeToggle},
				{"auto_close_command_overlay", "Auto-close Command Overlay", itemTypeToggle},
				{"keep_open_on_error", "Keep Open On Error", itemTypeToggle},
				{"auto_close_delay", "Auto-close Delay", itemTypeText},
				{"browser_enabled", "Browser Automation Enabled", itemTypeToggle},
				{"browser_headless", "Browser Headless Mode", itemTypeToggle},
			}

			for _, field := range uiFields {
				value := data[field.key]
				item := settingsItem{
					key:         field.key,
					displayName: field.displayName,
					value:       value,
					itemType:    field.itemType,
					modified:    false,
				}
				section.items = append(section.items, item)
			}

		case "memory":
			// Create toggle and text items for memory configuration
			memoryFields := []struct {
				key         string
				displayName string
				itemType    itemType
			}{
				{"enabled", "Enabled", itemTypeToggle},
				{"classifier_model", "Classifier Model", itemTypeText},
				{"hypothesis_model", "Hypothesis Model", itemTypeText},
				{"embedding_model", "Embedding Model", itemTypeText},
				{"embedding_base_url", "Embedding Base URL", itemTypeText},
				{"embedding_api_key", "Embedding API Key", itemTypeText},
				{"retrieval_top_k", "Retrieval Top-K", itemTypeText},
				{"retrieval_hop_depth", "Retrieval Hop Depth", itemTypeText},
				{"retrieval_hypothesis_count", "Retrieval Hypothesis Count", itemTypeText},
				{"injection_token_budget", "Injection Token Budget", itemTypeText},
			}

			for _, field := range memoryFields {
				value := data[field.key]
				// Render numeric values as strings for text fields
				if field.itemType == itemTypeText {
					value = fmt.Sprintf("%v", value)
				}
				item := settingsItem{
					key:         field.key,
					displayName: field.displayName,
					value:       value,
					itemType:    field.itemType,
					modified:    false,
				}
				section.items = append(section.items, item)
			}

		case "multimodal":
			// Create text items for multimodal configuration
			multimodalFields := []struct {
				key         string
				displayName string
				itemType    itemType
			}{
				{"model", "Model", itemTypeText},
				{"pdf_page_limit", "PDF Page Limit", itemTypeText},
			}

			for _, field := range multimodalFields {
				value := data[field.key]
				// Render numeric values as strings for text fields
				if field.itemType == itemTypeText {
					value = fmt.Sprintf("%v", value)
				}
				item := settingsItem{
					key:         field.key,
					displayName: field.displayName,
					value:       value,
					itemType:    field.itemType,
					modified:    false,
				}
				section.items = append(section.items, item)
			}
		}

		s.sections = append(s.sections, section)
	}
}

// cursorBlinkMsg is sent periodically to toggle cursor visibility
type cursorBlinkMsg struct{}

// tickCursorBlink returns a command that sends a cursor blink message
func tickCursorBlink() tea.Cmd {
	return tea.Tick(time.Millisecond*530, func(t time.Time) tea.Msg {
		return cursorBlinkMsg{}
	})
}

// Update handles messages for the interactive settings overlay
func (s *SettingsOverlay) Update(msg tea.Msg, state types.StateProvider, actions types.ActionHandler) (types.Overlay, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.termWidth = msg.Width
		s.termHeight = msg.Height
		s.width = types.ComputeOverlayWidth(msg.Width, 0.90, 60, 140)
		s.height = types.ComputeViewportHeight(msg.Height, 4)
		// Return without handling further
		return s, nil

	case cursorBlinkMsg:
		s.cursorBlink = !s.cursorBlink
		return s, tickCursorBlink()

	case tea.KeyMsg:
		// Handle active dialog input first
		if s.activeDialog != nil {
			return s.handleDialogInput(msg)
		}

		// Handle confirmation dialog
		if s.confirmDialog != nil {
			return s.handleConfirmInput(msg)
		}

		return s.handleKeyPress(msg)
	}
	return s, nil
}

// handleNavigationKey handles directional/scroll keys; returns true if consumed.
func (s *SettingsOverlay) handleNavigationKey(key string) bool {
	switch key {
	case "up", "k":
		s.navigateUp()
	case "down", "j":
		s.navigateDown()
	case "left", "h":
		s.navigateLeft()
	case "right", "l":
		s.navigateRight()
	case "pgup":
		s.scrollOffset -= s.getVisibleBodyHeight()
		if s.scrollOffset < 0 {
			s.scrollOffset = 0
		}
	case "pgdown":
		// Clip is handled by the View renderer; just advance offset.
		s.scrollOffset += s.getVisibleBodyHeight()
	case keyTab:
		s.nextSection()
	case "shift+tab":
		s.prevSection()
	default:
		return false
	}
	return true
}

// handleKeyPress processes keyboard input for the settings overlay
func (s *SettingsOverlay) handleKeyPress(keyMsg tea.KeyMsg) (types.Overlay, tea.Cmd) {
	switch keyMsg.String() {
	case keyEsc, "q":
		return s.handleEscape()
	case "ctrl+s":
		return s.handleSave()
	case " ":
		s.toggleCurrent()
	case keyEnter:
		return s, s.handleEnter()
	case "a":
		return s, s.handleAddPattern()
	case "e":
		return s, s.handleEditPattern()
	case "d":
		s.handleDeletePattern()
	default:
		s.handleNavigationKey(keyMsg.String())
	}
	return s, nil
}

// handleEscape handles the escape key press
func (s *SettingsOverlay) handleEscape() (types.Overlay, tea.Cmd) {
	if s.hasChanges {
		s.showUnsavedChangesDialog()
		return s, nil
	}
	// Return nil to signal close - caller will handle ClearOverlay()
	return nil, nil
}

// handleSave handles the save command
func (s *SettingsOverlay) handleSave() (types.Overlay, tea.Cmd) {
	if s.hasChanges {
		if err := s.saveSettings(); err == nil {
			s.hasChanges = false
			// Don't reload settings here - provider will be stale until next overlay open
			// The onLLMSettingsChange callback reloads the provider in the agent,
			// but this overlay instance still has the old provider reference
		}
	}
	return s, nil
}

// handleAddPattern handles adding a new pattern
func (s *SettingsOverlay) handleAddPattern() tea.Cmd {
	if s.isInWhitelistSection() {
		return s.showAddPatternDialog()
	}
	return nil
}

// handleEditPattern handles editing a selected pattern
func (s *SettingsOverlay) handleEditPattern() tea.Cmd {
	if s.isInWhitelistSection() && s.isPatternSelected() {
		return s.showEditPatternDialog()
	}
	return nil
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

	// Adjust scroll if necessary
	itemLine := s.calculateItemSelectedLine()
	if itemLine < s.scrollOffset {
		s.scrollOffset = itemLine
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

	// Adjust scroll if necessary
	itemLine := s.calculateItemSelectedLine()
	maxHeight := s.getVisibleBodyHeight()
	if itemLine >= s.scrollOffset+maxHeight {
		s.scrollOffset = itemLine - maxHeight + 1
	}
}

// getVisibleBodyHeight estimates the visible body height
func (s *SettingsOverlay) getVisibleBodyHeight() int {
	// border space = 2
	// header = ~3 lines (title, separator, blank space)
	// footer = ~2 lines (separator, help) + 1 if hasChanges
	h := s.height - 2 - 3 - 2
	if s.hasChanges {
		h--
	}
	if h < 1 {
		return 1
	}
	return h
}

// calculateItemSelectedLine estimates the line number of the selected item
func (s *SettingsOverlay) calculateItemSelectedLine() int {
	line := 0
	for i := 0; i < s.selectedSection; i++ {
		line += 1 // Title
		line += 1 // Blank line after title
		if s.sections[i].description != "" {
			line++
		}
		line += len(s.sections[i].items)
		line++ // Blank line between sections
	}
	line += 1 // Title of selected section
	line += 1 // Blank line after title
	if s.sections[s.selectedSection].description != "" {
		line++
	}
	line += s.selectedItem
	return line
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
		s.scrollOffset = 0 // Reset scroll when changing section
	}
}

// prevSection moves to the previous section
func (s *SettingsOverlay) prevSection() {
	if s.selectedSection > 0 {
		s.selectedSection--
		s.selectedItem = 0
		s.scrollOffset = 0 // Reset scroll when changing section
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

// handleEnter handles the enter key press for editing text items
func (s *SettingsOverlay) handleEnter() tea.Cmd {
	if len(s.sections) == 0 {
		return nil
	}

	section := &s.sections[s.selectedSection]
	if s.selectedItem >= len(section.items) {
		return nil
	}

	item := &section.items[s.selectedItem]

	// Handle different item types
	var cmd tea.Cmd
	switch item.itemType {
	case itemTypeToggle:
		// Space key already handles toggles, but enter should also work
		s.toggleCurrent()

	case itemTypeText:
		// Show edit dialog for text items (LLM settings)
		cmd = s.showTextFieldEditDialog()

	case itemTypeList:
		// For list items (like whitelist patterns), show edit dialog
		if section.id == sectionCommandWhitelist {
			cmd = s.handleEditPattern()
		}
	}
	return cmd
}

// saveSettings saves changes back to config
//
//nolint:gocyclo // UI settings saving has inherent complexity
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
		data := make(map[string]any)

		switch section.id {
		case "auto_approval":
			for _, item := range section.items {
				data[item.key] = item.value
			}

		case sectionCommandWhitelist:
			// Reconstruct patterns array
			patterns := make([]any, 0)
			for _, item := range section.items {
				if item.itemType == itemTypeList {
					patterns = append(patterns, item.value)
				}
			}
			data["patterns"] = patterns

		case llmSection:
			// Save text field values for LLM settings
			for _, item := range section.items {
				if item.itemType == itemTypeText {
					data[item.key] = item.value
				}
			}

		case "ui":
			// Save UI settings (toggles and text fields)
			for _, item := range section.items {
				data[item.key] = item.value
			}

		case "memory":
			// Save memory settings - numeric fields must be stored as int so
			// SetData's intFromAny helper can parse them correctly.
			numericMemoryKeys := map[string]bool{
				"retrieval_top_k":            true,
				"retrieval_hop_depth":        true,
				"retrieval_hypothesis_count": true,
				"injection_token_budget":     true,
			}
			for _, item := range section.items {
				if numericMemoryKeys[item.key] {
					if s, ok := item.value.(string); ok {
						if n, err := strconv.Atoi(s); err == nil {
							data[item.key] = n
							continue
						}
					}
				}
				data[item.key] = item.value
			}

		case "multimodal":
			// Save multimodal settings - pdf_page_limit must be stored as int
			for _, item := range section.items {
				if item.key == "pdf_page_limit" {
					if s, ok := item.value.(string); ok {
						if n, err := strconv.Atoi(s); err == nil {
							data[item.key] = n
							continue
						}
						// Invalid integer - skip this setting to avoid storing bad value
						continue
					}
				}
				data[item.key] = item.value
			}
		}

		// Update section
		if err := configSection.SetData(data); err != nil {
			return fmt.Errorf("failed to update section %s: %w", section.id, err)
		}
	}

	// Save all changes
	if err := manager.SaveAll(); err != nil {
		return err
	}

	// If LLM settings were changed and we have a callback, invoke it
	if s.onLLMSettingsChange != nil {
		for _, section := range s.sections {
			if section.id == "llm" {
				if err := s.onLLMSettingsChange(); err != nil {
					return fmt.Errorf("failed to reload LLM settings: %w", err)
				}
				break
			}
		}
	}

	// If UI settings were changed and we have a callback, invoke it
	if s.onUISettingsChange != nil {
		for _, section := range s.sections {
			if section.id == "ui" {
				if err := s.onUISettingsChange(); err != nil {
					return fmt.Errorf("failed to sync UI settings: %w", err)
				}
				break
			}
		}
	}

	return nil
}

// renderViewHeader renders the title bar and top separator.
func (s *SettingsOverlay) renderViewHeader(innerWidth int) string {
	padLeft := (innerWidth - lipgloss.Width("Settings")) / 2
	if (innerWidth-lipgloss.Width("Settings"))%2 != 0 {
		padLeft++
	}
	title := strings.Repeat(" ", padLeft) + types.OverlayTitleStyle.Render("Settings")
	sep := lipgloss.NewStyle().Foreground(types.MutedGray).Render(strings.Repeat(sepChar, innerWidth))
	return title + "\n" + sep + "\n\n"
}

// renderViewFooter renders the status bar, bottom separator and help text.
func (s *SettingsOverlay) renderViewFooter(innerWidth int) string {
	var b strings.Builder
	if s.hasChanges {
		b.WriteString(lipgloss.NewStyle().Foreground(types.SalmonPink).Bold(true).
			Render("● Unsaved changes - Press Ctrl+S to save"))
		b.WriteString("\n")
	}
	b.WriteString(lipgloss.NewStyle().Foreground(types.MutedGray).Render(strings.Repeat(sepChar, innerWidth)))
	b.WriteString("\n")

	helpText := s.buildHelpText()
	padLeftHelp := (innerWidth - lipgloss.Width(helpText)) / 2
	if (innerWidth-lipgloss.Width(helpText))%2 != 0 {
		padLeftHelp++
	}
	if padLeftHelp < 0 {
		padLeftHelp = 0
	}
	displayHelp := helpText
	if lipgloss.Width(displayHelp) > innerWidth && innerWidth > 0 {
		if runes := []rune(displayHelp); len(runes) > innerWidth {
			displayHelp = string(runes[:innerWidth])
		}
	}
	b.WriteString(strings.Repeat(" ", padLeftHelp) + types.OverlaySubtitleStyle.Render(displayHelp))
	return b.String()
}

// sliceBodyLines clamps the scroll offset and returns the visible window of body lines.
func (s *SettingsOverlay) sliceBodyLines(bodyStr string, maxHeight int) []string {
	lines := strings.Split(strings.TrimRight(bodyStr, "\n"), "\n")
	startIdx := s.scrollOffset
	if startIdx+maxHeight > len(lines) {
		startIdx = len(lines) - maxHeight
	}
	if startIdx < 0 {
		startIdx = 0
	}
	s.scrollOffset = startIdx

	visible := make([]string, maxHeight)
	for i := range maxHeight {
		if startIdx+i < len(lines) {
			visible[i] = lines[startIdx+i]
		}
	}
	return visible
}

// View renders the interactive settings overlay
func (s *SettingsOverlay) View() string {
	if !config.IsInitialized() {
		return s.renderError("Configuration not initialized")
	}
	if s.activeDialog != nil {
		return s.renderWithDialog()
	}
	if s.confirmDialog != nil {
		return s.renderWithConfirmation()
	}

	innerWidth := max(s.width-4, 0)

	var body strings.Builder
	for i, section := range s.sections {
		if i > 0 {
			body.WriteString("\n")
		}
		body.WriteString(s.renderSection(section, i == s.selectedSection))
	}

	visibleLines := s.sliceBodyLines(body.String(), s.getVisibleBodyHeight())

	var content strings.Builder
	content.WriteString(s.renderViewHeader(innerWidth))
	content.WriteString(strings.Join(visibleLines, "\n"))
	content.WriteString("\n")
	content.WriteString(s.renderViewFooter(innerWidth))

	// Height(s.height - 2): lipgloss Height() sets inner content height,
	// so outer height == s.height.
	containerStyle := types.CreateOverlayContainerStyle(s.width).Height(s.height - 2)

	return lipgloss.Place(
		s.termWidth,
		s.termHeight,
		lipgloss.Center,
		lipgloss.Center,
		containerStyle.Render(content.String()),
	)
}

// buildHelpText creates the help text based on current state
func (s *SettingsOverlay) buildHelpText() string {
	shortcuts := []string{
		"↑↓:Nav",
		"Tab:Section",
	}

	// Add context-specific shortcuts based on current item type
	if len(s.sections) > 0 && s.selectedSection < len(s.sections) {
		section := s.sections[s.selectedSection]
		if s.selectedItem < len(section.items) {
			item := section.items[s.selectedItem]

			switch item.itemType {
			case itemTypeToggle:
				shortcuts = append(shortcuts, "Space/Enter:Toggle")
			case itemTypeText:
				shortcuts = append(shortcuts, "Enter:Edit")
			case itemTypeList:
				shortcuts = append(shortcuts, "Enter:Edit")
			}
		}
	}

	// Add whitelist-specific shortcuts if in that section
	if s.isInWhitelistSection() {
		shortcuts = append(shortcuts, "a:Add", "d:Del")
	}

	shortcuts = append(shortcuts, "^S:Save", "Esc:Close")
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
		fmt.Fprintf(&out, "%s %s", check, labelStyle.Render(item.displayName))
	case itemTypeList:
		fmt.Fprintf(&out, "• %s", labelStyle.Render(item.displayName))
	case itemTypeText:
		out.WriteString(s.renderTextItem(item, isFocused, labelStyle))
	}

	if item.modified {
		out.WriteString(lipgloss.NewStyle().Foreground(types.SalmonPink).Render(" *"))
	}

	out.WriteString("\n")
	return out.String()
}

// renderTextItem renders a text setting field, masking API key values.
func (s *SettingsOverlay) renderTextItem(item settingsItem, isFocused bool, labelStyle lipgloss.Style) string {
	isAPIKey := strings.HasSuffix(item.key, "_api_key") || item.key == "api_key"

	displayValue := fmt.Sprintf("%v", item.value)
	if isAPIKey && displayValue != "" {
		displayValue = "••••••••"
	}

	valueStyle := lipgloss.NewStyle().Foreground(types.MutedGray)
	if isFocused {
		valueStyle = valueStyle.Foreground(types.BrightWhite)
	}

	return fmt.Sprintf("%s: %s",
		labelStyle.Render(item.displayName),
		valueStyle.Render(displayValue))
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

func (s *SettingsOverlay) showAddPatternDialog() tea.Cmd {
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
			s.activeDialog = nil
			return nil
		},
		onCancel: func() {
			s.activeDialog = nil
		},
	}
	return tickCursorBlink()
}

func (s *SettingsOverlay) showEditPatternDialog() tea.Cmd {
	section := s.sections[s.selectedSection]
	item := section.items[s.selectedItem]
	data, ok := item.value.(map[string]any)
	if !ok {
		return nil
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
			s.activeDialog = nil
			return nil
		},
		onCancel: func() {
			s.activeDialog = nil
		},
	}
	return tickCursorBlink()
}

func (s *SettingsOverlay) showDeleteConfirmation() {
	s.confirmDialog = &confirmDialog{
		title:       "Delete Pattern",
		message:     "Delete this command whitelist pattern?",
		yesLabel:    "Delete pattern",
		noLabel:     "Keep pattern",
		cancelLabel: "Cancel",
		onYes: func() {
			s.deletePattern(s.selectedItem)
			s.confirmDialog = nil
		},
		onNo: func() {
			s.confirmDialog = nil
		},
	}
}

func (s *SettingsOverlay) showUnsavedChangesDialog() {
	s.confirmDialog = &confirmDialog{
		title:       "Unsaved Changes",
		message:     "Save changes before closing settings?",
		yesLabel:    "Save and close",
		noLabel:     "Discard changes",
		cancelLabel: "Keep editing",
		onYes: func() {
			// Save settings and clear state - handleConfirmInput will close overlay
			_ = s.saveSettings() //nolint:errcheck
			s.hasChanges = false
			s.confirmDialog = nil
		},
		onNo: func() {
			// Discard changes and clear state - handleConfirmInput will close overlay
			s.hasChanges = false
			s.confirmDialog = nil
		},
	}
}

func (s *SettingsOverlay) showTextFieldEditDialog() tea.Cmd {
	section := &s.sections[s.selectedSection]
	if s.selectedItem >= len(section.items) {
		return nil
	}

	item := &section.items[s.selectedItem]

	// Determine if this is an API key field for masking
	isAPIKey := strings.HasSuffix(item.key, "_api_key") || item.key == "api_key"

	// Get current value as string
	currentValue := ""
	if str, ok := item.value.(string); ok {
		currentValue = str
	}

	// Determine field type based on whether it's an API key
	fType := fieldTypeText
	if isAPIKey {
		fType = fieldTypePassword
	}

	s.activeDialog = &inputDialog{
		title: fmt.Sprintf("Edit %s", item.displayName),
		fields: []inputField{
			{
				label:     item.displayName,
				key:       "value",
				value:     currentValue,
				fieldType: fType,
				maxLength: 500,
				validator: func(v string) error {
					if v == "" {
						return fmt.Errorf("%s cannot be empty", item.displayName)
					}
					return nil
				},
			},
		},
		selectedField: 0,
		onConfirm: func(values map[string]string) error {
			newValue := values["value"]
			item.value = newValue
			item.modified = true
			s.hasChanges = true
			s.activeDialog = nil
			return nil
		},
		onCancel: func() {
			s.activeDialog = nil
		},
	}
	return tickCursorBlink()
}

func (s *SettingsOverlay) addPattern(pattern, description, matchType string) {
	// Implementation of adding pattern to the local state
	// This updates s.sections directly
	for i, section := range s.sections {
		if section.id == sectionCommandWhitelist {
			newItem := settingsItem{
				key:         fmt.Sprintf("pattern_new_%d", len(section.items)),
				displayName: fmt.Sprintf("%s - %s", pattern, description),
				value: map[string]any{
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
				s.sections[i].items[index].value = map[string]any{
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

	// In Bubble Tea v1, bracketed paste arrives as a KeyMsg with Paste: true
	// and Type: KeyRunes. Handle it before the key-switch so pasted content
	// is appended to the active field rather than discarded.
	if keyMsg.Paste {
		return s.handleDialogPaste(string(keyMsg.Runes))
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
	case "ctrl+u":
		s.handleDialogClear()
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

	// Dialog is cleared by onConfirm callback
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
	} else if field.fieldType == fieldTypeText || field.fieldType == fieldTypePassword {
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
	if (field.fieldType == fieldTypeText || field.fieldType == fieldTypePassword) && len(field.value) > 0 {
		// Use rune-based slicing to properly handle multi-byte UTF-8 characters
		runes := []rune(field.value)
		if len(runes) > 0 {
			field.value = string(runes[:len(runes)-1])
			field.errorMsg = ""
		}
	}
}

// handleDialogClear clears the current field value
func (s *SettingsOverlay) handleDialogClear() {
	field := &s.activeDialog.fields[s.activeDialog.selectedField]
	if field.fieldType == fieldTypeText || field.fieldType == fieldTypePassword {
		field.value = ""
		field.errorMsg = ""
	}
}

// handleDialogCharInput handles character input in dialog
func (s *SettingsOverlay) handleDialogCharInput(keyMsg tea.KeyMsg) {
	field := &s.activeDialog.fields[s.activeDialog.selectedField]
	if field.fieldType == fieldTypeText || field.fieldType == fieldTypePassword {
		if keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) > 0 &&
			(field.maxLength == 0 || len([]rune(field.value)) < field.maxLength) {
			field.value += string(keyMsg.Runes)
			field.errorMsg = ""
		}
	}
}

// handleDialogPaste inserts pasted text into the currently focused settings field.
// Non-printable runes (newlines, carriage returns, tabs, escape sequences, etc.)
// are stripped — all settings fields are single-line values (API keys, model
// names, base URLs). Password managers sometimes append a trailing newline or
// tab to copied secrets; terminal escape sequences must not be saved to config.
func (s *SettingsOverlay) handleDialogPaste(text string) (types.Overlay, tea.Cmd) {
	text = strings.Map(func(r rune) rune {
		if !unicode.IsPrint(r) {
			return -1
		}
		return r
	}, text)

	if text == "" {
		return s, nil
	}

	field := &s.activeDialog.fields[s.activeDialog.selectedField]
	if field.fieldType == fieldTypeText || field.fieldType == fieldTypePassword {
		if field.maxLength == 0 {
			field.value += text
		} else {
			currentRunes := len([]rune(field.value))
			remainingRunes := field.maxLength - currentRunes
			if remainingRunes > 0 {
				runes := []rune(text)
				if len(runes) > remainingRunes {
					runes = runes[:remainingRunes]
				}
				field.value += string(runes)
			}
		}
		field.errorMsg = ""
	}

	return s, nil
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
			// After onYes callback, confirmDialog and hasChanges should both be cleared
			// Return nil to close the overlay
			return nil, nil

		case "n", "N":
			if s.confirmDialog.onNo != nil {
				s.confirmDialog.onNo()
			}
			// After onNo callback, confirmDialog and hasChanges should both be cleared
			// Return nil to close the overlay
			return nil, nil

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
//
//nolint:gocyclo // UI dialog rendering has inherent complexity
func (s *SettingsOverlay) renderInputDialog() string {
	if s.activeDialog == nil {
		return ""
	}

	var content strings.Builder

	// Flat modal redesign: Title + separator
	titleStyle := lipgloss.NewStyle().Foreground(types.SalmonPink).Bold(true)
	titleText := titleStyle.Render(s.activeDialog.title)

	// Create separator (fixed width dialog = 60 inner width, minus borders = 56)
	var sepStr strings.Builder
	for range 56 {
		sepStr.WriteString("─")
	}
	separator := lipgloss.NewStyle().Foreground(types.MutedGray).Render(sepStr.String())

	content.WriteString(titleText)
	content.WriteString("\n")
	content.WriteString(separator)
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
		case fieldTypeText, fieldTypePassword:
			// Text input field - set a fixed width
			const fieldWidth = 60

			fieldStyle := lipgloss.NewStyle().
				Foreground(types.BrightWhite).
				Padding(0, 1).
				Width(fieldWidth)

			// Use flat borders for selected inputs to match design system
			if isSelected {
				fieldStyle = fieldStyle.Border(lipgloss.NormalBorder()).
					BorderForeground(types.SalmonPink)
			}

			// Mask password fields
			value := field.value
			if field.fieldType == fieldTypePassword && value != "" {
				value = strings.Repeat("•", len(value))
			}
			if isSelected {
				// Blinking cursor
				if s.cursorBlink {
					value += "▸"
				} else {
					value += " " // Space maintains field width when cursor is hidden
				}
			}

			// Truncate from the left if value is too long (show end of string)
			// Account for padding (2 chars) when calculating max displayable length
			// Use rune-based slicing to avoid splitting multi-byte UTF-8 characters
			maxDisplayLen := fieldWidth - 2
			runes := []rune(value)
			if len(runes) > maxDisplayLen {
				value = string(runes[len(runes)-maxDisplayLen:])
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

	// Create dialog box with flat border matching redesign
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(types.MutedGray).
		Padding(1, 2)

	return dialogStyle.Render(content.String())
}

// renderConfirmDialog renders a confirmation dialog
func (s *SettingsOverlay) renderConfirmDialog() string {
	if s.confirmDialog == nil {
		return ""
	}

	var content strings.Builder

	// Flat modal redesign: Title + separator
	titleStyle := lipgloss.NewStyle().Foreground(types.SalmonPink).Bold(true)
	titleText := titleStyle.Render(s.confirmDialog.title)

	// Create separator (fixed width dialog = 60 inner width, minus borders = 56)
	separator := lipgloss.NewStyle().Foreground(types.MutedGray).Render(strings.Repeat(sepChar, 56))

	content.WriteString(titleText)
	content.WriteString("\n")
	content.WriteString(separator)
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
	yesLabel := s.confirmDialog.yesLabel
	if yesLabel == "" {
		yesLabel = "Yes"
	}
	noLabel := s.confirmDialog.noLabel
	if noLabel == "" {
		noLabel = "No"
	}
	cancelLabel := s.confirmDialog.cancelLabel
	if cancelLabel == "" {
		cancelLabel = "Cancel"
	}

	buttonRow := fmt.Sprintf("[y] %s    [n] %s    [Esc] %s", yesLabel, noLabel, cancelLabel)
	buttonStyle := lipgloss.NewStyle().Foreground(types.MutedGray)
	content.WriteString(buttonStyle.Render(buttonRow))

	// Create dialog box with flat border matching redesign
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(types.MutedGray).
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

	boxStyle := types.CreateOverlayContainerStyle(s.width).
		Height(s.height)

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
