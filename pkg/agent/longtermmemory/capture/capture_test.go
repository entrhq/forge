package capture

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/entrhq/forge/pkg/agent/longtermmemory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/types"
)

// ── fakes ────────────────────────────────────────────────────────────────────

type fakeProvider struct {
	response string
	err      error
}

func (f *fakeProvider) StreamCompletion(_ context.Context, _ []*types.Message) (<-chan *llm.StreamChunk, error) {
	ch := make(chan *llm.StreamChunk)
	close(ch)
	return ch, nil
}

func (f *fakeProvider) Complete(_ context.Context, _ []*types.Message) (*types.Message, error) {
	if f.err != nil {
		return nil, f.err
	}
	return types.NewAssistantMessage(f.response), nil
}

func (f *fakeProvider) GetModelInfo() *types.ModelInfo { return &types.ModelInfo{Name: "fake"} }
func (f *fakeProvider) GetModel() string               { return "fake" }
func (f *fakeProvider) GetBaseURL() string             { return "" }
func (f *fakeProvider) GetAPIKey() string              { return "" }

type fakeStore struct {
	mu      sync.Mutex
	written []*longtermmemory.MemoryFile
	byScope map[longtermmemory.Scope][]*longtermmemory.MemoryFile
	listErr error
}

func newFakeStore() *fakeStore {
	return &fakeStore{byScope: make(map[longtermmemory.Scope][]*longtermmemory.MemoryFile)}
}

func (s *fakeStore) Write(_ context.Context, m *longtermmemory.MemoryFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.written = append(s.written, m)
	s.byScope[m.Meta.Scope] = append(s.byScope[m.Meta.Scope], m)
	return nil
}

func (s *fakeStore) Read(_ context.Context, id string) (*longtermmemory.MemoryFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.written {
		if m.Meta.ID == id {
			return m, nil
		}
	}
	return nil, longtermmemory.ErrNotFound
}

func (s *fakeStore) List(_ context.Context) ([]*longtermmemory.MemoryFile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*longtermmemory.MemoryFile, len(s.written))
	copy(out, s.written)
	return out, nil
}

func (s *fakeStore) ListByScope(_ context.Context, scope longtermmemory.Scope) ([]*longtermmemory.MemoryFile, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	mems := s.byScope[scope]
	out := make([]*longtermmemory.MemoryFile, len(mems))
	copy(out, mems)
	return out, nil
}

func (s *fakeStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.written)
}

func makeMem(id string, scope longtermmemory.Scope, cat longtermmemory.Category, content string) *longtermmemory.MemoryFile {
	now := time.Now().UTC()
	return &longtermmemory.MemoryFile{
		Meta: longtermmemory.MemoryMeta{
			ID:        id,
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
			Scope:     scope,
			Category:  cat,
			SessionID: "sess_test",
			Trigger:   longtermmemory.TriggerCadence,
		},
		Content: content,
	}
}

// toolBlock builds a fake tool-call XML block without embedding literal XML
// tags in a way that would break CDATA boundaries during transmission.
func toolBlock(name string) string {
	open := "<" + "tool" + ">"
	close := "<" + "/tool" + ">"
	inner := "<" + "tool_name" + ">" + name + "<" + "/tool_name" + ">"
	_ = open
	_ = close
	_ = inner
	// Use raw string building via join to avoid encoder issues.
	return strings.Join([]string{"<tool>", "<tool_name>" + name + "</tool_name>", "</tool>"}, "\n")
}

// toolResultBlock builds a fake tool_result XML block.
func toolResultBlock(body string) string {
	return strings.Join([]string{"<tool_result>", body, "</tool_result>"}, "\n")
}

// ── StripToolContent ─────────────────────────────────────────────────────────

func TestStripToolContent_BasicFiltering(t *testing.T) {
	msgs := []*types.Message{
		types.NewSystemMessage("You are helpful."),
		types.NewUserMessage("Please write a function."),
		types.NewAssistantMessage("Sure, here it is."),
		types.NewToolMessage(`{"result":"ok"}`),
	}
	out := StripToolContent(msgs)
	if len(out) != 2 {
		t.Fatalf("expected 2 messages (user+assistant), got %d", len(out))
	}
	if out[0].Role != "user" || out[1].Role != "assistant" {
		t.Errorf("unexpected roles: %v, %v", out[0].Role, out[1].Role)
	}
}

