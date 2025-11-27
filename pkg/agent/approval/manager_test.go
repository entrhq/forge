package approval

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/types"
)

// mockEventEmitter captures emitted events for testing
type mockEventEmitter struct {
	events []*types.AgentEvent
	mu     sync.Mutex
}

func (m *mockEventEmitter) emit(event *types.AgentEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockEventEmitter) getEvents() []*types.AgentEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*types.AgentEvent{}, m.events...)
}

func TestNewManager(t *testing.T) {
	emitter := &mockEventEmitter{}
	timeout := 5 * time.Second

	manager := NewManager(timeout, emitter.emit)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.timeout != timeout {
		t.Errorf("timeout = %v, want %v", manager.timeout, timeout)
	}

	if manager.pendingApproval != nil {
		t.Error("expected no pending approval initially")
	}
}

func TestManager_SetupAndCleanupPendingApproval(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	responseChannel := make(chan *types.ApprovalResponse, 1)
	approvalID := "test-approval-123"

	// Test setup
	manager.setupPendingApproval(approvalID, toolCall, responseChannel)

	if manager.pendingApproval == nil {
		t.Fatal("expected pending approval to be set")
	}

	if manager.pendingApproval.approvalID != approvalID {
		t.Errorf("approvalID = %v, want %v", manager.pendingApproval.approvalID, approvalID)
	}

	if manager.pendingApproval.toolName != "test_tool" {
		t.Errorf("toolName = %v, want test_tool", manager.pendingApproval.toolName)
	}

	// Test cleanup
	manager.cleanupPendingApproval(responseChannel)

	if manager.pendingApproval != nil {
		t.Error("expected pending approval to be cleared")
	}

	// Verify channel is closed
	select {
	case _, ok := <-responseChannel:
		if ok {
			t.Error("expected channel to be closed")
		}
	default:
		t.Error("expected channel to be closed")
	}
}

func TestManager_CleanupPendingApproval_MultipleCallsSafe(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	responseChannel := make(chan *types.ApprovalResponse, 1)
	approvalID := "test-approval-123"

	manager.setupPendingApproval(approvalID, toolCall, responseChannel)

	// Call cleanup multiple times - should not panic
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.cleanupPendingApproval(responseChannel)
		}()
	}

	wg.Wait()

	// Should complete without panic
}

func TestManager_HandleResponse_MatchingApproval(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	responseChannel := make(chan *types.ApprovalResponse, 1)
	approvalID := "test-approval-123"

	manager.setupPendingApproval(approvalID, toolCall, responseChannel)

	// Create matching response
	response := types.NewApprovalResponse(approvalID, types.ApprovalGranted)

	// Handle response
	manager.HandleResponse(response)

	// Verify response was sent to channel
	select {
	case receivedResponse := <-responseChannel:
		if receivedResponse.ApprovalID != approvalID {
			t.Errorf("received approvalID = %v, want %v", receivedResponse.ApprovalID, approvalID)
		}
		if receivedResponse.Decision != types.ApprovalGranted {
			t.Errorf("received decision = %v, want %v", receivedResponse.Decision, types.ApprovalGranted)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected response to be sent to channel")
	}
}

func TestManager_HandleResponse_NoMatchingApproval(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	// No pending approval set up
	response := types.NewApprovalResponse("non-existent-id", types.ApprovalGranted)

	// Should not panic
	manager.HandleResponse(response)
}

func TestManager_HandleResponse_WrongApprovalID(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	responseChannel := make(chan *types.ApprovalResponse, 1)
	approvalID := "test-approval-123"

	manager.setupPendingApproval(approvalID, toolCall, responseChannel)

	// Create response with wrong ID
	response := types.NewApprovalResponse("wrong-id", types.ApprovalGranted)

	// Handle response
	manager.HandleResponse(response)

	// Verify nothing was sent to channel
	select {
	case <-responseChannel:
		t.Error("expected no response to be sent for wrong approval ID")
	case <-time.After(100 * time.Millisecond):
		// Expected - no response sent
	}
}

func TestManager_ParseToolArguments(t *testing.T) {
	tests := []struct {
		name     string
		toolCall tools.ToolCall
		wantLen  int
		wantKey  string
	}{
		{
			name: "valid arguments",
			toolCall: tools.ToolCall{
				ServerName: "local",
				ToolName:   "test_tool",
				Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<path>/test/path</path><content>test content</content>`)},
			},
			wantLen: 2,
			wantKey: "path",
		},
		{
			name: "empty arguments",
			toolCall: tools.ToolCall{
				ServerName: "local",
				ToolName:   "test_tool",
				Arguments:  tools.ArgumentsBlock{InnerXML: []byte(``)},
			},
			wantLen: 0,
		},
		{
			name: "invalid XML",
			toolCall: tools.ToolCall{
				ServerName: "local",
				ToolName:   "test_tool",
				Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<unclosed>`)},
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseToolArguments(tt.toolCall)

			if len(result) != tt.wantLen {
				t.Errorf("parseToolArguments() returned %d keys, want %d", len(result), tt.wantLen)
			}

			if tt.wantKey != "" {
				if _, ok := result[tt.wantKey]; !ok {
					t.Errorf("parseToolArguments() missing expected key %q", tt.wantKey)
				}
			}
		})
	}
}

