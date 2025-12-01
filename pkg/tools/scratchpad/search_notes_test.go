package scratchpad

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/entrhq/forge/pkg/agent/memory/notes"
)

func TestSearchNotesTool_Name(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	if got := tool.Name(); got != "search_notes" {
		t.Errorf("Name() = %v, want %v", got, "search_notes")
	}
}

func TestSearchNotesTool_Description(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !strings.Contains(strings.ToLower(desc), "search") {
		t.Error("Description() should mention 'search'")
	}
}

func TestSearchNotesTool_Schema(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	schema := tool.Schema()
	if schema == nil {
		t.Fatal("Schema() returned nil")
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema missing properties")
	}

	if _, ok := props["query"]; !ok {
		t.Error("Schema missing 'query' property")
	}
	if _, ok := props["tags"]; !ok {
		t.Error("Schema missing 'tags' property")
	}

	// Both query and tags are optional - required field might not exist or be empty
	if required, ok := schema["required"].([]interface{}); ok {
		if len(required) != 0 {
			t.Errorf("Expected 0 required fields, got %d", len(required))
		}
	}
}

func TestSearchNotesTool_IsLoopBreaking(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	if tool.IsLoopBreaking() {
		t.Error("IsLoopBreaking() should return false")
	}
}

func TestSearchNotesTool_Execute_NoResults(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	argsXML := []byte(`<arguments>
		<query>nonexistent</query>
	</arguments>`)

	result, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(result, "No notes found") {
		t.Errorf("Result should mention no notes found: %s", result)
	}

	if metadata == nil {
		t.Fatal("Execute() returned nil metadata")
	}

	resultCount, ok := metadata["result_count"].(int)
	if !ok || resultCount != 0 {
		t.Errorf("Expected result_count = 0, got %v", resultCount)
	}
}

