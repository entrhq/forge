package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/entrhq/forge/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_AnalyzeDocument_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/chat/completions", r.URL.Path)

		// Verify authorization header
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse request body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var reqBody map[string]any
		require.NoError(t, json.Unmarshal(body, &reqBody))

		// Verify model
		assert.Equal(t, "test-model", reqBody["model"])

		// Verify messages structure
		messages, ok := reqBody["messages"].([]any)
		require.True(t, ok)
		require.GreaterOrEqual(t, len(messages), 1, "should have at least 1 message")

		// Find user message (could be first or second depending on system prompt)
		var userMsg map[string]any
		for _, msg := range messages {
			m := msg.(map[string]any)
			if m["role"] == "user" {
				userMsg = m
				break
			}
		}
		require.NotNil(t, userMsg, "should have user message")

		// Verify content is array with text and image parts
		content, ok := userMsg["content"].([]any)
		require.True(t, ok)
		require.Len(t, content, 2)

		// Verify text part
		textPart, ok := content[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "text", textPart["type"])
		assert.Contains(t, textPart["text"], "test prompt")

		// Verify image part
		imagePart, ok := content[1].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "image_url", imagePart["type"])

		imageURL, ok := imagePart["image_url"].(map[string]any)
		require.True(t, ok)
		dataURL, ok := imageURL["url"].(string)
		require.True(t, ok)
		assert.True(t, strings.HasPrefix(dataURL, "data:image/png;base64,"))

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "test-id",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "test-model",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "This is the analysis result",
					},
					"finish_reason": "stop",
				},
			},
		})
	}))
	defer server.Close()

	// Create provider
	provider := &Provider{
		httpClient: http.DefaultClient,
		apiKey:     "test-key",
		baseURL:    server.URL,
		model:      "test-model",
		modelInfo: &types.ModelInfo{
			Name: "test-model",
		},
	}

	// Test image analysis
	result, err := provider.AnalyzeDocument(
		context.Background(),
		[]byte("fake image data"),
		"image/png",
		"test prompt",
	)

	require.NoError(t, err)
	assert.Equal(t, "This is the analysis result", result)
}

func TestProvider_AnalyzeDocument_PDFContentType(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var reqBody map[string]any
		require.NoError(t, json.Unmarshal(body, &reqBody))

		// Verify messages structure
		messages, ok := reqBody["messages"].([]any)
		require.True(t, ok)

		// Find user message
		var userMsg map[string]any
		for _, msg := range messages {
			m := msg.(map[string]any)
			if m["role"] == "user" {
				userMsg = m
				break
			}
		}
		require.NotNil(t, userMsg, "should have user message")

		content := userMsg["content"].([]any)

		// Verify PDF is sent as image_url data URI (OpenAI-compatible format)
		filePart, ok := content[1].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "image_url", filePart["type"])

		imageURL, ok := filePart["image_url"].(map[string]any)
		require.True(t, ok)
		url, ok := imageURL["url"].(string)
		require.True(t, ok)
		assert.Contains(t, url, "data:application/pdf;base64,")

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"content": "PDF analysis result",
					},
				},
			},
		})
	}))
	defer server.Close()

	provider := &Provider{
		httpClient: http.DefaultClient,
		apiKey:     "test-key",
		baseURL:    server.URL,
		model:      "test-model",
		modelInfo: &types.ModelInfo{
			Name: "test-model",
		},
	}

	// Test PDF analysis
	result, err := provider.AnalyzeDocument(
		context.Background(),
		[]byte("fake pdf data"),
		"application/pdf",
		"analyze this PDF",
	)

	require.NoError(t, err)
	assert.Equal(t, "PDF analysis result", result)
}

func TestProvider_AnalyzeDocument_ErrorResponse(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Invalid request",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	provider := &Provider{
		httpClient: http.DefaultClient,
		apiKey:     "test-key",
		baseURL:    server.URL,
		model:      "test-model",
		modelInfo: &types.ModelInfo{
			Name: "test-model",
		},
	}

	_, err := provider.AnalyzeDocument(
		context.Background(),
		[]byte("fake data"),
		"image/png",
		"test",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "API request failed with status 400")
}

func TestProvider_AnalyzeDocument_EmptyChoices(t *testing.T) {
	// Create test server that returns empty choices
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{},
		})
	}))
	defer server.Close()

	provider := &Provider{
		httpClient: http.DefaultClient,
		apiKey:     "test-key",
		baseURL:    server.URL,
		model:      "test-model",
		modelInfo: &types.ModelInfo{
			Name: "test-model",
		},
	}

	_, err := provider.AnalyzeDocument(
		context.Background(),
		[]byte("fake data"),
		"image/png",
		"test",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no response from API")
}

func TestProvider_AnalyzeDocument_ContextCanceled(t *testing.T) {
	// Create test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wait for context cancellation
		<-r.Context().Done()
	}))
	defer server.Close()

	provider := &Provider{
		httpClient: http.DefaultClient,
		apiKey:     "test-key",
		baseURL:    server.URL,
		model:      "test-model",
		modelInfo: &types.ModelInfo{
			Name: "test-model",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := provider.AnalyzeDocument(
		ctx,
		[]byte("fake data"),
		"image/png",
		"test",
	)

	require.Error(t, err)
}
