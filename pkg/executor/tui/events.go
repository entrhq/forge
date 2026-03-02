package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/executor/tui/overlay"
	"github.com/entrhq/forge/pkg/executor/tui/types"
	pkgtypes "github.com/entrhq/forge/pkg/types"
)

// handleAgentEvent processes events from the agent event stream.
// This is the main event handler that updates the UI based on agent activity.
//
//nolint:gocyclo
func (m *model) handleAgentEvent(event *pkgtypes.AgentEvent) {
	switch event.Type {
	case pkgtypes.EventTypeThinkingStart:
		m.handleThinkingStart()

	case pkgtypes.EventTypeThinkingContent:
		m.handleThinkingContent(event)
		return // Exit early to preserve streaming viewport update

	case pkgtypes.EventTypeThinkingEnd:
		m.handleThinkingEnd()

	case pkgtypes.EventTypeToolCallStart:
		m.handleToolCallStart(event)

	case pkgtypes.EventTypeToolCall:
		m.handleToolCall(event)

	case pkgtypes.EventTypeToolResult:
		m.handleToolResult(event)

	case pkgtypes.EventTypeMessageStart:
		m.handleMessageStart()

	case pkgtypes.EventTypeMessageContent:
		if m.handleMessageContent(event.Content) {
			return // Exit early to preserve streaming viewport update
		}

	case pkgtypes.EventTypeMessageEnd:
		m.handleMessageEnd()

	case pkgtypes.EventTypeError:
		m.handleError(event)

	case pkgtypes.EventTypeTurnEnd:
		m.handleTurnEnd()

	case pkgtypes.EventTypeUpdateBusy:
		m.handleUpdateBusy(event)

	case pkgtypes.EventTypeToolApprovalRequest:
		m.handleToolApprovalRequest(event)

	case pkgtypes.EventTypeToolApprovalGranted:
		m.handleToolApprovalGranted()

	case pkgtypes.EventTypeToolApprovalRejected:
		m.handleToolApprovalRejected()

	case pkgtypes.EventTypeToolApprovalTimeout:
		m.handleToolApprovalTimeout()

	case pkgtypes.EventTypeAPICallStart:
		m.handleApiCallStart(event)

	case pkgtypes.EventTypeTokenUsage:
		m.handleTokenUsage(event)

	case pkgtypes.EventTypeCommandExecutionStart:
		m.handleCommandExecutionStart(event)

	case pkgtypes.EventTypeCommandOutput:
		m.handleCommandExecutionOutput(event)

	case pkgtypes.EventTypeCommandExecutionComplete:
		m.handleCommandExecutionComplete(event)

	case pkgtypes.EventTypeContextSummarizationStart:
		m.handleContextSummarizationStart(event)

	case pkgtypes.EventTypeContextSummarizationProgress:
		m.handleContextSummarizationProgress(event)

	case pkgtypes.EventTypeContextSummarizationComplete:
		m.handleContextSummarizationComplete(event)

	case pkgtypes.EventTypeNotesData:
		m.handleNotesData(event)
	}

	m.recalculateLayout()
}

// scrollToBottomOrMark advances the viewport to the bottom when followScroll is
// true, or marks hasNewContent when the user has scrolled up (ADR-0048).
func (m *model) scrollToBottomOrMark() {
	if m.followScroll {
		// Only scroll to bottom if there's actual content that exceeds viewport height.
		// Calling GotoBottom() when content fits entirely in viewport causes rendering
		// corruption where the terminal scrolls the entire TUI frame upward.
		// Use YPosition + Height to check if we need to scroll without re-rendering.
		if len(m.messages) > 0 {
			totalHeight := m.viewport.TotalLineCount()
			needsScroll := totalHeight > m.viewport.Height

			if needsScroll {
				m.viewport.GotoBottom()
			}
		}
	} else if !m.hasNewContent {
		// First transition false→true: shrink the viewport immediately so the
		// scroll-lock indicator line is reserved before the next render.
		m.hasNewContent = true
		m.viewport.Height = m.calculateViewportHeight()
	}
}

