package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/entrhq/forge/internal/testing/configtest"
	"github.com/entrhq/forge/pkg/llm/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CliConfig represents the command-line flags that can be passed.
// It's a simplified version of the main.Config for testing purposes.
type CliConfig struct {
	Model   string
	BaseURL string
	APIKey  string
}

const defaultTestModel = "test-default-model"

func TestBuildProviderWithNoConfig(t *testing.T) {
	cliConfig := &CliConfig{Model: "cli-model", BaseURL: "https://cli.url", APIKey: "cli-key"}

	// No config file is written, and Initialize is not called.

	provider, err := BuildProvider(cliConfig.Model, cliConfig.BaseURL, cliConfig.APIKey, defaultTestModel)
	require.NoError(t, err)
	require.NotNil(t, provider)

	assert.Equal(t, "cli-model", provider.GetModel())
	assert.Equal(t, "https://cli.url", provider.GetBaseURL())
	if p, ok := provider.(*openai.OpenAIProvider); ok {
		assert.Equal(t, "cli-key", p.GetAPIKey())
	}

	ResetGlobalManager()
}

func TestBuildProvider(t *testing.T) {
	testCases := []struct {
		name            string
		cliConfig       *CliConfig
		fileContent     string
		expectedModel   string
		expectedBaseURL string
		expectedAPIKey  string
		expectError     bool
	}{
		{
			name:            "CLI flags only",
			cliConfig:       &CliConfig{Model: "cli-model", BaseURL: "https://cli.url", APIKey: "cli-key"},
			fileContent:     `{}`,
			expectedModel:   "cli-model",
			expectedBaseURL: "https://cli.url",
			expectedAPIKey:  "cli-key",
		},
		{
			name:            "Config file only",
			cliConfig:       &CliConfig{Model: defaultTestModel, BaseURL: "", APIKey: ""},
			fileContent:     `{"version":"1.0","sections":{"llm":{"model":"file-model","base_url":"https://file.url","api_key":"file-key"}}}`,
			expectedModel:   "file-model",
			expectedBaseURL: "https://file.url",
			expectedAPIKey:  "file-key",
		},
		{
			name:            "CLI overrides config file",
			cliConfig:       &CliConfig{Model: "cli-model", BaseURL: "https://cli.url", APIKey: "cli-key"},
			fileContent:     `{"version":"1.0","sections":{"llm":{"model":"file-model","base_url":"https://file.url","api_key":"file-key"}}}`,
			expectedModel:   "cli-model",
			expectedBaseURL: "https://cli.url",
			expectedAPIKey:  "cli-key",
		},
		{
			name:            "Partial CLI override (model only)",
			cliConfig:       &CliConfig{Model: "cli-model", BaseURL: "", APIKey: ""},
			fileContent:     `{"version":"1.0","sections":{"llm":{"model":"file-model","base_url":"https://file.url","api_key":"file-key"}}}`,
			expectedModel:   "cli-model",
			expectedBaseURL: "https://file.url",
			expectedAPIKey:  "file-key",
		},
		{
			name:            "Partial CLI override (URL and key)",
			cliConfig:       &CliConfig{Model: "", BaseURL: "https://cli.url", APIKey: "cli-key"},
			fileContent:     `{"version":"1.0","sections":{"llm":{"model":"file-model","base_url":"https://file.url","api_key":"file-key"}}}`,
			expectedModel:   "file-model",
			expectedBaseURL: "https://cli.url",
			expectedAPIKey:  "cli-key",
		},
		{
			name:            "Default model with file values",
			cliConfig:       &CliConfig{Model: "", BaseURL: "", APIKey: ""},
			fileContent:     `{"version":"1.0","sections":{"llm":{"base_url":"https://file.url","api_key":"file-key"}}}`,
			expectedModel:   defaultTestModel,
			expectedBaseURL: "https://file.url",
			expectedAPIKey:  "file-key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.json")
			err := os.WriteFile(configPath, []byte(tc.fileContent), 0600)
			require.NoError(t, err)

			err = Initialize(configPath)
			require.NoError(t, err)

			provider, err := BuildProvider(tc.cliConfig.Model, tc.cliConfig.BaseURL, tc.cliConfig.APIKey, defaultTestModel)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, provider)
				assert.Equal(t, tc.expectedModel, provider.GetModel())
				assert.Equal(t, tc.expectedBaseURL, provider.GetBaseURL())
				// We can't test the API key directly, but we can verify it was set.
				// A proper integration test would make a real API call.
				if provider, ok := provider.(*openai.OpenAIProvider); ok {
					assert.Equal(t, tc.expectedAPIKey, provider.GetAPIKey())
				}
			}

			ResetGlobalManager()
		})
	}
}

func TestBuildProviderWithEnvVar(t *testing.T) {
	// Set environment variables for LLM configuration
	t.Setenv("FORGE_LLM_MODEL", "env-model")
	t.Setenv("FORGE_LLM_BASE_URL", "https://env.url")
	t.Setenv("FORGE_LLM_API_KEY", "env-key")

	// Use a test helper to manage config initialization and cleanup
	configtest.WithGlobalManager(t, func() {
		provider, err := BuildProvider("", "", "", defaultTestModel)
		require.NoError(t, err)
		require.NotNil(t, provider)

		assert.Equal(t, "env-model", provider.GetModel())
		assert.Equal(t, "https://env.url", provider.GetBaseURL())
		if p, ok := provider.(*openai.OpenAIProvider); ok {
			assert.Equal(t, "env-key", p.GetAPIKey())
		}
	})
}
