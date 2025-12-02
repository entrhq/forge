package config

import (
	"fmt"
	"sync"
)

const (
	// SectionIDLLM is the identifier for the LLM settings section
	SectionIDLLM = "llm"
)

// LLMSection manages LLM provider configuration settings.
type LLMSection struct {
	Model   string
	BaseURL string
	APIKey  string
	mu      sync.RWMutex
}

// NewLLMSection creates a new LLM section with default settings.
func NewLLMSection() *LLMSection {
	return &LLMSection{
		Model:   "",
		BaseURL: "",
		APIKey:  "",
	}
}

// ID returns the section identifier.
func (s *LLMSection) ID() string {
	return SectionIDLLM
}

// Title returns the section title.
func (s *LLMSection) Title() string {
	return "LLM Settings"
}

// Description returns the section description.
func (s *LLMSection) Description() string {
	return "Configure LLM provider settings including model, base URL, and API key. CLI flags and environment variables take precedence."
}

// Data returns the current configuration data.
func (s *LLMSection) Data() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"model":    s.Model,
		"base_url": s.BaseURL,
		"api_key":  s.APIKey,
	}
}

// SetData updates the configuration from the provided data.
func (s *LLMSection) SetData(data map[string]any) error {
	if data == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if model, ok := data["model"].(string); ok {
		s.Model = model
	}

	if baseURL, ok := data["base_url"].(string); ok {
		s.BaseURL = baseURL
	}

	if apiKey, ok := data["api_key"].(string); ok {
		s.APIKey = apiKey
	}

	return nil
}

// Validate validates the current configuration.
func (s *LLMSection) Validate() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Model == "" {
		return fmt.Errorf("model cannot be empty")
	}

	return nil
}

// Reset resets the section to default configuration.
func (s *LLMSection) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Model = ""
	s.BaseURL = ""
	s.APIKey = ""
}

// GetModel returns the configured model name.
func (s *LLMSection) GetModel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Model
}

// SetModel sets the model name.
func (s *LLMSection) SetModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Model = model
}

// GetBaseURL returns the configured base URL.
func (s *LLMSection) GetBaseURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.BaseURL
}

// SetBaseURL sets the base URL.
func (s *LLMSection) SetBaseURL(baseURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.BaseURL = baseURL
}

// GetAPIKey returns the configured API key.
func (s *LLMSection) GetAPIKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.APIKey
}

// SetAPIKey sets the API key.
func (s *LLMSection) SetAPIKey(apiKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.APIKey = apiKey
}
