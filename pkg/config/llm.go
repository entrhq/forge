package config

import (
	"sync"
)

const (
	// SectionIDLLM is the identifier for the LLM settings section
	SectionIDLLM = "llm"
)

// LLMSection manages LLM provider configuration settings.
type LLMSection struct {
	Model              string
	BaseURL            string
	APIKey             string
	SummarizationModel string // optional; if empty, summarization uses Model
	mu                 sync.RWMutex
}

// NewLLMSection creates a new LLM section with default settings.
func NewLLMSection() *LLMSection {
	return &LLMSection{
		Model:              "",
		BaseURL:            "",
		APIKey:             "",
		SummarizationModel: "",
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
	return "Configure LLM provider settings. summarization_model is optional — if set, context summarization uses this model instead of model."
}

// Data returns the current configuration data.
func (s *LLMSection) Data() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"model":               s.Model,
		"base_url":            s.BaseURL,
		"api_key":             s.APIKey,
		"summarization_model": s.SummarizationModel,
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

	if summarizationModel, ok := data["summarization_model"].(string); ok {
		s.SummarizationModel = summarizationModel
	}

	return nil
}

// Validate validates the current configuration.
func (s *LLMSection) Validate() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// LLM configuration is optional - validation always passes
	// Actual validation happens at runtime when LLM is used
	return nil
}

// Reset resets the section to default configuration.
func (s *LLMSection) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Model = ""
	s.BaseURL = ""
	s.APIKey = ""
	s.SummarizationModel = ""
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

// GetSummarizationModel returns the configured summarization model name.
// An empty string means use the main model for summarization.
func (s *LLMSection) GetSummarizationModel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.SummarizationModel
}

// SetSummarizationModel sets the summarization model name.
// Pass an empty string to revert to using the main model.
func (s *LLMSection) SetSummarizationModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SummarizationModel = model
}
