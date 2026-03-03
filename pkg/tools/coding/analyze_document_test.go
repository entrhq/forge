package coding

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/security/workspace"
	"github.com/entrhq/forge/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAnalyzeProvider implements llm.Provider for testing
type mockAnalyzeProvider struct {
	response string
	err      error
}

func (m *mockAnalyzeProvider) StreamCompletion(ctx context.Context, messages []*types.Message) (<-chan *llm.StreamChunk, error) {
	return nil, errors.New("not implemented")
}

func (m *mockAnalyzeProvider) Complete(ctx context.Context, messages []*types.Message) (*types.Message, error) {
	return nil, errors.New("not implemented")
}

func (m *mockAnalyzeProvider) AnalyzeDocument(ctx context.Context, fileData []byte, mediaType string, prompt string) (string, error) {
	return m.response, m.err
}

func (m *mockAnalyzeProvider) GetModel() string {
	return "test-model"
}

func (m *mockAnalyzeProvider) GetAPIKey() string {
	return "test-key"
}

func (m *mockAnalyzeProvider) GetBaseURL() string {
	return "https://api.openai.com/v1"
}

func (m *mockAnalyzeProvider) GetModelInfo() *types.ModelInfo {
	return &types.ModelInfo{
		Name: "test-model",
	}
}

func TestAnalyzeDocumentTool_Name(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := workspace.NewGuard(tmpDir)
	require.NoError(t, err)
	provider := &mockAnalyzeProvider{}
	tool := NewAnalyzeDocumentTool(guard, provider)

	assert.Equal(t, "analyze_document", tool.Name())
}

func TestAnalyzeDocumentTool_Description(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := workspace.NewGuard(tmpDir)
	require.NoError(t, err)
	provider := &mockAnalyzeProvider{}
	tool := NewAnalyzeDocumentTool(guard, provider)

	desc := tool.Description()
	assert.Contains(t, desc, "Analyze a PNG, JPG, or PDF document")
}

func TestAnalyzeDocumentTool_Schema(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := workspace.NewGuard(tmpDir)
	require.NoError(t, err)
	provider := &mockAnalyzeProvider{}
	tool := NewAnalyzeDocumentTool(guard, provider)

	schema := tool.Schema()
	assert.NotNil(t, schema)
}

func TestAnalyzeDocumentTool_Execute_PathRequired(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := workspace.NewGuard(tmpDir)
	require.NoError(t, err)
	provider := &mockAnalyzeProvider{}
	tool := NewAnalyzeDocumentTool(guard, provider)

	xmlInput := []byte(`<arguments></arguments>`)
	_, _, err = tool.Execute(context.Background(), xmlInput)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path")
}

func TestAnalyzeDocumentTool_Execute_WorkspaceGuard(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := workspace.NewGuard(tmpDir)
	require.NoError(t, err)
	provider := &mockAnalyzeProvider{}
	tool := NewAnalyzeDocumentTool(guard, provider)

	// Try to access file outside workspace
	xmlInput := []byte(`<arguments><path>/etc/passwd</path></arguments>`)
	_, _, err = tool.Execute(context.Background(), xmlInput)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid path")
}

func TestAnalyzeDocumentTool_Execute_UnsupportedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := workspace.NewGuard(tmpDir)
	require.NoError(t, err)
	provider := &mockAnalyzeProvider{}
	tool := NewAnalyzeDocumentTool(guard, provider)

	// Create an unsupported file
	txtPath := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(txtPath, []byte("test"), 0644))

	xmlInput := []byte(`<arguments><path>test.txt</path></arguments>`)
	_, _, err = tool.Execute(context.Background(), xmlInput)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file format")
}