func TestStripToolContent_RemovesToolCallBlocks(t *testing.T) {
	block := toolBlock("read_file")
	raw := "I'll check that.\n" + block + "\nDone."
	msg := types.NewAssistantMessage(raw)
	out := StripToolContent([]*types.Message{msg})
	if len(out) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out))
	}
	if strings.Contains(out[0].Content, "tool_name") {
		t.Errorf("expected tool block stripped, still present in: %q", out[0].Content)
	}
	if !strings.Contains(out[0].Content, "I'll check that") {
		t.Errorf("expected prose preserved, got: %q", out[0].Content)
	}
}

func TestStripToolContent_RemovesToolResultBlocks(t *testing.T) {
	block := toolResultBlock("some output")
	raw := "Here:\n" + block + "\nHope that helps."
	msg := types.NewAssistantMessage(raw)
	out := StripToolContent([]*types.Message{msg})
	if len(out) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out))
	}
	if strings.Contains(out[0].Content, "tool_result") {
		t.Errorf("expected tool_result block stripped, got: %q", out[0].Content)
	}
	if !strings.Contains(out[0].Content, "Hope that helps") {
		t.Errorf("expected trailing prose preserved, got: %q", out[0].Content)
	}
}

func TestStripToolContent_PureToolCallDropped(t *testing.T) {
	// A message consisting only of a tool call block produces empty text and
	// should be omitted entirely from the output.
	raw := toolBlock("execute_command")
	msg := types.NewAssistantMessage(raw)
	out := StripToolContent([]*types.Message{msg})
	if len(out) != 0 {
		t.Errorf("expected 0 messages for pure tool call, got %d", len(out))
	}
}

func TestStripToolContent_NilInput(t *testing.T) {
	out := StripToolContent(nil)
	if len(out) != 0 {
		t.Errorf("expected empty output for nil input, got %d", len(out))
	}
}

func TestStripToolContent_PreservesRoles(t *testing.T) {
	msgs := []*types.Message{
		types.NewUserMessage("Hello"),
		types.NewAssistantMessage("World"),
	}
	out := StripToolContent(msgs)
	if len(out) != 2 {
		t.Fatalf("expected 2, got %d", len(out))
	}
	if out[0].Role != "user" || out[1].Role != "assistant" {
		t.Errorf("roles not preserved: %v, %v", out[0].Role, out[1].Role)
	}
}

// ── buildClassifierPrompt ────────────────────────────────────────────────────

func TestBuildClassifierPrompt_ContainsMessages(t *testing.T) {
	event := TriggerEvent{
		Kind: TriggerKindTurn,
		Messages: []ConversationMessage{
			{Role: "user", Content: "I prefer tabs."},
			{Role: "assistant", Content: "Noted."},
		},
		SessionID: "sess_1",
	}
	prompt := buildClassifierPrompt(event, nil)
	if !strings.Contains(prompt, "I prefer tabs.") {
		t.Errorf("user message missing from prompt")
	}
	if !strings.Contains(prompt, "Noted.") {
		t.Errorf("assistant message missing from prompt")
	}
}

func TestBuildClassifierPrompt_NoExistingMemoriesSection(t *testing.T) {
	event := TriggerEvent{
		Kind:      TriggerKindTurn,
		Messages:  []ConversationMessage{{Role: "user", Content: "hi"}},
		SessionID: "sess_1",
	}
	prompt := buildClassifierPrompt(event, nil)
	if strings.Contains(prompt, "EXISTING MEMORIES") {
		t.Errorf("expected no EXISTING MEMORIES section for nil existing list")
	}
}

