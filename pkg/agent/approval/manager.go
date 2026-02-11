package approval

import (
	"context"
	"sync"
	"time"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/types"
	"github.com/google/uuid"
)

// EventEmitter is a function type for emitting events
type EventEmitter func(event *types.AgentEvent)

// Manager handles tool approval requests and responses
type Manager struct {
	timeout          time.Duration
	pendingApprovals map[string]*pendingApproval
	mu               sync.Mutex
	emitEvent        EventEmitter
}

// pendingApproval tracks an approval request that is waiting for user response
type pendingApproval struct {
	approvalID string
	toolName   string
	toolCall   tools.ToolCall
	response   chan *types.ApprovalResponse
	closeOnce  sync.Once // Ensures channel is closed exactly once
}

// NewManager creates a new approval manager
func NewManager(timeout time.Duration, emitEvent EventEmitter) *Manager {
	return &Manager{
		timeout:          timeout,
		pendingApprovals: make(map[string]*pendingApproval),
		emitEvent:        emitEvent,
	}
}

// RequestApproval sends an approval request and waits for user response
// Returns (approved, timedOut) where:
//   - approved: true if user approved, false if rejected
//   - timedOut: true if the request timed out waiting for response
func (m *Manager) RequestApproval(ctx context.Context, toolCall tools.ToolCall, preview *tools.ToolPreview) (bool, bool) {
	// Generate unique approval ID
	approvalID := uuid.New().String()

	// Create response channel for this approval
	responseChannel := make(chan *types.ApprovalResponse, 1)

	// Store pending approval
	m.setupPendingApproval(approvalID, toolCall, responseChannel)

	// Clean up pending approval when done
	defer m.cleanupPendingApproval(approvalID, responseChannel)

	// Parse tool input for event
	argsMap := parseToolArguments(toolCall)

	// Check for auto-approval
	if approved, autoApproved := m.checkAutoApproval(approvalID, toolCall, argsMap); autoApproved {
		return approved, false
	}

	// Emit approval request event (tool requires manual approval)
	m.emitEvent(types.NewToolApprovalRequestEvent(approvalID, toolCall.ToolName, argsMap, preview))

	// Wait for response with timeout
	return m.waitForResponse(ctx, approvalID, toolCall, responseChannel)
}

// HandleResponse processes an approval response from the user
func (m *Manager) HandleResponse(response *types.ApprovalResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we have a pending approval matching this response
	pa, ok := m.pendingApprovals[response.ApprovalID]
	if !ok {
		// No matching pending approval - ignore this response
		return
	}

	// Send the response to the waiting goroutine
	// Use non-blocking send to prevent deadlock if channel is full or being cleaned up
	select {
	case pa.response <- response:
		// Response delivered successfully
	default:
		// Channel full, closed, or no receiver - this is safe to ignore
		// The cleanup process may have already started
	}
}

// setupPendingApproval stores the pending approval request
func (m *Manager) setupPendingApproval(approvalID string, toolCall tools.ToolCall, responseChannel chan *types.ApprovalResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pendingApprovals[approvalID] = &pendingApproval{
		approvalID: approvalID,
		toolName:   toolCall.ToolName,
		toolCall:   toolCall,
		response:   responseChannel,
	}
}

// cleanupPendingApproval cleans up the pending approval
// This method is safe to call multiple times due to sync.Once
func (m *Manager) cleanupPendingApproval(approvalID string, responseChannel chan *types.ApprovalResponse) {
	m.mu.Lock()
	pa, ok := m.pendingApprovals[approvalID]
	if ok {
		delete(m.pendingApprovals, approvalID)
	}
	m.mu.Unlock()

	// Close the channel exactly once using sync.Once
	// This prevents race conditions between cleanup and HandleResponse
	if ok && pa != nil {
		pa.closeOnce.Do(func() {
			close(responseChannel)
		})
	}
}

// parseToolArguments safely parses tool call arguments into a map
func parseToolArguments(toolCall tools.ToolCall) map[string]interface{} {
	argsMap, err := tools.XMLToMap(toolCall.GetArgumentsXML())
	if err != nil {
		return make(map[string]interface{})
	}
	return argsMap
}
