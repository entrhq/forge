package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSection is a test implementation of the Section interface
type mockSection struct {
	id          string
	title       string
	description string
	data        map[string]interface{}
	validateErr error
}

func (m *mockSection) ID() string                                { return m.id }
func (m *mockSection) Title() string                             { return m.title }
func (m *mockSection) Description() string                       { return m.description }
func (m *mockSection) Data() map[string]interface{}              { return m.data }
func (m *mockSection) SetData(data map[string]interface{}) error { m.data = data; return nil }
func (m *mockSection) Validate() error                           { return m.validateErr }
func (m *mockSection) Reset()                                    { m.data = make(map[string]interface{}) }

// mockStore is a test implementation of the Store interface
type mockStore struct {
	sections map[string]map[string]interface{}
	loadErr  error
	saveErr  error
}

func newMockStore() *mockStore {
	return &mockStore{
		sections: make(map[string]map[string]interface{}),
	}
}

func (m *mockStore) Load() error {
	return m.loadErr
}

func (m *mockStore) Save() error {
	return m.saveErr
}

func (m *mockStore) GetSection(sectionID string) (map[string]interface{}, error) {
	if data, exists := m.sections[sectionID]; exists {
		return data, nil
	}
	return make(map[string]interface{}), nil
}

func (m *mockStore) SetSection(sectionID string, data map[string]interface{}) error {
	m.sections[sectionID] = data
	return nil
}

func (m *mockStore) GetAll() (map[string]map[string]interface{}, error) {
	return m.sections, nil
}

func (m *mockStore) SetAll(data map[string]map[string]interface{}) error {
	m.sections = data
	return nil
}

func TestNewManager(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)

	require.NotNil(t, manager)
	assert.Same(t, store, manager.Store())
	assert.Empty(t, manager.GetSections())
}

func TestManager_RegisterSection(t *testing.T) {
	t.Run("registers section successfully", func(t *testing.T) {
		manager := NewManager(newMockStore())
		section := &mockSection{id: "test", title: "Test"}

		err := manager.RegisterSection(section)
		require.NoError(t, err)

		retrieved, ok := manager.GetSection("test")
		require.True(t, ok, "Section not found after registration")
		assert.Equal(t, "test", retrieved.ID())
	})

	t.Run("prevents duplicate registration", func(t *testing.T) {
		manager := NewManager(newMockStore())
		section1 := &mockSection{id: "test", title: "Test 1"}
		section2 := &mockSection{id: "test", title: "Test 2"}

		err := manager.RegisterSection(section1)
		require.NoError(t, err)

		err = manager.RegisterSection(section2)
		require.Error(t, err)
	})

	t.Run("maintains registration order", func(t *testing.T) {
		manager := NewManager(newMockStore())

		section1 := &mockSection{id: "first", title: "First"}
		section2 := &mockSection{id: "second", title: "Second"}
		section3 := &mockSection{id: "third", title: "Third"}

		require.NoError(t, manager.RegisterSection(section1))
		require.NoError(t, manager.RegisterSection(section2))
		require.NoError(t, manager.RegisterSection(section3))

		sections := manager.GetSections()
		require.Len(t, sections, 3)
		assert.Equal(t, "first", sections[0].ID())
		assert.Equal(t, "second", sections[1].ID())
		assert.Equal(t, "third", sections[2].ID())
	})
}

func TestManager_GetSection(t *testing.T) {
	t.Run("returns existing section", func(t *testing.T) {
		manager := NewManager(newMockStore())
		section := &mockSection{id: "test", title: "Test"}
		manager.RegisterSection(section)

		retrieved, ok := manager.GetSection("test")
		require.True(t, ok)
		assert.Equal(t, "test", retrieved.ID())
	})

	t.Run("returns false for non-existent section", func(t *testing.T) {
		manager := NewManager(newMockStore())
		_, ok := manager.GetSection("nonexistent")
		assert.False(t, ok)
	})
}

