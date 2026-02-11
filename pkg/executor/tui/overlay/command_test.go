package overlay

import (
	"testing"
	"time"

	"github.com/entrhq/forge/pkg/config"
	pkgtypes "github.com/entrhq/forge/pkg/types"
)

func init() {
	// Initialize config for tests with temporary directory for hermetic testing
	// Note: This still uses the default file store, but tests should use t.TempDir()
	// in TestMain for proper isolation
	config.Initialize("")
}

func TestCommandExecutionOverlay_MaybeAutoClose(t *testing.T) {
	// Setup: Get UI section
	uiConfig := config.GetUI()
	if uiConfig == nil {
		t.Fatal("Failed to get UI config")
	}

	tests := []struct {
		name          string
		autoClose     bool
		keepOpenOnErr bool
		exitCode      int
		expectCmd     bool
		description   string
	}{
		{
			name:          "auto-close disabled",
			autoClose:     false,
			keepOpenOnErr: false,
			exitCode:      0,
			expectCmd:     false,
			description:   "should not return command when auto-close is disabled",
		},
		{
			name:          "auto-close enabled - success",
			autoClose:     true,
			keepOpenOnErr: false,
			exitCode:      0,
			expectCmd:     true,
			description:   "should return command when auto-close is enabled and command succeeds",
		},
		{
			name:          "auto-close enabled - failure",
			autoClose:     true,
			keepOpenOnErr: false,
			exitCode:      1,
			expectCmd:     true,
			description:   "should return command when auto-close is enabled for all commands",
		},
		{
			name:          "auto-close with keep-open-on-error - success",
			autoClose:     true,
			keepOpenOnErr: true,
			exitCode:      0,
			expectCmd:     true,
			description:   "should return command when auto-close is enabled with keep-open-on-error and command succeeds",
		},
		{
			name:          "auto-close with keep-open-on-error - failure",
			autoClose:     true,
			keepOpenOnErr: true,
			exitCode:      1,
			expectCmd:     false,
			description:   "should not return command when keep-open-on-error is enabled and command fails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Configure UI settings for this test
			uiConfig.SetAutoCloseCommandOverlay(tt.autoClose)
			uiConfig.SetKeepOpenOnError(tt.keepOpenOnErr)
			uiConfig.SetAutoCloseDelay(2 * time.Second)

			// Create overlay with required parameters
			cancelChan := make(chan *pkgtypes.CancellationRequest, 1)
			overlay := NewCommandExecutionOverlay("test command", "/test/dir", "test-id", cancelChan)
			overlay.exitCode = tt.exitCode
			overlay.isRunning = false

			// Call maybeAutoClose
			cmd := overlay.maybeAutoClose()

			// Verify result
			if tt.expectCmd && cmd == nil {
				t.Errorf("%s: expected command but got nil", tt.description)
			}
			if !tt.expectCmd && cmd != nil {
				t.Errorf("%s: expected nil but got command", tt.description)
			}
		})
	}
}

func TestCommandExecutionOverlay_ShowExitCodeToast(t *testing.T) {
	// Setup: Get UI section
	uiConfig := config.GetUI()
	if uiConfig == nil {
		t.Fatal("Failed to get UI config")
	}

	tests := []struct {
		name        string
		autoClose   bool
		exitCode    int
		expectToast bool
		description string
	}{
		{
			name:        "auto-close disabled - no toast",
			autoClose:   false,
			exitCode:    0,
			expectToast: false,
			description: "should not show toast when auto-close is disabled",
		},
		{
			name:        "auto-close enabled - success toast",
			autoClose:   true,
			exitCode:    0,
			expectToast: true,
			description: "should show success toast when auto-close is enabled",
		},
		{
			name:        "auto-close enabled - failure toast",
			autoClose:   true,
			exitCode:    1,
			expectToast: true,
			description: "should show failure toast when auto-close is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Configure UI settings for this test
			uiConfig.SetAutoCloseCommandOverlay(tt.autoClose)
			uiConfig.SetKeepOpenOnError(false)
			uiConfig.SetAutoCloseDelay(2 * time.Second)

			// Create overlay with required parameters
			cancelChan := make(chan *pkgtypes.CancellationRequest, 1)
			overlay := NewCommandExecutionOverlay("test command", "/test/dir", "test-id", cancelChan)

			// Call showExitCodeToast
			cmd := overlay.showExitCodeToast(tt.exitCode, "1.5s")

			// Verify result
			if tt.expectToast && cmd == nil {
				t.Errorf("%s: expected toast command but got nil", tt.description)
			}
			if !tt.expectToast && cmd != nil {
				t.Errorf("%s: expected nil but got toast command", tt.description)
			}
		})
	}
}

func TestCommandExecutionOverlay_HandleAutoClose(t *testing.T) {
	cancelChan := make(chan *pkgtypes.CancellationRequest, 1)
	overlay := NewCommandExecutionOverlay("test command", "/test/dir", "test-id", cancelChan)
	overlay.isRunning = false

	// Test matching execution ID - should close overlay
	msg := autoCloseMsg{executionID: "test-id"}
	result, cmd := overlay.handleAutoClose(msg)

	if result != nil {
		t.Error("Expected overlay to close (return nil) but it didn't")
	}
	if cmd != nil {
		t.Error("Expected no command when closing overlay")
	}

	// Test non-matching execution ID - should not close
	cancelChan2 := make(chan *pkgtypes.CancellationRequest, 1)
	overlay2 := NewCommandExecutionOverlay("test command", "/test/dir", "test-id-2", cancelChan2)
	overlay2.isRunning = false

	msg2 := autoCloseMsg{executionID: "different-id"}
	result2, cmd2 := overlay2.handleAutoClose(msg2)

	if result2 == nil {
		t.Error("Expected overlay to remain open but it closed")
	}
	if cmd2 != nil {
		t.Error("Expected no command when overlay remains open")
	}

	// Test with still running command - should not close
	cancelChan3 := make(chan *pkgtypes.CancellationRequest, 1)
	overlay3 := NewCommandExecutionOverlay("test command", "/test/dir", "test-id-3", cancelChan3)
	overlay3.isRunning = true

	msg3 := autoCloseMsg{executionID: "test-id-3"}
	result3, cmd3 := overlay3.handleAutoClose(msg3)

	if result3 == nil {
		t.Error("Expected overlay to remain open while command is running")
	}
	if cmd3 != nil {
		t.Error("Expected no command when command is still running")
	}
}
