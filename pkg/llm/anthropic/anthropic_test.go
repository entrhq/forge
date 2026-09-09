package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/entrhq/forge/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// streamFixture is a minimal Messages API SSE response producing "Hello world".
const streamFixture = `event: message_start
data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[],"stop_reason":null,"usage":{"input_tokens":12,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}

event: message_stop
data: {"type":"message_stop"}

`

func clearEnv(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_BASE_URL", "")
}

func TestNewProvider_Defaults(t *testing.T) {
	clearEnv(t)

	p, err := NewProvider("test-key")
	require.NoError(t, err)

	assert.Equal(t, "test-key", p.GetAPIKey())
	assert.Equal(t, defaultModel, p.GetModel())
	assert.Equal(t, DefaultBaseURL, p.GetBaseURL())
	assert.Equal(t, "anthropic", p.GetModelInfo().Provider)
	assert.True(t, p.GetModelInfo().SupportsStreaming)
}

func TestNewProvider_NoAPIKey(t *testing.T) {
	clearEnv(t)

	_, err := NewProvider("")
	require.Error(t, err)
}

func TestNewProvider_EnvironmentFallback(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "env-key")
	t.Setenv("ANTHROPIC_BASE_URL", "https://env.example.com")

	p, err := NewProvider("")
	require.NoError(t, err)

	assert.Equal(t, "env-key", p.GetAPIKey())
	assert.Equal(t, "https://env.example.com", p.GetBaseURL())
	assert.Equal(t, "https://env.example.com", p.GetModelInfo().Metadata["base_url"])
}

func TestNewProvider_OptionsOverrideEnvironment(t *testing.T) {
	t.Setenv("ANTHROPIC_BASE_URL", "https://env.example.com")

	p, err := NewProvider("key", WithModel("claude-x"), WithBaseURL("https://opt.example.com"))
	require.NoError(t, err)

	assert.Equal(t, "claude-x", p.GetModel())
	assert.Equal(t, "https://opt.example.com", p.GetBaseURL())
}

func TestCloneWithModel(t *testing.T) {
	clearEnv(t)

	p, err := NewProvider("key", WithModel("a"))
	require.NoError(t, err)

	clone := p.CloneWithModel("b")
	assert.Equal(t, "b", clone.GetModel())
	assert.Equal(t, "b", clone.GetModelInfo().Name)
	assert.Equal(t, "a", p.GetModel())
	assert.Equal(t, "a", p.GetModelInfo().Name)
}

func TestBuildParams(t *testing.T) {
	clearEnv(t)
	p, err := NewProvider("key", WithModel("m"))
	require.NoError(t, err)

	params := p.buildParams([]*types.Message{
		types.NewSystemMessage("sys one"),
		types.NewSystemMessage("sys two"),
		types.NewUserMessage("hi"),
		types.NewAssistantMessage(""),
		types.NewAssistantMessage("hello"),
		types.NewToolMessage("tool output"),
	})

	require.Len(t, params.System, 2)
	assert.Equal(t, "sys one", params.System[0].Text)
	assert.Equal(t, "sys two", params.System[1].Text)

	require.Len(t, params.Messages, 3)
	assert.Equal(t, "user", string(params.Messages[0].Role))
	assert.Equal(t, "assistant", string(params.Messages[1].Role))
	assert.Equal(t, "user", string(params.Messages[2].Role), "tool role falls back to user")
	assert.Equal(t, int64(defaultStreamMaxTokens), params.MaxTokens)
}

