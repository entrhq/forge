# Product Requirements: Long-Term Persistent Memory System

**Feature:** Autonomous Cross-Session Memory  
**Version:** 1.0  
**Status:** Draft  
**Owner:** Core Team  
**Last Updated:** January 2025

---

## Product Vision

Forge accumulates knowledge across sessions — project conventions, user preferences, architectural decisions, and recurring patterns — and recalls that knowledge automatically when relevant. The user never has to re-explain their preferences, re-litigate past decisions, or correct the same mistake twice.

Memory formation and recall are entirely autonomous: a subconscious process that runs alongside the agent without consuming turns, interrupting the user, or requiring deliberate management. Like a skilled engineer who naturally internalises the conventions of a codebase and the working style of their colleagues, Forge builds a persistent understanding that compounds over time.

**Strategic Alignment:** Transforms Forge from a capable session-scoped tool into a long-term engineering partner — one that gets meaningfully better the more it is used.

---

## Problem Statement

Every Forge session starts from zero. The agent has no knowledge of what it learned yesterday, last week, or during months of prior work on the same project. This creates a set of compounding frustrations:

1. **Repeated corrections** — the user tells the agent the same preference ("use `errors.As`, not type assertions") session after session, with no retention
2. **Forgotten conventions** — project-specific patterns, directory structures, and team decisions must be re-discovered or re-explained each session
3. **Lost rationale** — decisions made with careful reasoning in a past session are invisible in the next; the same trade-offs get relitigated
4. **Inconsistent behaviour** — without memory of corrections, the agent makes the same class of mistake repeatedly
5. **No compounding value** — the agent's effectiveness does not improve with use; a user who has worked with Forge for a year gets no more value from it than one who started today

The existing in-session scratchpad (ADR-0032) and conversation memory (ADR-0007) address within-session continuity but explicitly do not persist across sessions. This feature fills that gap.

---

## Key Value Propositions

### For Individual Developers
- **Preferences respected permanently** — coding style, communication style, and tooling preferences are remembered and applied from session one, not re-taught each time
- **Compounding expertise** — every correction, every stated preference, every architectural decision adds to a growing body of knowledge about how this user works
- **No repeated explanations** — the agent adapts to the user over time, not the other way around

### For Project-Focused Work
- **Project institutional knowledge** — conventions, architectural decisions, and rationale accumulated over the life of a project, available to every session
- **Onboarding acceleration** — a new session on a familiar project starts with full context of what has been established, not a blank slate
- **Decision continuity** — past decisions and their rationale are preserved, preventing contradictory changes and re-opened debates

### Competitive Advantage
- Unlike stateless AI coding tools, Forge builds a persistent model of the user and their projects — knowledge that is owned by the user, stored locally, and grows with use
- Transparent and auditable — memories are human-readable files the user can inspect, edit, or delete at any time

---

## Target Users & Use Cases

### Primary Personas

**The Daily Driver**
- Uses Forge as their primary coding assistant every day
- Has strong preferences about code style, error handling patterns, and tooling
- Finds it deeply frustrating to re-explain the same preferences repeatedly
- Values: retention of preferences, consistent behaviour, reduced friction

**The Project Owner**
- Works on the same codebase over months or years
- Has a rich mental model of the project's architecture, decisions, and conventions
- Wants the agent to internalise that mental model over time
- Values: project convention retention, decision history, architectural consistency

**The Team Lead**
- Works across multiple repositories with different conventions
- Wants per-repo memory (conventions, patterns) combined with personal preferences
- Values: clean separation between project-specific and personal knowledge, auditability

### Core Use Cases

**UC-1: Preference Retention**
User corrects the agent in session 3: "stop using `fmt.Println` in library code — use structured logging." In session 4, the agent applies structured logging without being told. The correction is never needed again.

**UC-2: Project Convention Accumulation**
Over five sessions working on the same repo, the agent learns: where migrations live, what the error-handling pattern is, which packages own which responsibilities. By session six, the agent navigates the codebase with the fluency of someone who has worked on it for months.

**UC-3: Architectural Decision Preservation**
In session 2, the team decides to use optimistic locking over pessimistic for user updates, with documented rationale. In session 8, when a new concurrency issue arises, the agent recalls the decision and applies it consistently — and can explain why.

