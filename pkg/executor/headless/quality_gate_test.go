package headless

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestQualityGateRunner_RunAll(t *testing.T) {
	tests := []struct {
		name          string
		gates         []QualityGate
		wantAllPassed bool
		wantFailedLen int
	}{
		{
			name:          "no gates",
			gates:         []QualityGate{},
			wantAllPassed: true,
			wantFailedLen: 0,
		},
		{
			name: "all gates pass",
			gates: []QualityGate{
				NewCommandQualityGate("test1", "echo hello", true),
				NewCommandQualityGate("test2", "echo world", false),
			},
			wantAllPassed: true,
			wantFailedLen: 0,
		},
		{
			name: "required gate fails",
			gates: []QualityGate{
				NewCommandQualityGate("test1", "echo hello", false),
				NewCommandQualityGate("test2", "exit 1", true),
			},
			wantAllPassed: false,
			wantFailedLen: 1,
		},
		{
			name: "optional gate fails",
			gates: []QualityGate{
				NewCommandQualityGate("test1", "echo hello", true),
				NewCommandQualityGate("test2", "exit 1", false),
			},
			wantAllPassed: true,
			wantFailedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewQualityGateRunner(tt.gates)
			ctx := context.Background()

			// Create temp workspace
			tmpDir := t.TempDir()

			results := runner.RunAll(ctx, tmpDir)

			if results.AllPassed != tt.wantAllPassed {
				t.Errorf("AllPassed = %v, want %v", results.AllPassed, tt.wantAllPassed)
			}

			failedGates := results.GetFailedGates()
			if len(failedGates) != tt.wantFailedLen {
				t.Errorf("failed gates len = %d, want %d", len(failedGates), tt.wantFailedLen)
			}
		})
	}
}

func TestQualityGateResults_FormatFeedbackMessage(t *testing.T) {
	results := &QualityGateResults{
		AllPassed: false,
		Results: []QualityGateResult{
			{
				Name:     "lint",
				Required: true,
				Passed:   false,
				Error:    "quality gate 'lint' failed: exit status 1",
			},
			{
				Name:     "test",
				Required: true,
				Passed:   false,
				Error:    "quality gate 'test' failed: exit status 2",
			},
		},
	}

	msg := results.FormatFeedbackMessage(1, 3)

	if msg == "" {
		t.Error("Expected non-empty feedback message")
	}

	// Check that message contains key elements
	expectedParts := []string{
		"attempt 1/3",
		"❌ lint",
		"❌ test",
		"task_completion",
	}

	for _, part := range expectedParts {
		if !contains(msg, part) {
			t.Errorf("Message missing expected part: %s\nMessage: %s", part, msg)
		}
	}
}

func TestCommandQualityGate_Execute(t *testing.T) {
	tests := []struct {
		name    string
		gate    *CommandQualityGate
		wantErr bool
	}{
		{
			name:    "successful command",
			gate:    NewCommandQualityGate("test", "echo hello", true),
			wantErr: false,
		},
		{
			name:    "failing command",
			gate:    NewCommandQualityGate("test", "exit 1", true),
			wantErr: true,
		},
		{
			name:    "command that creates file",
			gate:    NewCommandQualityGate("test", "touch test.txt", false),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			ctx := context.Background()
			err := tt.gate.Execute(ctx, tmpDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			// For error cases, check that we get a QualityGateError
			if err != nil {
				if _, ok := err.(*QualityGateError); !ok {
					t.Errorf("Expected QualityGateError, got %T", err)
				}
			}

			// For the file creation test, verify the file was created
			if tt.name == "command that creates file" {
				testFile := filepath.Join(tmpDir, "test.txt")
				if _, err := os.Stat(testFile); os.IsNotExist(err) {
					t.Error("Expected test file to be created")
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
