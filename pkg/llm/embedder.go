package llm

import "context"

// Embedder computes dense vector embeddings for text inputs.
// Implementations must be safe for concurrent use.
//
// A nil Embedder is valid and means retrieval is disabled — callers must
// check for nil before calling any method.
type Embedder interface {
	// Embed returns a normalised float32 embedding vector for each input string.
	// The order of output vectors corresponds exactly to the order of inputs.
	// Returns an error if the provider call fails or any input exceeds the
	// model's token limit.
	Embed(ctx context.Context, inputs []string) ([][]float32, error)

	// Model returns the embedding model identifier string.
	Model() string
}
