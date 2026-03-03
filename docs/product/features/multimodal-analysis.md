# Multimodal Document Analysis

## Product Vision
Enable Forge to understand and reason about visual content (images and PDFs) by providing a built-in multimodal analysis capability. This advances Forge's mission as an AI coding assistant by allowing it to analyze diagrams, design mockups, documentation with visuals, and other non-textual artifacts that are essential to modern software development workflows.

## Key Value Propositions
- **For developers using Forge**: Seamlessly analyze diagrams, screenshots, and PDF documentation without leaving the Forge workflow
- **For diagram-heavy workflows**: Generate and review excalidraw diagrams by providing visual mockups or analyzing generated outputs
- **For documentation review**: Extract insights from PDF technical documents that combine text and visual elements
- **Competitive advantage**: Built-in multimodal capability with smart pagination enables incremental analysis of large documents, avoiding token waste and enabling targeted exploration

## Target Users & Use Cases

### Primary Personas
- **Agent (Forge AI)**: Needs to understand visual content provided by users or generated during workflows to make informed coding decisions
- **Developer**: Wants to share diagrams, mockups, or PDF documentation with Forge and get meaningful analysis without manual transcription

### Core Use Cases
1. **Excalidraw Diagram Review**: User provides hand-drawn mockup → Agent analyzes structure → Agent generates excalidraw diagram → Agent reviews generated PNG to validate accuracy
2. **Technical Documentation Analysis**: User references PDF architecture doc with diagrams → Agent extracts relevant information incrementally using lazy pagination → Agent applies insights to code changes
3. **UI/UX Mockup Analysis**: User shares screenshot or design file → Agent understands layout, components, and relationships → Agent implements matching code structure
4. **Error Screenshot Debugging**: User provides screenshot of error state → Agent analyzes visual context → Agent identifies root cause from visual cues

## Product Requirements

### Must Have (P0)
- Support PNG, JPG, and JPEG image formats for multimodal analysis
- Support PDF document format with multi-page handling
- Built-in tool implementation using existing provider abstraction pattern (similar to page analysis)
- Model selection via configuration (user can choose vision-capable model independently of chat model)
- Lazy pagination for PDFs: agent can specify page_start and page_end to analyze specific ranges
- Default pagination: analyze first N pages based on configurable pdf_page_limit (0 = all pages)
- Tool response includes pagination metadata: "Analyzed pages X-Y of Z total pages"
- Agent-optimized output format: information-dense textual analysis structured for agent reasoning (not human readability)
- Configuration in ~/.forge/config.yaml under new 'multimodal' section
- File path validation (workspace-relative, security checks)

### Should Have (P1)
- Clear error messages when files exceed limits or are unsupported formats
- File format validation with helpful feedback
- Page range validation for PDFs (start <= end, within document bounds)
- Optional prompt parameter for additional analysis instructions
- Tool result includes document metadata (type, total pages for PDFs)

### Could Have (P2)
- Support for additional image formats (WebP, TIFF)
- Batch analysis of multiple related images
- Comparison mode (analyze two images for differences)
- Token usage reporting in tool results
- Caching of analysis results for repeated requests
- Smart truncation strategies (e.g., analyze first + last pages)

## User Experience Flow

### Entry Points
- Agent calls analyze_document tool during conversation when visual analysis is needed
- User provides file path to image or PDF in conversation
- Agent proactively suggests multimodal analysis when detecting visual content references

### Core User Journey
```
[User shares diagram/PDF] → [Agent calls analyze_document] → [Provider processes with vision model]
     ↓
[Tool returns analysis]
     ↓
[PDF with many pages?]
     ↓
[Yes] → [Agent reviews first chunk] → [Agent decides to paginate] → [Agent calls with page_start/page_end]
[No]  → [Agent uses analysis in reasoning] → [Agent completes task]
```

### Success States
- **Image Analysis**: Agent receives comprehensive description of visual content and uses it to inform code decisions
- **PDF First Pass**: Agent gets overview from first N pages and determines if deeper analysis is needed
- **PDF Pagination**: Agent incrementally analyzes specific sections based on initial findings, optimizing token usage
- **Excalidraw Workflow**: Agent validates generated diagram matches requirements by analyzing PNG export

