// Package browser provides web browser automation capabilities through Playwright.
//
// The browser package enables Forge agents to interact with web browsers for testing,
// research, and automation tasks. It provides a session-based architecture where
// browser sessions persist across agent loop iterations, with dynamic tool registration
// based on active sessions.
//
// # Architecture
//
// The package is built around three core concepts:
//
// 1. Session: Encapsulates a Playwright browser instance with its context and page
// 2. SessionManager: Global registry managing all active browser sessions
// 3. Dynamic Tools: Browser operation tools that are registered/unregistered based on session state
//
// # Session Lifecycle
//
// Browser sessions follow this lifecycle:
//
//  1. Create: start_browser_session tool creates a new named session
//  2. Use: Navigation, interaction, and extraction tools operate on the session
//  3. Close: close_browser_session explicitly closes and cleans up resources
//  4. Timeout: Sessions auto-close after idle timeout (configurable)
//
// # Tool Registration
//
// Tools are dynamically registered based on session state:
//
//   - When no sessions exist: Only start_browser_session is available
//   - When sessions exist: All browser tools become available
//   - When last session closes: Dynamic tools are unregistered
//
// This approach reduces cognitive load when browser features aren't in use and
// provides clear feedback about browser availability.
//
// # Security
//
// Browser automation respects workspace boundaries and security constraints:
//
//   - Explicit opt-in required (browser_enabled setting)
//   - Headed mode default for transparency
//   - Resource limits (max sessions, idle timeout)
//   - Automatic cleanup on shutdown
//
// # Configuration
//
// Browser behavior is controlled via UISection settings:
//
//   - browser_enabled: Master toggle (default: false)
//   - browser_headless: Default mode for new sessions (default: true)
//
// # Example Usage
//
//	// Agent creates a session
//	session, err := manager.StartSession("research", SessionOptions{
//	    Headless: false,
//	    Viewport: &Viewport{Width: 1280, Height: 720},
//	})
//
//	// Navigate and extract content
//	err = session.Navigate("https://example.com", NavigateOptions{
//	    WaitUntil: "networkidle",
//	})
//	content, err := session.ExtractContent(ExtractOptions{
//	    Format: FormatMarkdown,
//	    MaxLength: 10000,
//	})
//
//	// Clean up
//	err = manager.CloseSession("research")
package browser
