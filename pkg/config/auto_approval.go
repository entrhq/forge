package config

import (
	"fmt"
	"maps"
)

const (
	// SectionIDAutoApproval is the identifier for the auto-approval section
	SectionIDAutoApproval = "auto_approval"
)

// AutoApprovalSection manages auto-approval settings for tools.
type AutoApprovalSection struct {
	// tools maps tool names to their auto-approval status
	tools map[string]bool
}

// NewAutoApprovalSection creates a new auto-approval section with default settings.
// Tools can be dynamically added as they are registered.
func NewAutoApprovalSection() *AutoApprovalSection {
	return &AutoApprovalSection{
		tools: make(map[string]bool),
	}
}

// ID returns the section identifier.
func (s *AutoApprovalSection) ID() string {
	return SectionIDAutoApproval
}

// Title returns the section title.
func (s *AutoApprovalSection) Title() string {
	return "Auto-Approval Settings"
}

// Description returns the section description.
func (s *AutoApprovalSection) Description() string {
	return "Configure which tools are automatically approved without prompts. Note: execute_command always requires approval or whitelist."
}

// Data returns the current configuration data.
func (s *AutoApprovalSection) Data() map[string]any {
	data := make(map[string]any, len(s.tools))
	for tool, enabled := range s.tools {
		data[tool] = enabled
	}
	return data
}

// SetData updates the configuration from the provided data.
func (s *AutoApprovalSection) SetData(data map[string]any) error {
	if data == nil {
		return nil
	}

	for tool, value := range data {
		if enabled, ok := value.(bool); ok {
			s.tools[tool] = enabled
		} else {
			return fmt.Errorf("invalid value type for tool '%s': expected bool, got %T", tool, value)
		}
	}

	return nil
}

// Validate validates the current configuration.
func (s *AutoApprovalSection) Validate() error {
	// Auto-approval settings are always valid (boolean values)
	return nil
}

// Reset resets the section to default configuration (all disabled).
func (s *AutoApprovalSection) Reset() {
	for tool := range s.tools {
		s.tools[tool] = false
	}
}

// EnsureToolExists ensures a tool exists in the map with a default value if not present.
// This allows tools to be registered dynamically.
func (s *AutoApprovalSection) EnsureToolExists(toolName string) {
	if _, exists := s.tools[toolName]; !exists {
		// Default to false (require approval) for new tools
		s.tools[toolName] = false
	}
}

// IsToolAutoApproved returns true if the specified tool is auto-approved.
// Returns false for unknown tools (default is to require approval).
func (s *AutoApprovalSection) IsToolAutoApproved(toolName string) bool {
	enabled, exists := s.tools[toolName]
	if !exists {
		// Unknown tool - ensure it exists with default value (false)
		s.EnsureToolExists(toolName)
		return false
	}
	return enabled
}

// SetToolAutoApproval sets the auto-approval status for a tool.
func (s *AutoApprovalSection) SetToolAutoApproval(toolName string, enabled bool) {
	s.tools[toolName] = enabled
}

// GetTools returns a map of all tool names to their auto-approval status.
func (s *AutoApprovalSection) GetTools() map[string]bool {
	// Return a copy to prevent external modification
	copy := make(map[string]bool, len(s.tools))
	maps.Copy(copy, s.tools)
	return copy
}
