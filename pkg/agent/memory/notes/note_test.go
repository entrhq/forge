package notes

import (
	"strings"
	"testing"
	"time"
)

func TestNewNote(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		tags        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid note",
			content:     "This is a test note",
			tags:        []string{"test", "valid"},
			expectError: false,
		},
		{
			name:        "single tag",
			content:     "Note with one tag",
			tags:        []string{"single"},
			expectError: false,
		},
		{
			name:        "max tags",
			content:     "Note with max tags",
			tags:        []string{"tag1", "tag2", "tag3", "tag4", "tag5"},
			expectError: false,
		},
		{
			name:        "empty content",
			content:     "",
			tags:        []string{"test"},
			expectError: true,
			errorMsg:    "content cannot be empty",
		},
		{
			name:        "content too long",
			content:     strings.Repeat("a", 801),
			tags:        []string{"test"},
			expectError: true,
			errorMsg:    "exceeds maximum length",
		},
		{
			name:        "no tags",
			content:     "Valid content",
			tags:        []string{},
			expectError: true,
			errorMsg:    "requires at least 1 tag",
		},
		{
			name:        "too many tags",
			content:     "Valid content",
			tags:        []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6"},
			expectError: true,
			errorMsg:    "requires 1-5 tags",
		},
		{
			name:        "empty tag",
			content:     "Valid content",
			tags:        []string{"valid", ""},
			expectError: true,
			errorMsg:    "tag at position 1 is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := NewNote(tt.content, tt.tags)

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

			if note.Content != tt.content {
				t.Errorf("content mismatch: got %q, want %q", note.Content, tt.content)
			}

			if len(note.Tags) != len(tt.tags) {
				t.Errorf("tag count mismatch: got %d, want %d", len(note.Tags), len(tt.tags))
			}

			if note.Scratched {
				t.Error("new note should not be scratched")
			}

			if !strings.HasPrefix(note.ID, IDPrefix) {
				t.Errorf("note ID should start with %q, got %q", IDPrefix, note.ID)
			}
		})
	}
}

func TestGenerateID(t *testing.T) {
	id1 := GenerateID()
	time.Sleep(2 * time.Millisecond)
	id2 := GenerateID()

	if id1 == id2 {
		t.Error("generated IDs should be unique")
	}

	if !strings.HasPrefix(id1, IDPrefix) {
		t.Errorf("ID should start with %q, got %q", IDPrefix, id1)
	}
}

func TestNoteUpdate(t *testing.T) {
	note, err := NewNote("Original content", []string{"original"})
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	originalTime := note.UpdatedAt
	time.Sleep(2 * time.Millisecond)

	tests := []struct {
		name        string
		content     *string
		tags        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "update content only",
			content:     stringPtr("New content"),
			tags:        nil,
			expectError: false,
		},
		{
			name:        "update tags only",
			content:     nil,
			tags:        []string{"new", "tags"},
			expectError: false,
		},
		{
			name:        "update both",
			content:     stringPtr("Updated content"),
			tags:        []string{"updated"},
			expectError: false,
		},
		{
			name:        "update neither",
			content:     nil,
			tags:        nil,
			expectError: true,
			errorMsg:    "at least one of content or tags must be provided",
		},
		{
			name:        "invalid content",
			content:     stringPtr(""),
			tags:        nil,
			expectError: true,
			errorMsg:    "content cannot be empty",
		},
		{
			name:        "invalid tags",
			content:     nil,
			tags:        []string{},
			expectError: true,
			errorMsg:    "requires at least 1 tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh note for each test
			testNote, _ := NewNote("Test content", []string{"test"})

			err := testNote.Update(tt.content, tt.tags)

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

			if tt.content != nil && testNote.Content != *tt.content {
				t.Errorf("content not updated: got %q, want %q", testNote.Content, *tt.content)
			}

			if tt.tags != nil && len(testNote.Tags) != len(tt.tags) {
				t.Errorf("tags not updated: got %d tags, want %d", len(testNote.Tags), len(tt.tags))
			}
		})
	}

	// Verify UpdatedAt changed
	newContent := "Final content"
	note.Update(&newContent, nil)
	if !note.UpdatedAt.After(originalTime) {
		t.Error("UpdatedAt should be updated after modification")
	}
}

func TestNoteScratch(t *testing.T) {
	note, err := NewNote("Test content", []string{"test"})
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	if note.Scratched {
		t.Error("new note should not be scratched")
	}

	originalTime := note.UpdatedAt
	time.Sleep(2 * time.Millisecond)

	note.Scratch()

	if !note.Scratched {
		t.Error("note should be scratched after Scratch()")
	}

	if !note.UpdatedAt.After(originalTime) {
		t.Error("UpdatedAt should be updated after scratching")
	}
}

func TestNoteUnscratched(t *testing.T) {
	note, err := NewNote("Test content", []string{"test"})
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	note.Scratch()
	if !note.Scratched {
		t.Error("note should be scratched")
	}

	originalTime := note.UpdatedAt
	time.Sleep(2 * time.Millisecond)

	note.Unscratched()

	if note.Scratched {
		t.Error("note should not be scratched after Unscratched()")
	}

	if !note.UpdatedAt.After(originalTime) {
		t.Error("UpdatedAt should be updated after unscratching")
	}
}

func TestNoteHasTag(t *testing.T) {
	note, _ := NewNote("Test content", []string{"Alpha", "BETA", "gamma"})

	tests := []struct {
		tag      string
		expected bool
	}{
		{"alpha", true},
		{"ALPHA", true},
		{"Beta", true},
		{"GAMMA", true},
		{"delta", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			result := note.HasTag(tt.tag)
			if result != tt.expected {
				t.Errorf("HasTag(%q) = %v, want %v", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestNoteMatchesAllTags(t *testing.T) {
	note, _ := NewNote("Test content", []string{"alpha", "beta", "gamma"})

	tests := []struct {
		name     string
		tags     []string
		expected bool
	}{
		{"empty tags", []string{}, true},
		{"single match", []string{"alpha"}, true},
		{"all match", []string{"alpha", "beta", "gamma"}, true},
		{"partial match", []string{"alpha", "delta"}, false},
		{"no match", []string{"delta"}, false},
		{"case insensitive", []string{"ALPHA", "Beta"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := note.MatchesAllTags(tt.tags)
			if result != tt.expected {
				t.Errorf("MatchesAllTags(%v) = %v, want %v", tt.tags, result, tt.expected)
			}
		})
	}
}

func TestNoteContainsText(t *testing.T) {
	note, _ := NewNote("The Quick Brown Fox Jumps", []string{"test"})

	tests := []struct {
		query    string
		expected bool
	}{
		{"quick", true},
		{"BROWN", true},
		{"fox jumps", true},
		{"", true},
		{"elephant", false},
		{"quick blue", false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			result := note.ContainsText(tt.query)
			if result != tt.expected {
				t.Errorf("ContainsText(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestNormalizeTags(t *testing.T) {
	input := []string{"  Alpha  ", "BETA", "gamma", "  DELTA"}
	expected := []string{"alpha", "beta", "gamma", "delta"}

	result := normalizeTags(input)

	if len(result) != len(expected) {
		t.Fatalf("length mismatch: got %d, want %d", len(result), len(expected))
	}

	for i, tag := range result {
		if tag != expected[i] {
			t.Errorf("tag[%d] = %q, want %q", i, tag, expected[i])
		}
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
