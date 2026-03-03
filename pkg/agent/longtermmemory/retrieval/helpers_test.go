package retrieval

import (
	"context"
	"errors"
	"sync"

	"github.com/entrhq/forge/pkg/agent/longtermmemory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/types"
)

// ── memory helpers ───────────────────────────────────────────────────────────

func makeMemoryFile(id, content, category string) *longtermmemory.MemoryFile {
	return &longtermmemory.MemoryFile{
		Meta: longtermmemory.MemoryMeta{
			ID:       id,
			Scope:    longtermmemory.ScopeRepo,
			Category: longtermmemory.Category(category),
		},
		Content: content,
	}
}

func makeMemoryFileWithRelated(id, content string, related []longtermmemory.RelatedMemory) *longtermmemory.MemoryFile {
	return &longtermmemory.MemoryFile{
		Meta: longtermmemory.MemoryMeta{
			ID:      id,
			Scope:   longtermmemory.ScopeRepo,
			Related: related,
		},
		Content: content,
	}
}

// ── fake store ───────────────────────────────────────────────────────────────

type fakeStore struct {
	mu      sync.Mutex
	files   []*longtermmemory.MemoryFile
	listErr error
}

func (s *fakeStore) Write(_ context.Context, m *longtermmemory.MemoryFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files = append(s.files, m)
	return nil
}

func (s *fakeStore) Read(_ context.Context, id string) (*longtermmemory.MemoryFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.files {
		if f.Meta.ID == id {
			return f, nil
		}
	}
	return nil, longtermmemory.ErrNotFound
}

func (s *fakeStore) List(_ context.Context) ([]*longtermmemory.MemoryFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listErr != nil {
		return nil, s.listErr
	}
	out := make([]*longtermmemory.MemoryFile, len(s.files))
	copy(out, s.files)
	return out, nil
}

func (s *fakeStore) ListByScope(_ context.Context, scope longtermmemory.Scope) ([]*longtermmemory.MemoryFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*longtermmemory.MemoryFile
	for _, f := range s.files {
		if f.Meta.Scope == scope {
			out = append(out, f)
		}
	}
	return out, nil
}

// ── fake embedder ─────────────────────────────────────────────────────────────

type fakeEmbedder struct {
	vecs     map[string][]float32
	dim      int
	embedErr error
}

func newFakeEmbedder(dim int) *fakeEmbedder {
	return &fakeEmbedder{dim: dim, vecs: make(map[string][]float32)}
}

func (e *fakeEmbedder) set(text string, v []float32) { e.vecs[text] = v }

func (e *fakeEmbedder) Embed(_ context.Context, inputs []string) ([][]float32, error) {
	if e.embedErr != nil {
		return nil, e.embedErr
	}
	out := make([][]float32, len(inputs))
	for i, s := range inputs {
		if v, ok := e.vecs[s]; ok {
			out[i] = v
		} else {
			out[i] = make([]float32, e.dim)
			if e.dim > 0 {
				out[i][0] = 0.1
			}
		}
	}
	return out, nil
}

func (e *fakeEmbedder) Model() string { return "fake-embed" }

// ── fake provider ─────────────────────────────────────────────────────────────

type fakeProvider struct {
	response string
	err      error
}

func (f *fakeProvider) Complete(_ context.Context, _ []*types.Message) (*types.Message, error) {
	if f.err != nil {
		return nil, f.err
	}
	return types.NewAssistantMessage(f.response), nil
}

func (f *fakeProvider) StreamCompletion(_ context.Context, _ []*types.Message) (<-chan *llm.StreamChunk, error) {
	ch := make(chan *llm.StreamChunk)
	close(ch)
	return ch, nil
}

func (f *fakeProvider) GetModelInfo() *types.ModelInfo { return &types.ModelInfo{Name: "fake"} }
func (f *fakeProvider) GetModel() string               { return "fake" }
func (f *fakeProvider) GetBaseURL() string             { return "" }
func (f *fakeProvider) GetAPIKey() string              { return "" }
func (f *fakeProvider) AnalyzeDocument(_ context.Context, _ []byte, _ string, _ string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.response, nil
}

// ── sentinel errors ──────────────────────────────────────────────────────────

var errEmbed = errors.New("embed error")
