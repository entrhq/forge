package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const DefaultBaseURL = "https://api.openai.com/v1"

// Provider implements Embedder using the OpenAI-compatible embeddings
// HTTP endpoint (POST {baseURL}/embeddings). Any provider that exposes this
// endpoint format is supported via WithBaseURL.
type Provider struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	model      string
}

// ProviderOption configures a Provider.
type ProviderOption func(*Provider)

// WithBaseURL overrides the API base URL. Use this to target any
// OpenAI-compatible embedding endpoint.
func WithBaseURL(url string) ProviderOption {
	return func(p *Provider) { p.baseURL = url }
}

// WithHTTPClient overrides the default http.Client (useful for testing).
func WithHTTPClient(c *http.Client) ProviderOption {
	return func(p *Provider) { p.httpClient = c }
}

// NewProvider creates an embedding Provider.
func NewProvider(apiKey, model string, opts ...ProviderOption) (*Provider, error) {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("embedding: API key required (set OPENAI_API_KEY or pass via config)")
	}
	if model == "" {
		return nil, fmt.Errorf("embedding: model name must not be empty")
	}
	p := &Provider{
		httpClient: &http.Client{},
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		model:      model,
	}
	for _, opt := range opts {
		opt(p)
	}
	// Environment variable fallback for base URL
	if p.baseURL == DefaultBaseURL {
		if env := os.Getenv("OPENAI_BASE_URL"); env != "" {
			p.baseURL = env
		}
	}
	return p, nil
}

func (p *Provider) Model() string { return p.model }

// Embed sends a batch embedding request and returns one vector per input.
func (p *Provider) Embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(map[string]any{
		"model": p.model,
		"input": inputs,
	})
	if err != nil {
		return nil, fmt.Errorf("embedding: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embedding: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding: http: %w", err)
	}
	defer resp.Body.Close()

	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("embedding: read response body: %w", readErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding: provider returned %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("embedding: decode response: %w", err)
	}

	// Validate the provider returned exactly one embedding per input.
	if len(result.Data) != len(inputs) {
		return nil, fmt.Errorf("embedding: provider returned %d embeddings for %d inputs", len(result.Data), len(inputs))
	}

	// Re-order by index — OpenAI guarantees order but be defensive.
	out := make([][]float32, len(inputs))
	seen := make([]bool, len(inputs))
	for _, d := range result.Data {
		if d.Index < 0 || d.Index >= len(out) {
			return nil, fmt.Errorf("embedding: provider returned out-of-range index %d", d.Index)
		}
		if seen[d.Index] {
			return nil, fmt.Errorf("embedding: provider returned duplicate embedding for index %d", d.Index)
		}
		out[d.Index] = d.Embedding
		seen[d.Index] = true
	}
	for i, ok := range seen {
		if !ok {
			return nil, fmt.Errorf("embedding: provider response missing embedding for index %d", i)
		}
	}
	return out, nil
}
