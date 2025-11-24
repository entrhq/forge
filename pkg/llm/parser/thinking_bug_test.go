package parser

import (
	"testing"
)

// TestThinkingParserWithLessThanGreaterThan reproduces the bug where
// thinking content containing < or > characters prevents </thinking> from being detected
func TestThinkingParserWithLessThanGreaterThan(t *testing.T) {
	parser := NewThinkingParser()

	// This is the scenario from the bug report:
	// Thinking content with < and > characters
	chunks := []string{
		"<thinking>",
		"Looking at code:\n",
		"1. Line 8: `var x int=5`\n",
		"2. Line 11: `if x>3{`\n",          // Contains >
		"3. Line 15: `for i:=0;i<10;i++{`\n", // Contains <
		"</thinking>",
		"\n\n<tool>test</tool>",
	}

	var thinkingContent string
	var messageContent string
	var stillInThinking bool

	for i, chunk := range chunks {
		thinking, message := parser.Parse(chunk)
		
		if thinking != nil {
			thinkingContent += thinking.Content
		}
		if message != nil {
			messageContent += message.Content
		}
		
		stillInThinking = parser.IsInThinking()
		
		t.Logf("Chunk %d: inThinking=%v, thinking=%q, message=%q", 
			i, stillInThinking, 
			func() string { if thinking != nil { return thinking.Content }; return "" }(),
			func() string { if message != nil { return message.Content }; return "" }())
	}

	// Flush any remaining content
	thinking, message := parser.Flush()
	if thinking != nil {
		thinkingContent += thinking.Content
	}
	if message != nil {
		messageContent += message.Content
	}

	t.Logf("Final: inThinking=%v", parser.IsInThinking())
	t.Logf("Thinking content: %q", thinkingContent)
	t.Logf("Message content: %q", messageContent)

	// BUG: The parser should NOT still be in thinking mode after </thinking>
	if stillInThinking {
		t.Error("Parser is still in thinking mode after </thinking> tag - BUG CONFIRMED")
	}

	// The </thinking> tag should have been detected and closed thinking mode
	if !contains(messageContent, "<tool>") {
		t.Error("Tool call should be in message content, not thinking content")
	}

	// Thinking content should contain the code examples with < and >
	if !contains(thinkingContent, "i<10") || !contains(thinkingContent, "x>3") {
		t.Errorf("Thinking content should preserve < and > characters. Got: %q", thinkingContent)
	}
}

// TestThinkingParserSimpleCase verifies the parser works without < > in content
func TestThinkingParserSimpleCase(t *testing.T) {
	parser := NewThinkingParser()

	chunks := []string{
		"<thinking>",
		"This is thinking",
		"</thinking>",
		"This is a message",
	}

	var messageContent string

	for _, chunk := range chunks {
		_, message := parser.Parse(chunk)
		if message != nil {
			messageContent += message.Content
		}
	}

	_, message := parser.Flush()
	if message != nil {
		messageContent += message.Content
	}

	if parser.IsInThinking() {
		t.Error("Parser should not be in thinking mode after </thinking>")
	}

	if !contains(messageContent, "This is a message") {
		t.Errorf("Message content should contain the message. Got: %q", messageContent)
	}
}

// TestThinkingParserLessThanOnly tests < without closing >
func TestThinkingParserLessThanOnly(t *testing.T) {
	parser := NewThinkingParser()

	chunks := []string{
		"<thinking>",
		"Code: x < 5",
		"</thinking>",
		"Done",
	}

	for _, chunk := range chunks {
		parser.Parse(chunk)
	}

	parser.Flush()

	if parser.IsInThinking() {
		t.Error("Parser should not be in thinking mode after </thinking> - even with < in content")
	}
}