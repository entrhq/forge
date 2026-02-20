# 41. Goal-Batch Compaction Strategy for Long-Session Context Management

**Status:** Proposed
**Date:** 2025-02-01
**Deciders:** Development Team
**Technical Story:** Extend the context management system to handle sessions long enough that the existing summarization strategies create an implicit floor they cannot compress below — a growing accumulation of user messages and `[SUMMARIZED]` blocks that no current strategy touches.

---

## Context

### Background

ADR-0040 introduced structured summarization prompts for two existing strategies:

- `ToolCallSummarizationStrategy` — compresses old tool call pairs into `[SUMMARIZED]` blocks
- `ThresholdSummarizationStrategy` — compresses assistant message blocks into `[SUMMARIZED]` blocks when token usage crosses a threshold

Together these strategies handle the common case: a session of moderate length where assistant output and tool call noise are the primary sources of token pressure.

They do not handle the long-session case.

### Problem Statement

After enough turns, the conversation history takes this shape:

```
system
user₁
[SUMMARIZED]
user₂
[SUMMARIZED]
user₃
[SUMMARIZED]
...
userₙ  ← live goal
live assistant interaction
```

Both existing strategies have stopped firing because there is nothing left for them to act on:

- `ThresholdSummarizationStrategy` skips user messages (they are not assistant blocks) and skips `[SUMMARIZED]` messages (already summarized). With only those remaining, `collectMessagesToSummarize` returns nothing.
- `ToolCallSummarizationStrategy` finds no unsummarized tool call pairs in the old window.

The context is full and no strategy can reduce it further. This is the **implicit floor** — a lower bound on context size that the existing system cannot breach.

The floor is made of: all user messages (verbatim, uncompressed) + all `[SUMMARIZED]` blocks. Neither is currently compressible.

Additionally, the existing strategies treat each user message and each `[SUMMARIZED]` block as independent units. In practice, a goal is rarely completed in a single user turn. The user may redirect, clarify, or extend mid-arc:

```
user: "refactor the config loader"
[SUMMARIZED] (initial exploration)
user: "actually, support yaml too"
[SUMMARIZED] (yaml added)
user: "add validation"
[SUMMARIZED] (validation complete)
```

This is one goal arc. It should be compacted as a unit. The existing system has no concept of a goal arc — it sees three independent user messages and three independent summaries.

### Goals

- Eliminate the implicit floor by making old `user + [SUMMARIZED]` turn sequences compressible.
- Preserve the semantic content that has long-term value: the user's intent and direction, constraints introduced mid-arc, dead ends that may recur, key artifacts.
- Treat multi-turn goal arcs as the natural unit of compaction — not individual messages.
- Fire proactively on age and batch size, before the context goes critical, not in response to it.
- Produce a structured `[GOAL BATCH]` summary that the consuming agent can orient around as clearly as a `[SUMMARIZED]` block.

### Non-Goals

- Replacing the existing `ToolCallSummarizationStrategy` or `ThresholdSummarizationStrategy`. This is an additive third tier.
- Summarizing live or recent turns. The strategy must never touch content close to the current position.
- Modifying the first-person episodic framing established in ADR-0040.
- Supporting real-time or streaming summarization.

---

## Decision Drivers

* **Long-session continuity**: Agents working on large, multi-goal tasks need the context to remain coherent across the entire session, not just the most recent turns.
* **Goal arc integrity**: Splitting a multi-turn goal arc at an arbitrary message boundary loses the causal relationship between the user's direction changes and the agent's responses. The arc must be compacted whole.
* **Human direction preservation**: User messages contain intent, corrections, and constraints that shaped the work. They cannot be discarded — they must be distilled into the summary as first-class content.
* **Proactive pressure management**: Waiting until the context is critically full before compacting goal arcs means compacting under pressure. The strategy should fire well before the limit, keeping a buffer of headroom.
* **Turn-based atomicity**: The natural unit of compaction is a complete turn (user message + its adjacent `[SUMMARIZED]` block(s)). Splitting a turn mid-way loses coherence and may discard a user clarification that explains why the following summary looks the way it does.

---

## Considered Options

### Option 1: Reactive meta-summarization at the floor

