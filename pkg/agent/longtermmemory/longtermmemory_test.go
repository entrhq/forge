package longtermmemory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestParseSerializeRoundTrip(t *testing.T) {
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	v1ID := "mem_v1"
	meta := MemoryMeta{
		ID:         "mem_test",
		CreatedAt:  now,
		UpdatedAt:  now,
		Version:    1,
		Scope:      ScopeRepo,
		Category:   CategoryCodingPreferences,
		SessionID:  "session_123",
		Trigger:    TriggerCadence,
		Supersedes: &v1ID,
		Related: []RelatedMemory{
			{ID: "mem_other", Relationship: RelationshipRelatesTo},
		},
	}

	m := &MemoryFile{
		Meta:    meta,
		Content: "This is a test memory.\nWith markdown.",
	}

	b, err := Serialize(m)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	parsed, err := Parse(b)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Meta.ID != m.Meta.ID {
		t.Errorf("Expected ID %s, got %s", m.Meta.ID, parsed.Meta.ID)
	}
	if parsed.Content != m.Content {
		t.Errorf("Expected Content %q, got %q", m.Content, parsed.Content)
	}
	if parsed.Meta.Supersedes == nil || *parsed.Meta.Supersedes != *m.Meta.Supersedes {
		t.Errorf("Expected Supersedes to match, got %v", parsed.Meta.Supersedes)
	}
	if len(parsed.Meta.Related) != 1 || parsed.Meta.Related[0].ID != "mem_other" {
		t.Errorf("Expected Related to round-trip, got %+v", parsed.Meta.Related)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		err  string
	}{
		{
			name: "missing delimiter",
			raw:  "just some text",
			err:  "missing front-matter delimiter",
		},
		{
			name: "unclosed block",
			raw:  "---\nfoo: bar\nno closing delimiter",
			err:  "unclosed front-matter block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.raw))
			if err == nil {
				t.Fatalf("Expected error %q, got none", tt.err)
			}
			if err.Error() != "longtermmemory: "+tt.err {
				t.Errorf("Expected error %q, got %q", tt.err, err.Error())
			}
		})
	}
}

func TestFileStore(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	userDir := filepath.Join(tmpDir, "user")

	fs, err := NewFileStore(repoDir, userDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	ctx := context.Background()

	// Test read not found
	_, err = fs.Read(ctx, "mem_nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Expected ErrNotFound, got %v", err)
	}

	// Test write and read repo scope
	now := time.Now().UTC()
	m1 := &MemoryFile{
		Meta: MemoryMeta{
			ID:        "mem_repo1",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
			Scope:     ScopeRepo,
			Category:  CategoryCodingPreferences,
		},
		Content: "repo memory",
	}

	if err := fs.Write(ctx, m1); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	readM1, err := fs.Read(ctx, "mem_repo1")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if readM1.Content != m1.Content {
		t.Errorf("Expected content %q, got %q", m1.Content, readM1.Content)
	}

	// Test overwrite
	err = fs.Write(ctx, m1)
	if err != ErrAlreadyExists {
		t.Errorf("Expected ErrAlreadyExists when writing to existing ID, got %v", err)
	}

	// Test write user scope
	m2 := &MemoryFile{
		Meta: MemoryMeta{
			ID:        "mem_user1",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
			Scope:     ScopeUser,
			Category:  CategoryCodingPreferences,
		},
		Content: "user memory",
	}
	if err := fs.Write(ctx, m2); err != nil {
		t.Fatalf("Write user memory failed: %v", err)
	}

	// Test list combines both scopes
	list, err := fs.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("Expected 2 memories in aggregated list, got %d", len(list))
	}

	// Test invalid ID traversal
	_, err = fs.pathForID("../test", ScopeRepo)
	if err == nil {
		t.Errorf("Expected error for path traversal ID, got nil")
	}

	// Test empty ID error
	_, err = fs.pathForID("", ScopeRepo)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("Expected error for empty ID, got %v", err)
	}

	// Write a corrupt file and test ListByScope
	err = os.WriteFile(filepath.Join(repoDir, "mem_corrupt.md"), []byte("corrupt"), 0o600)
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	list, err = fs.ListByScope(ctx, ScopeRepo)
	if err != nil {
		t.Fatalf("ListByScope failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("Expected 1 memory in list, skipped corrupt file, got %d", len(list))
	}
}