func TestManager_GetSections(t *testing.T) {
	t.Run("returns all sections in order", func(t *testing.T) {
		manager := NewManager(newMockStore())

		section1 := &mockSection{id: "a", title: "A"}
		section2 := &mockSection{id: "b", title: "B"}

		manager.RegisterSection(section1)
		manager.RegisterSection(section2)

		sections := manager.GetSections()
		require.Len(t, sections, 2)
		assert.Equal(t, "a", sections[0].ID())
		assert.Equal(t, "b", sections[1].ID())
	})

	t.Run("returns empty slice for no sections", func(t *testing.T) {
		manager := NewManager(newMockStore())
		sections := manager.GetSections()
		assert.Empty(t, sections)
	})
}

func TestManager_LoadAll(t *testing.T) {
	t.Run("loads all sections from store", func(t *testing.T) {
		store := newMockStore()
		store.sections["test"] = map[string]interface{}{
			"key": "value",
		}

		manager := NewManager(store)
		section := &mockSection{
			id:   "test",
			data: make(map[string]interface{}),
		}
		manager.RegisterSection(section)

		err := manager.LoadAll()
		require.NoError(t, err)

		assert.Equal(t, "value", section.data["key"])
	})

	t.Run("handles store load error", func(t *testing.T) {
		store := newMockStore()
		store.loadErr = fmt.Errorf("load error")

		manager := NewManager(store)
		err := manager.LoadAll()
		require.Error(t, err)
	})

	t.Run("loads multiple sections", func(t *testing.T) {
		store := newMockStore()
		store.sections["section1"] = map[string]interface{}{"key1": "value1"}
		store.sections["section2"] = map[string]interface{}{"key2": "value2"}

		manager := NewManager(store)
		section1 := &mockSection{id: "section1", data: make(map[string]interface{})}
		section2 := &mockSection{id: "section2", data: make(map[string]interface{})}

		manager.RegisterSection(section1)
		manager.RegisterSection(section2)

		err := manager.LoadAll()
		require.NoError(t, err)

		assert.Equal(t, "value1", section1.data["key1"])
		assert.Equal(t, "value2", section2.data["key2"])
	})
}

func TestManager_SaveAll(t *testing.T) {
	t.Run("saves all sections to store", func(t *testing.T) {
		store := newMockStore()
		manager := NewManager(store)

		section := &mockSection{
			id: "test",
			data: map[string]interface{}{
				"key": "value",
			},
		}
		manager.RegisterSection(section)

		err := manager.SaveAll()
		require.NoError(t, err)

		savedData := store.sections["test"]
		assert.Equal(t, "value", savedData["key"])
	})

	t.Run("validates sections before saving", func(t *testing.T) {
		store := newMockStore()
		manager := NewManager(store)

		section := &mockSection{
			id:          "test",
			data:        map[string]interface{}{"key": "value"},
			validateErr: fmt.Errorf("validation error"),
		}
		manager.RegisterSection(section)

		err := manager.SaveAll()
		require.Error(t, err)
	})

	t.Run("handles store save error", func(t *testing.T) {
		store := newMockStore()
		store.saveErr = fmt.Errorf("save error")
		manager := NewManager(store)
		section := &mockSection{id: "test", data: make(map[string]interface{})}
		manager.RegisterSection(section)

		err := manager.SaveAll()
		require.Error(t, err)
	})

	t.Run("saves multiple sections", func(t *testing.T) {
		store := newMockStore()
		manager := NewManager(store)
		section1 := &mockSection{id: "section1", data: map[string]interface{}{"key1": "value1"}}
		section2 := &mockSection{id: "section2", data: map[string]interface{}{"key2": "value2"}}
		manager.RegisterSection(section1)
		manager.RegisterSection(section2)

		err := manager.SaveAll()
		require.NoError(t, err)

		assert.Equal(t, "value1", store.sections["section1"]["key1"])
		assert.Equal(t, "value2", store.sections["section2"]["key2"])
	})
}

