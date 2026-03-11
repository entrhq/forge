package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrhq/forge/pkg/executor/tui/overlay"
	tuitypes "github.com/entrhq/forge/pkg/executor/tui/types"
	"github.com/entrhq/forge/pkg/types"
)

const (
	// Layout dimensions: horizontal padding for viewport and textarea
	viewportHorizontalPadding = 4 // 2 chars left + 2 chars right border/margin
	textareaHorizontalPadding = 8 // Wider padding for textarea (includes prompt space)

	// Header layout: compact header + separator + hints + blank spacer
	headerHeight = 4
)

// Update handles all state updates for the TUI model.
// This is the main event loop handler for Bubble Tea.
//
// Uses pointer receiver to ensure overlay mutations via ActionHandler persist.
// Without pointer receiver, &m passed to overlays points to a local copy,
// causing state changes (SetInput, ShowToast, etc.) to be lost.
//
//nolint:gocyclo
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check if quit was requested by an overlay or component
	if m.shouldQuit {
		return m, tea.Quit
	}

	var tiCmd, vpCmd, spinnerCmd tea.Cmd
	m.spinner, spinnerCmd = m.spinner.Update(msg)

	// Forward ALL messages to active overlay first (including custom messages like cursorBlinkMsg).
	// This ensures overlays can handle their own custom message types.
	if m.overlay.isActive() && m.overlay.overlay != nil {
		updatedOverlay, overlayCmd := m.overlay.overlay.Update(msg, m, m)

		// If overlay returns nil, it wants to close.
		if updatedOverlay == nil {
			m.ClearOverlay()
			if overlayCmd != nil {
				spinnerCmd = tea.Batch(spinnerCmd, overlayCmd)
			}
			// Continue processing the message in the main model.
		} else {
			m.overlay.overlay = updatedOverlay

			// For KeyMsg and MouseMsg, we still need to handle them in the main model too.
			// For other message types, the overlay handling is sufficient.
			shouldFallThrough := false
			switch msg.(type) {
			case tea.KeyMsg, tea.MouseMsg:
				shouldFallThrough = true
			case tea.WindowSizeMsg:
				// The overlay handles its own resize in its Update call above.
				// The main model must also process WindowSizeMsg so m.width/m.height
				// stay in sync and recalculateLayout re-wraps messages at new width.
				shouldFallThrough = true
			case *types.AgentEvent:
				shouldFallThrough = true
			case approvalRequestMsg:
				shouldFallThrough = true
			case agentErrMsg:
				shouldFallThrough = true
			case operationCompleteMsg, tuitypes.OperationCompleteMsg:
				shouldFallThrough = true
			case toastMsg, tuitypes.ToastMsg:
				shouldFallThrough = true
			}

			if !shouldFallThrough {
				return m, tea.Batch(overlayCmd, spinnerCmd)
			}

			spinnerCmd = tea.Batch(overlayCmd, spinnerCmd)
		}
	}

	// Handle command palette keyboard input BEFORE updating textarea.
	// This prevents Enter from being processed by textarea when palette is active.
	if keyMsg, ok := msg.(tea.KeyMsg); ok && m.commandPalette.IsActive() {
		switch keyMsg.Type {
		case tea.KeyEsc:
			m.commandPalette.Deactivate()
			m.textarea.Reset()
			return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
		case tea.KeyUp:
			m.commandPalette.SelectPrev()
			return m, spinnerCmd
		case tea.KeyDown:
			m.commandPalette.SelectNext()
			return m, spinnerCmd
		case tea.KeyTab:
			selected := m.commandPalette.GetSelected()
			if selected != nil {
				m.textarea.SetValue("/" + selected.Name + " ")
				m.textarea.CursorEnd()
			}
			m.commandPalette.Deactivate()
			return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
		case tea.KeyEnter:
			selected := m.commandPalette.GetSelected()
			if selected != nil {
				m.textarea.SetValue("/" + selected.Name)
			}
			m.commandPalette.Deactivate()
			return m.handleEnter(tiCmd, vpCmd, spinnerCmd)
		}
	}

	// ADR-0048: intercept 'g' key for scroll-lock BEFORE textarea update.
	if keyMsg, ok := msg.(tea.KeyMsg); ok && !m.followScroll && keyMsg.String() == "g" {
		m.resumeFollowScroll()
		m.viewport.GotoBottom()
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
	}

	// Only update textarea if no overlay or result list is active.
	// This prevents the textarea from capturing scroll events when an overlay is open.
	if !m.overlay.isActive() && !m.resultList.IsActive() {
		m.textarea, tiCmd = m.textarea.Update(msg)

		// Update height based on visual wrapping after every update.
		// updateTextAreaHeight() calls recalculateLayout() internally when height changes,
		// so no need to call it again here.
		if m.ready {
			m.updateTextAreaHeight()
		}

		// Handle command palette activation/deactivation based on input.
		value := m.textarea.Value()
		switch {
		case value == "/" && !m.commandPalette.IsActive():
			m.commandPalette.Activate()
			m.commandPalette.UpdateFilter("")
		case strings.HasPrefix(value, "/") && m.commandPalette.IsActive():
			m.commandPalette.UpdateFilter(strings.TrimPrefix(value, "/"))
		case !strings.HasPrefix(value, "/") && m.commandPalette.IsActive():
			m.commandPalette.Deactivate()
		}
	}

	switch msg := msg.(type) {
	case tuitypes.ViewResultMsg:
		m.handleViewResult(msg)
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)

	case tuitypes.ViewNoteMsg:
		m.handleViewNote(msg)
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)

	case tea.WindowSizeMsg:

		return m.handleWindowResize(msg)

	case slashCommandCompleteMsg:
		return m.handleSlashCommandComplete()

	case operationStartMsg:
		return m.handleOperationStart(msg)

	case tuitypes.OperationStartMsg:
		return m.handleOperationStart(operationStartMsg{message: msg.Message})

	case operationCompleteMsg:
		return m.handleOperationComplete(msg)

	case tuitypes.OperationCompleteMsg:
		return m.handleOperationComplete(operationCompleteMsg{
			result:       msg.Result,
			err:          msg.Err,
			successTitle: msg.SuccessTitle,
			successIcon:  msg.SuccessIcon,
			errorTitle:   msg.ErrorTitle,
			errorIcon:    msg.ErrorIcon,
		})

	case toastMsg:
		return m.handleToast(msg)

	case tuitypes.ToastMsg:
		return m.handleToast(toastMsg{
			message: msg.Message,
			details: msg.Details,
			icon:    msg.Icon,
			isError: msg.IsError,
		})

	case agentErrMsg:

		return m.handleAgentError(msg)

	case bashCommandResultMsg:
		return m.handleBashCommandResult(msg)

	case approvalRequestMsg:

		return m.handleApprovalRequest(msg)

	case *types.AgentEvent:
		// Note: AgentEvent forwarding to overlay is handled in the early forwarding section above.
		m.viewport, vpCmd = m.viewport.Update(msg)
		m.handleAgentEvent(msg)
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)

	case tea.MouseMsg:
		// Note: Mouse event forwarding to overlay is handled in the early forwarding section above.
		if !m.overlay.isActive() {
			// ADR-0048: track scroll direction before updating viewport.
			if msg.Button == tea.MouseButtonWheelUp {
				m.followScroll = false
			}
			m.viewport, vpCmd = m.viewport.Update(msg)
			if msg.Button == tea.MouseButtonWheelDown && m.viewport.AtBottom() {
				m.resumeFollowScroll()
			}
		}
		return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d", "esc", "enter", "tab", "up", "down", "pgup", "pgdown":
		}
		return m.handleKeyPress(msg, vpCmd, tiCmd, spinnerCmd)

	default:
		// Filter out high-frequency framework messages to prevent excessive
		// "unknown message type" logging from bubbles/viewport.
		switch msg.(type) {
		case spinner.TickMsg:
			if m.agentBusy {
				return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
			}
			return m, spinnerCmd
		case cursor.BlinkMsg:
			return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
		}
	}
	// Do NOT pass tea.KeyMsg to viewport.Update. The bubbles viewport has its own
	// built-in space/arrow key bindings that scroll independently of our layout —
	// forwarding key events here causes the viewport to scroll on Space/arrow presses
	// even while the user is typing. All viewport navigation is handled explicitly
	// in handleScrollKey. Only non-key messages (mouse, spinner ticks, etc.) need
	// to be forwarded to the viewport component.
	switch msg.(type) {
	case tea.KeyMsg, tea.WindowSizeMsg:
		// Do not forward KeyMsg or WindowSizeMsg to viewport
		// WindowSizeMsg makes the viewport internally set its viewport.Height
		// to the entire terminal window height (destroying our layout logic)
	default:
		m.viewport, vpCmd = m.viewport.Update(msg)
	}

	return m, tea.Batch(tiCmd, vpCmd, spinnerCmd)
}

