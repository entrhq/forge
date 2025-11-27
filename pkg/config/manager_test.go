package config

import (
	"fmt"
	"testing"
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

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.Store() != store {
		t.Error("Manager does not reference correct store")
	}

	sections := manager.GetSections()
	if len(sections) != 0 {
		t.Error("New manager should have no sections")
	}
}

func TestManager_RegisterSection(t *testing.T) {
	t.Run("registers section successfully", func(t *testing.T) {
		manager := NewManager(newMockStore())
		section := &mockSection{id: "test", title: "Test"}

		err := manager.RegisterSection(section)
		if err != nil {
			t.Fatalf("RegisterSection failed: %v", err)
		}

		retrieved, ok := manager.GetSection("test")
		if !ok {
			t.Fatal("Section not found after registration")
		}

		if retrieved.ID() != "test" {
			t.Error("Retrieved section has wrong ID")
		}
	})

	t.Run("prevents duplicate registration", func(t *testing.T) {
		manager := NewManager(newMockStore())
		section1 := &mockSection{id: "test", title: "Test 1"}
		section2 := &mockSection{id: "test", title: "Test 2"}

		if err := manager.RegisterSection(section1); err != nil {
			t.Fatalf("First registration failed: %v", err)
		}

		err := manager.RegisterSection(section2)
		if err == nil {
			t.Error("Expected error for duplicate registration")
		}
	})

	t.Run("maintains registration order", func(t *testing.T) {
		manager := NewManager(newMockStore())

		section1 := &mockSection{id: "first", title: "First"}
		section2 := &mockSection{id: "second", title: "Second"}
		section3 := &mockSection{id: "third", title: "Third"}

		manager.RegisterSection(section1)
		manager.RegisterSection(section2)
		manager.RegisterSection(section3)

		sections := manager.GetSections()
		if len(sections) != 3 {
			t.Fatalf("Expected 3 sections, got %d", len(sections))
		}

		if sections[0].ID() != "first" || sections[1].ID() != "second" || sections[2].ID() != "third" {
			t.Error("Sections not in registration order")
		}
	})
}

func TestManager_GetSection(t *testing.T) {
	t.Run("returns existing section", func(t *testing.T) {
		manager := NewManager(newMockStore())
		section := &mockSection{id: "test", title: "Test"}
		manager.RegisterSection(section)

		retrieved, ok := manager.GetSection("test")
		if !ok {
			t.Fatal("Section not found")
		}

		if retrieved.ID() != "test" {
			t.Error("Wrong section returned")
		}
	})

	t.Run("returns false for non-existent section", func(t *testing.T) {
		manager := NewManager(newMockStore())

		_, ok := manager.GetSection("nonexistent")
		if ok {
			t.Error("Should return false for non-existent section")
		}
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
		if len(sections) != 2 {
			t.Fatalf("Expected 2 sections, got %d", len(sections))
		}

		if sections[0].ID() != "a" || sections[1].ID() != "b" {
			t.Error("Sections not returned in correct order")
		}
	})

	t.Run("returns empty slice for no sections", func(t *testing.T) {
		manager := NewManager(newMockStore())

		sections := manager.GetSections()
		if len(sections) != 0 {
			t.Error("Expected empty slice")
		}
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

		if err := manager.LoadAll(); err != nil {
			t.Fatalf("LoadAll failed: %v", err)
		}

		if section.data["key"] != "value" {
			t.Error("Section data not loaded correctly")
		}
	})

	t.Run("handles store load error", func(t *testing.T) {
		store := newMockStore()
		store.loadErr = fmt.Errorf("load error")

		manager := NewManager(store)

		err := manager.LoadAll()
		if err == nil {
			t.Error("Expected error from store")
		}
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

		if err := manager.LoadAll(); err != nil {
			t.Fatalf("LoadAll failed: %v", err)
		}

		if section1.data["key1"] != "value1" {
			t.Error("Section1 not loaded correctly")
		}
		if section2.data["key2"] != "value2" {
			t.Error("Section2 not loaded correctly")
		}
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

		if err := manager.SaveAll(); err != nil {
			t.Fatalf("SaveAll failed: %v", err)
		}

		savedData := store.sections["test"]
		if savedData["key"] != "value" {
			t.Error("Section data not saved correctly")
		}
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
		if err == nil {
			t.Error("Expected validation error")
		}
	})

	t.Run("handles store save error", func(t *testing.T) {
		store := newMockStore()
		store.saveErr = fmt.Errorf("save error")

		manager := NewManager(store)
		section := &mockSection{id: "test", data: make(map[string]interface{})}
		manager.RegisterSection(section)

		err := manager.SaveAll()
		if err == nil {
			t.Error("Expected error from store")
		}
	})

	t.Run("saves multiple sections", func(t *testing.T) {
		store := newMockStore()
		manager := NewManager(store)

		section1 := &mockSection{id: "section1", data: map[string]interface{}{"key1": "value1"}}
		section2 := &mockSection{id: "section2", data: map[string]interface{}{"key2": "value2"}}

		manager.RegisterSection(section1)
		manager.RegisterSection(section2)

		if err := manager.SaveAll(); err != nil {
			t.Fatalf("SaveAll failed: %v", err)
		}

		if store.sections["section1"]["key1"] != "value1" {
			t.Error("Section1 not saved correctly")
		}
		if store.sections["section2"]["key2"] != "value2" {
			t.Error("Section2 not saved correctly")
		}
	})
}

func TestManager_ResetAll(t *testing.T) {
	t.Run("resets all sections", func(t *testing.T) {
		manager := NewManager(newMockStore())

		section1 := &mockSection{
			id:   "section1",
			data: map[string]interface{}{"key1": "value1"},
		}
		section2 := &mockSection{
			id:   "section2",
			data: map[string]interface{}{"key2": "value2"},
		}

		manager.RegisterSection(section1)
		manager.RegisterSection(section2)

		manager.ResetAll()

		if len(section1.data) != 0 {
			t.Error("Section1 not reset")
		}
		if len(section2.data) != 0 {
			t.Error("Section2 not reset")
		}
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

	if manager.Store() != store {
		t.Error("Store() returned wrong store")
	}
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
		if len(sections) != 10 {
			t.Errorf("Expected 10 sections, got %d", len(sections))
		}
	})
}
