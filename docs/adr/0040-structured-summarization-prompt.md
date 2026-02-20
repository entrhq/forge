# 40. Structured Summarization Prompt for Agent Context Compression

**Status:** Accepted
**Date:** 2025-01-31
**Deciders:** Development Team
**Technical Story:** Improve the quality and utility of LLM-generated conversation summaries used in context compression by replacing the generic summarization prompt with a structured, agent-aware prompt designed for AI-to-AI consumption.

---

## Context

### Background

When an agent's conversation history exceeds a configurable token threshold, older blocks of assistant messages are compressed into a single summary message via a call to the LLM (`ThresholdSummarizationStrategy`). This summary is injected back into the conversation history and must carry enough context for the agent to continue working coherently.

The summarization is performed by a secondary LLM call with a dedicated system + user prompt pair. The quality of this call directly determines whether the agent can maintain continuity across context compression boundaries.

### Problem Statement

The original summarization prompt is designed as if the consumer is a human reader:

> "You are a helpful assistant that creates concise summaries of agent conversations. You are excellent at preserving important context."

> "Please create a concise summary of the following conversation messages. Preserve key information, decisions, and context. Be brief but comprehensive."

This fails in several ways for an agent context-compression use case:

1. **Wrong consumer model.** The summary is not read by a human — it is injected into a future LLM context window. Writing for human readability produces verbose, hedged prose rather than dense, actionable technical state.

2. **No structure.** A free-form paragraph makes it hard for the consuming agent to quickly locate the most decision-relevant information (what was decided, what failed, what is in flight).

3. **No negative guidance.** The prompt does not tell the summarizer what to omit. It therefore includes conversational filler, re-statement of the obvious, and hedging language that wastes tokens without informing the next action.

4. **Critical artifacts under-specified.** File paths, function names, variable names, error messages, and test names are the concrete artifacts a coding agent needs to resume work. The prompt does not call these out as mandatory to preserve.

5. **Dead-end amnesia.** Approaches that were attempted and abandoned are high-value context — they prevent the agent from re-trying the same failed path. The original prompt has no mechanism for capturing them.

### Goals

- Produce summaries that are optimised for consumption by an LLM coding agent, not a human.
- Impose a consistent structure that separates completed milestones, key decisions, findings, abandoned approaches, current state, and open items.
- Guarantee preservation of concrete technical artifacts (file paths, function names, error strings, test names).
- Reduce wasted tokens by explicitly excluding conversational filler and obvious re-statements.
- Require the summarizer to record why decisions were made, not just what was decided.

### Non-Goals

- Changing which messages are selected for summarization (governed by `collectMessagesToSummarize`).
- Changing when summarization is triggered (governed by `ShouldRun`).
- Supporting multiple output formats or making the prompt configurable at runtime.
- Optimising for human readability of the summary.

---

## Decision Drivers

* **Agent continuity**: The agent must be able to resume complex multi-step work seamlessly after context compression without repeating already-done steps or re-attempting failed approaches.
* **Token efficiency**: Summaries occupy permanent space in the context window; every wasted token is a token that cannot hold live conversation.
* **Decision traceability**: Rationale for key choices (algorithm selected, file structure adopted, bug root cause) must survive compression so the agent does not contradict past decisions.
* **Failure memory**: Recording abandoned approaches prevents costly re-investigation of known dead ends.
* **Concrete artifact preservation**: Coding agents operate on named things — files, functions, tests. Losing these names forces re-discovery.

---

## Considered Options

### Option 1: Keep the existing generic prompt

**Description:** Leave the system and user prompts unchanged. Accept that summary quality is limited.

**Pros:**
- No change required.
- Works acceptably for short or simple tasks.

**Cons:**
- Produces human-oriented prose unsuited for LLM consumption.
- No guaranteed structure; downstream agent must infer state from narrative text.
- No mechanism for preserving failure history.
- Critical technical artifacts may be omitted or paraphrased into ambiguity.

### Option 2: Structured prompt with named sections and explicit agent-consumer framing

**Description:** Replace the system prompt with one that frames the summarizer as a "technical memory encoder" whose output will be read by another LLM agent. Replace the user prompt with a structured template that mandates six named sections and provides explicit positive and negative guidance.

**Pros:**
- Output is optimised for the actual consumer (an LLM).
- Consistent structure makes it easy for the consuming agent to locate specific information.
- Mandatory sections ensure milestones, decisions, failures, and open items are always captured.
- Explicit negative guidance reduces filler and wasted tokens.
- Concrete artifact preservation is called out as a first-class requirement.

**Cons:**
- Slightly longer user prompt (~30 extra tokens) — a one-time cost per compression event, not per turn.
- Requires the summarizing LLM to follow structured output instructions reliably (modern LLMs do this well).

### Option 3: JSON-structured output

**Description:** Ask the summarizer to emit JSON with named fields rather than a formatted markdown summary.

**Pros:**
- Machine-parseable; fields could be post-processed or indexed.

**Cons:**
- JSON increases token cost due to quoting and syntax overhead.
- The consuming agent reads the summary as prose in its context window — structured JSON is no easier for an LLM to parse than formatted markdown headings.
- Adds implementation complexity (JSON parsing, error handling for malformed output).
- Provides no practical benefit over structured markdown for the LLM consumer.

---

## Decision

**Chosen Option:** Option 2 — Structured prompt with named sections and explicit agent-consumer framing.

### Rationale

The core insight driving this decision is that **the consumer of the summary is an LLM, not a human**. Once that constraint is recognised, the requirements for the prompt change substantially: density and specificity are more valuable than readability, structure is more valuable than narrative flow, and explicit enumeration of what to omit is as important as specifying what to include.

