package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewFileStore(t *testing.T) {
	t.Run("creates store with custom path", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		store, err := NewFileStore(configPath)
		if err != nil {
			t.Fatalf("NewFileStore failed: %v", err)
		}

		if store.Path() != configPath {
			t.Errorf("Expected path %s, got %s", configPath, store.Path())
		}

		if store.IsModified() {
			t.Error("New store should not be modified")
		}
	})

	t.Run("creates store with default path when empty", func(t *testing.T) {
		store, err := NewFileStore("")
		if err != nil {
			t.Fatalf("NewFileStore with empty path failed: %v", err)
		}

		homeDir, _ := os.UserHomeDir()
		expectedPath := filepath.Join(homeDir, ".forge", "config.json")

		if store.Path() != expectedPath {
			t.Errorf("Expected default path %s, got %s", expectedPath, store.Path())
		}
	})

	t.Run("loads existing config file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		// Create a config file
		config := map[string]interface{}{
			"version": "1.0",
			"sections": map[string]map[string]interface{}{
				"test_section": {
					"key": "value",
				},
			},
		}

		data, _ := json.MarshalIndent(config, "", "  ")
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		store, err := NewFileStore(configPath)
		if err != nil {
			t.Fatalf("NewFileStore failed: %v", err)
		}

		// Verify the data was loaded
		section, err := store.GetSection("test_section")
		if err != nil {
			t.Fatalf("GetSection failed: %v", err)
		}

		if section["key"] != "value" {
			t.Errorf("Expected key=value, got %v", section["key"])
		}
	})
}

func TestFileStore_Load(t *testing.T) {
	t.Run("loads valid config file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		// Create a config file
		config := map[string]interface{}{
			"version": "1.0",
			"sections": map[string]map[string]interface{}{
				"section1": {"key1": "value1"},
				"section2": {"key2": "value2"},
			},
		}

		data, _ := json.MarshalIndent(config, "", "  ")
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		store := &FileStore{path: configPath}
		if err := store.Load(); err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		// Verify both sections loaded
		section1, _ := store.GetSection("section1")
		section2, _ := store.GetSection("section2")

		if section1["key1"] != "value1" {
			t.Error("Section1 not loaded correctly")
		}
		if section2["key2"] != "value2" {
			t.Error("Section2 not loaded correctly")
		}
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "nonexistent.json")

		store := &FileStore{path: configPath}
		if err := store.Load(); err != nil {
			t.Fatalf("Load should not fail for non-existent file: %v", err)
		}

		// Should have empty data
		all, _ := store.GetAll()
		if len(all) != 0 {
			t.Error("Expected empty config for non-existent file")
		}
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "invalid.json")

		// Write invalid JSON
		if err := os.WriteFile(configPath, []byte("{invalid json}"), 0644); err != nil {
			t.Fatalf("Failed to write invalid JSON: %v", err)
		}

		store := &FileStore{path: configPath}
		if err := store.Load(); err == nil {
			t.Error("Load should fail for invalid JSON")
		}
	})
}

func TestFileStore_Save(t *testing.T) {
	t.Run("saves config to file", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		store, _ := NewFileStore(configPath)

		// Set some data
		testData := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}
		if err := store.SetSection("test_section", testData); err != nil {
			t.Fatalf("SetSection failed: %v", err)
		}

		// Save
		if err := store.Save(); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		// Verify file exists and is valid JSON
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read saved config: %v", err)
		}

		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err != nil {
			t.Fatalf("Saved config is not valid JSON: %v", err)
		}

		// Verify structure
		if config["version"] != "1.0" {
			t.Error("Version not saved correctly")
		}

		sections, ok := config["sections"].(map[string]interface{})
		if !ok {
			t.Fatal("Sections not saved correctly")
		}

		testSection, ok := sections["test_section"].(map[string]interface{})
		if !ok {
			t.Fatal("Test section not found")
		}

		if testSection["key1"] != "value1" {
			t.Error("Data not saved correctly")
		}
	})

	t.Run("creates directory if needed", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "nested", "dir", "config.json")

		store, _ := NewFileStore(configPath)
		store.SetSection("test", map[string]interface{}{"key": "value"})

		if err := store.Save(); err != nil {
			t.Fatalf("Save should create nested directories: %v", err)
		}

		// Verify directory exists
		if _, err := os.Stat(filepath.Dir(configPath)); os.IsNotExist(err) {
			t.Error("Directory was not created")
		}
	})

	t.Run("clears modified flag after save", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		store, _ := NewFileStore(configPath)
		store.SetSection("test", map[string]interface{}{"key": "value"})

		if !store.IsModified() {
			t.Error("Store should be modified after SetSection")
		}

		if err := store.Save(); err != nil {
			t.Fatalf("Save failed: %v", err)
		}

		if store.IsModified() {
			t.Error("Store should not be modified after Save")
		}
	})
}