// resumeFollowScroll is the symmetric inverse of the hasNewContent true→false
// transition inside scrollToBottomOrMark. It clears the scroll-lock state and
// immediately expands the viewport height back by the line that was reserved
// for the indicator, preventing a blank gap at the bottom of the view.
func (m *model) resumeFollowScroll() {
	m.followScroll = true
	m.hasNewContent = false
	m.viewport.Height = m.calculateViewportHeight()
}

// Thinking event handlers

func (m *model) handleThinkingStart() {
	m.isThinking = true
	m.thinkingBuffer.Reset()
	m.thinkingStartTime = time.Now()
}

func (m *model) handleThinkingContent(event *pkgtypes.AgentEvent) {
	if event.Content == "" {
		return
	}
	m.thinkingBuffer.WriteString(sanitizeOutput(event.Content))

	// Ephemeral streaming preview — committed messages stay correct at viewport width;
	// the in-progress thinking fragment is appended temporarily.
	base := m.renderMessages(m.viewport.Width)
	if m.showThinking {
		header := "⸫ "
		formatted := formatEntry("", m.thinkingBuffer.String(), thinkingStyle, m.width)
		block := header + formatted
		indented := " " + strings.ReplaceAll(block, "\n", "\n ")
		m.viewport.SetContent(base + indented)
	} else {
		elapsed := int(time.Since(m.thinkingStartTime).Seconds())
		collapsed := thinkingStyle.Render(fmt.Sprintf(" ⸫ Thinking (%ds)", elapsed))
		m.viewport.SetContent(base + collapsed)
	}
	m.scrollToBottomOrMark() // ADR-0048
}

func (m *model) handleThinkingEnd() {
	if m.thinkingBuffer.Len() > 0 {
		// Capture raw text so the closure can reflow at any future width.
		thinkingText := m.thinkingBuffer.String()
		elapsed := int(time.Since(m.thinkingStartTime).Seconds())

		if m.showThinking {
			m.appendMsg(DisplayMessage{
				RenderFn: func(width int) string {
					header := "⸫ "
					formatted := formatEntry("", thinkingText, thinkingStyle, width)
					block := header + formatted
					return " " + strings.ReplaceAll(block, "\n", "\n ")
				},
				Trailing: "\n\n",
			})
		} else {
			// Fixed historical record: "Thought for Xs" — no reflow needed.
			summary := thinkingStyle.Render(fmt.Sprintf(" ⸫ Thought for %ds", elapsed))
			m.appendMsg(newRawMsg(summary, "\n\n"))
		}
	}
	m.isThinking = false
	m.thinkingBuffer.Reset()
}

// Tool event handlers

func (m *model) handleToolCallStart(event *pkgtypes.AgentEvent) {
	// Early detection: if tool name is available in metadata, show it immediately.
	if toolName, ok := event.Metadata["tool_name"].(string); ok && toolName != "" && !m.toolNameDisplayed {
		toolName = sanitizeOutput(toolName)
		m.appendMsg(newEntryMsg("✎ ", toolName, toolStyle, "\n"))
		m.recalculateLayout()
		m.toolNameDisplayed = true
	}
	// If no tool name yet, wait for EventTypeToolCall which always has ToolName.
}

func (m *model) handleToolCall(event *pkgtypes.AgentEvent) {
	// Only display if early detection in handleToolCallStart didn't fire.
	if !m.toolNameDisplayed {
		toolName := sanitizeOutput(event.ToolName)
		m.appendMsg(newEntryMsg("✎ ", toolName, toolStyle, "\n"))
	}
	// Track tool call for result display and caching.
	m.lastToolName = event.ToolName
	if event.ToolCallID != "" {
		m.lastToolCallID = event.ToolCallID
	} else {
		m.lastToolCallID = fmt.Sprintf("%d_%s", time.Now().UnixNano(), event.ToolName)
	}
	m.toolNameDisplayed = false // Reset for next tool call.
}

