package config

import (
	"fmt"
	"sync"
)

const (
	// SectionIDLLM is the identifier for the LLM settings section
	SectionIDLLM = "llm"

	// ProviderOpenAI selects the OpenAI-compatible Chat Completions provider.
	// This is the default and covers any OpenAI-compatible proxy such as OpenRouter.
	ProviderOpenAI = "openai"

	// ProviderAnthropic selects the native Anthropic Messages API provider.
	ProviderAnthropic = "anthropic"
)

// LLMSection manages LLM provider configuration settings.
type LLMSection struct {
	Provider             string // optional; empty resolves to ProviderOpenAI
	Model                string
	BaseURL              string
	APIKey               string
	SummarizationModel   string // optional; if empty, summarization uses Model
	BrowserAnalysisModel string // optional; if empty, browser page analysis uses Model
	MaxTokens            int    // optional; response token ceiling, 0 means the provider default
	mu                   sync.RWMutex
}

// NewLLMSection creates a new LLM section with default settings.
func NewLLMSection() *LLMSection {
	return &LLMSection{}
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
	return "Configure LLM provider settings. provider is \"openai\" (default, any OpenAI-compatible API) or \"anthropic\" (native Anthropic Messages API). summarization_model and browser_analysis_model are optional — if set, those operations use the specified model instead of the main model."
}

// Data returns the current configuration data.
func (s *LLMSection) Data() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"provider":               s.Provider,
		"model":                  s.Model,
		"base_url":               s.BaseURL,
		"api_key":                s.APIKey,
		"summarization_model":    s.SummarizationModel,
		"browser_analysis_model": s.BrowserAnalysisModel,
		"max_tokens":             s.MaxTokens,
	}
}

// SetData updates the configuration from the provided data.
func (s *LLMSection) SetData(data map[string]any) error {
	if data == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if provider, ok := data["provider"].(string); ok {
		s.Provider = provider
	}

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

	if browserAnalysisModel, ok := data["browser_analysis_model"].(string); ok {
		s.BrowserAnalysisModel = browserAnalysisModel
	}

	if maxTokens, ok := intFromAny(data["max_tokens"]); ok {
		s.MaxTokens = maxTokens
	}

	return nil
}

// Validate validates the current configuration.
// Only the provider name is checked here; credentials and model are validated
// at runtime when the provider is built, since LLM configuration is optional.
func (s *LLMSection) Validate() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	switch s.Provider {
	case "", ProviderOpenAI, ProviderAnthropic:
	default:
		return fmt.Errorf("unknown llm provider %q (expected %q or %q)", s.Provider, ProviderOpenAI, ProviderAnthropic)
	}
	if s.MaxTokens < 0 {
		return fmt.Errorf("max_tokens must be 0 (provider default) or positive, got %d", s.MaxTokens)
	}
	return nil
}

// Reset resets the section to default configuration.
func (s *LLMSection) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Provider = ""
	s.Model = ""
	s.BaseURL = ""
	s.APIKey = ""
	s.SummarizationModel = ""
	s.BrowserAnalysisModel = ""
	s.MaxTokens = 0
}

// GetMaxTokens returns the configured response token ceiling. Zero means use the provider default.
func (s *LLMSection) GetMaxTokens() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.MaxTokens
}

// SetMaxTokens sets the response token ceiling. Pass 0 to revert to the provider default.
func (s *LLMSection) SetMaxTokens(maxTokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MaxTokens = maxTokens
}

// GetProvider returns the configured provider name, resolving empty to ProviderOpenAI.
func (s *LLMSection) GetProvider() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.Provider == "" {
		return ProviderOpenAI
	}
	return s.Provider
}

// SetProvider sets the provider name.
func (s *LLMSection) SetProvider(provider string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Provider = provider
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

// GetBrowserAnalysisModel returns the configured browser analysis model name.
// An empty string means use the main model for browser page analysis.
func (s *LLMSection) GetBrowserAnalysisModel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.BrowserAnalysisModel
}

// SetBrowserAnalysisModel sets the browser analysis model name.
// Pass an empty string to revert to using the main model.
func (s *LLMSection) SetBrowserAnalysisModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.BrowserAnalysisModel = model
}
