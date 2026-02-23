package llm

import (
	"os"

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

	// Resolve base URL: config field > EMBEDDING_BASE_URL env var > OPENAI_BASE_URL env var (in embedding.go) > default.
	if baseURL := cfg.GetEmbeddingBaseURL(); baseURL != "" {
		opts = append(opts, embedding.WithBaseURL(baseURL))
	} else if envURL := os.Getenv("EMBEDDING_BASE_URL"); envURL != "" {
		opts = append(opts, embedding.WithBaseURL(envURL))
	}

	return embedding.NewProvider(
		apiKey,
		cfg.GetEmbeddingModel(),
		opts...,
	)
}
