package retrieval

import (
	"strings"

	"github.com/entrhq/forge/pkg/types"
)

const (
	// windowMessages is the number of recent messages to include in the
	// conversation window passed to the HyDE generator.
	windowMessages = 6

	// windowMaxChars caps the total character budget for the window string so
	// that very long messages don't inflate the HyDE prompt unnecessarily.
	windowMaxChars = 2000
)

// buildWindow returns a compact string representation of the last N messages
// from history, trimmed to windowMaxChars. Tool messages are omitted because
// their content is generally not useful for memory retrieval queries.
func buildWindow(history []*types.Message, userMessage string) string {
	// Collect the tail of the history.
	msgs := history
	if len(msgs) > windowMessages {
		msgs = msgs[len(msgs)-windowMessages:]
	}

	var sb strings.Builder
	for _, m := range msgs {
		if m.Role == types.RoleTool {
			continue
		}
		role := string(m.Role)
		sb.WriteString(role)
		sb.WriteString(": ")
		sb.WriteString(m.Content)
		sb.WriteString("\n")
	}
	if userMessage != "" {
		sb.WriteString("user: ")
		sb.WriteString(userMessage)
		sb.WriteString("\n")
	}

	out := sb.String()
	if len(out) > windowMaxChars {
		out = out[len(out)-windowMaxChars:]
	}
	return out
}
