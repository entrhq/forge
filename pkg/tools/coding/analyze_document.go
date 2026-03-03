package coding

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/security/workspace"
	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// AnalyzeDocumentTool analyzes PNG, JPG, or PDF documents using LLM vision capabilities.
type AnalyzeDocumentTool struct {
	guard    *workspace.Guard
	provider llm.Provider
}

// NewAnalyzeDocumentTool creates a new document analysis tool.
func NewAnalyzeDocumentTool(guard *workspace.Guard, provider llm.Provider) *AnalyzeDocumentTool {
	return &AnalyzeDocumentTool{
		guard:    guard,
		provider: provider,
	}
}

// Name returns the tool name.
func (t *AnalyzeDocumentTool) Name() string {
	return "analyze_document"
}

// Description returns the tool description.
func (t *AnalyzeDocumentTool) Description() string {
	return "Analyze a PNG, JPG, or PDF document using AI vision. For PDFs, can analyze specific page ranges or all pages. Returns structured analysis optimized for agent consumption."
}

// Schema returns the JSON schema for this tool's parameters.
func (t *AnalyzeDocumentTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the document file (relative to workspace). Supported formats: .png, .jpg, .jpeg, .pdf",
			},
			"page_start": map[string]interface{}{
				"type":        "integer",
				"description": "Optional: Starting page number for PDF analysis (1-based). Use 0 or omit for all pages (up to pdf_page_limit).",
			},
			"page_end": map[string]interface{}{
				"type":        "integer",
				"description": "Optional: Ending page number for PDF analysis (1-based, inclusive). Use 0 or omit to analyze from page_start to end (up to pdf_page_limit).",
			},
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "Optional: Specific analysis instructions or questions about the document",
			},
		},
		[]string{"path"},
	)
}

// analyzeDocumentInput defines the input parameters.
type analyzeDocumentInput struct {
	XMLName   xml.Name `xml:"arguments"`
	Path      string   `xml:"path"`
	PageStart int      `xml:"page_start"`
	PageEnd   int      `xml:"page_end"`
	Prompt    string   `xml:"prompt"`
}

// Execute analyzes the document using LLM vision.
func (t *AnalyzeDocumentTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	// Check if LLM provider is available
	if t.provider == nil {
		return "", nil, fmt.Errorf("LLM provider not available")
	}

	// Parse input
	var input analyzeDocumentInput
	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if input.Path == "" {
		return "", nil, fmt.Errorf("missing required parameter: path")
	}

	// Validate path with workspace guard
	if err := t.guard.ValidatePath(input.Path); err != nil {
		return "", nil, fmt.Errorf("invalid path: %w", err)
	}

	// Resolve to absolute path
	absPath, err := t.guard.ResolvePath(input.Path)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check if file is ignored
	if t.guard.ShouldIgnore(absPath) {
		return "", nil, fmt.Errorf("file '%s' is ignored by .gitignore, .forgeignore, or default patterns", input.Path)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); err != nil {
		return "", nil, fmt.Errorf("file not found: %s", input.Path)
	}

	// Determine file type from extension
	ext := strings.ToLower(filepath.Ext(absPath))
	switch ext {
	case ".png", ".jpg", ".jpeg":
		return t.analyzeImage(ctx, absPath, input.Path, input.Prompt)
	case ".pdf":
		return t.analyzePDF(ctx, absPath, input.Path, input.PageStart, input.PageEnd, input.Prompt)
	default:
		return "", nil, fmt.Errorf("unsupported file format: %s (supported: .png, .jpg, .jpeg, .pdf)", ext)
	}
}

