package browser

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/types"
)

// AnalyzePageTool uses LLM analysis to provide intelligent page summaries.
type AnalyzePageTool struct {
	manager  *SessionManager
	provider llm.Provider
}

// NewAnalyzePageTool creates a new analyze page tool.
func NewAnalyzePageTool(manager *SessionManager, provider llm.Provider) *AnalyzePageTool {
	return &AnalyzePageTool{
		manager:  manager,
		provider: provider,
	}
}

// Name returns the tool name.
func (t *AnalyzePageTool) Name() string {
	return "analyze_page"
}

// Description returns the tool description.
func (t *AnalyzePageTool) Description() string {
	return "Analyze the current page using AI to understand its purpose, key elements, and suggest relevant actions. Returns a structured analysis including page type, purpose, main content areas, and next steps."
}

// Schema returns the JSON schema for this tool's parameters.
func (t *AnalyzePageTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"session": map[string]interface{}{
				"type":        "string",
				"description": "Name of the browser session to analyze",
			},
			"focus": map[string]interface{}{
				"type":        "string",
				"description": "Optional: What to focus on in the analysis (e.g., 'forms', 'navigation', 'data extraction')",
			},
		},
		"required": []string{"session"},
	}
}

// analyzePageInput defines the input parameters.
type analyzePageInput struct {
	XMLName xml.Name `xml:"arguments"`
	Session string   `xml:"session"`
	Focus   string   `xml:"focus"`
}

// Execute analyzes the current page using LLM.
func (t *AnalyzePageTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Check if LLM provider is available first
	if t.provider == nil {
		return "", nil, fmt.Errorf("LLM provider not available")
	}

	// Parse input
	var input analyzePageInput
	if err := xml.Unmarshal(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("failed to parse arguments: %w", err)
	}

	if input.Session == "" {
		return "", nil, fmt.Errorf("session parameter is required")
	}

	// Get the session
	session, err := t.manager.GetSession(input.Session)
	if err != nil {
		return "", nil, fmt.Errorf("session not found: %s", input.Session)
	}

	// Extract content as cleaned HTML to preserve semantic structure
	extractOpts := ExtractOptions{
		Format:    FormatHTML,
		MaxLength: 50000, // Large enough for analysis but not unlimited
	}
	htmlContent, err := session.ExtractContent(extractOpts)
	if err != nil {
		return "", nil, fmt.Errorf("failed to extract content: %w", err)
	}

	// Get current URL and title for context
	currentURL := session.Page.URL()
	title, _ := session.Page.Title()

	// Build analysis prompt with HTML content
	prompt := buildAnalysisPrompt(currentURL, title, htmlContent, input.Focus)

	// Call LLM for analysis
	analysis, err := t.callLLMForAnalysis(ctx, prompt)
	if err != nil {
		return "", nil, fmt.Errorf("failed to analyze page: %w", err)
	}

	// Format result
	result := fmt.Sprintf(`Page Analysis Complete

URL: %s
Title: %s
Session: %s

%s

This analysis was generated using AI to help you understand the page structure and identify relevant next steps.`,
		currentURL,
		title,
		input.Session,
		analysis,
	)

	return result, nil, nil
}

// buildAnalysisPrompt creates the analysis prompt for the LLM.
func buildAnalysisPrompt(url, title, htmlContent, focus string) string {
	var prompt strings.Builder

	prompt.WriteString("Analyze the following web page and provide a structured summary. The content is cleaned HTML that preserves semantic structure and key targeting attributes.\n\n")
	prompt.WriteString(fmt.Sprintf("URL: %s\n", url))
	prompt.WriteString(fmt.Sprintf("Title: %s\n\n", title))

	if focus != "" {
		prompt.WriteString(fmt.Sprintf("Analysis Focus: %s\n\n", focus))
	}

	prompt.WriteString("Page HTML (cleaned, with semantic structure and targeting attributes):\n")
	prompt.WriteString("```html\n")
	prompt.WriteString(htmlContent)
	prompt.WriteString("\n```\n\n")

	prompt.WriteString("Provide a structured analysis with the following sections:\n\n")
	prompt.WriteString("1. PAGE TYPE: Identify the type of page and framework/platform based on HTML structure\n")
	prompt.WriteString("2. PURPOSE: Brief description of the page's main purpose and target audience\n")
	prompt.WriteString("3. KEY ELEMENTS &amp; SELECTORS: List important interactive elements with suggested CSS selectors (using id, class, data-* attributes)\n")
	prompt.WriteString("4. SEMANTIC STRUCTURE: Describe the DOM hierarchy (header, nav, main, sections, footer) and content organization\n")
	prompt.WriteString("5. INTERACTION OPPORTUNITIES: Identify key actions the user can take (navigation, form submission, clicking elements) with specific selectors\n\n")

	if focus != "" {
		prompt.WriteString(fmt.Sprintf("Focus your analysis on: %s\n\n", focus))
	}

	prompt.WriteString("Keep the analysis concise and actionable. Include specific selectors for targeting elements. Format as plain text with clear section headers.")

	return prompt.String()
}

// GeneratePreview implements the Previewable interface to show analyze page details before execution.
func (t *AnalyzePageTool) GeneratePreview(ctx context.Context, argsXML []byte) (*tools.ToolPreview, error) {
	var input analyzePageInput
	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.Session == "" {
		return nil, fmt.Errorf("session name is required")
	}

	description := fmt.Sprintf("This will analyze the current page in browser session '%s' using AI", input.Session)
	if input.Focus != "" {
		description += fmt.Sprintf(" with focus on: %s", input.Focus)
	}

	content := fmt.Sprintf("Session: %s\n\n", input.Session)
	if input.Focus != "" {
		content += fmt.Sprintf("Focus: %s\n\n", input.Focus)
	}
	content += "The AI will analyze the page structure, identify key elements, and suggest relevant actions."

	return &tools.ToolPreview{
		Type:        tools.PreviewTypeCommand,
		Title:       "Analyze Page with AI",
		Description: description,
		Content:     content,
		Metadata: map[string]interface{}{
			"session": input.Session,
			"focus":   input.Focus,
		},
	}, nil
}

// callLLMForAnalysis calls the LLM provider to analyze the page.
func (t *AnalyzePageTool) callLLMForAnalysis(ctx context.Context, prompt string) (string, error) {
	if t.provider == nil {
		return "", fmt.Errorf("LLM provider not available")
	}

	// Create messages for the LLM
	messages := []*types.Message{
		types.NewUserMessage(prompt),
	}

	// Call the provider
	response, err := t.provider.Complete(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	return response.Content, nil
}

// ShouldShow returns whether this tool should be visible.
// Only shown when browser automation is enabled and sessions exist.
func (t *AnalyzePageTool) ShouldShow() bool {
	if !config.IsInitialized() {
		return false
	}

	ui := config.GetUI()
	if !ui.IsBrowserEnabled() {
		return false
	}

	// Only show when there are active sessions
	return t.manager.HasSessions()
}

// IsLoopBreaking returns false as this is an operational tool.
func (t *AnalyzePageTool) IsLoopBreaking() bool {
	return false
}
