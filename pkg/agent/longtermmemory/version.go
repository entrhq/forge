package longtermmemory

import (
	"context"
	"fmt"
	"time"
)

var timeNow = time.Now // injected for testability

// NewVersion creates a new MemoryFile representing a subsequent version of the given predecessor.
// The Related parameter slice is deep-copied to prevent underlying array leakage.
func NewVersion(predecessor *MemoryFile, sessionID string, trigger Trigger) *MemoryFile {
	now := timeNow()
	predecessorID := predecessor.Meta.ID
	
	related := make([]RelatedMemory, len(predecessor.Meta.Related))
	copy(related, predecessor.Meta.Related)

	return &MemoryFile{
		Meta: MemoryMeta{
			ID:         NewMemoryID(),
			CreatedAt:  now,
			UpdatedAt:  now,
			Version:    predecessor.Meta.Version + 1,
			Scope:      predecessor.Meta.Scope,
			Category:   predecessor.Meta.Category,
			Supersedes: &predecessorID,
			Related:    related,
			SessionID:  sessionID,
			Trigger:    trigger,
		},
	}
}

// VersionChain retrieves the chronological ancestry of the specified memory up to maxDepth.
func VersionChain(ctx context.Context, store MemoryStore, id string, maxDepth int) ([]*MemoryFile, error) {
	var chain []*MemoryFile
	current := id
	for i := 0; i < maxDepth && current != ""; i++ {
		m, err := store.Read(ctx, current)
		if err != nil {
			return chain, fmt.Errorf("longtermmemory: version chain read %s: %w", current, err)
		}
		chain = append([]*MemoryFile{m}, chain...)
		if m.Meta.Supersedes == nil {
			break
		}
		current = *m.Meta.Supersedes
	}
	return chain, nil
}

// LatestVersion traverses forward along a "supersedes" chain to find the most recent memory ID.
// It detects and immediately breaks topological cycles to prevent infinite loops.
func LatestVersion(ctx context.Context, store MemoryStore, id string) (string, error) {
	all, err := store.List(ctx)
	if err != nil {
		return id, err
	}
	successor := make(map[string]string, len(all))
	for _, m := range all {
		if m.Meta.Supersedes != nil {
			successor[*m.Meta.Supersedes] = m.Meta.ID
		}
	}
	current := id
	visited := make(map[string]bool)
	for !visited[current] {
		visited[current] = true
		
		next, ok := successor[current]
		if !ok {
			return current, nil
		}
		current = next
	}
	return current, nil
}
