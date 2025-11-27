package config

import (
	"fmt"
	"strings"
)

const (
	// MatchTypePrefix indicates a prefix-based pattern match
	MatchTypePrefix = "prefix"
	// MatchTypeExact indicates an exact pattern match
	MatchTypeExact = "exact"
	// SectionIDCommandWhitelist is the identifier for the command whitelist section
	SectionIDCommandWhitelist = "command_whitelist"
)

// WhitelistPattern represents a command pattern that can be auto-approved.
type WhitelistPattern struct {
	Pattern     string `json:"pattern"`
	Description string `json:"description"`
	Type        string `json:"type"` // "prefix" or "exact"
}

// CommandWhitelistSection manages the whitelist of commands that can be auto-approved.
type CommandWhitelistSection struct {
	patterns []WhitelistPattern
}

// NewCommandWhitelistSection creates a new command whitelist section.
func NewCommandWhitelistSection() *CommandWhitelistSection {
	return &CommandWhitelistSection{
		patterns: []WhitelistPattern{
			{
				Pattern:     "git status",
				Description: "Git status and variations",
				Type:        MatchTypePrefix,
			},
			{
				Pattern:     "ls",
				Description: "List directory",
				Type:        MatchTypePrefix,
			},
			{
				Pattern:     "pwd",
				Description: "Get current directory",
				Type:        MatchTypeExact,
			},
		},
	}
}

// ID returns the section identifier.
func (s *CommandWhitelistSection) ID() string {
	return SectionIDCommandWhitelist
}

// Title returns the section title.
func (s *CommandWhitelistSection) Title() string {
	return "Command Whitelist"
}

// Description returns the section description.
func (s *CommandWhitelistSection) Description() string {
	return "Commands matching these patterns will auto-approve for execute_command tool"
}

// Data returns the current configuration data.
func (s *CommandWhitelistSection) Data() map[string]interface{} {
	// Convert patterns to interface{} slice
	patternsData := make([]interface{}, len(s.patterns))
	for i, p := range s.patterns {
		patternsData[i] = map[string]interface{}{
			"pattern":     p.Pattern,
			"description": p.Description,
			"type":        p.Type,
		}
	}

	return map[string]interface{}{
		"patterns": patternsData,
	}
}

// SetData updates the configuration from the provided data.
func (s *CommandWhitelistSection) SetData(data map[string]interface{}) error {
	if data == nil {
		return nil
	}

	patternsData, ok := data["patterns"]
	if !ok {
		return nil // No patterns key, keep defaults
	}

	patternsSlice, ok := patternsData.([]interface{})
	if !ok {
		return fmt.Errorf("invalid patterns type: expected []interface{}, got %T", patternsData)
	}

	patterns := make([]WhitelistPattern, 0, len(patternsSlice))
	for i, item := range patternsSlice {
		patternMap, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid pattern at index %d: expected map, got %T", i, item)
		}

		pattern, ok := patternMap["pattern"].(string)
		if !ok {
			return fmt.Errorf("invalid pattern at index %d: missing or invalid pattern field", i)
		}

		descriptionVal, hasDescription := patternMap["description"]
		description, ok := descriptionVal.(string)
		if hasDescription && !ok {
			return fmt.Errorf("invalid pattern at index %d: description field is not a string (got %T)", i, descriptionVal)
		}
		if !hasDescription {
			description = ""
		}

		// Get type field, default to "prefix" if not specified
		patternType := MatchTypePrefix
		if typeVal, hasType := patternMap["type"]; hasType {
			if typeStr, ok := typeVal.(string); ok {
				if typeStr == MatchTypeExact || typeStr == MatchTypePrefix {
					patternType = typeStr
				}
			}
		}

		patterns = append(patterns, WhitelistPattern{
			Pattern:     pattern,
			Description: description,
			Type:        patternType,
		})
	}

	s.patterns = patterns
	return nil
}

// Validate validates the current configuration.
func (s *CommandWhitelistSection) Validate() error {
	for i, pattern := range s.patterns {
		if strings.TrimSpace(pattern.Pattern) == "" {
			return fmt.Errorf("pattern at index %d is empty", i)
		}
	}
	return nil
}

