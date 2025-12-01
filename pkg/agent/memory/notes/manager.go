package notes

import (
	"fmt"
	"sort"
	"sync"
)

// Manager handles CRUD operations and search for scratchpad notes.
// All operations are thread-safe and session-scoped (in-memory only).
type Manager struct {
	notes map[string]*Note // Map of note ID to note
	mu    sync.RWMutex     // Read-write mutex for thread safety
}

// NewManager creates a new notes manager
func NewManager() *Manager {
	return &Manager{
		notes: make(map[string]*Note),
	}
}

// Add creates a new note with the given content and tags
func (m *Manager) Add(content string, tags []string) (*Note, error) {
	note, err := NewNote(content, tags)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.notes[note.ID] = note
	return note, nil
}

// Get retrieves a note by ID
func (m *Manager) Get(id string) (*Note, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	note, exists := m.notes[id]
	if !exists {
		return nil, fmt.Errorf("note not found: %s", id)
	}

	return note, nil
}

// Update modifies an existing note's content and/or tags
func (m *Manager) Update(id string, content *string, tags []string) (*Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	note, exists := m.notes[id]
	if !exists {
		return nil, fmt.Errorf("note not found: %s", id)
	}

	if err := note.Update(content, tags); err != nil {
		return nil, err
	}

	return note, nil
}

// Delete removes a note by ID
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.notes[id]; !exists {
		return fmt.Errorf("note not found: %s", id)
	}

	delete(m.notes, id)
	return nil
}

// Scratch marks a note as addressed/obsolete
func (m *Manager) Scratch(id string) (*Note, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	note, exists := m.notes[id]
	if !exists {
		return nil, fmt.Errorf("note not found: %s", id)
	}

	note.Scratch()
	return note, nil
}

// ListOptions configures the List operation
type ListOptions struct {
	Tag             string // Filter by single tag (optional)
	IncludeScratched bool   // Include scratched notes (default: false)
	Limit           int    // Maximum notes to return (default: 10)
}

// List retrieves notes with optional filtering and limiting
func (m *Manager) List(opts ListOptions) []*Note {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Set default limit
	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	// Collect matching notes
	var result []*Note
	for _, note := range m.notes {
		// Skip scratched notes unless explicitly included
		if note.Scratched && !opts.IncludeScratched {
			continue
		}

		// Filter by tag if specified
		if opts.Tag != "" && !note.HasTag(opts.Tag) {
			continue
		}

		result = append(result, note)
	}

	// Sort by UpdatedAt (most recent first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	// Apply limit
	if len(result) > opts.Limit {
		result = result[:opts.Limit]
	}

	return result
}

// SearchOptions configures the Search operation
type SearchOptions struct {
	Query            string   // Text to search for in content (case-insensitive)
	Tags             []string // Tags that must all be present (AND logic)
	Limit            int      // Maximum notes to return (default: 10)
	IncludeScratched bool     // Include scratched notes (default: false)
}

// Search finds notes matching the query and/or tags
func (m *Manager) Search(opts SearchOptions) []*Note {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Set default limit
	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	// Collect matching notes (excludes scratched notes by default)
	var result []*Note
	for _, note := range m.notes {
		// Skip scratched notes unless explicitly included
		if note.Scratched && !opts.IncludeScratched {
			continue
		}

		// Check text match
		if !note.ContainsText(opts.Query) {
			continue
		}

		// Check tag match (AND logic)
		if !note.MatchesAllTags(opts.Tags) {
			continue
		}

		result = append(result, note)
	}

	// Sort by UpdatedAt (most recent first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})

	// Apply limit
	if len(result) > opts.Limit {
		result = result[:opts.Limit]
	}

	return result
}

// ListTags returns all unique tags currently in use across all notes
func (m *Manager) ListTags() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tagSet := make(map[string]bool)
	for _, note := range m.notes {
		// Only include tags from active (non-scratched) notes
		if !note.Scratched {
			for _, tag := range note.Tags {
				tagSet[tag] = true
			}
		}
	}

	// Convert to sorted slice
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	return tags
}

// Count returns the total number of notes (including scratched)
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.notes)
}

// CountActive returns the number of active (non-scratched) notes
func (m *Manager) CountActive() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, note := range m.notes {
		if !note.Scratched {
			count++
		}
	}
	return count
}

// Clear removes all notes from the manager
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.notes = make(map[string]*Note)
}