func (m *model) handleToolResult(event *pkgtypes.AgentEvent) {
	resultStr := sanitizeOutput(fmt.Sprintf("%v", event.ToolOutput))

	// Classify the tool result to determine display strategy.
	tier := m.resultClassifier.ClassifyToolResult(m.lastToolName, resultStr)

	switch tier {
	case TierFullInline:
		m.appendMsg(newEntryMsg("    ✓ ", resultStr, toolResultStyle, "\n\n"))

	case TierSummaryWithPreview:
		summary := m.resultSummarizer.GenerateSummary(m.lastToolName, resultStr)
		preview := m.resultClassifier.GetPreviewLines(resultStr)
		m.appendMsg(newEntryMsg("    ✓ ", summary+"\n"+preview, toolResultStyle, "\n\n"))
		m.resultCache.store(m.lastToolCallID, m.lastToolName, resultStr, summary)

	case TierSummaryOnly:
		summary := m.resultSummarizer.GenerateSummary(m.lastToolName, resultStr)
		m.appendMsg(newEntryMsg("    ✓ ", summary, toolResultStyle, "\n\n"))
		m.resultCache.store(m.lastToolCallID, m.lastToolName, resultStr, summary)

	case TierOverlayOnly:
		// Command execution already handled by overlay; nothing inline.
	}
}

// Message event handlers

func (m *model) handleMessageStart() {
	m.messageBuffer.Reset()
}

func (m *model) handleMessageContent(content string) bool {
	content = sanitizeOutput(content)
	if strings.TrimSpace(content) != "" && !m.hasMessageContentStarted {
		m.hasMessageContentStarted = true
	}

	m.messageBuffer.WriteString(content)

	// Ephemeral streaming preview — committed messages form the base; the
	// in-progress message fragment is appended temporarily at the current width.
	base := m.renderMessages(m.viewport.Width)
	fragment := formatEntry("", m.messageBuffer.String(), lipgloss.NewStyle(), m.width)
	m.viewport.SetContent(base + fragment)
	m.scrollToBottomOrMark() // ADR-0048

	return true
}

func (m *model) handleMessageEnd() {
	if m.messageBuffer.Len() > 0 && m.hasMessageContentStarted {
		// Capture raw text for reflow-capable DisplayMessage.
		msgText := m.messageBuffer.String()
		m.appendMsg(newEntryMsg("", msgText, lipgloss.NewStyle(), "\n\n"))
		m.hasMessageContentStarted = false
	}
	m.messageBuffer.Reset()
}

// Error and state handlers

func (m *model) handleError(event *pkgtypes.AgentEvent) {
	errMsg := truncateLines(fmt.Sprintf("%v", event.Error), 2)
	rendered := warningStyle.Render(fmt.Sprintf(" ⚠ Agent encountered: %s", sanitizeOutput(errMsg)))
	m.appendMsg(newRawMsg(rendered, "\n\n"))
}

func (m *model) handleTurnEnd() {
	m.agentBusy = false
	m.resumeFollowScroll()
	m.recalculateLayout()
}

func (m *model) handleUpdateBusy(event *pkgtypes.AgentEvent) {
	wasBusy := m.agentBusy
	m.agentBusy = event.IsBusy
	if m.agentBusy {
		m.currentLoadingMessage = getRandomLoadingMessage()
	}
	if wasBusy != m.agentBusy {
		m.recalculateLayout()
	}
}

// Tool approval handlers

func (m *model) handleToolApprovalRequest(event *pkgtypes.AgentEvent) {
	m.appendMsg(newEntryMsg("  … ", "Requesting tool approval...", toolStyle, "\n"))
	m.recalculateLayout()

	if event.Preview != nil {
		preview, ok := event.Preview.(*tools.ToolPreview)
		if ok {
			responseFunc := func(response *pkgtypes.ApprovalResponse) {
				m.channels.Approval <- response
				m.overlay.deactivate()
				m.recalculateLayout()
			}

			diffViewer := overlay.NewDiffViewer(
				event.ApprovalID,
				event.ToolName,
				preview,
				m.width,
				m.height,
				responseFunc,
			)
			m.overlay.pushOverlay(types.OverlayModeDiffViewer, diffViewer)
		}
	}
}

