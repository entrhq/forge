package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLLMSection(t *testing.T) {
	section := NewLLMSection()
	assert.NotNil(t, section)
	assert.Equal(t, "", section.Model)
	assert.Equal(t, "", section.BaseURL)
	assert.Equal(t, "", section.APIKey)
}

func TestLLMSection_ID(t *testing.T) {
	section := NewLLMSection()
	assert.Equal(t, SectionIDLLM, section.ID())
	assert.Equal(t, "llm", section.ID())
}

func TestLLMSection_Title(t *testing.T) {
	section := NewLLMSection()
	assert.Equal(t, "LLM Settings", section.Title())
}

func TestLLMSection_Description(t *testing.T) {
	section := NewLLMSection()
	desc := section.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "LLM")
	assert.Contains(t, desc, "model")
}

func TestLLMSection_Data(t *testing.T) {
	section := NewLLMSection()
	section.Model = "gpt-4-turbo"
	section.BaseURL = "https://api.openai.com/v1"
	section.APIKey = "sk-test123"

	data := section.Data()
	assert.Equal(t, "gpt-4-turbo", data["model"])
	assert.Equal(t, "https://api.openai.com/v1", data["base_url"])
	assert.Equal(t, "sk-test123", data["api_key"])
}

func TestLLMSection_SetData(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]any
		initial     *LLMSection
		expectModel string
		expectURL   string
		expectKey   string
		expectError bool
	}{
		{
			name: "valid data on empty section",
			data: map[string]any{
				"model":    "gpt-4-turbo",
				"base_url": "https://custom.api.com",
				"api_key":  "sk-custom",
			},
			initial:     NewLLMSection(),
			expectModel: "gpt-4-turbo",
			expectURL:   "https://custom.api.com",
			expectKey:   "sk-custom",
			expectError: false,
		},
		{
			name: "partial data updates only specified fields",
			data: map[string]any{
				"model": "claude-3",
			},
			initial: &LLMSection{
				Model:   "old-model",
				BaseURL: "old-url",
				APIKey:  "old-key",
			},
			expectModel: "claude-3",
			expectURL:   "old-url", // should be preserved
			expectKey:   "old-key", // should be preserved
			expectError: false,
		},
		{
			name: "nil data does not change anything",
			data: nil,
			initial: &LLMSection{
				Model:   "existing-model",
				BaseURL: "existing-url",
				APIKey:  "existing-key",
			},
			expectModel: "existing-model",
			expectURL:   "existing-url",
			expectKey:   "existing-key",
			expectError: false,
		},
		{
			name: "empty data does not change anything",
			data: map[string]any{},
			initial: &LLMSection{
				Model:   "existing-model",
				BaseURL: "existing-url",
				APIKey:  "existing-key",
			},
			expectModel: "existing-model",
			expectURL:   "existing-url",
			expectKey:   "existing-key",
			expectError: false,
		},
		{
			name: "invalid data types are ignored",
			data: map[string]any{
				"model":    12345,
				"base_url": true,
				"api_key":  nil,
			},
			initial: &LLMSection{
				Model:   "existing-model",
				BaseURL: "existing-url",
				APIKey:  "existing-key",
			},
			expectModel: "existing-model",
			expectURL:   "existing-url",
			expectKey:   "existing-key",
			expectError: false,
		},
		{
			name: "empty strings clear values",
			data: map[string]any{
				"model":   "",
				"api_key": "",
			},
			initial: &LLMSection{
				Model:   "existing-model",
				BaseURL: "existing-url",
				APIKey:  "existing-key",
			},
			expectModel: "",
			expectURL:   "existing-url", // not in data, should be preserved
			expectKey:   "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := tt.initial
			err := section.SetData(tt.data)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectModel, section.Model)
				assert.Equal(t, tt.expectURL, section.BaseURL)
				assert.Equal(t, tt.expectKey, section.APIKey)
			}
		})
	}
}

func TestLLMSection_Validate(t *testing.T) {
	tests := []struct {
		name  string
		model string
	}{
		{
			name:  "valid model",
			model: "gpt-4o",
		},
		{
			name:  "empty model is allowed",
			model: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := NewLLMSection()
			section.Model = tt.model

			err := section.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestLLMSection_Reset(t *testing.T) {
	section := NewLLMSection()
	section.Model = "custom-model"
	section.BaseURL = "https://custom.api.com"
	section.APIKey = "sk-custom"

	section.Reset()

	assert.Equal(t, "", section.Model)
	assert.Equal(t, "", section.BaseURL)
	assert.Equal(t, "", section.APIKey)
}

func TestLLMSection_GettersSetters(t *testing.T) {
	section := NewLLMSection()

	// Test Model
	section.SetModel("gpt-4-turbo")
	assert.Equal(t, "gpt-4-turbo", section.GetModel())

	// Test BaseURL
	section.SetBaseURL("https://api.example.com")
	assert.Equal(t, "https://api.example.com", section.GetBaseURL())

	// Test APIKey
	section.SetAPIKey("sk-test123")
	assert.Equal(t, "sk-test123", section.GetAPIKey())
}

func TestLLMSection_ThreadSafety(t *testing.T) {
	section := NewLLMSection()

	// Test concurrent reads and writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			section.SetModel("model")
			_ = section.GetModel()
			section.SetBaseURL("url")
			_ = section.GetBaseURL()
			section.SetAPIKey("key")
			_ = section.GetAPIKey()
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestLLMSection_IntegrationWithManager(t *testing.T) {
	t.Run("Save and Load", func(t *testing.T) {
		// Create a temporary file store
		tmpFile := filepath.Join(t.TempDir(), "config.json")
		store, err := NewFileStore(tmpFile)
		require.NoError(t, err)

		manager := NewManager(store)

		// Register LLM section
		section := NewLLMSection()
		err = manager.RegisterSection(section)
		require.NoError(t, err)

		// Update configuration
		section.SetModel("gpt-4-turbo")
		section.SetBaseURL("https://api.openai.com/v1")
		section.SetAPIKey("sk-test")

		// Save configuration
		err = manager.SaveAll()
		require.NoError(t, err)

		// Create new section and manager to simulate restart
		newSection := NewLLMSection()
		newStore, err := NewFileStore(tmpFile)
		require.NoError(t, err)
		newManager := NewManager(newStore)
		err = newManager.RegisterSection(newSection)
		require.NoError(t, err)

		// Load configuration
		err = newManager.LoadAll()
		require.NoError(t, err)

		// Verify loaded values
		assert.Equal(t, "gpt-4-turbo", newSection.GetModel())
		assert.Equal(t, "https://api.openai.com/v1", newSection.GetBaseURL())
		assert.Equal(t, "sk-test", newSection.GetAPIKey())
	})
}
