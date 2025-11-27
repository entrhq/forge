package approval

import (
	"context"
	"testing"
	"time"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/types"
)

func TestManager_WaitForResponse_ApprovalGranted(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	ctx := context.Background()
	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	responseChannel := make(chan *types.ApprovalResponse, 1)
	approvalID := "test-approval-123"

	// Send granted response
	go func() {
		time.Sleep(50 * time.Millisecond)
		response := types.NewApprovalResponse(approvalID, types.ApprovalGranted)
		responseChannel <- response
	}()

	approved, timedOut := manager.waitForResponse(ctx, approvalID, toolCall, responseChannel)

	if !approved {
		t.Error("expected approval to be granted")
	}

	if timedOut {
		t.Error("expected no timeout")
	}

	// Verify ToolApprovalGranted event was emitted
	events := emitter.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.Type != types.EventTypeToolApprovalGranted {
		t.Errorf("expected ToolApprovalGranted event, got %v", event.Type)
	}

	if event.ApprovalID != approvalID {
		t.Errorf("event approvalID = %v, want %v", event.ApprovalID, approvalID)
	}

	if event.ToolName != "test_tool" {
		t.Errorf("event toolName = %v, want test_tool", event.ToolName)
	}
}

func TestManager_WaitForResponse_ApprovalRejected(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	ctx := context.Background()
	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	responseChannel := make(chan *types.ApprovalResponse, 1)
	approvalID := "test-approval-456"

	// Send rejected response
	go func() {
		time.Sleep(50 * time.Millisecond)
		response := types.NewApprovalResponse(approvalID, types.ApprovalRejected)
		responseChannel <- response
	}()

	approved, timedOut := manager.waitForResponse(ctx, approvalID, toolCall, responseChannel)

	if approved {
		t.Error("expected approval to be rejected")
	}

	if timedOut {
		t.Error("expected no timeout")
	}

	// Verify ToolApprovalRejected event was emitted
	events := emitter.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.Type != types.EventTypeToolApprovalRejected {
		t.Errorf("expected ToolApprovalRejected event, got %v", event.Type)
	}

	if event.ApprovalID != approvalID {
		t.Errorf("event approvalID = %v, want %v", event.ApprovalID, approvalID)
	}

	if event.ToolName != "test_tool" {
		t.Errorf("event toolName = %v, want test_tool", event.ToolName)
	}
}

func TestManager_WaitForResponse_Timeout(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(100*time.Millisecond, emitter.emit)

	ctx := context.Background()
	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	responseChannel := make(chan *types.ApprovalResponse, 1)
	approvalID := "test-approval-789"

	// Don't send any response - let it timeout
	approved, timedOut := manager.waitForResponse(ctx, approvalID, toolCall, responseChannel)

	if approved {
		t.Error("expected approval to be rejected on timeout")
	}

	if !timedOut {
		t.Error("expected timeout")
	}

	// Verify ToolApprovalTimeout event was emitted
	events := emitter.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.Type != types.EventTypeToolApprovalTimeout {
		t.Errorf("expected ToolApprovalTimeout event, got %v", event.Type)
	}

	if event.ApprovalID != approvalID {
		t.Errorf("event approvalID = %v, want %v", event.ApprovalID, approvalID)
	}

	if event.ToolName != "test_tool" {
		t.Errorf("event toolName = %v, want test_tool", event.ToolName)
	}
}

func TestManager_WaitForResponse_ContextCancellation(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	ctx, cancel := context.WithCancel(context.Background())
	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	responseChannel := make(chan *types.ApprovalResponse, 1)
	approvalID := "test-approval-ctx"

	// Cancel context after short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	approved, timedOut := manager.waitForResponse(ctx, approvalID, toolCall, responseChannel)

	if approved {
		t.Error("expected approval to be rejected on context cancellation")
	}

	if timedOut {
		t.Error("expected no timeout on context cancellation")
	}

	// Verify no events were emitted (context cancellation is silent)
	events := emitter.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events on context cancellation, got %d", len(events))
	}
}

func TestManager_WaitForResponse_ImmediateResponse(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	ctx := context.Background()
	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	responseChannel := make(chan *types.ApprovalResponse, 1)
	approvalID := "test-approval-immediate"

	// Send response immediately (already in buffer)
	response := types.NewApprovalResponse(approvalID, types.ApprovalGranted)
	responseChannel <- response

	approved, timedOut := manager.waitForResponse(ctx, approvalID, toolCall, responseChannel)

	if !approved {
		t.Error("expected immediate approval to be granted")
	}

	if timedOut {
		t.Error("expected no timeout")
	}
}

func TestManager_WaitForResponse_ClosedChannel(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	ctx := context.Background()
	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	responseChannel := make(chan *types.ApprovalResponse, 1)
	approvalID := "test-approval-closed"

	// Close channel immediately
	close(responseChannel)

	approved, timedOut := manager.waitForResponse(ctx, approvalID, toolCall, responseChannel)

	if approved {
		t.Error("expected closed channel to result in rejected approval")
	}

	if timedOut {
		t.Error("expected no timeout on closed channel")
	}

	// Verify ToolApprovalRejected event was emitted (nil response is treated as rejected)
	events := emitter.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.Type != types.EventTypeToolApprovalRejected {
		t.Errorf("expected ToolApprovalRejected event for closed channel, got %v", event.Type)
	}
}

func TestManager_WaitForResponse_MultipleResponses(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(5*time.Second, emitter.emit)

	ctx := context.Background()
	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	responseChannel := make(chan *types.ApprovalResponse, 2)
	approvalID := "test-approval-multiple"

	// Send multiple responses (only first should be processed)
	go func() {
		time.Sleep(50 * time.Millisecond)
		responseChannel <- types.NewApprovalResponse(approvalID, types.ApprovalGranted)
		time.Sleep(10 * time.Millisecond)
		responseChannel <- types.NewApprovalResponse(approvalID, types.ApprovalRejected)
	}()

	approved, timedOut := manager.waitForResponse(ctx, approvalID, toolCall, responseChannel)

	if !approved {
		t.Error("expected first response (granted) to be processed")
	}

	if timedOut {
		t.Error("expected no timeout")
	}

	// Verify only one event was emitted (first response)
	events := emitter.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != types.EventTypeToolApprovalGranted {
		t.Errorf("expected first response to be granted, got %v", events[0].Type)
	}
}

func TestManager_WaitForResponse_VeryShortTimeout(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(1*time.Millisecond, emitter.emit) // Very short timeout

	ctx := context.Background()
	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	responseChannel := make(chan *types.ApprovalResponse, 1)
	approvalID := "test-approval-short"

	// Response comes too late
	go func() {
		time.Sleep(100 * time.Millisecond)
		responseChannel <- types.NewApprovalResponse(approvalID, types.ApprovalGranted)
	}()

	approved, timedOut := manager.waitForResponse(ctx, approvalID, toolCall, responseChannel)

	if approved {
		t.Error("expected timeout before response")
	}

	if !timedOut {
		t.Error("expected timeout with very short timeout duration")
	}

	// Verify timeout event was emitted
	events := emitter.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != types.EventTypeToolApprovalTimeout {
		t.Errorf("expected ToolApprovalTimeout event, got %v", events[0].Type)
	}
}
