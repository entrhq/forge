package headless

import (
	"testing"

	"github.com/entrhq/forge/pkg/types"
)

func TestFileModificationTracker_TrackToolCall(t *testing.T) {
	tests := []struct {
		name          string
		event         *types.AgentEvent
		expectPending bool
		expectError   bool
	}{
		{
			name: "write_file with valid path",
			event: &types.AgentEvent{
				Type:     types.EventTypeToolCall,
				ToolName: "write_file",
				ToolInput: map[string]interface{}{
					"path":    "test.txt",
					"content": "hello world",
				},
			},
			expectPending: true,
			expectError:   false,
		},
		{
			name: "apply_diff with valid path",
			event: &types.AgentEvent{
				Type:     types.EventTypeToolCall,
				ToolName: "apply_diff",
				ToolInput: map[string]interface{}{
					"path": "src/main.go",
					"edits": []interface{}{
						map[string]interface{}{
							"search":  "old",
							"replace": "new",
						},
					},
				},
			},
			expectPending: true,
			expectError:   false,
		},
		{
			name: "read_file does not modify files",
			event: &types.AgentEvent{
				Type:     types.EventTypeToolCall,
				ToolName: "read_file",
				ToolInput: map[string]interface{}{
					"path": "test.txt",
				},
			},
			expectPending: false,
			expectError:   false,
		},
		{
			name: "write_file with missing path",
			event: &types.AgentEvent{
				Type:     types.EventTypeToolCall,
				ToolName: "write_file",
				ToolInput: map[string]interface{}{
					"content": "hello world",
				},
			},
			expectPending: false,
			expectError:   true,
		},
		{
			name: "write_file with nil input",
			event: &types.AgentEvent{
				Type:      types.EventTypeToolCall,
				ToolName:  "write_file",
				ToolInput: nil,
			},
			expectPending: false,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewFileModificationTracker(false)

			err := tracker.TrackToolCall(tt.event)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			pendingCount := tracker.GetPendingCount()
			if tt.expectPending && pendingCount != 1 {
				t.Errorf("expected 1 pending modification, got %d", pendingCount)
			}
			if !tt.expectPending && pendingCount != 0 {
				t.Errorf("expected 0 pending modifications, got %d", pendingCount)
			}
		})
	}
}

func TestFileModificationTracker_ConfirmModification(t *testing.T) {
	tracker := NewFileModificationTracker(false)

	// Track a tool call
	event := &types.AgentEvent{
		Type:     types.EventTypeToolCall,
		ToolName: "write_file",
		ToolInput: map[string]interface{}{
			"path":    "test.txt",
			"content": "hello",
		},
	}
	err := tracker.TrackToolCall(event)
	if err != nil {
		t.Fatalf("failed to track tool call: %v", err)
	}

	// Confirm the modification
	result := &types.AgentEvent{
		Type:     types.EventTypeToolResult,
		ToolName: "write_file",
	}
	tracker.ConfirmModification(result)

	// Check that the modification was confirmed
	modified := tracker.GetModifiedFiles()
	if len(modified) != 1 {
		t.Errorf("expected 1 modified file, got %d", len(modified))
	}
	if len(modified) > 0 && modified[0].Path != "test.txt" {
		t.Errorf("expected path 'test.txt', got '%s'", modified[0].Path)
	}

	// Check that pending was cleared
	if tracker.GetPendingCount() != 0 {
		t.Error("expected pending to be cleared after confirmation")
	}
}

func TestFileModificationTracker_CancelModification(t *testing.T) {
	tracker := NewFileModificationTracker(false)

	// Track a tool call
	event := &types.AgentEvent{
		Type:     types.EventTypeToolCall,
		ToolName: "write_file",
		ToolInput: map[string]interface{}{
			"path":    "test.txt",
			"content": "hello",
		},
	}
	err := tracker.TrackToolCall(event)
	if err != nil {
		t.Fatalf("failed to track tool call: %v", err)
	}

	// Cancel the modification (tool failed)
	errorEvent := &types.AgentEvent{
		Type:     types.EventTypeToolResultError,
		ToolName: "write_file",
	}
	tracker.CancelModification(errorEvent)

	// Check that no modification was recorded
	modified := tracker.GetModifiedFiles()
	if len(modified) != 0 {
		t.Errorf("expected 0 modified files, got %d", len(modified))
	}

	// Check that pending was cleared
	if tracker.GetPendingCount() != 0 {
		t.Error("expected pending to be cleared after cancellation")
	}
}

func TestFileModificationTracker_MultipleModifications(t *testing.T) {
	tracker := NewFileModificationTracker(false)

	// Track multiple tool calls
	files := []string{"file1.txt", "file2.go", "file3.md"}
	for _, file := range files {
		event := &types.AgentEvent{
			Type:     types.EventTypeToolCall,
			ToolName: "write_file",
			ToolInput: map[string]interface{}{
				"path":    file,
				"content": "test",
			},
		}
		if err := tracker.TrackToolCall(event); err != nil {
			t.Fatalf("failed to track tool call for %s: %v", file, err)
		}
	}

	// Confirm all modifications
	for range files {
		result := &types.AgentEvent{
			Type:     types.EventTypeToolResult,
			ToolName: "write_file",
		}
		tracker.ConfirmModification(result)
	}

	// Check that all modifications were recorded
	modified := tracker.GetModifiedFiles()
	if len(modified) != len(files) {
		t.Errorf("expected %d modified files, got %d", len(files), len(modified))
	}

	// Check that pending is empty
	if tracker.GetPendingCount() != 0 {
		t.Error("expected pending to be empty after all confirmations")
	}
}

func TestFileModificationTracker_WrongEventType(t *testing.T) {
	tracker := NewFileModificationTracker(false)

	// Try to track a non-tool-call event
	event := &types.AgentEvent{
		Type:     types.EventTypeToolResult,
		ToolName: "write_file",
	}

	err := tracker.TrackToolCall(event)
	if err == nil {
		t.Error("expected error when tracking non-tool-call event")
	}

	if tracker.GetPendingCount() != 0 {
		t.Error("expected no pending modifications")
	}
}
