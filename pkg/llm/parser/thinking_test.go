package parser

import (
	"strings"
	"testing"

	"github.com/entrhq/forge/pkg/llm"
)

func TestThinkingParser_SimpleThinking(t *testing.T) {
	parser := NewThinkingParser()

	// Single chunk with thinking tags
	thinking, message := parser.Parse("<thinking>Let me think...</thinking>The answer is 42")

	if thinking == nil {
		t.Fatal("Expected thinking chunk, got nil")
	}
	if thinking.Type != llm.ContentTypeThinking {
		t.Errorf("Expected thinking type, got %v", thinking.Type)
	}
	if thinking.Content != "Let me think..." {
		t.Errorf("Expected 'Let me think...', got '%s'", thinking.Content)
	}

	if message == nil {
		t.Fatal("Expected message chunk, got nil")
	}
	if message.Type != llm.ContentTypeMessage {
		t.Errorf("Expected message type, got %v", message.Type)
	}
	if message.Content != "The answer is 42" {
		t.Errorf("Expected 'The answer is 42', got '%s'", message.Content)
	}
}

func TestThinkingParser_StreamedThinking(t *testing.T) {
	parser := NewThinkingParser()

	// Simulate streaming chunks - tags split across chunks
	chunks := []string{
		"<think",            // Partial opening tag
		"ing>Let me ",       // Rest of tag + content
		"analyze this...",   // More thinking content
		"</thinking>",       // Closing tag
		"Here's the answer", // Message content
	}

	var thinkingContent strings.Builder
	var messageContent string

	for _, chunk := range chunks {
		thinking, message := parser.Parse(chunk)

		if thinking != nil {
			thinkingContent.WriteString(thinking.Content)
			if thinking.Type != llm.ContentTypeThinking {
				t.Errorf("Expected thinking type, got %v", thinking.Type)
			}
		}

		if message != nil {
			messageContent += message.Content
			if message.Type != llm.ContentTypeMessage {
				t.Errorf("Expected message type, got %v", message.Type)
			}
		}
	}

	// Flush to get any remaining buffered content
	thinkingFlush, messageFlush := parser.Flush()
	if thinkingFlush != nil {
		thinkingContent.WriteString(thinkingFlush.Content)
	}
	if messageFlush != nil {
		messageContent += messageFlush.Content
	}

	if thinkingContent.String() != "Let me analyze this..." {
		t.Errorf("Expected 'Let me analyze this...', got '%s'", thinkingContent.String())
	}

	if messageContent != "Here's the answer" {
		t.Errorf("Expected 'Here's the answer', got '%s'", messageContent)
	}
}

func TestThinkingParser_NoThinking(t *testing.T) {
	parser := NewThinkingParser()

	thinking, message := parser.Parse("Just a regular message")

	if thinking != nil {
		t.Error("Expected no thinking chunk")
	}

	if message == nil {
		t.Fatal("Expected message chunk, got nil")
	}
	if message.Content != "Just a regular message" {
		t.Errorf("Expected 'Just a regular message', got '%s'", message.Content)
	}
	if message.Type != llm.ContentTypeMessage {
		t.Errorf("Expected message type, got %v", message.Type)
	}
}

func TestThinkingParser_MultipleThinkingBlocks(t *testing.T) {
	parser := NewThinkingParser()

	thinking1, message1 := parser.Parse("<thinking>First thought</thinking>Answer 1<thinking>")
	thinking2, message2 := parser.Parse("Second thought</thinking>Answer 2")

	// First call should get first thinking and first message
	if thinking1 == nil || thinking1.Content != "First thought" {
		t.Errorf("Expected first thinking, got %v", thinking1)
	}
	if message1 == nil || message1.Content != "Answer 1" {
		t.Errorf("Expected first message, got %v", message1)
	}

	// Second call should get second thinking and second message
	if thinking2 == nil || thinking2.Content != "Second thought" {
		t.Errorf("Expected second thinking, got %v", thinking2)
	}
	if message2 == nil || message2.Content != "Answer 2" {
		t.Errorf("Expected second message, got %v", message2)
	}
}

func TestThinkingParser_MessageBeforeThinking(t *testing.T) {
	// Since the parser processes in a single pass, let's test step-by-step parsing
	parser := NewThinkingParser()

	// Parse prefix
	thinking, message := parser.Parse("Prefix text ")
	if message == nil || message.Content != "Prefix text " {
		t.Errorf("Expected 'Prefix text ', got %v", message)
	}
	if thinking != nil {
		t.Errorf("Expected no thinking in prefix, got %v", thinking)
	}

	// Parse thinking block
	thinking, message = parser.Parse("<thinking>Internal thoughts</thinking>")
	if thinking == nil || thinking.Content != "Internal thoughts" {
		t.Errorf("Expected thinking 'Internal thoughts', got %v", thinking)
	}
	if message != nil {
		t.Errorf("Expected no message in thinking block, got %v", message)
	}

	// Parse suffix
	thinking, message = parser.Parse("Suffix text")
	if message == nil || message.Content != "Suffix text" {
		t.Errorf("Expected 'Suffix text', got %v", message)
	}
	if thinking != nil {
		t.Errorf("Expected no thinking in suffix, got %v", thinking)
	}
}

func TestThinkingParser_Reset(t *testing.T) {
	parser := NewThinkingParser()

	// Start parsing
	parser.Parse("<thinking>Some content")

	if !parser.IsInThinking() {
		t.Error("Expected parser to be in thinking mode")
	}

	// Reset
	parser.Reset()

	if parser.IsInThinking() {
		t.Error("Expected parser to not be in thinking mode after reset")
	}

	// Should work normally after reset
	thinking, message := parser.Parse("New message")
	if thinking != nil {
		t.Error("Expected no thinking after reset")
	}
	if message == nil || message.Content != "New message" {
		t.Errorf("Expected 'New message', got %v", message)
	}
}
