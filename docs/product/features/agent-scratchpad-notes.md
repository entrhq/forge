# Product Requirements: Agent Scratchpad & Notes System

**Feature:** Agent Working Memory System  
**Version:** 1.0  
**Status:** In Development  
**Owner:** Core Team  
**Last Updated:** November 2025

---

## Product Vision

The Agent Scratchpad provides Forge with a working memory system that maintains context across the agent loop, enabling more efficient and consistent task execution. This feature transforms Forge from a stateless tool executor into an agent with short-term memory, capable of tracking discoveries, decisions, and progress throughout complex multi-step tasks.

By giving the agent the ability to "remember" and organize information within a session, we reduce redundant operations, improve decision consistency, and enable more sophisticated reasoning patterns that mirror how human developers work through complex problems.

## Key Value Propositions

- **For Software Engineers**: Experience an AI assistant that remembers important insights and decisions mid-task, leading to faster completion of complex refactoring and debugging work with fewer contradictions
- **For Development Teams**: Get more consistent and reliable automated changes, as the agent maintains architectural decisions and patterns throughout the session
- **Competitive Advantage**: Unlike stateless AI assistants, Forge builds and maintains a working understanding of relationships, trade-offs, and decisions, improving consistency and reducing backtracking

## Target Users & Use Cases

### Primary Personas

- **Senior Software Engineer**: Needs to refactor complex codebases with many interdependencies. Values consistency in architectural decisions and pattern application across files.
- **DevOps Engineer**: Debugs issues spanning multiple services and configuration files. Needs to track findings across investigation and correlate symptoms.
- **Technical Lead**: Reviews and guides large-scale changes. Wants transparency into the agent's reasoning and decision-making process.

### Core Use Cases

1. **Large-Scale Refactoring**: Agent discovers refactoring pattern, documents trade-offs, tracks implementation approach across components, and maintains consistency without re-discovering the same relationships
2. **Multi-System Debugging**: Agent investigates bug across multiple systems, documenting root cause analysis, fix rationale, and cross-component dependencies that aren't obvious from code alone
3. **Feature Implementation**: Agent plans feature across multiple modules, records architectural decisions with trade-offs, and tracks progress through complex multi-step implementations
4. **Code Review Preparation**: Agent documents reasoning behind changes made during session for inclusion in PR description, maintaining context about why certain approaches were chosen

## Product Requirements

### Must Have (P0)

- **Note CRUD Operations**: Create, read, update, and delete notes with 800-character limit
- **Required Tagging**: Minimum 1, maximum 5 tags per note for organization
- **Content Search**: Search notes by substring/keyword matching in content
- **Tag Filtering**: Filter notes by one or more tags with AND logic
- **Recency Ordering**: List notes prioritizing most recent when limiting results
- **Scratch Tracking**: Mark notes as scratched out when no longer needed and filter by scratched status
- **Session Scoped**: Notes exist only for current session, cleared on exit
- **Tag Discovery**: List all unique tags in use across notes

### Should Have (P1)

- **Smart List Limiting**: Default limit of 10 notes to prevent context pollution, configurable
- **Note Metadata**: Automatic timestamps (created_at, updated_at) for temporal context
- **Unique IDs**: Auto-generated stable identifiers for reliable note references
- **Tag Usage Stats**: Show count of notes per tag when listing tags
- **Validation**: Enforce character limits and tag count constraints with clear errors

### Could Have (P2)

- **Note Export**: Save session notes to file at end of session for documentation
- **UI Visualization**: Display active notes in TUI sidebar for user awareness
- **Auto-Tagging Suggestions**: Suggest tags based on note content using pattern matching
- **Note Relationships**: Link related notes or create hierarchies
- **Smart Pruning**: Auto-archive very old notes if count exceeds threshold
- **Search Highlighting**: Highlight matching terms in search results

## User Experience Flow

### Entry Points

