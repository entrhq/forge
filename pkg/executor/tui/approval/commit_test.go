package approval

import (
	"strings"
	"testing"

	"github.com/entrhq/forge/pkg/executor/tui/types"
)

func TestNewCommitRequest(t *testing.T) {
	files := []string{"file1.go", "file2.go"}
	message := "Test commit"
	diff := "some diff"
	args := "-m 'test'"

	req := NewCommitRequest(files, message, diff, args, nil)

	if req == nil {
		t.Fatal("NewCommitRequest returned nil")
	}
	if len(req.files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(req.files))
	}
	if req.message != message {
		t.Errorf("Expected message %q, got %q", message, req.message)
	}
	if req.diff != diff {
		t.Errorf("Expected diff %q, got %q", diff, req.diff)
	}
	if req.args != args {
		t.Errorf("Expected args %q, got %q", args, req.args)
	}
}

func TestCommitRequest_Title(t *testing.T) {
	req := NewCommitRequest(nil, "", "", "", nil)

	title := req.Title()

	expected := commitPreviewTitle
	if title != expected {
		t.Errorf("Title() = %q, want %q", title, expected)
	}
}

func TestCommitRequest_Content(t *testing.T) {
	tests := []struct {
		name        string
		files       []string
		message     string
		diff        string
		contains    []string
		notContains []string
	}{
		{
			name:    "complete commit with all fields",
			files:   []string{"main.go", "test.go"},
			message: "Add new feature",
			diff:    "+func NewFeature() {}",
			contains: []string{
				"Files to commit:",
				"main.go",
				"test.go",
				"Commit Message:",
				"Add new feature",
				"Changes:",
			},
		},
		{
			name:    "commit with only files",
			files:   []string{"config.yaml"},
			message: "",
			diff:    "",
			contains: []string{
				"Files to commit:",
				"config.yaml",
			},
			notContains: []string{
				"Commit Message:",
				"Changes:",
			},
		},
		{
			name:    "commit with only message",
			files:   nil,
			message: "Update documentation",
			diff:    "",
			contains: []string{
				"Commit Message:",
				"Update documentation",
			},
			notContains: []string{
				"Files to commit:",
				"Changes:",
			},
		},
		{
			name:    "commit with only diff",
			files:   nil,
			message: "",
			diff:    "-old line\n+new line",
			contains: []string{
				"Changes:",
			},
			notContains: []string{
				"Files to commit:",
				"Commit Message:",
			},
		},
		{
			name:     "empty commit",
			files:    nil,
			message:  "",
			diff:     "",
			contains: []string{},
			notContains: []string{
				"Files to commit:",
				"Commit Message:",
				"Changes:",
			},
		},
		{
			name:  "multiple files",
			files: []string{"a.go", "b.go", "c.go", "d.go"},
			contains: []string{
				"Files to commit:",
				"a.go",
				"b.go",
				"c.go",
				"d.go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewCommitRequest(tt.files, tt.message, tt.diff, "", nil)
			content := req.Content()

			for _, expected := range tt.contains {
				if !strings.Contains(content, expected) {
					t.Errorf("Content() missing expected string %q\nGot:\n%s", expected, content)
				}
			}

			for _, notExpected := range tt.notContains {
				if strings.Contains(content, notExpected) {
					t.Errorf("Content() contains unexpected string %q\nGot:\n%s", notExpected, content)
				}
			}
		})
	}
}

func TestCommitRequest_OnApprove(t *testing.T) {
	req := NewCommitRequest(
		[]string{"test.go"},
		"Test message",
		"+test",
		"-m 'test'",
		nil,
	)

	cmd := req.OnApprove()
	if cmd == nil {
		t.Fatal("OnApprove() returned nil command")
	}

	// OnApprove returns a tea.Batch command
	// We verify it's a valid command by checking it's not nil
	// The actual execution happens in the Bubble Tea runtime
	// Testing the full execution would require mocking the slash handler,
	// which is complex and not necessary for coverage
}

func TestCommitRequest_OnReject(t *testing.T) {
	req := NewCommitRequest(nil, "", "", "", nil)

	cmd := req.OnReject()
	if cmd == nil {
		t.Fatal("OnReject() returned nil command")
	}

	msg := cmd()

	toastMsg, ok := msg.(types.ToastMsg)
	if !ok {
		t.Fatalf("Expected ToastMsg, got %T", msg)
	}

	if toastMsg.Message != "Canceled" {
		t.Errorf("Expected message 'Canceled', got %q", toastMsg.Message)
	}

	if !strings.Contains(toastMsg.Details, "/commit") {
		t.Errorf("Expected details to mention /commit, got %q", toastMsg.Details)
	}

	if !strings.Contains(toastMsg.Details, "canceled") {
		t.Errorf("Expected details to mention canceled, got %q", toastMsg.Details)
	}

	if toastMsg.IsError {
		t.Error("Expected IsError to be false for cancel message")
	}

	if toastMsg.Icon != "ℹ️" {
		t.Errorf("Expected info icon, got %q", toastMsg.Icon)
	}
}

func TestCommitRequest_Interface(t *testing.T) {
	// Verify CommitRequest implements ApprovalRequest interface
	var _ ApprovalRequest = (*CommitRequest)(nil)
}
