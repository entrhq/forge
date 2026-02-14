# 39. LLM-Powered Page Analysis for Browser Automation

**Status:** Proposed
**Date:** 2025-01-17
**Deciders:** Engineering Team
**Technical Story:** Extend browser automation with intelligent page analysis using LLM to provide agents with higher-level understanding of web content beyond raw text extraction.

---

## Context

The browser automation system (ADR-0038) currently provides three content extraction formats (markdown, text, structured), which give agents access to raw page content. However, agents often struggle to efficiently understand and contextualize web pages, especially when:

1. Pages are content-heavy with mixed signal-to-noise ratio
2. The agent needs to identify key actions or elements quickly
3. Complex layouts make raw text extraction confusing
4. The agent needs domain understanding (e.g., "this is a login page", "this is a documentation site")

### Background

Current extraction tools return raw content that agents must parse and understand through their own reasoning. This works but has limitations:

- **Token consumption**: Large pages consume significant context even when only a summary is needed
- **Cognitive load**: Agents must infer page structure and purpose from raw content
- **Time inefficiency**: Multiple tool calls needed to explore and understand a page
- **Quality variance**: Agent understanding quality depends heavily on content format and organization

Other browser automation tools (e.g., browser-use, Claude computer use) demonstrate value in higher-level page understanding, but typically rely on:
- Screenshot analysis (expensive, requires vision models)
- Hardcoded heuristics (brittle, limited scope)
- No understanding at all (relies purely on element selectors)

### Problem Statement

How can we enable agents to quickly understand web page content, structure, and purpose without consuming excessive context or requiring multiple exploration iterations?

### Goals

