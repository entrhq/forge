package approval

import (
	"strings"
	"testing"

	"github.com/entrhq/forge/pkg/executor/tui/types"
)

func TestNewPRRequest(t *testing.T) {
	branch := "feature/test"
	prTitle := "Add new feature"
	prDesc := "This PR adds a new feature"
	changes := "1 commit, 5 files changed"
	args := "Custom PR title"

	req := NewPRRequest(branch, prTitle, prDesc, changes, args, nil)

	if req == nil {
		t.Fatal("NewPRRequest returned nil")
	}
	if req.branch != branch {
		t.Errorf("Expected branch %q, got %q", branch, req.branch)
	}
	if req.prTitle != prTitle {
		t.Errorf("Expected prTitle %q, got %q", prTitle, req.prTitle)
	}
	if req.prDesc != prDesc {
		t.Errorf("Expected prDesc %q, got %q", prDesc, req.prDesc)
	}
	if req.changes != changes {
		t.Errorf("Expected changes %q, got %q", changes, req.changes)
	}
	if req.args != args {
		t.Errorf("Expected args %q, got %q", args, req.args)
	}
}

func TestPRRequest_Title(t *testing.T) {
	tests := []struct {
		name     string
		prTitle  string
		args     string
		expected string
	}{
		{
			name:     "uses generated title when different from args",
			prTitle:  "feat: Add new feature",
			args:     "Custom title",
			expected: "feat: Add new feature",
		},
		{
			name:     "uses args when no generated title",
			prTitle:  "",
			args:     "User provided title",
			expected: "User provided title",
		},
		{
			name:     "fallback when both empty",
			prTitle:  "",
			args:     "",
			expected: "Pull Request Preview",
		},
		{
			name:     "uses args when same as generated",
			prTitle:  "Same Title",
			args:     "Same Title",
			expected: "Same Title",
		},
		{
			name:     "uses generated title when args empty",
			prTitle:  "Generated PR Title",
			args:     "",
			expected: "Generated PR Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewPRRequest("", tt.prTitle, "", "", tt.args, nil)
			title := req.Title()

			if title != tt.expected {
				t.Errorf("Title() = %q, want %q", title, tt.expected)
			}
		})
	}
}

func TestPRRequest_Content(t *testing.T) {
	tests := []struct {
		name        string
		branch      string
		prTitle     string
		prDesc      string
		changes     string
		args        string
		contains    []string
		notContains []string
	}{
		{
			name:    "complete PR with all fields",
			branch:  "feature/new-api",
			prTitle: "feat: Add new API endpoint",
			prDesc:  "This PR implements a new API endpoint for users",
			changes: "3 commits\n5 files changed, 120 insertions(+), 20 deletions(-)",
			args:    "Custom title",
			contains: []string{
				"Branch:",
				"feature/new-api",
				"Generated Title:",
				"feat: Add new API endpoint",
				"Description:",
				"This PR implements a new API endpoint for users",
				"Commits & Changes:",
				"3 commits",
			},
		},
		{
			name:    "PR with only branch and changes",
			branch:  "bugfix/fix-login",
			prTitle: "",
			prDesc:  "",
			changes: "1 commit, 2 files changed",
			args:    "",
			contains: []string{
				"Branch:",
				"bugfix/fix-login",
				"Commits & Changes:",
			},
			notContains: []string{
				"Generated Title:",
				"Description:",
			},
		},
		{
			name:    "PR with description but no title",
			branch:  "feature/docs",
			prTitle: "",
			prDesc:  "Update documentation for API",
			changes: "",
			args:    "",
			contains: []string{
				"Branch:",
				"Description:",
				"Update documentation for API",
			},
			notContains: []string{
				"Generated Title:",
				"Commits & Changes:",
			},
		},
		{
			name:    "PR with same title and args",
			branch:  "main",
			prTitle: "Same Title",
			prDesc:  "Description text",
			changes: "Changes text",
			args:    "Same Title",
			contains: []string{
				"Branch:",
				"Description:",
				"Commits & Changes:",
			},
			notContains: []string{
				"Generated Title:", // Should not show generated title when same as args
			},
		},
		{
			name:        "minimal PR - only branch",
			branch:      "develop",
			prTitle:     "",
			prDesc:      "",
			changes:     "",
			args:        "",
			contains:    []string{"Branch:", "develop"},
			notContains: []string{"Generated Title:", "Description:", "Commits & Changes:"},
		},
		{
			name:        "empty PR",
			branch:      "",
			prTitle:     "",
			prDesc:      "",
			changes:     "",
			args:        "",
			contains:    []string{},
			notContains: []string{"Branch:", "Generated Title:", "Description:", "Commits & Changes:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewPRRequest(tt.branch, tt.prTitle, tt.prDesc, tt.changes, tt.args, nil)
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

func TestPRRequest_OnApprove(t *testing.T) {
	req := NewPRRequest(
		"feature/test",
		"Test PR",
		"Test description",
		"1 commit",
		"My PR Title",
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

func TestPRRequest_OnReject(t *testing.T) {
	req := NewPRRequest("", "", "", "", "", nil)

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

	if !strings.Contains(toastMsg.Details, "/pr") {
		t.Errorf("Expected details to mention /pr, got %q", toastMsg.Details)
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

func TestPRRequest_Interface(t *testing.T) {
	// Verify PRRequest implements ApprovalRequest interface
	var _ ApprovalRequest = (*PRRequest)(nil)
}