func (m *model) handleToolApprovalGranted() {
	m.appendMsg(newEntryMsg("  ✓ ", "Tool approved - executing...", toolStyle, "\n"))
}

func (m *model) handleToolApprovalRejected() {
	m.appendMsg(newEntryMsg("  ✗ ", "Tool rejected by user", warningStyle, "\n"))
}

func (m *model) handleToolApprovalTimeout() {
	m.appendMsg(newEntryMsg("  ⏱ ", "Tool approval timed out", warningStyle, "\n"))
}

// API and token handlers

func (m *model) handleApiCallStart(event *pkgtypes.AgentEvent) {
	if event.APICallInfo != nil {
		m.currentContextTokens = event.APICallInfo.ContextTokens
		m.maxContextTokens = event.APICallInfo.MaxContextTokens
	}
}

func (m *model) handleTokenUsage(event *pkgtypes.AgentEvent) {
	if event.TokenUsage != nil {
		m.totalPromptTokens += event.TokenUsage.PromptTokens
		m.totalCompletionTokens += event.TokenUsage.CompletionTokens
		m.totalTokens += event.TokenUsage.TotalTokens
	}
}

// Command execution handlers

func (m *model) handleCommandExecutionStart(event *pkgtypes.AgentEvent) {
	if event.CommandExecution != nil {
		command := sanitizeOutput(event.CommandExecution.Command)
		m.appendMsg(newEntryMsg("  ❯ ", fmt.Sprintf("Executing: %s", command), toolStyle, "\n"))
		m.recalculateLayout()

		ol := overlay.NewCommandExecutionOverlay(
			event.CommandExecution.Command,
			event.CommandExecution.WorkingDir,
			event.CommandExecution.ExecutionID,
			m.channels.Cancel,
		)
		m.overlay.pushOverlay(types.OverlayModeCommandOutput, ol)
	}
}

func (m *model) handleCommandExecutionOutput(_ *pkgtypes.AgentEvent) {
	// Let overlay handle streaming directly; do not stream bulk to the chat output.
}

func (m *model) handleCommandExecutionComplete(_ *pkgtypes.AgentEvent) {
	// handleToolResult will catch the EventTypeToolResult that follows and
	// write a clean summary using tier logic (TierSummaryWithPreview).
}

// Context summarization handlers

func (m *model) handleContextSummarizationStart(event *pkgtypes.AgentEvent) {
	m.summarization.active = true
	m.summarization.startTime = time.Now()
	if event.ContextSummarization != nil {
		m.summarization.strategy = event.ContextSummarization.Strategy
		m.summarization.currentTokens = event.ContextSummarization.CurrentTokens
		m.summarization.maxTokens = event.ContextSummarization.MaxTokens
	}
}

func (m *model) handleContextSummarizationProgress(_ *pkgtypes.AgentEvent) {
	// Progress details have been removed; wait for completion.
}

func (m *model) handleContextSummarizationComplete(event *pkgtypes.AgentEvent) {
	if event.ContextSummarization != nil {
		oldTokens := m.summarization.currentTokens
		newTokens := event.ContextSummarization.NewTokenCount

		m.summarization.active = false
		duration := time.Since(m.summarization.startTime).Seconds()

		m.showToast(
			"✨ Context optimized",
			fmt.Sprintf("Reduced from %s to %s tokens (%.1fs)",
				formatTokenCount(oldTokens),
				formatTokenCount(newTokens),
				duration),
			"◆",
			false,
		)

		m.currentContextTokens = newTokens
	}
}

// Notes data handler

func (m *model) handleNotesData(event *pkgtypes.AgentEvent) {
	if event.NotesData == nil {
		return
	}

	m.agentBusy = false
	m.currentLoadingMessage = ""

	if !m.pendingNotesRequest {
		return
	}
	m.pendingNotesRequest = false

	notesOverlay := overlay.NewNotesOverlay(event.NotesData.Notes, m.width, m.height)
	m.overlay.pushOverlay(types.OverlayModeNotes, notesOverlay)
}
