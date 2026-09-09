package tui

import (
	"fmt"

	"github.com/entrhq/forge/pkg/agent"
	"github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/llm/factory"
)

// reloadLLMProvider hot-reloads the LLM provider configuration when settings change.
// This method creates a new provider with updated settings and updates the agent.
func (m *model) reloadLLMProvider() error {
	// Get the latest LLM settings from config
	llmConfig := config.GetLLM()
	if llmConfig == nil {
		return fmt.Errorf("LLM configuration not found")
	}

	model := llmConfig.GetModel()
	apiKey := llmConfig.GetAPIKey()
	if model == "" {
		return fmt.Errorf("model cannot be empty")
	}
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	provider, err := factory.NewProvider(factory.Settings{
		Provider:  llmConfig.GetProvider(),
		APIKey:    apiKey,
		Model:     model,
		BaseURL:   llmConfig.GetBaseURL(),
		MaxTokens: llmConfig.GetMaxTokens(),
	})
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// Update the agent's provider (thread-safe hot-reload)
	if err := m.agent.SetProvider(provider); err != nil {
		return fmt.Errorf("failed to update agent provider: %w", err)
	}

	// Update the model's provider reference so settings overlay gets fresh values
	m.provider = provider

	// Apply summarization model override (empty string reverts to main model)
	summarizationModel := llmConfig.GetSummarizationModel()
	if defaultAgent, ok := m.agent.(*agent.DefaultAgent); ok {
		defaultAgent.SetSummarizationModel(summarizationModel)
	}

	return nil
}