**Agent-Initiated**: Agent automatically creates notes during task execution when discovering important insights
- Cross-component dependencies and patterns trigger pattern notes
- Architectural decisions with trade-offs get decision tags
- Multi-step workflows and progress tracking documented with progress notes
- Root cause analysis and fix rationale documented in bug notes

**Implicit Access**: Notes are invisible to users unless explicitly displayed through UI (future)
- Agent uses notes transparently in agent loop
- Users trust agent to manage working memory appropriately

### Core User Journey

```
[Agent Explores Codebase]
     ↓
[Discovers Auth Flow Pattern] → add_note(content="Auth chain: Login validates → JWT middleware checks → Permission service authorizes. Breaking any link causes security bypass.", tags=["pattern","auth","security"])
     ↓
[Makes Architectural Decision] → add_note(content="Decision: Keep JWT but add refresh tokens. Access: 15min, Refresh: 7 days. Chose over sessions for microservice scaling.", tags=["decision","auth"])
     ↓
[Context Gets Compressed/Summarized]
     ↓
[Agent Needs to Recall Decision] → search_notes(tags=["decision","auth"])
     ↓
[Retrieves Previous Decision & Rationale] → Applies consistent approach with same trade-offs
     ↓
[Addresses Insight/Decision] → scratch_note(id="note_123", scratched=true)
     ↓
[Checks Active Notes] → list_notes(tags=["decision"], include_scratched=false)
```

### Success States

- **Efficient Exploration**: Agent creates 5-10 focused notes capturing insights and decisions during large codebase exploration without overwhelming context
- **Preserved Context**: Agent maintains understanding of cross-component relationships and trade-offs even after context compression
- **Consistent Decisions**: All changes follow architectural pattern and rationale documented early in session
- **Tracked Progress**: Agent knows which insights have been addressed (scratched) vs still active through multi-step implementations

### Error/Edge States

- **Character Limit Exceeded**: Clear error message prompting agent to split into multiple notes or condense
- **Invalid Tag Count**: Validation error if <1 or >5 tags provided
- **Note Not Found**: Graceful handling when updating/deleting non-existent note ID
- **Empty Search Results**: Return empty array with helpful context about search parameters
- **Context Overflow**: If notes grow too large, agent can delete obsolete notes or filter out scratched ones to reduce context load

## User Interface & Interaction Design

### Key Interactions

**Phase 1 (Invisible to User)**:
- Notes are pure agent-internal mechanism
- No direct user interaction required
- Agent manages notes autonomously

**Phase 2 (Future - User Visibility)**:
- Optional TUI sidebar showing active notes
- User can view but not edit agent's notes
- Provides transparency into agent's working memory
- Export notes to file for documentation

### Information Architecture

Notes organize around three dimensions:
1. **Type**: What kind of insight (decision, pattern, bug, dependency, workaround, progress)
2. **Domain**: What system/component (auth, api, database, ui, config, test, build)
3. **Status**: Lifecycle stage (active, investigating, resolved, future)

### Progressive Disclosure

- **Core Operation**: Add/search notes - always available
- **Advanced Filtering**: Tag combinations and scratched status filters - used as needed
- **Management**: Update/delete - only when notes become obsolete
- **Analytics**: Tag usage stats - for agent optimization of tagging strategy

## Feature Metrics & Success Criteria

### Key Performance Indicators

- **Adoption**: Percentage of complex agent sessions (5+ tool calls) that create at least one note
- **Decision Quality**: Reduction in contradictory decisions within same session
- **Context Preservation**: Notes survive context compression and remain useful
- **Efficiency**: Reduction in backtracking and re-analysis of already-solved problems

### Success Thresholds

- **60% of complex tasks** (10+ tool calls) should create at least one insight/decision note
- **Zero instances** of contradictory architectural decisions in same session
- **Measurable reduction** in backtracking (re-analyzing already-solved problems)
- **Improved consistency** in multi-step implementations following documented patterns

## User Enablement

### Discoverability

**For Users**: Feature is transparent - users benefit from improved agent efficiency without needing to learn anything new

