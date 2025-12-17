//go:build testing
// +build testing

package config

// ResetGlobalManager resets the global configuration manager.
// This is a test-only helper to ensure a clean state between tests.
func ResetGlobalManager() {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalManager = nil
}
