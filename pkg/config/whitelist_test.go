package config

import (
	"testing"
)

func TestNewCommandWhitelistSection(t *testing.T) {
	section := NewCommandWhitelistSection()

	if section == nil {
		t.Fatal("NewCommandWhitelistSection returned nil")
	}

	if section.ID() != "command_whitelist" {
		t.Errorf("Expected ID 'command_whitelist', got '%s'", section.ID())
	}

	if section.Title() == "" {
		t.Error("Title should not be empty")
	}

	if section.Description() == "" {
		t.Error("Description should not be empty")
	}

	// Should have default patterns
	patterns := section.GetPatterns()
	if len(patterns) == 0 {
		t.Error("New section should have default patterns")
	}
}

func TestCommandWhitelistSection_Data(t *testing.T) {
	t.Run("returns pattern data", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		data := section.Data()

		patternsData, ok := data["patterns"]
		if !ok {
			t.Fatal("Data should contain 'patterns' key")
		}

		patterns, ok := patternsData.([]interface{})
		if !ok {
			t.Fatal("Patterns should be a slice")
		}

		if len(patterns) == 0 {
			t.Error("Should have default patterns")
		}
	})

	t.Run("includes all pattern fields", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		section.AddPattern("test command", "Test description")

		data := section.Data()
		patterns := data["patterns"].([]interface{})

		// Find our test pattern
		found := false
		for _, p := range patterns {
			patternMap := p.(map[string]interface{})
			if patternMap["pattern"] == "test command" {
				found = true
				if patternMap["description"] != "Test description" {
					t.Error("Description not included in data")
				}
				if patternMap["type"] == "" {
					t.Error("Type not included in data")
				}
			}
		}

		if !found {
			t.Error("Added pattern not found in data")
		}
	})
}

func TestCommandWhitelistSection_SetData(t *testing.T) {
	t.Run("sets patterns from data", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		data := map[string]interface{}{
			"patterns": []interface{}{
				map[string]interface{}{
					"pattern":     "npm",
					"description": "NPM commands",
					"type":        "prefix",
				},
				map[string]interface{}{
					"pattern":     "git status",
					"description": "Git status",
					"type":        "exact",
				},
			},
		}

		err := section.SetData(data)
		if err != nil {
			t.Fatalf("SetData failed: %v", err)
		}

		patterns := section.GetPatterns()
		if len(patterns) != 2 {
			t.Errorf("Expected 2 patterns, got %d", len(patterns))
		}
	})

	t.Run("handles nil data", func(t *testing.T) {
		section := NewCommandWhitelistSection()

		err := section.SetData(nil)
		if err != nil {
			t.Errorf("SetData should handle nil gracefully: %v", err)
		}
	})

	t.Run("handles missing patterns key", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		data := map[string]interface{}{
			"other_key": "value",
		}

		err := section.SetData(data)
		if err != nil {
			t.Errorf("SetData should handle missing patterns key: %v", err)
		}
	})

	t.Run("rejects invalid patterns type", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		data := map[string]interface{}{
			"patterns": "not a slice",
		}

		err := section.SetData(data)
		if err == nil {
			t.Error("Expected error for invalid patterns type")
		}
	})

	t.Run("rejects invalid pattern item type", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		data := map[string]interface{}{
			"patterns": []interface{}{
				"not a map",
			},
		}

		err := section.SetData(data)
		if err == nil {
			t.Error("Expected error for invalid pattern item")
		}
	})

	t.Run("rejects missing pattern field", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		data := map[string]interface{}{
			"patterns": []interface{}{
				map[string]interface{}{
					"description": "Missing pattern",
				},
			},
		}

		err := section.SetData(data)
		if err == nil {
			t.Error("Expected error for missing pattern field")
		}
	})

	t.Run("defaults to prefix type", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		data := map[string]interface{}{
			"patterns": []interface{}{
				map[string]interface{}{
					"pattern":     "npm",
					"description": "NPM",
				},
			},
		}

		section.SetData(data)
		patterns := section.GetPatterns()

		if patterns[0].Type != "prefix" {
			t.Error("Should default to prefix type")
		}
	})

	t.Run("handles optional description", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		data := map[string]interface{}{
			"patterns": []interface{}{
				map[string]interface{}{
					"pattern": "npm",
				},
			},
		}

		err := section.SetData(data)
		if err != nil {
			t.Errorf("SetData should handle missing description: %v", err)
		}

		patterns := section.GetPatterns()
		if patterns[0].Description != "" {
			t.Error("Missing description should be empty string")
		}
	})
}

