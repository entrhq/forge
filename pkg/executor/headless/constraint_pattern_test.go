package headless

import (
	"testing"
)

func TestConstraintManager_FilePatternMatching(t *testing.T) {
	tests := []struct {
		name            string
		allowedPatterns []string
		deniedPatterns  []string
		toolName        string
		args            map[string]interface{}
		wantErr         bool
		errType         ViolationType
	}{
		{
			name:            "allowed pattern - write_file",
			allowedPatterns: []string{"src/*.go"},
			deniedPatterns:  []string{},
			toolName:        "write_file",
			args:            map[string]interface{}{"path": "src/main.go"},
			wantErr:         false,
		},
		{
			name:            "denied pattern - write_file",
			allowedPatterns: []string{"src/**"},
			deniedPatterns:  []string{"src/internal/**"},
			toolName:        "write_file",
			args:            map[string]interface{}{"path": "src/internal/secret.go"},
			wantErr:         true,
			errType:         ViolationFilePattern,
		},
		{
			name:            "disallowed pattern - write_file",
			allowedPatterns: []string{"src/*.go"},
			deniedPatterns:  []string{},
			toolName:        "write_file",
			args:            map[string]interface{}{"path": "tests/test.go"},
			wantErr:         true,
			errType:         ViolationFilePattern,
		},
		{
			name:            "allowed pattern - apply_diff",
			allowedPatterns: []string{"**/*.go"},
			deniedPatterns:  []string{"**/vendor/**"},
			toolName:        "apply_diff",
			args:            map[string]interface{}{"path": "src/pkg/utils.go"},
			wantErr:         false,
		},
		{
			name:            "denied pattern - apply_diff",
			allowedPatterns: []string{"**/*.go"},
			deniedPatterns:  []string{"vendor/**"},
			toolName:        "apply_diff",
			args:            map[string]interface{}{"path": "vendor/pkg/lib.go"},
			wantErr:         true,
			errType:         ViolationFilePattern,
		},
		{
			name:            "read_file not affected by patterns",
			allowedPatterns: []string{"src/*.go"},
			deniedPatterns:  []string{},
			toolName:        "read_file",
			args:            map[string]interface{}{"path": "tests/test.go"},
			wantErr:         false,
		},
		{
			name:            "no patterns - allow all",
			allowedPatterns: []string{},
			deniedPatterns:  []string{},
			toolName:        "write_file",
			args:            map[string]interface{}{"path": "anywhere/file.go"},
			wantErr:         false,
		},
		{
			name:            "complex pattern - allow src except tests",
			allowedPatterns: []string{"src/**"},
			deniedPatterns:  []string{"src/**/*_test.go"},
			toolName:        "write_file",
			args:            map[string]interface{}{"path": "src/pkg/main_test.go"},
			wantErr:         true,
			errType:         ViolationFilePattern,
		},
		{
			name:            "complex pattern - allow production code",
			allowedPatterns: []string{"src/**"},
			deniedPatterns:  []string{"src/**/*_test.go"},
			toolName:        "write_file",
			args:            map[string]interface{}{"path": "src/pkg/main.go"},
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ConstraintConfig{
				AllowedPatterns: tt.allowedPatterns,
				DeniedPatterns:  tt.deniedPatterns,
			}

			cm, err := NewConstraintManager(config, ModeWrite)
			if err != nil {
				t.Fatalf("NewConstraintManager() error = %v", err)
			}

			err = cm.ValidateToolCall(tt.toolName, tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateToolCall() expected error but got nil")
					return
				}
				violation, ok := err.(*ConstraintViolation)
				if !ok {
					t.Errorf("ValidateToolCall() error is not a ConstraintViolation: %v", err)
					return
				}
				if violation.Type != tt.errType {
					t.Errorf("ValidateToolCall() error type = %v, want %v", violation.Type, tt.errType)
				}
			} else if err != nil {
				t.Errorf("ValidateToolCall() unexpected error = %v", err)
			}
		})
	}
}

func TestConstraintManager_FilePatternMatchingWithReadOnlyMode(t *testing.T) {
	config := ConstraintConfig{
		AllowedPatterns: []string{"src/*.go"},
		DeniedPatterns:  []string{},
	}

	cm, err := NewConstraintManager(config, ModeReadOnly)
	if err != nil {
		t.Fatalf("NewConstraintManager() error = %v", err)
	}

	// In read-only mode, write operations should be blocked regardless of patterns
	err = cm.ValidateToolCall("write_file", map[string]interface{}{"path": "src/main.go"})
	if err == nil {
		t.Error("ValidateToolCall() expected error for write_file in read-only mode")
	}

	violation, ok := err.(*ConstraintViolation)
	if !ok {
		t.Fatalf("ValidateToolCall() error is not a ConstraintViolation: %v", err)
	}
	if violation.Type != ViolationReadOnlyMode {
		t.Errorf("ValidateToolCall() error type = %v, want %v", violation.Type, ViolationReadOnlyMode)
	}
}

func TestConstraintManager_FilePatternMatchingWithAllowedTools(t *testing.T) {
	config := ConstraintConfig{
		AllowedTools:    []string{"read_file", "write_file"},
		AllowedPatterns: []string{"src/*.go"},
		DeniedPatterns:  []string{},
	}

	cm, err := NewConstraintManager(config, ModeWrite)
	if err != nil {
		t.Fatalf("NewConstraintManager() error = %v", err)
	}

	// write_file is allowed by tool restrictions
	// Pattern should allow src/main.go
	err = cm.ValidateToolCall("write_file", map[string]interface{}{"path": "src/main.go"})
	if err != nil {
		t.Errorf("ValidateToolCall() unexpected error = %v", err)
	}

	// write_file is allowed by tool restrictions
	// Pattern should block tests/test.go
	err = cm.ValidateToolCall("write_file", map[string]interface{}{"path": "tests/test.go"})
	if err == nil {
		t.Error("ValidateToolCall() expected error for disallowed pattern")
	}

	violation, ok := err.(*ConstraintViolation)
	if !ok {
		t.Fatalf("ValidateToolCall() error is not a ConstraintViolation: %v", err)
	}
	if violation.Type != ViolationFilePattern {
		t.Errorf("ValidateToolCall() error type = %v, want %v", violation.Type, ViolationFilePattern)
	}
}

func TestConstraintManager_InvalidPatterns(t *testing.T) {
	tests := []struct {
		name            string
		allowedPatterns []string
		deniedPatterns  []string
		wantErr         bool
	}{
		{
			name:            "valid patterns",
			allowedPatterns: []string{"src/*.go", "**/*.txt"},
			deniedPatterns:  []string{"**/test_*.go"},
			wantErr:         false,
		},
		{
			name:            "invalid allowed pattern",
			allowedPatterns: []string{"[invalid"},
			deniedPatterns:  []string{},
			wantErr:         true,
		},
		{
			name:            "invalid denied pattern",
			allowedPatterns: []string{"src/*.go"},
			deniedPatterns:  []string{"[invalid"},
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ConstraintConfig{
				AllowedPatterns: tt.allowedPatterns,
				DeniedPatterns:  tt.deniedPatterns,
			}

			_, err := NewConstraintManager(config, ModeWrite)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewConstraintManager() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
