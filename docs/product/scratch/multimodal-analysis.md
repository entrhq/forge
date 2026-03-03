# Multimodal Document Analysis - Feature Scratch

**Status:** Initial concept  
**Date:** 2025-01-27  
**Related Features:** Excalidraw diagram generation/review

## Problem

As part of the excalidraw diagram generation and review workflow, the agent needs the ability to analyze visual content - specifically PNGs and PDFs. Currently, the agent can only process text-based content, which limits its ability to:

- Understand diagram screenshots or mockups provided by users
- Review generated excalidraw diagrams visually
- Extract information from PDF documentation with diagrams/charts
- Analyze design documents that combine text and visual elements

Without multimodal analysis capability, the agent has to rely on text descriptions of visual content, which is inefficient and error-prone for diagram-heavy workflows.

## Solution Approach

Create a **built-in tool** for multimodal document analysis (similar to the page analysis tool):

1. **Leverages existing provider abstraction** - Use the same `llm.Provider` interface pattern we use for text models and page analysis
2. **Supports model selection** - Similar to the "analyse page" feature, allow users to configure a specific model for multimodal analysis (e.g., GPT-4 Vision, Claude 3.5 Sonnet)
3. **Handles multiple formats** - Support PNG images and PDF documents initially
4. **Returns structured analysis** - Provide analysis results that can be used by the agent in its reasoning loop
5. **Built-in integration** - Part of core tools (pkg/tools/), not a custom tool, for tight integration with provider system

## Key Capabilities

### Input Handling
- Accept file paths (relative to workspace) for images and PDFs
- Validate file types and sizes
- Convert files to appropriate formats for LLM consumption (base64 encoding, etc.)

### Analysis Mode (MVP)
- **Single comprehensive analysis** - One analysis mode that provides:
  - Visual understanding of content (diagrams, charts, layouts)
  - Text extraction where present
  - Component and relationship identification
  - Technical details relevant to the document type
  - Analysis formatted for agent consumption (information-dense, minimal decoration)

### Configuration
- User-configurable model selection in ~/.forge/config.yaml
- Separate from main chat model (similar to page analysis)
- Support for different providers (OpenAI, Anthropic, etc.)

## Integration Points

### Excalidraw Workflow
- Analyze user-provided mockups/sketches before generating diagrams
- Review generated excalidraw exports (PNG) to validate against requirements
- Iterate on diagrams based on visual feedback

### Tool Interface
The tool should be invokable with parameters like:
```
tool: analyze_document
args:
  path: "diagrams/mockup.png"  # Supports: .png, .jpg, .jpeg, .pdf
  page_start: 10                # Optional: First page to analyze (PDFs only)
  page_end: 15                  # Optional: Last page to analyze (PDFs only)
  prompt: "Optional additional context/instructions for the analysis"
```

**Page Selection for PDFs:**
- If `page_start` and `page_end` not specified: Uses first N pages based on `pdf_page_limit` config
- If only `page_start` specified: Analyzes just that single page
- If both specified: Analyzes page range from start to end (inclusive)
- Page range must not exceed `pdf_page_limit` in total page count
- Enables "lazy pagination" - agent can analyze PDF incrementally based on findings

**Output Format:** Textual analysis optimized for agent reasoning, not human presentation. Should include:
- Document type and basic metadata (file type, page count for PDFs)
- **PDF pagination info:** "Analyzed pages 1-10 of 50 total pages" (if truncated)
- Visual structure (components, layout, relationships)
- Extracted text content
- Technical details (for diagrams: flow direction, component types, connections)
- Anomalies or notable features

**Example PDF Response:**
```
Document Type: PDF
Total Pages: 50
Analyzed: Pages 1-10 (40 pages remaining)

Content Summary:
[Analysis of pages 1-10...]

Note: To analyze remaining pages, call analyze_document with page_start and page_end parameters (e.g., page_start: 11, page_end: 20)
```

### Configuration
Configuration in ~/.forge/config.yaml:
```yaml
multimodal:
  model: "claude-3-5-sonnet-20241022"  # Uses same provider as main LLM
  pdf_page_limit: 10       # Maximum pages to process from PDFs (0 = all pages)
```

**Design Notes:**
- Uses the main LLM provider (no separate provider config in multimodal section)
- Model selection works like page analysis - user picks which model to use
- `pdf_page_limit: 0` means process all pages (shown in UI/docs as "0 (all pages)")
- No max_file_size_mb needed - file size is an implementation detail, not user-facing config

## MVP Decisions

1. **File Format Support** - ✅ Both images (PNG, JPG) and PDFs in MVP - similar processing, both add significant value
2. **Processing Model** - ✅ Single file at a time to avoid unwieldy multi-file analysis
3. **Caching** - ✅ No caching in MVP - keep implementation simple
4. **Output Format** - ✅ Textual analysis optimized for **agent consumption** (not human readability)
   - Similar philosophy to episodic memory - structured for agent reasoning
   - Should include: key components identified, relationships, technical details, extracted text
   - Avoid decorative formatting, focus on information density
5. **Tool Type** - ✅ Built-in tool (pkg/tools/) like page analysis, not custom tool
6. **Provider Integration** - ✅ Uses main LLM provider (same as chat), user selects model via config
7. **PDF Processing** - ✅ Smart pagination with truncation:
   - `pdf_page_limit: 10` (default) - max pages per analysis call (0 = all pages)
   - Default behavior: Analyze first N pages based on limit
   - **Lazy pagination:** Agent can specify `page_start` and `page_end` parameters to analyze specific ranges
   - Simple integer parameters (no string parsing, clear validation)
   - Tool response includes pagination metadata: "Analyzed pages X-Y of Z total"
   - Enables incremental analysis: agent reads first chunk, decides what to read next
8. **Truncation Strategy** - ✅ First N pages approach
   - Simple, predictable behavior
   - Tool result tells agent how many pages remain
   - Agent decides whether to continue pagination
9. **No file size limit config** - ✅ File size is implementation detail, not exposed to user

## Next Steps

1. Review this concept with user and refine the scope
2. Decide on MVP feature set
3. Create full PRD with user stories, success metrics, technical requirements
4. Design tool interface and provider integration
5. Write ADR for technical implementation approach
