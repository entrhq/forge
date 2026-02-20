# 14. Composable Context Management with Strategy Pattern

**Status:** Proposed
**Date:** 2025-11-08
**Deciders:** Development Team
**Technical Story:** Implementation of intelligent context window management for long-running agent sessions with composable summarization strategies

---

## Context

The Forge agent currently maintains full conversation history in memory, including all user messages, assistant responses, tool calls, and tool results. While this provides complete context for the LLM, it leads to token exhaustion during long coding sessions.

### Background

Modern LLM-based coding agents often engage in extended sessions involving:
- Multi-step refactoring across dozens of files
- Large file read operations (thousands of lines)
- Iterative debugging with multiple test runs
- Complex architectural changes requiring many tool calls
- Long conversations with detailed explanations

Each of these activities consumes tokens rapidly. For example:
- Reading a 2000-line file: ~8,000 tokens
- A typical tool call/result pair: ~500-2,000 tokens
- Complex diffs and modifications: ~1,000-5,000 tokens each

Without context management, a coding session can easily consume 50,000-100,000+ tokens, exceeding model context windows (typically 128K-200K tokens) and degrading performance.

The existing [`Prune()`](../../pkg/agent/memory/conversation.go) method in [`conversation.go`](../../pkg/agent/memory/conversation.go) uses a simple strategy: it estimates tokens at 4 characters per token and removes older messages when approaching the limit. While functional, this approach has several limitations:
- Rough estimation leads to inaccurate token counting
- Removes entire messages rather than summarizing them
- No differentiation between critical and less important context
- Loss of valuable historical information that could inform future decisions

### Problem Statement

We need an intelligent context management system that:
1. Accurately tracks token usage using proper tokenization
2. Preserves critical context while compressing less relevant information
3. Supports different summarization strategies for different content types
4. Operates transparently without disrupting the agent's workflow
5. Provides user visibility into summarization activities

### Goals

- Implement accurate token counting using tiktoken
- Design composable strategy pattern for flexible summarization
- Create intelligent strategies that preserve semantic meaning
- Maintain agent loop correctness while keeping TUI responsive
- Enable long-running coding sessions without token exhaustion
- Provide clear user feedback during summarization operations

### Non-Goals

- Full semantic search or vector database integration
- Automatic detection of "important" vs "unimportant" content beyond heuristics
- User-configurable summarization strategies (initially - may add later)
- Real-time summarization during tool execution (happens between turns)
- Compression of the current turn's context (only historical context)

---

## Decision Drivers

* **Token Accuracy**: Using tiktoken provides exact token counts matching LLM billing
* **Context Quality**: LLM-based summarization preserves semantic meaning better than simple truncation
* **Composability**: Multiple strategies can address different aspects of context management
* **User Experience**: Users should understand when/why summarization occurs
* **Maintainability**: Clear strategy pattern makes it easy to add new strategies
* **Correctness**: Agent loop must have accurate context; blocking is acceptable
* **Responsiveness**: TUI must stay interactive even when agent loop blocks

---

## Considered Options

### Option 1: Simple Threshold-Based Pruning

**Description:** When token count exceeds a threshold (e.g., 80% of max), remove the oldest messages until under the threshold.

**Pros:**
- Simple to implement
- Fast execution
- Predictable behavior
- No LLM calls required

**Cons:**
- Loss of potentially valuable context
- No semantic understanding of importance
- Abrupt context discontinuity
- All content types treated equally
- Agent may lose track of long-running tasks

### Option 2: Adaptive Priority-Based Summarization

**Description:** Assign priority scores to different message types and summarize lower-priority content first. Use heuristics like message age, tool type, and result size.

**Pros:**
- Preserves more important context
- Gradual degradation of detail
- Flexible prioritization rules
- Can be tuned over time

**Cons:**
- Complex heuristic design
- Difficult to define "importance" objectively
- Still requires manual rule creation
- May not capture semantic importance
- One-size-fits-all summarization approach

### Option 3: Composable Strategy Pattern with LLM Summarization

