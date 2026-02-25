package retrieval

import (
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/agent/longtermmemory"
)

const injectionHeader = `<long_term_memory>
The following memories were retrieved from long-term storage as relevant context for this conversation. Use them to inform your responses but do not repeat them verbatim unless asked.
`

const injectionFooter = `</long_term_memory>`

// FormatInjection formats a slice of memory files into a string suitable for
// prepending to the system prompt. The tokenBudget parameter sets a soft
// character ceiling (tokenBudget * 4); passing 0 disables the cap.
func FormatInjection(memories []*longtermmemory.MemoryFile, tokenBudget int) string {
	if len(memories) == 0 {
		return ""
	}

	charBudget := 0
	if tokenBudget > 0 {
		charBudget = tokenBudget * 4
	}

	var sb strings.Builder
	sb.WriteString(injectionHeader)

	used := len(injectionHeader) + len(injectionFooter)
	for i, m := range memories {
		entry := formatEntry(i+1, m)
		if charBudget > 0 && used+len(entry) > charBudget {
			break
		}
		sb.WriteString(entry)
		used += len(entry)
	}

	sb.WriteString(injectionFooter)
	return sb.String()
}

func formatEntry(n int, m *longtermmemory.MemoryFile) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n## Memory %d", n))
	if m.Meta.Category != "" {
		sb.WriteString(fmt.Sprintf(" [%s]", m.Meta.Category))
	}
	sb.WriteString("\n")
	sb.WriteString(strings.TrimSpace(m.Content))
	sb.WriteString("\n")
	return sb.String()
}
