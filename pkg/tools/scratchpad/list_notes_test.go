package scratchpad

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/entrhq/forge/pkg/agent/memory/notes"
)

func TestListNotesTool_Name(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	if got := tool.Name(); got != "list_notes" {
		t.Errorf("Name() = %v, want %v", got, "list_notes")
	}
}

func TestListNotesTool_Description(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !strings.Contains(strings.ToLower(desc), "list") {
		t.Error("Description() should mention 'list'")
	}
}

func TestListNotesTool_Schema(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	schema := tool.Schema()
	if schema == nil {
		t.Fatal("Schema() returned nil")
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema missing properties")
	}

	if _, ok := props["tag"]; !ok {
		t.Error("Schema missing 'tag' property")
	}
	if _, ok := props["include_scratched"]; !ok {
		t.Error("Schema missing 'include_scratched' property")
	}
	if _, ok := props["limit"]; !ok {
		t.Error("Schema missing 'limit' property")
	}
}

func TestListNotesTool_IsLoopBreaking(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	if tool.IsLoopBreaking() {
		t.Error("IsLoopBreaking() should return false")
	}
}

func TestListNotesTool_Execute_EmptyList(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	argsXML := []byte(`<arguments></arguments>`)

	result, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(result, "No notes found") {
		t.Errorf("Result should mention no notes: %s", result)
	}

	if metadata == nil {
		t.Fatal("Execute() returned nil metadata")
	}

	noteCount, ok := metadata["note_count"].(int)
	if !ok || noteCount != 0 {
		t.Errorf("Expected note_count = 0, got %v", noteCount)
	}
}

func TestListNotesTool_Execute_DefaultLimit(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	// Add 15 notes with small delay to ensure unique IDs
	for i := 0; i < 15; i++ {
		_, err := manager.Add(fmt.Sprintf("Note %d", i), []string{"test"})
		if err != nil {
			t.Fatalf("Failed to add note: %v", err)
		}
		time.Sleep(time.Microsecond) // Ensure unique timestamps
	}

	argsXML := []byte(`<arguments></arguments>`)

	_, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Default limit is 10
	noteCount := metadata["note_count"].(int)
	if noteCount != 10 {
		t.Errorf("Expected default limit of 10, got %d", noteCount)
	}

	limit := metadata["limit"].(int)
	if limit != 10 {
		t.Errorf("Expected metadata limit = 10, got %d", limit)
	}
}

func TestListNotesTool_Execute_CustomLimit(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	// Add 10 notes
	for i := 0; i < 10; i++ {
		manager.Add(fmt.Sprintf("Note %d", i), []string{"test"})
		time.Sleep(time.Microsecond) // Ensure unique timestamps
	}

	argsXML := []byte(`<arguments>
		<limit>5</limit>
	</arguments>`)

	_, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	noteCount := metadata["note_count"].(int)
	if noteCount != 5 {
		t.Errorf("Expected 5 notes with custom limit, got %d", noteCount)
	}
}

