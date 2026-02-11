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
		debugLog.Printf("Processing EventTypeError: %v", event.Error)
		m.handleError(event)

	case pkgtypes.EventTypeTurnEnd:
		m.handleTurnEnd()

	case pkgtypes.EventTypeUpdateBusy:
		m.handleUpdateBusy(event)

	case pkgtypes.EventTypeToolApprovalRequest:
		debugLog.Printf("Processing EventTypeToolApprovalRequest")
		m.handleToolApprovalRequest(event)

	case pkgtypes.EventTypeToolApprovalGranted:
		debugLog.Printf("Processing EventTypeToolApprovalGranted")
		m.handleToolApprovalGranted()

	case pkgtypes.EventTypeToolApprovalRejected:
		debugLog.Printf("Processing EventTypeToolApprovalRejected")
		m.handleToolApprovalRejected()

	case pkgtypes.EventTypeToolApprovalTimeout:
		m.handleToolApprovalTimeout()

	case pkgtypes.EventTypeApiCallStart:
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

	// Update viewport with current content
	m.viewport.SetContent(m.content.String())
	m.viewport.GotoBottom()
}

// Thinking event handlers

func (m *model) handleThinkingStart() {
	m.isThinking = true
	m.thinkingBuffer.Reset()
}

func (m *model) handleThinkingContent(event *pkgtypes.AgentEvent) {
	if event.Content == "" {
		return
	}
	// Buffer the thinking content
	m.thinkingBuffer.WriteString(event.Content)
	// Stream with "Thinking" label, content follows immediately
	header := "💭 Thinking "
	formatted := formatEntry("", m.thinkingBuffer.String(), thinkingStyle, m.width, false)
	m.viewport.SetContent(m.content.String() + header + formatted)
	m.viewport.GotoBottom()
}

func (m *model) handleThinkingEnd() {
	if m.thinkingBuffer.Len() > 0 {
		header := "💭 Thinking "
		formatted := formatEntry("", m.thinkingBuffer.String(), thinkingStyle, m.width, false)
		m.content.WriteString(header + formatted)
	}
	m.content.WriteString("\n\n")
	m.isThinking = false
	m.thinkingBuffer.Reset()
}

// Tool event handlers

func (m *model) handleToolCallStart(event *pkgtypes.AgentEvent) {
	// Check if we have early tool name detection in metadata
	if toolName, ok := event.Metadata["tool_name"].(string); ok && toolName != "" && !m.toolNameDisplayed {
		// Display the tool name immediately when detected early
		formatted := formatEntry("🔧 ", toolName, toolStyle, m.width, false)
		m.content.WriteString(formatted)
		m.content.WriteString("\n")
		m.viewport.SetContent(m.content.String())
		m.viewport.GotoBottom()
		m.toolNameDisplayed = true
	}
	// If no tool name yet, we'll wait for EventTypeToolCall
}

func (m *model) handleToolCall(event *pkgtypes.AgentEvent) {
	// Only display if we haven't already shown it from early detection
	if !m.toolNameDisplayed {
		formatted := formatEntry("🔧 ", event.ToolName, toolStyle, m.width, false)
		m.content.WriteString(formatted)
		m.content.WriteString("\n")
	}
	// Track tool call for result display
	m.lastToolName = event.ToolName
	// Generate a simple cache key using timestamp + tool name
	m.lastToolCallID = fmt.Sprintf("%d_%s", time.Now().UnixNano(), event.ToolName)
	m.toolNameDisplayed = false // Reset for next tool call
}

func (m *model) handleToolResult(event *pkgtypes.AgentEvent) {
	resultStr := fmt.Sprintf("%v", event.ToolOutput)

	// Classify the tool result to determine display strategy
	tier := m.resultClassifier.ClassifyToolResult(m.lastToolName, resultStr)

	switch tier {
	case TierFullInline:
		// Display full result inline (loop-breaking tools)
		formatted := formatEntry("    ✓ ", resultStr, toolResultStyle, m.width, false)
		m.content.WriteString(formatted)

	case TierSummaryWithPreview:
		// Display summary + preview lines
		summary := m.resultSummarizer.GenerateSummary(m.lastToolName, resultStr)
		preview := m.resultClassifier.GetPreviewLines(resultStr)
		displayText := summary + "\n" + preview
		formatted := formatEntry("    ✓ ", displayText, toolResultStyle, m.width, false)
		m.content.WriteString(formatted)
		// Cache the full result for viewing
		m.resultCache.store(m.lastToolCallID, m.lastToolName, resultStr, summary)

	case TierSummaryOnly:
		// Display summary only
		summary := m.resultSummarizer.GenerateSummary(m.lastToolName, resultStr)
		formatted := formatEntry("    ✓ ", summary, toolResultStyle, m.width, false)
		m.content.WriteString(formatted)
		// Cache the full result for viewing
		m.resultCache.store(m.lastToolCallID, m.lastToolName, resultStr, summary)

	case TierOverlayOnly:
		// Command execution already handled by overlay system
		// Don't display anything inline
	}

	m.content.WriteString("\n\n")
}