func TestAnalyzeDocumentTool_Execute_ImageSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := workspace.NewGuard(tmpDir)
	require.NoError(t, err)
	provider := &mockAnalyzeProvider{
		response: "This is a test image analysis",
	}
	tool := NewAnalyzeDocumentTool(guard, provider)

	// Create a fake PNG file (just needs .png extension for this test)
	pngPath := filepath.Join(tmpDir, "test.png")
	require.NoError(t, os.WriteFile(pngPath, []byte("fake png data"), 0644))

	xmlInput := []byte(`<arguments><path>test.png</path></arguments>`)
	result, _, err := tool.Execute(context.Background(), xmlInput)
	require.NoError(t, err)
	assert.Contains(t, result, "This is a test image analysis")
}

func TestAnalyzeDocumentTool_Execute_ProviderError(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := workspace.NewGuard(tmpDir)
	require.NoError(t, err)
	provider := &mockAnalyzeProvider{
		err: errors.New("provider error"),
	}
	tool := NewAnalyzeDocumentTool(guard, provider)

	// Create a fake image file
	pngPath := filepath.Join(tmpDir, "test.png")
	require.NoError(t, os.WriteFile(pngPath, []byte("fake png data"), 0644))

	xmlInput := []byte(`<arguments><path>test.png</path></arguments>`)
	_, _, err = tool.Execute(context.Background(), xmlInput)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider error")
}

func TestAnalyzeDocumentTool_Execute_CustomPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := workspace.NewGuard(tmpDir)
	require.NoError(t, err)
	provider := &mockAnalyzeProvider{
		response: "Custom analysis result",
	}
	tool := NewAnalyzeDocumentTool(guard, provider)

	// Create a fake image
	pngPath := filepath.Join(tmpDir, "test.png")
	require.NoError(t, os.WriteFile(pngPath, []byte("fake png data"), 0644))

	xmlInput := []byte(`<arguments><path>test.png</path><prompt>What colors are in this image?</prompt></arguments>`)
	result, _, err := tool.Execute(context.Background(), xmlInput)
	require.NoError(t, err)
	assert.Contains(t, result, "Custom analysis result")
}

func TestAnalyzeDocumentTool_calculatePageRange(t *testing.T) {
	tmpDir := t.TempDir()
	guard, err := workspace.NewGuard(tmpDir)
	require.NoError(t, err)
	provider := &mockAnalyzeProvider{}
	tool := NewAnalyzeDocumentTool(guard, provider)

	tests := []struct {
		name          string
		pageCount     int
		pageStart     int
		pageEnd       int
		pageLimit     int
		wantStart     int
		wantEnd       int
		wantErr       bool
		wantErrString string
	}{
		{
			name:      "no params - apply limit from page 1",
			pageCount: 100,
			pageStart: 0,
			pageEnd:   0,
			pageLimit: 10,
			wantStart: 1,
			wantEnd:   10,
			wantErr:   false,
		},
		{
			name:      "no params - limit 0 means all pages",
			pageCount: 100,
			pageStart: 0,
			pageEnd:   0,
			pageLimit: 0,
			wantStart: 1,
			wantEnd:   100,
			wantErr:   false,
		},
		{
			name:      "only page_start - apply limit from start",
			pageCount: 100,
			pageStart: 20,
			pageEnd:   0,
			pageLimit: 10,
			wantStart: 20,
			wantEnd:   29,
			wantErr:   false,
		},
		{
			name:      "both specified - honor explicit range",
			pageCount: 100,
			pageStart: 20,
			pageEnd:   30,
			pageLimit: 5,
			wantStart: 20,
			wantEnd:   30,
			wantErr:   false,
		},

		{
			name:      "end beyond page count - clamp to max",
			pageCount: 100,
			pageStart: 95,
			pageEnd:   150,
			pageLimit: 10,
			wantStart: 95,
			wantEnd:   100,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := tool.calculatePageRange(tt.pageStart, tt.pageEnd, tt.pageCount, tt.pageLimit)
			assert.Equal(t, tt.wantStart, gotStart)
			assert.Equal(t, tt.wantEnd, gotEnd)
		})
	}
}