func TestFileStore_GetSection(t *testing.T) {
	t.Run("returns existing section", func(t *testing.T) {
		store := &FileStore{
			data: map[string]map[string]interface{}{
				"test": {
					"key1": "value1",
					"key2": 42,
				},
			},
		}

		section, err := store.GetSection("test")
		if err != nil {
			t.Fatalf("GetSection failed: %v", err)
		}

		if section["key1"] != "value1" {
			t.Error("Section data not returned correctly")
		}
	})

	t.Run("returns empty map for non-existent section", func(t *testing.T) {
		store := &FileStore{
			data: make(map[string]map[string]interface{}),
		}

		section, err := store.GetSection("nonexistent")
		if err != nil {
			t.Fatalf("GetSection failed: %v", err)
		}

		if len(section) != 0 {
			t.Error("Expected empty map for non-existent section")
		}
	})

	t.Run("returns copy to prevent external modification", func(t *testing.T) {
		store := &FileStore{
			data: map[string]map[string]interface{}{
				"test": {"key": "value"},
			},
		}

		section1, _ := store.GetSection("test")
		section1["key"] = "modified"

		section2, _ := store.GetSection("test")
		if section2["key"] == "modified" {
			t.Error("External modification affected store data")
		}
	})
}

func TestFileStore_SetSection(t *testing.T) {
	t.Run("sets section data", func(t *testing.T) {
		store := &FileStore{
			data: make(map[string]map[string]interface{}),
		}

		testData := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}

		if err := store.SetSection("test", testData); err != nil {
			t.Fatalf("SetSection failed: %v", err)
		}

		section, _ := store.GetSection("test")
		if section["key1"] != "value1" || section["key2"] != 42 {
			t.Error("Section data not set correctly")
		}
	})

	t.Run("sets modified flag", func(t *testing.T) {
		store := &FileStore{
			data:     make(map[string]map[string]interface{}),
			modified: false,
		}

		store.SetSection("test", map[string]interface{}{"key": "value"})

		if !store.IsModified() {
			t.Error("Modified flag should be set")
		}
	})

	t.Run("stores copy to prevent external modification", func(t *testing.T) {
		store := &FileStore{
			data: make(map[string]map[string]interface{}),
		}

		testData := map[string]interface{}{"key": "value"}
		store.SetSection("test", testData)

		// Modify original
		testData["key"] = "modified"

		// Check store wasn't affected
		section, _ := store.GetSection("test")
		if section["key"] == "modified" {
			t.Error("External modification affected store data")
		}
	})
}

func TestFileStore_GetAll(t *testing.T) {
	t.Run("returns all sections", func(t *testing.T) {
		store := &FileStore{
			data: map[string]map[string]interface{}{
				"section1": {"key1": "value1"},
				"section2": {"key2": "value2"},
			},
		}

		all, err := store.GetAll()
		if err != nil {
			t.Fatalf("GetAll failed: %v", err)
		}

		if len(all) != 2 {
			t.Errorf("Expected 2 sections, got %d", len(all))
		}

		if all["section1"]["key1"] != "value1" {
			t.Error("Section1 data incorrect")
		}
		if all["section2"]["key2"] != "value2" {
			t.Error("Section2 data incorrect")
		}
	})

	t.Run("returns deep copy", func(t *testing.T) {
		store := &FileStore{
			data: map[string]map[string]interface{}{
				"test": {"key": "value"},
			},
		}

		all, _ := store.GetAll()
		all["test"]["key"] = "modified"

		// Verify store wasn't affected
		section, _ := store.GetSection("test")
		if section["key"] == "modified" {
			t.Error("External modification affected store data")
		}
	})
}

func TestFileStore_SetAll(t *testing.T) {
	t.Run("sets all sections", func(t *testing.T) {
		store := &FileStore{
			data: make(map[string]map[string]interface{}),
		}

		allData := map[string]map[string]interface{}{
			"section1": {"key1": "value1"},
			"section2": {"key2": "value2"},
		}

		if err := store.SetAll(allData); err != nil {
			t.Fatalf("SetAll failed: %v", err)
		}

		all, _ := store.GetAll()
		if len(all) != 2 {
			t.Error("Not all sections were set")
		}
	})

	t.Run("stores deep copy", func(t *testing.T) {
		store := &FileStore{
			data: make(map[string]map[string]interface{}),
		}

		allData := map[string]map[string]interface{}{
			"test": {"key": "value"},
		}

		store.SetAll(allData)

		// Modify original
		allData["test"]["key"] = "modified"

		// Verify store wasn't affected
		section, _ := store.GetSection("test")
		if section["key"] == "modified" {
			t.Error("External modification affected store data")
		}
	})
}