- Provide concise, actionable page summaries that help agents make decisions
- Reduce token consumption compared to full content extraction
- Identify key interactive elements and their purposes
- Understand page context (login page, documentation, dashboard, etc.)
- Enable intelligent navigation decisions based on page analysis
- Maintain privacy and security (analysis happens locally via user's LLM provider)

### Non-Goals

- Real-time page analysis (analysis is on-demand, not automatic)
- Screenshot-based visual analysis (text-based only for MVP)
- Automated decision making (agent still decides actions)
- Page classification training (use general-purpose LLM understanding)
- Multi-page workflows (each page analyzed independently)

---

## Decision Drivers

* **Context Efficiency**: Need to reduce token usage while improving understanding
* **Agent Autonomy**: Agents should make better navigation and interaction decisions
* **Implementation Simplicity**: Leverage existing LLM provider abstraction
* **Privacy**: Analysis must use user's configured LLM provider (no external services)
* **Performance**: Analysis should be fast enough for interactive workflows
* **Extensibility**: Should support future enhancements (custom prompts, domain-specific analysis)

---

## Considered Options

### Option 1: Screenshot-Based Analysis with Vision Models

**Description:** Capture page screenshots and use vision-capable LLMs (GPT-4V, Claude 3) to analyze visual content.

**Pros:**
- Can understand visual layout and design
- Sees exactly what users see
- Can identify UI patterns humans recognize visually
- Works for pages with poor accessibility

**Cons:**
- Requires vision model support (not all providers)
- Significantly higher token costs (images are expensive)
- Slower processing time (image encoding/transmission)
- Screenshot tool not yet implemented (Phase 2)
- Privacy concerns with image data
- Viewport-dependent (large pages require scrolling strategy)

### Option 2: Client-Side JavaScript Analysis

**Description:** Inject JavaScript into pages to analyze DOM structure, extract semantic information, and classify page type.

**Pros:**
- Fast execution (runs in browser)
- No additional LLM costs
- Can leverage browser APIs (accessibility tree, computed styles)
- Deterministic results

**Cons:**
- Requires complex JavaScript logic and heuristics
- Brittle (breaks when page structures change)
- Limited understanding (no semantic reasoning)
- Maintenance burden (rules need constant updates)
- Can't understand content meaning, only structure
- May not work on pages with heavy JavaScript frameworks

### Option 3: LLM-Powered Text Analysis

**Description:** Extract page content as text/markdown, send to LLM with analysis prompt, return structured understanding.

**Pros:**
- Leverages existing LLM provider infrastructure
- Uses user's configured model (privacy-preserving)
- Semantic understanding of content meaning
- Flexible analysis (prompt can evolve)
- No additional dependencies
- Works with any text-extractable content
- Cost-effective (text-only, can limit input size)

**Cons:**
- Additional LLM call (latency and cost)
- Limited to text content (can't see visual design)
- Quality depends on extraction format
- May hallucinate details not in content

### Option 4: Hybrid Local + Cloud Service

**Description:** Combine local DOM analysis with optional cloud-based specialized page understanding service.

**Pros:**
- Best of both worlds (fast structure + smart understanding)
- Could be highly optimized for page analysis
- Potential for caching common page types

**Cons:**
- Complex architecture (multiple services)
- Privacy concerns (external service sees user data)
- Additional cost model beyond LLM
- Vendor lock-in to analysis service
- Network dependency

---

## Decision

**Chosen Option:** Option 3 - LLM-Powered Text Analysis

We will implement a new browser tool `analyze_page` that extracts page content and uses the agent's configured LLM provider to generate an intelligent summary and analysis.

### Rationale

1. **Leverages Existing Infrastructure**: Uses the Provider interface we already have, no new dependencies
2. **Privacy-Preserving**: Analysis happens via user's own LLM provider, same as agent conversations
3. **Flexible and Extensible**: Analysis prompt can be refined and customized over time
4. **Semantic Understanding**: LLMs excel at understanding content meaning and context
5. **Cost-Effective**: Text analysis is far cheaper than vision models, and we can limit input size
6. **Implementation Simplicity**: Straightforward tool that composes existing capabilities
7. **Consistent Experience**: Uses same LLM model as agent, ensuring compatible understanding

The key insight is that for most web automation tasks, understanding *what* the page contains and *why* is more important than *how* it looks visually. Text-based semantic analysis provides this understanding efficiently.

---

## Consequences

### Positive

- Agents can understand pages in one tool call instead of multiple extract/search iterations
- Reduced total token consumption (concise summary vs. full content)
- Better navigation decisions (agent knows what page offers before clicking)
- Improved debugging (page analysis helps identify why automation failed)
- Natural language page descriptions are easier for agents to reason about
- Can identify actionable elements and their purposes
- Extensible foundation for future analysis enhancements

### Negative

- Additional LLM call adds latency (mitigated by using Complete() not streaming)
- LLM costs increase per page analyzed (but saves tokens overall from reduced exploration)
- Analysis quality depends on extraction quality (garbage in, garbage out)
- May hallucinate details not present in content (standard LLM limitation)
- Text-only analysis misses visual-only information (acceptable tradeoff for MVP)

### Neutral

- Adds another tool to browser toolkit (but only visible when sessions exist)
- Agents must learn when to use analyze vs. extract (but clear use cases)
- Analysis prompt will need iteration and refinement over time
- Different LLM models will produce different analysis quality

---

## Implementation

### Architecture

**New Tool: `analyze_page`**

```go
package browser

import (
    "context"
    "github.com/entrhq/forge/pkg/llm"
    "github.com/entrhq/forge/pkg/types"
)

type AnalyzePageTool struct {
    manager  *SessionManager
    provider llm.Provider
}

func NewAnalyzePageTool(manager *SessionManager, provider llm.Provider) *AnalyzePageTool {
    return &AnalyzePageTool{
        manager:  manager,
        provider: provider,
    }
}
```

**Tool Parameters:**
- `session` (required): Name of browser session to analyze
- `selector` (optional): CSS selector to analyze specific element instead of full page
- `focus` (optional): Analysis focus area: "navigation", "forms", "data", "general" (default: "general")

**Usage Guidance:**

Use `analyze_page` when:
- First visiting a page to understand its structure, purpose, and available actions
- Making navigation decisions (which link to click, where to go next)
- Identifying actionable elements (forms, buttons, key links) and their purposes
- Need a high-level summary to decide next steps

Use `extract_content` when:
- Need full page text for detailed reading or data extraction
- Looking for specific information you know exists on the page
- Require precise content for copy/paste or verification
- Already understand page structure and need raw content

**Analysis Prompt Template:**

```
You are analyzing a web page to help an AI agent understand its content and structure.

Page Content:
{extracted_content}

Please provide a concise analysis including:

1. **Page Type**: What kind of page is this? (e.g., login page, documentation, dashboard, product listing, article)

2. **Primary Purpose**: What is the main purpose or goal of this page?

3. **Key Elements**: List the most important interactive elements (buttons, links, forms) and what they do.

4. **Notable Content**: Summarize the key information or content presented.

5. **Suggested Actions**: What actions would a user typically take on this page?

6. **Navigation Options**: What navigation paths are available from this page?

Format your response as structured text, keeping it concise (under 500 words).
```

**Execution Flow:**

1. Validate parameters (session exists, valid focus option)
2. Extract page content using existing `ExtractContent` method (markdown format)
3. Build analysis prompt with extracted content
4. Call `provider.Complete()` with analysis prompt (provider handles content length limits)
5. Return structured analysis result
6. Update session LastUsedAt timestamp

**Response Format:**

```
Page Analysis for: {url}

Page Type: {type}

Purpose: {purpose}

Key Elements:
- {element_1}: {description}
- {element_2}: {description}
...

Notable Content:
{summary}

Suggested Actions:
- {action_1}
- {action_2}
...

Navigation Options:
- {nav_option_1}
- {nav_option_2}
...
```

### Integration Points

**Tool Registration:**
- Register in `browser.InitializeTools()` alongside other browser tools
- Pass LLM provider instance from agent/executor
- Implements `ConditionallyVisible` interface (only shown when sessions exist)
- Implements `Previewable` interface for approval flow

**Provider Injection:**
- Tools package receives provider from executor initialization
- Browser tool registry maintains provider reference
- Same provider instance used by agent for consistency

**Error Handling:**
- Handle extraction failures (delegate to ExtractContent error handling)
- Handle LLM API failures (timeout, rate limit, invalid response)
- Handle content too large (provider will handle via context limits)
- Clear error messages back to agent

### Configuration

**Settings (optional, future enhancement):**
- `browser_analysis_model`: Override model for analysis (default: use agent's model)
- `browser_analysis_timeout`: Analysis timeout in seconds (default: 30)

### Example Usage

Agent conversation:
```
User: "Navigate to playwright.dev and understand what it offers"

Agent: I'll navigate to the site and analyze it.

[Uses browser_navigate tool]
[Uses analyze_page tool]

Agent: "Playwright.dev is a technical documentation homepage for the Playwright browser 
automation library. The main purpose is to introduce developers to Playwright and guide 
them to getting started. Key elements include a 'Get Started' button leading to 
installation, an API Reference link, code examples section, and a search bar. The site 
supports multiple programming languages (JavaScript, Python, Java, .NET). I can help you 
get started with installation or explore specific features."
```

### Testing Strategy

**Unit Tests:**
- Test prompt construction with various content inputs
- Test parameter validation (session name, focus options)
- Test content truncation logic
- Mock LLM provider to test error handling
- Verify response format parsing

**Integration Tests:**
- Test with real browser sessions on sample pages
- Test with different page types (login, docs, e-commerce)
- Test with different focus options
- Test content length limits and truncation
- Test error scenarios (network failure, timeout)

**Quality Tests:**
- Test analysis quality on diverse page types
- Compare token usage vs. full content extraction
- Measure analysis latency across different providers
- Validate structured response consistency

---

## Validation

### Success Metrics

**Effectiveness:**
- Agents make correct navigation decisions based on analysis (measured via test scenarios)
- Average tool calls per task reduced by 20-30% (fewer exploration iterations)
- Agent task success rate improves by 10-15% on web automation tasks

**Efficiency:**
- Average analysis time < 3 seconds (acceptable latency for decision-making)
- Token savings: 40-60% reduction vs. extracting + reasoning over full content
- Analysis summaries stay within 500 word target 95% of time

**Quality:**
- Analysis accurately identifies page type 90%+ of time
- Key interactive elements identified 85%+ of time
- Suggested actions are relevant 80%+ of time

### Monitoring

- Track analyze_page tool usage frequency
- Monitor analysis latency distribution
- Track LLM provider errors for analysis calls
- Measure token consumption (analysis vs. extraction alternatives)
- Collect feedback on analysis accuracy (via agent behavior patterns)

---

## Migration Path

### Phase 1: MVP Implementation
1. Implement `AnalyzePageTool` with basic analysis prompt
2. Add provider injection to browser tools initialization
3. Register tool with conditional visibility
4. Add comprehensive tests

### Phase 2: Refinement
1. Gather usage data and refine analysis prompt
2. Add focus options for specialized analysis (forms, navigation, data)
3. Optimize content extraction for analysis (strip unnecessary elements)
4. Add caching for repeated page analyses (same URL + session)

### Phase 3: Advanced Features
1. Custom analysis prompts via configuration
2. Domain-specific analysis (e.g., e-commerce vs. documentation)
3. Comparative analysis (compare two pages)
4. Historical analysis tracking (understand page changes over time)

---

## Related Decisions

- [ADR-0038](0038-browser-automation-architecture.md) - Browser Automation Architecture (foundation)
- [ADR-0003](0003-provider-abstraction-layer.md) - Provider Abstraction Layer (LLM interface)
- [ADR-0019](0019-xml-cdata-tool-call-format.md) - XML Tool Call Format (tool schema)
- [ADR-0014](0014-composable-context-management.md) - Context Management (token optimization)

---

## References

- [Playwright Content Extraction](https://playwright.dev/docs/api/class-page#page-content)
- [Browser-Use Project](https://github.com/browser-use/browser-use) - LLM browser automation examples
- [Anthropic Computer Use](https://www.anthropic.com/news/claude-computer-use) - Vision-based page understanding
- LLM Provider Interface: `pkg/llm/provider.go`

---

## Notes

**Implementation Priority:** Phase 2 (P1) - Enhances browser automation significantly but not blocking MVP

**Security Considerations:**
- Analysis uses user's configured LLM provider (same security model as agent)
- Page content sent to LLM (user should be aware of data sharing with provider)
- No external services beyond user's chosen LLM provider
- Respect user's model choice and API key configuration

**Future Enhancements:**
- Vision model integration when screenshot tool is available (Phase 3)
- Accessibility analysis (ARIA, semantic HTML evaluation)
- Performance analysis (load times, resource sizes)
- SEO analysis (meta tags, structured data)
- Security analysis (HTTPS, mixed content, security headers)

**Last Updated:** 2025-01-17
