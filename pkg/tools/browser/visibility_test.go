package browser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/entrhq/forge/pkg/config"
)

func TestSessionManagementToolsConditionalVisibility(t *testing.T) {
	// Initialize config system with temp directory
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.json")
	if err := config.Initialize(configPath); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		browserEnabled bool
		expectedShow   bool
	}{
		{
			name:           "tools visible when browser enabled",
			browserEnabled: true,
			expectedShow:   true,
		},
		{
			name:           "tools hidden when browser disabled",
			browserEnabled: false,
			expectedShow:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set browser enabled state
			ui := config.GetUI()
			ui.SetBrowserEnabled(tt.browserEnabled)

			// Create session manager and tools
			manager := NewSessionManager()
			startTool := NewStartSessionTool(manager)
			listTool := NewListSessionsTool(manager)
			closeTool := NewCloseSessionTool(manager)

			// Check ShouldShow for all session management tools
			tools := []interface {
				ShouldShow() bool
			}{
				startTool,
				listTool,
				closeTool,
			}

			for _, tool := range tools {
				got := tool.ShouldShow()
				if got != tt.expectedShow {
					t.Errorf("ShouldShow() = %v, want %v for browser_enabled=%v",
						got, tt.expectedShow, tt.browserEnabled)
				}
			}
		})
	}
}

func TestSessionManagementToolsVisibilityWithoutConfig(t *testing.T) {
	// Don't initialize config - simulate uninitialized state
	manager := NewSessionManager()
	startTool := NewStartSessionTool(manager)
	listTool := NewListSessionsTool(manager)
	closeTool := NewCloseSessionTool(manager)

	// Tools should be hidden when config is not initialized
	if startTool.ShouldShow() {
		t.Error("start_browser_session should be hidden when config not initialized")
	}
	if listTool.ShouldShow() {
		t.Error("list_browser_sessions should be hidden when config not initialized")
	}
	if closeTool.ShouldShow() {
		t.Error("close_browser_session should be hidden when config not initialized")
	}
}