func TestCommandWhitelistSection_Validate(t *testing.T) {
	t.Run("validates non-empty patterns", func(t *testing.T) {
		section := NewCommandWhitelistSection()

		err := section.Validate()
		if err != nil {
			t.Errorf("Validation failed for valid patterns: %v", err)
		}
	})

	t.Run("rejects empty pattern", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		// Directly set patterns to bypass AddPattern validation
		section.patterns = []WhitelistPattern{
			{Pattern: "  ", Description: "Empty pattern", Type: "prefix"},
		}

		err := section.Validate()
		if err == nil {
			t.Error("Expected error for empty pattern")
		}
	})

	t.Run("validates after Reset", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		section.Reset()

		err := section.Validate()
		if err != nil {
			t.Errorf("Validation failed after Reset: %v", err)
		}
	})
}

func TestCommandWhitelistSection_Reset(t *testing.T) {
	t.Run("resets to default patterns", func(t *testing.T) {
		section := NewCommandWhitelistSection()

		// Get initial default count
		initialPatterns := section.GetPatterns()
		initialCount := len(initialPatterns)

		// Modify patterns
		section.AddPattern("custom command", "Custom")

		// Reset
		section.Reset()

		// Should have default patterns again
		patterns := section.GetPatterns()
		if len(patterns) != initialCount {
			t.Errorf("Expected %d patterns after reset, got %d", initialCount, len(patterns))
		}

		// Should not have custom pattern
		for _, p := range patterns {
			if p.Pattern == "custom command" {
				t.Error("Custom pattern should be removed after reset")
			}
		}
	})
}

func TestCommandWhitelistSection_IsCommandWhitelisted(t *testing.T) {
	tests := []struct {
		name        string
		pattern     WhitelistPattern
		command     string
		shouldMatch bool
	}{
		// Exact match type
		{"exact match - matches exactly", WhitelistPattern{Pattern: "pwd", Type: "exact"}, "pwd", true},
		{"exact match - doesn't match with args", WhitelistPattern{Pattern: "pwd", Type: "exact"}, "pwd -L", false},
		{"exact match - doesn't match prefix", WhitelistPattern{Pattern: "git status", Type: "exact"}, "git status --short", false},

		// Prefix match type
		{"prefix - matches exact", WhitelistPattern{Pattern: "npm", Type: "prefix"}, "npm", true},
		{"prefix - matches with args", WhitelistPattern{Pattern: "npm", Type: "prefix"}, "npm install", true},
		{"prefix - matches multi-word with args", WhitelistPattern{Pattern: "git status", Type: "prefix"}, "git status --short", true},
		{"prefix - doesn't match partial word", WhitelistPattern{Pattern: "npm", Type: "prefix"}, "npminstall", false},
		{"prefix - matches with space", WhitelistPattern{Pattern: "ls", Type: "prefix"}, "ls -la", true},

		// Whitespace handling
		{"handles leading whitespace", WhitelistPattern{Pattern: "npm", Type: "prefix"}, "  npm install", true},
		{"handles trailing whitespace", WhitelistPattern{Pattern: "npm", Type: "prefix"}, "npm install  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := NewCommandWhitelistSection()
			// Clear default patterns
			for i := len(section.GetPatterns()) - 1; i >= 0; i-- {
				section.RemovePattern(i)
			}
			// Add test pattern
			section.patterns = []WhitelistPattern{tt.pattern}

			result := section.IsCommandWhitelisted(tt.command)
			if result != tt.shouldMatch {
				t.Errorf("Pattern '%s' (type: %s) vs command '%s': expected %v, got %v",
					tt.pattern.Pattern, tt.pattern.Type, tt.command, tt.shouldMatch, result)
			}
		})
	}

	t.Run("returns false for empty command", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		if section.IsCommandWhitelisted("") {
			t.Error("Empty command should not be whitelisted")
		}
		if section.IsCommandWhitelisted("   ") {
			t.Error("Whitespace-only command should not be whitelisted")
		}
	})

	t.Run("checks all patterns", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		// Clear defaults
		for i := len(section.GetPatterns()) - 1; i >= 0; i-- {
			section.RemovePattern(i)
		}

		section.AddPattern("npm", "NPM")
		section.AddPattern("git", "Git")
		section.AddPattern("ls", "List")

		if !section.IsCommandWhitelisted("npm install") {
			t.Error("Should match first pattern")
		}
		if !section.IsCommandWhitelisted("git status") {
			t.Error("Should match second pattern")
		}
		if !section.IsCommandWhitelisted("ls -la") {
			t.Error("Should match third pattern")
		}
		if section.IsCommandWhitelisted("unknown command") {
			t.Error("Should not match unknown command")
		}
	})
}

