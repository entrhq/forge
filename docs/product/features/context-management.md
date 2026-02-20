# Context Management & Visibility

## Product Vision

Context management is fundamental to Forge's effectiveness as an AI coding assistant. As conversations grow longer and tool outputs accumulate, the LLM's context window fills up, potentially causing the agent to "forget" earlier context or fail entirely. This feature provides transparency into context usage and automatically manages context size, enabling longer, more productive coding sessions without manual intervention or context loss.

**Strategic Purpose:**
- **Enable long-running sessions** without context overflow errors
- **Build user trust** through transparency into what the agent "knows"
- **Reduce friction** by eliminating manual context management
- **Improve cost efficiency** by optimizing token usage

## Key Value Propositions

- **For Junior Developers**: Confidence that the agent maintains full context throughout extended debugging sessions, without worrying about technical limitations
- **For Senior Engineers**: Visibility into token usage and automatic optimization during complex refactoring tasks involving many files
- **For Cost-Conscious Users**: Automatic reduction of token consumption through intelligent summarization, lowering API costs
- **Competitive Advantage**: Proactive context management vs reactive "context full" errors in other tools

## Target Users & Use Cases

### Primary Personas

**1. Extended Session User**
- **Role**: Developer working on complex, multi-file features
- **Goals**: Maintain full context across 30+ minute sessions
- **Pain Points**: Other tools lose context mid-conversation, forcing restarts

**2. Cost-Conscious Developer**
- **Role**: Individual developer or small team watching API costs
- **Goals**: Understand and optimize token usage
- **Pain Points**: Unexpected high costs from inefficient context management

**3. Debugging Power User**
- **Role**: Developer investigating complex bugs across many files
- **Goals**: Keep full debugging history available while exploring codebase
- **Pain Points**: Context limits prevent thorough investigation

### Core Use Cases

1. **Long Refactoring Session**: Developer refactoring architecture across 15 files over 45 minutes. Agent automatically summarizes old file reads while keeping recent context, maintaining continuity without hitting limits.

2. **Token Budget Management**: Developer checks `/context` before starting large task, sees 60% usage, decides to start fresh session to avoid mid-task interruption.

3. **Debugging Investigation**: Developer reading many files to trace bug. Tool call summarization compresses old file reads to summaries, freeing space for new explorations while retaining key findings.

## Product Requirements

### Must Have (P0)

- **Context Visibility Command**: `/context` slash command shows real-time context statistics
- **Token Usage Display**: Clear visualization of current vs max tokens with percentage
- **Automatic Summarization**: System automatically reduces context size when approaching limits
- **Non-Disruptive Operation**: Summarization happens seamlessly without interrupting user
- **Progress Feedback**: Visual indication when summarization is occurring

### Should Have (P1)

- **Detailed Breakdown**: Show token allocation (system prompt, tools, messages)
- **Multiple Strategies**: Different summarization approaches (threshold-based, tool-focused)
- **Cumulative Tracking**: Total session token usage across all API calls
- **Smart Tool Handling**: Preserve recent tool calls, summarize old ones

### Could Have (P2)

- **Cost Estimation**: Show approximate API cost based on token usage
- **Workspace Statistics**: Display info about accessible files/directories
- **Historical Trends**: Track context usage over time
- **Export Capability**: Save context snapshot for debugging via `/exportcontext`

## User Experience Flow

### Entry Points

**Primary**: `/context` slash command in TUI
- Discoverable through autocomplete
- Available at any time during conversation
- Non-modal, doesn't interrupt workflow

**Secondary**: Automatic summarization
- Triggered automatically by system
- Brief notification overlay during processing
- No user action required

### Core User Journey

```
User working on task → Context grows → User types /context
                                              ↓
                                        [View statistics]
                                              ↓
                                  [Decision: Continue or restart?]
                                      ↙                    ↘
                            Continue working          Start fresh session
                                  ↓                          ↓
                        Context reaches 80%              Clean context
                                  ↓
                        Automatic summarization
                                  ↓
                        Brief progress indicator
                                  ↓
                        User continues uninterrupted
```

### Success States

- User sees context overlay and understands current usage
- User makes informed decision about session continuation
- Automatic summarization completes without user noticing
- Context stays below 90% throughout session
- User completes task without context overflow errors

