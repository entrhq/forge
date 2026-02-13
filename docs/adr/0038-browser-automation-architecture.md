# 38. Browser Automation with Playwright

**Status:** Proposed
**Date:** 2025-02-12
**Deciders:** Engineering Team
**Technical Story:** Enable agents to automate web browser interactions for testing, research, and data extraction tasks using Playwright.

---

## Context

Forge agents currently have strong capabilities for file operations, code editing, and command execution, but lack the ability to interact with web browsers. Many valuable use cases require browser automation:

1. **Testing**: End-to-end testing of web applications
2. **Research**: Extracting information from documentation sites or web pages
3. **Data Collection**: Gathering structured data from web sources
4. **Form Automation**: Filling forms and submitting data
5. **Screenshot Capture**: Visual testing and documentation

### Background

The existing tool system (ADR-0011, ADR-0019, ADR-0037) provides a clean abstraction for tool execution with XML-based calling conventions and support for both built-in and custom tools. The custom tools system enables agents to create persistent tools, but each tool is a separate subprocess with limited state management.

Browser automation requires maintaining persistent browser sessions across multiple operations, which doesn't fit the stateless tool model. Additionally, browser automation involves complex interactions like navigation, element waiting, JavaScript execution, and content extraction that benefit from a rich API rather than simple CLI flags.

### Problem Statement

How can we enable agents to perform browser automation with session management, rich interaction capabilities, and content extraction while maintaining security, efficiency, and integration with the existing tool system?

### Goals

- Enable persistent browser sessions that survive across multiple tool calls
- Support navigation, element interaction, content extraction, and screenshots
- Provide both headed (visible) and headless browser modes
- Maintain workspace isolation and security constraints
- Integrate seamlessly with existing tool system and approval flows
- Support common automation patterns (waiting, screenshots, form filling)
- Allow users to enable/disable browser automation via settings
- Support configurable default headless/headed mode preference

### Non-Goals

- Multi-browser support (focus on Chromium initially)
- Visual testing framework (future enhancement)
- Browser extension automation (future work)
- Mobile browser emulation (future work)
- Performance testing/profiling (future work)

---

## Decision Drivers

* **Session Management**: Need to maintain browser state across multiple operations
* **Security**: Must prevent unauthorized navigation or data access
* **Developer Experience**: Should be intuitive for agents to use
* **Performance**: Minimize browser startup overhead
* **Integration**: Must work with existing tool system architecture
* **Extensibility**: Should support future browser automation features

---

## Considered Options

### Option 1: Custom Tool per Browser Action

**Description:** Create separate custom tools for each browser action (navigate, click, extract, etc.) and manage sessions through environment variables or file-based state.

**Pros:**
- Fits existing custom tools pattern
- Each action is a discrete operation
- Easy to understand individual tools

**Cons:**
- Complex session management across tools
- High process spawning overhead
- Difficult to maintain browser state
- No type safety or compile-time validation
- Error-prone coordination between tools
- Cannot share browser instance across tools

### Option 2: Single Custom Tool with Subcommands

**Description:** Build one custom tool with subcommands (start, navigate, click, extract, close) and manage sessions through a daemon process or file-based registry.

**Pros:**
- Single entry point for browser automation
- Can manage sessions in one place
- Fewer tool definitions to maintain

**Cons:**
- Still spawns process per operation
- Daemon management adds complexity
- File-based session state is fragile
- Subcommand parsing adds boilerplate
- No benefits from Go type system

### Option 3: Built-in Browser Tool with Dynamic Subtools

**Description:** Implement browser automation as a built-in tool package in pkg/tools/browser/ with a session manager that dynamically loads/unloads tools based on active sessions.

**Pros:**
- Direct access to playwright-go library
- Efficient session management in-process
- Type-safe API with compile-time validation
- Can dynamically expose tools based on state
- Seamless integration with tool system
- No process spawning overhead
- Rich error handling and context

**Cons:**
- More complex implementation than custom tools
- Requires core Forge changes
- Tight coupling with Playwright library

### Option 4: Browser Service with REST API

**Description:** Create a separate browser service that agents communicate with via HTTP/REST calls.

**Pros:**
- Complete separation from Forge
- Could support multiple agents
- Language-agnostic interface

**Cons:**
- Significant architectural complexity
- Network overhead for every operation
- Service lifecycle management required
- Authentication/authorization needed
- Doesn't fit Forge's tool model
- Over-engineered for single-agent use case

---

## Decision

**Chosen Option:** Option 3 - Built-in Browser Tool with Dynamic Subtools

We will implement browser automation as a built-in tool package with a session manager that dynamically registers/unregisters tools based on active browser sessions.

### Rationale

This approach provides the best balance of functionality, performance, and integration:

