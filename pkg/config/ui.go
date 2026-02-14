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
	defaultBrowserEnabled          = false
	defaultBrowserHeadless         = true
)

// UISection manages user interface configuration settings.
type UISection struct {
	AutoCloseCommandOverlay bool          `json:"auto_close_command_overlay"`
	KeepOpenOnError         bool          `json:"keep_open_on_error"`
	AutoCloseDelay          time.Duration `json:"auto_close_delay"`
	BrowserEnabled          bool          `json:"browser_enabled"`
	BrowserHeadless         bool          `json:"browser_headless"`
	mu                      sync.RWMutex
}

// NewUISection creates a new UI section with default settings.
func NewUISection() *UISection {
	return &UISection{
		AutoCloseCommandOverlay: defaultAutoCloseCommandOverlay,
		KeepOpenOnError:         defaultKeepOpenOnError,
		AutoCloseDelay:          defaultAutoCloseDelay,
		BrowserEnabled:          defaultBrowserEnabled,
		BrowserHeadless:         defaultBrowserHeadless,
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
	return "Configure user interface behavior including command overlay auto-close and browser automation settings."
}

// Data returns the current configuration data.
func (s *UISection) Data() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]any{
		"auto_close_command_overlay": s.AutoCloseCommandOverlay,
		"keep_open_on_error":         s.KeepOpenOnError,
		"auto_close_delay":           s.AutoCloseDelay.String(),
		"browser_enabled":            s.BrowserEnabled,
		"browser_headless":           s.BrowserHeadless,
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
		if err := s.setField(key, value); err != nil {
			return err
		}
	}

	return nil
}

// setField sets a single configuration field. Extracted to reduce complexity.
func (s *UISection) setField(key string, value any) error {
	switch key {
	case "auto_close_command_overlay":
		return s.setBoolField(&s.AutoCloseCommandOverlay, value, key)

	case "keep_open_on_error":
		return s.setBoolField(&s.KeepOpenOnError, value, key)

	case "auto_close_delay":
		return s.setDurationField(&s.AutoCloseDelay, value)

	case "browser_enabled":
		return s.setBoolField(&s.BrowserEnabled, value, key)

	case "browser_headless":
		return s.setBoolField(&s.BrowserHeadless, value, key)

	default:
		// Ignore unknown keys for forward compatibility
		return nil
	}
}

// setBoolField sets a boolean configuration field with type validation.
func (s *UISection) setBoolField(field *bool, value any, fieldName string) error {
	enabled, ok := value.(bool)
	if !ok {
		return fmt.Errorf("invalid value type for %s: expected bool, got %T", fieldName, value)
	}
	*field = enabled
	return nil
}

// setDurationField sets a duration configuration field with type validation.
func (s *UISection) setDurationField(field *time.Duration, value any) error {
	switch v := value.(type) {
	case string:
		duration, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid duration string for auto_close_delay: %w", err)
		}
		*field = duration
	case float64:
		// For backward compatibility, treat JSON numbers as nanoseconds
		*field = time.Duration(v)
	case int64:
		// For backward compatibility, treat as nanoseconds
		*field = time.Duration(v)
	default:
		return fmt.Errorf("invalid value type for auto_close_delay: expected string or number, got %T", value)
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
	s.BrowserEnabled = defaultBrowserEnabled
	s.BrowserHeadless = defaultBrowserHeadless
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

// IsBrowserEnabled returns whether browser automation is enabled.
func (s *UISection) IsBrowserEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.BrowserEnabled
}

// SetBrowserEnabled sets whether browser automation is enabled.
func (s *UISection) SetBrowserEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.BrowserEnabled = enabled
}

// IsBrowserHeadless returns whether browser runs in headless mode by default.
func (s *UISection) IsBrowserHeadless() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.BrowserHeadless
}

// SetBrowserHeadless sets whether browser runs in headless mode by default.
func (s *UISection) SetBrowserHeadless(headless bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.BrowserHeadless = headless
}
