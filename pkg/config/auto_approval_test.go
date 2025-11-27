package config

import (
	"testing"
)

func TestNewAutoApprovalSection(t *testing.T) {
	section := NewAutoApprovalSection()

	if section == nil {
		t.Fatal("NewAutoApprovalSection returned nil")
	}

	if section.ID() != "auto_approval" {
		t.Errorf("Expected ID 'auto_approval', got '%s'", section.ID())
	}

	if section.Title() == "" {
		t.Error("Title should not be empty")
	}

	if section.Description() == "" {
		t.Error("Description should not be empty")
	}

	// Should start with empty tools map
	tools := section.GetTools()
	if len(tools) != 0 {
		t.Error("New section should have no tools")
	}
}

func TestAutoApprovalSection_Data(t *testing.T) {
	t.Run("returns empty data for new section", func(t *testing.T) {
		section := NewAutoApprovalSection()
		data := section.Data()

		if len(data) != 0 {
			t.Error("Expected empty data for new section")
		}
	})

	t.Run("returns all tool settings", func(t *testing.T) {
		section := NewAutoApprovalSection()
		section.SetToolAutoApproval("tool1", true)
		section.SetToolAutoApproval("tool2", false)

		data := section.Data()

		if len(data) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(data))
		}

		if data["tool1"] != true {
			t.Error("tool1 should be true")
		}

		if data["tool2"] != false {
			t.Error("tool2 should be false")
		}
	})
}

func TestAutoApprovalSection_SetData(t *testing.T) {
	t.Run("sets tool data correctly", func(t *testing.T) {
		section := NewAutoApprovalSection()
		data := map[string]interface{}{
			"tool1": true,
			"tool2": false,
			"tool3": true,
		}

		err := section.SetData(data)
		if err != nil {
			t.Fatalf("SetData failed: %v", err)
		}

		if !section.IsToolAutoApproved("tool1") {
			t.Error("tool1 should be auto-approved")
		}

		if section.IsToolAutoApproved("tool2") {
			t.Error("tool2 should not be auto-approved")
		}

		if !section.IsToolAutoApproved("tool3") {
			t.Error("tool3 should be auto-approved")
		}
	})

	t.Run("handles nil data", func(t *testing.T) {
		section := NewAutoApprovalSection()

		err := section.SetData(nil)
		if err != nil {
			t.Errorf("SetData should handle nil gracefully: %v", err)
		}
	})

	t.Run("rejects invalid value types", func(t *testing.T) {
		section := NewAutoApprovalSection()
		data := map[string]interface{}{
			"tool1": "not a bool",
		}

		err := section.SetData(data)
		if err == nil {
			t.Error("Expected error for invalid value type")
		}
	})

	t.Run("rejects non-bool values", func(t *testing.T) {
		section := NewAutoApprovalSection()
		data := map[string]interface{}{
			"tool1": 42,
		}

		err := section.SetData(data)
		if err == nil {
			t.Error("Expected error for non-bool value")
		}
	})
}

func TestAutoApprovalSection_Validate(t *testing.T) {
	t.Run("always validates successfully", func(t *testing.T) {
		section := NewAutoApprovalSection()

		// Empty section
		if err := section.Validate(); err != nil {
			t.Errorf("Empty section validation failed: %v", err)
		}

		// With data
		section.SetToolAutoApproval("tool1", true)
		section.SetToolAutoApproval("tool2", false)

		if err := section.Validate(); err != nil {
			t.Errorf("Section with data validation failed: %v", err)
		}
	})
}

func TestAutoApprovalSection_Reset(t *testing.T) {
	t.Run("resets all tools to false", func(t *testing.T) {
		section := NewAutoApprovalSection()
		section.SetToolAutoApproval("tool1", true)
		section.SetToolAutoApproval("tool2", true)
		section.SetToolAutoApproval("tool3", true)

		section.Reset()

		if section.IsToolAutoApproved("tool1") {
			t.Error("tool1 should be disabled after reset")
		}
		if section.IsToolAutoApproved("tool2") {
			t.Error("tool2 should be disabled after reset")
		}
		if section.IsToolAutoApproved("tool3") {
			t.Error("tool3 should be disabled after reset")
		}
	})

	t.Run("handles empty section", func(t *testing.T) {
		section := NewAutoApprovalSection()

		// Should not panic
		section.Reset()
	})
}