func TestCommandWhitelistSection_AddPattern(t *testing.T) {
	t.Run("adds pattern successfully", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		initialCount := len(section.GetPatterns())

		err := section.AddPattern("new command", "New description")
		if err != nil {
			t.Fatalf("AddPattern failed: %v", err)
		}

		patterns := section.GetPatterns()
		if len(patterns) != initialCount+1 {
			t.Error("Pattern was not added")
		}

		// Verify pattern was added correctly
		found := false
		for _, p := range patterns {
			if p.Pattern == "new command" && p.Description == "New description" {
				found = true
				if p.Type != "prefix" {
					t.Error("Should default to prefix type")
				}
			}
		}
		if !found {
			t.Error("Pattern not found after adding")
		}
	})

	t.Run("rejects empty pattern", func(t *testing.T) {
		section := NewCommandWhitelistSection()

		err := section.AddPattern("", "Description")
		if err == nil {
			t.Error("Expected error for empty pattern")
		}

		err = section.AddPattern("   ", "Description")
		if err == nil {
			t.Error("Expected error for whitespace-only pattern")
		}
	})

	t.Run("rejects duplicate pattern", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		section.AddPattern("test command", "First")

		err := section.AddPattern("test command", "Duplicate")
		if err == nil {
			t.Error("Expected error for duplicate pattern")
		}
	})

	t.Run("trims whitespace from pattern", func(t *testing.T) {
		section := NewCommandWhitelistSection()

		err := section.AddPattern("  trimmed  ", "Description")
		if err != nil {
			t.Fatalf("AddPattern failed: %v", err)
		}

		patterns := section.GetPatterns()
		found := false
		for _, p := range patterns {
			if p.Pattern == "trimmed" {
				found = true
			}
		}
		if !found {
			t.Error("Pattern was not trimmed")
		}
	})
}

func TestCommandWhitelistSection_RemovePattern(t *testing.T) {
	t.Run("removes pattern by index", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		// Clear defaults
		for i := len(section.GetPatterns()) - 1; i >= 0; i-- {
			section.RemovePattern(i)
		}

		section.AddPattern("pattern1", "First")
		section.AddPattern("pattern2", "Second")
		section.AddPattern("pattern3", "Third")

		err := section.RemovePattern(1)
		if err != nil {
			t.Fatalf("RemovePattern failed: %v", err)
		}

		patterns := section.GetPatterns()
		if len(patterns) != 2 {
			t.Errorf("Expected 2 patterns, got %d", len(patterns))
		}

		// Should have removed pattern2
		for _, p := range patterns {
			if p.Pattern == "pattern2" {
				t.Error("Pattern2 should have been removed")
			}
		}
	})

	t.Run("rejects invalid index", func(t *testing.T) {
		section := NewCommandWhitelistSection()

		err := section.RemovePattern(-1)
		if err == nil {
			t.Error("Expected error for negative index")
		}

		err = section.RemovePattern(999)
		if err == nil {
			t.Error("Expected error for out-of-bounds index")
		}
	})
}