func TestManager_ResetAll(t *testing.T) {
	t.Run("resets all sections", func(t *testing.T) {
		manager := NewManager(newMockStore())
		section1 := &mockSection{id: "section1", data: map[string]interface{}{"key1": "value1"}}
		section2 := &mockSection{id: "section2", data: map[string]interface{}{"key2": "value2"}}
		manager.RegisterSection(section1)
		manager.RegisterSection(section2)

		manager.ResetAll()

		assert.Empty(t, section1.data)
		assert.Empty(t, section2.data)
	})

	t.Run("handles empty manager", func(t *testing.T) {
		manager := NewManager(newMockStore())
		// Should not panic
		manager.ResetAll()
	})
}

func TestManager_Store(t *testing.T) {
	store := newMockStore()
	manager := NewManager(store)
	assert.Same(t, store, manager.Store())
}

func TestManager_LiveReload(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "config.json")
	store, err := NewFileStore(tmpFile)
	require.NoError(t, err)

	manager := NewManager(store)
	llmSection := NewLLMSection()
	err = manager.RegisterSection(llmSection)
	require.NoError(t, err)

	// 1. Set initial values and save
	llmSection.SetModel("model-v1")
	llmSection.SetBaseURL("url-v1")
	llmSection.SetAPIKey("key-v1")
	err = manager.SaveAll()
	require.NoError(t, err)

	assert.Equal(t, "model-v1", llmSection.GetModel())

	// 2. Simulate external change by writing directly to the file
	// Note: We are not calling manager.SaveAll() here
	updatedConfigContent := `{
		"version": "1.0",
		"sections": {
			"llm": {
				"model": "model-v2-reloaded",
				"base_url": "url-v2-reloaded",
				"api_key": "key-v2-reloaded"
			}
		}
	}`
	// Add a small delay for systems where file modification time has 1s resolution
	time.Sleep(1 * time.Second)
	err = os.WriteFile(tmpFile, []byte(updatedConfigContent), 0600)
	require.NoError(t, err)

	// 3. Manually trigger a reload, simulating what the file watcher would do
	err = manager.LoadAll()
	require.NoError(t, err)

	// 4. Verify the section is updated with the new values
	assert.Equal(t, "model-v2-reloaded", llmSection.GetModel())
	assert.Equal(t, "url-v2-reloaded", llmSection.GetBaseURL())
	assert.Equal(t, "key-v2-reloaded", llmSection.GetAPIKey())

	// 5. Verify that saving again persists the reloaded values
	err = manager.SaveAll()
	require.NoError(t, err)

	// 6. Create a new manager and load from the same file to confirm persistence
	newStore, err := NewFileStore(tmpFile)
	require.NoError(t, err)
	newManager := NewManager(newStore)
	newLLMSection := NewLLMSection()
	err = newManager.RegisterSection(newLLMSection)
	require.NoError(t, err)
	err = newManager.LoadAll()
	require.NoError(t, err)

	assert.Equal(t, "model-v2-reloaded", newLLMSection.GetModel())
}

func TestManager_Concurrency(t *testing.T) {
	t.Run("concurrent reads are safe", func(t *testing.T) {
		manager := NewManager(newMockStore())
		section := &mockSection{id: "test", title: "Test"}
		manager.RegisterSection(section)

		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				manager.GetSection("test")
				manager.GetSections()
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("concurrent writes are safe", func(t *testing.T) {
		manager := NewManager(newMockStore())

		done := make(chan bool)
		for i := 0; i < 10; i++ {
			i := i
			go func() {
				section := &mockSection{
					id:    fmt.Sprintf("section%d", i),
					title: fmt.Sprintf("Section %d", i),
				}
				manager.RegisterSection(section)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		sections := manager.GetSections()
		assert.Len(t, sections, 10)
	})
}