func TestAutoApprovalSection_EnsureToolExists(t *testing.T) {
	t.Run("adds new tool with default false", func(t *testing.T) {
		section := NewAutoApprovalSection()

		section.EnsureToolExists("new_tool")

		if section.IsToolAutoApproved("new_tool") {
			t.Error("New tool should default to not auto-approved")
		}

		tools := section.GetTools()
		if _, exists := tools["new_tool"]; !exists {
			t.Error("Tool was not added")
		}
	})

	t.Run("does not overwrite existing tool", func(t *testing.T) {
		section := NewAutoApprovalSection()
		section.SetToolAutoApproval("existing", true)

		section.EnsureToolExists("existing")

		if !section.IsToolAutoApproved("existing") {
			t.Error("Existing tool value was overwritten")
		}
	})
}

func TestAutoApprovalSection_IsToolAutoApproved(t *testing.T) {
	t.Run("returns true for enabled tool", func(t *testing.T) {
		section := NewAutoApprovalSection()
		section.SetToolAutoApproval("enabled_tool", true)

		if !section.IsToolAutoApproved("enabled_tool") {
			t.Error("Enabled tool should be auto-approved")
		}
	})

	t.Run("returns false for disabled tool", func(t *testing.T) {
		section := NewAutoApprovalSection()
		section.SetToolAutoApproval("disabled_tool", false)

		if section.IsToolAutoApproved("disabled_tool") {
			t.Error("Disabled tool should not be auto-approved")
		}
	})

	t.Run("returns false for unknown tool", func(t *testing.T) {
		section := NewAutoApprovalSection()

		if section.IsToolAutoApproved("unknown_tool") {
			t.Error("Unknown tool should not be auto-approved")
		}
	})

	t.Run("auto-registers unknown tool", func(t *testing.T) {
		section := NewAutoApprovalSection()

		// First call should return false and register the tool
		section.IsToolAutoApproved("new_tool")

		// Tool should now exist
		tools := section.GetTools()
		if _, exists := tools["new_tool"]; !exists {
			t.Error("Unknown tool should be auto-registered")
		}
	})
}

func TestAutoApprovalSection_SetToolAutoApproval(t *testing.T) {
	t.Run("sets tool approval status", func(t *testing.T) {
		section := NewAutoApprovalSection()

		section.SetToolAutoApproval("tool1", true)
		if !section.IsToolAutoApproved("tool1") {
			t.Error("Tool should be auto-approved")
		}

		section.SetToolAutoApproval("tool1", false)
		if section.IsToolAutoApproved("tool1") {
			t.Error("Tool should not be auto-approved")
		}
	})

	t.Run("can toggle tool status", func(t *testing.T) {
		section := NewAutoApprovalSection()

		section.SetToolAutoApproval("toggle_tool", true)
		section.SetToolAutoApproval("toggle_tool", false)
		section.SetToolAutoApproval("toggle_tool", true)

		if !section.IsToolAutoApproved("toggle_tool") {
			t.Error("Tool should be auto-approved after toggle")
		}
	})
}

func TestAutoApprovalSection_GetTools(t *testing.T) {
	t.Run("returns all tools", func(t *testing.T) {
		section := NewAutoApprovalSection()
		section.SetToolAutoApproval("tool1", true)
		section.SetToolAutoApproval("tool2", false)
		section.SetToolAutoApproval("tool3", true)

		tools := section.GetTools()

		if len(tools) != 3 {
			t.Errorf("Expected 3 tools, got %d", len(tools))
		}

		if tools["tool1"] != true {
			t.Error("tool1 value incorrect")
		}
		if tools["tool2"] != false {
			t.Error("tool2 value incorrect")
		}
		if tools["tool3"] != true {
			t.Error("tool3 value incorrect")
		}
	})

	t.Run("returns copy to prevent external modification", func(t *testing.T) {
		section := NewAutoApprovalSection()
		section.SetToolAutoApproval("tool1", true)

		tools := section.GetTools()
		tools["tool1"] = false
		tools["new_tool"] = true

		// Original should not be affected
		if !section.IsToolAutoApproved("tool1") {
			t.Error("External modification affected section")
		}

		if section.IsToolAutoApproved("new_tool") {
			t.Error("External modification added tool to section")
		}
	})

	t.Run("returns empty map for new section", func(t *testing.T) {
		section := NewAutoApprovalSection()
		tools := section.GetTools()

		if len(tools) != 0 {
			t.Error("Expected empty map for new section")
		}
	})
}
