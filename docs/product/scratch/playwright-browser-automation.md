# Feature Idea: Playwright Browser Automation

**Status:** Draft  
**Priority:** High Impact, Medium-Term  
**Last Updated:** December 2024

---

## Overview

Integrate Playwright browser automation into Forge, enabling the agent to interact with web browsers for knowledge access, application testing, and automated workflows. This transforms Forge from a code-focused tool into a computer-use agent that can browse the web, extract information, test applications, and perform web-based tasks on behalf of the user.

---

## Problem Statement

Developers and AI agents face significant limitations without browser access:

**Knowledge Access Barriers:**
- AI training data has cutoff dates (stale knowledge)
- Latest documentation and API changes not available
- Real-time web content inaccessible
- Dynamic web applications can't be inspected
- No way to verify current state of web services

**Development Workflow Gaps:**
- Can't test web applications directly
- Unable to verify UI changes without manual testing
- No automated way to check application behavior
- Missing context from running applications
- Can't perform end-to-end validation

**Agent Limitations:**
- Can't access information beyond training data
- Unable to perform web-based research
- No way to interact with web tools/services
- Limited to file-based operations
- Missing "computer use" capabilities

This leads to:
- Incomplete development cycles (can't test what you build)
- Stale or missing information
- Manual verification required
- Context switching to browser for information
- Reduced agent autonomy and capability

---

## Key Capabilities

### Phase 1: Core Navigation & Extraction (MVP)

**Session Management:**
- Create named browser sessions (persistent across agent loop)
- Multiple concurrent sessions support
- Session lifecycle management (create, use, close)
- Dynamic tool loading (tools appear only when session active)
- Headless and headed mode support

**Basic Navigation:**
- Navigate to URLs
- Wait for page load/navigation completion
- Get current URL and page metadata
- Go back/forward in history
- Reload pages

**Content Extraction:**
- Extract page content in multiple formats:
  - Markdown (preserves structure, links, headings)
  - Plain text (just visible text)
  - Structured data (title, headings, links, body)
- Search for elements on current page
- Get element text/attributes
- Extract specific sections by selector

**Interactive Elements:**
- Click elements (by selector, text, role)
- Fill form inputs
- Select dropdown options
- Check/uncheck boxes
- Submit forms
- Wait for elements to appear/disappear

**Context Management:**
- Filter and trim large page content
- Extract only relevant sections
- Search current page for specific content
- Avoid polluting agent context with full page HTML
- Smart content summarization

### Phase 2: Advanced Features (Future)

**Scriptable Workflows:**
- Agent writes Playwright Go code
- Execute multi-step automation scripts
- Reusable workflow templates
- Error handling and retries

**Enhanced Capabilities:**
- Screenshot capture (on-demand or on error)
- Network request interception/monitoring
- Cookie and localStorage manipulation
- Multiple page/tab management
- File download handling
- Iframe navigation

**Testing Integration:**
- Assertion support for validation
- Test case generation
- Visual regression testing
- Performance monitoring
- Accessibility checking

---

## Example Use Cases

### 1. Knowledge Access - Documentation Lookup

```
User: What are the breaking changes in React 19?

Agent: Let me check the latest React documentation...

[Creating browser session: "react_docs"]
[Navigating to: https://react.dev/blog]
[Extracting content in markdown format...]

React 19 - Breaking Changes:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Released: Beta (December 2024)

Major Breaking Changes:

1. IE11 Support Removed
   - No longer compatible with Internet Explorer
   - Minimum browser versions increased

2. Deprecated APIs Removed
   - ReactDOM.render() → Use createRoot()
   - react-test-renderer → Use React Testing Library

3. New JSX Transform Required
   - Must use React 17+ JSX transform
   - Update babel/typescript config

4. Stricter Development Warnings
   - More aggressive warnings for anti-patterns
   - Potential runtime changes in strict mode

Migration Guide: https://react.dev/blog/2024/12/react-19-upgrade-guide

Want me to help you plan a migration for your project?

[Session "react_docs" remains active for follow-up questions]
```

### 2. Application Testing - Login Flow

```
User: Test the login flow on localhost:3000

Agent: I'll test your login functionality...

[Creating browser session: "app_test" (headed mode)]
[Navigating to: http://localhost:3000]
[Waiting for page load...]

Testing Login Flow:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Step 1: Navigate to login page
✓ Login page loaded
✓ Found email and password inputs
✓ Found submit button

Step 2: Fill login form
[Filling email: test@example.com]
[Filling password: ********]
[Clicking submit button]

Step 3: Wait for navigation
✓ Redirected to: http://localhost:3000/dashboard
✓ Found user profile element
✓ Login successful

Step 4: Verify dashboard content
[Extracting dashboard content...]
✓ User name displayed: "Test User"
✓ Navigation menu present
✓ Main content area loaded

Test Result: ✓ PASSED
All login flow steps completed successfully.

[Session "app_test" active - you can see the browser window]

Want me to test any specific functionality on the dashboard?
```

### 3. Web Research - Finding Solutions

```
User: How do I implement rate limiting in Go?

Agent: Let me search for current best practices...

[Creating browser session: "research"]
[Navigating to: https://www.google.com/search?q=rate+limiting+go+best+practices]
[Extracting top results...]

Top Resources Found:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. golang.org/x/time/rate (Official Package)
   [Following link...]
   
   The official rate limiter uses token bucket algorithm:
   
   ```go
   import "golang.org/x/time/rate"
   
   // Create limiter: 10 requests per second, burst of 5
   limiter := rate.NewLimiter(10, 5)
   
   // In your handler
   if !limiter.Allow() {
       http.Error(w, "Rate limit exceeded", 429)
       return
   }
   ```

2. github.com/uber-go/ratelimit (Production-Grade)
   [Navigating to repo...]
   
   Uber's library with simpler API:
   
   ```go
   import "go.uber.org/ratelimit"
   
   rl := ratelimit.New(100) // 100 requests per second
   
   for i := 0; i < 10; i++ {
       rl.Take() // Blocks until allowed
       makeRequest()
   }
   ```

3. Per-User Rate Limiting Pattern
   [Extracting code example...]
   
   Using sync.Map for per-user limits:
   [Full example code extracted...]

Which approach would work best for your use case?

[Closing browser session: "research"]
```

### 4. Form Automation - Newsletter Signup

```
User: Sign me up for the Go Weekly newsletter

Agent: I'll handle the newsletter signup...

[Creating browser session: "signup"]
[Navigating to: https://golangweekly.com]
[Searching for signup form...]

Newsletter Signup:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Found signup form with fields:
- Email address (required)
- Name (optional)

I need your email address to complete the signup.
What email should I use?

[Agent asks question, waits for user response]

User: user@example.com

Agent: Completing signup...

[Filling email: user@example.com]
[Clicking subscribe button]
[Waiting for confirmation...]

✓ Signup successful!
✓ Confirmation message: "Please check your email to verify your subscription"

Next steps:
1. Check your inbox for verification email
2. Click the verification link
3. You'll receive weekly newsletters

[Closing browser session: "signup"]
```

### 5. Testing Multiple Scenarios

```
User: Test our checkout flow with different payment methods

Agent: I'll test all payment scenarios...

[Creating browser session: "checkout_test"]

Testing Checkout Flow:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Scenario 1: Credit Card Payment
[Navigating to: http://localhost:3000/products]
[Adding item to cart...]
[Proceeding to checkout...]
[Selecting payment method: Credit Card]
[Filling test card details...]
✓ Payment successful
✓ Order confirmation displayed
✓ Order ID: #12345

Scenario 2: PayPal Payment
[Starting new checkout flow...]
[Selecting payment method: PayPal]
[Clicking PayPal button...]
⚠️ Issue detected: PayPal button not responding
❌ Timeout waiting for PayPal popup

Scenario 3: Bank Transfer
[Starting new checkout flow...]
[Selecting payment method: Bank Transfer]
[Filling bank details...]
✓ Instructions displayed correctly
✓ Order placed (pending payment)

Test Summary:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ Credit Card: PASSED
❌ PayPal: FAILED (button not responding)
✓ Bank Transfer: PASSED

Found issue that needs attention: PayPal integration

Want me to investigate the PayPal button issue?

[Session "checkout_test" active for debugging]
```

---

## Technical Approach

### Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                     Forge Agent                         │
│  ┌───────────────────────────────────────────────────┐ │
│  │           Agent Loop & Prompt System              │ │
│  │  - Dynamic tool loading based on session state    │ │
│  │  - Tool availability managed by BrowserManager    │ │
│  └───────────────────────────────────────────────────┘ │
│                          │                              │
│                          ▼                              │
│  ┌───────────────────────────────────────────────────┐ │
│  │            Browser Tool Interface                 │ │
│  │  - create_browser_session (always available)      │ │
│  │  - navigate, extract, click, fill (session-gated) │ │
│  │  - close_browser_session (session-gated)          │ │
│  └───────────────────────────────────────────────────┘ │
│                          │                              │
│                          ▼                              │
│  ┌───────────────────────────────────────────────────┐ │
│  │           Browser Session Manager                 │ │
│  │  - Named session tracking                         │ │
│  │  - Lifecycle management (create/close)            │ │
│  │  - Tool availability control                      │ │
│  │  - Multiple concurrent sessions                   │ │
│  └───────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│           Playwright Go Integration                     │
│  github.com/playwright-community/playwright-go          │
│                                                         │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐ │
│  │   Browser   │  │    Page      │  │   Elements    │ │
│  │  Instance   │  │  Navigation  │  │  Interaction  │ │
│  └─────────────┘  └──────────────┘  └───────────────┘ │
│                                                         │
│  - Chromium/Firefox/WebKit support                     │
│  - Headless and headed modes                           │
│  - Network interception ready (future)                 │
│  - Screenshot capabilities (future)                    │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
                  ┌───────────────┐
                  │  Web Browser  │
                  │   (Chromium)  │
                  └───────────────┘
```

### Session Management Implementation

```go
// pkg/browser/manager.go
package browser

import (
    "context"
    "fmt"
    "sync"
    
    "github.com/playwright-community/playwright-go"
)

// SessionManager manages browser sessions and their lifecycle
type SessionManager struct {
    mu       sync.RWMutex
    sessions map[string]*Session
    pw       *playwright.Playwright
    config   *Config
}

// Session represents a browser session
type Session struct {
    ID       string
    Browser  playwright.Browser
    Context  playwright.BrowserContext
    Page     playwright.Page
    Created  time.Time
    Config   SessionConfig
}

// SessionConfig holds session-specific configuration
type SessionConfig struct {
    Headless bool
    UserAgent string
    Viewport *Viewport
}

// Config holds global browser configuration
type Config struct {
    DefaultHeadless bool
    BrowserType     string // chromium, firefox, webkit
    UserDataDir     string
}

// NewSessionManager creates a new session manager
func NewSessionManager(config *Config) (*SessionManager, error) {
    pw, err := playwright.Run()
    if err != nil {
        return nil, fmt.Errorf("failed to start playwright: %w", err)
    }
    
    return &SessionManager{
        sessions: make(map[string]*Session),
        pw:       pw,
        config:   config,
    }, nil
}

// CreateSession creates a new browser session
func (sm *SessionManager) CreateSession(id string, config SessionConfig) (*Session, error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    if _, exists := sm.sessions[id]; exists {
        return nil, fmt.Errorf("session %s already exists", id)
    }
    
    // Launch browser
    browser, err := sm.launchBrowser(config)
    if err != nil {
        return nil, err
    }
    
    // Create context
    context, err := browser.NewContext(playwright.BrowserNewContextOptions{
        UserAgent: playwright.String(config.UserAgent),
    })
    if err != nil {
        browser.Close()
        return nil, err
    }
    
    // Create page
    page, err := context.NewPage()
    if err != nil {
        context.Close()
        browser.Close()
        return nil, err
    }
    
    session := &Session{
        ID:      id,
        Browser: browser,
        Context: context,
        Page:    page,
        Created: time.Now(),
        Config:  config,
    }
    
    sm.sessions[id] = session
    return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(id string) (*Session, error) {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    
    session, exists := sm.sessions[id]
    if !exists {
        return nil, fmt.Errorf("session %s not found", id)
    }
    return session, nil
}

// CloseSession closes and removes a session
func (sm *SessionManager) CloseSession(id string) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    session, exists := sm.sessions[id]
    if !exists {
        return fmt.Errorf("session %s not found", id)
    }
    
    // Clean up in reverse order
    if err := session.Page.Close(); err != nil {
        return err
    }
    if err := session.Context.Close(); err != nil {
        return err
    }
    if err := session.Browser.Close(); err != nil {
        return err
    }
    
    delete(sm.sessions, id)
    return nil
}

// HasActiveSessions returns true if any sessions exist
func (sm *SessionManager) HasActiveSessions() bool {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return len(sm.sessions) > 0
}

// ListSessions returns all active session IDs
func (sm *SessionManager) ListSessions() []string {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    
    ids := make([]string, 0, len(sm.sessions))
    for id := range sm.sessions {
        ids = append(ids, id)
    }
    return ids
}

// Shutdown closes all sessions and stops playwright
func (sm *SessionManager) Shutdown() error {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    for id := range sm.sessions {
        sm.CloseSession(id)
    }
    
    return sm.pw.Stop()
}
```

### Dynamic Tool Loading

```go
// pkg/tools/browser/loader.go
package browser

import (
    "github.com/tmc/forge/pkg/tools"
)

// ToolLoader manages dynamic browser tool availability
type ToolLoader struct {
    manager *SessionManager
}

// GetAvailableTools returns tools based on session state
func (tl *ToolLoader) GetAvailableTools() []tools.Tool {
    // create_browser_session is always available
    baseTools := []tools.Tool{
        &CreateSessionTool{manager: tl.manager},
    }
    
    // If no sessions exist, only show create tool
    if !tl.manager.HasActiveSessions() {
        return baseTools
    }
    
    // If sessions exist, show all browser tools
    return append(baseTools,
        &NavigateTool{manager: tl.manager},
        &ExtractContentTool{manager: tl.manager},
        &ClickElementTool{manager: tl.manager},
        &FillInputTool{manager: tl.manager},
        &SearchPageTool{manager: tl.manager},
        &WaitForElementTool{manager: tl.manager},
        &CloseSessionTool{manager: tl.manager},
        &ListSessionsTool{manager: tl.manager},
    )
}

// ShouldReloadTools returns true if tool availability changed
func (tl *ToolLoader) ShouldReloadTools(previousState bool) bool {
    currentState := tl.manager.HasActiveSessions()
    return previousState != currentState
}
```

### Content Extraction with Context Protection

```go
// pkg/browser/extraction.go
package browser

import (
    "strings"
    
    "github.com/JohannesKaufmann/html-to-markdown"
)

// ContentExtractor handles page content extraction
type ContentExtractor struct {
    converter *md.Converter
}

// ExtractFormat defines output format
type ExtractFormat string

const (
    FormatMarkdown   ExtractFormat = "markdown"
    FormatPlainText  ExtractFormat = "text"
    FormatStructured ExtractFormat = "structured"
)

// ExtractOptions configures content extraction
type ExtractOptions struct {
    Format       ExtractFormat
    MaxLength    int  // Character limit to protect context
    Selector     string // Optional: extract specific element
    IncludeLinks bool
    IncludeImages bool
}

// Extract extracts page content based on options
func (ce *ContentExtractor) Extract(page playwright.Page, opts ExtractOptions) (*ExtractedContent, error) {
    var content string
    var err error
    
    // Get HTML content
    if opts.Selector != "" {
        // Extract specific element
        element := page.Locator(opts.Selector)
        content, err = element.InnerHTML()
    } else {
        // Extract full page
        content, err = page.Content()
    }
    
    if err != nil {
        return nil, err
    }
    
    // Convert based on format
    switch opts.Format {
    case FormatMarkdown:
        content = ce.toMarkdown(content, opts)
    case FormatPlainText:
        content = ce.toPlainText(content)
    case FormatStructured:
        return ce.toStructured(page)
    }
    
    // Enforce length limit to protect context
    if opts.MaxLength > 0 && len(content) > opts.MaxLength {
        content = content[:opts.MaxLength] + "\n\n[Content truncated - use selector or search to get specific sections]"
    }
    
    return &ExtractedContent{
        Content: content,
        Format:  opts.Format,
        URL:     page.URL(),
        Title:   ce.getTitle(page),
    }, nil
}

// SearchPage searches for content on current page
func (ce *ContentExtractor) SearchPage(page playwright.Page, query string) ([]SearchMatch, error) {
    // Get all text content
    textContent, _ := page.TextContent("body")
    
    // Find matches (case-insensitive)
    matches := []SearchMatch{}
    lowerContent := strings.ToLower(textContent)
    lowerQuery := strings.ToLower(query)
    
    index := 0
    for {
        pos := strings.Index(lowerContent[index:], lowerQuery)
        if pos == -1 {
            break
        }
        
        actualPos := index + pos
        start := max(0, actualPos-100)
        end := min(len(textContent), actualPos+len(query)+100)
        
        matches = append(matches, SearchMatch{
            Text:     textContent[start:end],
            Position: actualPos,
            Context:  extractContext(textContent, actualPos, 200),
        })
        
        index = actualPos + len(query)
    }
    
    return matches, nil
}

// ExtractedContent represents extracted page content
type ExtractedContent struct {
    Content string
    Format  ExtractFormat
    URL     string
    Title   string
}

// SearchMatch represents a search result on the page
type SearchMatch struct {
    Text     string
    Position int
    Context  string
}
```

### Configuration Integration

```go
// pkg/config/browser.go
package config

// BrowserConfig holds browser-specific configuration
type BrowserConfig struct {
    Enabled         bool   `yaml:"enabled"`
    DefaultHeadless bool   `yaml:"default_headless"`
    BrowserType     string `yaml:"browser_type"` // chromium, firefox, webkit
    UserAgent       string `yaml:"user_agent"`
    MaxSessions     int    `yaml:"max_sessions"`
}

// Default configuration
func DefaultBrowserConfig() BrowserConfig {
    return BrowserConfig{
        Enabled:         false, // Disabled by default, user must enable
        DefaultHeadless: false, // Default to headed mode as specified
        BrowserType:     "chromium",
        UserAgent:       "Forge AI Agent",
        MaxSessions:     5,
    }
}
```

### Tool Implementation Examples

```go
// pkg/tools/browser/navigate.go
package browser

// NavigateTool navigates to a URL
type NavigateTool struct {
    manager *SessionManager
}

func (t *NavigateTool) Execute(params map[string]interface{}) (*ToolResult, error) {
    sessionID := params["session_id"].(string)
    url := params["url"].(string)
    waitUntil := params["wait_until"].(string) // load, domcontentloaded, networkidle
    
    session, err := t.manager.GetSession(sessionID)
    if err != nil {
        return nil, err
    }
    
    // Navigate
    _, err = session.Page.Goto(url, playwright.PageGotoOptions{
        WaitUntil: playwright.WaitUntilState(waitUntil),
    })
    if err != nil {
        return nil, err
    }
    
    // Get page info
    title, _ := session.Page.Title()
    currentURL := session.Page.URL()
    
    return &ToolResult{
        Success: true,
        Message: fmt.Sprintf("Navigated to: %s\nTitle: %s", currentURL, title),
        Data: map[string]interface{}{
            "url":   currentURL,
            "title": title,
        },
    }, nil
}

// pkg/tools/browser/extract.go
package browser

// ExtractContentTool extracts page content
type ExtractContentTool struct {
    manager   *SessionManager
    extractor *ContentExtractor
}

func (t *ExtractContentTool) Execute(params map[string]interface{}) (*ToolResult, error) {
    sessionID := params["session_id"].(string)
    format := params["format"].(string) // markdown, text, structured
    selector := params["selector"].(string) // optional
    maxLength := params["max_length"].(int) // default: 10000
    
    session, err := t.manager.GetSession(sessionID)
    if err != nil {
        return nil, err
    }
    
    // Extract content
    content, err := t.extractor.Extract(session.Page, ExtractOptions{
        Format:    ExtractFormat(format),
        MaxLength: maxLength,
        Selector:  selector,
    })
    if err != nil {
        return nil, err
    }
    
    return &ToolResult{
        Success: true,
        Message: fmt.Sprintf("Extracted content from: %s", content.URL),
        Data: map[string]interface{}{
            "content": content.Content,
            "format":  content.Format,
            "url":     content.URL,
            "title":   content.Title,
        },
    }, nil
}
```

---

## Integration Points

### Agent Loop Integration

**Tool Availability Check:**
```go
// pkg/agent/loop.go
func (a *Agent) getAvailableTools() []tools.Tool {
    baseTools := a.baseTools // read_file, write_file, etc.
    
    // Add browser tools if any sessions exist
    if a.browserManager != nil {
        browserTools := a.browserLoader.GetAvailableTools()
        baseTools = append(baseTools, browserTools...)
    }
    
    return baseTools
}

// After each tool execution, check if tools changed
func (a *Agent) afterToolExecution() {
    if a.browserManager != nil {
        if a.browserLoader.ShouldReloadTools(a.lastBrowserState) {
            a.reloadSystemPrompt() // Rebuild prompt with new tools
            a.lastBrowserState = a.browserManager.HasActiveSessions()
        }
    }
}
```

### Settings Menu Integration

**Browser Settings Section:**
```go
// pkg/executor/tui/settings.go
func (s *SettingsView) getBrowserSection() SettingsSection {
    return SettingsSection{
        Title: "Browser",
        Settings: []Setting{
            {
                Key:         "browser.enabled",
                Label:       "Enable Browser Automation",
                Type:        SettingTypeBool,
                Value:       s.config.Browser.Enabled,
                Description: "Enable Playwright browser automation features",
            },
            {
                Key:         "browser.default_headless",
                Label:       "Default to Headless Mode",
                Type:        SettingTypeBool,
                Value:       s.config.Browser.DefaultHeadless,
                Description: "Run browser in headless mode by default (no visible window)",
            },
            {
                Key:         "browser.browser_type",
                Label:       "Browser Type",
                Type:        SettingTypeSelect,
                Options:     []string{"chromium", "firefox", "webkit"},
                Value:       s.config.Browser.BrowserType,
                Description: "Which browser engine to use",
            },
            {
                Key:         "browser.max_sessions",
                Label:       "Max Concurrent Sessions",
                Type:        SettingTypeNumber,
                Value:       s.config.Browser.MaxSessions,
                Description: "Maximum number of browser sessions allowed",
            },
        },
    }
}
```

---

## Security & Safety Considerations

### Resource Management
- Limit maximum concurrent sessions (default: 5)
- Session timeout for idle sessions (configurable)
- Memory limits for browser processes
- Automatic cleanup on agent shutdown

### User Control
- Browser features disabled by default
- Explicit opt-in via configuration
- Headed mode by default (user can see what agent is doing)
- Session names visible to user
- User can force-close sessions

### Future Protections (Phase 2)
- Domain whitelist/blacklist
- URL pattern restrictions
- User approval for navigation (optional)
- Sensitive data detection in forms
- Rate limiting for requests

---

## Implementation Phases

### Phase 1: MVP (4-6 weeks)
**Week 1-2: Core Infrastructure**
- Playwright Go integration
- Session manager implementation
- Basic tool structure
- Configuration integration

**Week 3-4: Navigation & Extraction**
- Navigate tool
- Content extraction (markdown, text, structured)
- Search page functionality
- Wait for elements

**Week 5-6: Interaction & Polish**
- Click element tool
- Fill input tool
- Dynamic tool loading
- Settings menu integration
- Documentation
- Testing

**Deliverables:**
- create_browser_session, close_browser_session
- navigate, extract_content, search_page
- click_element, fill_input, wait_for_element
- list_sessions tool
- Config file and /settings integration
- Basic test coverage

### Phase 2: Advanced Features (Future)
- Scriptable workflows (agent writes Playwright code)
- Screenshot capabilities
- Network interception
- Multi-page/tab support
- Enhanced error handling with screenshots
- Performance monitoring
- Advanced selectors (XPath, role-based)

---

## Success Metrics

### Adoption
- 40%+ of users enable browser features
- 30%+ create at least one browser session per week
- 20%+ use browser for application testing
- 50%+ use browser for knowledge access

### Quality
- 90%+ navigation success rate
- 85%+ element interaction success rate
- <5% session crashes
- 95%+ accurate content extraction

### Performance
- Session creation <2 seconds
- Page navigation <3 seconds average
- Content extraction <1 second
- Tool response time <500ms

### User Satisfaction
- 4.5+ rating for browser features
- "Game changer for testing" feedback
- "Never leave terminal" comments
- "Better than Copilot for web research" reviews

---

## Open Questions

1. **Playwright Installation**: Should we bundle Playwright browsers or require separate install?
   - **Leaning toward**: Separate install (keeps Forge binary small, user controls versions)

2. **Session Persistence**: Should sessions survive agent restarts?
   - **Decision**: No for MVP (in-memory only), add disk persistence in Phase 2 if needed

3. **Tool Naming**: Should tools be prefixed (e.g., `browser_navigate` vs `navigate`)?
   - **Decision**: Prefix with `browser_` to avoid conflicts and make purpose clear

4. **Error Screenshots**: Should we auto-screenshot on errors even though screenshots are Phase 2?
   - **Leaning toward**: Yes, save to temp directory for debugging, don't show to agent

5. **Multi-session Strategy**: How should agent decide when to create new sessions vs reuse?
   - **Decision**: Agent decides based on context (separate tasks = separate sessions)

---

## Related Features

- **Code Execution Sandbox**: Both provide isolated execution environments
- **Docker Integration**: Could run browser in container for better isolation
- **Testing Automation**: Browser is core to automated testing workflows
- **Documentation Knowledge**: Browser enables real-time doc access
- **Web Search**: Natural extension of browser capabilities

---

## Research & Resources

### Playwright Go Library
- **Repo**: https://github.com/playwright-community/playwright-go
- **Docs**: https://playwright.dev/docs/intro
- **License**: Apache 2.0
- **Status**: Active, community-maintained
- **API Coverage**: ~95% of Playwright features

### Similar Implementations
- **Browser Use (Python)**: https://github.com/browser-use/browser-use
  - Anthropic's computer use with Playwright
  - Good reference for patterns
- **Playwright Python**: https://playwright.dev/python/
  - Official Python bindings
  - Mature API design
- **Puppeteer Go**: https://github.com/chromedp/chromedp
  - Alternative, Chrome DevTools Protocol
  - Lower-level than Playwright

### Best Practices
- Token bucket for rate limiting sessions
- Graceful shutdown handling
- Context managers for resource cleanup
- Retry logic for flaky operations
- Structured logging for debugging

---

## Next Steps

1. **Validate Approach**: Review this document with team
2. **Prototype**: Build minimal session manager + navigate tool
3. **Test Integration**: Verify Playwright Go works in Forge environment
4. **Refine Design**: Iterate based on prototype learnings
5. **Create PRD**: Promote to full PRD when design is solid
6. **Create ADR**: Document technical decisions
7. **Implementation**: Build Phase 1 features
8. **Beta Testing**: Get user feedback on MVP
9. **Iterate**: Improve based on real usage
10. **Phase 2 Planning**: Decide on advanced features based on demand
