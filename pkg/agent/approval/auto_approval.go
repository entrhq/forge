package approval

import (
	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/types"
)

// checkAutoApproval checks if the tool or command should be auto-approved
// Returns (approved, autoApproved) where autoApproved indicates if a decision was made
func (m *Manager) checkAutoApproval(approvalID string, toolCall tools.ToolCall, argsMap map[string]any) (bool, bool) {
	// Special handling for execute_command: always check command whitelist first
	// The execute_command tool uses a per-command whitelist, not tool-level auto-approval
	if toolCall.ToolName == "execute_command" {
		if m.isCommandWhitelisted(approvalID, argsMap) {
			return true, true
		}
		return false, false
	}

	// For all other tools, check if tool is auto-approved
	if config.IsToolAutoApproved(toolCall.ToolName) {
		m.emitEvent(types.NewToolApprovalGrantedEvent(approvalID, toolCall.ToolName))
		return true, true
	}

	return false, false
}

// isCommandWhitelisted checks if a command is whitelisted
func (m *Manager) isCommandWhitelisted(approvalID string, argsMap map[string]any) bool {
	cmdInterface, ok := argsMap["command"]
	if !ok {
		return false
	}

	cmd, ok := cmdInterface.(string)
	if !ok {
		return false
	}

	if config.IsCommandWhitelisted(cmd) {
		m.emitEvent(types.NewToolApprovalGrantedEvent(approvalID, "execute_command"))
		return true
	}

	return false
}
