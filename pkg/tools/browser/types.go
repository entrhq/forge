package browser

import (
	"time"

	"github.com/playwright-community/playwright-go"
)

// Session represents an active browser session with its associated resources.
type Session struct {
	// Name is the unique identifier for this session
	Name string

	// Browser is the Playwright browser instance
	Browser playwright.Browser

	// Context is the browser context (isolated session)
	Context playwright.BrowserContext

	// Page is the current active page
	Page playwright.Page

	// Headless indicates if the browser is running in headless mode
	Headless bool

	// CreatedAt is the timestamp when the session was created
	CreatedAt time.Time

	// LastUsedAt is the timestamp of the last operation on this session
	LastUsedAt time.Time

	// CurrentURL is the URL of the current page
	CurrentURL string
}

// SessionOptions configures a new browser session.
type SessionOptions struct {
	// Headless controls whether the browser runs without a visible window
	Headless bool

	// Viewport sets the initial viewport size
	Viewport *Viewport

	// Timeout sets the default timeout for operations (in milliseconds)
	Timeout float64
}

// Viewport represents the browser viewport dimensions.
type Viewport struct {
	Width  int
	Height int
}

// NavigateOptions configures page navigation behavior.
type NavigateOptions struct {
	// WaitUntil specifies when to consider navigation successful
	// Valid values: "load", "domcontentloaded", "networkidle"
	WaitUntil string

	// Timeout in milliseconds (0 means default)
	Timeout float64
}

// ExtractFormat specifies the format for content extraction.
type ExtractFormat string

const (
	// FormatMarkdown extracts content as Markdown (default)
	FormatMarkdown ExtractFormat = "markdown"

	// FormatText extracts plain text only
	FormatText ExtractFormat = "text"

	// FormatStructured extracts content as structured JSON
	FormatStructured ExtractFormat = "structured"
)

// ExtractOptions configures content extraction.
type ExtractOptions struct {
	// Format specifies the extraction format
	Format ExtractFormat

	// Selector optionally limits extraction to matching elements
	Selector string

	// MaxLength limits the extracted content length (characters)
	MaxLength int
}

// StructuredContent represents content extracted in structured format.
type StructuredContent struct {
	Title    string         `json:"title"`
	Headings []string       `json:"headings"`
	Links    []Link         `json:"links"`
	Body     string         `json:"body"`
}

// Link represents a hyperlink with text and URL.
type Link struct {
	Text string `json:"text"`
	Href string `json:"href"`
}

// ClickOptions configures element clicking behavior.
type ClickOptions struct {
	// Selector identifies the element to click
	Selector string

	// Button specifies which mouse button to use (left, right, middle)
	Button string

	// ClickCount is the number of times to click (1 for single, 2 for double)
	ClickCount int

	// Timeout in milliseconds
	Timeout float64
}

// FillOptions configures form input filling.
type FillOptions struct {
	// Selector identifies the input element
	Selector string

	// Value is the text to fill
	Value string

	// Timeout in milliseconds
	Timeout float64
}

// WaitOptions configures waiting behavior.
type WaitOptions struct {
	// Selector to wait for (if waiting for element)
	Selector string

	// State to wait for: "attached", "detached", "visible", "hidden"
	State string

	// Timeout in milliseconds
	Timeout float64
}

// SearchOptions configures page search.
type SearchOptions struct {
	// Pattern is the text or regex pattern to search for
	Pattern string

	// CaseSensitive controls case-sensitive matching
	CaseSensitive bool

	// MaxResults limits the number of results returned
	MaxResults int
}

// SearchResult represents a single search match.
type SearchResult struct {
	Text    string `json:"text"`
	Context string `json:"context"`
}

// Default values for various operations
const (
	DefaultTimeout       = 30000.0 // 30 seconds in milliseconds
	DefaultMaxLength     = 10000   // 10,000 characters
	DefaultViewportWidth = 1280
	DefaultViewportHeight = 720
	DefaultMaxSessions   = 5
	DefaultIdleTimeout   = 300 // 5 minutes in seconds
)
