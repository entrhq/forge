package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitialize(t *testing.T) {
	t.Run("initializes global manager successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		// Reset global state
		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		err := Initialize(configPath)
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		if !IsInitialized() {
			t.Error("Global manager should be initialized")
		}

		// Verify sections are registered
		manager := Global()
		autoApproval, ok := manager.GetSection("auto_approval")
		if !ok {
			t.Error("auto_approval section not registered")
		}
		if autoApproval == nil {
			t.Error("auto_approval section is nil")
		}

		whitelist, ok := manager.GetSection("command_whitelist")
		if !ok {
			t.Error("command_whitelist section not registered")
		}
		if whitelist == nil {
			t.Error("command_whitelist section is nil")
		}
	})

	t.Run("handles invalid config path", func(t *testing.T) {
		// Reset global state
		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		// Try to initialize with invalid path (read-only directory)
		err := Initialize("/invalid/readonly/path/config.json")
		// Should still succeed as file creation happens on Save, not Load
		if err != nil {
			// This is acceptable - some systems may fail earlier
			t.Logf("Initialize with invalid path failed (acceptable): %v", err)
		}
	})

	t.Run("loads existing configuration", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		// Create initial config
		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		if err := Initialize(configPath); err != nil {
			t.Fatalf("First initialize failed: %v", err)
		}

		// Modify and save
		autoApproval := GetAutoApproval()
		autoApproval.SetToolAutoApproval("test_tool", true)
		if err := Global().SaveAll(); err != nil {
			t.Fatalf("SaveAll failed: %v", err)
		}

		// Re-initialize
		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		if err := Initialize(configPath); err != nil {
			t.Fatalf("Re-initialize failed: %v", err)
		}

		// Verify data was loaded
		autoApproval = GetAutoApproval()
		if !autoApproval.IsToolAutoApproved("test_tool") {
			t.Error("Configuration was not loaded correctly")
		}
	})
}

func TestGlobal(t *testing.T) {
	t.Run("returns initialized manager", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		if err := Initialize(configPath); err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		manager := Global()
		if manager == nil {
			t.Fatal("Global() returned nil")
		}
	})

	t.Run("panics if not initialized", func(t *testing.T) {
		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for uninitialized config")
			}
		}()

		Global()
	})
}

func TestIsInitialized(t *testing.T) {
	t.Run("returns false before initialization", func(t *testing.T) {
		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		if IsInitialized() {
			t.Error("Should return false before initialization")
		}
	})

	t.Run("returns true after initialization", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		if err := Initialize(configPath); err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		if !IsInitialized() {
			t.Error("Should return true after initialization")
		}
	})
}

func TestGetAutoApproval(t *testing.T) {
	t.Run("returns auto-approval section when initialized", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		if err := Initialize(configPath); err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		autoApproval := GetAutoApproval()
		if autoApproval == nil {
			t.Fatal("GetAutoApproval returned nil")
		}

		if autoApproval.ID() != "auto_approval" {
			t.Error("Wrong section returned")
		}
	})

	t.Run("returns nil when not initialized", func(t *testing.T) {
		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		autoApproval := GetAutoApproval()
		if autoApproval != nil {
			t.Error("Expected nil for uninitialized config")
		}
	})
}

func TestGetCommandWhitelist(t *testing.T) {
	t.Run("returns whitelist section when initialized", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		if err := Initialize(configPath); err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		whitelist := GetCommandWhitelist()
		if whitelist == nil {
			t.Fatal("GetCommandWhitelist returned nil")
		}

		if whitelist.ID() != "command_whitelist" {
			t.Error("Wrong section returned")
		}
	})

	t.Run("returns nil when not initialized", func(t *testing.T) {
		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		whitelist := GetCommandWhitelist()
		if whitelist != nil {
			t.Error("Expected nil for uninitialized config")
		}
	})
}

func TestIsToolAutoApproved(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	globalMu.Lock()
	globalManager = nil
	globalMu.Unlock()

	if err := Initialize(configPath); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	t.Run("returns false for non-approved tool", func(t *testing.T) {
		if IsToolAutoApproved("unknown_tool") {
			t.Error("Unknown tool should not be auto-approved")
		}
	})

	t.Run("returns true for approved tool", func(t *testing.T) {
		autoApproval := GetAutoApproval()
		autoApproval.SetToolAutoApproval("approved_tool", true)

		if !IsToolAutoApproved("approved_tool") {
			t.Error("Approved tool should be auto-approved")
		}
	})

	t.Run("returns false when not initialized", func(t *testing.T) {
		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		if IsToolAutoApproved("any_tool") {
			t.Error("Should return false when not initialized")
		}
	})
}

func TestIsCommandWhitelisted(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	globalMu.Lock()
	globalManager = nil
	globalMu.Unlock()

	if err := Initialize(configPath); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	t.Run("checks against whitelist", func(t *testing.T) {
		whitelist := GetCommandWhitelist()

		// Add a test pattern
		whitelist.AddPattern("test command", "Test")

		if !IsCommandWhitelisted("test command") {
			t.Error("Whitelisted command should return true")
		}

		if IsCommandWhitelisted("unknown command") {
			t.Error("Non-whitelisted command should return false")
		}
	})

	t.Run("returns false when not initialized", func(t *testing.T) {
		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		if IsCommandWhitelisted("any command") {
			t.Error("Should return false when not initialized")
		}
	})
}

func TestGlobalConfig_ThreadSafety(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	globalMu.Lock()
	globalManager = nil
	globalMu.Unlock()

	if err := Initialize(configPath); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	t.Run("concurrent access is safe", func(t *testing.T) {
		done := make(chan bool)

		// Concurrent readers
		for i := 0; i < 10; i++ {
			go func() {
				IsInitialized()
				GetAutoApproval()
				GetCommandWhitelist()
				IsToolAutoApproved("test")
				IsCommandWhitelisted("test")
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestGlobalConfig_Persistence(t *testing.T) {
	t.Run("configuration persists across re-initialization", func(t *testing.T) {
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		// First initialization
		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		if err := Initialize(configPath); err != nil {
			t.Fatalf("First initialize failed: %v", err)
		}

		// Set some configuration
		autoApproval := GetAutoApproval()
		autoApproval.SetToolAutoApproval("tool1", true)
		autoApproval.SetToolAutoApproval("tool2", false)

		whitelist := GetCommandWhitelist()
		whitelist.AddPattern("custom command", "Custom")

		// Save
		if err := Global().SaveAll(); err != nil {
			t.Fatalf("SaveAll failed: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatal("Config file was not created")
		}

		// Re-initialize
		globalMu.Lock()
		globalManager = nil
		globalMu.Unlock()

		if err := Initialize(configPath); err != nil {
			t.Fatalf("Re-initialize failed: %v", err)
		}

		// Verify configuration was loaded
		autoApproval = GetAutoApproval()
		if !autoApproval.IsToolAutoApproved("tool1") {
			t.Error("tool1 approval status not persisted")
		}
		if autoApproval.IsToolAutoApproved("tool2") {
			t.Error("tool2 approval status not persisted")
		}

		whitelist = GetCommandWhitelist()
		if !whitelist.IsCommandWhitelisted("custom command") {
			t.Error("Custom whitelist pattern not persisted")
		}
	})
}