func TestSearchNotesTool_Execute_ByQuery(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	// Add some test notes
	_, err := manager.Add("This is a test note", []string{"test"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	_, err = manager.Add("Another example", []string{"example"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	_, err = manager.Add("Test example combined", []string{"both"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	argsXML := []byte(`<arguments>
		<query>test</query>
	</arguments>`)

	_, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultCount, ok := metadata["result_count"].(int)
	if !ok || resultCount != 2 {
		t.Errorf("Expected result_count = 2, got %v", resultCount)
	}
}

func TestSearchNotesTool_Execute_ByTags(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	// Add test notes with different tags
	_, err := manager.Add("Note 1", []string{"important", "todo"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	_, err = manager.Add("Note 2", []string{"important", "done"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	_, err = manager.Add("Note 3", []string{"todo"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	argsXML := []byte(`<arguments>
		<tags>
			<tag>important</tag>
		</tags>
	</arguments>`)

	_, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultCount := metadata["result_count"].(int)
	if resultCount != 2 {
		t.Errorf("Expected 2 notes with 'important' tag, got %d", resultCount)
	}
}

func TestSearchNotesTool_Execute_QueryAndTags(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	// Add test notes
	_, err := manager.Add("Important task to complete", []string{"important", "todo"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	_, err = manager.Add("Important but done", []string{"important", "done"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	_, err = manager.Add("Task to do", []string{"todo"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	argsXML := []byte(`<arguments>
		<query>task</query>
		<tags>
			<tag>important</tag>
		</tags>
	</arguments>`)

	_, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultCount := metadata["result_count"].(int)
	if resultCount != 1 {
		t.Errorf("Expected 1 note matching both query and tags, got %d", resultCount)
	}
}

func TestSearchNotesTool_Execute_EmptySearch(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	// Add some notes
	_, err := manager.Add("Note 1", []string{"tag1"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	_, err = manager.Add("Note 2", []string{"tag2"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	// Empty search should return all notes
	argsXML := []byte(`<arguments></arguments>`)

	_, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultCount := metadata["result_count"].(int)
	if resultCount != 2 {
		t.Errorf("Expected 2 notes with empty search, got %d", resultCount)
	}
}

func TestSearchNotesTool_Execute_CaseInsensitive(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	_, err := manager.Add("UPPERCASE content", []string{"test"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	_, err = manager.Add("lowercase content", []string{"test"})
	if err != nil {
		t.Fatalf("Failed to add note: %v", err)
	}

	argsXML := []byte(`<arguments>
		<query>CoNtEnT</query>
	</arguments>`)

	_, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultCount := metadata["result_count"].(int)
	if resultCount != 2 {
		t.Errorf("Expected case-insensitive search to find 2 notes, got %d", resultCount)
	}
}

func TestSearchNotesTool_Execute_IncludesScratched(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	// Add and scratch a note
	note, _ := manager.Add("Scratched note", []string{"test"})
	manager.Scratch(note.ID)

	time.Sleep(time.Microsecond) // Ensure unique timestamps
	// Add an active note
	manager.Add("Active note", []string{"test"})

	argsXML := []byte(`<arguments>
		<query>note</query>
	</arguments>`)

	result, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Search includes scratched notes
	resultCount := metadata["result_count"].(int)
	if resultCount != 2 {
		t.Errorf("Expected search to include scratched notes, got %d", resultCount)
	}

	// Verify result shows scratched status
	if !strings.Contains(result, "[SCRATCHED]") {
		t.Error("Result should show scratched status")
	}
}

func TestSearchNotesTool_Execute_InvalidXML(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	argsXML := []byte(`<invalid>xml</invalid>`)

	_, _, err := tool.Execute(context.Background(), argsXML)
	if err == nil {
		t.Error("Execute() should fail with invalid XML")
	}
}

func TestSearchNotesTool_Execute_ResultFormat(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	note, _ := manager.Add("Test content", []string{"tag1", "tag2"})

	argsXML := []byte(`<arguments>
		<query>test</query>
	</arguments>`)

	result, _, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Check result format includes ID, tags, and content
	if !strings.Contains(result, note.ID) {
		t.Error("Result should include note ID")
	}
	if !strings.Contains(result, "tag1") || !strings.Contains(result, "tag2") {
		t.Error("Result should include all tags")
	}
	if !strings.Contains(result, "Test content") {
		t.Error("Result should include note content")
	}
}

func TestSearchNotesTool_Execute_MultipleTagsAND(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	// Add notes with different tag combinations
	manager.Add("Both tags", []string{"important", "urgent"})
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	manager.Add("Only important", []string{"important"})
	time.Sleep(time.Microsecond) // Ensure unique timestamps
	manager.Add("Only urgent", []string{"urgent"})

	// Search with multiple tags (AND logic)
	argsXML := []byte(`<arguments>
		<tags>
			<tag>important</tag>
			<tag>urgent</tag>
		</tags>
	</arguments>`)

	_, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	resultCount := metadata["result_count"].(int)
	if resultCount != 1 {
		t.Errorf("Expected 1 note with both tags (AND logic), got %d", resultCount)
	}
}

func TestSearchNotesTool_Execute_MetadataComplete(t *testing.T) {
	manager := notes.NewManager()
	tool := NewSearchNotesTool(manager)

	manager.Add("Test note", []string{"test"})

	argsXML := []byte(`<arguments>
		<query>test</query>
		<tags>
			<tag>test</tag>
		</tags>
	</arguments>`)

	_, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Check all expected metadata fields
	if _, ok := metadata["result_count"]; !ok {
		t.Error("Metadata missing result_count")
	}
	if _, ok := metadata["query"]; !ok {
		t.Error("Metadata missing query")
	}
	if _, ok := metadata["tags"]; !ok {
		t.Error("Metadata missing tags")
	}

	// Verify metadata values
	if metadata["query"].(string) != "test" {
		t.Error("Metadata query doesn't match input")
	}
	tags := metadata["tags"].([]string)
	if len(tags) != 1 || tags[0] != "test" {
		t.Error("Metadata tags don't match input")
	}
}
