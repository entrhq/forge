package approval

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestApprovalRequestInterface verifies that the ApprovalRequest interface
// is properly defined and can be implemented
func TestApprovalRequestInterface(t *testing.T) {
	// Create instances to verify they implement the interface
	var commit ApprovalRequest = NewCommitRequest(nil, "", "", "", nil)
	var pr ApprovalRequest = NewPRRequest("", "", "", "", "", nil)

	// Verify all interface methods are callable
	_ = commit.Title()
	_ = commit.Content()
	_ = commit.OnApprove()
	_ = commit.OnReject()

	_ = pr.Title()
	_ = pr.Content()
	_ = pr.OnApprove()
	_ = pr.OnReject()
}

// TestRequestMsg verifies the RequestMsg struct
func TestRequestMsg(t *testing.T) {
	req := NewCommitRequest(nil, "test", "", "", nil)
	msg := RequestMsg{Request: req}

	if msg.Request == nil {
		t.Error("RequestMsg.Request should not be nil")
	}

	if msg.Request.Title() != "Commit Preview" {
		t.Errorf("Expected title 'Commit Preview', got %q", msg.Request.Title())
	}
}

// mockApprovalRequest is a simple mock implementation for testing
type mockApprovalRequest struct {
	title        string
	content      string
	approveCmd   tea.Cmd
	rejectCmd    tea.Cmd
	approveCalls int
	rejectCalls  int
}

func (m *mockApprovalRequest) Title() string {
	return m.title
}

func (m *mockApprovalRequest) Content() string {
	return m.content
}

func (m *mockApprovalRequest) OnApprove() tea.Cmd {
	m.approveCalls++
	return m.approveCmd
}

func (m *mockApprovalRequest) OnReject() tea.Cmd {
	m.rejectCalls++
	return m.rejectCmd
}

// TestMockApprovalRequest verifies our mock implementation
func TestMockApprovalRequest(t *testing.T) {
	mock := &mockApprovalRequest{
		title:   "Test Title",
		content: "Test Content",
	}

	// Verify interface compliance
	var _ ApprovalRequest = mock

	// Test Title
	if mock.Title() != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %q", mock.Title())
	}

	// Test Content
	if mock.Content() != "Test Content" {
		t.Errorf("Expected content 'Test Content', got %q", mock.Content())
	}

	// Test OnApprove
	mock.OnApprove()
	if mock.approveCalls != 1 {
		t.Errorf("Expected 1 approve call, got %d", mock.approveCalls)
	}

	// Test OnReject
	mock.OnReject()
	if mock.rejectCalls != 1 {
		t.Errorf("Expected 1 reject call, got %d", mock.rejectCalls)
	}
}

// TestRequestMsgWithDifferentRequests verifies RequestMsg works with different implementations
func TestRequestMsgWithDifferentRequests(t *testing.T) {
	tests := []struct {
		name    string
		request ApprovalRequest
		wantMsg bool
	}{
		{
			name:    "commit request",
			request: NewCommitRequest([]string{"file.go"}, "message", "diff", "", nil),
			wantMsg: true,
		},
		{
			name:    "pr request",
			request: NewPRRequest("branch", "title", "desc", "changes", "", nil),
			wantMsg: true,
		},
		{
			name: "mock request",
			request: &mockApprovalRequest{
				title:   "Mock",
				content: "Content",
			},
			wantMsg: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := RequestMsg{Request: tt.request}

			if (msg.Request != nil) != tt.wantMsg {
				t.Errorf("RequestMsg.Request presence = %v, want %v", msg.Request != nil, tt.wantMsg)
			}

			if msg.Request != nil {
				// Verify we can call interface methods
				_ = msg.Request.Title()
				_ = msg.Request.Content()
			}
		})
	}
}
