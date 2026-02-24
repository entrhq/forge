package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemorySection(t *testing.T) {
	section := NewMemorySection()
	assert.NotNil(t, section)
	assert.False(t, section.Enabled) // default is false — requires explicit opt-in
	assert.Equal(t, "", section.ClassifierModel)
	assert.Equal(t, "", section.HypothesisModel)
	assert.Equal(t, "", section.EmbeddingModel)
	assert.Equal(t, "", section.EmbeddingBaseURL)
	assert.Equal(t, 10, section.RetrievalTopK)
	assert.Equal(t, 1, section.RetrievalHopDepth)
	assert.Equal(t, 5, section.RetrievalHypothesisCount)
	assert.Equal(t, 0, section.InjectionTokenBudget)
}

func TestMemorySection_ID(t *testing.T) {
	section := NewMemorySection()
	assert.Equal(t, SectionIDMemory, section.ID())
	assert.Equal(t, "memory", section.ID())
}

func TestMemorySection_Title(t *testing.T) {
	section := NewMemorySection()
	assert.Equal(t, "Memory Settings", section.Title())
}

func TestMemorySection_Description(t *testing.T) {
	section := NewMemorySection()
	desc := section.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "Configure long-term memory")
}

func TestMemorySection_Data(t *testing.T) {
	section := NewMemorySection()
	section.Enabled = false
	section.ClassifierModel = "gpt-4o"
	section.EmbeddingModel = "text-embedding-3-small"
	section.RetrievalTopK = 20

	data := section.Data()
	assert.Equal(t, false, data["enabled"])
	assert.Equal(t, "gpt-4o", data["classifier_model"])
	assert.Equal(t, "text-embedding-3-small", data["embedding_model"])
	assert.Equal(t, 20, data["retrieval_top_k"])
}

func TestMemorySection_SetData(t *testing.T) {
	tests := []struct {
		name                 string
		data                 map[string]any
		initial              *MemorySection
		expectEnabled        bool
		expectEmbeddingModel string
		expectTopK           int
		expectError          bool
	}{
		{
			name: "valid data on empty section",
			data: map[string]any{
				"enabled":         false,
				"embedding_model": "text-embedding-3-small",
				"retrieval_top_k": 15,
			},
			initial:              NewMemorySection(),
			expectEnabled:        false,
			expectEmbeddingModel: "text-embedding-3-small",
			expectTopK:           15,
			expectError:          false,
		},
		{
			name: "float64 parsed from json",
			data: map[string]any{
				"retrieval_top_k": float64(25),
			},
			initial:              NewMemorySection(),
			expectEnabled:        false, // unchanged from default (false)
			expectEmbeddingModel: "",
			expectTopK:           25,
			expectError:          false,
		},
		{
			name:                 "nil data does not change anything",
			data:                 nil,
			initial:              NewMemorySection(),
			expectEnabled:        false, // unchanged from default (false)
			expectEmbeddingModel: "",
			expectTopK:           10,
			expectError:          false,
		},
		{
			name: "invalid data types are ignored",
			data: map[string]any{
				"enabled":         "true", // string — ignored by SetData
				"retrieval_top_k": "10",   // string — ignored by SetData
			},
			initial:              NewMemorySection(),
			expectEnabled:        false, // unchanged from default (false)
			expectEmbeddingModel: "",
			expectTopK:           10,
			expectError:          false,
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
				assert.Equal(t, tt.expectEnabled, section.Enabled)
				assert.Equal(t, tt.expectEmbeddingModel, section.EmbeddingModel)
				assert.Equal(t, tt.expectTopK, section.RetrievalTopK)
			}
		})
	}
}

func TestMemorySection_Validate(t *testing.T) {
	section := NewMemorySection()
	err := section.Validate()
	assert.NoError(t, err)
}

func TestMemorySection_Reset(t *testing.T) {
	section := NewMemorySection()
	section.Enabled = true // change from default (false) to verify Reset restores it
	section.EmbeddingModel = "custom-model"
	section.RetrievalTopK = 50

	section.Reset()

	assert.Equal(t, false, section.Enabled) // Reset returns to default (false)
	assert.Equal(t, "", section.EmbeddingModel)
	assert.Equal(t, 10, section.RetrievalTopK)
}

func TestMemorySection_ThreadSafety(t *testing.T) {
	section := NewMemorySection()

	// Test concurrent reads and writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			section.SetEmbeddingModel("model")
			_ = section.GetEmbeddingModel()
			section.SetRetrievalTopK(10)
			_ = section.GetRetrievalTopK()
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMemorySection_IntegrationWithManager(t *testing.T) {
	t.Run("Save and Load", func(t *testing.T) {
		// Create a temporary file store
		tmpFile := filepath.Join(t.TempDir(), "config.json")
		store, err := NewFileStore(tmpFile)
		require.NoError(t, err)

		manager := NewManager(store)

		// Register Memory section
		section := NewMemorySection()
		err = manager.RegisterSection(section)
		require.NoError(t, err)

		// Update configuration
		section.SetEnabled(false)
		section.SetEmbeddingModel("text-embedding-3-small")
		section.SetRetrievalTopK(25)

		// Save configuration
		err = manager.SaveAll()
		require.NoError(t, err)

		// Create new section and manager to simulate restart
		newSection := NewMemorySection()
		newStore, err := NewFileStore(tmpFile)
		require.NoError(t, err)
		newManager := NewManager(newStore)
		err = newManager.RegisterSection(newSection)
		require.NoError(t, err)

		// Load configuration
		err = newManager.LoadAll()
		require.NoError(t, err)

		// Verify loaded values
		assert.Equal(t, false, newSection.IsEnabled())
		assert.Equal(t, "text-embedding-3-small", newSection.GetEmbeddingModel())
		assert.Equal(t, 25, newSection.GetRetrievalTopK())
	})
}