**Description:** Extend `ThresholdSummarizationStrategy` with a second pass that fires when the first pass finds nothing to do and the context is still above threshold. The second pass folds `user + [SUMMARIZED]` pairs into higher-level summaries.

**Pros:**
- Minimal new code — extends an existing strategy.
- Fires only when needed (reactive to token pressure).

**Cons:**
- Fires at the worst possible moment — context is critically full, latency is already high.
- Does not handle the goal arc concept — still treats each user+summary pair as independent.
- No proactive management; sessions that approach the limit gradually will degrade silently until the floor is hit.

### Option 2: Relevance-weighted pruning of old user messages

**Description:** When context pressure is high, classify old user messages as "goal-setting" (high value, preserve) or "transactional" (low value, discard). Drop transactional messages silently.

**Pros:**
- No LLM call needed for the discard path.
- Simple heuristic (message length + content signals).

**Cons:**
- Heuristics for classifying user messages are fragile and culturally biased.
- Silent discard with no summary means lost information — the agent has no record that the turn existed.
- Does not compress `[SUMMARIZED]` blocks, so the floor is only partially addressed.
- No concept of goal arc.

### Option 3: Proactive two-tier goal lifecycle (Tier 1 → Tier 2 → Tier 3)

**Description:** A new strategy promotes old content through tiers: `[SUMMARIZED]` → `[GOAL: complete]` → `[GOAL: archived]`. Each promotion reduces resolution. Fires proactively on age.

**Pros:**
- Graceful degradation — content gets less detailed over time, never suddenly disappears.
- Handles deep history cleanly with single-sentence `[GOAL: archived]` records.

**Cons:**
- Three tiers means two separate LLM calls per goal arc (one per promotion).
- Tiers are goal-level (individual user turns), not arc-level — does not naturally handle multi-turn goal arcs.
- More implementation complexity: two promotion paths, two prompts, two metadata states.

### Option 4: Proactive goal-batch compaction with turn-based atomicity

**Description:** A new strategy — `GoalBatchCompactionStrategy` — that identifies complete turns (user + adjacent `[SUMMARIZED]` block(s)) older than a configurable message threshold, batches the oldest N turns, and compresses them into a single `[GOAL BATCH]` block in one LLM call. Fires proactively on turn count and age, not on token pressure.

**Pros:**
- Treats multi-turn goal arcs as the natural unit — a batch may span several user turns, preserving arc coherence.
- One LLM call per compaction run, regardless of how many turns are in the batch.
- Human direction and mid-arc corrections are preserved as first-class content in the `[GOAL BATCH]` prompt.
- Proactive firing means context pressure never builds silently to a crisis.
- Clean separation of concerns: existing strategies handle live operational detail; this strategy handles aged goal history.
- Turn-based atomicity (never split a turn) is a hard invariant enforced at the boundary detection level.

**Cons:**
- Requires a new strategy implementation.
- `[GOAL BATCH]` blocks are coarser than `[SUMMARIZED]` blocks — older content is always less detailed. Acceptable by design, but a trade-off.
- LLM call at compaction time adds latency to a turn (though this is true of all summarization strategies).

---

## Decision

**Chosen Option:** Option 4 — Proactive goal-batch compaction with turn-based atomicity.

### Rationale

The core insight that drives this choice is that the problem is architectural, not parametric. The existing strategies are not failing because their parameters are wrong — they are failing because they have no concept of a goal arc, and no mechanism for compressing content that is already summarized. Option 4 addresses both gaps directly.

Turn-based atomicity is the right granularity. A user message and its adjacent `[SUMMARIZED]` block(s) are causally linked — the summary is a record of the agent's response to that message. Splitting them is always a loss. Batching whole turns into a single compaction call preserves that causal link while still achieving significant compression.

Proactive firing is the right trigger model. Token-reactive compaction (Options 1 and 2) operates under pressure, at the moment when the system is least able to afford the latency of an LLM call. Age-and-count-based proactive firing keeps the context managed before it becomes critical.

The single-LLM-call batch design (compressing N turns in one call) mirrors the design principle established in ADR-0040's tool call batch summarization: the summarizer seeing the full sequence can infer connecting intent across turns, producing a summary that reflects arc-level strategy rather than turn-level operations.

