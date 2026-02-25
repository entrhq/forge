package retrieval

import (
	"sync"

	"github.com/entrhq/forge/pkg/agent/longtermmemory"
)

// MemoryVector pairs a memory file with its pre-computed embedding.
type MemoryVector struct {
	Memory *longtermmemory.MemoryFile
	Vector []float32
}

// VectorMap is a thread-safe in-memory index that maps memory IDs to their
// corresponding embedding vectors. Reads are served under a shared read-lock;
// the entire map is swapped atomically on rebuild.
type VectorMap struct {
	mu      sync.RWMutex
	entries []MemoryVector
	byID    map[string]*MemoryVector
}

// NewVectorMap allocates an empty VectorMap.
func NewVectorMap() *VectorMap {
	return &VectorMap{
		byID: make(map[string]*MemoryVector),
	}
}

// Swap atomically replaces the entire index with a new set of entries.
// Called by the builder goroutine after a full re-embed cycle.
func (vm *VectorMap) Swap(entries []MemoryVector) {
	byID := make(map[string]*MemoryVector, len(entries))
	for i := range entries {
		e := &entries[i]
		byID[e.Memory.Meta.ID] = e
	}
	vm.mu.Lock()
	vm.entries = entries
	vm.byID = byID
	vm.mu.Unlock()
}

// TopK returns up to k memories whose embedding is closest to the query
// vector, sorted by descending cosine similarity.
func (vm *VectorMap) TopK(query []float32, k int) []MemoryVector {
	vm.mu.RLock()
	entries := vm.entries
	vm.mu.RUnlock()
	return cosineTopK(entries, query, k)
}

// Lookup returns the MemoryVector for a given memory ID, or nil if absent.
func (vm *VectorMap) Lookup(id string) *MemoryVector {
	vm.mu.RLock()
	e := vm.byID[id]
	vm.mu.RUnlock()
	return e
}

// Len returns the number of entries in the index.
func (vm *VectorMap) Len() int {
	vm.mu.RLock()
	n := len(vm.entries)
	vm.mu.RUnlock()
	return n
}
