package overlay

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/types"
)

// mockProvider implements llm.Provider for testing
type mockProvider struct {
	model   string
	baseURL string
	apiKey  string
}

func (m *mockProvider) GetModel() string {
	return m.model
}

func (m *mockProvider) GetBaseURL() string {
	return m.baseURL
}

func (m *mockProvider) GetAPIKey() string {
	return m.apiKey
}

func (m *mockProvider) StreamCompletion(ctx context.Context, messages []*types.Message) (<-chan *llm.StreamChunk, error) {
	return nil, nil
}

func (m *mockProvider) Complete(ctx context.Context, messages []*types.Message) (*types.Message, error) {
	return nil, nil
}

func (m *mockProvider) GetModelInfo() *types.ModelInfo {
	return &types.ModelInfo{Name: m.model}
}

func TestNewSettingsOverlay(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{
			name:   "standard dimensions",
			width:  100,
			height: 50,
		},
		{
			name:   "small dimensions",
			width:  40,
			height: 20,
		},
		{
			name:   "large dimensions",
			width:  200,
			height: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overlay := NewSettingsOverlay(tt.width, tt.height)

			if overlay == nil {
				t.Fatal("NewSettingsOverlay returned nil")
			}

			if overlay.width != tt.width {
				t.Errorf("width = %d, want %d", overlay.width, tt.width)
			}

			if overlay.height != tt.height {
				t.Errorf("height = %d, want %d", overlay.height, tt.height)
			}

			if !overlay.focused {
				t.Error("overlay should be focused by default")
			}

			if overlay.editMode {
				t.Error("overlay should not be in edit mode by default")
			}

			if overlay.hasChanges {
				t.Error("overlay should not have changes by default")
			}

			if !overlay.cursorBlink {
				t.Error("cursor blink should be enabled by default")
			}
		})
	}
}

func TestNewSettingsOverlayWithCallback(t *testing.T) {
	provider := &mockProvider{
		model:   "test-model",
		baseURL: "https://api.test.com",
		apiKey:  "test-key-123",
	}

	callbackCalled := false
	callback := func() error {
		callbackCalled = true
		return nil
	}

	overlay := NewSettingsOverlayWithCallback(100, 50, callback, provider)

	if overlay == nil {
		t.Fatal("NewSettingsOverlayWithCallback returned nil")
	}

	if overlay.provider == nil {
		t.Error("provider not set")
	}

	if overlay.onLLMSettingsChange == nil {
		t.Error("callback not set")
	}

	// Test callback execution
	if overlay.onLLMSettingsChange != nil {
		err := overlay.onLLMSettingsChange()
		if err != nil {
			t.Errorf("callback returned error: %v", err)
		}
		if !callbackCalled {
			t.Error("callback was not called")
		}
	}
}

func TestSettingsOverlay_CursorBlink(t *testing.T) {
	overlay := NewSettingsOverlay(100, 50)

	// Initial state
	if !overlay.cursorBlink {
		t.Error("cursor should be visible initially")
	}

	// Send blink message
	result, cmd := overlay.Update(cursorBlinkMsg{}, nil, nil)
	overlay = result.(*SettingsOverlay)

	// Cursor should toggle
	if overlay.cursorBlink {
		t.Error("cursor should be invisible after first blink")
	}

	// Command should be returned to continue blinking
	if cmd == nil {
		t.Error("blink command should be returned")
	}

	// Send another blink message
	result, cmd = overlay.Update(cursorBlinkMsg{}, nil, nil)
	overlay = result.(*SettingsOverlay)

	// Cursor should toggle back
	if !overlay.cursorBlink {
		t.Error("cursor should be visible after second blink")
	}

	if cmd == nil {
		t.Error("blink command should be returned")
	}
}

func TestSettingsOverlay_Focused(t *testing.T) {
	overlay := NewSettingsOverlay(100, 50)

	if !overlay.Focused() {
		t.Error("overlay should be focused by default")
	}

	// Defocus
	overlay.focused = false

	if overlay.Focused() {
		t.Error("overlay should not be focused when focus is false")
	}
}

func TestSettingsOverlay_Dimensions(t *testing.T) {
	overlay := NewSettingsOverlay(100, 50)

	if overlay.Width() != 100 {
		t.Errorf("Width() = %d, want 100", overlay.Width())
	}

	if overlay.Height() != 50 {
		t.Errorf("Height() = %d, want 50", overlay.Height())
	}

	// Test with different dimensions
	overlay2 := NewSettingsOverlay(80, 40)

	if overlay2.Width() != 80 {
		t.Errorf("Width() = %d, want 80", overlay2.Width())
	}

	if overlay2.Height() != 40 {
		t.Errorf("Height() = %d, want 40", overlay2.Height())
	}
}