func TestVersionChain(t *testing.T) {
	tmpDir := t.TempDir()
	fs, _ := NewFileStore(filepath.Join(tmpDir, "repo"), filepath.Join(tmpDir, "user"))
	ctx := context.Background()

	// Initial time and ID
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	m1 := &MemoryFile{
		Meta: MemoryMeta{
			ID:        "mem_v1",
			CreatedAt: t1,
			UpdatedAt: t1,
			Version:   1,
			Scope:     ScopeRepo,
		},
		Content: "version 1",
	}

	v1ID := "mem_v1"
	m2 := &MemoryFile{
		Meta: MemoryMeta{
			ID:         "mem_v2",
			CreatedAt:  t2,
			UpdatedAt:  t2,
			Version:    2,
			Scope:      ScopeRepo,
			Supersedes: &v1ID,
		},
		Content: "version 2",
	}

	_ = fs.Write(ctx, m1)
	_ = fs.Write(ctx, m2)

	chain, err := VersionChain(ctx, fs, "mem_v2", 10)
	if err != nil {
		t.Fatalf("VersionChain failed: %v", err)
	}

	if len(chain) != 2 {
		t.Fatalf("Expected chain length 2, got %d", len(chain))
	}

	if chain[0].Meta.ID != "mem_v1" {
		t.Errorf("Expected first in chain to be mem_v1, got %s", chain[0].Meta.ID)
	}
	if chain[1].Meta.ID != "mem_v2" {
		t.Errorf("Expected second in chain to be mem_v2, got %s", chain[1].Meta.ID)
	}

	// Latest version test
	latest, err := LatestVersion(ctx, fs, "mem_v1")
	if err != nil {
		t.Fatalf("LatestVersion failed: %v", err)
	}
	if latest != "mem_v2" {
		t.Errorf("Expected latest to be mem_v2, got %s", latest)
	}
}

func TestCycleDetection(t *testing.T) {
	tmpDir := t.TempDir()
	fs, _ := NewFileStore(filepath.Join(tmpDir, "repo"), filepath.Join(tmpDir, "user"))
	ctx := context.Background()

	id1 := "mem_cycle1"
	id2 := "mem_cycle2"

	m1 := &MemoryFile{
		Meta: MemoryMeta{
			ID:         id1,
			Version:    1,
			Scope:      ScopeRepo,
			Supersedes: &id2,
		},
		Content: "node 1",
	}

	m2 := &MemoryFile{
		Meta: MemoryMeta{
			ID:         id2,
			Version:    2,
			Scope:      ScopeRepo,
			Supersedes: &id1,
		},
		Content: "node 2",
	}

	_ = fs.Write(ctx, m1)
	_ = fs.Write(ctx, m2)

	// Test LatestVersion cycle breaking
	latest, err := LatestVersion(ctx, fs, id1)
	if err != nil {
		t.Fatalf("LatestVersion failed on cycle: %v", err)
	}
	if latest != id1 && latest != id2 {
		t.Errorf("LatestVersion should return a valid node in the cycle, got %s", latest)
	}

	// Test VersionChain loop limit via maxDepth
	chain, err := VersionChain(ctx, fs, id1, 5)
	if err != nil {
		t.Fatalf("VersionChain failed on cycle: %v", err)
	}
	if len(chain) != 5 {
		t.Errorf("Expected chain length bounded to maxDepth 5, got %d", len(chain))
	}
}

func TestNewVersion(t *testing.T) {
	now := time.Now()
	timeNow = func() time.Time { return now }
	defer func() { timeNow = time.Now }()

	v1 := &MemoryFile{
		Meta: MemoryMeta{
			ID:       "mem_1",
			Version:  1,
			Scope:    ScopeUser,
			Category: CategoryUserFacts,
			Related:  []RelatedMemory{{ID: "mem_rel", Relationship: RelationshipRelatesTo}},
		},
	}

	v2 := NewVersion(v1, "session_new", TriggerCompaction)

	if *v2.Meta.Supersedes != "mem_1" {
		t.Errorf("Expected supersedes to be mem_1, got %v", *v2.Meta.Supersedes)
	}
	if v2.Meta.Version != 2 {
		t.Errorf("Expected version 2, got %d", v2.Meta.Version)
	}
	if len(v2.Meta.Related) != 1 || v2.Meta.Related[0].ID != "mem_rel" {
		t.Errorf("Expected related to copy over, got %+v", v2.Meta.Related)
	}
	if v2.Meta.Scope != ScopeUser {
		t.Errorf("Expected scope user, got %s", v2.Meta.Scope)
	}

	v1.Meta.Related[0].ID = "mutated"
	if v2.Meta.Related[0].ID == "mutated" {
		t.Errorf("NewVersion related slice was copied by reference, mutating predecessor affected new version")
	}
}

func TestNewMemoryID(t *testing.T) {
	id := NewMemoryID()
	if !strings.HasPrefix(id, "mem_") {
		t.Errorf("Expected memory ID to start with mem_, got %q", id)
	}
	if len(id) < 10 {
		t.Errorf("Expected memory ID length > 10, got %d", len(id))
	}
}

func TestConcurrentWrite(t *testing.T) {
	tmpDir := t.TempDir()
	fs, _ := NewFileStore(filepath.Join(tmpDir, "repo"), filepath.Join(tmpDir, "user"))
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			m := &MemoryFile{
				Meta: MemoryMeta{
					ID:      fmt.Sprintf("mem_conc_%d", i),
					Version: 1,
					Scope:   ScopeRepo,
				},
				Content: "concurrent write test",
			}
			_ = fs.Write(ctx, m)
		}(i)
	}
	wg.Wait()

	list, _ := fs.List(ctx)
	if len(list) != 20 {
		t.Errorf("Expected 20 files successfully written sequentially or concurrently, got %d", len(list))
	}
}