**For Agent**: System prompt includes notes capability with clear usage guidelines and examples

### Onboarding

**Agent Training**:
- System prompt emphasizes when to create notes (insights, decisions, patterns, trade-offs)
- Examples demonstrate focusing on relationships and rationale, not searchable facts
- Guidance on using notes for context that would require significant re-work to rediscover
- Clear anti-patterns: avoid noting file locations, function names, or temporary states

**No User Onboarding Needed**: Feature is invisible to users in initial release

### Mastery Path

**Agent Optimization**:
- Learn effective note content (insights and decisions vs searchable facts)
- Develop consistent tagging taxonomy aligned with note purpose
- Balance note creation vs context overhead (quality over quantity)
- Recognize when to scratch finished notes (keep for reference) vs delete obsolete ones (remove entirely)
- Understand when to search files vs search notes

## Risk & Mitigation

### User Risks

**Risk**: Agent creates too many notes with easily-searchable information, polluting its own context
- **Mitigation**: 800-character limit forces conciseness; agent prompt emphasizes insights over facts; usage guidelines document anti-patterns

**Risk**: Agent notes file locations instead of using search, creating maintenance burden
- **Mitigation**: Clear usage guidelines distinguish searchable facts from valuable insights; examples show good vs bad note content

**Risk**: Inconsistent tagging makes notes hard to find
- **Mitigation**: System prompt provides tag taxonomy examples; list_tags helps agent see patterns; focus on intent-based tags (decision, pattern, bug)

**Risk**: Notes contain incorrect/outdated information leading to bad decisions
- **Mitigation**: Agent can update/delete notes; timestamps help identify stale notes; scratch mechanism for resolved items

### Adoption Risks

**Risk**: Agent doesn't naturally use notes, feature goes unused
- **Mitigation**: Strong system prompt guidance with compelling examples showing value; comprehensive usage guidelines; measure adoption in telemetry

**Risk**: Agent misuses notes as file location cache instead of insight tracker
- **Mitigation**: Usage guidelines with clear good/bad examples; decision framework for when to create notes; emphasis on relationships over facts

**Risk**: Notes overhead negates performance benefits
- **Mitigation**: Strict character limits; usage philosophy of quality over quantity; encourage scratching finished work; measure net efficiency gains

## Dependencies & Integration Points

### Feature Dependencies

- **Agent Loop Architecture**: Notes must persist across loop iterations within session
- **Context Management**: Notes should survive context compression/summarization
- **Tool System**: New tools (add_note, search_notes, etc.) integrate with existing tool calling

### System Integration

- **Session Management**: Notes scoped to session lifecycle
- **Tool Call Schema**: Notes tools follow same XML schema as existing tools
- **Error Handling**: Validation errors follow standard error response format

### External Dependencies

- None - purely internal agent capability

## Constraints & Trade-offs

### Design Decisions

**Decision**: Session-scoped only (not persistent across sessions)
- **Rationale**: Simpler implementation; avoids stale/irrelevant notes accumulating; each session starts fresh
- **Trade-off**: Can't build long-term knowledge base; every session starts from zero

**Decision**: Insight-focused content (800 chars as safety limit)
- **Rationale**: Notes should capture insights, decisions, and relationships - not easily searchable facts like file locations
- **Trade-off**: Agent must think about what's worth noting vs what's better searched

**Decision**: Required tagging (1-5 tags mandatory)
- **Rationale**: Forces intentional categorization; makes search effective; encourages quality over quantity
- **Trade-off**: Agent must think about categorization; slight overhead per note creation

**Decision**: No hierarchical relationships between notes
- **Rationale**: Keeps data model simple; avoids complexity of graph traversal
- **Trade-off**: Can't express "note B depends on note A" explicitly

### Known Limitations

- No cross-session persistence (future consideration)
- No collaborative notes between multiple agent instances
- No user editing of agent notes (agent-only in v1)
- No automatic note summarization when count grows large
- Linear search only (no semantic similarity search)

