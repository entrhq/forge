# Long-Term Persistent Memory System

**Status:** Draft / Ideas Stage  
**Category:** Agent Core Capability  
**Priority:** High Impact, Near-Term  
**Last Updated:** January 2025

---

## 🎯 Overview

Give Forge a persistent, autonomous long-term memory that accumulates knowledge across sessions — project conventions, user preferences, architectural decisions, and recurring corrections — and recalls it intelligently when relevant, without the agent needing to spend turns thinking about memory management.

This is the complement to the existing in-session scratchpad (ADR-0032) and conversation memory (ADR-0007). Where those systems manage *within-session* context, this feature builds *cross-session* knowledge.

---

## 🔥 Problem Statement

Every Forge session starts from zero. The agent has no knowledge of:
- Preferences the user has expressed in the past ("always use `errors.As`", "I prefer flat package structure")
- Decisions made in previous sessions ("we chose X over Y because Z")
- Project conventions discovered through prior work ("migrations live in `/db/migrations`", "this repo uses the factory pattern for tools")
- Corrections the agent had to be given ("stop using `fmt.Println` in library code — you've been told this before")

This leads to:
- Repeated corrections across sessions
- Inconsistent behavior as the agent "forgets" established patterns
- Lost rationale for past decisions, leading to the same discussions being relitigated
- No accumulation of project-specific or user-specific expertise over time

The agent gets smarter within a session but resets to zero at the start of the next one.

---

## 💡 Core Concept

### The "Subconscious" Model

Memory formation in humans is largely subconscious — we don't stop mid-task to deliberately decide what to remember. Forge's long-term memory should work the same way:

- **Capture is autonomous** — a side-channel observer watches the conversation and classifies memories independently of the main agent loop
- **No turns consumed** — the memory process never takes a turn away from the user's task
- **Recall is autonomous** — the RAG engine injects relevant memories into context at the start of each turn, without the agent needing to "ask" for them

### Two Memory Tiers

**Repo-scoped** (`.forge/memory/`) — specific to the current project:
- Architectural decisions and rationale
- Project conventions and patterns
- Codebase-specific gotchas and workarounds
- Team preferences for this repo

**User-scoped** (`~/.forge/memory/`) — follows the user across all projects:
- Coding style preferences
- Communication preferences
- General programming philosophy
- Cross-project corrections and learning

---

## 🧠 How It Works

### Capture: The Observer

A lightweight observer watches the conversation stream — specifically user↔assistant exchanges (not tool calls). It fires on two triggers:

1. **Cadence trigger** — every N turns (user-configurable, default 5), the observer sends a batch of recent exchanges to the memory classifier
2. **Compaction trigger** — when goal-based compaction fires (ADR-0041), the observer gets a second, richer pass over a complete goal arc — what was attempted, what worked, what decisions were made

The classifier is a separate, non-blocking LLM call (configurable model) that runs async and never impacts the user's session.

### The Classifier

The LLM classifier is the intelligence layer. For each batch it:
- Determines whether the content contains memory-worthy information
- Assigns scope (repo vs. user) and a category
- Queries existing memories via embedding similarity to detect relationships
- Decides how the new memory relates to existing ones (new, supersedes, refines, contradicts, relates-to)
- Writes the memory file with appropriate front-matter and relationships

### Storage: Graph-Versioned Memory Files

Each memory is a standalone file in `YAML front-matter + markdown body` format. The YAML carries structured metadata including explicit graph edges to related memories. Old memories are never deleted — when something changes, a new memory is written that `supersedes` the old one, creating a linear version chain.

This means:
- Users can audit the full history of what the agent has learned
- The agent can reason over version history ("this preference has changed 3 times — it may still be in flux")
- Related memories are connected by explicit, typed edges

### Retrieval: The RAG Engine

At the start of each turn, the RAG engine:
1. Embeds the current conversation context
2. Queries both memory stores for semantically similar memories
3. Traverses graph edges to pull in related/neighboring memories
4. Injects the relevant set into the system context

This is fully autonomous — the agent doesn't need to ask for memories; they arrive as natural context.

---

## 📊 Memory File Format

```
---
id: mem_<uuid>
created_at: 2025-01-15T10:30:00Z
updated_at: 2025-01-15T10:30:00Z
version: 1
scope: repo | user
category: coding-preferences | project-conventions | architectural-decisions | user-facts | corrections
supersedes: null
related:
  - id: mem_<uuid>
    relationship: refines | contradicts | relates-to
session_id: <session-uuid>
trigger: cadence | compaction
---

Memory content in plain markdown. Can be a short atomic fact or a richer narrative
depending on what the classifier determined was appropriate.
```

---

## ⚙️ Configuration

```yaml
memory:
  enabled: true
  cadence_turns: 5          # How many turns between capture passes (1-10)
  classifier_model: ""      # Model for classification (defaults to summarization model)
  embedding_model: ""       # Model for embeddings (same provider as LLM)
  retrieval_top_k: 10       # How many memories to retrieve per turn
  retrieval_hop_depth: 1    # How many graph hops to traverse from retrieved memories
```

---

## 🔗 Relationship to Existing Memory Systems

| System | Scope | Trigger | Persistence |
|--------|-------|---------|-------------|
| ConversationMemory (ADR-0007) | In-session messages | Every LLM call | Session only |
| Scratchpad Notes (ADR-0032) | In-session insights | Agent-initiated | Session only |
| **Long-Term Memory (this)** | Cross-session knowledge | Autonomous observer | Persistent |

These systems are complementary, not competing. Long-term memory is the persistence layer that lets insights from one session inform the next.

---

## 🚀 Stretch Feature

**Memory Consolidation** — a background process that periodically identifies clusters of small, related memories and merges them into a single richer memory, with graph edges back to the source fragments. Keeps the memory store from growing noisy over months of use.

---

## 🚧 Open Questions

1. **Embedding model dependency** — do we always assume same provider? What if no embedding model is available?
2. **Memory visibility UX** — beyond raw file editing, should there be a `/memory` slash command for inspection?
3. **Privacy boundaries** — should repo memories ever reference user-specific information and vice versa?
4. **Cold start** — how do we handle the first few sessions before there are any memories to retrieve?
5. **Memory cap** — do we impose a size/count limit per store, or let it grow indefinitely?
6. **Injection size** — how many tokens can retrieved memories consume before they crowd out actual conversation context?

---

## 🔗 Related Features

- **Conversation Memory** (ADR-0007) — in-session message pruning
- **Agent Scratchpad Notes** (ADR-0032) — in-session working memory
- **Goal-Based Compaction** (ADR-0041) — triggers the compaction memory pass
- **Summarization Model Override** (ADR-0042) — pattern for per-task model configuration
- **Context Management** (ADR-0014) — composable context system memories inject into

---

## Next Steps

1. **Write PRD** — full requirements, personas, success metrics
2. **Write ADR** — technical design, implementation plan, package structure
3. **Spike: embedding integration** — validate embedding model dependency with existing providers
4. **Spike: async classifier** — validate non-blocking goroutine pattern fits agent loop architecture
