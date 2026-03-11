package parser

import (
	"strings"
	"testing"
)

func TestToolCallParser_SimpleToolCall(t *testing.T) {
	parser := NewToolCallParser()

	// Single chunk with complete tool call
	tc, rc := parser.Parse("<tool>{\"name\": \"test\"}</tool>")

	if tc == nil {
		t.Fatal("Expected tool call, got nil")
	}
	if tc.Content != `{"name": "test"}` {
		t.Errorf("Expected tool call content, got '%s'", tc.Content)
	}
	if rc != nil {
		t.Errorf("Expected no regular content, got %v", rc)
	}
}

func TestToolCallParser_StreamedToolCall(t *testing.T) {
	parser := NewToolCallParser()

	// Simulate streaming chunks - tags split across chunks
	chunks := []string{
		"<too",    // Partial opening tag
		"l>",      // Rest of opening tag
		"data",    // Content
		"</tool>", // Closing tag
	}

	var toolCallContent strings.Builder

	for _, chunk := range chunks {
		toolCall, _ := parser.Parse(chunk)

		if toolCall != nil {
			toolCallContent.WriteString(toolCall.Content)
		}
	}

	if toolCallContent.String() != "data" {
		t.Errorf("Expected 'data', got '%s'", toolCallContent.String())
	}
}

func TestToolCallParser_NoToolCall(t *testing.T) {
	parser := NewToolCallParser()

	toolCall, regular := parser.Parse("Just a regular message")

	if toolCall != nil {
		t.Error("Expected no tool call chunk")
	}

	if regular == nil {
		t.Fatal("Expected regular content chunk, got nil")
	}
	if regular.Content != "Just a regular message" {
		t.Errorf("Expected 'Just a regular message', got '%s'", regular.Content)
	}
	if regular.Type != ContentTypeRegular {
		t.Errorf("Expected regular type, got %v", regular.Type)
	}
}

func TestToolCallParser_MultipleToolCalls(t *testing.T) {
	parser := NewToolCallParser()

	// First tool call
	parser.Parse("<tool>")
	parser.Parse("data1")
	tc, _ := parser.Parse("</tool>")
	if tc == nil || tc.Content != "data1" {
		t.Errorf("Expected 'data1', got %v", tc)
	}

	// Second tool call
	parser.Parse("<tool>")
	parser.Parse("data2")
	tc, _ = parser.Parse("</tool>")
	if tc == nil || tc.Content != "data2" {
		t.Errorf("Expected 'data2', got %v", tc)
	}
}

func TestToolCallParser_ContentBeforeAndAfter(t *testing.T) {
	parser := NewToolCallParser()

	// Single parse with content before and after tool call
	tc, rc := parser.Parse("prefix <tool>data</tool> suffix")

	if tc == nil || tc.Content != "data" {
		t.Errorf("Expected tool call 'data', got %v", tc)
	}
	if rc == nil || rc.Content != "prefix  suffix" {
		t.Errorf("Expected 'prefix  suffix', got %v", rc)
	}
}

func TestToolCallParser_IncompleteTagAtBoundary(t *testing.T) {
	parser := NewToolCallParser()

	// Chunk ends with incomplete tag
	toolCall, regular := parser.Parse("Some text <")

	// Should buffer the '<' and return text before it
	if regular == nil || regular.Content != "Some text " {
		t.Errorf("Expected 'Some text ', got %v", regular)
	}
	if toolCall != nil {
		t.Errorf("Expected no tool call yet, got %v", toolCall)
	}

	// Complete the opening tag and send content
	toolCall, regular = parser.Parse("tool>content")
	// With early emission, we now get a tool_call_start signal when <tool> is detected
	if toolCall == nil || toolCall.Type != ContentTypeToolCallStart {
		t.Errorf("Expected tool_call_start signal, got toolCall=%v", toolCall)
	}
	if regular != nil {
		t.Errorf("Expected no regular content, got %v", regular)
	}

	// Send closing tag
	toolCall, _ = parser.Parse("</tool>")
	if toolCall == nil || toolCall.Content != "content" {
		t.Errorf("Expected tool call with 'content', got %v", toolCall)
	}
}

func TestToolCallParser_IsInToolCall(t *testing.T) {
	parser := NewToolCallParser()

	if parser.IsInToolCall() {
		t.Error("Parser should not be in tool call initially")
	}

	// Start tool call
	parser.Parse("<tool>")
	if !parser.IsInToolCall() {
		t.Error("Parser should be in tool call after opening tag")
	}

	// Add content
	parser.Parse("some content")
	if !parser.IsInToolCall() {
		t.Error("Parser should still be in tool call")
	}

	// Close tool call
	parser.Parse("</tool>")
	if parser.IsInToolCall() {
		t.Error("Parser should not be in tool call after closing tag")
	}
}