### Error/Edge States
- **Unsupported Format**: Clear error with list of supported formats (.png, .jpg, .jpeg, .pdf)
- **File Not Found**: Error with workspace-relative path guidance
- **Invalid Page Range**: Error explaining page bounds and valid range syntax
- **Pages Exceed Limit**: Error indicating total pages requested exceeds pdf_page_limit configuration
- **Provider Error**: Graceful handling of LLM provider failures with retry guidance

## User Interface & Interaction Design

### Key Interactions
- **Configuration**: User sets multimodal.model and multimodal.pdf_page_limit in config.yaml
- **Tool Invocation**: Agent calls analyze_document with path, optional page_start/page_end, optional prompt
- **Pagination Discovery**: Tool response clearly indicates when content is truncated and how to access remaining pages
- **Model Selection**: Similar to page analysis - user chooses vision-capable model in configuration

### Information Architecture
```
analyze_document tool result:
├── Document Metadata
│   ├── Type (PNG/JPG/PDF)
│   ├── Total Pages (PDFs only)
│   └── Analyzed Range (PDFs only)
├── Content Analysis
│   ├── Visual Structure
│   ├── Extracted Text
│   ├── Components & Relationships
│   └── Technical Details
└── Pagination Guidance (if truncated)
    └── How to access remaining content
```

### Progressive Disclosure
- Initial analysis provides overview without overwhelming detail
- Agent can drill down into specific pages/sections based on findings
- Pagination metadata guides agent toward relevant sections
- Optional prompt parameter allows targeted analysis focus

## Feature Metrics & Success Criteria

### Key Performance Indicators
- **Adoption**: Percentage of sessions where analyze_document tool is used
- **PDF Pagination Rate**: How often agents use page_start/page_end vs. default behavior
- **Analysis Accuracy**: Qualitative assessment of whether agent correctly interprets visual content
- **Token Efficiency**: Average tokens per analysis vs. theoretical maximum (indicates smart pagination usage)

### Success Thresholds
- 50%+ of excalidraw generation tasks use multimodal analysis for validation
- <5% error rate for file format validation and page range handling
- Agent successfully uses lazy pagination in 80%+ of multi-page PDF analyses
- Positive user feedback on analysis quality and relevance

## User Enablement

### Discoverability
- Documentation in how-to guides showing excalidraw workflow
- AGENTS.md reference for tool parameters and usage patterns
- Example conversations demonstrating PDF pagination strategy
- Tool description in available_tools list explains capabilities

### Onboarding
- Default configuration works out-of-box for common use cases (10 page limit)
- Clear configuration comments explaining pdf_page_limit: 0 means all pages
- Error messages guide users to correct configuration when needed
- Examples showing both simple (image) and complex (PDF pagination) usage

### Mastery Path
- **Novice**: Use default settings for simple image analysis
- **Intermediate**: Configure pdf_page_limit based on typical document sizes
- **Advanced**: Strategic use of page_start/page_end for efficient large document exploration
- **Expert**: Combine multimodal analysis with other tools (search_files, read_file) for comprehensive document understanding

## Risk & Mitigation

### User Risks
- **Token Cost**: Vision models are expensive; large PDFs could burn through budget
  - *Mitigation*: Configurable pdf_page_limit with sensible default (10 pages), lazy pagination encourages targeted analysis
- **Analysis Quality**: Vision models may misinterpret complex diagrams
  - *Mitigation*: Agent-optimized output format emphasizes factual extraction over interpretation, optional prompt parameter for focus
- **Privacy**: Sensitive documents sent to third-party LLM providers
  - *Mitigation*: Same provider security model as existing chat - user controls which provider/model, files stay workspace-bound

### Adoption Risks
- **Configuration Complexity**: Users may not understand pdf_page_limit settings
  - *Mitigation*: Clear defaults, bracket notation for "0 (all pages)", documentation with examples
- **Agent Misuse**: Agent might analyze wrong pages or waste tokens on irrelevant content
  - *Mitigation*: Tool result includes clear pagination guidance, agent can self-correct on next call
- **Provider Compatibility**: Not all LLM providers support vision models
  - *Mitigation*: Clear error when configured model doesn't support multimodal, documentation lists compatible models

## Dependencies & Integration Points

### Feature Dependencies
- **Provider Abstraction**: Requires llm.Provider interface to support multimodal content
- **File System Access**: Uses existing file tools security model (workspace isolation)
- **Configuration System**: Extends config.yaml with new multimodal section
- **Tool System**: Built-in tool registration and execution framework