func TestInputDialog_Navigation(t *testing.T) {
	overlay := NewSettingsOverlay(100, 50)

	// Create a dialog with multiple fields
	dialog := &inputDialog{
		title: "Test Dialog",
		fields: []inputField{
			{label: "Field 1", key: "field1", value: "", fieldType: fieldTypeText},
			{label: "Field 2", key: "field2", value: "", fieldType: fieldTypeText},
			{label: "Field 3", key: "field3", value: "", fieldType: fieldTypeText},
		},
		selectedField: 0,
	}

	overlay.activeDialog = dialog

	// Test Tab navigation (forward)
	result, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyTab}, nil, nil)
	overlay = result.(*SettingsOverlay)

	if overlay.activeDialog.selectedField != 1 {
		t.Errorf("Tab should move to next field, got field %d", overlay.activeDialog.selectedField)
	}

	// Test Shift+Tab navigation (backward)
	result, _ = overlay.Update(tea.KeyMsg{Type: tea.KeyShiftTab}, nil, nil)
	overlay = result.(*SettingsOverlay)

	if overlay.activeDialog.selectedField != 0 {
		t.Errorf("Shift+Tab should move to previous field, got field %d", overlay.activeDialog.selectedField)
	}

	// Test wrapping at end
	overlay.activeDialog.selectedField = 2
	result, _ = overlay.Update(tea.KeyMsg{Type: tea.KeyTab}, nil, nil)
	overlay = result.(*SettingsOverlay)

	if overlay.activeDialog.selectedField != 0 {
		t.Errorf("Tab at last field should wrap to first, got field %d", overlay.activeDialog.selectedField)
	}

	// Test wrapping at beginning
	overlay.activeDialog.selectedField = 0
	result, _ = overlay.Update(tea.KeyMsg{Type: tea.KeyShiftTab}, nil, nil)
	overlay = result.(*SettingsOverlay)

	if overlay.activeDialog.selectedField != 2 {
		t.Errorf("Shift+Tab at first field should wrap to last, got field %d", overlay.activeDialog.selectedField)
	}
}

func TestInputDialog_TextInput(t *testing.T) {
	overlay := NewSettingsOverlay(100, 50)

	dialog := &inputDialog{
		title: "Test Dialog",
		fields: []inputField{
			{label: "Name", key: "name", value: "", fieldType: fieldTypeText, maxLength: 10},
		},
		selectedField: 0,
	}

	overlay.activeDialog = dialog

	tests := []struct {
		name          string
		input         string
		expectedValue string
		shouldAccept  bool
	}{
		{
			name:          "normal text",
			input:         "test",
			expectedValue: "test",
			shouldAccept:  true,
		},
		{
			name:          "exceeds max length",
			input:         "verylongtext",
			expectedValue: "verylongte", // Truncated to 10 chars
			shouldAccept:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset field
			overlay.activeDialog.fields[0].value = ""

			// Input each character
			for _, ch := range tt.input {
				result, _ := overlay.Update(tea.KeyMsg{
					Type:  tea.KeyRunes,
					Runes: []rune{ch},
				}, nil, nil)
				overlay = result.(*SettingsOverlay)
			}

			actualValue := overlay.activeDialog.fields[0].value
			if actualValue != tt.expectedValue {
				t.Errorf("field value = %q, want %q", actualValue, tt.expectedValue)
			}
		})
	}
}

func TestInputDialog_Backspace(t *testing.T) {
	overlay := NewSettingsOverlay(100, 50)

	dialog := &inputDialog{
		title: "Test Dialog",
		fields: []inputField{
			{label: "Name", key: "name", value: "test", fieldType: fieldTypeText},
		},
		selectedField: 0,
	}

	overlay.activeDialog = dialog

	// Press backspace
	result, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyBackspace}, nil, nil)
	overlay = result.(*SettingsOverlay)

	if overlay.activeDialog.fields[0].value != "tes" {
		t.Errorf("backspace should remove last character, got %q", overlay.activeDialog.fields[0].value)
	}

	// Backspace until empty
	for i := 0; i < 3; i++ {
		result, _ = overlay.Update(tea.KeyMsg{Type: tea.KeyBackspace}, nil, nil)
		overlay = result.(*SettingsOverlay)
	}

	if overlay.activeDialog.fields[0].value != "" {
		t.Errorf("field should be empty after all backspaces, got %q", overlay.activeDialog.fields[0].value)
	}

	// Backspace on empty field should not error
	result, _ = overlay.Update(tea.KeyMsg{Type: tea.KeyBackspace}, nil, nil)
	overlay = result.(*SettingsOverlay)

	if overlay.activeDialog.fields[0].value != "" {
		t.Errorf("backspace on empty field should keep it empty, got %q", overlay.activeDialog.fields[0].value)
	}
}