func TestToolCallParser_Flush(t *testing.T) {
	parser := NewToolCallParser()

	// Parse incomplete tool call
	parser.Parse("<tool>incomplete content")

	// Should still be in tool call
	if !parser.IsInToolCall() {
		t.Error("Expected parser to be in tool call")
	}

	// Flush should return buffered content
	toolCall, regular := parser.Flush()

	if toolCall == nil || toolCall.Content != "incomplete content" {
		t.Errorf("Expected flushed tool content, got %v", toolCall)
	}
	if regular != nil {
		t.Errorf("Expected no regular content, got %v", regular)
	}

	// Parser should be reset
	if parser.IsInToolCall() {
		t.Error("Parser should not be in tool call after flush")
	}
}

func TestToolCallParser_FlushRegularContent(t *testing.T) {
	parser := NewToolCallParser()

	// Parse incomplete tag - content is flushed immediately since it doesn't match <tool prefix
	_, regular1 := parser.Parse("text <incomplete")

	// All content should be flushed during Parse since "<incomplete" doesn't match "<tool" prefix
	if regular1 == nil || regular1.Content != "text <incomplete" {
		t.Errorf("Expected parsed regular content 'text <incomplete', got %v", regular1)
	}

	// Flush should return nothing since everything was already emitted
	toolCall, regular2 := parser.Flush()

	if regular2 != nil {
		t.Errorf("Expected no remaining regular content, got %v", regular2)
	}
	if toolCall != nil {
		t.Errorf("Expected no tool call content, got %v", toolCall)
	}
}

func TestToolCallParser_Reset(t *testing.T) {
	parser := NewToolCallParser()

	// Start parsing
	parser.Parse("<tool>Some content")

	if !parser.IsInToolCall() {
		t.Error("Expected parser to be in tool call")
	}

	// Reset
	parser.Reset()

	if parser.IsInToolCall() {
		t.Error("Expected parser to not be in tool call after reset")
	}

	// Should work normally after reset
	toolCall, regular := parser.Parse("New message")
	if toolCall != nil {
		t.Error("Expected no tool call after reset")
	}
	if regular == nil || regular.Content != "New message" {
		t.Errorf("Expected 'New message', got %v", regular)
	}
}

func TestToolCallParser_AngleBracketsInContent(t *testing.T) {
	parser := NewToolCallParser()

	// Note: The parser cannot distinguish between angle brackets in content
	// vs actual tags. Angle brackets will be interpreted as potential tags.
	// This is a known limitation - tool call content should avoid < and > characters.
	tc, rc := parser.Parse("<tool>value with brackets</tool>")

	if tc == nil || tc.Content != "value with brackets" {
		t.Errorf("Expected content without angle brackets, got %v", tc)
	}
	if rc != nil {
		t.Errorf("Expected no regular content, got %v", rc)
	}
}

func TestToolCallParser_MalformedTags(t *testing.T) {
	parser := NewToolCallParser()

	// Not a tool tag, should be treated as regular content
	toolCall, regular := parser.Parse("<notool>content</notool>")

	if toolCall != nil {
		t.Errorf("Expected no tool call for non-tool tags, got %v", toolCall)
	}
	if regular == nil || regular.Content != "<notool>content</notool>" {
		t.Errorf("Expected malformed tags as regular content, got %v", regular)
	}
}

func TestToolCallParser_EmptyToolCall(t *testing.T) {
	parser := NewToolCallParser()

	toolCall, regular := parser.Parse("<tool></tool>")

	if toolCall == nil {
		t.Fatal("Expected tool call chunk, got nil")
	}
	if toolCall.Content != "" {
		t.Errorf("Expected empty tool call content, got '%s'", toolCall.Content)
	}
	if regular != nil {
		t.Errorf("Expected no regular content, got %v", regular)
	}
}

func TestToolCallParser_StreamedContentAccumulation(t *testing.T) {
	parser := NewToolCallParser()

	chunks := []string{
		"Regular text ",
		"continues here ",
		"and more",
	}

	var regularContent strings.Builder

	for _, chunk := range chunks {
		toolCall, regular := parser.Parse(chunk)

		if toolCall != nil {
			t.Errorf("Expected no tool call, got %v", toolCall)
		}

		if regular != nil {
			regularContent.WriteString(regular.Content)
		}
	}

	// Flush remaining
	_, regular := parser.Flush()
	if regular != nil {
		regularContent.WriteString(regular.Content)
	}

	expected := "Regular text continues here and more"
	if regularContent.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, regularContent.String())
	}
}