// calculateViewportHeight computes the appropriate viewport height based on current model state.
func (m *model) calculateViewportHeight() int {
	// inputZoneHeight accounts for:
	//   - 1 line for the horizontal rule (──────)
	//   - 1 line for the prompt + textarea first line (❯ <input>)
	//   - additional lines if textarea wraps (m.textarea.Height() - 1)
	// So: rule(1) + prompt+first_line(1) + additional_textarea_lines = 2 + (m.textarea.Height() - 1)
	inputZoneHeight := 1 + m.textarea.Height()
	statusBarHeight := 1

	loadingHeight := 0
	if m.agentBusy {
		loadingHeight = 1
	}

	scrollIndicatorHeight := 0
	if !m.followScroll && m.hasNewContent {
		scrollIndicatorHeight = 1
	}

	// Visual spacer line between header and viewport (assembleBaseView line 231 adds "")
	const spacerHeight = 1
	viewportHeight := max(m.height-headerHeight-spacerHeight-inputZoneHeight-statusBarHeight-loadingHeight-scrollIndicatorHeight, 1)
	return viewportHeight
}

// handleWindowResize processes window size change events.
func (m *model) handleWindowResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Do NOT call m.viewport.Update(msg) here. The bubbles viewport component
	// sets its internal Height to msg.Height (full terminal size) and adjusts
	// scroll offsets for that full height. We then set a smaller correct height,
	// leaving the offsets calibrated for the wrong dimensions — which causes the
	// viewport to snap/jump on the next recalculateLayout call (e.g. when space
	// causes word-wrap after a height resize). We own the viewport dimensions
	// directly; no need to delegate WindowSizeMsg to the component itself.

	m.width = msg.Width
	m.height = msg.Height

	maxInputLines := max(m.height/3, 1)
	m.textarea.MaxHeight = maxInputLines

	m.textarea.SetWidth(m.width - textareaHorizontalPadding - promptWidth)
	m.viewport.Width = m.width - viewportHorizontalPadding
	m.ready = true

	// Do NOT call GotoBottom() here - let recalculateLayout handle scroll positioning
	// via scrollToBottomOrMark(). Calling GotoBottom() before recalculateLayout shrinks
	// viewport.Height causes a double-GotoBottom: once here (with stale height), once
	// inside scrollToBottomOrMark (with new height), leaving YOffset corrupted.

	m.updateTextAreaHeight()
	m.recalculateLayout()

	// Also update result list if active.
	if m.resultList.IsActive() {
		updated, listCmd := m.resultList.Update(msg, m, m)
		if rl, ok := updated.(*overlay.ResultListModel); ok {
			m.resultList = *rl
		}
		cmd = listCmd
	}

	return m, cmd
}

// recalculateLayout updates the viewport height and re-renders content.
// It re-renders all messages at the current viewport width so window resize causes
// correct reflow rather than clipping pre-wrapped ANSI strings.
func (m *model) recalculateLayout() {
	newVpHeight := m.calculateViewportHeight()

	m.viewport.Height = newVpHeight

	renderedContent := m.renderMessages(m.viewport.Width)

	m.viewport.SetContent(renderedContent)
	// ADR-0048: scrollToBottomOrMark updates viewport.Height itself on the
	// first false→true transition of hasNewContent, so no second call needed.
	m.scrollToBottomOrMark()
}
