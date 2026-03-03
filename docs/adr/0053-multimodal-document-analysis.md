# 53. Multimodal Document Analysis

**Status:** Proposed
**Date:** 2025-01-20
**Deciders:** Engineering Team, Product Team
**Technical Story:** Enable Forge to analyze visual content (images and PDFs) to support excalidraw diagram generation workflow and general document understanding.

---

## Context

Forge currently operates exclusively with text-based content. As we build features like excalidraw diagram generation, the agent needs the ability to understand and reason about visual content - both for analyzing user-provided mockups and validating generated diagrams. Additionally, developers frequently work with PDF documentation containing diagrams, screenshots, and visual elements that are essential context for coding decisions.

### Background

Current limitations:
- **No visual analysis**: Agent cannot see diagrams, mockups, screenshots, or visual documentation
- **Manual transcription required**: Users must describe visual content in text
- **PDF limitations**: Can read text from PDFs but cannot process visual elements or understand layout
- **Excalidraw workflow gap**: Cannot validate generated diagrams by analyzing PNG exports
- **Documentation blind spot**: Technical PDFs with diagrams require manual extraction of visual information

The excalidraw diagram generation feature creates a specific need: the agent needs to analyze hand-drawn mockups to understand requirements and validate generated diagram PNGs to ensure accuracy.

### Problem Statement

How can we enable Forge to understand and reason about visual content (images and PDFs) efficiently, with smart pagination for large documents, and output optimized for agent consumption?

### Goals

- Support analysis of PNG/JPG images (diagrams, mockups, screenshots)
- Support analysis of PDF documents with multi-page handling
- Enable lazy pagination for large PDFs to optimize token usage
- Provide agent-optimized textual analysis output (not human-readable prose)
- Use existing LLM provider abstraction with user-selectable vision model
- Keep configuration simple and intuitive
- Maintain security model consistent with other file tools

### Non-Goals

- Multi-file batch analysis (too unwieldy, single file at a time only)
- Video or animated content support
- Real-time screen capture analysis
- Caching of analysis results (future enhancement)
- Screenshot-based visual layout analysis beyond content extraction
- Automated diagram generation from images (separate feature)

---

## Decision Drivers

* **Excalidraw Workflow**: Primary use case is analyzing mockups and validating generated diagrams
* **Token Efficiency**: Large PDFs should not consume excessive context - need smart pagination
* **Agent-Optimized Output**: Analysis should be structured for agent reasoning, not human readability
* **Provider Consistency**: Should use same LLM provider pattern as existing tools
* **User Control**: Users should control which vision model is used
* **Implementation Simplicity**: Leverage existing provider abstraction, minimal new infrastructure
* **Security**: Must respect workspace isolation and file access controls

---

## Considered Options

### Option 1: Custom Tool (External Binary)

**Description:** Build multimodal analysis as a custom tool (~/.forge/tools/) with external dependencies for image/PDF processing.

**Pros:**
- User can install/update independently
- Could use specialized libraries for PDF processing
- Doesn't increase core binary size
- Easier to iterate on without core changes

