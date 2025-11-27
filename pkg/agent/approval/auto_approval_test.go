package approval

import (
	"path/filepath"
	"testing"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/types"
)

func TestManager_CheckAutoApproval_ExecuteCommand_Whitelisted(t *testing.T) {
	// Setup: Initialize config for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	if err := config.Initialize(configPath); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	// Whitelist a test command
	testCommand := "echo test"
	whitelist := config.GetCommandWhitelist()
	if whitelist == nil {
		t.Fatal("failed to get command whitelist")
	}
	if err := whitelist.AddPattern(testCommand, "Test command"); err != nil {
		t.Fatalf("failed to add pattern: %v", err)
	}

	emitter := &mockEventEmitter{}
	manager := NewManager(0, emitter.emit)

	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "execute_command",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<command>echo test</command>`)},
	}

	argsMap := map[string]interface{}{
		"command": testCommand,
	}

	approved, autoApproved := manager.checkAutoApproval("test-id", toolCall, argsMap)

	if !approved {
		t.Error("expected whitelisted command to be approved")
	}

	if !autoApproved {
		t.Error("expected whitelisted command to be auto-approved")
	}

	// Verify approval event was emitted
	events := emitter.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != types.EventTypeToolApprovalGranted {
		t.Errorf("expected ToolApprovalGranted event, got %v", events[0].Type)
	}
}

func TestManager_CheckAutoApproval_ExecuteCommand_NotWhitelisted(t *testing.T) {
	// Setup: Initialize config for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	if err := config.Initialize(configPath); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	emitter := &mockEventEmitter{}
	manager := NewManager(0, emitter.emit)

	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "execute_command",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<command>rm -rf /</command>`)},
	}

	argsMap := map[string]interface{}{
		"command": "rm -rf /",
	}

	approved, autoApproved := manager.checkAutoApproval("test-id", toolCall, argsMap)

	if approved {
		t.Error("expected non-whitelisted command to not be approved")
	}

	if autoApproved {
		t.Error("expected non-whitelisted command to not be auto-approved")
	}

	// Verify no events were emitted
	events := emitter.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestManager_CheckAutoApproval_RegularTool_AutoApproved(t *testing.T) {
	// Setup: Initialize config for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	if err := config.Initialize(configPath); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	// Auto-approve read_file tool
	autoApproval := config.GetAutoApproval()
	if autoApproval == nil {
		t.Fatal("failed to get auto approval section")
	}
	autoApproval.SetToolAutoApproval("read_file", true)

	emitter := &mockEventEmitter{}
	manager := NewManager(0, emitter.emit)

	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "read_file",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<path>test.txt</path>`)},
	}

	argsMap := map[string]interface{}{
		"path": "test.txt",
	}

	approved, autoApproved := manager.checkAutoApproval("test-id", toolCall, argsMap)

	if !approved {
		t.Error("expected auto-approved tool to be approved")
	}

	if !autoApproved {
		t.Error("expected tool to be auto-approved")
	}

	// Verify approval event was emitted
	events := emitter.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != types.EventTypeToolApprovalGranted {
		t.Errorf("expected ToolApprovalGranted event, got %v", events[0].Type)
	}
}

func TestManager_CheckAutoApproval_RegularTool_NotAutoApproved(t *testing.T) {
	// Setup: Initialize config for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	if err := config.Initialize(configPath); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	emitter := &mockEventEmitter{}
	manager := NewManager(0, emitter.emit)

	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "write_file",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(`<path>test.txt</path><content>data</content>`)},
	}

	argsMap := map[string]interface{}{
		"path":    "test.txt",
		"content": "data",
	}

	approved, autoApproved := manager.checkAutoApproval("test-id", toolCall, argsMap)

	if approved {
		t.Error("expected non-auto-approved tool to not be approved")
	}

	if autoApproved {
		t.Error("expected tool to not be auto-approved")
	}

	// Verify no events were emitted
	events := emitter.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestManager_IsCommandWhitelisted_ValidCommand(t *testing.T) {
	// Setup: Initialize config for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	if err := config.Initialize(configPath); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	// Use a command that's already in the default whitelist
	testCommand := "git status"

	emitter := &mockEventEmitter{}
	manager := NewManager(0, emitter.emit)

	argsMap := map[string]interface{}{
		"command": testCommand,
	}

	result := manager.isCommandWhitelisted("test-id", argsMap)

	if !result {
		t.Error("expected whitelisted command to return true")
	}

	// Verify approval event was emitted
	events := emitter.getEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].Type != types.EventTypeToolApprovalGranted {
		t.Errorf("expected ToolApprovalGranted event, got %v", events[0].Type)
	}
}

func TestManager_IsCommandWhitelisted_MissingCommandKey(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(0, emitter.emit)

	argsMap := map[string]interface{}{
		"other_key": "value",
	}

	result := manager.isCommandWhitelisted("test-id", argsMap)

	if result {
		t.Error("expected missing command key to return false")
	}

	// Verify no events were emitted
	events := emitter.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestManager_IsCommandWhitelisted_InvalidCommandType(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(0, emitter.emit)

	argsMap := map[string]interface{}{
		"command": 123, // Not a string
	}

	result := manager.isCommandWhitelisted("test-id", argsMap)

	if result {
		t.Error("expected invalid command type to return false")
	}

	// Verify no events were emitted
	events := emitter.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestManager_IsCommandWhitelisted_NotWhitelisted(t *testing.T) {
	// Setup: Initialize config for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	if err := config.Initialize(configPath); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	emitter := &mockEventEmitter{}
	manager := NewManager(0, emitter.emit)

	argsMap := map[string]interface{}{
		"command": "dangerous_command",
	}

	result := manager.isCommandWhitelisted("test-id", argsMap)

	if result {
		t.Error("expected non-whitelisted command to return false")
	}

	// Verify no events were emitted
	events := emitter.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestManager_CheckAutoApproval_ExecuteCommand_EmptyArgs(t *testing.T) {
	emitter := &mockEventEmitter{}
	manager := NewManager(0, emitter.emit)

	toolCall := tools.ToolCall{
		ServerName: "local",
		ToolName:   "execute_command",
		Arguments:  tools.ArgumentsBlock{InnerXML: []byte(``)},
	}

	argsMap := map[string]interface{}{}

	approved, autoApproved := manager.checkAutoApproval("test-id", toolCall, argsMap)

	if approved {
		t.Error("expected empty args to not be approved")
	}

	if autoApproved {
		t.Error("expected empty args to not be auto-approved")
	}

	// Verify no events were emitted
	events := emitter.getEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}