---

## Implementation

### New File

`pkg/agent/context/goal_batch_strategy.go`

### Strategy Struct

```go
// GoalBatchCompactionStrategy compacts old completed-turn sequences (user message +
// adjacent [SUMMARIZED] block(s)) into single [GOAL BATCH] blocks. It fires
// proactively when enough old complete turns accumulate, rather than reactively
// when the context reaches a token limit.
//
// The atomic unit is a "turn": one user message plus all [SUMMARIZED] blocks that
// immediately follow it before the next user message. Turns are never split.
type GoalBatchCompactionStrategy struct {
    // minTurnsOldThreshold is how many messages back from the current position
    // a turn must be before it is eligible for compaction.
    minTurnsOldThreshold int

    // minTurnsToCompact is the minimum number of eligible complete turns required
    // to trigger compaction. Prevents firing on isolated old turns.
    minTurnsToCompact int

    // maxTurnsPerBatch is the maximum number of complete turns to compact in a
    // single LLM call. Bounds the size of the compaction prompt.
    maxTurnsPerBatch int

    eventChannel chan<- *types.AgentEvent
}
```

### ShouldRun Logic

1. Walk messages from oldest to newest, stopping `minTurnsOldThreshold` messages before the end.
2. Count complete turns: a complete turn is a user message followed by at least one `[SUMMARIZED]` block before the next user message or the eligibility boundary.
3. Return true if complete turn count >= `minTurnsToCompact`.

### Summarize Logic

1. Collect the oldest `maxTurnsPerBatch` complete turns from the eligible window.
2. Build the batch prompt (see below).
3. Make a single LLM call.
4. Replace all collected turn messages with a single `[GOAL BATCH]` message.
5. Mark with metadata: `"summarized": true`, `"summary_type": "goal_batch"`, `"turn_count": N`.

### Output Message Format

```
[GOAL BATCH]
## Goal Arc
[What was being worked on. How the goal evolved across turns if the user redirected.]

## Human Direction
[Key instructions, corrections, constraints the user introduced mid-arc. Preserves
the user's actual intent as it developed.]

## What Was Achieved
[Concrete outcomes: what is now in place, what passed, what was completed.]

## Dead Ends
[Approaches tried and abandoned across the arc. First-person lessons: 'I tried X — abandoned because Y.']

## Lasting Constraints
[Anything from this arc that still applies to future work.]

## Key Artifacts
[Exact file paths, function names, error strings, test names.]
```

Sections with nothing meaningful to say are omitted (same pattern as ADR-0040).

### Prompt Design

System prompt (same episodic framing as ADR-0040):
```
You are writing episodic memory for an AI coding agent. Your output will be injected
directly into the agent's context window as its own recalled experience. Write entirely
in operational first-person. Uncertainty markers are forbidden. Be dense, exact, and technical.
Preserve every concrete artifact. Omit XML markup, role labels, conversational filler,
and hedging language.
```

User prompt:
```
The following is a sequence of complete turns from your own recent past. Each turn is
a user instruction followed by your execution summary. Together they represent a goal arc —
the user's intent may have evolved across turns through redirections and clarifications.

Write a [GOAL BATCH] summary using the six-section structure below. The user's direction
changes are first-class content — preserve them in Human Direction. Dead ends from any
turn in the arc belong in Dead Ends. Constraints introduced at any point in the arc
belong in Lasting Constraints.

[six-section structure as above]

MUST PRESERVE: user intent and mid-arc redirections, dead ends, lasting constraints,
key artifacts (file paths, function names, error strings, test names).
MUST NOT INCLUDE: operational detail superseded within the arc, tool call mechanics,
intermediate states that were overwritten.
MUST USE: first-person throughout ('I tried', 'I found', 'I abandoned') — never 'the agent'.

Turns to compact:

--- Turn 1 ---
[user]: ...
[SUMMARIZED]: ...

--- Turn 2 ---
[user]: ...
[SUMMARIZED]: ...
...
```

### Strategy Registration Order

In the context `Manager`, strategies should be evaluated in this order:

1. `ToolCallSummarizationStrategy` — raw tool call pairs → `[SUMMARIZED]`
2. `ThresholdSummarizationStrategy` — assistant blocks → `[SUMMARIZED]` at token threshold
3. `GoalBatchCompactionStrategy` — old complete turns → `[GOAL BATCH]` proactively

Each strategy leaves the context in a state that benefits the next. By the time `GoalBatchCompactionStrategy` fires, all raw tool call noise and assistant blocks have already been cleaned up by strategies 1 and 2.

### Default Parameter Values

| Parameter | Default | Rationale |
|---|---|---|
| `minTurnsOldThreshold` | 20 messages | Keeps the most recent ~10 turns in full detail |
| `minTurnsToCompact` | 3 turns | Avoids firing on 1-2 isolated old turns |
| `maxTurnsPerBatch` | 6 turns | Bounds prompt size; large enough to capture a full goal arc |

---

## Consequences

### Positive

- The implicit floor on context size is eliminated. Very long sessions can now run indefinitely without hitting an uncompressible minimum.
- Multi-turn goal arcs are preserved as coherent units — the user's direction changes and the agent's responses are kept in their causal relationship.
- The agent's long-term memory contains structured goal history rather than a flat sequence of independent summaries.
- Dead ends and constraints from old goal arcs survive in the `[GOAL BATCH]` format, preventing re-investigation of resolved paths.
- Proactive firing means token pressure never silently builds to a crisis.

### Negative

- `[GOAL BATCH]` blocks are coarser than `[SUMMARIZED]` blocks. Operational detail from old turns is deliberately discarded. This is the intended trade-off but it is a trade-off.
- One additional LLM call per compaction run adds latency to the turn in which it fires. This is the same cost paid by the existing strategies.
- Very old user messages are absorbed into the `[GOAL BATCH]` and their verbatim content is lost. Their intent and direction survive, but the original wording does not.

### Neutral

- The `[GOAL BATCH]` tag is a new epistemic marker alongside `[SUMMARIZED]`. The agent must correctly interpret both as reconstructed memory. The existing `[SUMMARIZED]` tag convention is preserved for `[GOAL BATCH]` — same metadata (`"summarized": true`), different `"summary_type"` value.
- Existing sessions with accumulated `user + [SUMMARIZED]` history will be eligible for compaction on the first run after this strategy is added, if they meet the age and count thresholds.

---

## Validation

### Success Metrics

- Sessions running 50+ turns maintain coherent goal history without hitting an uncompressible context floor.
- `[GOAL BATCH]` blocks correctly capture user direction changes present in the original turn sequence.
- Dead ends from early turns survive into `[GOAL BATCH]` and prevent re-investigation in later turns.
- Token count after compaction is measurably lower than the pre-compaction floor that the existing strategies could not breach.

### Monitoring

- Manual review of `[GOAL BATCH]` output in long test sessions: verify section presence, arc coherence, and artifact preservation.
- Regression: existing tests in `pkg/agent/context/` continue to pass.
- Log the number of turns compacted and approximate tokens saved per `GoalBatchCompactionStrategy` run.

---

## Related Decisions

- [ADR-0014](0014-composable-context-management.md) — Composable context management; defines the Manager and Strategy interface this implementation extends.
- [ADR-0018](0018-selective-tool-call-summarization.md) — Selective tool call summarization; sibling strategy in the same pipeline.
- [ADR-0040](0040-structured-summarization-prompt.md) — Structured summarization prompt; establishes the episodic first-person framing and `[SUMMARIZED]` tag convention that `[GOAL BATCH]` extends.

---

## Notes

The turn-based atomicity invariant (never split a user message from its adjacent summaries) is the key structural decision. It means the batch boundary detection must scan for complete turns, not raw message counts. A turn is complete when a user message has at least one `[SUMMARIZED]` block following it before the next user message. An incomplete turn (user message with no following summary yet) is never eligible — it may still be in progress.

The `Human Direction` section of the `[GOAL BATCH]` prompt is the most important addition relative to the existing summarization prompts. The existing prompts treat the agent's actions as the primary content. This prompt treats the user's evolving direction as equally primary — because in a multi-turn goal arc, the shape of the work is determined as much by the user's corrections as by the agent's execution.

**Last Updated:** 2025-02-01