func TestCommandWhitelistSection_GetPatterns(t *testing.T) {
	t.Run("returns all patterns", func(t *testing.T) {
		section := NewCommandWhitelistSection()

		patterns := section.GetPatterns()
		if len(patterns) == 0 {
			t.Error("Should have default patterns")
		}
	})

	t.Run("returns copy to prevent external modification", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		section.AddPattern("test", "Test")

		patterns := section.GetPatterns()
		originalLen := len(patterns)

		// Modify returned slice
		patterns[0].Pattern = "modified"
		_ = append(patterns, WhitelistPattern{Pattern: "new", Description: "New"})

		// Original should not be affected
		newPatterns := section.GetPatterns()
		if len(newPatterns) != originalLen {
			t.Error("External modification affected section")
		}

		for _, p := range newPatterns {
			if p.Pattern == "modified" || p.Pattern == "new" {
				t.Error("External modification affected pattern data")
			}
		}
	})
}

func TestCommandWhitelistSection_UpdatePattern(t *testing.T) {
	t.Run("updates pattern successfully", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		section.AddPattern("old pattern", "Old description")

		// Find the index
		patterns := section.GetPatterns()
		index := -1
		for i, p := range patterns {
			if p.Pattern == "old pattern" {
				index = i
				break
			}
		}

		err := section.UpdatePattern(index, "new pattern", "New description")
		if err != nil {
			t.Fatalf("UpdatePattern failed: %v", err)
		}

		patterns = section.GetPatterns()
		if patterns[index].Pattern != "new pattern" {
			t.Error("Pattern was not updated")
		}
		if patterns[index].Description != "New description" {
			t.Error("Description was not updated")
		}
	})

	t.Run("preserves type field", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		// Manually set a pattern with exact type
		section.patterns = []WhitelistPattern{
			{Pattern: "test", Description: "Test", Type: "exact"},
		}

		err := section.UpdatePattern(0, "updated", "Updated")
		if err != nil {
			t.Fatalf("UpdatePattern failed: %v", err)
		}

		patterns := section.GetPatterns()
		if patterns[0].Type != "exact" {
			t.Error("Type field was not preserved")
		}
	})

	t.Run("rejects invalid index", func(t *testing.T) {
		section := NewCommandWhitelistSection()

		err := section.UpdatePattern(-1, "pattern", "Description")
		if err == nil {
			t.Error("Expected error for negative index")
		}

		err = section.UpdatePattern(999, "pattern", "Description")
		if err == nil {
			t.Error("Expected error for out-of-bounds index")
		}
	})

	t.Run("rejects empty pattern", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		section.AddPattern("test", "Test")

		err := section.UpdatePattern(0, "", "Empty")
		if err == nil {
			t.Error("Expected error for empty pattern")
		}

		err = section.UpdatePattern(0, "   ", "Whitespace")
		if err == nil {
			t.Error("Expected error for whitespace-only pattern")
		}
	})

	t.Run("rejects duplicate pattern", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		// Clear defaults
		for i := len(section.GetPatterns()) - 1; i >= 0; i-- {
			section.RemovePattern(i)
		}

		section.AddPattern("pattern1", "First")
		section.AddPattern("pattern2", "Second")

		err := section.UpdatePattern(0, "pattern2", "Duplicate")
		if err == nil {
			t.Error("Expected error for duplicate pattern")
		}
	})

	t.Run("allows updating to same pattern", func(t *testing.T) {
		section := NewCommandWhitelistSection()
		section.AddPattern("test", "Test")

		patterns := section.GetPatterns()
		index := -1
		for i, p := range patterns {
			if p.Pattern == "test" {
				index = i
				break
			}
		}

		err := section.UpdatePattern(index, "test", "Updated description")
		if err != nil {
			t.Errorf("Should allow updating to same pattern: %v", err)
		}
	})
}
