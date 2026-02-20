# Context Summarization Bugs

## Date
2025-01-27

## Problem Statement
The `/context` view shows only 1 `[SUMMARIZED]` block despite the summarization tooltip having fired 5+ times. Token length does shrink on each summarization, but prior summaries disappear rather than accumulating. The expected message structure is:

```
[user1] [summary1] [user2] [summary2] ... [userN] [live messages]
```

## Root Cause 1: Previous summaries are silently dropped on each run

In `ToolCallSummarizationStrategy.Summarize` (`pkg/agent/context/tool_call_strategy.go`):

```go
retained := retainNonSummarizedMessages(oldMessages)  // BUG: drops old [SUMMARIZED] blocks
retained = append(retained, summarizedMessages...)     // only the NEW summary is added
retained = append(retained, recentMessages...)
```

`retainNonSummarizedMessages` keeps only `RoleSystem` and `RoleUser` messages. Previous `[SUMMARIZED]` blocks have `RoleAssistant` — they are silently dropped on every subsequent summarization run.

**Effect:** Run 1 produces `[SUMMARIZED_1]`. Run 2 produces `[SUMMARIZED_2]` but `[SUMMARIZED_1]` is gone. You always end up with exactly 1 summary block, containing only the most recent batch. All historical episodic memory is lost.

## Root Cause 2: Summary is placed after all user messages, breaking interleaved order

The reconstruction appends all retained user messages first, then the summary:

```
[sys] [user1] [user2] [user3] [SUMMARIZED] [recent...]
```

The correct structure is to insert the summary at the position where the first summarized tool call was, preserving the natural order of the conversation:

```
[sys] [user1] [SUMMARIZED_A] [user2] [SUMMARIZED_B] [user3] [recent...]
```

This matters because the LLM reads the context linearly — a summary placed after user messages it summarizes breaks causal ordering, making it harder for the model to correctly attribute which work happened in response to which user request.

## Fix

Instead of `retainNonSummarizedMessages` + append, use a pointer-set walk (the same pattern `GoalBatchCompactionStrategy` already uses correctly):

1. Build a `groupSet` of all message pointers that belong to groups being summarized.
2. Walk `oldMessages` in order; when hitting the first message in `groupSet`, insert the new summary block(s) and skip all grouped messages.
3. All other messages — system, user, **and existing `[SUMMARIZED]` blocks** — are kept in their original positions.

This ensures:
- Previous summaries are never dropped.
- New summaries are inserted at the correct position in the conversation timeline.
- The interleaved structure `[user] [summary] [user] [summary]` is maintained naturally.

## Files Affected
- `pkg/agent/context/tool_call_strategy.go` — `Summarize()` and `retainNonSummarizedMessages()`
- `pkg/agent/context/tool_call_strategy_test.go` — new tests for accumulation across multiple runs