func TestStreamCompletion(t *testing.T) {
	clearEnv(t)

	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/messages", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &gotBody))

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, streamFixture)
	}))
	defer server.Close()

	p, err := NewProvider("test-key", WithModel("claude-sonnet-4-5"), WithBaseURL(server.URL))
	require.NoError(t, err)

	stream, err := p.StreamCompletion(context.Background(), []*types.Message{
		types.NewSystemMessage("be brief"),
		types.NewUserMessage("hi"),
	})
	require.NoError(t, err)

	var content strings.Builder
	var roles []string
	var finished bool
	for chunk := range stream {
		require.NoError(t, chunk.Error)
		content.WriteString(chunk.Content)
		if chunk.Role != "" {
			roles = append(roles, chunk.Role)
		}
		if chunk.Finished {
			finished = true
			require.NotNil(t, chunk.Usage)
			assert.Equal(t, 12, chunk.Usage.PromptTokens)
			assert.Equal(t, 3, chunk.Usage.CompletionTokens)
		}
	}

	assert.Equal(t, "Hello world", content.String())
	assert.Equal(t, []string{"assistant"}, roles, "role is sent exactly once")
	assert.True(t, finished)
	assert.Equal(t, true, gotBody["stream"])
	assert.Equal(t, "claude-sonnet-4-5", gotBody["model"])
}

func TestComplete_ErrorResponse(t *testing.T) {
	clearEnv(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"bad key"}}`)
	}))
	defer server.Close()

	p, err := NewProvider("bad", WithBaseURL(server.URL))
	require.NoError(t, err)

	_, err = p.Complete(context.Background(), []*types.Message{types.NewUserMessage("hi")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestAnalyzeDocument(t *testing.T) {
	clearEnv(t)

	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &gotBody))
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"msg_1","type":"message","role":"assistant","model":"m","content":[{"type":"text","text":"A cat."}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`)
	}))
	defer server.Close()

	p, err := NewProvider("key", WithBaseURL(server.URL))
	require.NoError(t, err)

	tests := []struct {
		mediaType string
		wantType  string
	}{
		{"image/png", "image"},
		{"application/pdf", "document"},
	}
	for _, tt := range tests {
		t.Run(tt.mediaType, func(t *testing.T) {
			text, err := p.AnalyzeDocument(context.Background(), []byte("data"), tt.mediaType, "")
			require.NoError(t, err)
			assert.Equal(t, "A cat.", text)

			messages := gotBody["messages"].([]any)
			content := messages[0].(map[string]any)["content"].([]any)
			require.Len(t, content, 2)
			assert.Equal(t, "text", content[0].(map[string]any)["type"])
			assert.Equal(t, tt.wantType, content[1].(map[string]any)["type"])
		})
	}
}

func TestAnalyzeDocument_UnsupportedMediaType(t *testing.T) {
	clearEnv(t)
	p, err := NewProvider("key")
	require.NoError(t, err)

	_, err = p.AnalyzeDocument(context.Background(), []byte("x"), "text/plain", "")
	require.Error(t, err)
}

