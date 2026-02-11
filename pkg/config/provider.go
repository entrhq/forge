package config

import (
	"fmt"
	"github.com/entrhq/forge/pkg/llm/openai"
	"os"
)

// BuildProvider creates an LLM provider based on configuration precedence:
// CLI flags > Environment variables > Config file > Defaults
func BuildProvider(cliModel, cliBaseURL, cliAPIKey, defaultModel string) (*openai.Provider, error) {
	// Start with CLI values (empty strings if not provided)
	finalModel := cliModel
	finalBaseURL := cliBaseURL
	finalAPIKey := cliAPIKey

	// Fall back to environment variables if CLI values are empty
	if finalAPIKey == "" {
		finalAPIKey = os.Getenv("OPENAI_API_KEY")
	}
	if finalBaseURL == "" {
		finalBaseURL = os.Getenv("OPENAI_BASE_URL")
	}

	// Get config file settings
	llmConfigFromFile := GetLLM()

	// Fall back to config file if still empty
	if llmConfigFromFile != nil {
		// Model: Use config file only if CLI didn't set a non-default value
		if cliModel == "" || cliModel == defaultModel {
			if configFileModel := llmConfigFromFile.GetModel(); configFileModel != "" {
				finalModel = configFileModel
			}
		}
		// BaseURL: Use config file if still empty after env check
		if finalBaseURL == "" {
			if configFileBaseURL := llmConfigFromFile.GetBaseURL(); configFileBaseURL != "" {
				finalBaseURL = configFileBaseURL
			}
		}
		// APIKey: Use config file if still empty after env check
		if finalAPIKey == "" {
			if configFileAPIKey := llmConfigFromFile.GetAPIKey(); configFileAPIKey != "" {
				finalAPIKey = configFileAPIKey
			}
		}
	}

	// Use default model if still not set
	if finalModel == "" {
		finalModel = defaultModel
	}

	// Validate that API key was resolved
	if finalAPIKey == "" {
		return nil, fmt.Errorf("API key is required. Set OPENAI_API_KEY environment variable, use -api-key flag, or configure in ~/.config/forge/config.yaml")
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
