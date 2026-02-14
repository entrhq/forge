package browser

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/agent/tools"
)

// SearchTool searches for text in the current page.
type SearchTool struct {
	manager *SessionManager
}

// NewSearchTool creates a new search tool.
func NewSearchTool(manager *SessionManager) *SearchTool {
	return &SearchTool{
		manager: manager,
	}
}

// Name returns the tool name.
func (t *SearchTool) Name() string {
	return "browser_search"
}

// Description returns the tool description.
func (t *SearchTool) Description() string {
	return "Search for text patterns in the current page content. Returns matching text with surrounding context."
}

// Schema returns the tool's JSON schema.
func (t *SearchTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"session": map[string]interface{}{
				"type":        "string",
				"description": "Name of the browser session to search in",
			},
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Text pattern to search for in the page content",
			},
			"case_sensitive": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether the search should be case-sensitive. Default: false",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return. Default: 10",
			},
		},
		[]string{"session", "pattern"},
	)
}

// SearchInput represents the parameters for searching.
type SearchInput struct {
	XMLName       xml.Name `xml:"arguments"`
	Session       string   `xml:"session"`
	Pattern       string   `xml:"pattern"`
	CaseSensitive *bool    `xml:"case_sensitive"`
	MaxResults    *int     `xml:"max_results"`
}

// Execute searches the page.
func (t *SearchTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Parse parameters
	var input SearchInput
	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate parameters
	if input.Session == "" {
		return "", nil, fmt.Errorf("session name is required")
	}
	if input.Pattern == "" {
		return "", nil, fmt.Errorf("search pattern is required")
	}

	// Get session
	session, err := t.manager.GetSession(input.Session)
	if err != nil {
		return "", nil, err
	}

	// Build search options
	opts := SearchOptions{
		Pattern:       input.Pattern,
		CaseSensitive: false, // Default
		MaxResults:    10,    // Default
	}

	if input.CaseSensitive != nil {
		opts.CaseSensitive = *input.CaseSensitive
	}

	if input.MaxResults != nil {
		if *input.MaxResults < 1 || *input.MaxResults > 100 {
			return "", nil, fmt.Errorf("max_results must be between 1 and 100")
		}
		opts.MaxResults = *input.MaxResults
	}

	// Search the page
	results, err := session.Search(opts)
	if err != nil {
		return "", nil, err
	}

	// Build result message
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf(`Search completed successfully

Search Details:
- Session: %s
- Pattern: "%s"
- Case Sensitive: %v
- Results Found: %d
- Current URL: %s

`,
		input.Session,
		input.Pattern,
		opts.CaseSensitive,
		len(results),
		session.CurrentURL,
	))

	if len(results) == 0 {
		resultText.WriteString("No matches found for the search pattern.")
	} else {
		resultText.WriteString("Matches:\n\n")
		for i, result := range results {
			resultText.WriteString(fmt.Sprintf("Match %d:\n", i+1))
			resultText.WriteString(fmt.Sprintf("Text: %q\n", result.Text))
			resultText.WriteString(fmt.Sprintf("Context: %s\n\n", result.Context))
		}

		if len(results) == opts.MaxResults {
			resultText.WriteString(fmt.Sprintf("\n[Limited to %d results. There may be more matches in the page.]", opts.MaxResults))
		}
	}

	return resultText.String(), nil, nil
}

// IsLoopBreaking returns whether this tool breaks the agent loop.
func (t *SearchTool) IsLoopBreaking() bool {
	return false
}

// ShouldShow returns whether this tool should be visible.
// Search tools are only shown when there are active sessions.
func (t *SearchTool) ShouldShow() bool {
	return t.manager.HasSessions()
}