**UC-4: Cross-Project Personal Style**
The user always prefers table-driven tests, functional options patterns, and short variable names. These preferences are captured as user-scoped memories and apply across every project Forge works on with this user.

**UC-5: Recurring Correction Elimination**
The agent made the same mistake (over-use of goroutines without proper lifecycle management) in sessions 1, 3, and 5 before the user corrected it each time. After the correction is captured as a memory, it is applied from session 6 onward — the pattern of the mistake and its correction are preserved.

---

## Product Requirements

### Must Have (P0)

**Memory Capture**
- Autonomous side-channel observer captures user↔assistant exchanges without consuming agent loop turns
- Capture is non-blocking — the main agent loop is never delayed by memory operations
- Two capture triggers: configurable turn-cadence (default every 5 turns) and goal-based compaction events (ADR-0041)
- Tool call content is explicitly excluded from memory analysis — only user and assistant messages are processed
- Classifier determines whether content is memory-worthy; not every exchange produces a memory
- Classifier LLM call is async and out-of-band; failures are non-fatal and silently retried
- Classifier assigns a human-readable `category` to each memory (see Data Model)

**Memory Storage**
- Two-tier persistent storage: `.forge/memory/` (repo-scoped) and `~/.forge/memory/` (user-scoped)
- Each memory is a standalone file identified by a UUID
- File format: YAML front-matter + markdown body (human-readable and human-editable)
- YAML front-matter schema (see Data Model section)
- Old memories are never deleted on update — a new file is written that `supersedes` the old one, forming a linear version chain
- Version chain is fully auditable by following `supersedes` links

**Memory Graph**
- Classifier draws explicit, typed relationship edges between memories: `supersedes`, `refines`, `contradicts`, `relates-to`
- Edges stored in YAML front-matter — visible and editable by the user
- Both the user (for auditing) and the agent (for reasoning) are first-class consumers of graph structure and version history

**Memory Retrieval**
- RAG engine runs autonomously at the start of each turn, before the agent responds; it is on the critical path and must complete before the agent begins composing its response
- Uses a two-stage retrieval strategy: a pre-RAG hypothesis generation hook followed by embedding-based semantic search (HyDE — Hypothetical Document Embeddings)
- **Conversation window**: all user and assistant messages since the last summarisation event; if no summarisation has occurred yet, the entire conversation history is used; tool call content is excluded in all cases
- **Stage 1 — Pre-RAG Hook**: the conversation window is sent to a lightweight LLM (configurable, defaults to a fast/flash-class model) which generates N hypothetical sentences that a relevant memory *might* contain — these hypotheses are semantically closer to stored memory files than raw conversation text, dramatically improving retrieval precision
- **Stage 2 — Embedding Search**: each hypothesis is independently embedded and used to query both memory stores; result sets from all hypotheses are unioned and deduplicated by memory UUID, retaining the highest similarity score per candidate across all hypothesis seeds
- From the deduplicated result set, traverse the configured number of graph hops to surface related neighbouring memories
- Retrieved memories and their full version history are injected into the system context before the agent responds
- Agent can reason over version chains (e.g. "this preference has changed 3 times — it may still be evolving")
- **Graceful degradation**: if the pre-RAG hook LLM call times out or fails, retrieval is skipped entirely for that turn; there is no keyword fallback; the session continues without injected memories and no error is surfaced to the user

**Configuration**
- `memory.enabled` — global on/off (default: true)
- `memory.cadence_turns` — turns between cadence-triggered capture passes (default: 5, range: 1–10)
- `memory.classifier_model` — LLM model for the capture classifier; intended for a higher-reasoning model for accurate memory formation, relationship detection, scope assignment, and category assignment (default: inherits summarization model from ADR-0042)
- `memory.retrieval_model` — LLM model for the pre-RAG hypothesis generation hook; must be a lightweight/flash-class model to minimise turn latency since this call is on the critical path (no default — if unset, retrieval is disabled even if embedding model is configured)
- `memory.embedding_model` — embedding model for semantic similarity search (same provider as configured LLM, no default — must be explicitly configured; retrieval is disabled until set)
- `memory.retrieval_top_k` — number of candidate memories to retrieve per hypothesis seed before deduplication (default: 10)
- `memory.retrieval_hop_depth` — how many graph hops to traverse from the deduplicated retrieved set when pulling graph neighbours (default: 1, range: 1–3)
- `memory.retrieval_hypothesis_count` — number of hypothetical sentences the pre-RAG hook generates per turn to use as retrieval seeds (default: 5, range: 1–10)

