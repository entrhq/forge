package config

import (
	"fmt"
	"github.com/entrhq/forge/pkg/llm/openai"
)

// BuildProvider creates an LLM provider based on CLI flags and config file settings.
// CLI flags take precedence over config file settings.
func BuildProvider(cliModel, cliBaseURL, cliAPIKey, defaultModel string) (*openai.Provider, error) {
	// Start with CLI values
	finalModel := cliModel
	finalBaseURL := cliBaseURL
	finalAPIKey := cliAPIKey

	// Get config file settings
	llmConfigFromFile := GetLLM()

	// If a CLI flag was not set to a non-default/empty value, consider the config file value.
	if llmConfigFromFile != nil {
		// Model: CLI default is not empty, so we check if the CLI value is the default.
		if cliModel == defaultModel {
			if configFileModel := llmConfigFromFile.GetModel(); configFileModel != "" {
				finalModel = configFileModel
			}
		}
		// BaseURL: CLI default is empty string.
		if cliBaseURL == "" {
			if configFileBaseURL := llmConfigFromFile.GetBaseURL(); configFileBaseURL != "" {
				finalBaseURL = configFileBaseURL
			}
		}
		// APIKey: CLI default is empty string.
		if cliAPIKey == "" {
			if configFileAPIKey := llmConfigFromFile.GetAPIKey(); configFileAPIKey != "" {
				finalAPIKey = configFileAPIKey
			}
		}
	}

	// Create OpenAI provider with the final, resolved configuration
	providerOpts := []openai.ProviderOption{
		openai.WithModel(finalModel),
	}
	if finalBaseURL != "" {
		providerOpts = append(providerOpts, openai.WithBaseURL(finalBaseURL))
	}

	provider, err := openai.NewProvider(finalAPIKey, providerOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM provider: %w", err)
	}

	return provider, nil
}
