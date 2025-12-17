package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/entrhq/forge/internal/testing/configtest"
	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/llm/openai"
	"github.com/entrhq/forge/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestTool struct {
	tools.BaseTool
}

func (t *TestTool) Name() string {
	return "test_tool"
}

func (t *TestTool) Execute(ctx context.Context, args map[string]any) (*tools.ToolResult, error) {
	message, _ := args["message"].(string)
	return &tools.ToolResult{
		Message: fmt.Sprintf("Tool executed with message: %s", message),
	}, nil
}

func TestNew(t *testing.T) {
	testCases := []struct {
		name        string
		options     []Option
		expectedErr string
	}{
		{
			name: "WithLLM",
			options: []Option{
				WithLLM(&llm.MockProvider{}),
			},
		},
		{
			name:        "NoLLM",
			options:     []Option{},
			expectedErr: "LLM provider is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configtest.WithGlobalManager(t, func() {
				_, err := New(tc.options...)
				if tc.expectedErr != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tc.expectedErr)
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}

func TestAgent_ToolExecution(t *testing.T) {
	configtest.WithGlobalManager(t, func() {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			// Respond with a simple tool call
			io.WriteString(w, `data: {"id":"chatcmpl-123", "object":"chat.completion.chunk", "created":1694268190, "model":"gpt-4", "choices":[{"index":0, "delta":{"role":"assistant"}, "finish_reason":null}]}`+"\n\n")
			io.WriteString(w, `data: {"id":"chatcmpl-123", "object":"chat.completion.chunk", "created":1694268190, "model":"gpt-4", "choices":[{"index":0, "delta":{"content":"<tool><server_name>local</server_name><tool_name>test_tool</tool_name><arguments><message>Hello</message></arguments></tool>"}, "finish_reason":null}]}`+"\n\n")
			io.WriteString(w, `data: [DONE]`+"\n\n")
		}))
		defer mockServer.Close()

		provider,
			err := openai.New(
			openai.WithBaseURL(mockServer.URL),
			openai.WithAPIKey("test-key"),
		)
		require.NoError(t, err)

		agent,
			err := New(
			WithLLM(provider),
			WithTools(&TestTool{}),
		)
		require.NoError(t, err)

		ctx,
			cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = agent.Run(ctx, types.NewUserInput("test"))
		require.NoError(t, err)
	})
}

func TestAgent_ReloadLLM(t *testing.T) {
	configtest.WithGlobalManager(t, func() {
		manager := config.Global()

		llmSection := config.NewLLMSection()
		err := manager.RegisterSection(llmSection)
		require.NoError(t, err)

		llmSection.SetModel("initial-model")
		err = manager.SaveAll()
		require.NoError(t, err)

		agent,
			err := New()
		require.NoError(t, err)
		assert.Equal(t, "initial-model", agent.llm.GetModel())

		llmSection.SetModel("reloaded-model")
		err = manager.SaveAll()
		require.NoError(t, err)

		err = agent.ReloadLLMProvider()
		require.NoError(t, err)

		assert.Equal(t, "reloaded-model", agent.llm.GetModel())
	})
}

