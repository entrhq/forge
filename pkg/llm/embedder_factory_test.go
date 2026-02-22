package llm_test

import (
	"testing"

	"github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/llm"
)

func TestNewEmbedder(t *testing.T) {
	t.Run("returns nil,nil when disabled", func(t *testing.T) {
		cfg := config.NewMemorySection()
		cfg.SetData(map[string]any{
			"enabled":         false,
			"embedding_model": "test-model",
		})
		embedder, err := llm.NewEmbedder(cfg, "fake-key")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if embedder != nil {
			t.Fatalf("expected nil embedder, got %v", embedder)
		}
	})

	t.Run("returns nil,nil when model empty", func(t *testing.T) {
		cfg := config.NewMemorySection()
		cfg.SetData(map[string]any{
			"enabled":         true,
			"embedding_model": "",
		})
		embedder, err := llm.NewEmbedder(cfg, "fake-key")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if embedder != nil {
			t.Fatalf("expected nil embedder, got %v", embedder)
		}
	})

	t.Run("constructs provider when valid", func(t *testing.T) {
		cfg := config.NewMemorySection()
		cfg.SetData(map[string]any{
			"enabled":            true,
			"embedding_model":    "text-embedding-3-small",
			"embedding_base_url": "https://custom.api/v1",
		})
		embedder, err := llm.NewEmbedder(cfg, "fake-key")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if embedder == nil {
			t.Fatalf("expected embedder, got nil")
		}
		if embedder.Model() != "text-embedding-3-small" {
			t.Errorf("expected model text-embedding-3-small, got %s", embedder.Model())
		}
	})
}