1. **Efficient Session Management**: In-process session manager avoids file I/O and process coordination complexity
2. **Type Safety**: Direct use of playwright-go provides compile-time validation and rich API
3. **Dynamic Tool Loading**: Tools appear/disappear based on session state, guiding agent behavior
4. **Performance**: No process spawning overhead, browser instances stay warm
5. **Integration**: Works seamlessly with existing tool system and approval flows
6. **Extensibility**: Easy to add new browser capabilities as methods
7. **Security**: Workspace guard and approval flows apply naturally

The complexity of a built-in implementation is justified by the richness of browser automation and the poor fit with stateless custom tools.

---

## Consequences

### Positive

- Agents gain powerful web automation capabilities
- Session management is transparent and efficient
- Type-safe API reduces runtime errors
- Dynamic tool loading provides clear state feedback
- Performance is excellent (no process spawning)
- Easy to extend with new browser features
- Natural integration with approval system

### Negative

- Adds Playwright dependency to core Forge
- More complex than custom tools approach
- Session cleanup must be explicit or automatic
- Requires careful memory management for long-running sessions
- Browser crashes could affect Forge stability (mitigated by recovery)

### Neutral

- Browser automation tools only available when session is active
- Headed mode is default (can be overridden to headless)
- Sessions are named and isolated (no cross-session interference)

---

## Implementation

### Architecture

**Package Structure:**
```
pkg/tools/browser/
├── doc.go                  # Package documentation
├── session.go              # Session struct and lifecycle
├── manager.go              # SessionManager for global registry
├── start_session.go        # StartSessionTool (always available)
├── close_session.go        # CloseSessionTool (dynamic)
├── navigate.go             # NavigateTool (dynamic)
├── click.go                # ClickTool (dynamic)
├── extract_content.go      # ExtractContentTool (dynamic)
├── screenshot.go           # ScreenshotTool (dynamic)
├── fill_form.go            # FillFormTool (dynamic)
├── wait.go                 # WaitTool (dynamic)
└── *_test.go               # Comprehensive tests
```

**Core Components:**

1. **Session**: Encapsulates playwright.Browser, playwright.BrowserContext, playwright.Page
2. **SessionManager**: Global registry mapping session names to Session instances
3. **ToolRegistry Integration**: Dynamic tool loading based on active sessions
4. **StartSessionTool**: Creates new browser session, registers dynamic tools
5. **Dynamic Tools**: Navigation, interaction, extraction tools (only available with active session)

### Session Lifecycle

```go
type Session struct {
    Name        string
    Browser     playwright.Browser
    Context     playwright.BrowserContext
    Page        playwright.Page
    Headless    bool
    CreatedAt   time.Time
    LastUsedAt  time.Time
}

type SessionManager struct {
    mu       sync.RWMutex
    sessions map[string]*Session
    registry *tools.Registry
}

// Start creates session and registers dynamic tools
func (m *SessionManager) StartSession(name string, headless bool) error

// Close removes session and unregisters dynamic tools
func (m *SessionManager) CloseSession(name string) error

// Get retrieves active session
func (m *SessionManager) GetSession(name string) (*Session, error)

// List returns all active session names
func (m *SessionManager) ListSessions() []string
```

### Tool Definitions

All browser tools follow standard XML tool call format. Dynamic tools are only available after a session is started.

**Session Management:**
- start_browser_session: Creates new browser session with viewport options
- close_browser_session: Closes session and cleans up resources

**Navigation & Interaction:**
- browser_navigate: Navigate to URL with wait conditions
- browser_click: Click element by selector
- browser_fill_form: Fill form fields with values
- browser_wait: Wait for element, navigation, or timeout

**Content Extraction:**
- browser_extract_content: Extract text, HTML, or attributes from elements
- browser_screenshot: Capture full page or element screenshots

### Dynamic Tool Registration

When a session starts, the SessionManager automatically registers the dynamic tools:

```go
func (m *SessionManager) StartSession(name string, opts SessionOptions) error {
    // 1. Create Playwright browser/context/page
    session := &Session{...}
    
    // 2. Store in registry
    m.sessions[name] = session
    
    // 3. Register dynamic tools
    m.registerDynamicTools(name)
    
    return nil
}

func (m *SessionManager) registerDynamicTools(sessionName string) {
    tools := []tools.Tool{
        &NavigateTool{sessionName: sessionName},
        &ClickTool{sessionName: sessionName},
        &ExtractContentTool{sessionName: sessionName},
        &ScreenshotTool{sessionName: sessionName},
        &FillFormTool{sessionName: sessionName},
        &WaitTool{sessionName: sessionName},
        &CloseSessionTool{sessionName: sessionName},
    }
    
    for _, tool := range tools {
        m.registry.RegisterDynamicTool(tool)
    }
}
```

When a session closes, tools are automatically unregistered:

```go
func (m *SessionManager) CloseSession(name string) error {
    session := m.sessions[name]
    
    // 1. Cleanup Playwright resources
    session.Page.Close()
    session.Context.Close()
    session.Browser.Close()
    
    // 2. Remove from registry
    delete(m.sessions, name)
    
    // 3. Unregister dynamic tools
    m.unregisterDynamicTools(name)
    
    return nil
}
```