// Message event handlers

func (m *model) handleMessageStart() {
	m.messageBuffer.Reset()
}

func (m *model) handleMessageContent(content string) bool {
	if strings.TrimSpace(content) != "" && !m.hasMessageContentStarted {
		m.hasMessageContentStarted = true
	}

	// Buffer the message content
	m.messageBuffer.WriteString(content)

	// Stream message content as it arrives
	formatted := formatEntry("", m.messageBuffer.String(), lipgloss.NewStyle(), m.width, false)
	m.viewport.SetContent(m.content.String() + formatted)
	m.viewport.GotoBottom()

	return true
}

func (m *model) handleMessageEnd() {
	// Finalize message content (like thinking does)
	if m.messageBuffer.Len() > 0 && m.hasMessageContentStarted {
		formatted := formatEntry("", m.messageBuffer.String(), lipgloss.NewStyle(), m.width, false)
		m.content.WriteString(formatted)
		m.content.WriteString("\n\n")
		m.hasMessageContentStarted = false
	}
	m.messageBuffer.Reset()
}

// Error and state handlers

func (m *model) handleError(event *pkgtypes.AgentEvent) {
	m.content.WriteString(errorStyle.Render(fmt.Sprintf("  ❌ Error: %v", event.Error)))
	m.content.WriteString("\n\n")
}

func (m *model) handleTurnEnd() {
	// Turn end - clear busy state
	m.agentBusy = false
	m.recalculateLayout()
}

func (m *model) handleUpdateBusy(event *pkgtypes.AgentEvent) {
	// Update busy state based on event
	wasBusy := m.agentBusy
	m.agentBusy = event.IsBusy
	if m.agentBusy {
		// Pick a random loading message when becoming busy
		m.currentLoadingMessage = getRandomLoadingMessage()
	}
	// Recalculate layout if busy state changed
	if wasBusy != m.agentBusy {
		m.recalculateLayout()
	}
}

// Tool approval handlers

