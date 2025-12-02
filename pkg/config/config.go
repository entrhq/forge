package config

import (
	"sync"
)

var (
	// globalManager is the singleton configuration manager instance
	globalManager *Manager
	globalMu      sync.Mutex
)

// Initialize creates and initializes the global configuration manager.
// This should be called once at application startup.
func Initialize(configPath string) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	// Create file store
	store, err := NewFileStore(configPath)
	if err != nil {
		return err
	}

	// Create manager
	manager := NewManager(store)

	// Register default sections
	if err := manager.RegisterSection(NewAutoApprovalSection()); err != nil {
		return err
	}

	if err := manager.RegisterSection(NewCommandWhitelistSection()); err != nil {
		return err
	}

	if err := manager.RegisterSection(NewLLMSection()); err != nil {
		return err
	}

	// Load configuration
	if err := manager.LoadAll(); err != nil {
		return err
	}

	globalManager = manager
	return nil
}

// Global returns the global configuration manager.
// Panics if Initialize has not been called.
func Global() *Manager {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalManager == nil {
		panic("config not initialized: call config.Initialize first")
	}

	return globalManager
}

// IsInitialized returns true if the global configuration has been initialized.
func IsInitialized() bool {
	globalMu.Lock()
	defer globalMu.Unlock()
	return globalManager != nil
}

// GetAutoApproval returns the auto-approval section from global config.
// Returns nil if config is not initialized.
func GetAutoApproval() *AutoApprovalSection {
	if !IsInitialized() {
		return nil
	}

	section, ok := Global().GetSection("auto_approval")
	if !ok {
		return nil
	}

	autoApproval, ok := section.(*AutoApprovalSection)
	if !ok {
		return nil
	}

	return autoApproval
}

// GetCommandWhitelist returns the command whitelist section from global config.
// Returns nil if config is not initialized.
func GetCommandWhitelist() *CommandWhitelistSection {
	if !IsInitialized() {
		return nil
	}

	section, ok := Global().GetSection("command_whitelist")
	if !ok {
		return nil
	}

	whitelist, ok := section.(*CommandWhitelistSection)
	if !ok {
		return nil
	}

	return whitelist
}

// IsToolAutoApproved checks if a tool is configured for auto-approval.
// Returns false if config is not initialized.
func IsToolAutoApproved(toolName string) bool {
	autoApproval := GetAutoApproval()
	if autoApproval == nil {
		return false
	}
	return autoApproval.IsToolAutoApproved(toolName)
}

// GetLLM returns the LLM settings section from global config.
// Returns nil if config is not initialized.
func GetLLM() *LLMSection {
	if !IsInitialized() {
		return nil
	}

	section, ok := Global().GetSection("llm")
	if !ok {
		return nil
	}

	llm, ok := section.(*LLMSection)
	if !ok {
		return nil
	}

	return llm
}

// IsCommandWhitelisted checks if a command is whitelisted for auto-approval.
// Returns false if config is not initialized.
func IsCommandWhitelisted(command string) bool {
	whitelist := GetCommandWhitelist()
	if whitelist == nil {
		return false
	}
	return whitelist.IsCommandWhitelisted(command)
}