**Description:** Use multiple composable strategies that can be combined and reused. Each strategy evaluates whether it should trigger and performs intelligent summarization using the LLM itself. Strategies focus on specific aspects (old tool calls, token thresholds, etc.).

**Pros:**
- Highly flexible and extensible
- Semantic summarization preserves meaning
- Strategies can be composed for different scenarios
- Clear separation of concerns
- LLM understands what it summarized
- Easy to add new strategies without modifying existing ones
- Each strategy has single responsibility

**Cons:**
- More complex implementation
- Additional LLM calls for summarization (cost and latency)
- Requires careful prompt design for summarization
- Strategy evaluation adds per-turn overhead
- Potential for strategies to conflict if not coordinated

---

## Decision

**Chosen Option:** Option 3 - Composable Strategy Pattern with LLM Summarization

### Rationale

The composable strategy pattern best addresses our needs while maintaining flexibility for future enhancements:

1. **Semantic Preservation**: LLM-based summarization understands the content and preserves what's important, unlike simple truncation

2. **Flexibility**: Different strategies can target different aspects of context management (age-based, size-based, type-based) and be combined as needed

3. **Extensibility**: New strategies can be added without modifying existing ones, following the Open/Closed Principle

4. **Single Responsibility**: Each strategy has one job and does it well (e.g., ToolCallSummarizationStrategy only handles old tool calls)

5. **Testability**: Strategies can be tested independently and composed for integration tests

6. **Context Quality**: The LLM that will receive the summarized context is the same LLM that created it, ensuring coherence

7. **User Transparency**: Event-based architecture allows TUI to show summarization progress, similar to command execution (ADR-0013)

The additional cost of LLM summarization calls is acceptable because:
- Summarization happens infrequently (only when thresholds trigger)
- Summarization is cheaper than sending full context repeatedly
- The alternative (losing context or session failure) is worse for user experience

---

## Consequences

### Positive

- Long coding sessions can continue indefinitely without token exhaustion
- Critical context (recent activity, system messages) is always preserved
- Semantic meaning maintained through intelligent summarization
- Clear visibility into when and why summarization occurs
- Foundation for future enhancements (vector search, etc.)
- Strategies can be tuned independently based on metrics
- Composable design allows different strategies for different agent types

### Negative

- Additional LLM API calls increase cost (typically small compared to main conversation)
- Summarization adds latency when triggered (blocks agent loop)
- Requires careful prompt engineering for quality summaries
- More complex than simple pruning approaches
- Per-turn strategy evaluation adds minor overhead
- Need to manage strategy execution order and potential conflicts

### Neutral

- Token counting switches from estimation to exact (may change behavior slightly)
- Agent loop blocks during summarization (but TUI stays responsive via events)
- Summarization happens between turns, not during tool execution
- Old conversation history becomes compressed rather than removed

---

## Implementation

### Core Strategy Interface

New package: `pkg/agent/context/`

```go
// Strategy defines the interface for context summarization strategies
type Strategy interface {
    // Name returns the strategy's identifier for logging and debugging
    Name() string
    
    // ShouldRun evaluates whether this strategy should execute on this turn
    // Parameters: current conversation, total tokens, max allowed tokens
    ShouldRun(conv *memory.Conversation, currentTokens, maxTokens int) bool
    
    // Summarize performs the actual summarization
    // Returns: modified conversation, tokens saved, error
    Summarize(ctx context.Context, conv *memory.Conversation, llm llm.Provider) (*memory.Conversation, int, error)
}
```

### Initial Strategies

#### 1. ToolCallSummarizationStrategy

Summarizes individual tool calls and their results when they exceed a configurable age threshold (default: 10 turns old).

```go
type ToolCallSummarizationStrategy struct {
    turnThreshold int  // Default: 10
}

func (s *ToolCallSummarizationStrategy) ShouldRun(conv *memory.Conversation, currentTokens, maxTokens int) bool {
    // Check if any tool call/result pairs are older than threshold
    currentTurn := conv.TurnCount()
    for _, msg := range conv.Messages() {
        if msg.Role == "assistant" && msg.HasToolCalls() {
            turnAge := currentTurn - msg.Turn
            if turnAge >= s.turnThreshold && !msg.IsSummarized {
                return true
            }
        }
    }
    return false
}
```

