package browser

import (
	"fmt"
	"time"

	"github.com/playwright-community/playwright-go"
)

// UpdateLastUsed updates the LastUsedAt timestamp to the current time.
func (s *Session) UpdateLastUsed() {
	s.LastUsedAt = time.Now()
}

// Navigate navigates the session's page to the specified URL.
func (s *Session) Navigate(url string, opts NavigateOptions) error {
	s.UpdateLastUsed()

	// Build Playwright navigation options
	playwrightOpts := playwright.PageGotoOptions{}

	if opts.WaitUntil != "" {
		waitUntil := playwright.WaitUntilState(opts.WaitUntil)
		playwrightOpts.WaitUntil = &waitUntil
	}

	if opts.Timeout > 0 {
		playwrightOpts.Timeout = &opts.Timeout
	}

	// Navigate
	_, err := s.Page.Goto(url, playwrightOpts)
	if err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	// Update current URL
	s.CurrentURL = s.Page.URL()
	return nil
}

// ExtractContent extracts page content in the specified format.
func (s *Session) ExtractContent(opts ExtractOptions) (string, error) {
	s.UpdateLastUsed()

	// Set defaults
	if opts.Format == "" {
		opts.Format = FormatMarkdown
	}
	if opts.MaxLength == 0 {
		opts.MaxLength = DefaultMaxLength
	}

	switch opts.Format {
	case FormatMarkdown:
		return s.extractMarkdown(opts)
	case FormatText:
		return s.extractText(opts)
	case FormatStructured:
		return s.extractStructured(opts)
	default:
		return "", fmt.Errorf("unsupported format: %s", opts.Format)
	}
}

// extractText extracts plain text content from the page or selector.
func (s *Session) extractText(opts ExtractOptions) (string, error) {
	var content string

	if opts.Selector != "" {
		// Extract from specific element using locator-based API
		locator := s.Page.Locator(opts.Selector)
		count, err := locator.Count()
		if err != nil {
			return "", fmt.Errorf("selector query failed: %w", err)
		}
		if count == 0 {
			return "", fmt.Errorf("no element found matching selector: %s", opts.Selector)
		}
		content, err = locator.First().TextContent()
		if err != nil {
			return "", fmt.Errorf("text extraction failed: %w", err)
		}
	} else {
		// Extract from body using locator-based API
		bodyLocator := s.Page.Locator("body")
		count, err := bodyLocator.Count()
		if err != nil {
			return "", fmt.Errorf("body query failed: %w", err)
		}
		if count == 0 {
			return "", fmt.Errorf("no body element found")
		}
		content, err = bodyLocator.TextContent()
		if err != nil {
			return "", fmt.Errorf("text extraction failed: %w", err)
		}
	}

	// Truncate if needed
	if len(content) > opts.MaxLength {
		truncated := content[:opts.MaxLength]
		warning := fmt.Sprintf("\n\n[Content truncated: %d of %d characters shown]", opts.MaxLength, len(content))
		return truncated + warning, nil
	}

	return content, nil
}

// extractMarkdown extracts content and converts it to Markdown format.
func (s *Session) extractMarkdown(opts ExtractOptions) (string, error) {
	// For MVP, we'll use a simplified markdown extraction
	// This can be enhanced later with better formatting

	var markdown string

	// Get page title
	title, err := s.Page.Title()
	if err == nil && title != "" {
		markdown = fmt.Sprintf("# %s\n\n", title)
	}

	// Get main content
	text, err := s.extractText(opts)
	if err != nil {
		return "", err
	}

	markdown += text
	return markdown, nil
}

// extractStructured extracts content in structured JSON format.
func (s *Session) extractStructured(opts ExtractOptions) (string, error) {
	structured := StructuredContent{}

	// Get title
	title, err := s.Page.Title()
	if err == nil {
		structured.Title = title
	}

	// Get headings using locator-based API
	headingLocator := s.Page.Locator("h1, h2, h3, h4, h5, h6")
	count, err := headingLocator.Count()
	if err == nil && count > 0 {
		for i := 0; i < count; i++ {
			text, textErr := headingLocator.Nth(i).TextContent()
			if textErr == nil && text != "" {
				structured.Headings = append(structured.Headings, text)
			}
		}
	}

	// Get links using locator-based API
	linkLocator := s.Page.Locator("a[href]")
	linkCount, err := linkLocator.Count()
	if err == nil && linkCount > 0 {
		for i := 0; i < linkCount; i++ {
			linkElem := linkLocator.Nth(i)
			text, _ := linkElem.TextContent()
			href, _ := linkElem.GetAttribute("href")
			if href != "" {
				structured.Links = append(structured.Links, Link{
					Text: text,
					Href: href,
				})
			}
		}
	}

	// Get body text
	bodyText, err := s.extractText(opts)
	if err == nil {
		structured.Body = bodyText
	}

	// Convert to JSON
	// Note: In real implementation, use json.Marshal
	// For now, return a formatted string
	result := fmt.Sprintf(`{
  "title": %q,
  "headings": %v,
  "links": %d links,
  "body": %q
}`, structured.Title, len(structured.Headings), len(structured.Links),
		truncateString(structured.Body, opts.MaxLength))

	return result, nil
}

