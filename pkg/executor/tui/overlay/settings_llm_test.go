package overlay

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrhq/forge/internal/testing/configtest"
	"github.com/entrhq/forge/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingsOverlay_LLMDataLoading(t *testing.T) {
	configtest.WithGlobalManager(t, func() {
		// 1. Get the global manager instance set up by the test helper
		manager := config.Global()

		// 2. Register a new LLM section
		llmSection := config.NewLLMSection()
		err := manager.RegisterSection(llmSection)
		require.NoError(t, err)

		// 3. Set values in the config
		llmSection.SetModel("file-model")
		llmSection.SetBaseURL("https://file.url")
		llmSection.SetAPIKey("file-key")
		err = manager.SaveAll()
		require.NoError(t, err)

		// 4. Create a mock provider with different values to test provider override
		provider := &mockProvider{
			model:   "provider-model",
			baseURL: "https://provider.url",
			apiKey:  "provider-key",
		}

		// 5. Create the overlay
		overlay := NewSettingsOverlayWithCallback(120, 80, nil, provider)
		require.NotNil(t, overlay)

		// 6. Find the LLM section
		var foundSection *settingsSection
		for i, section := range overlay.sections {
			if section.id == "llm" {
				foundSection = &overlay.sections[i]
				break
			}
		}
		require.NotNil(t, foundSection, "LLM section not found in overlay")

		// 7. Assert that the values from the provider take precedence
		assert.Equal(t, "provider-model", getItemValue(t, foundSection.items, "model"), "Model should come from provider")
		assert.Equal(t, "https://provider.url", getItemValue(t, foundSection.items, "base_url"), "Base URL should come from provider")
		assert.Equal(t, "provider-key", getItemValue(t, foundSection.items, "api_key"), "API key should come from provider")
	})
}

func TestSettingsOverlay_EditAndSaveLLMSettings(t *testing.T) {
	configtest.WithGlobalManager(t, func() {
		manager := config.Global()
		llmSection := config.NewLLMSection()
		err := manager.RegisterSection(llmSection)
		require.NoError(t, err)

		callbackCalled := false
		callback := func() error {
			callbackCalled = true
			return nil
		}

		provider := &mockProvider{
			model:   "initial-model",
			baseURL: "https://initial.url",
			apiKey:  "initial-key",
		}

		overlay := NewSettingsOverlayWithCallback(120, 80, callback, provider)
		require.NotNil(t, overlay)

		// Navigate to the LLM section (assuming it's the second section after auto_approval)
		overlay.selectedSection = 1

		// --- Edit Model ---
		overlay.selectedItem = 0 // "Model" field
		overlay, cmd := overlay.handleKeyPress(tea.KeyMsg{Type: tea.KeyEnter})
		require.Nil(t, cmd)
		require.NotNil(t, overlay.activeDialog)

		// Simulate typing new model name
		overlay.activeDialog.fields[0].value = "" // clear validator's default
		overlay = sendCh(overlay, "new-model")
		overlay, cmd = overlay.handleDialogConfirm()
		require.Nil(t, cmd)
		assert.Nil(t, overlay.activeDialog, "Dialog should be closed after confirm")
		assert.True(t, overlay.hasChanges, "hasChanges should be true after edit")
		assert.Equal(t, "new-model", getItemValue(t, overlay.sections[1].items, "model"))

		// --- Save Changes ---
		overlay, cmd = overlay.handleKeyPress(tea.KeyMsg{Type: tea.KeyCtrlS})
		require.Nil(t, cmd)
		assert.False(t, overlay.hasChanges, "hasChanges should be false after save")
		assert.True(t, callbackCalled, "onLLMSettingsChange callback should be called on save")

		// --- Verify config file was updated ---
		reloadedSection := config.NewLLMSection()
		err = manager.RegisterSection(reloadedSection)
		require.NoError(t, err)
		err = manager.LoadAll()
		require.NoError(t, err)

		assert.Equal(t, "new-model", reloadedSection.GetModel(), "Saved model should be in config")
		assert.Equal(t, "https://initial.url", reloadedSection.GetBaseURL(), "BaseURL should be unchanged")
	})
}

func TestSettingsOverlay_PasswordMaskingForAPIKey(t *testing.T) {
	configtest.WithGlobalManager(t, func() {
		manager := config.Global()
		llmSection := config.NewLLMSection()
		err := manager.RegisterSection(llmSection)
		require.NoError(t, err)

		provider := &mockProvider{apiKey: "super-secret-key"}
		overlay := NewSettingsOverlayWithCallback(120, 80, nil, provider)

		// Go to LLM section
		overlay.selectedSection = 1
		// Select API key field
		overlay.selectedItem = 2

		// --- Check masking in the main view ---
		view := overlay.View()
		assert.NotContains(t, view, "super-secret-key")
		assert.Contains(t, view, "••••••••", "API key should be masked in the view")

		// --- Check masking in the edit dialog ---
		overlay, _ = overlay.handleKeyPress(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, overlay.activeDialog)

		dialogView := overlay.renderInputDialog()
		assert.NotContains(t, dialogView, "super-secret-key")
		assert.Contains(t, dialogView, "••••••••••••••", "API key should be masked in the edit dialog")
	})
}

// Helper function to get a value from a slice of settingsItems
func getItemValue(t *testing.T, items []settingsItem, key string) interface{} {
	t.Helper()
	for _, item := range items {
		if item.key == key {
			return item.value
		}
	}
	t.Fatalf("item with key %q not found", key)
	return nil
}

// sendCh simulates typing a string into an active dialog
func sendCh(s *SettingsOverlay, text string) *SettingsOverlay {
	var cmd tea.Cmd
	for _, ch := range text {
		s, cmd = s.handleDialogInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		if cmd != nil {
			// Handle any commands that might be returned, like ticks
			s.Update(cmd(), nil, nil)
		}
	}
	return s
}