**Summarization prompt:**
```
You are summarizing an old tool call to compress context. Provide a concise summary including:

1. Tool name and purpose
2. Key input parameters (not full content)
3. Why the tool was called (inferred from context)
4. Summary of result (success/failure, key outcomes)

Original tool call: <tool_name>{name}</tool_name>
Input: {abbreviated inputs}
Result: {abbreviated result}

Provide a 2-3 sentence summary that captures the essential information.
```

#### 2. ThresholdSummarizationStrategy

Triggers when token usage exceeds a percentage threshold (default: 80% of max). This strategy is designed to perform a "half-compaction," reducing token count to a target percentage (e.g., 50%) to create headroom without fully summarizing the entire history.

It was temporarily disabled due to a bug causing infinite loops but has been re-enabled and refactored.

#### 3. GoalBatchCompactionStrategy

Summarizes old, completed turns into episodic memory blocks. A "turn" is defined as a user message followed by one or more `[SUMMARIZED]` blocks from the other strategies. This strategy addresses the "implicit floor" problem where the context fills with user messages and summary blocks that other strategies cannot compress further.

It fires proactively based on the age and count of completed turns, not reactively based on token pressure. See [ADR-0041](0041-goal-batch-compaction-strategy.md) for full details.

```go
type ThresholdSummarizationStrategy struct {
    thresholdPercent float64  // Default: 0.80 (80%)
}

func (s *ThresholdSummarizationStrategy) ShouldRun(conv *memory.Conversation, currentTokens, maxTokens int) bool {
    usagePercent := float64(currentTokens) / float64(maxTokens)
    return usagePercent >= s.thresholdPercent
}

func (s *ThresholdSummarizationStrategy) Summarize(ctx context.Context, conv *memory.Conversation, llm llm.Provider) (*memory.Conversation, int, error) {
    // Identify older messages (excluding recent N turns, system messages)
    // Group related messages (user message + assistant response)
    // Summarize each group with LLM
    // Replace original messages with summary messages
    // Return tokens saved
}
```

### Context Manager

Orchestrates strategy execution:

```go
type Manager struct {
    strategies []Strategy
    llm        llm.Provider
    tokenizer  *tokenizer.Tokenizer
    maxTokens  int
    eventChan  chan *types.AgentEvent
}

func (m *Manager) EvaluateAndSummarize(ctx context.Context, conv *memory.Conversation) error {
    // 1. Count current tokens accurately
    currentTokens := m.tokenizer.CountConversationTokens(conv)
    
    // 2. Check each strategy in order
    for _, strategy := range m.strategies {
        if strategy.ShouldRun(conv, currentTokens, m.maxTokens) {
            // 3. Emit start event
            m.emitEvent(&types.AgentEvent{
                Type: types.EventTypeContextSummarizationStart,
                Data: &types.ContextSummarizationStart{
                    Strategy:      strategy.Name(),
                    CurrentTokens: currentTokens,
                    MaxTokens:     m.maxTokens,
                },
            })
            
            // 4. Execute strategy (blocking)
            newConv, saved, err := strategy.Summarize(ctx, conv, m.llm)
            if err != nil {
                m.emitEvent(&types.AgentEvent{
                    Type: types.EventTypeContextSummarizationError,
                    Data: &types.ContextSummarizationError{
                        Strategy: strategy.Name(),
                        Error:    err.Error(),
                    },
                })
                return err
            }
            
            // 5. Emit completion event
            m.emitEvent(&types.AgentEvent{
                Type: types.EventTypeContextSummarizationComplete,
                Data: &types.ContextSummarizationComplete{
                    Strategy:      strategy.Name(),
                    TokensSaved:   saved,
                    NewTokenCount: currentTokens - saved,
                },
            })
            
            // 6. Update conversation reference
            *conv = *newConv
            currentTokens -= saved
        }
    }
    
    return nil
}
```