### Error/Edge States

- **Context overlay shows >95% usage**: Warning color (red) with suggestion to start fresh
- **Summarization fails**: Graceful degradation - continue without summarization, notify user
- **Very short context window**: Disable automatic summarization, rely on user awareness
- **Rapid context growth**: Multiple summarizations in quick succession - show cumulative savings

## User Interface & Interaction Design

### Key Interactions

**Context Overlay (`/context` command)**:
- Modal dialog with scrollable content
- Fixed width for consistency (80 columns)
- Organized sections with clear headers
- Color-coded progress bar (green/yellow/red)
- ESC or Enter to dismiss

**Summarization Status**:
- Brief overlay during processing
- Shows strategy name and progress
- Displays token savings in real-time
- Auto-dismisses on completion

### Information Architecture

```
Context Information Overlay
├── System Section
│   ├── System prompt tokens
│   └── Custom instructions status
├── Tool System Section  
│   ├── Available tools count
│   └── Tool tokens
├── Message History Section
│   ├── Total messages
│   ├── Conversation turns
│   └── Conversation tokens
├── Current Context Section
│   ├── Used vs max tokens (percentage)
│   ├── Visual progress bar
│   └── Free space remaining
└── Cumulative Usage Section
    ├── Total input tokens
    ├── Total output tokens
    └── Session total
```

### Progressive Disclosure

- **Level 1** (Always visible): Overall percentage, progress bar
- **Level 2** (In overlay): Breakdown by component (system, tools, messages)
- **Level 3** (Future): Per-message token counts, detailed history

## Feature Metrics & Success Criteria

### Key Performance Indicators

**Adoption Metrics:**
- \>40% of users check context info at least once per session
- \>60% of power users (sessions >20 min) use `/context` regularly
- \>80% feature discovery within first week

**Effectiveness Metrics:**
- 70% reduction in context overflow errors vs baseline
- 25% decrease in average token usage per session (via summarization)
- \>90% of sessions stay below 90% context usage
- Average 2-3 automatic summarizations per 30-minute session

**Usability Metrics:**
- \>85% of users understand context info on first view
- \<5% of users report confusion about summarization
- \>30% of context checks lead to user action (restart, optimization)

### Success Thresholds

**Minimum Viable:**
- Context visibility works correctly 100% of time
- Automatic summarization prevents overflow in 90% of cases
- No user-reported data loss from summarization

**Target:**
- \<2s average summarization time
- 40%+ token reduction from summarization
- \<1% user complaints about summarization quality

## User Enablement

### Discoverability

- `/context` appears in slash command autocomplete
- First-time welcome message mentions context management
- Documentation includes context management section
- In-app help text explains feature

### Onboarding

**First Use:**
1. User types `/context` (or discovers via autocomplete)
2. Overlay shows current statistics with brief explanatory text
3. User understands current context state
4. User discovers feature is always available

**First Summarization:**
1. Brief notification: "Optimizing context..."
2. Progress overlay shows token savings
3. User sees work continues uninterrupted
4. User trusts automatic management

### Mastery Path

**Novice**: Checks context occasionally, relies on automatic summarization
**Intermediate**: Proactively checks before large tasks, understands token budgeting
**Expert**: Uses context info to optimize workflow, starts fresh at strategic points

## Risk & Mitigation

### User Risks

**Risk**: Summarization loses critical context
- **Impact**: High - User loses important information
- **Probability**: Low
- **Mitigation**: 
  - Conservative summarization (only old messages)
  - Preserve recent context completely
  - Test summarization quality extensively
  - Allow user to view original messages if needed (future)

**Risk**: Users don't understand token counts
- **Impact**: Medium - Feature provides less value
- **Probability**: Medium
- **Mitigation**:
  - Use percentages and progress bars (more intuitive)
  - Provide human-readable formatting (K/M suffixes)
  - Color-code for quick understanding
  - Include "what this means" explanatory text

**Risk**: Summarization interrupts workflow
- **Impact**: High - User experience degraded
- **Probability**: Low
- **Mitigation**:
  - Make summarization fast (<3s target)
  - Show progress to set expectations
  - Never block user input
  - Auto-dismiss completion notification

