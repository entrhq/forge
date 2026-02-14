package browser

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

// SessionManager manages all active browser sessions and coordinates
// with the tool registry for dynamic tool registration.
type SessionManager struct {
	mu          sync.RWMutex
	sessions    map[string]*Session
	playwright  *playwright.Playwright
	maxSessions int
	idleTimeout time.Duration
	initialized bool
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:    make(map[string]*Session),
		maxSessions: DefaultMaxSessions,
		idleTimeout: time.Duration(DefaultIdleTimeout) * time.Second,
		initialized: false,
	}
}

// Initialize initializes the Playwright instance.
// This must be called before creating any sessions.
func (m *SessionManager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return nil
	}

	// Install and run Playwright with verbose=false and discard output to avoid interfering with TUI
	opts := &playwright.RunOptions{
		Verbose: false,
		Stdout:  io.Discard,
		Stderr:  io.Discard,
	}

	err := playwright.Install(opts)
	if err != nil {
		return fmt.Errorf("failed to install playwright: %w", err)
	}

	pw, err := playwright.Run(opts)
	if err != nil {
		return fmt.Errorf("failed to start playwright: %w", err)
	}

	m.playwright = pw
	m.initialized = true
	return nil
}

// StartSession creates a new browser session with the given name and options.
func (m *SessionManager) StartSession(name string, opts SessionOptions) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if session already exists
	if _, exists := m.sessions[name]; exists {
		return nil, fmt.Errorf("session %q already exists", name)
	}

	// Check session limit
	if len(m.sessions) >= m.maxSessions {
		return nil, fmt.Errorf("maximum number of sessions (%d) reached", m.maxSessions)
	}

	// Ensure Playwright is initialized
	if !m.initialized {
		return nil, fmt.Errorf("session manager not initialized")
	}

	// Set defaults
	if opts.Viewport == nil {
		opts.Viewport = &Viewport{
			Width:  DefaultViewportWidth,
			Height: DefaultViewportHeight,
		}
	}
	if opts.Timeout == 0 {
		opts.Timeout = DefaultTimeout
	}

	// Launch browser
	launchOpts := playwright.BrowserTypeLaunchOptions{
		Headless: &opts.Headless,
	}
	browser, err := m.playwright.Chromium.Launch(launchOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	// Create context
	contextOpts := playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  opts.Viewport.Width,
			Height: opts.Viewport.Height,
		},
	}
	context, err := browser.NewContext(contextOpts)
	if err != nil {
		browser.Close()
		return nil, fmt.Errorf("failed to create context: %w", err)
	}

	// Create page
	page, err := context.NewPage()
	if err != nil {
		context.Close()
		browser.Close()
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// Set default timeout
	page.SetDefaultTimeout(opts.Timeout)

	// Create session
	now := time.Now()
	session := &Session{
		Name:       name,
		Browser:    browser,
		Context:    context,
		Page:       page,
		Headless:   opts.Headless,
		CreatedAt:  now,
		LastUsedAt: now,
		CurrentURL: "about:blank",
	}

	m.sessions[name] = session
	return session, nil
}

// CloseSession closes and removes a browser session.
func (m *SessionManager) CloseSession(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[name]
	if !exists {
		return fmt.Errorf("session %q not found", name)
	}

	// Close Playwright resources
	_ = session.Page.Close()    // Ignore errors, continue cleanup
	_ = session.Context.Close() // Ignore errors, continue cleanup
	_ = session.Browser.Close() // Ignore errors, continue cleanup

	delete(m.sessions, name)
	return nil
}

// GetSession retrieves an active session by name.
func (m *SessionManager) GetSession(name string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[name]
	if !exists {
		return nil, fmt.Errorf("session %q not found", name)
	}

	return session, nil
}

// ListSessions returns information about all active sessions.
func (m *SessionManager) ListSessions() []SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]SessionInfo, 0, len(m.sessions))
	for _, session := range m.sessions {
		infos = append(infos, SessionInfo{
			Name:       session.Name,
			CurrentURL: session.CurrentURL,
			Headless:   session.Headless,
			CreatedAt:  session.CreatedAt,
			LastUsedAt: session.LastUsedAt,
		})
	}

	return infos
}

// HasSessions returns true if there are any active sessions.
func (m *SessionManager) HasSessions() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions) > 0
}

// CloseAll closes all active sessions.
func (m *SessionManager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name := range m.sessions {
		session := m.sessions[name]

		// Close resources
		if err := session.Page.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := session.Context.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := session.Browser.Close(); err != nil {
			errs = append(errs, err)
		}

		delete(m.sessions, name)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing sessions: %v", errs)
	}
	return nil
}

// Shutdown closes all sessions and cleans up Playwright.
func (m *SessionManager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close all sessions
	for name := range m.sessions {
		session := m.sessions[name]
		session.Page.Close()
		session.Context.Close()
		session.Browser.Close()
		delete(m.sessions, name)
	}

	// Stop Playwright
	if m.initialized && m.playwright != nil {
		if err := m.playwright.Stop(); err != nil {
			return fmt.Errorf("failed to stop playwright: %w", err)
		}
		m.initialized = false
	}

	return nil
}

// CleanupIdleSessions closes sessions that have been idle for longer than the timeout.
func (m *SessionManager) CleanupIdleSessions() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var toClose []string

	for name, session := range m.sessions {
		if now.Sub(session.LastUsedAt) > m.idleTimeout {
			toClose = append(toClose, name)
		}
	}

	// Close idle sessions
	var errs []error
	for _, name := range toClose {
		session := m.sessions[name]

		if err := session.Page.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := session.Context.Close(); err != nil {
			errs = append(errs, err)
		}
		if err := session.Browser.Close(); err != nil {
			errs = append(errs, err)
		}

		delete(m.sessions, name)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during cleanup: %v", errs)
	}
	return nil
}

// SetMaxSessions sets the maximum number of concurrent sessions.
func (m *SessionManager) SetMaxSessions(max int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxSessions = max
}

// SetIdleTimeout sets the idle timeout duration.
func (m *SessionManager) SetIdleTimeout(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idleTimeout = timeout
}

// SessionInfo contains metadata about a browser session.
type SessionInfo struct {
	Name       string
	CurrentURL string
	Headless   bool
	CreatedAt  time.Time
	LastUsedAt time.Time
}
