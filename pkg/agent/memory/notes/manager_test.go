package notes

import (
	"strings"
	"testing"
	"time"
)

func TestManagerAdd(t *testing.T) {
	m := NewManager()

	tests := []struct {
		name        string
		content     string
		tags        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid note",
			content:     "Test note",
			tags:        []string{"test"},
			expectError: false,
		},
		{
			name:        "invalid content",
			content:     "",
			tags:        []string{"test"},
			expectError: true,
			errorMsg:    "content cannot be empty",
		},
		{
			name:        "invalid tags",
			content:     "Valid content",
			tags:        []string{},
			expectError: true,
			errorMsg:    "requires at least 1 tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := m.Add(tt.content, tt.tags)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if note == nil {
				t.Fatal("note should not be nil")
			}

			// Verify note was added
			retrieved, err := m.Get(note.ID)
			if err != nil {
				t.Errorf("failed to retrieve added note: %v", err)
			}

			if retrieved.Content != tt.content {
				t.Errorf("content mismatch: got %q, want %q", retrieved.Content, tt.content)
			}
		})
	}
}

func TestManagerGet(t *testing.T) {
	m := NewManager()

	note, err := m.Add("Test note", []string{"test"})
	if err != nil {
		t.Fatalf("failed to add note: %v", err)
	}

	t.Run("get existing note", func(t *testing.T) {
		retrieved, err := m.Get(note.ID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if retrieved.ID != note.ID {
			t.Errorf("ID mismatch: got %q, want %q", retrieved.ID, note.ID)
		}
	})

	t.Run("get non-existent note", func(t *testing.T) {
		_, err := m.Get("note_nonexistent")
		if err == nil {
			t.Error("expected error for non-existent note")
		}

		if !strings.Contains(err.Error(), "note not found") {
			t.Errorf("expected 'note not found' error, got %q", err.Error())
		}
	})
}

func TestManagerUpdate(t *testing.T) {
	m := NewManager()

	note, err := m.Add("Original content", []string{"original"})
	if err != nil {
		t.Fatalf("failed to add note: %v", err)
	}

	t.Run("update content", func(t *testing.T) {
		newContent := "Updated content"
		updated, err := m.Update(note.ID, &newContent, nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if updated.Content != newContent {
			t.Errorf("content not updated: got %q, want %q", updated.Content, newContent)
		}
	})

	t.Run("update tags", func(t *testing.T) {
		newTags := []string{"updated", "tags"}
		updated, err := m.Update(note.ID, nil, newTags)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(updated.Tags) != len(newTags) {
			t.Errorf("tags not updated: got %d tags, want %d", len(updated.Tags), len(newTags))
		}
	})

	t.Run("update non-existent note", func(t *testing.T) {
		content := "New content"
		_, err := m.Update("note_nonexistent", &content, nil)
		if err == nil {
			t.Error("expected error for non-existent note")
		}

		if !strings.Contains(err.Error(), "note not found") {
			t.Errorf("expected 'note not found' error, got %q", err.Error())
		}
	})

	t.Run("update with invalid data", func(t *testing.T) {
		emptyContent := ""
		_, err := m.Update(note.ID, &emptyContent, nil)
		if err == nil {
			t.Error("expected error for empty content")
		}
	})
}

func TestManagerDelete(t *testing.T) {
	m := NewManager()

	note, err := m.Add("Test note", []string{"test"})
	if err != nil {
		t.Fatalf("failed to add note: %v", err)
	}

	t.Run("delete existing note", func(t *testing.T) {
		err := m.Delete(note.ID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify note is deleted
		_, err = m.Get(note.ID)
		if err == nil {
			t.Error("note should be deleted")
		}
	})

	t.Run("delete non-existent note", func(t *testing.T) {
		err := m.Delete("note_nonexistent")
		if err == nil {
			t.Error("expected error for non-existent note")
		}

		if !strings.Contains(err.Error(), "note not found") {
			t.Errorf("expected 'note not found' error, got %q", err.Error())
		}
	})
}

func TestManagerScratch(t *testing.T) {
	m := NewManager()

	note, err := m.Add("Test note", []string{"test"})
	if err != nil {
		t.Fatalf("failed to add note: %v", err)
	}

	t.Run("scratch existing note", func(t *testing.T) {
		scratched, err := m.Scratch(note.ID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if !scratched.Scratched {
			t.Error("note should be scratched")
		}

		// Verify persistence
		retrieved, _ := m.Get(note.ID)
		if !retrieved.Scratched {
			t.Error("scratched state should persist")
		}
	})

	t.Run("scratch non-existent note", func(t *testing.T) {
		_, err := m.Scratch("note_nonexistent")
		if err == nil {
			t.Error("expected error for non-existent note")
		}

		if !strings.Contains(err.Error(), "note not found") {
			t.Errorf("expected 'note not found' error, got %q", err.Error())
		}
	})
}

func TestManagerList(t *testing.T) {
	setupNotes := func() *Manager {
		m := NewManager()
		m.Add("Note 1", []string{"alpha", "beta"})
		time.Sleep(time.Millisecond) // Ensure unique IDs
		m.Add("Note 2", []string{"alpha"})
		time.Sleep(time.Millisecond)
		m.Add("Note 3", []string{"gamma"})
		time.Sleep(time.Millisecond)
		scratched, _ := m.Add("Note 4", []string{"alpha"})
		m.Scratch(scratched.ID)
		return m
	}

	t.Run("list all active notes", func(t *testing.T) {
		m := setupNotes()
		notes := m.List(ListOptions{})
		if len(notes) != 3 {
			t.Errorf("expected 3 active notes, got %d", len(notes))
		}
	})

	t.Run("list with scratched included", func(t *testing.T) {
		m := setupNotes()
		notes := m.List(ListOptions{IncludeScratched: true})
		if len(notes) != 4 {
			t.Errorf("expected 4 notes including scratched, got %d", len(notes))
		}
	})

	t.Run("list by tag", func(t *testing.T) {
		m := setupNotes()
		notes := m.List(ListOptions{Tag: "alpha"})
		if len(notes) != 2 {
			t.Errorf("expected 2 notes with tag 'alpha', got %d", len(notes))
		}
	})

	t.Run("list with limit", func(t *testing.T) {
		m := setupNotes()
		notes := m.List(ListOptions{Limit: 2})
		if len(notes) != 2 {
			t.Errorf("expected 2 notes with limit, got %d", len(notes))
		}
	})

	t.Run("list sorted by recency", func(t *testing.T) {
		m := setupNotes()
		notes := m.List(ListOptions{})
		for i := 1; i < len(notes); i++ {
			if notes[i].UpdatedAt.After(notes[i-1].UpdatedAt) {
				t.Error("notes should be sorted by recency (newest first)")
			}
		}
	})
}

func TestManagerSearch(t *testing.T) {
	setupNotes := func() *Manager {
		m := NewManager()
		m.Add("The quick brown fox", []string{"animal", "test"})
		time.Sleep(time.Millisecond)
		m.Add("The lazy dog", []string{"animal"})
		time.Sleep(time.Millisecond)
		m.Add("Random content", []string{"test"})
		time.Sleep(time.Millisecond)
		scratched, _ := m.Add("Quick fox content", []string{"animal"})
		m.Scratch(scratched.ID)
		return m
	}

	t.Run("search by text", func(t *testing.T) {
		m := setupNotes()
		notes := m.Search(SearchOptions{Query: "quick"})
		if len(notes) != 1 {
			t.Errorf("expected 1 note with 'quick', got %d", len(notes))
		}
	})

	t.Run("search by tags", func(t *testing.T) {
		m := setupNotes()
		notes := m.Search(SearchOptions{Tags: []string{"animal"}})
		if len(notes) != 2 {
			t.Errorf("expected 2 notes with tag 'animal' (excluding scratched), got %d", len(notes))
		}
	})

	t.Run("search by text and tags", func(t *testing.T) {
		m := setupNotes()
		notes := m.Search(SearchOptions{
			Query: "fox",
			Tags:  []string{"animal"},
		})
		if len(notes) != 1 {
			t.Errorf("expected 1 note matching both criteria, got %d", len(notes))
		}
	})

	t.Run("search with limit", func(t *testing.T) {
		m := setupNotes()
		notes := m.Search(SearchOptions{Limit: 1})
		if len(notes) != 1 {
			t.Errorf("expected 1 note with limit, got %d", len(notes))
		}
	})

	t.Run("search excludes scratched", func(t *testing.T) {
		m := setupNotes()
		notes := m.Search(SearchOptions{Query: "quick"})
		for _, note := range notes {
			if note.Scratched {
				t.Error("search should exclude scratched notes")
			}
		}
	})
}

func TestManagerListTags(t *testing.T) {
	m := NewManager()

	// Add test notes
	m.Add("Note 1", []string{"alpha", "beta"})
	time.Sleep(time.Millisecond)
	m.Add("Note 2", []string{"beta", "gamma"})
	time.Sleep(time.Millisecond)
	m.Add("Note 3", []string{"delta"})
	time.Sleep(time.Millisecond)
	scratched, _ := m.Add("Note 4", []string{"epsilon"})
	m.Scratch(scratched.ID)

	tags := m.ListTags()

	t.Run("correct tag count", func(t *testing.T) {
		if len(tags) != 4 {
			t.Errorf("expected 4 unique tags (excluding scratched), got %d", len(tags))
		}
	})

	t.Run("tags are sorted", func(t *testing.T) {
		for i := 1; i < len(tags); i++ {
			if tags[i] < tags[i-1] {
				t.Errorf("tags should be sorted alphabetically, got %v", tags)
				break
			}
		}
	})

	t.Run("excludes tags from scratched notes", func(t *testing.T) {
		for _, tag := range tags {
			if tag == "epsilon" {
				t.Error("should not include tags from scratched notes")
			}
		}
	})
}

func TestManagerCount(t *testing.T) {
	m := NewManager()

	if m.Count() != 0 {
		t.Errorf("new manager should have 0 notes, got %d", m.Count())
	}

	m.Add("Note 1", []string{"test"})
	time.Sleep(time.Millisecond)
	m.Add("Note 2", []string{"test"})
	time.Sleep(time.Millisecond)
	note3, _ := m.Add("Note 3", []string{"test"})

	if m.Count() != 3 {
		t.Errorf("expected 3 notes, got %d", m.Count())
	}

	m.Scratch(note3.ID)

	if m.Count() != 3 {
		t.Errorf("Count should include scratched notes, got %d", m.Count())
	}

	if m.CountActive() != 2 {
		t.Errorf("CountActive should exclude scratched notes, got %d", m.CountActive())
	}

	m.Delete(note3.ID)

	if m.Count() != 2 {
		t.Errorf("expected 2 notes after deletion, got %d", m.Count())
	}
}

func TestManagerClear(t *testing.T) {
	m := NewManager()

	m.Add("Note 1", []string{"test"})
	time.Sleep(time.Millisecond)
	m.Add("Note 2", []string{"test"})
	time.Sleep(time.Millisecond)
	m.Add("Note 3", []string{"test"})

	if m.Count() != 3 {
		t.Fatalf("expected 3 notes, got %d", m.Count())
	}

	m.Clear()

	if m.Count() != 0 {
		t.Errorf("expected 0 notes after clear, got %d", m.Count())
	}

	tags := m.ListTags()
	if len(tags) != 0 {
		t.Errorf("expected 0 tags after clear, got %d", len(tags))
	}
}

func TestManagerConcurrency(t *testing.T) {
	m := NewManager()

	// Test concurrent adds
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			// Add a small sleep to ensure unique nanosecond timestamps
			time.Sleep(time.Microsecond * time.Duration(n*10))
			_, err := m.Add("Concurrent note", []string{"test"})
			if err != nil {
				t.Errorf("concurrent add failed: %v", err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if m.Count() != 10 {
		t.Errorf("expected 10 notes after concurrent adds, got %d", m.Count())
	}
}
