package headless

import (
	"fmt"
	"log"

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
	log.Printf("[FileTracker] TrackToolCall called - Type: %s, ToolName: %s", event.Type, event.ToolName)

	if event.Type != types.EventTypeToolCall {
		return fmt.Errorf("expected EventTypeToolCall, got %s", event.Type)
	}

	log.Printf("[FileTracker] Checking if %s is file-modifying tool", event.ToolName)

	// Check if this is a file-modifying tool
	if !isFileModifyingTool(event.ToolName) {
		log.Printf("[FileTracker] Tool %s does not modify files, skipping", event.ToolName)
		return nil
	}

	log.Printf("[FileTracker] ToolInput type: %T, value: %+v", event.ToolInput, event.ToolInput)

	// Extract file path from tool input
	path, err := extractFilePath(event.ToolInput)
	if err != nil {
		log.Printf("[FileTracker] Failed to extract file path from %s: %v", event.ToolName, err)
		log.Printf("[FileTracker] Tool input was: %+v", event.ToolInput)
		return err
	}

	log.Printf("[FileTracker] Successfully extracted path: %s", path)

	if path == "" {
		log.Printf("[FileTracker] WARNING: Empty path extracted from tool %s", event.ToolName)
		if t.verbose {
			log.Printf("[FileTracker] Tool input was: %+v", event.ToolInput)
		}
		return nil
	}

	// Generate unique ID for this tool call
	t.nextID++
	callID := fmt.Sprintf("%s_%d", event.ToolName, t.nextID)

	// Store pending modification with unique ID
	t.pending[callID] = path
	log.Printf("[FileTracker] Pending file modification: %s via %s (ID: %s)", path, event.ToolName, callID)
	log.Printf("[FileTracker] Total pending modifications: %d", len(t.pending))

	return nil
}

// ConfirmModification processes a successful tool result and confirms the file modification.
func (t *FileModificationTracker) ConfirmModification(event *types.AgentEvent) {
	log.Printf("[FileTracker] ConfirmModification called - Type: %s, ToolName: %s", event.Type, event.ToolName)

	if event.Type != types.EventTypeToolResult {
		log.Printf("[FileTracker] Wrong event type, expected EventTypeToolResult, got %s", event.Type)
		return
	}

	log.Printf("[FileTracker] Looking for pending modification for tool: %s", event.ToolName)
	log.Printf("[FileTracker] Current pending modifications: %+v", t.pending)

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
		log.Printf("[FileTracker] No pending modification found for tool: %s", event.ToolName)
		return
	}

	path := t.pending[confirmedID]
	log.Printf("[FileTracker] Found pending modification - ID: %s, Path: %s", confirmedID, path)

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
	log.Printf("[FileTracker] File modified: %s (+%d/-%d lines, total modified: %d)",
		path, linesAdded, linesRemoved, len(t.modified))

	// Clean up pending modification
	delete(t.pending, confirmedID)
	log.Printf("[FileTracker] Remaining pending modifications: %d", len(t.pending))
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
		log.Printf("[FileTracker] File modification failed for tool: %s (ID: %s)", event.ToolName, canceledID)
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