### Adoption Risks

**Risk**: Users don't discover `/context` command
- **Mitigation**: 
  - Include in autocomplete
  - Mention in welcome message
  - Document prominently
  - Consider proactive suggestion at high usage

**Risk**: Users distrust automatic summarization
- **Mitigation**:
  - Clear messaging about what's being summarized
  - Show token savings to demonstrate value
  - Provide transparency into process
  - Never lose user messages (only summarize tool results/agent responses)

## Dependencies & Integration Points

### Feature Dependencies

- **Memory System**: Must access conversation history for counting and summarization
- **Agent Loop**: Integration point for triggering summarization
- **Tool System**: Knowledge of available tools for token counting
- **TUI System**: Overlay rendering and slash command handling

### System Integration

- **LLM Provider**: Uses provider for generating summaries
- **Token Counter**: Requires accurate tokenization library
- **Event System**: Emits events for TUI updates during summarization

### External Dependencies

- Tokenization library (tiktoken-go or equivalent)
- LLM API for generating summaries
- No third-party analytics or tracking

## Constraints & Trade-offs

### Design Decisions

**Decision**: Automatic vs Manual Summarization
- **Chosen**: Automatic with visibility
- **Rationale**: Reduces user burden, prevents errors, but provide transparency
- **Trade-off**: Less user control, but better UX for most users

**Decision**: Multiple Summarization Strategies
- **Chosen**: Pluggable strategy system
- **Rationale**: Different use cases benefit from different approaches
- **Trade-off**: More complexity, but better optimization

**Decision**: Real-time Token Counting
- **Chosen**: Count on every context check
- **Rationale**: Always accurate, worth small performance cost
- **Trade-off**: Slight overhead, but negligible in practice

### Known Limitations

- Token counts are estimates (±2% accuracy)
- Summarization quality depends on LLM capability
- Cannot recover original messages after summarization
- Context display is read-only (no editing)

### Future Considerations

- Message-level token breakdown (detailed view mode)
- Context export for debugging (`/exportcontext`)
- Multi-session context tracking
- Advanced analytics and trends

## Competitive Analysis

**GitHub Copilot Chat**: No context visibility, frequent context overflow errors
**Cursor**: Basic token counter, no automatic management
**Aider**: Manual `/clear` command, user must manage context
**Continue**: No context management features

**Forge Advantage**: Automatic + transparent. Users get best of both worlds - hands-off management with full visibility when needed.

## Go-to-Market Considerations

### Positioning

**Message**: "Never lose context. Forge automatically manages your conversation, keeping what matters while optimizing what doesn't. Check `/context` anytime to see what your AI assistant knows."

**Key Benefits**:
- No context overflow errors
- Longer, more productive sessions
- Lower token costs
- Full transparency

### Documentation Needs

- How-to guide: "Understanding Context Management"
- FAQ: "What is token usage and why does it matter?"
- Troubleshooting: "Context-related issues"
- Best practices: "Optimizing long coding sessions"

### Support Requirements

Support teams should know:
- How to interpret context statistics
- When summarization occurs and why
- Troubleshooting summarization failures
- Explaining token counting to users

## Evolution & Roadmap

### Version History

- **v1.0**: Core visibility and automatic summarization
- **v1.1**: Multiple strategies, improved UI
- **v2.0**: Cost estimation, workspace stats (future)

### Future Vision

**Phase 2** (3-6 months):
- Cost estimation and budgeting
- Workspace context in overlay
- Message-level token breakdown
- Context export capability (implemented with `/exportcontext`)

**Phase 3** (6-12 months):
- Multi-session context tracking
- Advanced analytics and trends
- AI-powered summarization optimization
- Context sharing and templates

### Deprecation Strategy

Not applicable - core feature integral to product.

## Technical References

- **Architecture**: See `docs/architecture/context-management.md` for implementation details
- **API Reference**: `pkg/agent/context/` package documentation
- **Configuration**: Context manager and strategy configuration options

## Related Documentation

- **Agent Loop**: `docs/product/agent-loop.md`
- **Tool System**: `docs/product/tool-system.md`  
- **Memory System**: `docs/product/memory-system.md`
- **TUI Interface**: `docs/product/tui-executor.md`