func TestInputDialog_Escape(t *testing.T) {
	overlay := NewSettingsOverlay(100, 50)

	cancelCalled := false
	dialog := &inputDialog{
		title: "Test Dialog",
		fields: []inputField{
			{label: "Name", key: "name", value: "test", fieldType: fieldTypeText},
		},
		selectedField: 0,
		onCancel: func() {
			cancelCalled = true
			overlay.activeDialog = nil
		},
	}

	overlay.activeDialog = dialog

	// Press Escape
	result, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyEsc}, nil, nil)
	overlay = result.(*SettingsOverlay)

	if !cancelCalled {
		t.Error("Escape should call onCancel")
	}

	if overlay.activeDialog != nil {
		t.Error("Dialog should be closed after Escape")
	}
}

func TestInputDialog_Enter(t *testing.T) {
	overlay := NewSettingsOverlay(100, 50)

	confirmCalled := false
	var capturedValues map[string]string

	dialog := &inputDialog{
		title: "Test Dialog",
		fields: []inputField{
			{label: "Name", key: "name", value: "John", fieldType: fieldTypeText},
			{label: "Email", key: "email", value: "john@test.com", fieldType: fieldTypeText},
		},
		selectedField: 0,
		onConfirm: func(values map[string]string) error {
			confirmCalled = true
			capturedValues = values
			overlay.activeDialog = nil
			return nil
		},
	}

	overlay.activeDialog = dialog

	// Press Enter
	result, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyEnter}, nil, nil)
	overlay = result.(*SettingsOverlay)

	if !confirmCalled {
		t.Error("Enter should call onConfirm")
	}

	if overlay.activeDialog != nil {
		t.Error("Dialog should be closed after Enter")
	}

	// Verify captured values
	if capturedValues["name"] != "John" {
		t.Errorf("name = %q, want %q", capturedValues["name"], "John")
	}

	if capturedValues["email"] != "john@test.com" {
		t.Errorf("email = %q, want %q", capturedValues["email"], "john@test.com")
	}
}

func TestConfirmDialog_YesNo(t *testing.T) {
	overlay := NewSettingsOverlay(100, 50)

	yesCalled := false
	noCalled := false

	dialog := &confirmDialog{
		title:   "Confirm Action",
		message: "Are you sure?",
		onYes: func() {
			yesCalled = true
			overlay.confirmDialog = nil
		},
		onNo: func() {
			noCalled = true
			overlay.confirmDialog = nil
		},
	}

	overlay.confirmDialog = dialog

	// Test 'y' key - returns nil to close overlay
	result, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}, nil, nil)

	if !yesCalled {
		t.Error("'y' key should call onYes")
	}

	if result != nil {
		t.Error("Overlay should be closed (nil) after yes")
	}

	// Reset for no test
	overlay = NewSettingsOverlay(100, 50)
	yesCalled = false
	overlay.confirmDialog = dialog

	// Test 'n' key - returns nil to close overlay
	result, _ = overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}, nil, nil)

	if !noCalled {
		t.Error("'n' key should call onNo")
	}

	if result != nil {
		t.Error("Overlay should be closed (nil) after no")
	}
}

func TestConfirmDialog_Escape(t *testing.T) {
	overlay := NewSettingsOverlay(100, 50)

	dialog := &confirmDialog{
		title:   "Confirm Action",
		message: "Are you sure?",
		onNo: func() {
			t.Error("Escape should NOT call onNo - it just cancels")
		},
	}

	overlay.confirmDialog = dialog

	// Press Escape (should just close dialog without calling callbacks)
	result, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyEsc}, nil, nil)
	overlay = result.(*SettingsOverlay)

	if overlay.confirmDialog != nil {
		t.Error("Dialog should be closed after Escape")
	}
}
