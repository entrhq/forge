package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store provides persistence for configuration data.
type Store interface {
	// Load loads the configuration from disk
	Load() error

	// Save saves the configuration to disk
	Save() error

	// GetSection retrieves configuration data for a specific section
	GetSection(sectionID string) (map[string]interface{}, error)

	// SetSection stores configuration data for a specific section
	SetSection(sectionID string, data map[string]interface{}) error

	// GetAll retrieves all configuration data
	GetAll() (map[string]map[string]interface{}, error)

	// SetAll stores all configuration data
	SetAll(data map[string]map[string]interface{}) error
}

// FileStore implements Store using a JSON file.
type FileStore struct {
	path     string
	data     map[string]map[string]interface{}
	mu       sync.RWMutex
	version  string
	modified bool
}

// NewFileStore creates a new file-based configuration store.
// If path is empty, defaults to ~/.forge/config.json
func NewFileStore(path string) (*FileStore, error) {
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		path = filepath.Join(homeDir, ".forge", "config.json")
	}

	store := &FileStore{
		path:    path,
		data:    make(map[string]map[string]interface{}),
		version: "1.0",
	}

	// Try to load existing config, but don't fail if it doesn't exist
	if err := store.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load config from %s: %w", path, err)
	}

	return store, nil
}

// Load loads the configuration from disk.
func (s *FileStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, use empty config
			s.data = make(map[string]map[string]interface{})
			return nil
		}
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config struct {
		Version  string                            `json:"version"`
		Sections map[string]map[string]interface{} `json:"sections"`
	}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return fmt.Errorf("failed to decode config file: %w", err)
	}

	s.version = config.Version
	if config.Sections != nil {
		s.data = config.Sections
	} else {
		s.data = make(map[string]map[string]interface{})
	}
	s.modified = false

	return nil
}

// Save saves the configuration to disk.
func (s *FileStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create temp file for atomic write
	tempPath := s.path + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp config file: %w", err)
	}

	config := struct {
		Version  string                            `json:"version"`
		Sections map[string]map[string]interface{} `json:"sections"`
	}{
		Version:  s.version,
		Sections: s.data,
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		file.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to encode config: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, s.path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	s.modified = false
	return nil
}

// GetSection retrieves configuration data for a specific section.
func (s *FileStore) GetSection(sectionID string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if data, exists := s.data[sectionID]; exists {
		// Return a copy to prevent external modification
		dataCopy := make(map[string]interface{}, len(data))
		for k, v := range data {
			dataCopy[k] = v
		}
		return dataCopy, nil
	}

	// Return empty map if section doesn't exist
	return make(map[string]interface{}), nil
}

// SetSection stores configuration data for a specific section.
func (s *FileStore) SetSection(sectionID string, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store a copy to prevent external modification
	dataCopy := make(map[string]interface{}, len(data))
	for k, v := range data {
		dataCopy[k] = v
	}

	s.data[sectionID] = dataCopy
	s.modified = true
	return nil
}

// GetAll retrieves all configuration data.
func (s *FileStore) GetAll() (map[string]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a deep copy
	dataCopy := make(map[string]map[string]interface{}, len(s.data))
	for sectionID, sectionData := range s.data {
		sectionCopy := make(map[string]interface{}, len(sectionData))
		for k, v := range sectionData {
			sectionCopy[k] = v
		}
		dataCopy[sectionID] = sectionCopy
	}

	return dataCopy, nil
}

// SetAll stores all configuration data.
func (s *FileStore) SetAll(data map[string]map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Store a deep copy
	dataCopy := make(map[string]map[string]interface{}, len(data))
	for sectionID, sectionData := range data {
		sectionCopy := make(map[string]interface{}, len(sectionData))
		for k, v := range sectionData {
			sectionCopy[k] = v
		}
		dataCopy[sectionID] = sectionCopy
	}

	s.data = dataCopy
	s.modified = true
	return nil
}

// IsModified returns true if the store has unsaved changes.
func (s *FileStore) IsModified() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modified
}

// Path returns the file path of the store.
func (s *FileStore) Path() string {
	return s.path
}