// Reset resets the section to default configuration.
func (s *CommandWhitelistSection) Reset() {
	s.patterns = []WhitelistPattern{
		{
			Pattern:     "npm",
			Description: "All npm commands",
		},
		{
			Pattern:     "git status",
			Description: "Git status and variations",
		},
		{
			Pattern:     "ls",
			Description: "List directory",
		},
	}
}

// IsCommandWhitelisted checks if a command matches any whitelist pattern.
//
// Pattern matching rules:
//   - Type "exact": Command must exactly match the pattern
//   - Type "prefix": Command must start with the pattern (followed by space or end)
//
// Examples for prefix type:
//   - Pattern "npm" matches: "npm", "npm install", "npm run build"
//   - Pattern "npm install" matches: "npm install", "npm install express"
//   - Pattern "git status" matches: "git status", "git status --short"
//
// Examples for exact type:
//   - Pattern "ls" matches only: "ls" (not "ls -la")
//   - Pattern "git status" matches only: "git status" (not "git status --short")
func (s *CommandWhitelistSection) IsCommandWhitelisted(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}

	for _, pattern := range s.patterns {
		if matchesPattern(command, pattern.Pattern, pattern.Type) {
			return true
		}
	}

	return false
}

// matchesPattern checks if a command matches a pattern based on the match type.
func matchesPattern(command, pattern, matchType string) bool {
	pattern = strings.TrimSpace(pattern)
	command = strings.TrimSpace(command)

	// Exact match (works for both types)
	if command == pattern {
		return true
	}

	// For exact type, only exact match is allowed
	if matchType == "exact" {
		return false
	}

	// For prefix type (or unspecified), check if command starts with pattern followed by space
	// This ensures "npm install" matches "npm install express" but not "npminstall"
	if strings.HasPrefix(command, pattern+" ") {
		return true
	}

	// For single-word patterns, check if command starts with pattern
	// This allows "npm" to match "npm install" but requires a space boundary
	if !strings.Contains(pattern, " ") {
		// Single word pattern
		if strings.HasPrefix(command, pattern) {
			// Check if it's followed by space or is exact match
			if len(command) == len(pattern) || command[len(pattern)] == ' ' {
				return true
			}
		}
	}

	return false
}

// AddPattern adds a new pattern to the whitelist.
// Defaults to "prefix" type for backward compatibility.
func (s *CommandWhitelistSection) AddPattern(pattern, description string) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	// Check for duplicates
	for _, p := range s.patterns {
		if p.Pattern == pattern {
			return fmt.Errorf("pattern '%s' already exists", pattern)
		}
	}

	s.patterns = append(s.patterns, WhitelistPattern{
		Pattern:     pattern,
		Description: description,
		Type:        "prefix", // Default to prefix for consistency with SetData
	})

	return nil
}

// RemovePattern removes a pattern from the whitelist by index.
func (s *CommandWhitelistSection) RemovePattern(index int) error {
	if index < 0 || index >= len(s.patterns) {
		return fmt.Errorf("invalid pattern index: %d", index)
	}

	s.patterns = append(s.patterns[:index], s.patterns[index+1:]...)
	return nil
}

// GetPatterns returns a copy of all patterns.
func (s *CommandWhitelistSection) GetPatterns() []WhitelistPattern {
	copy := make([]WhitelistPattern, len(s.patterns))
	for i, p := range s.patterns {
		copy[i] = WhitelistPattern{
			Pattern:     p.Pattern,
			Description: p.Description,
			Type:        p.Type,
		}
	}
	return copy
}

// UpdatePattern updates a pattern at the specified index.
// Preserves the existing Type field.
func (s *CommandWhitelistSection) UpdatePattern(index int, pattern, description string) error {
	if index < 0 || index >= len(s.patterns) {
		return fmt.Errorf("invalid pattern index: %d", index)
	}

	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	// Check for duplicates (excluding current index)
	for i, p := range s.patterns {
		if i != index && p.Pattern == pattern {
			return fmt.Errorf("pattern '%s' already exists", pattern)
		}
	}

	s.patterns[index] = WhitelistPattern{
		Pattern:     pattern,
		Description: description,
		Type:        s.patterns[index].Type, // Preserve existing type
	}

	return nil
}
