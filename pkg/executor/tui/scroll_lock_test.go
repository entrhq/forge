package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// newScrollTestModel returns the minimal model needed for scroll-lock tests.
func newScrollTestModel() *model {
	vp := viewport.New(80, 10)
	// Fill with enough content that the viewport is not trivially at the bottom.
	vp.SetContent(strings.Repeat("line\n", 50))
	vp.GotoBottom()
	return &model{
		viewport:      vp,
		followScroll:  true,
		hasNewContent: false,
		content:       &strings.Builder{},
	}
}

// ---------------------------------------------------------------------------
// scrollToBottomOrMark
// ---------------------------------------------------------------------------

// TestScrollToBottomOrMark_WhenFollowing verifies that scrollToBottomOrMark
// calls GotoBottom() and does not set hasNewContent when followScroll is true.
func TestScrollToBottomOrMark_WhenFollowing(t *testing.T) {
	m := newScrollTestModel()
	m.followScroll = true
	m.hasNewContent = false

	// Manually scroll away from the bottom so we can detect the GotoBottom call.
	m.viewport.LineUp(5)
	if m.viewport.AtBottom() {
		t.Skip("viewport too small to scroll away from bottom")
	}

	m.scrollToBottomOrMark()

	if !m.viewport.AtBottom() {
		t.Error("expected viewport to be at bottom after scrollToBottomOrMark with followScroll=true")
	}
	if m.hasNewContent {
		t.Error("hasNewContent should remain false when followScroll=true")
	}
}

// TestScrollToBottomOrMark_WhenLocked verifies that scrollToBottomOrMark sets
// hasNewContent = true and does NOT move the viewport when followScroll is false.
func TestScrollToBottomOrMark_WhenLocked(t *testing.T) {
	m := newScrollTestModel()
	m.followScroll = false
	m.hasNewContent = false

	// Scroll up so we are away from the bottom.
	m.viewport.LineUp(5)
	if m.viewport.AtBottom() {
		t.Skip("viewport too small to scroll away from bottom")
	}

	offsetBefore := m.viewport.ScrollPercent()
	m.scrollToBottomOrMark()

	if m.viewport.AtBottom() {
		t.Error("viewport should NOT jump to bottom when followScroll=false")
	}
	if m.viewport.ScrollPercent() != offsetBefore {
		t.Error("viewport offset should be unchanged when followScroll=false")
	}
	if !m.hasNewContent {
		t.Error("expected hasNewContent=true after scrollToBottomOrMark with followScroll=false")
	}
}

// ---------------------------------------------------------------------------
// handleScrollKey
// ---------------------------------------------------------------------------

// TestHandleScrollKey_PgUp locks scroll and scrolls up.
func TestHandleScrollKey_PgUp(t *testing.T) {
	m := newScrollTestModel()
	m.followScroll = true

	msg := tea.KeyMsg{Type: tea.KeyPgUp}
	handled, _, _ := m.handleScrollKey(msg, nil, nil, nil)

	if !handled {
		t.Fatal("expected handleScrollKey to handle PgUp")
	}
	if m.followScroll {
		t.Error("followScroll should be false after PgUp")
	}
}

// TestHandleScrollKey_CtrlB locks scroll (alias for PgUp per ADR-0048).
func TestHandleScrollKey_CtrlB(t *testing.T) {
	m := newScrollTestModel()
	m.followScroll = true

	msg := tea.KeyMsg{Type: tea.KeyCtrlB}
	handled, _, _ := m.handleScrollKey(msg, nil, nil, nil)

	if !handled {
		t.Fatal("expected handleScrollKey to handle ctrl+b")
	}
	if m.followScroll {
		t.Error("followScroll should be false after ctrl+b")
	}
}

// TestHandleScrollKey_GKey_WhenLocked resumes following when user presses g.
func TestHandleScrollKey_GKey_WhenLocked(t *testing.T) {
	m := newScrollTestModel()
	m.followScroll = false
	m.hasNewContent = true

	// Scroll away from bottom so we can verify GotoBottom was called.
	m.viewport.LineUp(5)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	handled, _, _ := m.handleScrollKey(msg, nil, nil, nil)

	if !handled {
		t.Fatal("expected handleScrollKey to handle 'g' when locked")
	}
	if !m.followScroll {
		t.Error("followScroll should be true after 'g'")
	}
	if m.hasNewContent {
		t.Error("hasNewContent should be false after 'g'")
	}
	if !m.viewport.AtBottom() {
		t.Error("viewport should be at bottom after 'g'")
	}
}

// TestHandleScrollKey_GKey_WhenAlreadyFollowing does nothing (not handled) so
// normal textarea input of 'g' is not swallowed during normal operation.
func TestHandleScrollKey_GKey_WhenAlreadyFollowing(t *testing.T) {
	m := newScrollTestModel()
	m.followScroll = true

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	handled, _, _ := m.handleScrollKey(msg, nil, nil, nil)

	if handled {
		t.Error("handleScrollKey should not handle 'g' when followScroll=true (would swallow textarea input)")
	}
}

// TestHandleScrollKey_PgDn_ResumesAtBottom resumes following when PgDn is
// pressed and the viewport reaches the bottom.
func TestHandleScrollKey_PgDn_ResumesAtBottom(t *testing.T) {
	m := newScrollTestModel()
	m.followScroll = false
	m.hasNewContent = true

	// Move near the bottom so that a half-page down lands at the bottom.
	m.viewport.GotoBottom()

	msg := tea.KeyMsg{Type: tea.KeyPgDown}
	handled, _, _ := m.handleScrollKey(msg, nil, nil, nil)

	if !handled {
		t.Fatal("expected handleScrollKey to handle PgDown")
	}
	if !m.viewport.AtBottom() {
		t.Skip("viewport did not reach bottom after HalfPageDown; cannot test resume")
	}
	if !m.followScroll {
		t.Error("followScroll should be true after PgDn reaches bottom")
	}
	if m.hasNewContent {
		t.Error("hasNewContent should be false after PgDn reaches bottom")
	}
}

// TestHandleScrollKey_UnknownKey passes unrecognised keys through unhandled.
func TestHandleScrollKey_UnknownKey(t *testing.T) {
	m := newScrollTestModel()

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	handled, _, _ := m.handleScrollKey(msg, nil, nil, nil)

	if handled {
		t.Error("handleScrollKey should not handle Enter")
	}
}
