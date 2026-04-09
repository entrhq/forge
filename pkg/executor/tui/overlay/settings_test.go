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

func (m *mockProvider) AnalyzeDocument(ctx context.Context, fileData []byte, mediaType string, prompt string) (string, error) {
	return "mock analysis", nil
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

			// In redesign, settings overlay width uses ComputeOverlayWidth(width, 0.90, 60, 140)
			// and height uses ComputeViewportHeight(height, 4)
			expectedWidth := max(tt.width*90/100, 60)
			if expectedWidth > 140 {
				expectedWidth = 140
			}

			expectedHeight := max(
				// terminal height - chrome - safe margin
				tt.height-4-4, 10)

			if overlay.width != expectedWidth {
				t.Errorf("width = %d, want %d", overlay.width, expectedWidth)
			}

			if overlay.height != expectedHeight {
				t.Errorf("height = %d, want %d", overlay.height, expectedHeight)
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

	expectedWidth1 := 90 // 100 * 0.9
	expectedHeight1 := 50 - 8

	if overlay.Width() != expectedWidth1 {
		t.Errorf("Width() = %d, want %d", overlay.Width(), expectedWidth1)
	}

	if overlay.Height() != expectedHeight1 {
		t.Errorf("Height() = %d, want %d", overlay.Height(), expectedHeight1)
	}

	// Test with different dimensions
	overlay2 := NewSettingsOverlay(80, 40)

	expectedWidth2 := 72 // 80 * 0.9
	expectedHeight2 := 40 - 8

	if overlay2.Width() != expectedWidth2 {
		t.Errorf("Width() = %d, want %d", overlay2.Width(), expectedWidth2)
	}

	if overlay2.Height() != expectedHeight2 {
		t.Errorf("Height() = %d, want %d", overlay2.Height(), expectedHeight2)
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
	for range 3 {
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

	dialog := newConfirmDialog(
		"Confirm Action",
		"Are you sure?",
		"Yes",
		"No",
		"Cancel",
		func() {
			yesCalled = true
			overlay.confirmDialog = nil
		},
		func() {
			noCalled = true
			overlay.confirmDialog = nil
		},
	)

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

	dialog := newConfirmDialog(
		"Confirm Action",
		"Are you sure?",
		"Yes",
		"No",
		"Cancel",
		nil,
		func() {
			t.Error("Escape should NOT call onNo - it just cancels")
		},
	)

	overlay.confirmDialog = dialog

	// Press Escape (should just close dialog without calling callbacks)
	result, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyEsc}, nil, nil)
	overlay = result.(*SettingsOverlay)

	if overlay.confirmDialog != nil {
		t.Error("Dialog should be closed after Escape")
	}
}

func TestShowUnsavedChangesDialog_UsesActionLabels(t *testing.T) {
	overlay := NewSettingsOverlay(100, 50)

	overlay.showUnsavedChangesDialog()

	if overlay.confirmDialog == nil {
		t.Fatal("expected unsaved changes dialog to be shown")
	}
	if got := overlay.confirmDialog.message; got != "Save changes before closing settings?" {
		t.Fatalf("message = %q, want %q", got, "Save changes before closing settings?")
	}
	if got := overlay.confirmDialog.yesLabel; got != "Save and close" {
		t.Fatalf("yesLabel = %q, want %q", got, "Save and close")
	}
	if got := overlay.confirmDialog.noLabel; got != "Discard changes" {
		t.Fatalf("noLabel = %q, want %q", got, "Discard changes")
	}
	if got := overlay.confirmDialog.cancelLabel; got != "Keep editing" {
		t.Fatalf("cancelLabel = %q, want %q", got, "Keep editing")
	}
}

func TestInputDialog_CharInput(t *testing.T) {
	newOverlayWithTextField := func(maxLength int) *SettingsOverlay {
		overlay := NewSettingsOverlay(100, 50)
		overlay.activeDialog = &inputDialog{
			title: "Test Dialog",
			fields: []inputField{
				{label: "API Key", key: "apiKey", value: "", fieldType: fieldTypeText, maxLength: maxLength},
			},
			selectedField: 0,
		}
		return overlay
	}

	t.Run("KeyRunes input is inserted into field", func(t *testing.T) {
		overlay := newOverlayWithTextField(0)
		result, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}, nil, nil)
		overlay = result.(*SettingsOverlay)
		if got := overlay.activeDialog.fields[0].value; got != "a" {
			t.Errorf("field value = %q, want %q", got, "a")
		}
	})

	t.Run("ctrl+c key is not inserted into field", func(t *testing.T) {
		overlay := newOverlayWithTextField(0)
		result, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyCtrlC}, nil, nil)
		overlay = result.(*SettingsOverlay)
		if got := overlay.activeDialog.fields[0].value; got != "" {
			t.Errorf("field value = %q, want empty (ctrl+c should not be typed)", got)
		}
	})

	t.Run("ctrl+v key is not inserted as literal text", func(t *testing.T) {
		overlay := newOverlayWithTextField(0)
		result, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyCtrlV}, nil, nil)
		overlay = result.(*SettingsOverlay)
		if got := overlay.activeDialog.fields[0].value; got != "" {
			t.Errorf("field value = %q, want empty (ctrl+v should not type literal text)", got)
		}
	})

	t.Run("maxLength is enforced using rune count not bytes", func(t *testing.T) {
		overlay := newOverlayWithTextField(3)
		overlay.activeDialog.fields[0].value = "ab"
		// Multi-byte rune — should be accepted (value is 2 runes, limit is 3)
		result, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("é")}, nil, nil)
		overlay = result.(*SettingsOverlay)
		if got := overlay.activeDialog.fields[0].value; got != "abé" {
			t.Errorf("field value = %q, want %q", got, "abé")
		}
		// Now at maxLength (3 runes) — further input should be dropped
		result, _ = overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}, nil, nil)
		overlay = result.(*SettingsOverlay)
		if got := overlay.activeDialog.fields[0].value; got != "abé" {
			t.Errorf("field value = %q, want %q (maxLength exceeded)", got, "abé")
		}
	})

	t.Run("unicode rune input is inserted correctly", func(t *testing.T) {
		overlay := newOverlayWithTextField(0)
		result, _ := overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ñ")}, nil, nil)
		overlay = result.(*SettingsOverlay)
		if got := overlay.activeDialog.fields[0].value; got != "ñ" {
			t.Errorf("field value = %q, want %q", got, "ñ")
		}
	})
}

