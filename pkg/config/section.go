package config

// Section represents a configurable section in the settings system.
// Each section manages its own configuration data and validation logic.
type Section interface {
	// ID returns a unique identifier for this section
	ID() string

	// Title returns the display title for this section
	Title() string

	// Description returns a brief description of what this section configures
	Description() string

	// Data returns the current configuration data for this section
	// The data should be JSON-serializable
	Data() map[string]any

	// SetData updates the section's configuration from the provided data
	// Returns an error if the data is invalid
	SetData(data map[string]any) error

	// Validate validates the current configuration
	// Returns an error if the configuration is invalid
	Validate() error

	// Reset resets the section to its default configuration
	Reset()
}
