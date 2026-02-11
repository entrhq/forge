package config

import (
	"testing"
	"time"
)

func TestUISection_DefaultValues(t *testing.T) {
	ui := NewUISection()

	autoClose, keepOpen, delay := ui.GetAutoCloseSettings()
	if autoClose {
		t.Error("Expected auto-close to be disabled by default")
	}
	if keepOpen {
		t.Error("Expected keepOpen to be false by default")
	}
	if delay != 1*time.Second {
		t.Errorf("Expected default delay of 1s, got %v", delay)
	}
}

func TestUISection_SetAutoCloseSettings(t *testing.T) {
	ui := NewUISection()

	// Test enabling auto-close for all commands
	ui.SetAutoCloseCommandOverlay(true)
	ui.SetKeepOpenOnError(false)
	ui.SetAutoCloseDelay(3 * time.Second)
	autoClose, keepOpen, delay := ui.GetAutoCloseSettings()

	if !autoClose {
		t.Error("Expected auto-close to be enabled")
	}
	if keepOpen {
		t.Error("Expected keepOpen to be false")
	}
	if delay != 3*time.Second {
		t.Errorf("Expected delay of 3s, got %v", delay)
	}

	// Test enabling auto-close only for successful commands
	ui.SetAutoCloseCommandOverlay(true)
	ui.SetKeepOpenOnError(true)
	ui.SetAutoCloseDelay(1 * time.Second)
	autoClose, keepOpen, delay = ui.GetAutoCloseSettings()

	if !autoClose {
		t.Error("Expected auto-close to be enabled")
	}
	if !keepOpen {
		t.Error("Expected keepOpen to be true")
	}
	if delay != 1*time.Second {
		t.Errorf("Expected delay of 1s, got %v", delay)
	}
}

func TestUISection_ShouldAutoClose(t *testing.T) {
	ui := NewUISection()

	tests := []struct {
		name         string
		autoClose    bool
		onlySuccess  bool
		exitCode     int
		shouldClose  bool
		description  string
	}{
		{
			name:        "disabled - success",
			autoClose:   false,
			onlySuccess: false,
			exitCode:    0,
			shouldClose: false,
			description: "auto-close disabled, should not close even on success",
		},
		{
			name:        "disabled - failure",
			autoClose:   false,
			onlySuccess: false,
			exitCode:    1,
			shouldClose: false,
			description: "auto-close disabled, should not close on failure",
		},
		{
			name:        "enabled all - success",
			autoClose:   true,
			onlySuccess: false,
			exitCode:    0,
			shouldClose: true,
			description: "auto-close all enabled, should close on success",
		},
		{
			name:        "enabled all - failure",
			autoClose:   true,
			onlySuccess: false,
			exitCode:    1,
			shouldClose: true,
			description: "auto-close all enabled, should close on failure",
		},
		{
			name:        "enabled success only - success",
			autoClose:   true,
			onlySuccess: true,
			exitCode:    0,
			shouldClose: true,
			description: "auto-close success only, should close on success",
		},
		{
			name:        "enabled success only - failure",
			autoClose:   true,
			onlySuccess: true,
			exitCode:    1,
			shouldClose: false,
			description: "auto-close success only, should not close on failure",
		},
		{
			name:        "enabled success only - non-zero success",
			autoClose:   true,
			onlySuccess: true,
			exitCode:    2,
			shouldClose: false,
			description: "auto-close success only, should not close on non-zero exit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui.SetAutoCloseCommandOverlay(tt.autoClose)
			ui.SetKeepOpenOnError(tt.onlySuccess)
			ui.SetAutoCloseDelay(2 * time.Second)
			result := ui.ShouldAutoClose(tt.exitCode)

			if result != tt.shouldClose {
				t.Errorf("%s: expected ShouldAutoClose(%d) = %v, got %v",
					tt.description, tt.exitCode, tt.shouldClose, result)
			}
		})
	}
}

func TestUISection_SetAutoCloseDelay_Validation(t *testing.T) {
	tests := []struct {
		name          string
		delay         time.Duration
		expectedDelay time.Duration
	}{
		{
			name:          "valid delay - 1 second",
			delay:         1 * time.Second,
			expectedDelay: 1 * time.Second,
		},
		{
			name:          "valid delay - 5 seconds",
			delay:         5 * time.Second,
			expectedDelay: 5 * time.Second,
		},
		{
			name:          "valid delay - 100ms minimum",
			delay:         100 * time.Millisecond,
			expectedDelay: 100 * time.Millisecond,
		},
		{
			name:          "valid delay - 10s maximum",
			delay:         10 * time.Second,
			expectedDelay: 10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := NewUISection()
			ui.SetAutoCloseDelay(tt.delay)

			_, _, actualDelay := ui.GetAutoCloseSettings()
			if actualDelay != tt.expectedDelay {
				t.Errorf("Expected delay %v, got %v", tt.expectedDelay, actualDelay)
			}

			// Verify validation accepts this value
			if err := ui.Validate(); err != nil {
				t.Errorf("Validation failed for valid delay %v: %v", tt.delay, err)
			}
		})
	}

	// Test invalid delays through validation
	invalidTests := []struct {
		name  string
		delay time.Duration
	}{
		{
			name:  "invalid delay - too short",
			delay: 50 * time.Millisecond,
		},
		{
			name:  "invalid delay - too long",
			delay: 15 * time.Second,
		},
		{
			name:  "invalid delay - zero",
			delay: 0,
		},
	}

	for _, tt := range invalidTests {
		t.Run(tt.name, func(t *testing.T) {
			ui := NewUISection()
			ui.SetAutoCloseDelay(tt.delay)

			// Validation should fail
			if err := ui.Validate(); err == nil {
				t.Errorf("Expected validation error for delay %v but got nil", tt.delay)
			}
		})
	}
}

func TestUISection_ThreadSafety(t *testing.T) {
	ui := NewUISection()

	// Run concurrent reads and writes
	done := make(chan bool)
	
	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			ui.SetAutoCloseCommandOverlay(i%2 == 0)
			ui.SetKeepOpenOnError(i%3 == 0)
			ui.SetAutoCloseDelay(time.Duration(i+1) * time.Second)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			ui.GetAutoCloseSettings()
			ui.ShouldAutoClose(i % 3)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// If we get here without a race condition, test passes
}