// Click clicks an element matching the selector.
func (s *Session) Click(opts ClickOptions) error {
	s.UpdateLastUsed()

	locator := s.Page.Locator(opts.Selector)
	
	playwrightOpts := playwright.LocatorClickOptions{}

	if opts.Button != "" {
		button := playwright.MouseButton(opts.Button)
		playwrightOpts.Button = &button
	}

	if opts.ClickCount > 0 {
		playwrightOpts.ClickCount = &opts.ClickCount
	}

	if opts.Timeout > 0 {
		playwrightOpts.Timeout = &opts.Timeout
	}

	err := locator.Click(playwrightOpts)
	if err != nil {
		return fmt.Errorf("click failed: %w", err)
	}

	// Update current URL in case click caused navigation
	s.CurrentURL = s.Page.URL()
	return nil
}

// Fill fills an input element with the specified value.
func (s *Session) Fill(opts FillOptions) error {
	s.UpdateLastUsed()

	locator := s.Page.Locator(opts.Selector)
	
	playwrightOpts := playwright.LocatorFillOptions{}

	if opts.Timeout > 0 {
		playwrightOpts.Timeout = &opts.Timeout
	}

	err := locator.Fill(opts.Value, playwrightOpts)
	if err != nil {
		return fmt.Errorf("fill failed: %w", err)
	}

	return nil
}

// Wait waits for an element or condition.
func (s *Session) Wait(opts WaitOptions) error {
	s.UpdateLastUsed()

	if opts.Selector == "" {
		return fmt.Errorf("selector is required for wait")
	}

	locator := s.Page.Locator(opts.Selector)
	
	playwrightOpts := playwright.LocatorWaitForOptions{}

	if opts.State != "" {
		state := playwright.WaitForSelectorState(opts.State)
		playwrightOpts.State = &state
	}

	if opts.Timeout > 0 {
		playwrightOpts.Timeout = &opts.Timeout
	}

	err := locator.WaitFor(playwrightOpts)
	if err != nil {
		return fmt.Errorf("wait failed: %w", err)
	}

	return nil
}

// Search searches the page for text matching the pattern.
func (s *Session) Search(opts SearchOptions) ([]SearchResult, error) {
	s.UpdateLastUsed()

	// For MVP, do a simple text search in the body content
	bodyText, err := s.extractText(ExtractOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get page text: %w", err)
	}

	// Simple substring search (can be enhanced with regex later)
	var results []SearchResult
	searchText := opts.Pattern
	if !opts.CaseSensitive {
		bodyText = toLowerCase(bodyText)
		searchText = toLowerCase(searchText)
	}

	// Find all occurrences
	index := 0
	for {
		pos := indexString(bodyText[index:], searchText)
		if pos == -1 {
			break
		}

		actualPos := index + pos

		// Extract context (50 chars before and after)
		contextStart := max(0, actualPos-50)
		contextEnd := min(len(bodyText), actualPos+len(searchText)+50)
		context := bodyText[contextStart:contextEnd]

		results = append(results, SearchResult{
			Text:    bodyText[actualPos : actualPos+len(searchText)],
			Context: context,
		})

		index = actualPos + len(searchText)

		// Limit results
		if opts.MaxResults > 0 && len(results) >= opts.MaxResults {
			break
		}
	}

	return results, nil
}

// GetMetadata returns current page metadata.
func (s *Session) GetMetadata() (map[string]string, error) {
	s.UpdateLastUsed()

	title, err := s.Page.Title()
	if err != nil {
		title = ""
	}

	return map[string]string{
		"title": title,
		"url":   s.Page.URL(),
	}, nil
}

// Helper functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func toLowerCase(s string) string {
	// Simple ASCII lowercase (can use strings.ToLower for full Unicode)
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func indexString(s, substr string) int {
	// Simple substring search
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
