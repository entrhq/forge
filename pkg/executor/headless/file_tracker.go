package headless

import (
	"fmt"

	"github.com/entrhq/forge/pkg/types"
)

// FileModificationTracker tracks file modifications during execution.
// It maintains a map of pending modifications (by tool call ID) and converts
// them to confirmed modifications when tools succeed.
type FileModificationTracker struct {
	pending  map[string]string  // toolCallID -> filePath for pending modifications
	modified []FileModification // confirmed file modifications
	verbose  bool               // enable verbose logging
	nextID   int                // counter for generating unique tool call IDs
}

// NewFileModificationTracker creates a new file modification tracker.
func NewFileModificationTracker(verbose bool) *FileModificationTracker {
	return &FileModificationTracker{
		pending:  make(map[string]string),
		modified: make([]FileModification, 0),
		verbose:  verbose,
		nextID:   0,
	}
}

// TrackToolCall processes a tool call event and records pending file modifications.
func (t *FileModificationTracker) TrackToolCall(event *types.AgentEvent) error {
	if event.Type != types.EventTypeToolCall {
		return fmt.Errorf("expected EventTypeToolCall, got %s", event.Type)
	}

	// Check if this is a file-modifying tool
	if !isFileModifyingTool(event.ToolName) {
		return nil
	}

	// Extract file path from tool input
	path, err := extractFilePath(event.ToolInput)
	if err != nil {
		return err
	}

	if path == "" {
		return nil
	}

	// Generate unique ID for this tool call
	t.nextID++
	callID := fmt.Sprintf("%s_%d", event.ToolName, t.nextID)

	// Store pending modification with unique ID
	t.pending[callID] = path

	return nil
}

// ConfirmModification processes a successful tool result and confirms the file modification.
func (t *FileModificationTracker) ConfirmModification(event *types.AgentEvent) {
	if event.Type != types.EventTypeToolResult {
		return
	}

	// Find and confirm the oldest pending modification for this tool
	// This implements FIFO behavior for multiple calls to the same tool
	var confirmedID string
	for id := range t.pending {
		// Check if this ID belongs to the current tool
		if len(id) >= len(event.ToolName) && id[:len(event.ToolName)] == event.ToolName {
			confirmedID = id
			break
		}
	}

	if confirmedID == "" {
		return
	}

	path := t.pending[confirmedID]

	// Extract line changes from event metadata if available
	linesAdded := 0
	linesRemoved := 0
	if event.Metadata != nil {
		if la, ok := event.Metadata["lines_added"].(int); ok {
			linesAdded = la
		}
		if lr, ok := event.Metadata["lines_removed"].(int); ok {
			linesRemoved = lr
		}
	}

	// Tool succeeded - record the file modification
	t.modified = append(t.modified, FileModification{
		Path:         path,
		LinesAdded:   linesAdded,
		LinesRemoved: linesRemoved,
	})

	// Clean up pending modification
	delete(t.pending, confirmedID)
}

// CancelModification processes a failed tool result and removes the pending modification.
func (t *FileModificationTracker) CancelModification(event *types.AgentEvent) {
	if event.Type != types.EventTypeToolResultError {
		return
	}

	// Find and cancel the oldest pending modification for this tool
	var canceledID string
	for id := range t.pending {
		// Check if this ID belongs to the current tool
		if len(id) >= len(event.ToolName) && id[:len(event.ToolName)] == event.ToolName {
			canceledID = id
			break
		}
	}

	if canceledID != "" {
		delete(t.pending, canceledID)
	}
}

// GetModifiedFiles returns the list of confirmed file modifications.
func (t *FileModificationTracker) GetModifiedFiles() []FileModification {
	return t.modified
}

// GetPendingCount returns the number of pending modifications (for diagnostics).
func (t *FileModificationTracker) GetPendingCount() int {
	return len(t.pending)
}
