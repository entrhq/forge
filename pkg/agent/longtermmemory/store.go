package longtermmemory

import "context"

// MemoryStore is the read/write interface for persisted memory files.
type MemoryStore interface {
	Write(ctx context.Context, m *MemoryFile) error
	Read(ctx context.Context, id string) (*MemoryFile, error)
	List(ctx context.Context) ([]*MemoryFile, error)
	ListByScope(ctx context.Context, scope Scope) ([]*MemoryFile, error)
}
