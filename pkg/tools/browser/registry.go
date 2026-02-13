package browser

import (
	"github.com/entrhq/forge/pkg/agent/tools"
)

// ToolRegistry manages dynamic browser tool registration.
type ToolRegistry struct {
	manager *SessionManager
	tools   []tools.Tool
}

// NewToolRegistry creates a new browser tool registry.
func NewToolRegistry(manager *SessionManager) *ToolRegistry {
	return &ToolRegistry{
		manager: manager,
		tools:   make([]tools.Tool, 0),
	}
}

// RegisterTools creates and returns all browser tools.
// This should be called by the main tool registry to get the browser tools.
func (r *ToolRegistry) RegisterTools() []tools.Tool {
	if len(r.tools) > 0 {
		return r.tools
	}

	// Session management tools (always available)
	r.tools = append(r.tools,
		NewStartSessionTool(r.manager),
		NewListSessionsTool(r.manager),
		NewCloseSessionTool(r.manager),
	)

	// Browser interaction tools (available when sessions exist)
	r.tools = append(r.tools,
		NewNavigateTool(r.manager),
		NewExtractContentTool(r.manager),
		NewClickTool(r.manager),
		NewFillTool(r.manager),
		NewWaitTool(r.manager),
		NewSearchTool(r.manager),
	)

	return r.tools
}

// GetTools returns the current set of registered tools.
func (r *ToolRegistry) GetTools() []tools.Tool {
	return r.tools
}

// ShouldShowBrowserTools returns true if browser interaction tools should be visible.
// This is used for conditional tool visibility based on session state.
func (r *ToolRegistry) ShouldShowBrowserTools() bool {
	return r.manager.HasSessions()
}

// GetSessionManager returns the underlying session manager.
// This allows external code to check session state if needed.
func (r *ToolRegistry) GetSessionManager() *SessionManager {
	return r.manager
}