### Future Considerations

**v2 Enhancements**:
- Optional persistence to workspace-local notes file
- UI for viewing agent's notes in real-time
- Note export for documentation/PR descriptions
- Auto-pruning of old scratched notes
- Semantic search using embeddings

**v3 Vision**:
- Shared notes across agent sessions for same workspace
- User-created notes that agent can reference
- Note templates for common patterns
- AI-powered note summarization and consolidation

## Competitive Analysis

**GitHub Copilot**: No working memory - each completion is stateless
**Cursor**: Some context awareness but no explicit note-taking mechanism for insights
**Aider**: Maintains conversation history but no structured notes for decisions/patterns
**Claude Code**: Can reference previous conversation but no organized system for tracking trade-offs

**Forge's Advantage**: Explicit working memory focused on insights, decisions, and relationships rather than just conversation history. Makes complex multi-step tasks more reliable through preserved context and consistent decision-making.

## Go-to-Market Considerations

### Positioning

"Forge now maintains working memory throughout complex tasks, reducing errors and improving efficiency by tracking discoveries, decisions, and progress automatically."

### Documentation Needs

**For Users**:
- Brief mention in release notes: "Improved agent memory for complex tasks"
- No user-facing documentation needed (transparent feature)

**For Developers**:
- System prompt guidelines for effective note usage
- ADR documenting notes data model and tool specifications
- Internal metrics dashboard for measuring adoption and impact

### Support Requirements

- Support team should understand notes exist but are agent-internal
- No user-facing troubleshooting needed unless notes visible in UI
- Telemetry to identify if notes causing performance issues

## Evolution & Roadmap

### Version History

- **v1.0**: Core note CRUD operations, search, tagging, scratch tracking (session-scoped)
- **v1.1**: Tag usage analytics, smart default limits, enhanced search relevance
- **v2.0**: Optional UI visibility, note export, workspace-local persistence

### Future Vision

**Advanced Memory System**:
- Hybrid short-term (notes) and long-term (persistent knowledge base)
- Automatic promotion of important patterns to long-term memory
- Cross-session learning from common note patterns
- Integration with project documentation for shared team knowledge

**Semantic Capabilities**:
- Embedding-based semantic search for conceptual note retrieval
- Automatic note clustering to identify related information
- Smart summarization when note count exceeds threshold

### Deprecation Strategy

Unlikely - core capability that becomes more valuable over time. If superseded, would migrate to more sophisticated memory system rather than remove.

## Technical References

- **Architecture**: [To be created - ADR for Notes Data Model]
- **Implementation**: [To be created - ADR for Notes Tool Integration]
- **API Specification**: Tool schemas defined in this PRD

## Appendix

### Research & Validation

**User Research Insights**:
- Developers naturally maintain mental notes during debugging (sticky notes, comments, scratch files)
- Context loss between terminal sessions frustrates complex multi-day tasks
- Consistency in large refactoring requires explicit tracking mechanisms

**Competitive Analysis**:
- Most AI assistants lack working memory, forcing users to re-explain context
- Some IDEs offer scratchpad features but not AI-integrated
- Opportunity to differentiate with AI that "thinks" more like human developer

### Design Artifacts

**Data Model Schema**:
```json
{
  "id": "note_[timestamp]",
  "content": "string (max 800 chars)",
  "tags": ["string"] (1-5 items),
  "created_at": "ISO 8601 timestamp",
  "updated_at": "ISO 8601 timestamp", 
  "scratched": boolean
}
```

**Tool Schemas**: See scratch document for complete XML specifications

**Tag Taxonomy Examples**:
- **Type**: decision, pattern, bug, dependency, workaround, progress
- **Domain**: auth, api, database, ui, config, test, build
- **Priority**: critical, blocking, tech-debt
- **Status**: investigating, resolved, future

**Note**: Avoid location-based tags - use domain tags instead to capture the system/component context