func (m *model) handleToolApprovalRequest(event *pkgtypes.AgentEvent) {
	// Show "Requesting approval" message before overlay
	formatted := formatEntry("  ⏳ ", "Requesting tool approval...", toolStyle, m.width, false)
	m.content.WriteString(formatted)
	m.content.WriteString("\n")
	m.viewport.SetContent(m.content.String())
	m.viewport.GotoBottom()

	// Handle tool approval request by showing overlay
	if event.Preview != nil {
		preview, ok := event.Preview.(*tools.ToolPreview)
		if ok {
			// Create response callback that will be called by the overlay
			responseFunc := func(response *pkgtypes.ApprovalResponse) {
				// Send approval response to agent
				m.channels.Approval <- response

				// Close overlay and update viewport
				m.overlay.deactivate()
				m.viewport.SetContent(m.content.String())
				m.viewport.GotoBottom()
			}

			// Create and activate diff viewer overlay
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
	// Approval granted - show confirmation
	formatted := formatEntry("  ✓ ", "Tool approved - executing...", toolStyle, m.width, false)
	m.content.WriteString(formatted)
	m.content.WriteString("\n")
}

func (m *model) handleToolApprovalRejected() {
	// Approval rejected - log it
	formatted := formatEntry("  ✗ ", "Tool rejected by user", errorStyle, m.width, false)
	m.content.WriteString(formatted)
	m.content.WriteString("\n")
}

func (m *model) handleToolApprovalTimeout() {
	// Approval timeout - log it
	formatted := formatEntry("  ⏱ ", "Tool approval timed out", errorStyle, m.width, false)
	m.content.WriteString(formatted)
	m.content.WriteString("\n")
}

// API and token handlers

func (m *model) handleApiCallStart(event *pkgtypes.AgentEvent) {
	// Update context token information
	if event.ApiCallInfo != nil {
		m.currentContextTokens = event.ApiCallInfo.ContextTokens
		m.maxContextTokens = event.ApiCallInfo.MaxContextTokens
	}
}

func (m *model) handleTokenUsage(event *pkgtypes.AgentEvent) {
	// Update token usage counts
	if event.TokenUsage != nil {
		m.totalPromptTokens += event.TokenUsage.PromptTokens
		m.totalCompletionTokens += event.TokenUsage.CompletionTokens
		m.totalTokens += event.TokenUsage.TotalTokens
	}
}

// Command execution handlers

func (m *model) handleCommandExecutionStart(event *pkgtypes.AgentEvent) {
	// Show command execution started message
	if event.CommandExecution != nil {
		formatted := formatEntry("  🚀 ", fmt.Sprintf("Executing: %s", event.CommandExecution.Command), toolStyle, m.width, false)
		m.content.WriteString(formatted)
		m.content.WriteString("\n")
		m.viewport.SetContent(m.content.String())
		m.viewport.GotoBottom()

		// Create and activate command execution overlay
		overlay := overlay.NewCommandExecutionOverlay(
			event.CommandExecution.Command,
			event.CommandExecution.WorkingDir,
			event.CommandExecution.ExecutionID,
			m.channels.Cancel,
		)
		m.overlay.pushOverlay(types.OverlayModeCommandOutput, overlay)
	}
}

func (m *model) handleCommandExecutionOutput(event *pkgtypes.AgentEvent) {
	// Stream command output as it arrives
	// Write output directly without styling to preserve formatting/indentation
	if event.CommandExecution != nil && event.CommandExecution.Output != "" {
		m.content.WriteString(event.CommandExecution.Output)
	}
}

func (m *model) handleCommandExecutionComplete(event *pkgtypes.AgentEvent) {
	// Show command completion status
	if event.CommandExecution != nil {
		if event.CommandExecution.ExitCode == 0 {
			formatted := formatEntry("  ✓ ", "Command completed successfully", toolStyle, m.width, false)
			m.content.WriteString(formatted)
		} else {
			formatted := formatEntry("  ✗ ", fmt.Sprintf("Command failed with exit code %d", event.CommandExecution.ExitCode), errorStyle, m.width, false)
			m.content.WriteString(formatted)
		}
		m.content.WriteString("\n")
	}
}

// Context summarization handlers

func (m *model) handleContextSummarizationStart(event *pkgtypes.AgentEvent) {
	m.summarization.active = true
	m.summarization.startTime = time.Now()
	if event.ContextSummarization != nil {
		m.summarization.strategy = event.ContextSummarization.Strategy
		m.summarization.currentTokens = event.ContextSummarization.CurrentTokens
		m.summarization.maxTokens = event.ContextSummarization.MaxTokens
		m.summarization.totalItems = event.ContextSummarization.TotalItems
	}
}

func (m *model) handleContextSummarizationProgress(event *pkgtypes.AgentEvent) {
	if event.ContextSummarization != nil {
		m.summarization.itemsProcessed = event.ContextSummarization.ItemsProcessed
		// Calculate progress percentage from items processed
		if event.ContextSummarization.TotalItems > 0 {
			m.summarization.progressPercent = float64(event.ContextSummarization.ItemsProcessed) / float64(event.ContextSummarization.TotalItems) * 100
		}
	}
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
			"🧠",
			false,
		)

		// Update current context tokens
		m.currentContextTokens = newTokens
	}
}

// Notes data handler
func (m *model) handleNotesData(event *pkgtypes.AgentEvent) {
	if event.NotesData == nil {
		return
	}

	// End loading state
	m.agentBusy = false
	m.currentLoadingMessage = ""

	// Notes data should only come from explicit /notes command request
	if !m.pendingNotesRequest {
		return
	}
	m.pendingNotesRequest = false

	// Create and activate notes overlay
	notesOverlay := overlay.NewNotesOverlay(event.NotesData.Notes, m.width, m.height)
	m.overlay.pushOverlay(types.OverlayModeNotes, notesOverlay)
}