func TestManager_RequestApproval_GrantedResponse(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	ctx := context.Background()
	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	preview := &tools.ToolPreview{
		Type:    tools.PreviewTypeDiff,
		Title:   "Test Preview",
		Content: "preview content",
	}

	// Capture approval ID from event
	var approvalID string
	var approvalIDMu sync.Mutex

	// Start goroutine to respond to approval request
	go func() {
		time.Sleep(50 * time.Millisecond)
		events := emitter.getEvents()
		for _, event := range events {
			if event.Type == types.EventTypeToolApprovalRequest {
				approvalIDMu.Lock()
				approvalID = event.ApprovalID
				approvalIDMu.Unlock()

				response := types.NewApprovalResponse(approvalID, types.ApprovalGranted)
				manager.HandleResponse(response)
				break
			}
		}
	}()

	approved, timedOut := manager.RequestApproval(ctx, toolCall, preview)

	if !approved {
		t.Error("expected approval to be granted")
	}

	if timedOut {
		t.Error("expected no timeout")
	}

	// Verify events were emitted
	events := emitter.getEvents()
	if len(events) < 2 {
		t.Fatalf("expected at least 2 events (request + granted), got %d", len(events))
	}

	// Check for approval request event
	foundRequest := false
	foundGranted := false
	for _, event := range events {
		if event.Type == types.EventTypeToolApprovalRequest {
			foundRequest = true
		}
		if event.Type == types.EventTypeToolApprovalGranted {
			foundGranted = true
		}
	}

	if !foundRequest {
		t.Error("expected ToolApprovalRequest event")
	}

	if !foundGranted {
		t.Error("expected ToolApprovalGranted event")
	}
}

func TestManager_RequestApproval_RejectedResponse(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	ctx := context.Background()
	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	// Start goroutine to respond to approval request
	go func() {
		time.Sleep(50 * time.Millisecond)
		events := emitter.getEvents()
		for _, event := range events {
			if event.Type == types.EventTypeToolApprovalRequest {
				response := types.NewApprovalResponse(event.ApprovalID, types.ApprovalRejected)
				manager.HandleResponse(response)
				break
			}
		}
	}()

	approved, timedOut := manager.RequestApproval(ctx, toolCall, nil)

	if approved {
		t.Error("expected approval to be rejected")
	}

	if timedOut {
		t.Error("expected no timeout")
	}

	// Verify rejection event was emitted
	events := emitter.getEvents()
	foundRejected := false
	for _, event := range events {
		if event.Type == types.EventTypeToolApprovalRejected {
			foundRejected = true
			break
		}
	}

	if !foundRejected {
		t.Error("expected ToolApprovalRejected event")
	}
}

func TestManager_RequestApproval_Timeout(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(100*time.Millisecond, emitter.emit)

	ctx := context.Background()
	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	// Don't send any response - let it timeout
	approved, timedOut := manager.RequestApproval(ctx, toolCall, nil)

	if approved {
		t.Error("expected approval to be rejected on timeout")
	}

	if !timedOut {
		t.Error("expected timeout")
	}

	// Verify timeout event was emitted
	events := emitter.getEvents()
	foundTimeout := false
	for _, event := range events {
		if event.Type == types.EventTypeToolApprovalTimeout {
			foundTimeout = true
			break
		}
	}

	if !foundTimeout {
		t.Error("expected ToolApprovalTimeout event")
	}
}

func TestManager_RequestApproval_ContextCancellation(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	ctx, cancel := context.WithCancel(context.Background())
	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	// Cancel context after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	approved, timedOut := manager.RequestApproval(ctx, toolCall, nil)

	if approved {
		t.Error("expected approval to be rejected on context cancellation")
	}

	if timedOut {
		t.Error("expected no timeout on context cancellation")
	}
}