// analyzeImage analyzes a PNG or JPG image.
func (t *AnalyzeDocumentTool) analyzeImage(ctx context.Context, absPath, relPath, userPrompt string) (string, map[string]interface{}, error) {
	// Read image file
	imageData, err := os.ReadFile(absPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read image: %w", err)
	}

	// Determine media type
	ext := strings.ToLower(filepath.Ext(absPath))
	var mediaType string
	switch ext {
	case ".png":
		mediaType = "image/png"
	case ".jpg", ".jpeg":
		mediaType = "image/jpeg"
	}

	// Build analysis prompt
	prompt := buildImageAnalysisPrompt(relPath, userPrompt)

	// Get provider for analysis
	provider := t.providerForAnalysis()

	// Call LLM with multimodal content
	analysisResult, err := provider.AnalyzeDocument(ctx, imageData, mediaType, prompt)
	if err != nil {
		return "", nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	// Format result
	result := fmt.Sprintf(`Document Analysis Complete

Document Type: Image (%s)
Path: %s

%s`, ext, relPath, analysisResult)

	metadata := map[string]interface{}{
		"path":          relPath,
		"document_type": "image",
		"format":        ext,
	}

	return result, metadata, nil
}

// analyzePDF analyzes a PDF document with optional page range.
func (t *AnalyzeDocumentTool) analyzePDF(ctx context.Context, absPath, relPath string, pageStart, pageEnd int, userPrompt string) (string, map[string]interface{}, error) {
	// Get PDF page count
	pageCount, err := api.PageCountFile(absPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read PDF: %w", err)
	}

	// Get PDF page limit from config
	var pdfPageLimit int
	if config.IsInitialized() {
		if multimodal := config.GetMultimodal(); multimodal != nil {
			pdfPageLimit = multimodal.GetPDFPageLimit()
		}
	}
	if pdfPageLimit == 0 {
		pdfPageLimit = pageCount // 0 = all pages
	}

	// Determine actual page range
	actualStart, actualEnd := t.calculatePageRange(pageStart, pageEnd, pageCount, pdfPageLimit)

	// Validate page range
	if actualStart < 1 || actualStart > pageCount {
		return "", nil, fmt.Errorf("invalid page range: start page %d exceeds document length (%d pages)", actualStart, pageCount)
	}
	if actualEnd < actualStart {
		return "", nil, fmt.Errorf("invalid page range: end page %d is before start page %d", actualEnd, actualStart)
	}
	if actualEnd > pageCount {
		actualEnd = pageCount
	}

	// Build analysis prompt
	pagesAnalyzed := actualEnd - actualStart + 1
	pagesRemaining := pageCount - actualEnd
	prompt := buildPDFAnalysisPrompt(relPath, pageCount, actualStart, actualEnd, pagesAnalyzed, pagesRemaining, userPrompt)

	// Read PDF file bytes (trim to selected pages if needed)
	var pdfBytes []byte
	if actualStart == 1 && actualEnd == pageCount {
		// Analyze entire PDF - read directly
		pdfBytes, err = os.ReadFile(absPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to read PDF file: %w", err)
		}
	} else {
		// Extract selected page range to temporary file
		var trimErr error
		pdfBytes, trimErr = t.extractPDFPages(absPath, actualStart, actualEnd)
		if trimErr != nil {
			return "", nil, trimErr
		}
	}

	// Get provider for analysis
	provider := t.providerForAnalysis()

	// Call LLM with PDF content
	analysisResult, err := provider.AnalyzeDocument(ctx, pdfBytes, "application/pdf", prompt)
	if err != nil {
		return "", nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	// Format result with pagination metadata
	result := fmt.Sprintf(`Document Analysis Complete

Document Type: PDF
Path: %s
Total Pages: %d
Analyzed: Pages %d-%d (%d pages remaining)

%s`, relPath, pageCount, actualStart, actualEnd, pagesRemaining, analysisResult)

	metadata := map[string]interface{}{
		"path":            relPath,
		"document_type":   "pdf",
		"total_pages":     pageCount,
		"analyzed_start":  actualStart,
		"analyzed_end":    actualEnd,
		"pages_analyzed":  pagesAnalyzed,
		"pages_remaining": pagesRemaining,
	}

	return result, metadata, nil
}

// extractPDFPages extracts a page range from a PDF file and returns the bytes.
func (t *AnalyzeDocumentTool) extractPDFPages(absPath string, startPage, endPage int) ([]byte, error) {
	tmpFile, tmpErr := os.CreateTemp("", "forge-pdf-*.pdf")
	if tmpErr != nil {
		return nil, fmt.Errorf("failed to create temporary file: %w", tmpErr)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Build page range string for pdfcpu (e.g., "5-10")
	pageRange := fmt.Sprintf("%d-%d", startPage, endPage)
	if trimErr := api.TrimFile(absPath, tmpPath, []string{pageRange}, nil); trimErr != nil {
		return nil, fmt.Errorf("failed to extract pages %d-%d: %w", startPage, endPage, trimErr)
	}

	// Read trimmed PDF bytes
	pdfBytes, readErr := os.ReadFile(tmpPath)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read trimmed PDF: %w", readErr)
	}

	return pdfBytes, nil
}

// calculatePageRange determines the actual page range to analyze based on inputs and limits.
func (t *AnalyzeDocumentTool) calculatePageRange(pageStart, pageEnd, pageCount, pdfPageLimit int) (int, int) {
	// If both are 0, use default limit from start
	if pageStart == 0 && pageEnd == 0 {
		start := 1
		end := pdfPageLimit
		// If limit is 0, analyze all pages
		if pdfPageLimit == 0 || end > pageCount {
			end = pageCount
		}
		return start, end
	}

	// If only pageStart is specified, apply limit from that start point
	if pageStart > 0 && pageEnd == 0 {
		start := pageStart
		end := pageStart + pdfPageLimit - 1
		if end > pageCount {
			end = pageCount
		}
		return start, end
	}

	// If both are specified, honor user's explicit range (ignoring limit)
	if pageStart > 0 && pageEnd > 0 {
		// Clamp to page count
		actualEnd := pageEnd
		if actualEnd > pageCount {
			actualEnd = pageCount
		}
		return pageStart, actualEnd
	}

	// Fallback (shouldn't reach here)
	return 1, pageCount
}

// buildImageAnalysisPrompt creates the analysis prompt for images.
func buildImageAnalysisPrompt(path, userPrompt string) string {
	var prompt strings.Builder

	prompt.WriteString("You are analyzing a visual document for an AI coding agent. Your analysis will be used by the agent to understand the document's content, structure, and purpose. Provide dense, structured information optimized for machine processing.\n\n")
	fmt.Fprintf(&prompt, "Document Path: %s\n", path)
	prompt.WriteString("Document Format: Image (PNG/JPG/JPEG)\n\n")

	if userPrompt != "" {
		fmt.Fprintf(&prompt, "Agent Query: %s\n\n", userPrompt)
		prompt.WriteString("Address the agent's query within the structured analysis below.\n\n")
	}

	prompt.WriteString("Provide a comprehensive analysis with the following structure:\n\n")
	prompt.WriteString("1. DOCUMENT TYPE & PURPOSE\n")
	prompt.WriteString("   - Classify the document (diagram, architecture sketch, UI mockup, flowchart, screenshot, design spec, wireframe, etc.)\n")
	prompt.WriteString("   - Identify the intended purpose and audience\n")
	prompt.WriteString("   - Note any framework, tool, or methodology being used (e.g., Excalidraw, Figma, hand-drawn)\n\n")

	prompt.WriteString("2. VISUAL STRUCTURE & LAYOUT\n")
	prompt.WriteString("   - Describe the spatial organization and layout patterns\n")
	prompt.WriteString("   - Identify distinct sections, regions, or hierarchical levels\n")
	prompt.WriteString("   - Note visual groupings, containers, or boundaries\n\n")

	prompt.WriteString("3. COMPONENTS & ELEMENTS\n")
	prompt.WriteString("   - List all significant visual elements with their positions and relationships\n")
	prompt.WriteString("   - Identify shapes, icons, symbols, and their meanings\n")
	prompt.WriteString("   - Describe connections, arrows, lines, and flow indicators\n")
	prompt.WriteString("   - Note colors, styling, or visual emphasis patterns\n\n")

	prompt.WriteString("4. TEXT CONTENT\n")
	prompt.WriteString("   - Extract ALL visible text, labels, annotations, and captions verbatim\n")
	prompt.WriteString("   - Preserve the hierarchical relationship of text elements\n")
	prompt.WriteString("   - Note text positioning relative to visual elements\n")
	prompt.WriteString("   - Include any URLs, code snippets, or technical notation\n\n")

	prompt.WriteString("5. RELATIONSHIPS & FLOWS\n")
	prompt.WriteString("   - Map connections between components (what flows to what)\n")
	prompt.WriteString("   - Identify hierarchies, dependencies, or sequences\n")
	prompt.WriteString("   - Describe interaction patterns or process flows\n")
	prompt.WriteString("   - Note any conditional logic or branching\n\n")

	prompt.WriteString("6. TECHNICAL INSIGHTS\n")
	prompt.WriteString("   - Extract architectural patterns, design decisions, or system boundaries\n")
	prompt.WriteString("   - Identify technologies, protocols, or standards referenced\n")
	prompt.WriteString("   - Note any constraints, requirements, or annotations\n")
	prompt.WriteString("   - Point out missing information or areas needing clarification\n\n")

	if userPrompt != "" {
		prompt.WriteString("7. RESPONSE TO AGENT QUERY\n")
		fmt.Fprintf(&prompt, "   - Directly address: %s\n", userPrompt)
		prompt.WriteString("   - Reference specific elements from the analysis above\n\n")
	}

	prompt.WriteString("FORMAT REQUIREMENTS:\n")
	prompt.WriteString("- Use plain text with clear section headers (no markdown)\n")
	prompt.WriteString("- Be information-dense and precise\n")
	prompt.WriteString("- Use bullet points and structured lists\n")
	prompt.WriteString("- Preserve all technical terminology and proper names exactly\n")
	prompt.WriteString("- Prioritize completeness over brevity - the agent needs full context")

	return prompt.String()
}

// buildPDFAnalysisPrompt creates the analysis prompt for PDFs.
func buildPDFAnalysisPrompt(path string, totalPages, startPage, endPage, analyzed, remaining int, userPrompt string) string {
	var prompt strings.Builder

	prompt.WriteString("You are analyzing a PDF document for an AI coding agent. Your analysis will be used by the agent to understand the document's content, structure, and purpose. Provide dense, structured information optimized for machine processing.\n\n")

	prompt.WriteString("DOCUMENT CONTEXT:\n")
	prompt.WriteString(fmt.Sprintf("- Path: %s\n", path))
	prompt.WriteString(fmt.Sprintf("- Total Pages: %d\n", totalPages))
	prompt.WriteString(fmt.Sprintf("- Current Analysis Scope: Pages %d-%d (%d pages)\n", startPage, endPage, analyzed))
	if remaining > 0 {
		prompt.WriteString(fmt.Sprintf("- Remaining Unanalyzed: %d pages\n", remaining))
	}
	prompt.WriteString("\n")

	if userPrompt != "" {
		prompt.WriteString(fmt.Sprintf("AGENT QUERY: %s\n\n", userPrompt))
		prompt.WriteString("Address the agent's query within the structured analysis below.\n\n")
	}

	prompt.WriteString("Provide a comprehensive analysis with the following structure:\n\n")

	prompt.WriteString("1. DOCUMENT TYPE & PURPOSE\n")
	prompt.WriteString("   - Classify the document (presentation slides, technical spec, design document, report, whitepaper, API docs, etc.)\n")
	prompt.WriteString("   - Identify the intended purpose and target audience\n")
	prompt.WriteString("   - Note any authoring tool or template patterns (PowerPoint, Keynote, LaTeX, etc.)\n\n")

	prompt.WriteString("2. STRUCTURAL OVERVIEW\n")
	prompt.WriteString("   - Describe the document's organizational structure (sections, chapters, slide sequence)\n")
	prompt.WriteString("   - Identify navigation patterns, headers, footers, and page numbering\n")
	prompt.WriteString("   - Map the high-level content flow across analyzed pages\n\n")

	prompt.WriteString("3. PAGE-BY-PAGE BREAKDOWN\n")
	prompt.WriteString("   For each page in the analyzed range:\n")
	prompt.WriteString("   - Page number and title/heading (if present)\n")
	prompt.WriteString("   - Main content type (text, diagrams, code, tables, images)\n")
	prompt.WriteString("   - Key topics or themes\n")
	prompt.WriteString("   - Visual elements and their purpose\n\n")

	prompt.WriteString("4. TEXT CONTENT EXTRACTION\n")
	prompt.WriteString("   - Extract all significant text verbatim (headings, body text, captions, labels)\n")
	prompt.WriteString("   - Preserve hierarchical structure (H1, H2, bullet points, etc.)\n")
	prompt.WriteString("   - Include any code snippets, commands, or technical notation exactly as shown\n")
	prompt.WriteString("   - Note URLs, references, or citations\n\n")

	prompt.WriteString("5. VISUAL ELEMENTS & DIAGRAMS\n")
	prompt.WriteString("   - Describe all diagrams, charts, graphs, or illustrations\n")
	prompt.WriteString("   - Identify diagram types (flowcharts, architecture diagrams, UML, ER diagrams, wireframes, etc.)\n")
	prompt.WriteString("   - Extract labels, annotations, and legend information\n")
	prompt.WriteString("   - Map relationships shown in visual elements\n\n")

	prompt.WriteString("6. TECHNICAL CONTENT\n")
	prompt.WriteString("   - Identify technologies, frameworks, APIs, or protocols mentioned\n")
	prompt.WriteString("   - Extract architectural patterns, design decisions, or system components\n")
	prompt.WriteString("   - Note configuration examples, data formats, or schemas\n")
	prompt.WriteString("   - List any requirements, constraints, or specifications\n\n")

	prompt.WriteString("7. RELATIONSHIPS & CONTINUITY\n")
	prompt.WriteString("   - Identify cross-page references or continued topics\n")
	prompt.WriteString("   - Note dependencies between sections\n")
	prompt.WriteString("   - Describe any narrative or argument progression\n\n")

	if userPrompt != "" {
		prompt.WriteString("8. RESPONSE TO AGENT QUERY\n")
		fmt.Fprintf(&prompt, "   - Directly address: %s\n", userPrompt)
		prompt.WriteString("   - Reference specific pages and elements from the analysis above\n")
		prompt.WriteString("   - Highlight relevant sections that answer the query\n\n")
	}

	if remaining > 0 {
		fmt.Fprintf(&prompt, "9. CONTINUATION CONTEXT\n")
		prompt.WriteString(fmt.Sprintf("   - %d pages remain unanalyzed (pages %d-%d)\n", remaining, endPage+1, totalPages))
		prompt.WriteString("   - Note any topics that appear to continue beyond the analyzed range\n")
		prompt.WriteString("   - Suggest whether additional pages would provide valuable context\n\n")
	}

	prompt.WriteString("FORMAT REQUIREMENTS:\n")
	prompt.WriteString("- Use plain text with clear section headers (no markdown)\n")
	prompt.WriteString("- Be information-dense and exhaustive\n")
	prompt.WriteString("- Use bullet points and structured lists\n")
	prompt.WriteString("- Preserve all technical terminology, proper names, and code exactly\n")
	prompt.WriteString("- Include page numbers for all references\n")
	prompt.WriteString("- Prioritize completeness over brevity - the agent needs full context")

	return prompt.String()
}

// providerForAnalysis returns the provider to use for document analysis.
// If multimodal.model is configured and the provider supports ModelCloner,
// returns a clone directed at that model. Otherwise returns t.provider.
func (t *AnalyzeDocumentTool) providerForAnalysis() llm.Provider {
	if config.IsInitialized() {
		if multimodal := config.GetMultimodal(); multimodal != nil {
			if model := multimodal.GetModel(); model != "" {
				if cloner, ok := t.provider.(llm.ModelCloner); ok {
					return cloner.CloneWithModel(model)
				}
			}
		}
	}
	return t.provider
}

// IsLoopBreaking returns false as this is an operational tool.
func (t *AnalyzeDocumentTool) IsLoopBreaking() bool {
	return false
}
