package longtermmemory

import (
	"crypto/rand"
	"fmt"
)

// NewMemoryID generates a new unique memory identifier.
func NewMemoryID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// In a real panic scenario, this means the OS crypto source failed,
		// which is a critical unrecoverable application state.
		panic(fmt.Errorf("crypto/rand failed: %w", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("mem_%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
