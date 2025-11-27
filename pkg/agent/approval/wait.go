package approval

import (
	"context"
	"time"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/types"
)

// waitForResponse waits for the user's approval response
func (m *Manager) waitForResponse(ctx context.Context, approvalID string, toolCall tools.ToolCall, responseChannel chan *types.ApprovalResponse) (bool, bool) {
	timeout := time.NewTimer(m.timeout)
	defer timeout.Stop()

	select {
	case <-ctx.Done():
		return false, false

	case <-timeout.C:
		m.emitEvent(types.NewToolApprovalTimeoutEvent(approvalID, toolCall.ToolName))
		return false, true

	case response, ok := <-responseChannel:
		if !ok {
			// Channel closed, treat as rejection
			m.emitEvent(types.NewToolApprovalRejectedEvent(approvalID, toolCall.ToolName))
			return false, false
		}
		if response.IsGranted() {
			m.emitEvent(types.NewToolApprovalGrantedEvent(approvalID, toolCall.ToolName))
			return true, false
		}
		m.emitEvent(types.NewToolApprovalRejectedEvent(approvalID, toolCall.ToolName))
		return false, false
	}
}