### Event System Extensions

Add to `pkg/types/event.go`:

```go
EventTypeContextSummarizationStart    EventType = "context_summarization_start"
EventTypeContextSummarizationProgress EventType = "context_summarization_progress"
EventTypeContextSummarizationComplete EventType = "context_summarization_complete"
EventTypeContextSummarizationError    EventType = "context_summarization_error"
```

Event payloads:

```go
type ContextSummarizationStart struct {
    Strategy      string
    CurrentTokens int
    MaxTokens     int
    Timestamp     time.Time
}

type ContextSummarizationProgress struct {
    Strategy       string
    ItemsProcessed int
    TotalItems     int
    TokensSaved    int
    Timestamp      time.Time
}

type ContextSummarizationComplete struct {
    Strategy      string
    TokensSaved   int
    NewTokenCount int
    Duration      time.Duration
    Timestamp     time.Time
}

type ContextSummarizationError struct {
    Strategy  string
    Error     string
    Timestamp time.Time
}
```

### Agent Loop Integration

In `pkg/agent/default.go`, add per-turn evaluation:

```go
func (a *DefaultAgent) processInput(ctx context.Context, input *types.UserInput) error {
    // ... existing code ...
    
    // Before each LLM call, evaluate context strategies
    if err := a.contextManager.EvaluateAndSummarize(ctx, a.conversation); err != nil {
        return fmt.Errorf("context summarization failed: %w", err)
    }
    
    // Continue with normal LLM call
    stream, err := a.llm.CreateChatCompletionStream(ctx, a.conversation.Messages())
    // ... rest of processing ...
}
```

### TUI Integration

Similar to command execution overlay (ADR-0013), add status indicator:

```go
// In pkg/executor/tui/executor.go event handler
case types.EventTypeContextSummarizationStart:
    data := event.Data.(*types.ContextSummarizationStart)
    e.showSummarizationStatus(data.Strategy, data.CurrentTokens, data.MaxTokens)

case types.EventTypeContextSummarizationProgress:
    data := event.Data.(*types.ContextSummarizationProgress)
    e.updateSummarizationProgress(data)

case types.EventTypeContextSummarizationComplete:
    data := event.Data.(*types.ContextSummarizationComplete)
    e.hideSummarizationStatus()
    e.showToast(fmt.Sprintf("Context optimized: saved %d tokens", data.TokensSaved))
```

