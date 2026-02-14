package agent

import (
	"context"
	"testing"

	"github.com/entrhq/forge/pkg/agent/tools"
)

// mockConditionalTool is a mock tool that implements ConditionallyVisible
type mockConditionalTool struct {
	name       string
	shouldShow bool
}

func (m *mockConditionalTool) Name() string        { return m.name }
func (m *mockConditionalTool) Description() string { return "mock tool" }
func (m *mockConditionalTool) Schema() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}
func (m *mockConditionalTool) Execute(ctx context.Context, args []byte) (string, map[string]interface{}, error) {
	return "", nil, nil
}
func (m *mockConditionalTool) IsLoopBreaking() bool { return false }
func (m *mockConditionalTool) RequiresApproval(params map[string]any) bool {
	return false
}
func (m *mockConditionalTool) ApprovalMessage(params map[string]any) string { return "" }
func (m *mockConditionalTool) ShouldShow() bool                             { return m.shouldShow }

// mockRegularTool is a mock tool that does not implement ConditionallyVisible
type mockRegularTool struct {
	name string
}

func (m *mockRegularTool) Name() string        { return m.name }
func (m *mockRegularTool) Description() string { return "mock tool" }
func (m *mockRegularTool) Schema() map[string]interface{} {
	return map[string]interface{}{"type": "object"}
}
func (m *mockRegularTool) Execute(ctx context.Context, args []byte) (string, map[string]interface{}, error) {
	return "", nil, nil
}
func (m *mockRegularTool) IsLoopBreaking() bool                         { return false }
func (m *mockRegularTool) RequiresApproval(params map[string]any) bool  { return false }
func (m *mockRegularTool) ApprovalMessage(params map[string]any) string { return "" }

func TestGetToolsList_ConditionalVisibility(t *testing.T) {
	tests := []struct {
		name            string
		registeredTools []tools.Tool
		expectedNames   []string
	}{
		{
			name: "all regular tools shown",
			registeredTools: []tools.Tool{
				&mockRegularTool{name: "tool1"},
				&mockRegularTool{name: "tool2"},
			},
			expectedNames: []string{"tool1", "tool2"},
		},
		{
			name: "conditional tool shown when ShouldShow is true",
			registeredTools: []tools.Tool{
				&mockRegularTool{name: "tool1"},
				&mockConditionalTool{name: "conditional1", shouldShow: true},
			},
			expectedNames: []string{"tool1", "conditional1"},
		},
		{
			name: "conditional tool hidden when ShouldShow is false",
			registeredTools: []tools.Tool{
				&mockRegularTool{name: "tool1"},
				&mockConditionalTool{name: "conditional1", shouldShow: false},
			},
			expectedNames: []string{"tool1"},
		},
		{
			name: "mixed conditional tools",
			registeredTools: []tools.Tool{
				&mockRegularTool{name: "tool1"},
				&mockConditionalTool{name: "conditional1", shouldShow: true},
				&mockConditionalTool{name: "conditional2", shouldShow: false},
				&mockRegularTool{name: "tool2"},
			},
			expectedNames: []string{"tool1", "conditional1", "tool2"},
		},
		{
			name: "all conditional tools hidden",
			registeredTools: []tools.Tool{
				&mockConditionalTool{name: "conditional1", shouldShow: false},
				&mockConditionalTool{name: "conditional2", shouldShow: false},
			},
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create agent with mock tools
			agent := &DefaultAgent{
				tools: make(map[string]tools.Tool),
			}

			// Register the mock tools
			for _, tool := range tt.registeredTools {
				agent.tools[tool.Name()] = tool
			}

			// Get tools list
			toolsList := agent.getToolsList()

			// Verify correct tools are returned
			if len(toolsList) != len(tt.expectedNames) {
				t.Errorf("expected %d tools, got %d", len(tt.expectedNames), len(toolsList))
			}

			// Build map of returned tool names for easy lookup
			returnedNames := make(map[string]bool)
			for _, tool := range toolsList {
				returnedNames[tool.Name()] = true
			}

			// Verify all expected tools are present
			for _, name := range tt.expectedNames {
				if !returnedNames[name] {
					t.Errorf("expected tool %s not found in returned list", name)
				}
			}

			// Verify no unexpected tools are present
			if len(returnedNames) != len(tt.expectedNames) {
				t.Error("returned tools list contains unexpected tools")
			}
		})
	}
}
