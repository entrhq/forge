package browser

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/config"
)

// ListSessionsTool lists all active browser sessions.
type ListSessionsTool struct {
	manager *SessionManager
}

// NewListSessionsTool creates a new list sessions tool.
func NewListSessionsTool(manager *SessionManager) *ListSessionsTool {
	return &ListSessionsTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *ListSessionsTool) Name() string {
	return "list_browser_sessions"
}

// Description returns the tool description.
func (t *ListSessionsTool) Description() string {
	return "List all active browser sessions with their current state and metadata."
}

// Schema returns the tool's JSON schema.
func (t *ListSessionsTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{},
		[]string{},
	)
}

// ListSessionsInput represents the parameters (none for this tool).
type ListSessionsInput struct {
	XMLName xml.Name `xml:"arguments"`
}

// Execute lists all sessions.
func (t *ListSessionsTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Get all sessions
	sessions := t.manager.ListSessions()

	if len(sessions) == 0 {
		return "No active browser sessions.\n\nUse start_browser_session to create a new session.", nil, nil
	}

	// Format sessions
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Active Browser Sessions: %d\n\n", len(sessions)))

	for i, session := range sessions {
		mode := "headed"
		if session.Headless {
			mode = "headless"
		}

		idleTime := time.Since(session.LastUsedAt)
		age := time.Since(session.CreatedAt)

		result.WriteString(fmt.Sprintf(`%d. %s
   URL: %s
   Mode: %s
   Age: %s
   Last Used: %s ago

`,
			i+1,
			session.Name,
			session.CurrentURL,
			mode,
			formatDuration(age),
			formatDuration(idleTime),
		))
	}

	result.WriteString("Use close_browser_session to close a session when finished.")

	return result.String(), nil, nil
}

// RequiresApproval returns whether this tool requires approval.
func (t *ListSessionsTool) RequiresApproval() bool {
	return false // Listing sessions doesn't require approval
}

// IsLoopBreaking returns whether this tool breaks the agent loop.
func (t *ListSessionsTool) IsLoopBreaking() bool {
	return false
}

// ApprovalMessage returns the message shown when requesting approval.
func (t *ListSessionsTool) ApprovalMessage(params map[string]any) string {
	return ""
}

// ShouldShow returns whether this tool should be visible.
// Session management tools are only shown when browser automation is enabled in settings.
func (t *ListSessionsTool) ShouldShow() bool {
	if !config.IsInitialized() {
		return false
	}
	ui := config.GetUI()
	if ui == nil {
		return false
	}
	return ui.IsBrowserEnabled()
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