func TestBuildParams_CacheBreakpoints(t *testing.T) {
	clearEnv(t)
	p, err := NewProvider("key")
	require.NoError(t, err)

	hasCache := func(block sdk.TextBlockParam) bool {
		return block.CacheControl.Type != ""
	}
	messageHasCache := func(msg sdk.MessageParam) bool {
		return msg.Content[0].OfText.CacheControl.Type != ""
	}

	t.Run("all four candidates present", func(t *testing.T) {
		params := p.buildParams([]*types.Message{
			types.NewSystemMessage("static a").WithStability(types.StabilityStatic),
			types.NewSystemMessage("static b").WithStability(types.StabilityStatic),
			types.NewSystemMessage("tools").WithStability(types.StabilitySession),
			types.NewUserMessage("old turn"),
			types.NewAssistantMessage("[SUMMARIZED] old").WithStability(types.StabilitySession),
			types.NewUserMessage("new turn"),
			types.NewAssistantMessage("reply"),
			types.NewUserMessage("latest"),
		})

		require.Len(t, params.System, 3)
		assert.False(t, hasCache(params.System[0]))
		assert.True(t, hasCache(params.System[1]), "last static system block")
		assert.True(t, hasCache(params.System[2]), "last system block")

		require.Len(t, params.Messages, 5)
		assert.False(t, messageHasCache(params.Messages[0]))
		assert.True(t, messageHasCache(params.Messages[1]), "most recent compaction summary")
		assert.False(t, messageHasCache(params.Messages[2]))
		assert.False(t, messageHasCache(params.Messages[3]))
		assert.True(t, messageHasCache(params.Messages[4]), "last message")
	})

	t.Run("no system prompt and no compaction", func(t *testing.T) {
		params := p.buildParams([]*types.Message{
			types.NewUserMessage("hi"),
			types.NewAssistantMessage("hello"),
			types.NewUserMessage("again"),
		})

		assert.Empty(t, params.System)
		require.Len(t, params.Messages, 3)
		assert.False(t, messageHasCache(params.Messages[0]))
		assert.False(t, messageHasCache(params.Messages[1]))
		assert.True(t, messageHasCache(params.Messages[2]), "only the last message is marked")
	})

	t.Run("single untagged system message", func(t *testing.T) {
		params := p.buildParams([]*types.Message{
			types.NewSystemMessage("sys"),
			types.NewUserMessage("hi"),
		})

		require.Len(t, params.System, 1)
		assert.True(t, hasCache(params.System[0]), "last system block is marked even without a static run")
		assert.True(t, messageHasCache(params.Messages[0]))
	})
}

func TestStreamCompletion_SendsCacheControlOnWire(t *testing.T) {
	clearEnv(t)

	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &gotBody))
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, streamFixture)
	}))
	defer server.Close()

	p, err := NewProvider("key", WithBaseURL(server.URL))
	require.NoError(t, err)

	stream, err := p.StreamCompletion(context.Background(), []*types.Message{
		types.NewSystemMessage("sys").WithStability(types.StabilityStatic),
		types.NewUserMessage("hi"),
	})
	require.NoError(t, err)
	for range stream {
	}

	system := gotBody["system"].([]any)
	cache := system[0].(map[string]any)["cache_control"].(map[string]any)
	assert.Equal(t, "ephemeral", cache["type"])
	_, hasTTL := cache["ttl"]
	assert.False(t, hasTTL, "default 5 minute TTL is left implicit")

	messages := gotBody["messages"].([]any)
	content := messages[0].(map[string]any)["content"].([]any)
	assert.Contains(t, content[0].(map[string]any), "cache_control")
}

func TestMaxTokens(t *testing.T) {
	clearEnv(t)

	t.Run("defaults", func(t *testing.T) {
		p, err := NewProvider("key")
		require.NoError(t, err)
		assert.Equal(t, int64(defaultStreamMaxTokens), p.streamMaxTokens())
		assert.Equal(t, int64(defaultDocumentMaxTokens), p.documentMaxTokens())
		assert.Equal(t, defaultStreamMaxTokens, p.GetModelInfo().MaxTokens)
	})

	t.Run("configured value applies to both, capped for documents", func(t *testing.T) {
		p, err := NewProvider("key", WithMaxTokens(100000))
		require.NoError(t, err)
		assert.Equal(t, int64(100000), p.streamMaxTokens())
		assert.Equal(t, int64(defaultDocumentMaxTokens), p.documentMaxTokens())
	})

	t.Run("low configured value caps documents too", func(t *testing.T) {
		p, err := NewProvider("key", WithMaxTokens(4096))
		require.NoError(t, err)
		assert.Equal(t, int64(4096), p.streamMaxTokens())
		assert.Equal(t, int64(4096), p.documentMaxTokens())
	})

	t.Run("zero and negative are ignored", func(t *testing.T) {
		p, err := NewProvider("key", WithMaxTokens(0), WithMaxTokens(-1))
		require.NoError(t, err)
		assert.Equal(t, int64(defaultStreamMaxTokens), p.streamMaxTokens())
	})
}