### System Integration
- **Provider Selection**: Reuses main LLM provider but allows separate model configuration
- **Security**: Workspace guard validates file paths like other file tools
- **Tool Response Format**: Returns text that agent can reason about (consistent with episodic memory style)

### External Dependencies
- Vision-capable LLM provider (OpenAI GPT-4o, Anthropic Claude 3.5 Sonnet, etc.)
- Provider API must support image input (base64 or URL)
- Provider API must support PDF processing or page extraction

## Constraints & Trade-offs

### Design Decisions

**Built-in Tool vs. Custom Tool**
- Decision: Built-in tool in pkg/tools/
- Rationale: Core functionality that needs tight provider integration, should evolve with codebase, not experimental

**Page Parameters: page_start/page_end vs. Range String**
- Decision: Two separate integer parameters
- Rationale: Simpler for agent to generate correctly, no string parsing, clear validation, explicit tool schema

**Config Section: multimodal vs. llm.vision_model**
- Decision: New top-level multimodal section
- Rationale: Clear separation, room for future multimodal settings (truncation limits, etc.), doesn't clutter existing llm section

**No File Size Limit Config**
- Decision: Don't expose max_file_size_mb to user
- Rationale: File size is implementation detail, pdf_page_limit is the user-facing control, simpler mental model

### Known Limitations
- MVP supports only PNG, JPG, JPEG, PDF - no video, WebP, TIFF, etc.
- Single file at a time - no batch analysis or multi-file comparison
- No caching - repeated analysis of same file costs tokens each time
- No token usage reporting - users can't see analysis cost directly
- Page selection is simple range - no smart sampling or content-based selection

### Future Considerations
- **Multi-file analysis**: Compare multiple diagrams or documents
- **Video support**: Analyze screen recordings or video documentation
- **Caching layer**: Deduplicate repeated analyses
- **Smart truncation**: Analyze first + last pages, or sample based on content density
- **Token budgets**: Expose token costs to user, warn before expensive operations
- **Async analysis**: Queue large PDF analysis for background processing

## Competitive Analysis

**GitHub Copilot**: No native multimodal analysis - requires manual description of visual content  
**Cursor**: Supports image analysis but no PDF pagination - must process entire document  
**Cody**: Limited image support, no PDF handling  
**Aider**: No multimodal capabilities

**Forge advantage**: Built-in lazy pagination for PDFs enables efficient exploration of large documents without token waste. Agent-optimized output format ensures analysis is immediately actionable.

## Go-to-Market Considerations

### Positioning
"Forge now understands visual content - share diagrams, mockups, and PDF documentation directly in your workflow. No more manual transcription."

### Documentation Needs
- How-to guide: "Analyzing Diagrams and PDFs with Forge"
- Configuration reference: multimodal section in config.yaml docs
- AGENTS.md update: analyze_document tool reference
- Example conversation: excalidraw generation with visual validation
- Example conversation: PDF documentation exploration with pagination

### Support Requirements
- Common question: "Which models support multimodal analysis?" (Answer: List of vision-capable models)
- Common question: "Why is my PDF analysis truncated?" (Answer: Explain pdf_page_limit and how to paginate)
- Troubleshooting: Provider errors when model doesn't support vision

## Evolution & Roadmap

### Version History
- **v1.0 (MVP)**: PNG/JPG/PDF support, lazy pagination, agent-optimized output, model configuration

### Future Vision
- **v1.1**: Additional formats (WebP, TIFF), token usage reporting
- **v1.2**: Multi-file comparison mode, caching layer
- **v1.3**: Smart truncation strategies, async processing for large documents
- **v2.0**: Video analysis, real-time screen capture analysis

### Deprecation Strategy
Not applicable - core capability expected to remain indefinitely

## Technical References
- **Architecture**: ADR-TBD (multimodal document analysis implementation)
- **Implementation**: See pkg/tools/analyze_document.go (to be created)
- **Scratch Doc**: docs/product/scratch/multimodal-analysis.md

## Appendix

### Research & Validation
- User feedback from excalidraw feature requests highlighted need for visual validation
- Existing workflows require manual screenshot description, slowing iteration
- PDF documentation analysis currently requires reading entire file into context (token waste)

### Design Artifacts
- Scratch document with iteration history: docs/product/scratch/multimodal-analysis.md
- Configuration examples in this PRD
- Tool interface specification in this PRD
