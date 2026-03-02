package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"

	"github.com/entrhq/forge/pkg/executor/tui/overlay"
)

// newHeightTestModel returns a minimal model for viewport height calculation tests.
func newHeightTestModel(terminalHeight int) *model {
	vp := viewport.New(80, 10)
	ta := textarea.New()
	ta.SetWidth(80)
	ta.SetHeight(1) // Default to 1-line height for baseline tests

	return &model{
		viewport:       vp,
		textarea:       ta,
		height:         terminalHeight,
		width:          80,
		followScroll:   true,
		hasNewContent:  false,
		agentBusy:      false,
		ready:          true,
		workspaceDir:   "/test",
		overlay:        newOverlayState(),
		commandPalette: overlay.NewCommandPalette(nil),
		resultList:     overlay.NewResultListModel(),
		summarization:  &summarizationStatus{},
		toast:          &toastNotification{},
	}
}

// TestCalculateViewportHeight_AccountsForSpacer verifies that the viewport
// height calculation subtracts the 1-line spacer added by assembleBaseView.
// This test prevents regression of the bug fixed in ADR-0052.
func TestCalculateViewportHeight_AccountsForSpacer(t *testing.T) {
	tests := []struct {
		name             string
		terminalHeight   int
		textareaLines    int
		agentBusy        bool
		hasNewContent    bool
		expectedViewport int
	}{
		{
			name:           "baseline 20-line terminal, 1-line textarea",
			terminalHeight: 20,
			textareaLines:  1,
			agentBusy:      false,
			hasNewContent:  false,
			// Formula: height - headerHeight(4) - spacerHeight(1) - inputZoneHeight(1+textareaH) - statusBarHeight(1)
			// 20 - 4 - 1 - 2 - 1 = 12
			expectedViewport: 12,
		},
		{
			name:           "with loading indicator",
			terminalHeight: 20,
			textareaLines:  1,
			agentBusy:      true,
			hasNewContent:  false,
			// 20 - 4 - 1 - 2 - 1 - 1(loading) = 11
			expectedViewport: 11,
		},
		{
			name:           "with scroll indicator",
			terminalHeight: 20,
			textareaLines:  1,
			agentBusy:      false,
			hasNewContent:  true,
			// 20 - 4 - 1 - 2 - 1 - 1(scroll) = 11
			expectedViewport: 11,
		},
		{
			name:           "with multi-line textarea",
			terminalHeight: 20,
			textareaLines:  3,
			agentBusy:      false,
			hasNewContent:  false,
			// 20 - 4 - 1 - 4(1+3) - 1 = 10
			expectedViewport: 10,
		},
		{
			name:           "small terminal",
			terminalHeight: 10,
			textareaLines:  1,
			agentBusy:      false,
			hasNewContent:  false,
			// 10 - 4 - 1 - 2 - 1 = 2
			expectedViewport: 2,
		},
		{
			name:             "minimum viewport (should never go below 1)",
			terminalHeight:   5,
			textareaLines:    1,
			agentBusy:        false,
			hasNewContent:    false,
			expectedViewport: 1, // floor enforced by calculateViewportHeight
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newHeightTestModel(tt.terminalHeight)
			m.agentBusy = tt.agentBusy
			m.hasNewContent = tt.hasNewContent
			m.followScroll = !tt.hasNewContent // scroll indicator only shows when locked

			// Set textarea to have the specified number of lines
			// Set textarea to have the specified number of lines
			if tt.textareaLines > 1 {
				m.textarea.SetValue(strings.Repeat("line\n", tt.textareaLines-1) + "line")
				m.textarea.SetHeight(tt.textareaLines)
			} else {
				m.textarea.SetValue("")
				m.textarea.SetHeight(1)
			}

			got := m.calculateViewportHeight()

			if got != tt.expectedViewport {
				t.Errorf("calculateViewportHeight() = %d, want %d\nConfiguration: height=%d, textareaLines=%d, agentBusy=%v, hasNewContent=%v",
					got, tt.expectedViewport, tt.terminalHeight, tt.textareaLines, tt.agentBusy, tt.hasNewContent)
			}
		})
	}
}

// TestCalculateViewportHeight_SpacerConstant verifies that the spacerHeight
// constant is defined and equals 1, as required by the fix in ADR-0052.
func TestCalculateViewportHeight_SpacerConstant(t *testing.T) {
	m := newHeightTestModel(20)

	// The spacer is a constant inside calculateViewportHeight, but we can verify
	// its effect by checking that the calculation matches our expected formula.
	// For a 20-line terminal with 1-line textarea and no indicators:
	// viewport = 20 - 4(header) - 1(spacer) - 2(input) - 1(status) = 12

	expected := 12
	got := m.calculateViewportHeight()

	if got != expected {
		t.Errorf("calculateViewportHeight() with baseline config = %d, want %d (spacer may not be accounted for)", got, expected)
	}
}

// TestViewportHeight_NoOverflow verifies that assembleBaseView produces output
// that matches the terminal height exactly when all components are accounted for.
func TestViewportHeight_NoOverflow(t *testing.T) {
	tests := []struct {
		name           string
		terminalHeight int
		textareaValue  string
		agentBusy      bool
	}{
		{
			name:           "20-line terminal, empty textarea",
			terminalHeight: 20,
			textareaValue:  "",
			agentBusy:      false,
		},
		{
			name:           "20-line terminal, multi-line textarea",
			terminalHeight: 20,
			textareaValue:  "line1\nline2\nline3",
			agentBusy:      false,
		},
		{
			name:           "15-line terminal, agent busy",
			terminalHeight: 15,
			textareaValue:  "",
			agentBusy:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newHeightTestModel(tt.terminalHeight)
			m.agentBusy = tt.agentBusy
			m.textarea.SetValue(tt.textareaValue)

			// Set viewport content so it's not empty
			m.viewport.SetContent(strings.Repeat("content line\n", 50))

			// Recalculate layout to update viewport height
			m.recalculateLayout()

			// Render the full view
			view := m.View()

			// Count actual lines in output
			actualLines := strings.Count(view, "\n") + 1

			if actualLines > tt.terminalHeight {
				t.Errorf("View() overflow: produced %d lines for terminal height %d (overflow=%d)",
					actualLines, tt.terminalHeight, actualLines-tt.terminalHeight)

				// Debug: show which component heights were calculated
				t.Logf("Debug: viewport.Height=%d, textarea.Height()=%d, agentBusy=%v",
					m.viewport.Height, m.textarea.Height(), m.agentBusy)
			}
		})
	}
}
