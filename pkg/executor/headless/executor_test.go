package headless

import (
	"context"
	"testing"
	"time"

	"github.com/entrhq/forge/pkg/agent"
	"github.com/entrhq/forge/pkg/types"
)

// mockAgent implements a minimal agent.Agent for testing
type mockAgent struct {
	channels *types.AgentChannels
}

func (m *mockAgent) Start(ctx context.Context) error {
	// Simulate agent event loop that closes Done when Input is closed
	go func() {
		<-m.channels.Input // Wait for Input channel to close
		close(m.channels.Done)
	}()
	return nil
}

func (m *mockAgent) Shutdown(ctx context.Context) error {
	return nil
}

func (m *mockAgent) GetChannels() *types.AgentChannels {
	return m.channels
}

func (m *mockAgent) GetTool(name string) interface{} {
	return nil
}

func (m *mockAgent) GetTools() []interface{} {
	return nil
}

func (m *mockAgent) GetContextInfo() *agent.ContextInfo {
	return &agent.ContextInfo{}
}

func TestFileModificationTracking(t *testing.T) {
	// Create mock channels
	channels := &types.AgentChannels{
		Event:    make(chan *types.AgentEvent, 10),
		Input:    make(chan *types.Input, 1),
		Shutdown: make(chan struct{}),
		Done:     make(chan struct{}),
	}

	mockAg := &mockAgent{channels: channels}

	config := &Config{
		Task:         "test task",
		Mode:         ModeWrite,
		WorkspaceDir: t.TempDir(),
		Constraints: ConstraintConfig{
			MaxFiles:        10,
			MaxLinesChanged: 1000,
			Timeout:         30 * time.Second,
		},
		Artifacts: ArtifactConfig{
			Enabled:   false,
			OutputDir: "",
		},
		Git: GitConfig{
			AutoCommit: false,
		},
	}

	executor, err := NewExecutor(mockAg, config)
	if err != nil {
		t.Fatalf("Failed to create executor: %v", err)
	}

	ctx := context.Background()

	// Start executor in goroutine
	done := make(chan error, 1)
	go func() {
		done <- executor.Run(ctx)
	}()

	// Simulate file modification events
	// 1. Tool call to write_file
	channels.Event <- types.NewToolCallEvent("write_file", map[string]interface{}{
		"path":    "test.go",
		"content": "package main",
	})

	// 2. Successful tool result
	channels.Event <- types.NewToolResultEvent("write_file", "File written successfully")

	// 3. Tool call to apply_diff
	channels.Event <- types.NewToolCallEvent("apply_diff", map[string]interface{}{
		"path": "test2.go",
		"edits": []interface{}{
			map[string]interface{}{
				"search":  "old",
				"replace": "new",
			},
		},
	})

	// 4. Successful tool result
	channels.Event <- types.NewToolResultEvent("apply_diff", "Diff applied successfully")

	// 5. Tool call that fails
	channels.Event <- types.NewToolCallEvent("write_file", map[string]interface{}{
		"path":    "test3.go",
		"content": "package main",
	})

	// 6. Failed tool result
	channels.Event <- types.NewToolResultErrorEvent("write_file", nil)

	// 7. Send turn end to complete execution
	time.Sleep(100 * time.Millisecond) // Give time for events to process
	channels.Event <- types.NewTurnEndEvent()

	// Close Event channel after turn end to signal no more events
	close(channels.Event)

	// Wait for executor to complete
	if err := <-done; err != nil {
		t.Fatalf("Executor failed: %v", err)
	}

	// Verify file modifications were tracked
	if len(executor.summary.FilesModified) != 2 {
		t.Errorf("Expected 2 files modified, got %d", len(executor.summary.FilesModified))
	}

	// Verify the correct files were tracked
	expectedFiles := map[string]bool{
		"test.go":  false,
		"test2.go": false,
	}

	for _, fm := range executor.summary.FilesModified {
		if _, exists := expectedFiles[fm.Path]; exists {
			expectedFiles[fm.Path] = true
		} else {
			t.Errorf("Unexpected file modification tracked: %s", fm.Path)
		}
	}

	for file, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file modification not tracked: %s", file)
		}
	}

	// Verify metrics
	if executor.summary.Metrics.FilesModified != 2 {
		t.Errorf("Expected metrics to show 2 files modified, got %d", executor.summary.Metrics.FilesModified)
	}
}