**User Visibility & Management**
- Memory files are directly readable and editable by the user in their filesystem
- YAML + markdown format allows manual editing without special tooling
- Version history is auditable by tracing `supersedes` links across files
- Users can delete memory files directly to remove specific memories permanently

### Should Have (P1)

- Injection token budget cap — configurable limit on how many tokens retrieved memories may consume in context (`memory.injection_token_budget`); when the budget is exceeded, memories are truncated in descending similarity score order so the most relevant are always included

### Could Have (P2)

- `/memory` slash command — list, search, and inspect memories from within the TUI
- Memory stats in context overlay — show count of active memories and last retrieval timestamp
- Per-repo memory enable/disable override — suppress memory for sensitive or temporary repos
- Memory export — dump all memories for a project or user to a single readable document

### Stretch (Future)

- **Memory Consolidation** — a background process that periodically identifies clusters of related memory fragments and merges them into a single richer memory, with graph edges back to the source nodes. Prevents memory store degrading into noise over months of use.

---

## Data Model

### Memory File Schema

```
---
id: mem_<uuid>
created_at: 2025-01-15T10:30:00Z
updated_at: 2025-01-15T14:22:00Z
version: 1
scope: repo | user
category: coding-preferences | project-conventions | architectural-decisions | user-facts | corrections | patterns
supersedes: null | mem_<uuid>
related:
  - id: mem_<uuid>
    relationship: refines | contradicts | relates-to
session_id: <session-uuid>
trigger: cadence | compaction
---

Memory content written as plain markdown. The classifier determines appropriate
granularity — from a short atomic fact to a richer narrative with rationale.

For architectural decisions, content should capture not just what was decided
but why, and what alternatives were considered.
```

### Category Definitions

| Category | Description | Typical Scope |
|---|---|---|
| `coding-preferences` | Style, idiom, and tooling choices | User |
| `project-conventions` | Repo-specific patterns and standards | Repo |
| `architectural-decisions` | Design decisions with rationale | Repo |
| `user-facts` | Facts about how the user works or thinks | User |
| `corrections` | Mistakes the agent made and was corrected on | Both |
| `patterns` | Non-obvious relationships or recurring structures | Both |

### Relationship Edge Types

| Relationship | Meaning |
|---|---|
| `supersedes` | This memory replaces the linked memory (version update) |
| `refines` | This memory adds nuance to the linked memory without replacing it |
| `contradicts` | This memory conflicts with the linked memory (for agent awareness) |
| `relates-to` | These memories are topically connected |

### Storage Layout

```
# Repo-scoped
.forge/memory/
  mem_abc123.md
  mem_def456.md
  ...

# User-scoped
~/.forge/memory/
  mem_xyz789.md
  mem_uvw012.md
  ...
```

---

## User Experience Flow

### Memory Capture Flow

```
[User Message]
      ↓
[Main Agent Loop — unaffected]
      ↓
[Assistant Response]
      ↓
[Turn Counter Increments]
      |
      ├── [If N turns reached OR compaction triggered]
      |         ↓
      |   [Observer: extract user+assistant exchanges]
      |         ↓
      |   [Async: send to Classifier LLM]
      |         ↓
      |   [Classifier: memory-worthy? → scope? → category? → relationships?]
      |         ↓
      |   [Classifier: query embedding store for existing related memories]
      |         ↓
      |   [Classifier: write memory file(s) with graph edges]
      |
      └── [Main loop continues — never blocked]
```

### Memory Retrieval Flow

