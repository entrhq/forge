package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// minimalModel returns a *model with only the fields needed for message tests.
func minimalModel() *model {
	return &model{
		messages: nil,
		width:    80,
		viewport: viewport.New(76, 20), // width - 4, matching handleWindowResize
	}
}

// TestAppendMsg_TrimAt500 verifies that appendMsg enforces the 500-message cap
// and retains only the last 400 entries when the cap is exceeded.
func TestAppendMsg_TrimAt500(t *testing.T) {
	m := minimalModel()

	// Fill to exactly the cap.
	for i := 0; i < 500; i++ {
		m.appendMsg(newRawMsg("line", "\n"))
	}
	if len(m.messages) != 500 {
		t.Fatalf("want 500 messages before trim, got %d", len(m.messages))
	}

	// The 501st append should trigger a trim to 400.
	m.appendMsg(newRawMsg("trigger", "\n"))
	if len(m.messages) != 400 {
		t.Fatalf("want 400 messages after trim, got %d", len(m.messages))
	}
}

// TestAppendMsg_RetainsNewest verifies that after a trim the retained entries
// are the most recent ones (i.e. the oldest are discarded, not the newest).
func TestAppendMsg_RetainsNewest(t *testing.T) {
	m := minimalModel()

	// Append 500 messages numbered 0–499.
	for i := 0; i < 500; i++ {
		idx := i // capture for closure
		m.appendMsg(DisplayMessage{
			RenderFn: func(_ int) string { return strings.Repeat("x", idx+1) },
			Trailing: "\n",
		})
	}

	// Trigger trim with message #500 (content "sentinel").
	m.appendMsg(newRawMsg("sentinel", "\n"))

	// After trim we keep the last 400 of the original 501.
	// The first retained message should be #101 (index 101 in the 0-based
	// original run), whose RenderFn returns 102 x's.
	got := m.messages[0].RenderFn(0)
	want := strings.Repeat("x", 102)
	if got != want {
		t.Fatalf("after trim: first retained message: got len=%d want len=%d", len(got), len(want))
	}

	// The last retained message should be our "sentinel".
	last := m.messages[len(m.messages)-1].RenderFn(0)
	if last != "sentinel" {
		t.Fatalf("last message after trim: got %q want %q", last, "sentinel")
	}
}

// TestRenderMessages_PassesWidthToRenderFn verifies that renderMessages calls
// each RenderFn with the exact width argument supplied — not a captured value.
func TestRenderMessages_PassesWidthToRenderFn(t *testing.T) {
	m := minimalModel()

	var gotWidth int
	m.appendMsg(DisplayMessage{
		RenderFn: func(w int) string {
			gotWidth = w
			return "hello"
		},
		Trailing: "",
	})

	m.renderMessages(123)

	if gotWidth != 123 {
		t.Fatalf("renderMessages called RenderFn with width %d, want 123", gotWidth)
	}
}

// TestRenderMessages_Concatenates verifies that renderMessages joins all
// messages' text and trailing strings in order.
func TestRenderMessages_Concatenates(t *testing.T) {
	m := minimalModel()
	m.appendMsg(newRawMsg("alpha", "\n"))
	m.appendMsg(newRawMsg("beta", "\n"))
	m.appendMsg(newRawMsg("gamma", ""))

	got := m.renderMessages(80)
	want := "alpha\nbeta\ngamma"
	if got != want {
		t.Fatalf("renderMessages output:\ngot:  %q\nwant: %q", got, want)
	}
}

// TestRenderMessages_ResizeReflow verifies that the same committed message
// produces different line-wrapping when rendered at different widths.
// This is the core property that makes resize reflow correct.
func TestRenderMessages_ResizeReflow(t *testing.T) {
	m := minimalModel()

	// A long word-wrapped sentence. formatEntry wraps at width-4.
	text := "The quick brown fox jumps over the lazy dog and keeps on running far beyond the horizon"
	m.appendMsg(newEntryMsg("", text, lipgloss.NewStyle(), "\n"))

	narrow := m.renderMessages(40)
	wide := m.renderMessages(120)

	narrowLines := strings.Count(narrow, "\n")
	wideLines := strings.Count(wide, "\n")

	if narrowLines <= wideLines {
		t.Fatalf("expected more lines at width=40 than width=120; got narrow=%d wide=%d lines", narrowLines, wideLines)
	}
}

// TestNewRawMsg_VerbatimIgnoresWidth verifies that newRawMsg returns the
// pre-rendered string unchanged regardless of the width argument.
func TestNewRawMsg_VerbatimIgnoresWidth(t *testing.T) {
	const rendered = "\x1b[31mcolored output\x1b[0m"
	msg := newRawMsg(rendered, "")

	for _, w := range []int{0, 20, 80, 200} {
		got := msg.RenderFn(w)
		if got != rendered {
			t.Errorf("width=%d: got %q, want %q", w, got, rendered)
		}
	}
}

// TestStreamingPreview_DoesNotMutateMessages verifies the streaming pattern:
// reading renderMessages as a base and appending an ephemeral fragment must
// not change the length of m.messages.
func TestStreamingPreview_DoesNotMutateMessages(t *testing.T) {
	m := minimalModel()
	m.appendMsg(newRawMsg("committed message", "\n"))

	before := len(m.messages)

	// Simulate what handleMessageContent does on every streaming tick.
	base := m.renderMessages(m.width)
	fragment := formatEntry("", "...streaming...", lipgloss.NewStyle(), m.width)
	_ = base + fragment // combined string set on viewport — never stored

	after := len(m.messages)

	if before != after {
		t.Fatalf("streaming preview mutated m.messages: before=%d after=%d", before, after)
	}
}
