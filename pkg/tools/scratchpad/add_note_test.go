package scratchpad

import (
	"context"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/entrhq/forge/pkg/agent/memory/notes"
)

func TestAddNoteTool_Name(t *testing.T) {
	manager := notes.NewManager()
	tool := NewAddNoteTool(manager)

	if got := tool.Name(); got != "add_note" {
		t.Errorf("Name() = %v, want %v", got, "add_note")
	}
}

func TestAddNoteTool_Description(t *testing.T) {
	manager := notes.NewManager()
	tool := NewAddNoteTool(manager)

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	if !strings.Contains(strings.ToLower(desc), "note") {
		t.Error("Description() should mention 'note'")
	}
}

func TestAddNoteTool_Schema(t *testing.T) {
	manager := notes.NewManager()
	tool := NewAddNoteTool(manager)

	schema := tool.Schema()
	if schema == nil {
		t.Fatal("Schema() returned nil")
	}

	// Check required fields
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Schema missing properties")
	}

	if _, ok := props["content"]; !ok {
		t.Error("Schema missing 'content' property")
	}
	if _, ok := props["tags"]; !ok {
		t.Error("Schema missing 'tags' property")
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("Schema missing required array")
	}
	if len(required) != 2 {
		t.Errorf("Expected 2 required fields, got %d", len(required))
	}
}

func TestAddNoteTool_IsLoopBreaking(t *testing.T) {
	manager := notes.NewManager()
	tool := NewAddNoteTool(manager)

	if tool.IsLoopBreaking() {
		t.Error("IsLoopBreaking() should return false")
	}
}

func TestAddNoteTool_Execute_Success(t *testing.T) {
	manager := notes.NewManager()
	tool := NewAddNoteTool(manager)

	argsXML := []byte(`<arguments>
		<content>Test note content</content>
		<tags>
			<tag>test</tag>
			<tag>example</tag>
		</tags>
	</arguments>`)

	result, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result == "" {
		t.Error("Execute() returned empty result")
	}
	if !strings.Contains(result, "created successfully") {
		t.Errorf("Result should mention creation: %s", result)
	}

	if metadata == nil {
		t.Fatal("Execute() returned nil metadata")
	}

	noteID, ok := metadata["note_id"].(string)
	if !ok || noteID == "" {
		t.Error("Metadata missing note_id")
	}

	totalNotes, ok := metadata["total_notes"].(int)
	if !ok || totalNotes != 1 {
		t.Errorf("Expected total_notes = 1, got %v", totalNotes)
	}

	// Verify note was actually created
	if manager.Count() != 1 {
		t.Errorf("Expected 1 note in manager, got %d", manager.Count())
	}
}

func TestAddNoteTool_Execute_InvalidXML(t *testing.T) {
	manager := notes.NewManager()
	tool := NewAddNoteTool(manager)

	argsXML := []byte(`<invalid>xml</invalid>`)

	_, _, err := tool.Execute(context.Background(), argsXML)
	if err == nil {
		t.Error("Execute() should fail with invalid XML")
	}
}

func TestAddNoteTool_Execute_MissingContent(t *testing.T) {
	manager := notes.NewManager()
	tool := NewAddNoteTool(manager)

	argsXML := []byte(`<arguments>
		<tags>
			<tag>test</tag>
		</tags>
	</arguments>`)

	_, _, err := tool.Execute(context.Background(), argsXML)
	if err == nil {
		t.Error("Execute() should fail with missing content")
	}
	if !strings.Contains(err.Error(), "content") {
		t.Errorf("Error should mention content: %v", err)
	}
}

func TestAddNoteTool_Execute_MissingTags(t *testing.T) {
	manager := notes.NewManager()
	tool := NewAddNoteTool(manager)

	argsXML := []byte(`<arguments>
		<content>Test note</content>
	</arguments>`)

	_, _, err := tool.Execute(context.Background(), argsXML)
	if err == nil {
		t.Error("Execute() should fail with missing tags")
	}
	if !strings.Contains(err.Error(), "tag") {
		t.Errorf("Error should mention tags: %v", err)
	}
}

