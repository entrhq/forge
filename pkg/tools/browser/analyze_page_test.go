package browser

import (
	"context"
	"testing"

	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAnalyzeProvider implements a minimal llm.Provider for testing
type mockAnalyzeProvider struct {
	response string
	err      error
}

func (m *mockAnalyzeProvider) StreamCompletion(ctx context.Context, messages []*types.Message) (<-chan *llm.StreamChunk, error) {
	ch := make(chan *llm.StreamChunk, 1)
	go func() {
		defer close(ch)
		if m.err != nil {
			ch <- &llm.StreamChunk{Error: m.err}
			return
		}
		ch <- &llm.StreamChunk{
			Content:  m.response,
			Finished: true,
		}
	}()
	return ch, nil
}

func (m *mockAnalyzeProvider) Complete(ctx context.Context, messages []*types.Message) (*types.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	return types.NewAssistantMessage(m.response), nil
}

func (m *mockAnalyzeProvider) GetModelInfo() *types.ModelInfo {
	return &types.ModelInfo{
		Name:              "mock-model",
		Provider:          "mock",
		MaxTokens:         100000,
		SupportsStreaming: true,
	}
}

func (m *mockAnalyzeProvider) GetModel() string {
	return "mock-model"
}

func (m *mockAnalyzeProvider) GetBaseURL() string {
	return "http://localhost"
}

func (m *mockAnalyzeProvider) GetAPIKey() string {
	return "mock-key"
}

func TestAnalyzePageTool_Metadata(t *testing.T) {
	manager := NewSessionManager()
	provider := &mockAnalyzeProvider{response: "test analysis"}
	tool := NewAnalyzePageTool(manager, provider)

	assert.Equal(t, "analyze_page", tool.Name())
	assert.NotEmpty(t, tool.Description())
	assert.NotEmpty(t, tool.Schema())
}

func TestAnalyzePageTool_NoProvider(t *testing.T) {
	manager := NewSessionManager()
	tool := NewAnalyzePageTool(manager, nil)

	argsXML := []byte(`<arguments><session>test</session></arguments>`)
	result, metadata, err := tool.Execute(context.Background(), argsXML)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM provider not available")
	assert.Empty(t, result)
	assert.Nil(t, metadata)
}

func TestAnalyzePageTool_NoSession(t *testing.T) {
	manager := NewSessionManager()
	provider := &mockAnalyzeProvider{response: "test analysis"}
	tool := NewAnalyzePageTool(manager, provider)

	argsXML := []byte(`<arguments><session>nonexistent</session></arguments>`)
	result, metadata, err := tool.Execute(context.Background(), argsXML)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session not found")
	assert.Empty(t, result)
	assert.Nil(t, metadata)
}

func TestAnalyzePageTool_PreviewGeneration(t *testing.T) {
	manager := NewSessionManager()
	provider := &mockAnalyzeProvider{
		response: "Page Type: Documentation\nPurpose: API Reference",
	}
	tool := NewAnalyzePageTool(manager, provider)

	argsXML := []byte(`<arguments><session>test</session></arguments>`)
	preview, err := tool.GeneratePreview(context.Background(), argsXML)
	require.NoError(t, err)
	require.NotNil(t, preview)
	assert.Equal(t, "Analyze Page with AI", preview.Title)
	assert.Contains(t, preview.Description, "test")
}

func TestAnalyzePageTool_BuildPrompt(t *testing.T) {
	markdown := "# Test Page\n\nSome content here"
	url := "https://example.com"
	title := "Test Page"

	prompt := buildAnalysisPrompt(url, title, markdown, "")

	assert.Contains(t, prompt, url)
	assert.Contains(t, prompt, title)
	assert.Contains(t, prompt, markdown)
	assert.Contains(t, prompt, "PAGE TYPE")
	assert.Contains(t, prompt, "PURPOSE")
	assert.Contains(t, prompt, "KEY ELEMENTS")
}