```
[New Turn Begins]
      ↓
[RAG Engine: build conversation window]
  → All user+assistant messages since last summarisation
  → Fallback: full context if no summarisation has occurred yet
  → Tool call content excluded in all cases
      ↓
[Stage 1 — Pre-RAG Hook (on critical path)]
  → Send conversation window to retrieval_model (flash-class LLM)
  → LLM generates N hypothetical sentences a relevant memory might contain
  → If hook times out or fails → skip retrieval entirely, session continues unaffected
      ↓
[Stage 2 — Embedding Search]
  → Embed each hypothesis independently using embedding_model
  → Query .forge/memory/ + ~/.forge/memory/ for top-k per hypothesis
  → Union all result sets, deduplicate by memory UUID
  → Retain highest similarity score per memory across all hypothesis seeds
      ↓
[Graph Traversal]
  → From deduplicated result set, traverse up to retrieval_hop_depth hops
  → Collect neighbour memories via typed graph edges in YAML front-matter
      ↓
[Context Injection]
  → Inject retrieved memories + full version chains into system context
      ↓
[Agent responds with full memory context available]
```

### Version Chain Example

```
Session 3:
  mem_001.md  →  "User prefers pessimistic locking for concurrent writes"
                  supersedes: null

Session 7 (user changed their mind after a performance issue):
  mem_002.md  →  "User prefers optimistic locking with retry for concurrent writes"
                  supersedes: mem_001
                  related: [{id: mem_003, relationship: relates-to}]

Agent in session 8 retrieves mem_002, follows supersedes chain to mem_001,
and injects both — reasoning: "this preference changed once, due to a
performance issue; optimistic locking is current preference."
```

### Success States

- **Silent accumulation** — user notices the agent is getting better over time without explicitly teaching it; preferences are applied without prompting
- **Convention fluency** — agent navigates a familiar project as if it already knows the codebase; no re-explanation of patterns required
- **Decision consistency** — past architectural decisions are honoured across sessions; no contradictory changes introduced
- **Correction elimination** — a class of mistake the agent was corrected on never reappears after the correction is captured

### Error & Edge States

- **Retrieval model not configured** — retrieval is silently disabled for all turns; capture still runs; memories accumulate for when retrieval is later configured; user sees a one-time warning at session start
- **Embedding model not configured** — retrieval is silently disabled; capture still runs (memories accumulate for when embedding is later configured); user sees a one-time warning in config
- **Pre-RAG hook LLM failure or timeout** — retrieval is skipped entirely for that turn; no fallback is attempted; session continues normally without injected memories; no error is surfaced to the user
- **Classifier LLM failure** — capture pass is silently skipped for that trigger; no impact on session; retried on next trigger
- **Corrupt memory file** — file is skipped during retrieval; warning logged at debug level; no session impact
- **Memory injection token overflow** — retrieved memories are truncated to configured token budget (`memory.injection_token_budget`); memories are dropped in ascending similarity score order so the most relevant are always preserved
- **Empty memory store (cold start)** — retrieval returns nothing; session proceeds normally; capture begins accumulating from the first session onward

---

## User Interface & Interaction Design

### Phase 1 — Fully Autonomous (this release)

Memory is entirely invisible to the user in the normal flow. The agent simply gets better over time. No UI, no notifications, no management interface required. Users who want to inspect or edit memories do so directly via their filesystem.

### Phase 2 — Observability (P2 / future)

- Context overlay badge showing "N memories loaded" for the current turn
- `/memory list` — list recent memories from within the TUI, filtered by scope or category
- `/memory search <query>` — semantic search over memory store
- `/memory show <id>` — display a specific memory and its version chain

### Information Architecture

Memories are organised across three dimensions:
1. **Scope**: repo-scoped (project knowledge) vs. user-scoped (personal knowledge)
2. **Category**: what kind of thing was remembered (preference, convention, decision, correction, pattern)
3. **Recency**: when the memory was created or last updated

---

## Feature Metrics & Success Criteria

### Key Performance Indicators

- **Correction recurrence rate** — percentage of corrections that reappear in subsequent sessions (target: near zero after first capture)
- **Convention re-explanation rate** — how often users re-explain project conventions that should already be known (target: decreasing over first 10 sessions on a project)
- **Memory retrieval relevance** — qualitative user assessment of whether injected memories are useful vs. noisy (target: useful in majority of retrieval events)
- **Session startup quality** — subjective rating of how "up to speed" the agent feels at the start of a familiar-project session

### Success Thresholds