**TUI Display:**
- Small status indicator in header/footer (not full overlay since user can't interact)
- Shows: "🔄 Optimizing context... (45% complete)"
- On completion: Success toast with tokens saved
- On error: Error toast with brief message

### Configuration

Add to agent configuration:

```go
type ContextConfig struct {
    MaxTokens            int     // Default: 100000 (leave headroom for model's 128K limit)
    ThresholdPercent     float64 // Default: 0.80 (80%)
    ToolCallTurnAge      int     // Default: 10 turns
    EnableSummarization  bool    // Default: true
}
```

### Usage Example

End users configure context management by providing strategies when creating an agent:

```go
// Example 1: Default configuration (recommended for most use cases)
agent := agent.NewDefaultAgent(
    provider,
    agent.WithContextManagement(
        context.NewToolCallSummarizationStrategy(10),  // Summarize tool calls > 10 turns old
        context.NewThresholdSummarizationStrategy(
            100000,  // Max tokens
            0.80,    // Trigger at 80%
        ),
    ),
)

// Example 2: Custom configuration for long sessions
agent := agent.NewDefaultAgent(
    provider,
    agent.WithContextManagement(
        // Summarize tool calls earlier (more aggressive)
        context.NewToolCallSummarizationStrategy(5),
        
        // Higher token limit, lower threshold
        context.NewThresholdSummarizationStrategy(
            150000,  // Use more of the model's 200K context window
            0.75,    // Start summarizing at 75% (112,500 tokens)
        ),
    ),
)

// Example 3: Conservative configuration (preserve more context)
agent := agent.NewDefaultAgent(
    provider,
    agent.WithContextManagement(
        // Only summarize very old tool calls
        context.NewToolCallSummarizationStrategy(20),
        
        // Trigger later to preserve more raw context
        context.NewThresholdSummarizationStrategy(
            100000,
            0.90,    // Wait until 90% before summarizing
        ),
    ),
)

// Example 4: Disable context management (not recommended for long sessions)
agent := agent.NewDefaultAgent(
    provider,
    // No WithContextManagement option - uses basic pruning only
)
```

**Strategy Execution Order:**

Strategies execute in the order they're provided to `WithContextManagement()`. Best practice:
1. Place specific strategies first (e.g., ToolCallSummarizationStrategy)
2. Place broad strategies last (e.g., ThresholdSummarizationStrategy as safety net)

This ensures targeted optimizations happen before general pruning.

---

## Validation

### Success Metrics

- Context summarization reduces token usage by 40-60% for sessions > 50K tokens
- Long coding sessions (100+ turns) complete without context window errors
- Summarization completes within 5 seconds for typical scenarios
- TUI remains responsive during summarization (no visible freezing)
- Agent maintains coherent understanding of task despite summarization
- No critical context loss (recent activity, current task always preserved)
- Token counting accuracy within 2% of actual LLM billing

### Monitoring

- Track summarization frequency and duration per session
- Measure tokens saved vs. summarization cost
- Monitor strategy trigger rates (which strategies activate most)
- Collect user feedback on context quality after summarization
- Log any context-related errors or confusion in agent responses
- Track long-session completion rates before and after implementation

### Testing Scenarios

1. **Long Coding Session** - 100+ turn session with multiple file edits
2. **Large File Operations** - Reading and modifying files > 5000 lines
3. **Repeated Tool Calls** - Same tool called 20+ times (e.g., debugging loop)
4. **Strategy Composition** - Both strategies trigger in single session
5. **Token Threshold Edge Cases** - Approach 100% of max tokens
6. **Summarization Quality** - Agent can still reference old summarized content
7. **TUI Responsiveness** - UI stays interactive during blocking summarization

---

## Related Decisions

- [ADR-0005](0005-channel-based-agent-communication.md) - Channel-based communication enables event streaming for summarization status
- [ADR-0007](0007-memory-system-design.md) - Memory system design provides foundation for conversation management
- [ADR-0013](0013-streaming-command-execution.md) - Established pattern for blocking operations with event-based TUI updates
- [ADR-0040](0040-structured-summarization-prompt.md) - Defines the shift to structured, first-person episodic memory prompts used by all summarization strategies.
- [ADR-0041](0041-goal-batch-compaction-strategy.md) - Details the `GoalBatchCompactionStrategy`.
- [ADR-0042](0042-summarization-model-override.md) - Documents the ability to use a separate model for summarization tasks.
- Future ADR - Vector database integration could enhance strategy selection

---

## Implementation Notes

### Blocking vs. Async Design Rationale

**Why Agent Loop Blocks:**

Context summarization must complete before the next LLM call to ensure accuracy. If summarization were async:
1. Agent might send old context to LLM (wasting tokens)
2. Race conditions between summarization and conversation updates
3. Difficult to guarantee consistency of conversation state
4. Agent might make decisions based on stale token counts

**Why TUI Stays Responsive:**

Following the pattern from [ADR-0013](0013-streaming-command-execution.md), the agent emits events during the blocking operation:
1. Agent loop blocks in `EvaluateAndSummarize()`
2. Context manager emits Start/Progress/Complete events
3. TUI event handler processes events asynchronously
4. User sees progress indicator and remains informed
5. No user input needed (unlike tool approval), so blocking is acceptable

This architecture prioritizes **correctness** (agent loop) while maintaining **usability** (responsive TUI).

### Strategy Execution Order

Strategies execute in the order they are registered with the Context Manager. The recommended order is:

1.  **`ToolCallSummarizationStrategy`**: Compresses raw tool call/result pairs into `[SUMMARIZED]` blocks. This is the most granular and frequently-run strategy.
2.  **`ThresholdSummarizationStrategy`**: When total token count exceeds a high-water mark, it compacts older assistant message blocks into `[SUMMARIZED]` blocks. This acts as a primary defense against token overflow.
3.  **`GoalBatchCompactionStrategy`**: After many turns, the context can fill with `user` messages and `[SUMMARIZED]` blocks. This strategy compacts these completed "turns" into higher-level `[GOAL BATCH]` summaries, ensuring long-session stability.

This order ensures that low-level noise is cleaned up first, followed by reactive compaction, and finally proactive, long-term memory management.

### Summarization Prompt Design Principles

**For Tool Calls:**
- Focus on semantic meaning, not verbatim content
- Capture the "why" (intent) not just the "what" (action)
- Include outcome (success/failure) for context continuity
- Keep summaries concise (2-3 sentences max)

**For Conversation Groups:**
- Preserve key decisions and conclusions
- Maintain causal relationships between exchanges
- Note any errors or problems encountered
- Omit repetitive content (e.g., multiple similar attempts)

### Future Enhancement Opportunities

1. **Vector Database Integration**
   - Store full context in vector DB
   - Use semantic search to retrieve relevant context
   - Only send most relevant historical context to LLM

2. **User-Configurable Strategies**
   - Allow users to enable/disable strategies
   - Adjust thresholds per user preference
   - Create custom strategies via configuration

3. **Intelligent Strategy Selection**
   - ML model predicts which strategy will be most effective
   - Adaptive thresholds based on session characteristics
   - Learn from user feedback on summary quality

4. **Multi-Level Summarization**
   - Progressive summarization (summary of summaries)
   - Hierarchical context structure
   - Importance scoring for content retention

---

## References

- [Tiktoken](https://github.com/openai/tiktoken) - Accurate token counting library
- [Strategy Pattern](https://refactoring.guru/design-patterns/strategy) - Design pattern reference
- [Context Window Management](https://www.anthropic.com/index/prompting-long-context) - Best practices for long contexts

---

## Notes

### Design Philosophy

The composable strategy approach embodies several key principles:

1. **Separation of Concerns**: Each strategy handles one aspect of context management
2. **Open/Closed Principle**: New strategies can be added without modifying existing code
3. **Composability**: Strategies work together to solve the problem holistically
4. **Transparency**: Events provide visibility into what's happening and why
5. **Pragmatism**: Accept LLM cost/latency trade-off for better semantic quality

### Comparison with ADR-0013

This ADR follows a very similar pattern to [ADR-0013 (Streaming Command Execution)](0013-streaming-command-execution.md):

| Aspect | Command Execution | Context Summarization |
|--------|------------------|----------------------|
| **Blocking** | Agent loop blocks waiting for command | Agent loop blocks during summarization |
| **Events** | Start, Output chunks, Complete, Error | Start, Progress, Complete, Error |
| **TUI Response** | Show overlay with live output | Show status indicator with progress |
| **User Control** | Can cancel with Ctrl+C | No interaction needed (automatic) |
| **Duration** | Varies (seconds to minutes) | Typically 2-5 seconds |
| **Frequency** | Every execute_command call | Infrequent (when thresholds met) |

Both ADRs demonstrate how to handle **blocking operations with responsive UI** using the event-driven architecture established in [ADR-0005](0005-channel-based-agent-communication.md).

### Alternative Approaches Not Chosen

**Summarization During Tool Execution:**
- Could summarize while waiting for tools to complete
- Rejected because: tool execution time unpredictable, adds complexity, may not save time if tool is quick

**Continuous/Real-time Summarization:**
- Summarize every N turns automatically regardless of token count
- Rejected because: wasteful if tokens are low, adds unnecessary latency, harder to tune

**Client-Side Summarization:**
- Use local models or simple algorithms instead of LLM API
- Rejected because: lower quality summaries, agent wouldn't understand them as well, requires additional dependencies

---

**Last Updated:** 2025-11-08
**Implementation Status:** Proposed