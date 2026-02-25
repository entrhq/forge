package embedding_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/entrhq/forge/pkg/llm/embedding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	// Clear potentially interfering env vars
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_BASE_URL")

	t.Run("success with explicit config", func(t *testing.T) {
		p, err := embedding.NewProvider("test-key", "test-model", embedding.WithBaseURL("https://custom.api.com"))
		require.NoError(t, err)
		assert.Equal(t, "test-model", p.Model())
	})

	t.Run("success with env var key", func(t *testing.T) {
		os.Setenv("OPENAI_API_KEY", "env-key")
		defer os.Unsetenv("OPENAI_API_KEY")

		p, err := embedding.NewProvider("", "test-model")
		require.NoError(t, err)
		assert.NotNil(t, p)
	})

	t.Run("success with env var base url", func(t *testing.T) {
		os.Setenv("OPENAI_BASE_URL", "https://env.api.com")
		defer os.Unsetenv("OPENAI_BASE_URL")

		p, err := embedding.NewProvider("test-key", "test-model")
		require.NoError(t, err)
		assert.NotNil(t, p)
		// We indirectly test baseURL through a failed Embed call's error string if needed,
		// but testing the constructor logic is sufficient for now.
	})

	t.Run("missing api key", func(t *testing.T) {
		_, err := embedding.NewProvider("", "test-model")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key required")
	})

	t.Run("missing model", func(t *testing.T) {
		_, err := embedding.NewProvider("test-key", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model name must not be empty")
	})
}

func TestEmbed_Success(t *testing.T) {
	// Create a mock server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/embeddings", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Decode the request body to verify model and input
		var body struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)

		assert.Equal(t, "test-model", body.Model)
		assert.Equal(t, []string{"Hello", "World"}, body.Input)

		// Send mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{
					"embedding": [0.1, 0.2, 0.3],
					"index": 0
				},
				{
					"embedding": [0.4, 0.5, 0.6],
					"index": 1
				}
			]
		}`))
	}))
	defer ts.Close()

	// Initialize provider pointing to the mock server
	p, err := embedding.NewProvider("test-key", "test-model",
		embedding.WithBaseURL(ts.URL),
		embedding.WithHTTPClient(ts.Client()),
	)
	require.NoError(t, err)

	// Test Embed
	ctx := context.Background()
	results, err := p.Embed(ctx, []string{"Hello", "World"})
	require.NoError(t, err)

	assert.Len(t, results, 2)
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, results[0])
	assert.Equal(t, []float32{0.4, 0.5, 0.6}, results[1])
}

func TestEmbed_ReorderByIndex(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return results out of order
		w.Write([]byte(`{
			"data": [
				{
					"embedding": [0.4, 0.5, 0.6],
					"index": 1
				},
				{
					"embedding": [0.1, 0.2, 0.3],
					"index": 0
				}
			]
		}`))
	}))
	defer ts.Close()

	p, err := embedding.NewProvider("test-key", "test-model",
		embedding.WithBaseURL(ts.URL),
		embedding.WithHTTPClient(ts.Client()),
	)
	require.NoError(t, err)

	results, err := p.Embed(context.Background(), []string{"Hello", "World"})
	require.NoError(t, err)

	assert.Len(t, results, 2)
	assert.Equal(t, []float32{0.1, 0.2, 0.3}, results[0]) // index 0 comes first in array
	assert.Equal(t, []float32{0.4, 0.5, 0.6}, results[1]) // index 1 comes second
}

func TestEmbed_ProviderError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid api key"}`))
	}))
	defer ts.Close()

	p, err := embedding.NewProvider("test-key", "test-model",
		embedding.WithBaseURL(ts.URL),
		embedding.WithHTTPClient(ts.Client()),
	)
	require.NoError(t, err)

	_, err = p.Embed(context.Background(), []string{"Hello"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider returned 401")
	assert.Contains(t, err.Error(), "invalid api key")
}

func TestEmbed_EmptyInput(t *testing.T) {
	p, err := embedding.NewProvider("test-key", "test-model")
	require.NoError(t, err)

	results, err := p.Embed(context.Background(), []string{})
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestEmbed_InvalidJSONResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid-json}`))
	}))
	defer ts.Close()

	p, err := embedding.NewProvider("test-key", "test-model",
		embedding.WithBaseURL(ts.URL),
		embedding.WithHTTPClient(ts.Client()),
	)
	require.NoError(t, err)

	_, err = p.Embed(context.Background(), []string{"Hello"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}
