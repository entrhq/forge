package retrieval

import (
	"strings"
	"testing"

	"github.com/entrhq/forge/pkg/types"
)

func TestBuildWindow_Empty(t *testing.T) {
	got := buildWindow(nil, "")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestBuildWindow_UserMessageOnly(t *testing.T) {
	got := buildWindow(nil, "hello world")
	if !strings.Contains(got, "hello world") {
		t.Errorf("window does not contain user message: %q", got)
	}
	if !strings.HasPrefix(got, "user: ") {
		t.Errorf("window does not start with 'user: ': %q", got)
	}
}

func TestBuildWindow_SkipsToolMessages(t *testing.T) {
	history := []*types.Message{
		types.NewUserMessage("hi"),
		{Role: types.RoleTool, Content: "tool output"},
		types.NewAssistantMessage("response"),
	}
	got := buildWindow(history, "")
	if strings.Contains(got, "tool output") {
		t.Errorf("window should not contain tool messages, got: %q", got)
	}
	if !strings.Contains(got, "hi") {
		t.Errorf("window missing user message, got: %q", got)
	}
	if !strings.Contains(got, "response") {
		t.Errorf("window missing assistant message, got: %q", got)
	}
}

func TestBuildWindow_TailsToWindowMessages(t *testing.T) {
	// Create more than windowMessages (6) messages.
	history := make([]*types.Message, 0, 10)
	for range 10 {
		history = append(history, types.NewUserMessage("msg"))
	}
	// Add a distinguishable first message that should be excluded.
	first := types.NewUserMessage("FIRST_SHOULD_BE_EXCLUDED")
	all := append([]*types.Message{first}, history...)

	got := buildWindow(all, "")
	if strings.Contains(got, "FIRST_SHOULD_BE_EXCLUDED") {
		t.Error("window should not include messages beyond the last windowMessages entries")
	}
}

func TestBuildWindow_TruncatesToMaxChars(t *testing.T) {
	// Single very long message.
	longMsg := strings.Repeat("x", windowMaxChars*3)
	history := []*types.Message{types.NewUserMessage(longMsg)}
	got := buildWindow(history, "")
	if len(got) > windowMaxChars {
		t.Errorf("window len = %d, want <= %d", len(got), windowMaxChars)
	}
}

func TestBuildWindow_IncludesAllRoles(t *testing.T) {
	history := []*types.Message{
		types.NewSystemMessage("sys prompt"),
		types.NewUserMessage("user question"),
		types.NewAssistantMessage("assistant answer"),
	}
	got := buildWindow(history, "follow up")
	if !strings.Contains(got, "user question") {
		t.Error("window missing user message")
	}
	if !strings.Contains(got, "assistant answer") {
		t.Error("window missing assistant message")
	}
	if !strings.Contains(got, "follow up") {
		t.Error("window missing current user message")
	}
}