- **80% of stated user preferences** are applied correctly in the session immediately following their capture
- **Zero instances** of the agent contradicting a past architectural decision that has been captured as a memory
- **Capture latency** — classifier completes within 5 seconds of trigger; never delays user interaction
- **Retrieval latency** — end-to-end RAG pipeline (pre-RAG hook + embedding search + graph traversal + injection) completes within 2 seconds per turn; the pre-RAG hook LLM call is the dominant cost and must use a flash-class model to meet this budget

---

## User Enablement

### Discoverability

**For Users**: The feature is intentionally invisible in Phase 1. Users discover it through improved agent behaviour — noticing that preferences are remembered, conventions are applied, and corrections stick. No onboarding is required.

**Documentation**: How-to guide explaining the two memory tiers, file format, and how to inspect/edit memories directly.

### Configuration Guidance

Users who want to control memory behaviour configure it via the standard settings system:
- Enable/disable globally (`memory.enabled`)
- Set capture cadence — more frequent means finer-grained memories and more classifier calls (`memory.cadence_turns`)
- Choose the capture classifier model — use a capable reasoning model for high-quality memory formation (`memory.classifier_model`)
- Choose the retrieval model — must be a flash-class model; this call is on the critical path (`memory.retrieval_model`)
- Set the embedding model — both stored memories and retrieval hypotheses are embedded through this model (`memory.embedding_model`)
- Tune retrieval parameters: top-k candidates per hypothesis, graph hop depth, hypothesis count (`memory.retrieval_top_k`, `memory.retrieval_hop_depth`, `memory.retrieval_hypothesis_count`)

**Minimum configuration to activate retrieval:** `memory.retrieval_model` and `memory.embedding_model` must both be set. Capture runs regardless.

---

## Risk & Mitigation

### Quality Risks

**Risk**: Classifier captures low-signal information, filling memory store with noise  
**Mitigation**: Classifier prompt engineering emphasises memory-worthiness threshold; compaction trigger provides higher-signal second pass; users can delete noise directly

**Risk**: Retrieved memories consume too much context, crowding out actual conversation  
**Mitigation**: Configurable token budget cap for memory injection; top-k limit; graph traversal depth limit

**Risk**: Version chain grows unwieldy as memories are updated frequently  
**Mitigation**: Stretch-feature consolidation addresses long-term noise; classifier assesses whether to supersede or refine rather than always creating new nodes

**Risk**: Embeddings drift over time as conversation context changes, retrieving stale memories  
**Mitigation**: Retrieval uses fresh hypothesis generation and embedding each turn; HyDE hypotheses reflect the current conversational context, so stale memories are naturally outscored by more relevant ones

### Privacy & Security Risks

**Risk**: Sensitive information (credentials, personal data) captured and persisted  
**Mitigation**: Classifier prompt explicitly instructs against capturing sensitive literals; user owns and controls all memory files; no memory sync or upload

**Risk**: Repo-scoped memories contain information that should not persist (e.g. a temporary workspace)  
**Mitigation**: P2 per-repo memory disable override; users can delete `.forge/memory/` directory at any time

### Adoption Risks

**Risk**: Feature adds latency or instability, eroding trust  
**Mitigation**: Capture is strictly async and never blocks the main loop; retrieval is on the critical path but uses a flash-class model and has a hard 2-second budget with skip-on-failure semantics — the agent always responds, with or without injected memories

**Risk**: Users distrust memory they can't see  
**Mitigation**: Human-readable file format; direct filesystem access; Phase 2 observability commands; clear documentation of what is and isn't captured

---

## Dependencies & Integration Points

### Feature Dependencies

| Dependency | ADR | Role |
|---|---|---|
| Goal-Based Compaction | ADR-0041 | Triggers compaction memory pass |
| Summarization Model Override | ADR-0042 | Pattern for classifier model config |
| Composable Context Management | ADR-0014 | Injection point for retrieved memories |
| Agent Scratchpad Notes | ADR-0032 | Complementary in-session memory (not replaced) |
| Conversation Memory | ADR-0007 | Complementary in-session history (not replaced) |

### System Integration