Option 2 satisfies all goals with a modest increase in prompt token cost. Option 3 adds complexity without a meaningful advantage given that the consuming agent reads context as natural language. Option 1 is ruled out because it produces demonstrably inadequate summaries for complex multi-step coding tasks.

---

## Implementation

### System Prompt

```
You are a technical memory encoder for an AI coding agent. Your summaries
replace a section of that agent's conversation history. The agent must be
able to continue its work seamlessly by reading your summary alone — write
for an AI consumer, not a human reader. Be dense, specific, and technical.
```

### Conversation Block User Prompt (`threshold_strategy.go`)

The prompt instructs the summarizer to produce exactly six sections:

1. **Milestones** — Work that was completed (files created/edited, tests passed, features shipped, commands run successfully).
2. **Key Decisions** — What was chosen and the rationale (not just "used X" but "used X because Y").
3. **Findings** — Important discoveries: bugs identified, constraints uncovered, patterns observed, API behaviors confirmed.
4. **Attempted & Abandoned** — Approaches that were tried and discarded, and why. This is the highest-value section for preventing re-investigation of dead ends.
5. **Current State** — Precise description of where things stand right now: what is in progress, what is partially complete, what is broken.
6. **Open Items** — Unresolved questions, blockers, and confirmed next steps.

The prompt also provides explicit mandatory-preserve and do-not-include guidance for both strategies:

**Must preserve:** file paths, function names, variable names, error messages, test names, line numbers where relevant.

**Must not include:** conversational filler, re-statements of what is already obvious from context, hedging language ("it seems like", "possibly", "I think"), offers of help, apologies.

### Tool Call Batch Prompt (`tool_call_strategy.go`)

All eligible tool call groups are sent to the LLM in a single call (`summarizeBatch`). The original per-call parallelism (`summarizeGroupsParallel`) was replaced because:

- **Fewer API calls**: N tool calls → 1 LLM call regardless of batch size.
- **Cross-call context**: The summarizer sees the full operation sequence and can infer connecting intent across calls — something per-call summarization cannot do.

When a preceding user message is available it is prepended as a `## User Goal` section so the summarizer can evaluate strategy success relative to what was actually requested.

The prompt then produces a structured output with up to seven sections (sections with nothing meaningful to report are omitted):

| Section | Content |
|---|---|
| **Strategy** | One sentence — the approach taken to address the goal |
| **Operations** | One line per tool call: `**tool_name** \| Inputs: <exact values> \| Outcome: <success/failure + key result data>` |
| **Discoveries** | Facts confirmed or found: file contents, API behaviour, test results, existing code structure (exact values required) |
| **Dead Ends** | Approaches tried and abandoned, written as personal lessons: "I tried X — abandoned because Y" |
| **What Worked** | The approach that succeeded, with enough detail to build on |
| **Critical Artifacts** | Exact file paths, function names, error strings, command outputs, line numbers, test names — one item per line |
| **Status** | One of `COMPLETE \| PARTIAL \| BLOCKED` followed by one sentence on current state |

The numbered raw message blocks (`--- Operation N ---`) give the LLM a stable reference for each group while keeping the output concise. The same artifact-preservation and no-filler rules apply, with additional instructions to strip XML markup and role labels and to use strict first-person voice throughout ("I tried", "I found" — never "the agent").

### Files Changed

- `pkg/agent/context/threshold_strategy.go` — `generateSummary` (system prompt string) and `buildSummarizationPrompt` (user prompt builder).
- `pkg/agent/context/tool_call_strategy.go` — replaced per-call `summarizeGroup` + `summarizeGroupsParallel` with a single `summarizeBatch` function. All eligible tool call groups are compressed in one LLM call, producing one structured operation-batch summary message.

---

## Consequences

### Positive

- Agent continuity across context compression boundaries is substantially improved for long-running, multi-step coding tasks.
- Failed approaches are recorded, preventing the agent from re-trying known dead ends.
- Decision rationale survives compression, preventing contradictory choices in later turns.
- Concrete technical artifacts (file paths, function names) are reliably preserved.
- Consistent section structure makes summaries predictable for the consuming agent.

### Negative

- The user prompt is slightly longer (~30 additional tokens per compression event). This is a negligible cost relative to the tokens saved by compression.
- The structured output depends on the summarizing LLM following formatting instructions; a weak model may produce inconsistent section headers.

### Neutral

- The change is backward-compatible: existing summarized messages in memory remain valid; only newly generated summaries use the new format.
- The system and user prompt strings are implementation details inside `generateSummary` and `buildSummarizationPrompt`; no public API changes.

---

## Validation

### Success Metrics

- Agent completes multi-step tasks that cross a context compression boundary without repeating already-completed steps.
- Agent does not re-attempt approaches that were explicitly abandoned within a compressed block.
- Post-compression context windows contain correct file paths and function names from pre-compression work.

### Monitoring

- Manual review of generated summaries in test sessions to verify section presence and artifact preservation.
- Regression: existing tests in `pkg/agent/context/` continue to pass.

---

## Related Decisions

- [ADR-0014](0014-composable-context-management.md) — Composable context management; defines the broader summarization pipeline this prompt operates within.
- [ADR-0015](0015-buffered-tool-call-summarization.md) — Buffered tool call summarization; a sibling strategy with its own summarization prompt (not changed here).
- [ADR-0018](0018-selective-tool-call-summarization.md) — Selective tool call summarization strategy.

---

## Notes

The six-section structure was chosen to match the information an agent actually needs to resume work rather than to follow any external summarization standard. The sections map directly to questions the agent would ask itself when picking up a task: "What did I do? What did I decide? What did I learn? What didn't work? Where am I now? What's next?"

**Last Updated:** 2025-01-31
