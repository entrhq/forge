// pkg/llm/mock_embedder_test.go
package llm_test

import (
	"context"

	"github.com/entrhq/forge/pkg/llm"
)

// Compile-time interface check.
var _ llm.Embedder = (*MockEmbedder)(nil)

// MockEmbedder is a test double for llm.Embedder.
type MockEmbedder struct {
	EmbedFn  func(ctx context.Context, inputs []string) ([][]float32, error)
	ModelStr string
}

func (m *MockEmbedder) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if m.EmbedFn == nil {
		return nil, nil // Default safe behavior
	}
	return m.EmbedFn(ctx, inputs)
}

func (m *MockEmbedder) Model() string {
	return m.ModelStr
}
