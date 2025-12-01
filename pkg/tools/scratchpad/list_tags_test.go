package scratchpad

import (
	"context"
	"testing"

	"github.com/entrhq/forge/pkg/agent/memory/notes"
)

func TestListTagsTool_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("empty scratchpad", func(t *testing.T) {
		manager := notes.NewManager()
		tool := NewListTagsTool(manager)

		result, metadata, err := tool.Execute(ctx, []byte(`<arguments/>`))
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		if result != "No tags found in active notes." {
			t.Errorf("Expected empty message, got: %s", result)
		}

		tags := metadata["tags"].([]string)
		if len(tags) != 0 {
			t.Errorf("Expected 0 tags, got %d", len(tags))
		}
	})

	t.Run("lists unique tags from active notes", func(t *testing.T) {
		manager := notes.NewManager()
		tool := NewListTagsTool(manager)

		// Add notes with various tags
		manager.Add("Note 1", []string{"bug", "critical"})
		manager.Add("Note 2", []string{"bug", "backend"})

		result, metadata, err := tool.Execute(ctx, []byte(`<arguments/>`))
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		tags := metadata["tags"].([]string)
		expectedTags := []string{"backend", "bug", "critical"}

		if len(tags) != len(expectedTags) {
			t.Errorf("Expected %d tags, got %d", len(expectedTags), len(tags))
		}

		// Verify tags are sorted
		for i, tag := range tags {
			if tag != expectedTags[i] {
				t.Errorf("Expected tag[%d] = %s, got %s", i, expectedTags[i], tag)
			}
		}

		// Verify metadata
		if metadata["tag_count"].(int) != 3 {
			t.Errorf("Expected tag_count = 3, got %d", metadata["tag_count"])
		}

		if metadata["active_notes"].(int) != 2 {
			t.Errorf("Expected active_notes = 2, got %d", metadata["active_notes"])
		}

		// Verify result message contains count
		if !contains(result, "3 unique tag(s)") {
			t.Errorf("Expected result to mention 3 tags, got: %s", result)
		}
	})

	t.Run("excludes tags from scratched notes", func(t *testing.T) {
		manager := notes.NewManager()
		tool := NewListTagsTool(manager)

		// Add and scratch a note first
		note, _ := manager.Add("Old note", []string{"old", "obsolete"})
		manager.Scratch(note.ID)

		// Add active note after scratching
		manager.Add("Active note", []string{"active", "current"})

		_, metadata, err := tool.Execute(ctx, []byte(`<arguments/>`))
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		tags := metadata["tags"].([]string)

		// Should only have tags from active note
		if len(tags) != 2 {
			t.Errorf("Expected 2 tags from active notes, got %d", len(tags))
		}

		// Verify scratched note tags are not included
		for _, tag := range tags {
			if tag == "old" || tag == "obsolete" {
				t.Errorf("Scratched note tags should not be included, found: %s", tag)
			}
		}

		// Verify metadata shows correct counts
		if metadata["active_notes"].(int) != 1 {
			t.Errorf("Expected 1 active note, got %d", metadata["active_notes"])
		}

		if metadata["total_notes"].(int) != 2 {
			t.Errorf("Expected 2 total notes, got %d", metadata["total_notes"])
		}
	})

	t.Run("handles duplicate tags across notes", func(t *testing.T) {
		manager := notes.NewManager()
		tool := NewListTagsTool(manager)

		// Add notes with overlapping tags
		manager.Add("Note 1", []string{"bug", "critical"})
		manager.Add("Note 2", []string{"bug", "backend"})

		_, metadata, err := tool.Execute(ctx, []byte(`<arguments/>`))
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		tags := metadata["tags"].([]string)

		// "bug" appears in all 3 notes but should only be listed once
		bugCount := 0
		for _, tag := range tags {
			if tag == "bug" {
				bugCount++
			}
		}

		if bugCount != 1 {
			t.Errorf("Expected 'bug' to appear once, appeared %d times", bugCount)
		}

		// Total unique tags should be 3: backend, bug, critical
		if len(tags) != 3 {
			t.Errorf("Expected 3 unique tags, got %d", len(tags))
		}
	})
}

func TestListTagsTool_Metadata(t *testing.T) {
	tool := NewListTagsTool(notes.NewManager())

	if tool.Name() != "list_tags" {
		t.Errorf("Expected name 'list_tags', got '%s'", tool.Name())
	}

	if tool.IsLoopBreaking() {
		t.Error("Expected IsLoopBreaking to be false")
	}

	schema := tool.Schema()
	if schema == nil {
		t.Fatal("Expected schema, got nil")
	}

	// Verify no required parameters (field should not exist or be empty)
	if required, ok := schema["required"]; ok {
		requiredSlice, isSlice := required.([]string)
		if !isSlice {
			t.Fatal("Expected 'required' field to be []string")
		}
		if len(requiredSlice) != 0 {
			t.Errorf("Expected no required parameters, got %d", len(requiredSlice))
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
