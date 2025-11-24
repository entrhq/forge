package headless

import (
	"testing"
)

func TestConstraintManager_ReadOnlyMode(t *testing.T) {
	config := ConstraintConfig{
		AllowedTools: []string{}, // Empty means all tools allowed by default
	}

	// Create constraint manager in read-only mode
	cm, err := NewConstraintManager(config, ModeReadOnly)
	if err != nil {
		t.Fatalf("Failed to create constraint manager: %v", err)
	}

	tests := []struct {
		name      string
		toolName  string
		wantError bool
		errType   ViolationType
	}{
		{
			name:      "read_file allowed in read-only mode",
			toolName:  "read_file",
			wantError: false,
		},
		{
			name:      "list_files allowed in read-only mode",
			toolName:  "list_files",
			wantError: false,
		},
		{
			name:      "search_files allowed in read-only mode",
			toolName:  "search_files",
			wantError: false,
		},
		{
			name:      "write_file blocked in read-only mode",
			toolName:  "write_file",
			wantError: true,
			errType:   ViolationReadOnlyMode,
		},
		{
			name:      "apply_diff blocked in read-only mode",
			toolName:  "apply_diff",
			wantError: true,
			errType:   ViolationReadOnlyMode,
		},
		{
			name:      "execute_command blocked in read-only mode",
			toolName:  "execute_command",
			wantError: true,
			errType:   ViolationReadOnlyMode,
		},
		{
			name:      "task_completion allowed (loop-breaking tool)",
			toolName:  "task_completion",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.ValidateToolCall(tt.toolName, nil)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error for tool %s in read-only mode, got nil", tt.toolName)
					return
				}
				violation, ok := err.(*ConstraintViolation)
				if !ok {
					t.Errorf("Expected ConstraintViolation error, got %T", err)
					return
				}
				if violation.Type != tt.errType {
					t.Errorf("Expected violation type %s, got %s", tt.errType, violation.Type)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for tool %s: %v", tt.toolName, err)
				}
			}
		})
	}
}

func TestConstraintManager_WriteMode(t *testing.T) {
	config := ConstraintConfig{
		AllowedTools: []string{}, // Empty means all tools allowed
	}

	// Create constraint manager in write mode
	cm, err := NewConstraintManager(config, ModeWrite)
	if err != nil {
		t.Fatalf("Failed to create constraint manager: %v", err)
	}

	// In write mode, all tools should be allowed (when allowed_tools is empty)
	tools := []string{
		"read_file",
		"write_file",
		"apply_diff",
		"execute_command",
		"list_files",
		"search_files",
	}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			err := cm.ValidateToolCall(tool, nil)
			if err != nil {
				t.Errorf("Tool %s should be allowed in write mode, got error: %v", tool, err)
			}
		})
	}
}

func TestConstraintManager_AllowedToolsRestriction(t *testing.T) {
	// Create a constraint manager with a restricted allowed_tools list
	config := ConstraintConfig{
		AllowedTools: []string{"read_file", "write_file"},
	}

	cm, err := NewConstraintManager(config, "write")
	if err != nil {
		t.Fatalf("Failed to create constraint manager: %v", err)
	}

	tests := []struct {
		name      string
		toolName  string
		wantError bool
	}{
		{
			name:      "allowed tool: read_file",
			toolName:  "read_file",
			wantError: false,
		},
		{
			name:      "allowed tool: write_file",
			toolName:  "write_file",
			wantError: false,
		},
		{
			name:      "disallowed tool: execute_command",
			toolName:  "execute_command",
			wantError: true,
		},
		{
			name:      "disallowed tool: list_files",
			toolName:  "list_files",
			wantError: true,
		},
		{
			name:      "loop-breaking tool always allowed: task_completion",
			toolName:  "task_completion",
			wantError: false,
		},
		{
			name:      "loop-breaking tool always allowed: ask_question",
			toolName:  "ask_question",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.ValidateToolCall(tt.toolName, nil)
			if tt.wantError && err == nil {
				t.Errorf("Expected error for disallowed tool %s, got nil", tt.toolName)
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error for allowed tool %s: %v", tt.toolName, err)
			}
		})
	}
}

func TestConstraintManager_EmptyAllowedTools(t *testing.T) {
	// When allowed_tools is empty, all tools should be allowed
	config := ConstraintConfig{
		AllowedTools: []string{},
	}

	cm, err := NewConstraintManager(config, "write")
	if err != nil {
		t.Fatalf("Failed to create constraint manager: %v", err)
	}

	tools := []string{
		"read_file",
		"write_file",
		"execute_command",
		"list_files",
		"search_files",
		"apply_diff",
	}

	for _, tool := range tools {
		err := cm.ValidateToolCall(tool, nil)
		if err != nil {
			t.Errorf("Tool %s should be allowed when allowed_tools is empty, got error: %v", tool, err)
		}
	}
}

func TestConstraintManager_ReadOnlyModeWithAllowedTools(t *testing.T) {
	// Test that read-only mode enforcement takes precedence over allowed_tools list
	config := ConstraintConfig{
		AllowedTools: []string{"read_file", "write_file"}, // write_file is in allowed list
	}

	cm, err := NewConstraintManager(config, ModeReadOnly)
	if err != nil {
		t.Fatalf("Failed to create constraint manager: %v", err)
	}

	// write_file should still be blocked because of read-only mode
	err = cm.ValidateToolCall("write_file", nil)
	if err == nil {
		t.Error("write_file should be blocked in read-only mode even if in allowed_tools list")
	}

	violation, ok := err.(*ConstraintViolation)
	if !ok {
		t.Errorf("Expected ConstraintViolation error, got %T", err)
		return
	}
	if violation.Type != ViolationReadOnlyMode {
		t.Errorf("Expected ViolationReadOnlyMode, got %s", violation.Type)
	}

	// read_file should still work
	err = cm.ValidateToolCall("read_file", nil)
	if err != nil {
		t.Errorf("read_file should be allowed: %v", err)
	}
}
