package config

import (
	"fmt"
	"sync"
	"time"
)

const (
	// SectionIDUI is the identifier for the UI settings section
	SectionIDUI = "ui"

	// Default values for UI settings
	defaultAutoCloseCommandOverlay = false
	defaultKeepOpenOnError         = false
	defaultAutoCloseDelay          = 1 * time.Second
)

// UISection manages user interface configuration settings.
type UISection struct {
	AutoCloseCommandOverlay bool          `json:"auto_close_command_overlay"`
	KeepOpenOnError         bool          `json:"keep_open_on_error"`
	AutoCloseDelay          time.Duration `json:"auto_close_delay"`
	mu                      sync.RWMutex
}

// NewUISection creates a new UI section with default settings.
func NewUISection() *UISection {
	return &UISection{
		AutoCloseCommandOverlay: defaultAutoCloseCommandOverlay,
		KeepOpenOnError:         defaultKeepOpenOnError,
		AutoCloseDelay:          defaultAutoCloseDelay,
	}
}

// ID returns the section identifier.
func (s *UISection) ID() string {
	return SectionIDUI
}

// Title returns the section title.
func (s *UISection) Title() string {
	return "UI Settings"
}

// Description returns the section description.
func (s *UISection) Description() string {
	return "Configure user interface behavior including command overlay auto-close settings."
}

// Data returns the current configuration data.
func (s *UISection) Data() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]any{
		"auto_close_command_overlay": s.AutoCloseCommandOverlay,
		"keep_open_on_error":         s.KeepOpenOnError,
		"auto_close_delay":           s.AutoCloseDelay.String(),
	}
}

// SetData updates the configuration from the provided data.
func (s *UISection) SetData(data map[string]any) error {
	if data == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for key, value := range data {
		switch key {
		case "auto_close_command_overlay":
			if enabled, ok := value.(bool); ok {
				s.AutoCloseCommandOverlay = enabled
			} else {
				return fmt.Errorf("invalid value type for auto_close_command_overlay: expected bool, got %T", value)
			}

		case "keep_open_on_error":
			if enabled, ok := value.(bool); ok {
				s.KeepOpenOnError = enabled
			} else {
				return fmt.Errorf("invalid value type for keep_open_on_error: expected bool, got %T", value)
			}

		case "auto_close_delay":
			// Handle both string and numeric duration values
			switch v := value.(type) {
			case string:
				duration, err := time.ParseDuration(v)
				if err != nil {
					return fmt.Errorf("invalid duration string for auto_close_delay: %w", err)
				}
				s.AutoCloseDelay = duration
			case float64:
				// JSON numbers come as float64
				s.AutoCloseDelay = time.Duration(v)
			case int64:
				s.AutoCloseDelay = time.Duration(v)
			default:
				return fmt.Errorf("invalid value type for auto_close_delay: expected string or number, got %T", value)
			}

		default:
			// Ignore unknown keys for forward compatibility
			continue
		}
	}

	return nil
}

// Validate validates the current configuration.
func (s *UISection) Validate() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Validate auto-close delay is reasonable (between 100ms and 10s)
	if s.AutoCloseDelay < 100*time.Millisecond || s.AutoCloseDelay > 10*time.Second {
		return fmt.Errorf("auto_close_delay must be between 100ms and 10s, got %v", s.AutoCloseDelay)
	}

	return nil
}

// Reset resets the section to default configuration.
func (s *UISection) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.AutoCloseCommandOverlay = defaultAutoCloseCommandOverlay
	s.KeepOpenOnError = defaultKeepOpenOnError
	s.AutoCloseDelay = defaultAutoCloseDelay
}

// GetAutoCloseSettings returns the current auto-close configuration.
// Returns (enabled, keepOpenOnError, delay).
func (s *UISection) GetAutoCloseSettings() (bool, bool, time.Duration) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.AutoCloseCommandOverlay, s.KeepOpenOnError, s.AutoCloseDelay
}

// SetAutoCloseCommandOverlay sets whether command overlays should auto-close.
func (s *UISection) SetAutoCloseCommandOverlay(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AutoCloseCommandOverlay = enabled
}

// SetKeepOpenOnError sets whether to keep overlays open on error.
func (s *UISection) SetKeepOpenOnError(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.KeepOpenOnError = enabled
}

// SetAutoCloseDelay sets the delay before auto-closing overlays.
func (s *UISection) SetAutoCloseDelay(delay time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AutoCloseDelay = delay
}

// ShouldAutoClose determines if an overlay should auto-close based on exit code.
// Returns true if the overlay should close automatically.
func (s *UISection) ShouldAutoClose(exitCode int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// If auto-close is disabled, never auto-close
	if !s.AutoCloseCommandOverlay {
		return false
	}

	// If command succeeded (exit code 0), always auto-close
	if exitCode == 0 {
		return true
	}

	// Command failed - check keep_open_on_error setting
	// If keep_open_on_error is true, don't auto-close errors
	// If keep_open_on_error is false, auto-close errors too
	return !s.KeepOpenOnError
}
