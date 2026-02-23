package capture

import (
	"regexp"
	"strings"

	"github.com/entrhq/forge/pkg/types"
)

// toolCallPattern matches XML tool call blocks as used by Forge (ADR-0019).
// The <tool>...</tool> blocks may span multiple lines, hence the (?s) flag.
var toolCallPattern = regexp.MustCompile(`(?s)<tool>.*?</tool>`)

// toolResultPattern matches residual <tool_result> wrappers that may appear
// inside assistant messages alongside human-language prose.
var toolResultPattern = regexp.MustCompile(`(?s)<tool_result>.*?</tool_result>`)

// StripToolContent filters a message list to retain only human-language
// user and assistant content. Messages with the "tool" role are dropped entirely.
// Tool call blocks and tool result blocks embedded in user/assistant messages are
// also removed, leaving only prose.
//
// This is the preprocessing step required before enqueueing a TriggerEvent
// (ADR-0046 §Tool Call Content Stripping).
func StripToolContent(messages []*types.Message) []ConversationMessage {
	out := make([]ConversationMessage, 0, len(messages))
	for _, msg := range messages {
		if msg.Role != types.RoleUser && msg.Role != types.RoleAssistant {
			continue
		}
		text := extractTextContent(msg)
		if text == "" {
			continue
		}
		out = append(out, ConversationMessage{
			Role:    string(msg.Role),
			Content: text,
		})
	}
	return out
}

// extractTextContent returns the human-language portions of a message,
// stripping XML tool call syntax and tool result blocks (ADR-0019).
//
// Returns an empty string if the message contained only tool-related content
// with no accompanying prose.
func extractTextContent(msg *types.Message) string {
	text := msg.Content
	text = toolCallPattern.ReplaceAllString(text, "")
	text = toolResultPattern.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}