func TestInputDialog_BracketedPaste(t *testing.T) {
	newOverlayWithTextField := func(maxLength int) *SettingsOverlay {
		overlay := NewSettingsOverlay(100, 50)
		overlay.activeDialog = &inputDialog{
			title: "Test Dialog",
			fields: []inputField{
				{label: "API Key", key: "apiKey", value: "", fieldType: fieldTypeText, maxLength: maxLength},
			},
			selectedField: 0,
		}
		return overlay
	}

	t.Run("pastes text into empty field", func(t *testing.T) {
		overlay := newOverlayWithTextField(0)

		result, _ := overlay.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune("sk-abc123"),
			Paste: true,
		}, nil, nil)
		overlay = result.(*SettingsOverlay)

		if got := overlay.activeDialog.fields[0].value; got != "sk-abc123" {
			t.Errorf("field value = %q, want %q", got, "sk-abc123")
		}
	})

	t.Run("appends to existing field value", func(t *testing.T) {
		overlay := newOverlayWithTextField(0)
		overlay.activeDialog.fields[0].value = "prefix-"

		result, _ := overlay.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune("suffix"),
			Paste: true,
		}, nil, nil)
		overlay = result.(*SettingsOverlay)

		if got := overlay.activeDialog.fields[0].value; got != "prefix-suffix" {
			t.Errorf("field value = %q, want %q", got, "prefix-suffix")
		}
	})

	t.Run("strips newlines from pasted text", func(t *testing.T) {
		overlay := newOverlayWithTextField(0)

		result, _ := overlay.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune("line1\nline2\r\n"),
			Paste: true,
		}, nil, nil)
		overlay = result.(*SettingsOverlay)

		if got := overlay.activeDialog.fields[0].value; got != "line1line2" {
			t.Errorf("field value = %q, want %q (newlines should be stripped)", got, "line1line2")
		}
	})

	t.Run("strips tabs from pasted text", func(t *testing.T) {
		overlay := newOverlayWithTextField(0)

		result, _ := overlay.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune("col1\tcol2"),
			Paste: true,
		}, nil, nil)
		overlay = result.(*SettingsOverlay)

		if got := overlay.activeDialog.fields[0].value; got != "col1col2" {
			t.Errorf("field value = %q, want %q (tabs should be stripped)", got, "col1col2")
		}
	})

	t.Run("respects maxLength when pasting", func(t *testing.T) {
		overlay := newOverlayWithTextField(10)

		result, _ := overlay.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune("this-is-way-too-long"),
			Paste: true,
		}, nil, nil)
		overlay = result.(*SettingsOverlay)

		got := overlay.activeDialog.fields[0].value
		if len([]rune(got)) > 10 {
			t.Errorf("field value %q exceeds maxLength 10", got)
		}
		if got != "this-is-wa" {
			t.Errorf("field value = %q, want %q", got, "this-is-wa")
		}
	})

	t.Run("respects maxLength when field already has content", func(t *testing.T) {
		overlay := newOverlayWithTextField(10)
		overlay.activeDialog.fields[0].value = "12345"

		result, _ := overlay.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune("abcdefgh"),
			Paste: true,
		}, nil, nil)
		overlay = result.(*SettingsOverlay)

		got := overlay.activeDialog.fields[0].value
		if got != "12345abcde" {
			t.Errorf("field value = %q, want %q", got, "12345abcde")
		}
	})

	t.Run("paste only whitespace is a no-op", func(t *testing.T) {
		overlay := newOverlayWithTextField(0)
		overlay.activeDialog.fields[0].value = "existing"

		result, _ := overlay.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune("\n\r\t"),
			Paste: true,
		}, nil, nil)
		overlay = result.(*SettingsOverlay)

		if got := overlay.activeDialog.fields[0].value; got != "existing" {
			t.Errorf("field value = %q, want %q (whitespace-only paste should be no-op)", got, "existing")
		}
	})

	t.Run("paste into password field works", func(t *testing.T) {
		overlay := NewSettingsOverlay(100, 50)
		overlay.activeDialog = &inputDialog{
			title: "Test Dialog",
			fields: []inputField{
				{label: "Password", key: "pass", value: "", fieldType: fieldTypePassword},
			},
			selectedField: 0,
		}

		result, _ := overlay.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune("s3cr3t!"),
			Paste: true,
		}, nil, nil)
		overlay = result.(*SettingsOverlay)

		if got := overlay.activeDialog.fields[0].value; got != "s3cr3t!" {
			t.Errorf("password field value = %q, want %q", got, "s3cr3t!")
		}
	})

	t.Run("clears error message on successful paste", func(t *testing.T) {
		overlay := newOverlayWithTextField(0)
		overlay.activeDialog.fields[0].errorMsg = "previous error"

		result, _ := overlay.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune("valid-input"),
			Paste: true,
		}, nil, nil)
		overlay = result.(*SettingsOverlay)

		if got := overlay.activeDialog.fields[0].errorMsg; got != "" {
			t.Errorf("errorMsg = %q, want empty after successful paste", got)
		}
	})

	t.Run("strips escape sequences from pasted text", func(t *testing.T) {
		overlay := newOverlayWithTextField(0)

		result, _ := overlay.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune("abc\x1b[31mred\x1b[0m"),
			Paste: true,
		}, nil, nil)
		overlay = result.(*SettingsOverlay)

		if got := overlay.activeDialog.fields[0].value; got != "abc[31mred[0m" {
			t.Errorf("field value = %q, want %q (escape sequences should be stripped)", got, "abc[31mred[0m")
		}
	})

	t.Run("maxLength uses rune count not byte count", func(t *testing.T) {
		overlay := newOverlayWithTextField(5)
		// Seed with 3 multi-byte runes (9 bytes, 3 runes)
		overlay.activeDialog.fields[0].value = "éàü"

		result, _ := overlay.Update(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune("xy"),
			Paste: true,
		}, nil, nil)
		overlay = result.(*SettingsOverlay)

		got := overlay.activeDialog.fields[0].value
		if len([]rune(got)) > 5 {
			t.Errorf("field rune count %d exceeds maxLength 5", len([]rune(got)))
		}
		if got != "éàüxy" {
			t.Errorf("field value = %q, want %q", got, "éàüxy")
		}
	})
}
