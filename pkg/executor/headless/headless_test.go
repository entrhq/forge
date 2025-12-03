package headless

import (
	"context"
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Task:         "test task",
				Mode:         ModeWrite,
				WorkspaceDir: "/tmp/test",
				Constraints: ConstraintConfig{
					MaxFiles:        10,
					MaxLinesChanged: 500,
					Timeout:         5 * time.Minute,
				},
			},
			wantErr: false,
		},
		{
			name: "missing task",
			config: &Config{
				Mode:         ModeWrite,
				WorkspaceDir: "/tmp/test",
			},
			wantErr: true,
		},
		{
			name: "invalid mode",
			config: &Config{
				Task:         "test",
				Mode:         "invalid",
				WorkspaceDir: "/tmp/test",
			},
			wantErr: true,
		},
		{
			name: "missing workspace",
			config: &Config{
				Task: "test",
				Mode: ModeWrite,
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			config: &Config{
				Task:         "test",
				Mode:         ModeWrite,
				WorkspaceDir: "/tmp/test",
				Constraints: ConstraintConfig{
					Timeout: -1 * time.Minute,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Mode != ModeWrite {
		t.Errorf("DefaultConfig() mode = %v, want %v", config.Mode, ModeWrite)
	}

	if config.Constraints.MaxFiles != 10 {
		t.Errorf("DefaultConfig() max_files = %v, want 10", config.Constraints.MaxFiles)
	}

	if config.Constraints.MaxLinesChanged != 500 {
		t.Errorf("DefaultConfig() max_lines_changed = %v, want 500", config.Constraints.MaxLinesChanged)
	}

	if len(config.Constraints.AllowedTools) == 0 {
		t.Error("DefaultConfig() should have allowed tools")
	}
}

func TestConstraintManager_ValidateToolCall(t *testing.T) {
	config := ConstraintConfig{
		AllowedTools: []string{"read_file", "write_file"},
	}

	cm, err := NewConstraintManager(config, "write")
	if err != nil {
		t.Fatalf("NewConstraintManager() error = %v", err)
	}

	tests := []struct {
		name     string
		toolName string
		wantErr  bool
	}{
		{
			name:     "allowed tool",
			toolName: "read_file",
			wantErr:  false,
		},
		{
			name:     "disallowed tool",
			toolName: "execute_command",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.ValidateToolCall(tt.toolName, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToolCall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConstraintManager_RecordFileModification(t *testing.T) {
	config := ConstraintConfig{
		MaxFiles:        2,
		MaxLinesChanged: 50,
	}

	cm, err := NewConstraintManager(config, "write")
	if err != nil {
		t.Fatalf("NewConstraintManager() error = %v", err)
	}

	// Record first file modification
	err = cm.RecordFileModification("file1.go", 10, 5)
	if err != nil {
		t.Errorf("RecordFileModification() error = %v", err)
	}

	// Record second file modification
	err = cm.RecordFileModification("file2.go", 15, 10)
	if err != nil {
		t.Errorf("RecordFileModification() error = %v", err)
	}

	// Third file should exceed limit
	err = cm.RecordFileModification("file3.go", 5, 3)
	if err == nil {
		t.Error("RecordFileModification() should fail when max_files exceeded")
	}

	// Check total lines changed
	state := cm.GetCurrentState()
	expectedAdded := 10 + 15  // file1 added + file2 added
	expectedRemoved := 5 + 10 // file1 removed + file2 removed
	if state.TotalLinesAdded != expectedAdded {
		t.Errorf("TotalLinesAdded = %v, want %v", state.TotalLinesAdded, expectedAdded)
	}
	if state.TotalLinesRemoved != expectedRemoved {
		t.Errorf("TotalLinesRemoved = %v, want %v", state.TotalLinesRemoved, expectedRemoved)
	}
}

func TestConstraintManager_TokenUsage(t *testing.T) {
	config := ConstraintConfig{
		MaxTokens: 100,
	}

	cm, err := NewConstraintManager(config, "write")
	if err != nil {
		t.Fatalf("NewConstraintManager() error = %v", err)
	}

	// Record token usage within limit
	err = cm.RecordTokenUsage(50)
	if err != nil {
		t.Errorf("RecordTokenUsage() error = %v", err)
	}

	// Record more tokens
	err = cm.RecordTokenUsage(40)
	if err != nil {
		t.Errorf("RecordTokenUsage() error = %v", err)
	}

	// Exceed limit
	err = cm.RecordTokenUsage(20)
	if err == nil {
		t.Error("RecordTokenUsage() should fail when max_tokens exceeded")
	}
}

func TestConstraintManager_Timeout(t *testing.T) {
	config := ConstraintConfig{
		Timeout: 100 * time.Millisecond,
	}

	cm, err := NewConstraintManager(config, "write")
	if err != nil {
		t.Fatalf("NewConstraintManager() error = %v", err)
	}

	// Should not timeout immediately
	err = cm.CheckTimeout()
	if err != nil {
		t.Errorf("CheckTimeout() should not fail immediately, got: %v", err)
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should timeout now
	err = cm.CheckTimeout()
	if err == nil {
		t.Error("CheckTimeout() should fail after timeout period")
	}
}

func TestPatternMatcher(t *testing.T) {
	pm, err := NewPatternMatcher(
		[]string{"**/*.go", "*.md"},
		[]string{"vendor/**"},
	)
	if err != nil {
		t.Fatalf("NewPatternMatcher() error = %v", err)
	}

	tests := []struct {
		name    string
		path    string
		allowed bool
	}{
		{
			name:    "allowed go file",
			path:    "pkg/agent/default.go",
			allowed: true,
		},
		{
			name:    "allowed md file",
			path:    "README.md",
			allowed: true,
		},
		{
			name:    "denied vendor file",
			path:    "vendor/github.com/test/file.go",
			allowed: false,
		},
		{
			name:    "disallowed extension",
			path:    "main.txt",
			allowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := pm.IsAllowed(tt.path)
			if allowed != tt.allowed {
				t.Errorf("IsAllowed(%s) = %v, want %v", tt.path, allowed, tt.allowed)
			}
		})
	}
}

func TestQualityGateRunner(t *testing.T) {
	// Create a mock quality gate that always passes
	passingGate := &mockQualityGate{
		name:     "Passing Gate",
		required: true,
		passFunc: func() error { return nil },
	}

	// Create a mock quality gate that always fails
	failingGate := &mockQualityGate{
		name:     "Failing Gate",
		required: true,
		passFunc: func() error { return context.DeadlineExceeded },
	}

	tests := []struct {
		name       string
		gates      []QualityGate
		wantPassed bool
	}{
		{
			name:       "all gates pass",
			gates:      []QualityGate{passingGate},
			wantPassed: true,
		},
		{
			name:       "required gate fails",
			gates:      []QualityGate{failingGate},
			wantPassed: false,
		},
		{
			name:       "mixed results",
			gates:      []QualityGate{passingGate, failingGate},
			wantPassed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewQualityGateRunner(tt.gates)
			results := runner.RunAll(context.Background(), "/tmp", nil)

			if results.AllPassed != tt.wantPassed {
				t.Errorf("RunAll() AllPassed = %v, want %v", results.AllPassed, tt.wantPassed)
			}
		})
	}
}

// mockQualityGate is a mock implementation of QualityGate for testing
type mockQualityGate struct {
	name     string
	required bool
	passFunc func() error
}

func (m *mockQualityGate) Name() string {
	return m.name
}

func (m *mockQualityGate) Required() bool {
	return m.required
}

func (m *mockQualityGate) Execute(ctx context.Context, workspaceDir string) error {
	return m.passFunc()
}