func TestListNotesTool_Execute_FilterByTag(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	// Add notes with different tags
	_, err := manager.Add("Note 1", []string{"important", "work"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	_, err = manager.Add("Note 2", []string{"important", "personal"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	_, err = manager.Add("Note 3", []string{"work"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	argsXML := []byte(`<arguments>
		<tag>important</tag>
	</arguments>`)

	_, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	noteCount := metadata["note_count"].(int)
	if noteCount != 2 {
		t.Errorf("Expected 2 notes with 'important' tag, got %d", noteCount)
	}

	tagFilter := metadata["tag_filter"].(string)
	if tagFilter != "important" {
		t.Errorf("Expected tag_filter = 'important', got %s", tagFilter)
	}
}

func TestListNotesTool_Execute_ExcludeScratched(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	// Add notes and scratch one
	_, _ = manager.Add("Active note", []string{"test"})
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	note2, _ := manager.Add("Scratched note", []string{"test"})
	manager.Scratch(note2.ID)

	// Default: exclude scratched
	argsXML := []byte(`<arguments></arguments>`)

	result, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	noteCount := metadata["note_count"].(int)
	if noteCount != 1 {
		t.Errorf("Expected 1 active note, got %d", noteCount)
	}

	if strings.Contains(result, "[SCRATCHED]") {
		t.Error("Result should not include scratched notes by default")
	}
}

func TestListNotesTool_Execute_IncludeScratched(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	// Add notes and scratch one
	note1, err := manager.Add("Active note", []string{"test"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	note2, err := manager.Add("Scratched note", []string{"test"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	_, err = manager.Scratch(note2.ID)
	if err != nil {
		t.Fatalf("Failed to scratch note: %v", err)
	}

	argsXML := []byte(`<arguments>
		<include_scratched>true</include_scratched>
	</arguments>`)

	result, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	noteCount := metadata["note_count"].(int)
	if noteCount != 2 {
		t.Errorf("Expected 2 notes including scratched, got %d. Active: %s, Scratched: %s", noteCount, note1.ID, note2.ID)
	}

	if !strings.Contains(result, "[SCRATCHED]") {
		t.Error("Result should show scratched status when included")
	}

	includeScratched := metadata["include_scratched"].(bool)
	if !includeScratched {
		t.Error("Metadata should reflect include_scratched = true")
	}
}

func TestListNotesTool_Execute_InvalidXML(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	argsXML := []byte(`<invalid>xml</invalid>`)

	_, _, err := tool.Execute(context.Background(), argsXML)
	if err == nil {
		t.Error("Execute() should fail with invalid XML")
	}
}

func TestListNotesTool_Execute_ResultFormat(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	note, _ := manager.Add("Test content", []string{"tag1", "tag2"})

	argsXML := []byte(`<arguments></arguments>`)

	result, _, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Check result format
	if !strings.Contains(result, note.ID) {
		t.Error("Result should include note ID")
	}
	if !strings.Contains(result, "tag1") || !strings.Contains(result, "tag2") {
		t.Error("Result should include all tags")
	}
	if !strings.Contains(result, "Test content") {
		t.Error("Result should include note content")
	}
	if !strings.Contains(result, "Found 1 note") {
		t.Error("Result should show count")
	}
}

func TestListNotesTool_Execute_SortedByTime(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	// Add notes with slight delays to ensure different timestamps
	note1, _ := manager.Add("First note", []string{"test"})
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	note2, _ := manager.Add("Second note", []string{"test"})
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	note3, _ := manager.Add("Third note", []string{"test"})

	argsXML := []byte(`<arguments></arguments>`)

	result, _, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Notes should be in reverse chronological order (newest first)
	idx1 := strings.Index(result, note1.ID)
	idx2 := strings.Index(result, note2.ID)
	idx3 := strings.Index(result, note3.ID)

	if idx3 > idx2 || idx2 > idx1 {
		t.Error("Notes should be sorted newest first")
	}
}

func TestListNotesTool_Execute_TagAndScratched(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	// Add notes with different tags and scratch states
	note1, err := manager.Add("Active important", []string{"important"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	note2, err := manager.Add("Scratched important", []string{"important"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	_, err = manager.Scratch(note2.ID)
	if err != nil {
		t.Fatalf("Failed to scratch note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	note3, err := manager.Add("Active other", []string{"other"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	// Filter by tag and include scratched
	argsXML := []byte(`<arguments>
		<tag>important</tag>
		<include_scratched>true</include_scratched>
	</arguments>`)

	result, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	noteCount := metadata["note_count"].(int)
	if noteCount != 2 {
		t.Errorf("Expected 2 important notes (1 active, 1 scratched), got %d", noteCount)
	}

	// Should include both IDs
	if !strings.Contains(result, note1.ID) || !strings.Contains(result, note2.ID) {
		t.Error("Result should include both important notes")
	}
	if strings.Contains(result, note3.ID) {
		t.Error("Result should not include note with different tag")
	}
}

func TestListNotesTool_Execute_LimitLessThanTotal(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	// Add 5 notes
	for i := 0; i < 5; i++ {
		manager.Add(fmt.Sprintf("Note %d", i), []string{"test"})
		time.Sleep(time.Microsecond) // Ensure unique timestamps
	}

	argsXML := []byte(`<arguments>
		<limit>3</limit>
	</arguments>`)

	_, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	noteCount := metadata["note_count"].(int)
	if noteCount != 3 {
		t.Errorf("Expected 3 notes with limit, got %d", noteCount)
	}
}

func TestListNotesTool_Execute_LimitGreaterThanTotal(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	// Add 2 notes
	_, err := manager.Add("Note 1", []string{"test"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	_, err = manager.Add("Note 2", []string{"test"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	argsXML := []byte(`<arguments>
		<limit>10</limit>
	</arguments>`)

	_, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Should return all available notes
	noteCount := metadata["note_count"].(int)
	if noteCount != 2 {
		t.Errorf("Expected 2 notes (all available), got %d", noteCount)
	}
}

func TestListNotesTool_Execute_MetadataComplete(t *testing.T) {
	manager := notes.NewManager()
	tool := NewListNotesTool(manager)

	manager.Add("Test note", []string{"test"})

	argsXML := []byte(`<arguments>
		<tag>test</tag>
		<include_scratched>true</include_scratched>
		<limit>5</limit>
	</arguments>`)

	_, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Check all expected metadata fields
	if _, ok := metadata["note_count"]; !ok {
		t.Error("Metadata missing note_count")
	}
	if _, ok := metadata["tag_filter"]; !ok {
		t.Error("Metadata missing tag_filter")
	}
	if _, ok := metadata["include_scratched"]; !ok {
		t.Error("Metadata missing include_scratched")
	}
	if _, ok := metadata["limit"]; !ok {
		t.Error("Metadata missing limit")
	}

	// Verify values
	if metadata["tag_filter"].(string) != "test" {
		t.Error("Metadata tag_filter doesn't match input")
	}
	if !metadata["include_scratched"].(bool) {
		t.Error("Metadata include_scratched should be true")
	}
	if metadata["limit"].(int) != 5 {
		t.Error("Metadata limit doesn't match input")
	}
}