**Cons:**
- Requires installation step (friction)
- More complex provider integration (needs to call back to Forge's LLM provider)
- Versioning and compatibility challenges
- Less discoverable than built-in tools
- Security model unclear (tool has its own file access)
- Doesn't fit the "core capability" profile

### Option 2: Built-in Tool with Separate Provider

**Description:** Built-in tool in pkg/tools/ but with dedicated multimodal provider abstraction separate from main LLM provider.

**Pros:**
- Clear separation of concerns
- Could have different provider interface for multimodal
- Easier to swap provider implementations
- Dedicated error handling for multimodal failures

**Cons:**
- Unnecessary complexity - vision models are just LLM provider variants
- Duplicates provider configuration
- User has to manage two provider configs
- More code to maintain
- Doesn't match existing pattern (analyse_page uses main provider)

### Option 3: Built-in Tool with Main Provider (CHOSEN)

**Description:** Built-in tool in pkg/tools/ that uses existing llm.Provider interface, with user-selectable vision model in config.

**Pros:**
- Consistent with existing tools (analyse_page pattern)
- Leverages proven provider abstraction
- Single provider configuration model
- Users already understand model selection pattern
- Minimal new infrastructure needed
- Built-in tools are more discoverable
- Security model is well-established

**Cons:**
- Increases core binary size slightly
- Couples multimodal to provider interface evolution
- All users get the code even if they don't use it (acceptable tradeoff)

---

## Decision

**Chosen Option:** Option 3 - Built-in Tool with Main Provider

We will implement `analyze_document` as a built-in tool in pkg/tools/ that uses the existing llm.Provider interface with a user-configurable vision model, following the same pattern as the analyse_page tool.

### Rationale

1. **Proven Pattern**: The analyse_page tool already demonstrates this architecture working well - separate model selection for specialized tasks using the main provider
2. **User Simplicity**: Single provider configuration, users just select a different model for multimodal analysis
3. **Implementation Efficiency**: Reuses all existing provider infrastructure (error handling, retries, rate limits, streaming)
4. **Discoverability**: Built-in tools are always available, no installation friction
5. **Security Consistency**: Uses same workspace guard and file validation as other file tools
6. **Core Capability**: Visual analysis is fundamental enough to warrant built-in status

The key insight: Vision models are just LLM provider variants that accept image input. No need for separate abstraction.

---

## Consequences

### Positive

- Agents can analyze diagrams and PDFs without manual transcription
- Excalidraw workflow becomes feasible (analyze mockup → generate → validate output)
- Lazy pagination enables efficient exploration of large PDFs
- Agent-optimized output format ensures analysis is immediately actionable
- Consistent user experience with existing model selection pattern
- No new infrastructure needed beyond tool implementation
- Tool result includes clear pagination guidance for multi-page PDFs

### Negative

- Vision model API calls are more expensive than text-only (acceptable - user controls usage)
- Analysis quality depends on vision model capabilities (varies by provider)
- PDF processing requires page extraction logic (implementation complexity)
- No caching means repeated analysis costs tokens each time (future enhancement)
- Single file at a time may require multiple tool calls for related documents

### Neutral

- Adds another tool to the toolkit (but only used when needed)
- Configuration surface area increases slightly (one model setting)
- Different providers have different vision model support (documented limitation)
- Agents must learn when to use analyze_document vs. read_file

---

## Implementation

### Architecture

**New Tool: `analyze_document`**

```go
package tools

import (
    "context"
    "github.com/entrhq/forge/pkg/llm"
    "github.com/entrhq/forge/pkg/types"
)

type AnalyzeDocumentTool struct {
    provider      llm.Provider
    workspaceRoot string
    model         string // From config: multimodal.model
}

func NewAnalyzeDocumentTool(provider llm.Provider, workspaceRoot, model string) *AnalyzeDocumentTool {
    return &AnalyzeDocumentTool{
        provider:      provider,
        workspaceRoot: workspaceRoot,
        model:         model,
    }
}
```

**Tool Parameters:**
```xml
<tool>
<tool_name>analyze_document</tool_name>
<arguments>
  <path>diagrams/mockup.png</path>              <!-- Required: workspace-relative path -->
  <page_start>10</page_start>                    <!-- Optional: first page (PDFs only) -->
  <page_end>15</page_end>                        <!-- Optional: last page (PDFs only) -->
  <prompt>Focus on the login flow</prompt>      <!-- Optional: analysis instructions -->
</arguments>
</tool>
```

**Parameter Validation:**
- `path`: Required, must be within workspace, must exist, must be supported format (.png, .jpg, .jpeg, .pdf)
- `page_start`: Optional integer, PDFs only, must be >= 1 and <= total pages
- `page_end`: Optional integer, PDFs only, must be >= page_start and <= total pages
- `prompt`: Optional string, additional analysis instructions

**Page Selection Logic (PDFs):**
- Neither `page_start` nor `page_end` specified: Use first N pages (config: `multimodal.pdf_page_limit`)
- Only `page_start` specified: Analyze just that single page
- Both specified: Analyze range from page_start to page_end (inclusive)
- Range must not exceed `pdf_page_limit` in total page count
- `pdf_page_limit: 0` means all pages (displayed as "0 (all pages)" in UI/docs)

**Supported Formats (MVP):**
- Images: .png, .jpg, .jpeg
- Documents: .pdf

**Configuration (`~/.forge/config.yaml`):**
```yaml
multimodal:
  model: "gpt-4o"              # Vision-capable model for analysis
  pdf_page_limit: 10           # Max pages per analysis (0 = all pages)
```

### Analysis Prompt Template

```
You are analyzing a document to help an AI coding assistant understand its content.

Document Type: {image|PDF}
{For PDFs: Total Pages: X / Analyzing: Pages Y-Z}

Please provide a concise, structured analysis optimized for agent reasoning:

1. **Document Type & Purpose**: What is this document and why does it exist?

2. **Visual Structure**: Key visual elements, layout, organization

3. **Technical Content**: Code, diagrams, architecture, technical details

4. **Actionable Information**: What can the agent do with this information?

5. **Key Relationships**: How components/concepts relate to each other

{If custom prompt provided: Additional Focus: {prompt}}

Format your response as structured text optimized for agent consumption (information-dense, minimal decoration).
Keep analysis under 800 words unless document complexity requires more detail.
```

### Execution Flow

**For Images (PNG/JPG):**
1. Validate file path and format
2. Read image file and encode to base64
3. Build analysis prompt with image attachment
4. Call `provider.Complete()` with multimodal.model
5. Return structured analysis result

**For PDFs:**
1. Validate file path and format
2. Extract total page count
3. Determine page range:
   - If page_start/page_end not specified: pages 1 to min(pdf_page_limit, total_pages)
   - If page_start specified: validate range and use
   - If both specified: validate range doesn't exceed pdf_page_limit
4. Extract specified pages as images
5. Build analysis prompt with page context
6. Call `provider.Complete()` with multimodal.model
7. Return analysis with pagination metadata

**Error Handling:**
- File not found: "File not found: {path}. Check workspace-relative path."
- Unsupported format: "Unsupported format: {ext}. Supported: .png, .jpg, .jpeg, .pdf"
- Invalid page range: "Invalid page range: {start}-{end}. Document has {total} pages."
- Pages exceed limit: "Page range ({count} pages) exceeds pdf_page_limit ({limit}). Reduce range or adjust config."
- Provider error: "Analysis failed: {error}. Check model supports vision and retry."

### Response Format

**Image Analysis:**
```
Document Type: PNG Image
File: diagrams/mockup.png

Visual Structure:
- Hand-drawn wireframe with 3 main sections
- Header with navigation elements
- Central content area with form fields
- Footer with action buttons

Technical Content:
- Login form with email/password fields
- "Remember me" checkbox
- "Forgot password?" link
- Primary "Sign In" button (blue)
- Secondary "Create Account" link

Actionable Information:
- Agent should implement login form with email/password validation
- Include remember-me functionality
- Add password reset flow
- Use primary/secondary button styling pattern

Key Relationships:
- Login form is entry point to authenticated features
- Create account flow is alternative path
- Password reset is recovery mechanism
```

**PDF Analysis (with pagination):**
```
Document Type: PDF
File: docs/architecture.pdf
Total Pages: 50
Analyzed: Pages 1-10 (40 pages remaining)

Visual Structure:
- Technical documentation with diagrams
- Architecture diagrams using C4 model notation
- Code examples in Go
- Sequence diagrams for key flows

Technical Content:
[Pages 1-10 cover system overview and high-level architecture]
- Microservices architecture with 6 core services
- Event-driven communication via message bus
- Shared PostgreSQL database with service-specific schemas
- Redis for caching and session management

Actionable Information:
- Agent should understand service boundaries when making changes
- Database migrations must coordinate across services
- Events must follow documented schema contracts

Key Relationships:
- Auth service is dependency for all user-facing services
- Payment service communicates async with notification service
- API gateway routes to appropriate service based on path

Pagination Note: To analyze remaining pages (11-50), call analyze_document with page_start and page_end parameters. Recommended next: pages 11-20 for API specifications.
```

### Integration Points

**Tool Registration:**
```go
// pkg/tools/registry.go
func InitializeTools(cfg *config.Config, provider llm.Provider, workspaceRoot string) ([]Tool, error) {
    tools := []Tool{
        // ... existing tools ...
        NewAnalyzeDocumentTool(
            provider,
            workspaceRoot,
            cfg.Multimodal.Model, // Model selection from config
        ),
    }
    return tools, nil
}
```

**Provider Usage:**
```go
// Tool uses provider.Complete() for multimodal analysis
req := types.CompletionRequest{
    Model: t.model, // Use configured multimodal.model
    Messages: []types.Message{
        {
            Role: "user",
            Content: []types.ContentPart{
                {Type: "text", Text: analysisPrompt},
                {Type: "image", ImageData: base64Image},
            },
        },
    },
    MaxTokens: 2000, // Sufficient for detailed analysis
}

resp, err := t.provider.Complete(ctx, req)
```

**File Validation:**
```go
// Reuse existing workspace security
import "github.com/entrhq/forge/pkg/security"

func (t *AnalyzeDocumentTool) validatePath(path string) error {
    if err := security.ValidateWorkspacePath(t.workspaceRoot, path); err != nil {
        return fmt.Errorf("invalid path: %w", err)
    }
    
    ext := filepath.Ext(path)
    supported := map[string]bool{
        ".png": true, ".jpg": true, ".jpeg": true, ".pdf": true,
    }
    if !supported[ext] {
        return fmt.Errorf("unsupported format: %s", ext)
    }
    
    return nil
}
```

**PDF Processing:**
```go
// Use existing PDF library or add dependency
import "github.com/pdfcpu/pdfcpu/pkg/api"

func (t *AnalyzeDocumentTool) extractPDFPages(path string, start, end int) ([][]byte, error) {
    // Extract pages as images
    // Return slice of image bytes (PNG format)
    // Handle page range validation
}

func (t *AnalyzeDocumentTool) getPDFPageCount(path string) (int, error) {
    // Return total page count for validation
}
```

### Configuration Schema

```go
// pkg/config/config.go
type Config struct {
    // ... existing fields ...
    Multimodal MultimodalConfig `yaml:"multimodal"`
}

type MultimodalConfig struct {
    Model         string `yaml:"model"`           // Vision-capable model
    PDFPageLimit  int    `yaml:"pdf_page_limit"`  // Max pages per analysis (0 = all)
}

// Default configuration
func DefaultConfig() *Config {
    return &Config{
        // ... existing defaults ...
        Multimodal: MultimodalConfig{
            Model:        "gpt-4o",  // OpenAI's vision model
            PDFPageLimit: 10,        // Reasonable default
        },
    }
}
```

### Testing Strategy

**Unit Tests:**
- Test parameter validation (path, page ranges, formats)
- Test page selection logic (default, single page, range)
- Test prompt construction with different inputs
- Test error handling (file not found, invalid format, provider errors)
- Mock provider to test analysis flow
- Test response format parsing

**Integration Tests:**
- Test with real images (PNG/JPG) on sample diagrams
- Test with real PDFs (single page, multi-page)
- Test pagination (first N, specific range, all pages)
- Test with different vision models (GPT-4o, Claude 3.5 Sonnet)
- Test error scenarios (unsupported file, missing file, invalid pages)
- Test workspace security (path traversal attempts)

**Quality Tests:**
- Test analysis quality on excalidraw mockups
- Test analysis accuracy on technical PDFs with diagrams
- Measure token usage (images vs. PDFs, different page counts)
- Compare analysis consistency across providers
- Validate agent can use analysis results effectively

### Implementation Phases

**Phase 1: Core Implementation (8-12 hours)**
- Tool structure and parameter validation (2h)
- Image processing (PNG/JPG) with base64 encoding (2h)
- PDF page extraction and image conversion (3h)
- Provider integration with multimodal content (2h)
- Basic error handling and response formatting (1h)

**Phase 2: Configuration & Polish (4-6 hours)**
- Configuration schema and defaults (1h)
- Settings UI for multimodal section (2h)
- Comprehensive error messages (1h)
- Documentation and examples (2h)

**Phase 3: Testing & Validation (6-8 hours)**
- Unit tests for all components (3h)
- Integration tests with real files (2h)
- Quality validation with diverse documents (2h)
- Bug fixes and refinements (1h)

**Total Estimate: 18-26 hours**

---

## Validation

### Success Metrics

**Functionality:**
- Tool successfully analyzes PNG/JPG images 95%+ of time
- Tool successfully analyzes PDFs with correct pagination 95%+ of time
- Analysis output enables agent to make informed decisions 85%+ of time
- Page range validation catches invalid inputs 100% of time

**Efficiency:**
- Average analysis time < 5 seconds for images
- Average analysis time < 10 seconds for 10-page PDF
- Lazy pagination reduces token usage by 50%+ vs. analyzing entire large PDF
- Analysis output stays under 1000 words 90%+ of time

**Quality:**
- Excalidraw mockup analysis captures key elements 90%+ of time
- PDF technical content extraction is accurate 85%+ of time
- Agent successfully uses analysis in downstream tasks 80%+ of time
- Pagination guidance is clear and actionable 95%+ of time

### Monitoring

- Track analyze_document tool usage frequency
- Monitor analysis success/failure rates by file type
- Measure token consumption per analysis
- Track pagination usage patterns (default vs. custom ranges)
- Collect provider error rates for vision models
- Monitor configuration patterns (model choices, pdf_page_limit settings)

---

## Related Decisions

- [ADR-0039](0039-llm-powered-page-analysis.md) - LLM-Powered Page Analysis (similar pattern for web content)
- [ADR-0003](0003-provider-abstraction-layer.md) - Provider Abstraction Layer (foundation for LLM integration)
- [ADR-0011](0011-coding-tools-architecture.md) - Coding Tools Architecture (file tool patterns)
- [ADR-0027](0027-safety-constraint-system.md) - Safety Constraint System (workspace security)

---

## References

- [Product PRD](../product/features/multimodal-analysis.md) - Full product requirements
- [Scratch Document](../product/scratch/multimodal-analysis.md) - Design iteration history
- [OpenAI Vision API](https://platform.openai.com/docs/guides/vision) - Vision model documentation
- [Anthropic Claude 3](https://www.anthropic.com/claude) - Vision capabilities
- PDF Processing Library: [pdfcpu](https://github.com/pdfcpu/pdfcpu) - Go PDF manipulation

---

## Notes

**Security Considerations:**
- All file access goes through workspace guard (same as read_file, write_file)
- Vision model receives base64-encoded images (no direct file access)
- User controls which provider/model is used (same privacy model as chat)
- No external services beyond user's configured LLM provider

**Model Compatibility:**
Current vision-capable models:
- OpenAI: gpt-4o, gpt-4-turbo, gpt-4o-mini
- Anthropic: claude-3-5-sonnet-20241022, claude-3-opus, claude-3-sonnet
- Google: gemini-1.5-pro, gemini-1.5-flash

Users must configure a vision-capable model or analysis will fail with clear error.

**Future Enhancements:**
- Caching layer for repeated analyses (same file hash)
- Additional image formats (WebP, TIFF, SVG)
- Multi-file comparison mode (analyze 2+ images/PDFs together)
- Smart truncation strategies (first + last pages, content-based sampling)
- Token usage reporting in tool results
- Video frame extraction and analysis
- OCR fallback for scanned PDFs

**Design Evolution:**
- Initial design used `pages` string parameter (e.g., "1-5"), changed to `page_start`/`page_end` integers for simpler validation
- Considered `max_file_size_mb` config but removed as unnecessary abstraction - `pdf_page_limit` is the user-facing control
- Changed from `null` meaning all pages to `0` meaning all pages (clearer, more explicit)

**Last Updated:** 2025-01-20
