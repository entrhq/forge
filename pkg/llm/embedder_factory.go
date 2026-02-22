package llm

import (
	"github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/llm/embedding"
)

// NewEmbedder creates an Embedder based on the provided memory configuration.
// It returns (nil, nil) if memory or embeddings are disabled gracefully.
// The provided apiKey is used for authentication; if empty, typical provider
// defaults (like falling back to environment variables) apply.
func NewEmbedder(cfg *config.MemorySection, apiKey string, opts ...embedding.ProviderOption) (Embedder, error) {
	if cfg == nil || !cfg.IsEnabled() || cfg.GetEmbeddingModel() == "" {
		return nil, nil // gracefully disabled
	}

	// Wire up base URL from config if set, otherwise fallback to provider defaults.
	if baseURL := cfg.GetEmbeddingBaseURL(); baseURL != "" {
		opts = append(opts, embedding.WithBaseURL(baseURL))
	}

	return embedding.NewProvider(
		apiKey,
		cfg.GetEmbeddingModel(),
		opts...,
	)
}