func TestAddNoteTool_Execute_InvalidNote(t *testing.T) {
	manager := notes.NewManager()
	tool := NewAddNoteTool(manager)

	// Content too long
	longContent := strings.Repeat("a", 1000)
	argsXML := []byte(`<arguments>
		<content>` + longContent + `</content>
		<tags>
			<tag>test</tag>
		</tags>
	</arguments>`)

	_, _, err := tool.Execute(context.Background(), argsXML)
	if err == nil {
		t.Error("Execute() should fail with content too long")
	}
}

func TestAddNoteTool_Execute_MultipleNotes(t *testing.T) {
	manager := notes.NewManager()
	tool := NewAddNoteTool(manager)

	// Add first note
	argsXML1 := []byte(`<arguments>
		<content>First note</content>
		<tags>
			<tag>first</tag>
		</tags>
	</arguments>`)

	result1, metadata1, err := tool.Execute(context.Background(), argsXML1)
	if err != nil {
		t.Fatalf("First Execute() error = %v", err)
	}

	// Add second note
	argsXML2 := []byte(`<arguments>
		<content>Second note</content>
		<tags>
			<tag>second</tag>
		</tags>
	</arguments>`)

	result2, metadata2, err := tool.Execute(context.Background(), argsXML2)
	if err != nil {
		t.Fatalf("Second Execute() error = %v", err)
	}

	// Verify different IDs
	id1 := metadata1["note_id"].(string)
	id2 := metadata2["note_id"].(string)
	if id1 == id2 {
		t.Error("Two notes should have different IDs")
	}

	// Verify count increases
	count1 := metadata1["total_notes"].(int)
	count2 := metadata2["total_notes"].(int)
	if count1 != 1 || count2 != 2 {
		t.Errorf("Expected counts 1 and 2, got %d and %d", count1, count2)
	}

	// Verify both results mention creation
	if !strings.Contains(result1, "created") || !strings.Contains(result2, "created") {
		t.Error("Both results should mention creation")
	}
}

func TestAddNoteTool_Execute_SpecialCharacters(t *testing.T) {
	manager := notes.NewManager()
	tool := NewAddNoteTool(manager)

	argsXML := []byte(`<arguments>
		<content>Special chars: &lt;&gt;&amp;"'</content>
		<tags>
			<tag>special</tag>
		</tags>
	</arguments>`)

	_, _, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() should handle special characters: %v", err)
	}
}

func TestAddNoteTool_Execute_XMLUnmarshalError(t *testing.T) {
	manager := notes.NewManager()
	tool := NewAddNoteTool(manager)

	// Malformed XML
	argsXML := []byte(`<arguments><content>Test</arguments>`)

	_, _, err := tool.Execute(context.Background(), argsXML)
	if err == nil {
		t.Error("Execute() should fail with malformed XML")
	}
}

// Helper to create valid arguments XML
func createAddNoteXML(content string, tags []string) []byte {
	type Args struct {
		XMLName xml.Name `xml:"arguments"`
		Content string   `xml:"content"`
		Tags    []string `xml:"tags>tag"`
	}

	args := Args{
		Content: content,
		Tags:    tags,
	}

	data, _ := xml.Marshal(args)
	return data
}

func TestAddNoteTool_Execute_WithHelper(t *testing.T) {
	manager := notes.NewManager()
	tool := NewAddNoteTool(manager)

	argsXML := createAddNoteXML("Helper test", []string{"helper", "test"})

	result, metadata, err := tool.Execute(context.Background(), argsXML)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(result, "created") {
		t.Error("Result should mention creation")
	}

	if metadata["total_notes"].(int) != 1 {
		t.Error("Expected 1 note after creation")
	}
}
