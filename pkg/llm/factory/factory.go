// Package factory constructs the configured llm.Provider.
//
// It is the single place that knows which concrete provider packages exist.
// Callers pass resolved settings and get back an llm.Provider without
// importing pkg/llm/openai or pkg/llm/anthropic themselves.
package factory

import (
	"fmt"
	"os"

	"github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/llm/anthropic"
	"github.com/entrhq/forge/pkg/llm/openai"
)

// Settings are the resolved values a provider is built from.
type Settings struct {
	Provider  string // config.ProviderOpenAI or config.ProviderAnthropic
	APIKey    string
	Model     string
	BaseURL   string // empty keeps the provider's default endpoint
	MaxTokens int    // response ceiling; 0 means the provider default. Only the Anthropic provider sends max_tokens.
}

// NewProvider constructs the provider named by s.Provider.
func NewProvider(s Settings) (llm.Provider, error) {
	switch s.Provider {
	case config.ProviderOpenAI:
		opts := []openai.ProviderOption{openai.WithModel(s.Model)}
		if s.BaseURL != "" {
			opts = append(opts, openai.WithBaseURL(s.BaseURL))
		}
		return openai.NewProvider(s.APIKey, opts...)
	case config.ProviderAnthropic:
		opts := []anthropic.ProviderOption{anthropic.WithModel(s.Model), anthropic.WithMaxTokens(s.MaxTokens)}
		if s.BaseURL != "" {
			opts = append(opts, anthropic.WithBaseURL(s.BaseURL))
		}
		return anthropic.NewProvider(s.APIKey, opts...)
	default:
		return nil, fmt.Errorf("unknown llm provider %q", s.Provider)
	}
}

// Provider-neutral environment variables, read after the provider-specific
// ones. They let one credential serve an endpoint that speaks both APIs.
const (
	envForgeAPIKey  = "FORGE_API_KEY" //nolint:gosec // G101: this is the variable's name, not a credential
	envForgeBaseURL = "FORGE_BASE_URL"
)

// BuildProvider resolves LLM settings with the precedence
// CLI flags > provider environment variables > FORGE_* environment variables >
// config file > defaults, then constructs the configured provider.
//
// The provider type and max_tokens come from the config file only. The
// provider type selects which environment variables are consulted first:
// OPENAI_API_KEY/OPENAI_BASE_URL for the OpenAI provider,
// ANTHROPIC_API_KEY/ANTHROPIC_BASE_URL for Anthropic. FORGE_API_KEY and
// FORGE_BASE_URL apply to either.
func BuildProvider(cliModel, cliBaseURL, cliAPIKey, defaultModel string) (llm.Provider, error) {
	s := Settings{
		Provider: config.ProviderOpenAI,
		Model:    cliModel,
		BaseURL:  cliBaseURL,
		APIKey:   cliAPIKey,
	}
	llmConfig := config.GetLLM()
	if llmConfig != nil {
		s.Provider = llmConfig.GetProvider()
		s.MaxTokens = llmConfig.GetMaxTokens()
	}
	envAPIKey, envBaseURL := envVarNames(s.Provider)

	if s.APIKey == "" {
		s.APIKey = firstEnv(envAPIKey, envForgeAPIKey)
	}
	if s.BaseURL == "" {
		s.BaseURL = firstEnv(envBaseURL, envForgeBaseURL)
	}

	if llmConfig != nil {
		// The default model counts as "not set" so the config file can override it.
		if cliModel == "" || cliModel == defaultModel {
			if configModel := llmConfig.GetModel(); configModel != "" {
				s.Model = configModel
			}
		}
		if s.BaseURL == "" {
			s.BaseURL = llmConfig.GetBaseURL()
		}
		if s.APIKey == "" {
			s.APIKey = llmConfig.GetAPIKey()
		}
	}

	if s.Model == "" {
		s.Model = defaultModel
	}
	if s.APIKey == "" {
		return nil, fmt.Errorf("API key is required. Set %s or %s environment variable, use -api-key flag, or configure in ~/.forge/config.json", envAPIKey, envForgeAPIKey)
	}

	provider, err := NewProvider(s)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}
	return provider, nil
}

// envVarNames returns the API key and base URL environment variable names for a provider type.
func envVarNames(providerType string) (apiKey, baseURL string) {
	if providerType == config.ProviderAnthropic {
		return "ANTHROPIC_API_KEY", "ANTHROPIC_BASE_URL"
	}
	return "OPENAI_API_KEY", "OPENAI_BASE_URL"
}

// firstEnv returns the value of the first named environment variable that is set.
func firstEnv(names ...string) string {
	for _, name := range names {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	return ""
}
