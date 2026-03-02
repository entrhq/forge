package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DisplayMessage is an immutable unit of chat history owned by the TUI.
// It stores the raw text and a render function that is called at the current
// terminal width on demand — enabling correct reflow on window resize.
//
// The agent's conversation memory is separate and may be compacted; this slice
// is append-only and never pruned by the agent.
type DisplayMessage struct {
	// RenderFn is called with the current terminal width each time the
	// viewport content is rebuilt. It must be a pure function of width.
	RenderFn func(width int) string

	// Trailing is appended verbatim after RenderFn's output.
	// Typically "\n" for inline items or "\n\n" for paragraph breaks.
	Trailing string
}

// renderMessages re-renders all messages at the given width and returns the
// combined string for use with viewport.SetContent. This is the only path
// that produces the viewport string — never pre-rendered ANSI buffers.
func (m *model) renderMessages(width int) string {
	var b strings.Builder
	for _, msg := range m.messages {
		b.WriteString(msg.RenderFn(width))
		b.WriteString(msg.Trailing)
	}
	return b.String()
}

// appendMsg appends a DisplayMessage to the conversation slice and trims the
// oldest entries when the slice exceeds maxMessages.
func (m *model) appendMsg(msg DisplayMessage) {
	const maxMessages = 500
	const retainMessages = 400
	m.messages = append(m.messages, msg)
	if len(m.messages) > maxMessages {
		m.messages = m.messages[len(m.messages)-retainMessages:]
	}
}

// newEntryMsg returns a DisplayMessage whose output is produced by formatEntry
// at render time with the supplied icon, raw text, and style.
// Use this for all standard chat entries.
func newEntryMsg(icon, text string, style lipgloss.Style, trailing string) DisplayMessage {
	return DisplayMessage{
		RenderFn: func(width int) string {
			return formatEntry(icon, text, style, width)
		},
		Trailing: trailing,
	}
}

// newRawMsg returns a DisplayMessage that emits pre-rendered text verbatim,
// ignoring width. Use only when the caller has already handled formatting
// (e.g. bash prompt lines rendered with bashPromptStyle).
func newRawMsg(text, trailing string) DisplayMessage {
	return DisplayMessage{
		RenderFn: func(_ int) string { return text },
		Trailing: trailing,
	}
}
