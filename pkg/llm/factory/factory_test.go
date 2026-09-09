package factory

import (
	"os"
	"testing"

	"github.com/entrhq/forge/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		providerType string
		wantProvider string
		wantBaseURL  string
	}{
		{config.ProviderOpenAI, "openai", "https://example.com/v1"},
		{config.ProviderAnthropic, "anthropic", "https://example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.providerType, func(t *testing.T) {
			p, err := NewProvider(Settings{Provider: tt.providerType, APIKey: "key", Model: "model-x", BaseURL: tt.wantBaseURL})
			require.NoError(t, err)
			assert.Equal(t, tt.wantProvider, p.GetModelInfo().Provider)
			assert.Equal(t, "model-x", p.GetModel())
			assert.Equal(t, tt.wantBaseURL, p.GetBaseURL())
		})
	}
}

func TestNewProvider_Unknown(t *testing.T) {
	_, err := NewProvider(Settings{Provider: "gemini", APIKey: "key", Model: "m"})
	require.Error(t, err)
}

func TestBuildProvider_AnthropicFromConfig(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-env-key")
	t.Setenv("ANTHROPIC_BASE_URL", "")

	require.NoError(t, config.Initialize(t.TempDir()+"/config.json"))
	config.GetLLM().SetProvider(config.ProviderAnthropic)
	config.GetLLM().SetModel("claude-x")

	p, err := BuildProvider("", "", "", "default-model")
	require.NoError(t, err)
	assert.Equal(t, "anthropic", p.GetModelInfo().Provider)
	assert.Equal(t, "claude-x", p.GetModel())
	assert.Equal(t, "anthropic-env-key", p.GetAPIKey())
}

func TestBuildProvider_DefaultsToOpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "openai-env-key")
	t.Setenv("OPENAI_BASE_URL", "")

	require.NoError(t, config.Initialize(t.TempDir()+"/config.json"))

	p, err := BuildProvider("", "", "", "default-model")
	require.NoError(t, err)
	assert.Equal(t, "openai", p.GetModelInfo().Provider)
	assert.Equal(t, "default-model", p.GetModel())
}

func TestBuildProvider_MissingKeyNamesProviderEnvVar(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")

	// FORGE_API_KEY/FORGE_BASE_URL are a provider-neutral fallback (see
	// pkg/llm/factory) and must be cleared here too, otherwise a
	// FORGE_API_KEY set in the outer shell environment would satisfy
	// BuildProvider and this test would never hit the error path. Save the
	// originals and restore them explicitly rather than relying on an
	// empty t.Setenv, matching the save/restore pattern used elsewhere for
	// ambient env vars.
	originalForgeAPIKey := os.Getenv("FORGE_API_KEY")
	originalForgeBaseURL := os.Getenv("FORGE_BASE_URL")
	defer func() {
		if originalForgeAPIKey != "" {
			os.Setenv("FORGE_API_KEY", originalForgeAPIKey)
		} else {
			os.Unsetenv("FORGE_API_KEY")
		}
		if originalForgeBaseURL != "" {
			os.Setenv("FORGE_BASE_URL", originalForgeBaseURL)
		} else {
			os.Unsetenv("FORGE_BASE_URL")
		}
	}()
	os.Unsetenv("FORGE_API_KEY")
	os.Unsetenv("FORGE_BASE_URL")

	require.NoError(t, config.Initialize(t.TempDir()+"/config.json"))
	config.GetLLM().SetProvider(config.ProviderAnthropic)

	_, err := BuildProvider("", "", "", "m")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ANTHROPIC_API_KEY")
}

func TestBuildProvider_MaxTokensFromConfig(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("ANTHROPIC_BASE_URL", "")

	require.NoError(t, config.Initialize(t.TempDir()+"/config.json"))
	config.GetLLM().SetProvider(config.ProviderAnthropic)
	config.GetLLM().SetMaxTokens(4096)

	p, err := BuildProvider("", "", "", "m")
	require.NoError(t, err)
	assert.Equal(t, 4096, p.GetModelInfo().MaxTokens)
}

func TestBuildProvider_ForgeEnvFallback(t *testing.T) {
	require.NoError(t, config.Initialize(t.TempDir()+"/config.json"))
	config.GetLLM().SetProvider(config.ProviderAnthropic)

	t.Run("FORGE_* used when provider vars are unset", func(t *testing.T) {
		t.Setenv("ANTHROPIC_API_KEY", "")
		t.Setenv("ANTHROPIC_BASE_URL", "")
		t.Setenv("FORGE_API_KEY", "forge-key")
		t.Setenv("FORGE_BASE_URL", "https://forge.example.com")

		p, err := BuildProvider("", "", "", "m")
		require.NoError(t, err)
		assert.Equal(t, "forge-key", p.GetAPIKey())
		assert.Equal(t, "https://forge.example.com", p.GetBaseURL())
	})

	t.Run("provider vars win over FORGE_*", func(t *testing.T) {
		t.Setenv("ANTHROPIC_API_KEY", "anthropic-key")
		t.Setenv("ANTHROPIC_BASE_URL", "")
		t.Setenv("FORGE_API_KEY", "forge-key")
		t.Setenv("FORGE_BASE_URL", "https://forge.example.com")

		p, err := BuildProvider("", "", "", "m")
		require.NoError(t, err)
		assert.Equal(t, "anthropic-key", p.GetAPIKey())
		assert.Equal(t, "https://forge.example.com", p.GetBaseURL(), "each variable falls back independently")
	})

	t.Run("error names both variables", func(t *testing.T) {
		t.Setenv("ANTHROPIC_API_KEY", "")
		t.Setenv("FORGE_API_KEY", "")

		_, err := BuildProvider("", "", "", "m")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ANTHROPIC_API_KEY")
		assert.Contains(t, err.Error(), "FORGE_API_KEY")
	})
}