- **Provider Abstraction Layer** (ADR-0003) — both embedding model calls and pre-RAG hook LLM calls must go through the provider abstraction
- **Settings System** (ADR-0017) — memory configuration added to existing settings schema; two new model config keys (`memory.retrieval_model`, `memory.embedding_model`) require explicit user configuration before retrieval activates
- **Agent Loop** — observer hooks into turn counting (capture, async); retrieval pipeline hooks into turn start and is on the critical path (must complete before agent responds); capture is fully async and never blocks the loop
- **Summarisation System** — the retrieval engine reads the last summarisation boundary to determine the conversation window; it must subscribe to summarisation events to maintain an accurate window pointer

### External Dependencies

- **Embedding model** — requires a configured embedding model from the user's LLM provider (e.g. `text-embedding-3-small` for OpenAI users); used to embed both stored memories at write time and generated hypotheses at retrieval time
- **Retrieval model (flash LLM)** — requires a configured fast/lightweight LLM from the user's provider for the pre-RAG hypothesis generation hook (e.g. `gemini-2.0-flash`, `gpt-4o-mini`); this is a separate model from the classifier and must be explicitly set in `memory.retrieval_model`

Both are assumed to be available from the user's already-configured LLM provider. No new provider integrations are required.

---

## Constraints & Trade-offs

### Design Decisions

**Decision**: Capture is turn-cadence + compaction triggered, not streaming every message  
**Rationale**: True per-message streaming would be expensive; batching provides better classifier context (multiple exchanges) and more signal for the compaction pass  
**Trade-off**: A preference stated and then immediately contradicted within the same batch may create an ambiguous memory; classifier must handle this case

**Decision**: Memory files are never deleted on update — version chain via `supersedes`  
**Rationale**: Auditability and agent reasoning over history are first-class requirements; deletion would break both  
**Trade-off**: Memory store grows over time; mitigated by future consolidation feature

**Decision**: Explicit graph edges in YAML front-matter (not implicit embedding similarity)  
**Rationale**: User auditability requires visible, traceable relationships; agent reasoning over history requires structured data; implicit graphs are opaque  
**Trade-off**: Classifier must do more reasoning work to assign edges; edges may occasionally be incorrect (user can correct by editing files)

**Decision**: Embedding model assumed from same provider as LLM  
**Rationale**: Avoids introducing a second provider dependency; most providers offer both LLM and embedding models  
**Trade-off**: Users on providers without embedding support cannot use retrieval; capture still works and memories accumulate for future use

**Decision**: Memory capture excludes tool call content  
**Rationale**: Tool calls are operational noise; the signal worth remembering lives in the human-language exchanges that describe intent, decisions, and preferences  
**Trade-off**: Some tool call outcomes (e.g. a bug discovered via a specific tool) may be worth remembering; the compaction pass partially addresses this by operating on the full goal arc summary

**Decision**: HyDE (Hypothetical Document Embeddings) as the retrieval query strategy, not direct conversation embedding  
**Rationale**: There is a fundamental semantic asymmetry between raw conversation text (verbose, noisy, dialogue-style) and stored memory files (terse, declarative, distilled). Directly embedding a conversation window produces low-precision retrieval because the query and the document live in very different semantic registers. Generating N hypothetical sentences that a relevant memory *might* contain bridges this gap — the hypothesis and the stored memory are both written in the same compressed, declarative style and are far more semantically proximate in embedding space. This improves retrieval precision without adding a new architectural component; it reuses the existing LLM provider abstraction.  
**Trade-off**: The pre-RAG hook is an LLM call on the critical path, introducing latency that direct conversation embedding would not. Mitigated by: (a) mandating a flash-class model for `memory.retrieval_model`, (b) a 2-second retrieval budget with graceful skip-on-failure, (c) no degraded fallback — a complete skip is preferable to noisy low-precision results.

**Decision**: `memory.classifier_model` and `memory.retrieval_model` are separate config keys targeting different model classes  
**Rationale**: The capture classifier does heavy reasoning — assessing memory-worthiness, assigning categories, detecting relationships, drawing graph edges — and benefits from a capable model. The retrieval hook generates simple hypothetical sentences and runs on the critical path; it must be fast and cheap. A single shared model config forces an irreconcilable trade-off between capture quality and retrieval latency.  
**Trade-off**: Two model config keys to manage; mitigated by sensible defaults (classifier inherits from summarisation model config, retrieval should be set explicitly to provider's fastest model)

