package headless

import (
	"testing"
)

func TestIsLoopBreakingTool(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		expected bool
	}{
		{
			name:     "task_completion is loop-breaking",
			toolName: "task_completion",
			expected: true,
		},
		{
			name:     "ask_question is loop-breaking",
			toolName: "ask_question",
			expected: true,
		},
		{
			name:     "converse is loop-breaking",
			toolName: "converse",
			expected: true,
		},
		{
			name:     "write_file is not loop-breaking",
			toolName: "write_file",
			expected: false,
		},
		{
			name:     "execute_command is not loop-breaking",
			toolName: "execute_command",
			expected: false,
		},
		{
			name:     "read_file is not loop-breaking",
			toolName: "read_file",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLoopBreakingTool(tt.toolName)
			if result != tt.expected {
				t.Errorf("isLoopBreakingTool(%q) = %v, expected %v", tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestConstraintManager_ValidateToolCall_LoopBreakingTools(t *testing.T) {
	// Create a constraint manager with a restricted allowed_tools list
	config := ConstraintConfig{
		AllowedTools: []string{"read_file", "write_file"},
	}

	cm, err := NewConstraintManager(config)
	if err != nil {
		t.Fatalf("Failed to create constraint manager: %v", err)
	}

	tests := []struct {
		name        string
		toolName    string
		shouldError bool
	}{
		{
			name:        "task_completion should always be allowed",
			toolName:    "task_completion",
			shouldError: false,
		},
		{
			name:        "ask_question should always be allowed",
			toolName:    "ask_question",
			shouldError: false,
		},
		{
			name:        "converse should always be allowed",
			toolName:    "converse",
			shouldError: false,
		},
		{
			name:        "read_file is in allowed list",
			toolName:    "read_file",
			shouldError: false,
		},
		{
			name:        "write_file is in allowed list",
			toolName:    "write_file",
			shouldError: false,
		},
		{
			name:        "execute_command is not in allowed list",
			toolName:    "execute_command",
			shouldError: true,
		},
		{
			name:        "list_files is not in allowed list",
			toolName:    "list_files",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.ValidateToolCall(tt.toolName, nil)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for tool %q, but got nil", tt.toolName)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for tool %q, but got: %v", tt.toolName, err)
				}
			}
		})
	}
}

func TestConstraintManager_ValidateToolCall_EmptyAllowedTools(t *testing.T) {
	// When allowed_tools is empty, all tools should be allowed
	config := ConstraintConfig{
		AllowedTools: []string{},
	}

	cm, err := NewConstraintManager(config)
	if err != nil {
		t.Fatalf("Failed to create constraint manager: %v", err)
	}

	tools := []string{
		"task_completion",
		"ask_question",
		"converse",
		"read_file",
		"write_file",
		"execute_command",
		"list_files",
	}

	for _, toolName := range tools {
		t.Run(toolName, func(t *testing.T) {
			err := cm.ValidateToolCall(toolName, nil)
			if err != nil {
				t.Errorf("Expected no error for tool %q when allowed_tools is empty, but got: %v", toolName, err)
			}
		})
	}
}