### Dynamic Tool Registration

When a session starts, the SessionManager automatically registers the dynamic tools. When a session closes, tools are automatically unregistered.

### Content Extraction

The ExtractContentTool provides multiple extraction modes:

**Text Extraction** (default):
- Extracts visible text from element
- Preserves basic structure (paragraphs, lists)
- Strips scripts and styles
- Limits to 10,000 characters with truncation

**HTML Extraction**:
- Returns outer HTML of matched element
- Useful for structure preservation
- Subject to same size limits

**Attribute Extraction**:
- Extracts specific attribute value
- Common for links (href), images (src), etc.

**Multi-Element Extraction**:
- When selector matches multiple elements
- Returns array of extracted values
- Useful for lists, tables, repeated structures

---

## Security Considerations

### Workspace Isolation

Browser sessions must respect workspace boundaries:

1. **Navigation Restrictions**: No file:// URLs outside workspace
2. **Download Directory**: Downloads restricted to workspace subdirectory
3. **Upload Files**: Only files within workspace can be uploaded
4. **Local Storage**: Isolated per session, cleaned on close

### Approval Flow

Browser tools follow standard approval requirements:

- **start_browser_session**: Requires approval (spawns browser process)
- **Navigation**: No approval required
- **Screenshots**: No approval required
- **All other operations**: Follow existing tool approval rules

### Resource Management

1. **Session Limits**: Maximum N concurrent sessions (default: 3)
2. **Memory Cleanup**: Automatic cleanup on agent exit
3. **Timeout Protection**: Sessions auto-close after inactivity period
4. **Browser Crashes**: Graceful recovery with error reporting

### Data Privacy

1. **No Persistent Cookies**: Each session starts fresh unless explicitly configured
2. **No Browser History**: History is not persisted between sessions
3. **Screenshot Storage**: Screenshots saved to workspace only, never temp dirs
4. **Form Data**: Not persisted, exists only during session

---

## Testing Strategy

### Unit Tests

- Session lifecycle management
- Tool registration/unregistration
- Parameter validation
- Error handling paths

### Integration Tests

- End-to-end browser automation workflows
- Multi-session scenarios
- Dynamic tool availability
- Session cleanup and recovery

### Security Tests

- Workspace boundary enforcement
- Path traversal prevention
- Resource limit validation
- Approval flow verification

---

## Future Enhancements

### Phase 2 (Post-MVP)

1. **Multi-Browser Support**: Firefox, Safari via Playwright
2. **Mobile Emulation**: Device simulation for responsive testing
3. **Network Interception**: Request/response modification
4. **PDF Generation**: Convert pages to PDF documents
5. **Geolocation/Permissions**: Control browser permissions

### Phase 3 (Advanced)

1. **Visual Testing**: Screenshot comparison and diff detection
2. **Performance Profiling**: Network timing, JS execution metrics
3. **Accessibility Testing**: ARIA validation, color contrast
4. **Browser Extensions**: Load custom extensions for testing
5. **Multi-Tab Support**: Handle multiple pages per session

---

## Configuration

Browser automation is controlled via UI configuration settings in `pkg/config/ui.go`:

### Settings

- `browser_enabled` (bool): Master toggle to enable/disable browser automation
  - Default: `false`
  - When disabled, browser tools are not registered
  - Prevents accidental browser usage when not needed

- `browser_headless` (bool): Default mode for new browser sessions
  - Default: `true` (headless mode)
  - Can be overridden per-session via `start_browser_session` tool
  - Headed mode useful for debugging and visual verification

### Configuration Access

```go
// Check if browser automation is enabled
if config.GetUI().IsBrowserEnabled() {
    // Register browser tools
}

// Get default headless mode
headless := config.GetUI().IsBrowserHeadless()
```

### Future: Settings UI

A `/settings` slash command could provide interactive configuration:
- Toggle browser automation on/off
- Set default headless/headed mode preference
- Configure session limits and timeouts
- Manage custom browser paths (future)

Settings persist in the config file and apply to all future sessions.

---

## Migration Path

This is a new capability with no migration required. Implementation plan:

1. **Phase 0**: Configuration infrastructure (✓ Complete)
   - Add browser settings to UISection
   - Implement getters/setters with thread safety
   - Add comprehensive test coverage

2. **Phase 1**: Core session management and basic tools (navigate, click, extract)
   - Conditional tool registration based on `browser_enabled` setting
   - Respect `browser_headless` default in session creation

3. **Phase 2**: Add screenshot, form filling, waiting capabilities
4. **Phase 3**: Documentation, examples, and integration tests
5. **Phase 4**: Security hardening and resource limits
6. **Phase 5**: Advanced features based on user feedback

---

## References

- ADR-0011: Tool System Architecture
- ADR-0019: XML Tool Calling Convention
- ADR-0037: Custom Tools System
- [Playwright Go Documentation](https://playwright.community/docs/intro)
- [Playwright API Reference](https://playwright.dev/docs/api/class-playwright)
