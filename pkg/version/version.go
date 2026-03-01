// Package version exposes the application version as a shared constant so that
// any package (TUI header, CLI banner, headless output) can reference it from a
// single source of truth without creating an import cycle.
package version

// Version is the current Forge application version.
const Version = "0.1.0"
