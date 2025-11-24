package parser

import (
	"testing"
)

// TestToolCallAfterThinkingWithLessThan tests the scenario from the bug report:
// Thinking content contains < and > characters (like "i<10" in code examples),
// followed by a tool call. The buffering logic should not be confused by the
// < characters in the thinking content.
func TestToolCallAfterThinkingWithLessThan(t *testing.T) {
	parser := NewToolCallParser()

	// Simulate the exact scenario from the logs:
	// Thinking content with < and > characters, then </thinking>, then <tool>
	chunks := []string{
		"Looking at the code:\n",
		"1. Line 8: `var x int=5` - missing spaces\n",
		"2. Line 11: `if x>3{` - missing spaces\n",
		"3. Line 15: `for i:=0;i<10;i++{` - missing spaces\n",
		"</thinking>\n\n",
		"<tool>\n",
		"<server_name>local</server_name>\n",
		"<tool_name>execute_command</tool_name>\n",
		"<arguments>\n",
		"<command>go vet sample.go</command>\n",
		"</arguments>\n",
		"</tool>",
	}

	var toolCallContent, regularContent *ParsedContent
	for _, chunk := range chunks {
		tc, rc := parser.Parse(chunk)
		if tc != nil {
			toolCallContent = tc
		}
		if rc != nil {
			if regularContent == nil {
				regularContent = rc
			} else {
				regularContent.Content += rc.Content
			}
		}
	}

	// Flush any remaining content
	tc, rc := parser.Flush()
	if tc != nil {
		toolCallContent = tc
	}
	if rc != nil && regularContent != nil {
		regularContent.Content += rc.Content
	}

	// Verify we got a tool call
	if toolCallContent == nil {
		t.Fatal("Expected to detect tool call, but got nil")
	}

	if toolCallContent.Type != ContentTypeToolCall {
		t.Errorf("Expected ContentTypeToolCall, got %s", toolCallContent.Type)
	}

	expectedToolContent := `<server_name>local</server_name>
<tool_name>execute_command</tool_name>
<arguments>
<command>go vet sample.go</command>
</arguments>`

	if toolCallContent.Content != expectedToolContent {
		t.Errorf("Tool call content mismatch.\nExpected:\n%s\n\nGot:\n%s", expectedToolContent, toolCallContent.Content)
	}

	// Verify regular content contains the thinking text
	if regularContent == nil {
		t.Fatal("Expected regular content with thinking text, but got nil")
	}

	if !contains(regularContent.Content, "i<10") || !contains(regularContent.Content, "x>3") {
		t.Errorf("Regular content should contain thinking text with < and > characters.\nGot: %s", regularContent.Content)
	}
}

// TestToolCallWithHtmlEntities tests tool calls that contain HTML entities like &gt; and &lt;
func TestToolCallWithHtmlEntities(t *testing.T) {
	parser := NewToolCallParser()

	// Tool call with HTML entities in the content
	chunks := []string{
		"<tool>\n",
		"<server_name>local</server_name>\n",
		"<tool_name>apply_diff</tool_name>\n",
		"<arguments>\n",
		"<path>sample.go</path>\n",
		"<edits>\n",
		"<edit>\n",
		"<search>if x&gt;3{</search>\n",
		"<replace>if x &gt; 3 {</replace>\n",
		"</edit>\n",
		"</edits>\n",
		"</arguments>\n",
		"</tool>",
	}

	var toolCallContent *ParsedContent
	for _, chunk := range chunks {
		tc, _ := parser.Parse(chunk)
		if tc != nil && tc.Type == ContentTypeToolCall {
			toolCallContent = tc
		}
	}

	// Flush any remaining content
	tc, _ := parser.Flush()
	if tc != nil && tc.Type == ContentTypeToolCall {
		toolCallContent = tc
	}

	// Verify we got a tool call
	if toolCallContent == nil {
		t.Fatal("Expected to detect tool call with HTML entities, but got nil")
	}

	// The tool call should contain the HTML entities as-is
	if !contains(toolCallContent.Content, "&gt;") {
		t.Errorf("Tool call should preserve HTML entities.\nGot: %s", toolCallContent.Content)
	}
}

// TestBufferingWithLessThanGreaterThan tests that < and > in regular content don't break buffering
func TestBufferingWithLessThanGreaterThan(t *testing.T) {
	parser := NewToolCallParser()

	// Content with various < and > characters that are NOT tags
	chunks := []string{
		"Comparison: x < 5\n",
		"Another: y > 10\n",
		"Math: i<j && k>m\n",
		"<tool>\n",
		"<server_name>local</server_name>\n",
		"<tool_name>test</tool_name>\n",
		"<arguments><param>value</param></arguments>\n",
		"</tool>\n",
		"More text: a<b>c\n",
	}

	var toolCallFound bool
	var regularContent string

	for _, chunk := range chunks {
		tc, rc := parser.Parse(chunk)
		if tc != nil && tc.Type == ContentTypeToolCall {
			toolCallFound = true
		}
		if rc != nil {
			regularContent += rc.Content
		}
	}

	tc, rc := parser.Flush()
	if tc != nil && tc.Type == ContentTypeToolCall {
		toolCallFound = true
	}
	if rc != nil {
		regularContent += rc.Content
	}

	if !toolCallFound {
		t.Error("Tool call should be detected despite < and > in surrounding content")
	}

	// Regular content should contain the comparison operators
	if !contains(regularContent, "x < 5") || !contains(regularContent, "y > 10") {
		t.Errorf("Regular content should preserve < and > characters.\nGot: %s", regularContent)
	}
}