func TestBuildClassifierPrompt_WithExistingMemories(t *testing.T) {
	existing := []*longtermmemory.MemoryFile{
		makeMem("mem_abc", longtermmemory.ScopeRepo, longtermmemory.CategoryCodingPreferences, "Use gofmt always."),
		makeMem("mem_xyz", longtermmemory.ScopeUser, longtermmemory.CategoryUserFacts, "Prefers minimal diffs."),
	}
	event := TriggerEvent{
		Kind:      TriggerKindCompaction,
		Messages:  []ConversationMessage{{Role: "user", Content: "ok"}},
		SessionID: "sess_2",
	}
	prompt := buildClassifierPrompt(event, existing)
	if !strings.Contains(prompt, "EXISTING MEMORIES") {
		t.Errorf("expected EXISTING MEMORIES section")
	}
	if !strings.Contains(prompt, "mem_abc") || !strings.Contains(prompt, "mem_xyz") {
		t.Errorf("expected both IDs in prompt")
	}
	if !strings.Contains(prompt, "Use gofmt always.") {
		t.Errorf("expected first-line summary for mem_abc")
	}
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello world", "Hello world"},
		{"First\nSecond", "First"},
		{"\nActual first", "Actual first"},
		{"", ""},
		{"   \n  \n", ""},
	}
	for _, tt := range tests {
		got := firstLine(tt.input)
		if got != tt.want {
			t.Errorf("firstLine(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── triggerToMemoryTrigger ───────────────────────────────────────────────────

func TestTriggerToMemoryTrigger(t *testing.T) {
	tests := []struct {
		kind TriggerKind
		want longtermmemory.Trigger
	}{
		{TriggerKindTurn, longtermmemory.TriggerCadence},
		{TriggerKindCompaction, longtermmemory.TriggerCompaction},
		{TriggerKind("unknown"), longtermmemory.TriggerCadence},
	}
	for _, tt := range tests {
		got := triggerToMemoryTrigger(tt.kind)
		if got != tt.want {
			t.Errorf("triggerToMemoryTrigger(%q) = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

// ── Classifier ───────────────────────────────────────────────────────────────

func TestClassifier_ValidMemory(t *testing.T) {
	mems := []classifiedMemory{
		{Content: "Use errors.As.", Scope: longtermmemory.ScopeUser, Category: longtermmemory.CategoryCodingPreferences},
	}
	raw, _ := json.Marshal(mems)
	c := NewClassifier(&fakeProvider{response: string(raw)}, "", newFakeStore(), nil)

	got, err := c.Classify(context.Background(), TriggerEvent{
		Kind:      TriggerKindTurn,
		Messages:  []ConversationMessage{{Role: "user", Content: "always use errors.As"}},
		SessionID: "sess_1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(got))
	}
	if got[0].Meta.Scope != longtermmemory.ScopeUser {
		t.Errorf("scope mismatch: %q", got[0].Meta.Scope)
	}
	if got[0].Meta.Trigger != longtermmemory.TriggerCadence {
		t.Errorf("trigger mismatch: %q", got[0].Meta.Trigger)
	}
	if got[0].Meta.ID == "" {
		t.Errorf("expected non-empty memory ID")
	}
}

func TestClassifier_EmptyArrayResponse(t *testing.T) {
	c := NewClassifier(&fakeProvider{response: "[]"}, "", newFakeStore(), nil)
	got, err := c.Classify(context.Background(), TriggerEvent{
		Kind:      TriggerKindTurn,
		Messages:  []ConversationMessage{{Role: "user", Content: "nothing memorable"}},
		SessionID: "sess_1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 memories, got %d", len(got))
	}
}

func TestClassifier_NonJSONResponse(t *testing.T) {
	c := NewClassifier(&fakeProvider{response: "nothing to save"}, "", newFakeStore(), nil)
	got, err := c.Classify(context.Background(), TriggerEvent{
		Kind:      TriggerKindTurn,
		Messages:  []ConversationMessage{{Role: "user", Content: "test"}},
		SessionID: "sess_1",
	})
	if err != nil {
		t.Fatalf("non-JSON should be treated as empty, not error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for non-JSON response, got %v", got)
	}
}

func TestClassifier_LLMError(t *testing.T) {
	c := NewClassifier(&fakeProvider{err: context.DeadlineExceeded}, "", newFakeStore(), nil)
	_, err := c.Classify(context.Background(), TriggerEvent{
		Kind:      TriggerKindTurn,
		Messages:  []ConversationMessage{{Role: "user", Content: "test"}},
		SessionID: "sess_1",
	})
	if err == nil {
		t.Fatal("expected error from LLM, got nil")
	}
}

func TestClassifier_EmptyMessages(t *testing.T) {
	c := NewClassifier(&fakeProvider{response: "[]"}, "", newFakeStore(), nil)
	got, err := c.Classify(context.Background(), TriggerEvent{Kind: TriggerKindTurn, SessionID: "s"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for empty messages, got %v", got)
	}
}

func TestClassifier_RejectsUnknownScope(t *testing.T) {
	mems := []classifiedMemory{
		{Content: "bad scope", Scope: "invalid", Category: longtermmemory.CategoryCodingPreferences},
		{Content: "good memory", Scope: longtermmemory.ScopeUser, Category: longtermmemory.CategoryUserFacts},
	}
	raw, _ := json.Marshal(mems)
	c := NewClassifier(&fakeProvider{response: string(raw)}, "", newFakeStore(), nil)

	got, err := c.Classify(context.Background(), TriggerEvent{
		Kind:      TriggerKindTurn,
		Messages:  []ConversationMessage{{Role: "user", Content: "test"}},
		SessionID: "sess_1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Content != "good memory" {
		t.Errorf("expected only valid-scope memory, got %v", got)
	}
}

func TestClassifier_RejectsUnknownCategory(t *testing.T) {
	mems := []classifiedMemory{
		{Content: "bad cat", Scope: longtermmemory.ScopeRepo, Category: "gibberish"},
	}
	raw, _ := json.Marshal(mems)
	c := NewClassifier(&fakeProvider{response: string(raw)}, "", newFakeStore(), nil)

	got, err := c.Classify(context.Background(), TriggerEvent{
		Kind:      TriggerKindTurn,
		Messages:  []ConversationMessage{{Role: "user", Content: "test"}},
		SessionID: "sess_1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 memories for unknown category, got %d", len(got))
	}
}

func TestClassifier_SupersedesResolution(t *testing.T) {
	store := newFakeStore()
	predecessor := makeMem("mem_old", longtermmemory.ScopeUser, longtermmemory.CategoryUserFacts, "Old fact.")
	_ = store.Write(context.Background(), predecessor)

	mems := []classifiedMemory{
		{Content: "Updated fact.", Scope: longtermmemory.ScopeUser, Category: longtermmemory.CategoryUserFacts, Supersedes: "mem_old"},
	}
	raw, _ := json.Marshal(mems)
	c := NewClassifier(&fakeProvider{response: string(raw)}, "", store, nil)

	got, err := c.Classify(context.Background(), TriggerEvent{
		Kind:      TriggerKindTurn,
		Messages:  []ConversationMessage{{Role: "user", Content: "correction"}},
		SessionID: "sess_1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(got))
	}
	if got[0].Meta.Supersedes == nil || *got[0].Meta.Supersedes != "mem_old" {
		t.Errorf("expected Supersedes=mem_old, got %v", got[0].Meta.Supersedes)
	}
	if got[0].Meta.Version != 2 {
		t.Errorf("expected version 2, got %d", got[0].Meta.Version)
	}
}

func TestClassifier_DanglingSupersedesCleared(t *testing.T) {
	mems := []classifiedMemory{
		{Content: "New memory.", Scope: longtermmemory.ScopeRepo, Category: longtermmemory.CategoryPatterns, Supersedes: "mem_ghost"},
	}
	raw, _ := json.Marshal(mems)
	c := NewClassifier(&fakeProvider{response: string(raw)}, "", newFakeStore(), nil)

	got, err := c.Classify(context.Background(), TriggerEvent{
		Kind:      TriggerKindTurn,
		Messages:  []ConversationMessage{{Role: "user", Content: "test"}},
		SessionID: "sess_1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(got))
	}
	if got[0].Meta.Supersedes != nil {
		t.Errorf("expected dangling supersedes cleared, got %v", *got[0].Meta.Supersedes)
	}
	if got[0].Meta.Version != 1 {
		t.Errorf("expected version 1, got %d", got[0].Meta.Version)
	}
}

func TestClassifier_CompactionTriggerMaps(t *testing.T) {
	mems := []classifiedMemory{
		{Content: "arch decision", Scope: longtermmemory.ScopeRepo, Category: longtermmemory.CategoryArchitecturalDecisions},
	}
	raw, _ := json.Marshal(mems)
	c := NewClassifier(&fakeProvider{response: string(raw)}, "", newFakeStore(), nil)

	got, err := c.Classify(context.Background(), TriggerEvent{
		Kind:      TriggerKindCompaction,
		Messages:  []ConversationMessage{{Role: "user", Content: "decision"}},
		SessionID: "sess_1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(got))
	}
	if got[0].Meta.Trigger != longtermmemory.TriggerCompaction {
		t.Errorf("expected TriggerCompaction, got %q", got[0].Meta.Trigger)
	}
}

// ── Pipeline ─────────────────────────────────────────────────────────────────

func TestPipeline_WritesMemoriesToStore(t *testing.T) {
	mems := []classifiedMemory{
		{Content: "pipeline memory", Scope: longtermmemory.ScopeUser, Category: longtermmemory.CategoryUserFacts},
	}
	raw, _ := json.Marshal(mems)
	store := newFakeStore()

	var mu sync.Mutex
	var rebuilt bool
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := NewPipeline(&fakeProvider{response: string(raw)}, "", store, func() {
		mu.Lock()
		rebuilt = true
		mu.Unlock()
	}, nil)
	p.Start(ctx)

	p.Enqueue(TriggerEvent{
		Kind:      TriggerKindTurn,
		Messages:  []ConversationMessage{{Role: "user", Content: "test"}},
		SessionID: "sess_1",
	})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if store.count() >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if store.count() != 1 {
		t.Errorf("expected 1 written memory, got %d", store.count())
	}
	mu.Lock()
	called := rebuilt
	mu.Unlock()
	if !called {
		t.Errorf("expected rebuildFn called after write")
	}
}

func TestPipeline_RebuildNotCalledForEmptyResult(t *testing.T) {
	store := newFakeStore()
	var mu sync.Mutex
	var rebuilt bool

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := NewPipeline(&fakeProvider{response: "[]"}, "", store, func() {
		mu.Lock()
		rebuilt = true
		mu.Unlock()
	}, nil)
	p.Start(ctx)

	p.Enqueue(TriggerEvent{
		Kind:      TriggerKindTurn,
		Messages:  []ConversationMessage{{Role: "user", Content: "nothing"}},
		SessionID: "sess_1",
	})

	time.Sleep(150 * time.Millisecond)

	mu.Lock()
	called := rebuilt
	mu.Unlock()
	if called {
		t.Errorf("expected rebuildFn NOT called when nothing written")
	}
}

func TestPipeline_EnqueueIsNonBlocking(t *testing.T) {
	// Do NOT start the pipeline so the buffer fills; excess events must be dropped.
	p := NewPipeline(&fakeProvider{response: "[]"}, "", newFakeStore(), nil, nil)

	done := make(chan struct{})
	go func() {
		for i := 0; i < triggerBufferSize*3; i++ {
			p.Enqueue(TriggerEvent{
				Kind:      TriggerKindTurn,
				Messages:  []ConversationMessage{{Role: "user", Content: "hi"}},
				SessionID: "sess",
			})
		}
		close(done)
	}()

	select {
	case <-done:
		// pass
	case <-time.After(2 * time.Second):
		t.Fatal("Enqueue blocked when pipeline buffer was full")
	}
}

func TestPipeline_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := NewPipeline(&fakeProvider{response: "[]"}, "", newFakeStore(), nil, nil)
	p.Start(ctx)
	cancel()
	time.Sleep(50 * time.Millisecond)
	// Enqueue after cancel must not panic or deadlock.
	p.Enqueue(TriggerEvent{
		Kind:      TriggerKindTurn,
		Messages:  []ConversationMessage{{Role: "user", Content: "after cancel"}},
		SessionID: "sess",
	})
}

// ── Observer ─────────────────────────────────────────────────────────────────

func TestObserver_OnTurnComplete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewPipeline(&fakeProvider{response: "[]"}, "", newFakeStore(), nil, nil)
	p.Start(ctx)
	obs := NewObserver(p)

	msgs := []*types.Message{
		types.NewUserMessage("I prefer snake_case."),
		types.NewAssistantMessage("Noted."),
	}
	obs.OnTurnComplete(msgs, "sess_1")
	time.Sleep(50 * time.Millisecond)
	// No panic = pass.
}

func TestObserver_OnCompaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewPipeline(&fakeProvider{response: "[]"}, "", newFakeStore(), nil, nil)
	p.Start(ctx)
	obs := NewObserver(p)

	msgs := []*types.Message{
		types.NewUserMessage("We chose hexagonal architecture."),
		types.NewAssistantMessage("Understood."),
	}
	obs.OnCompaction(msgs, "sess_compaction")
	time.Sleep(50 * time.Millisecond)
}

func TestObserver_SkipsSystemAndToolMessages(t *testing.T) {
	store := newFakeStore()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewPipeline(&fakeProvider{response: "[]"}, "", store, nil, nil)
	p.Start(ctx)
	obs := NewObserver(p)

	msgs := []*types.Message{
		types.NewSystemMessage("System prompt."),
		types.NewToolMessage(`{"result":"ok"}`),
	}
	obs.OnTurnComplete(msgs, "sess_1")
	obs.OnCompaction(msgs, "sess_1")
	time.Sleep(100 * time.Millisecond)

	if store.count() != 0 {
		t.Errorf("expected nothing written for system/tool-only messages, got %d", store.count())
	}
}
