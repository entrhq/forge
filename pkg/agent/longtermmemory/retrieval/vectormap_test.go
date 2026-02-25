package retrieval

import (
	"sync"
	"testing"
)

func TestVectorMap_Empty(t *testing.T) {
	vm := NewVectorMap()
	if vm.Len() != 0 {
		t.Errorf("Len() = %d, want 0", vm.Len())
	}
	if vm.Lookup("any") != nil {
		t.Error("Lookup on empty map should return nil")
	}
	got := vm.TopK([]float32{1, 0}, 5)
	if len(got) != 0 {
		t.Errorf("TopK on empty map returned %d entries", len(got))
	}
}

func TestVectorMap_SwapAndLookup(t *testing.T) {
	vm := NewVectorMap()

	fa := makeMemoryFile("a", "content a", "")
	fb := makeMemoryFile("b", "content b", "")

	entries := []MemoryVector{
		{Memory: fa, Vector: Normalise([]float32{1, 0})},
		{Memory: fb, Vector: Normalise([]float32{0, 1})},
	}
	vm.Swap(entries)

	if vm.Len() != 2 {
		t.Errorf("Len() = %d, want 2", vm.Len())
	}

	ea := vm.Lookup("a")
	if ea == nil {
		t.Fatal("Lookup('a') returned nil")
	}
	if ea.Memory.Meta.ID != "a" {
		t.Errorf("ID = %q, want %q", ea.Memory.Meta.ID, "a")
	}

	if vm.Lookup("missing") != nil {
		t.Error("Lookup('missing') should return nil")
	}
}

func TestVectorMap_SwapClearsOldEntries(t *testing.T) {
	vm := NewVectorMap()
	old := []MemoryVector{
		{Memory: makeMemoryFile("old", "old content", ""), Vector: Normalise([]float32{1, 0})},
	}
	vm.Swap(old)

	newEntries := []MemoryVector{
		{Memory: makeMemoryFile("new", "new content", ""), Vector: Normalise([]float32{0, 1})},
	}
	vm.Swap(newEntries)

	if vm.Lookup("old") != nil {
		t.Error("old entry should be gone after Swap")
	}
	if vm.Lookup("new") == nil {
		t.Error("new entry should be present after Swap")
	}
	if vm.Len() != 1 {
		t.Errorf("Len() = %d, want 1", vm.Len())
	}
}

func TestVectorMap_SwapNilClearsEntries(t *testing.T) {
	vm := NewVectorMap()
	vm.Swap([]MemoryVector{
		{Memory: makeMemoryFile("x", "x", ""), Vector: []float32{1}},
	})
	vm.Swap(nil)
	if vm.Len() != 0 {
		t.Errorf("Len() = %d after nil Swap, want 0", vm.Len())
	}
}

func TestVectorMap_TopK(t *testing.T) {
	vm := NewVectorMap()
	fa := makeMemoryFile("a", "alpha", "")
	fb := makeMemoryFile("b", "beta", "")
	fc := makeMemoryFile("c", "gamma", "")

	vm.Swap([]MemoryVector{
		{Memory: fa, Vector: Normalise([]float32{1, 0})},
		{Memory: fb, Vector: Normalise([]float32{0, 1})},
		{Memory: fc, Vector: Normalise([]float32{1, 1})},
	})

	query := Normalise([]float32{1, 0}) // most similar to "a"
	got := vm.TopK(query, 2)
	if len(got) != 2 {
		t.Fatalf("TopK(2) returned %d results", len(got))
	}
	if got[0].Memory.Meta.ID != "a" {
		t.Errorf("top result ID = %q, want %q", got[0].Memory.Meta.ID, "a")
	}
}

func TestVectorMap_ConcurrentAccess(t *testing.T) {
	vm := NewVectorMap()
	entries := []MemoryVector{
		{Memory: makeMemoryFile("x", "x", ""), Vector: Normalise([]float32{1, 0})},
	}
	vm.Swap(entries)

	var wg sync.WaitGroup
	query := Normalise([]float32{1, 0})

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vm.TopK(query, 1)
			vm.Lookup("x")
			vm.Len()
		}()
	}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			vm.Swap(entries)
		}()
	}
	wg.Wait()
}
