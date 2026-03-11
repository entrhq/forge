package agent

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/entrhq/forge/pkg/agent/approval"
	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/types"
)

// TestApprovalSystem_ConcurrentCleanupRace tests the race condition between
// waitForResponse and cleanupPendingApproval to ensure proper synchronization
func TestApprovalSystem_ConcurrentCleanupRace(t *testing.T) {
	ctx := context.Background()
	channels := types.NewAgentChannels(1000) // Large buffer to prevent blocking

	// Drain events in background to prevent channel from filling up
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-channels.Event:
				// Drain events
			case <-done:
				return
			}
		}
	}()
	defer close(done)

	emitEvent := func(event *types.AgentEvent) {
		select {
		case channels.Event <- event:
		default:
			// Non-blocking send, drop if full
		}
	}

	// Very short timeout to trigger cleanup quickly
	agent := &DefaultAgent{
		channels:        channels,
		approvalManager: approval.NewManager(10*time.Millisecond, emitEvent),
	}

	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "test_tool",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<arg>value</arg>`)},
	}

	// Run many iterations to increase chance of hitting the race
	// This test should pass with the sync.Once fix and fail without it
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// This should not panic even with concurrent cleanup
			agent.requestApproval(ctx, toolCall, nil)
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// If we get here without panicking, the race condition is handled correctly
}

func TestApprovalSystem_RequestApproval(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name             string
		sendResponse     bool
		responseDecision types.ApprovalDecision
		timeout          time.Duration
		expectApproved   bool
		expectTimedOut   bool
	}{
		{
			name:             "approval granted",
			sendResponse:     true,
			responseDecision: types.ApprovalGranted,
			timeout:          1 * time.Second,
			expectApproved:   true,
			expectTimedOut:   false,
		},
		{
			name:             "approval rejected",
			sendResponse:     true,
			responseDecision: types.ApprovalRejected,
			timeout:          1 * time.Second,
			expectApproved:   false,
			expectTimedOut:   false,
		},
		{
			name:           "approval timeout",
			sendResponse:   false,
			timeout:        100 * time.Millisecond,
			expectApproved: false,
			expectTimedOut: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channels := types.NewAgentChannels(10)

			// Track events for verification - use mutex to prevent data race
			var lastApprovalID string
			var approvalIDMutex sync.Mutex

			emitEvent := func(event *types.AgentEvent) {
				if event.Type == types.EventTypeToolApprovalRequest {
					approvalIDMutex.Lock()
					lastApprovalID = event.ApprovalID
					approvalIDMutex.Unlock()
				}
				channels.Event <- event
			}

			agent := &DefaultAgent{
				channels:        channels,
				approvalManager: approval.NewManager(tt.timeout, emitEvent),
			}

			toolCall := tools.ToolCall{
				ServerName: "local",
				ToolName:   "test_tool",
				Arguments: tools.ArgumentsBlock{
					InnerXML: []byte(`<arg>value</arg>`),
				},
			}

			preview := &tools.ToolPreview{
				Type:    tools.PreviewTypeDiff,
				Title:   "Test preview",
				Content: "preview content",
			}

			if tt.sendResponse {
				go func() {
					// Wait for approval request event
					time.Sleep(50 * time.Millisecond)

					approvalIDMutex.Lock()
					id := lastApprovalID
					approvalIDMutex.Unlock()

					if id != "" {
						response := types.NewApprovalResponse(id, tt.responseDecision)
						agent.handleApprovalResponse(response)
					}
				}()
			}

			approved, timedOut := agent.requestApproval(ctx, toolCall, preview)

			if approved != tt.expectApproved {
				t.Errorf("approved = %v, want %v", approved, tt.expectApproved)
			}

			if timedOut != tt.expectTimedOut {
				t.Errorf("timedOut = %v, want %v", timedOut, tt.expectTimedOut)
			}
		})
	}
}
