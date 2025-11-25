package tools

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const (
	defaultServerName = "local"
	maxXMLSize        = 10 * 1024 * 1024 // 10MB limit for XML tool calls
	argumentsTagName  = "arguments"
)

// Compile regex once at package level for efficiency
var toolRegex = regexp.MustCompile(`(?s)<tool>.*?</tool>`)

// ampersandEntityRegex matches ampersands that are already part of XML entities
// to avoid double-escaping them. Matches: &amp; &lt; &gt; &quot; &apos; &#123; &#xAB;
var ampersandEntityRegex = regexp.MustCompile(`&(?:amp|lt|gt|quot|apos|#\d+|#x[0-9a-fA-F]+);`)

// ParseToolCall extracts a tool call from an LLM response that contains
// XML-formatted tool invocations.
//
// Expected format (Pure XML with CDATA):
//
//	<tool>
//	<server_name>local</server_name>
//	<tool_name>apply_diff</tool_name>
//	<arguments>
//	  <path>file.go</path>
//	  <edits>
//	    <edit>
//	      <search><![CDATA[old code]]></search>
//	      <replace><![CDATA[new code]]></replace>
//	    </edit>
//	  </edits>
//	</arguments>
//	</tool>
//
// Returns the parsed ToolCall and the remaining text after removing the tool call,
// or an error if parsing fails.
func ParseToolCall(text string) (*ToolCall, string, error) {
	// Check XML size limit to prevent DOS attacks
	if len(text) > maxXMLSize {
		return nil, text, fmt.Errorf("tool call XML exceeds maximum size of %d bytes", maxXMLSize)
	}

	matches := toolRegex.FindStringSubmatch(text)
	if len(matches) < 1 {
		return nil, text, fmt.Errorf("no tool call found in text")
	}

	// Extract the full <tool> element including tags
	toolXML := strings.TrimSpace(matches[0])

	var toolCall ToolCall
	if err := UnmarshalXMLWithFallback([]byte(toolXML), &toolCall); err != nil {
		// Include XML snippet in error for better debugging
		snippet := toolXML
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, text, fmt.Errorf("failed to unmarshal tool call XML: %w\nXML snippet: %s", err, snippet)
	}

	// Validate required fields
	if toolCall.ToolName == "" {
		return nil, text, fmt.Errorf("tool_name is required in tool call")
	}

	// Server name defaults to "local" if not specified
	if toolCall.ServerName == "" {
		toolCall.ServerName = defaultServerName
	}

	// Remove the tool call from the text
	remainingText := toolRegex.ReplaceAllString(text, "")
	remainingText = strings.TrimSpace(remainingText)

	return &toolCall, remainingText, nil
}

// ExtractThinkingAndToolCall separates thinking content from a tool call.
// If a tool call is found, it returns the thinking text (before the tool call),
// the tool call itself, and any remaining text after the tool call.
// If no tool call is found, it returns the entire text as thinking with nil tool call.
func ExtractThinkingAndToolCall(text string) (thinking string, toolCall *ToolCall, remaining string, err error) {
	if !toolRegex.MatchString(text) {
		return text, nil, "", nil
	}

	loc := toolRegex.FindStringIndex(text)
	if loc == nil {
		return text, nil, "", nil
	}

	thinking = strings.TrimSpace(text[:loc[0]])
	toolCallText := text[loc[0]:loc[1]]
	remaining = strings.TrimSpace(text[loc[1]:])

	toolCall, _, err = ParseToolCall(toolCallText)
	if err != nil {
		return thinking, nil, remaining, err
	}

	return thinking, toolCall, remaining, nil
}

// HasToolCall checks if the text contains a tool call.
func HasToolCall(text string) bool {
	return toolRegex.MatchString(text)
}

// ValidateToolCall checks if a ToolCall has all required fields.
func ValidateToolCall(tc *ToolCall) error {
	if tc == nil {
		return fmt.Errorf("tool call is nil")
	}
	if tc.ToolName == "" {
		return fmt.Errorf("tool_name is required")
	}
	if tc.ServerName == "" {
		return fmt.Errorf("server_name is required")
	}
	return nil
}

// UnmarshalXMLWithFallback attempts to unmarshal XML, with fallback to
// escape unescaped ampersands if the initial parse fails.
// This improves robustness when LLMs generate unescaped & characters.
func UnmarshalXMLWithFallback(data []byte, v interface{}) error {
	// Try normal unmarshaling first
	err := xml.Unmarshal(data, v)
	if err == nil {
		return nil
	}

	// If parse failed, try escaping unescaped ampersands
	escaped := escapeUnescapedAmpersands(data)
	return xml.Unmarshal(escaped, v)
}

// escapeUnescapedAmpersands replaces bare & with &amp; while preserving
// existing entities (&amp;, &lt;, &gt;, &quot;, &apos;, &#..;)
func escapeUnescapedAmpersands(data []byte) []byte {
	// Convert to string for regex processing
	text := string(data)

	// Find all positions of ampersands that are already part of entities
	entityPositions := make(map[int]bool)
	matches := ampersandEntityRegex.FindAllStringIndex(text, -1)
	for _, match := range matches {
		// Mark the position of the & that starts this entity
		entityPositions[match[0]] = true
	}

	// Build result by escaping ampersands that aren't in entityPositions
	var result strings.Builder
	result.Grow(len(text) + 20) // Pre-allocate with some extra space for escapes

	for i := 0; i < len(text); i++ {
		if text[i] == '&' && !entityPositions[i] {
			// This is an unescaped ampersand - escape it
			result.WriteString("&amp;")
		} else {
			result.WriteByte(text[i])
		}
	}

	return []byte(result.String())
}

// XMLToMap converts XML bytes to a map[string]interface{} by parsing the XML structure.
// This is useful for extracting arguments from tool calls in a generic way.
func XMLToMap(data []byte) (map[string]interface{}, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	result := make(map[string]interface{})

	var currentPath []string
	var currentText strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse XML: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Skip the root <arguments> tag
			if t.Name.Local == argumentsTagName && len(currentPath) == 0 {
				currentPath = append(currentPath, t.Name.Local)
				continue
			}
			currentPath = append(currentPath, t.Name.Local)
			currentText.Reset()

		case xml.EndElement:
			if len(currentPath) == 0 {
				continue
			}

			elementName := currentPath[len(currentPath)-1]
			currentPath = currentPath[:len(currentPath)-1]

			// Skip the root </arguments> tag
			if elementName == "arguments" && len(currentPath) == 0 {
				continue
			}

			// Only process elements that are direct children of <arguments>
			if len(currentPath) == 1 && currentPath[0] == "arguments" {
				text := strings.TrimSpace(currentText.String())
				if text != "" {
					result[elementName] = text
				}
			}
			currentText.Reset()

		case xml.CharData:
			currentText.Write(t)
		}
	}

	return result, nil
}
