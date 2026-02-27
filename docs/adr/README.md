# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records (ADRs) for the Forge project.

## What is an ADR?

An Architecture Decision Record (ADR) is a document that captures an important architectural decision made along with its context and consequences.

## Why ADRs?

ADRs help us:
- **Document the "why"** behind important decisions
- **Provide context** for future contributors and maintainers
- **Track evolution** of the architecture over time
- **Prevent revisiting** decisions without understanding past context
- **Enable knowledge sharing** across the team

## ADR Format

We follow the format proposed by Michael Nygard with these sections:

1. **Title**: Short descriptive name
2. **Status**: Proposed, Accepted, Deprecated, Superseded
3. **Context**: What is the issue we're facing?
4. **Decision**: What is the change we're proposing/have made?
5. **Consequences**: What becomes easier or harder as a result?

## Naming Convention

ADRs are numbered sequentially and named with the pattern:

```
NNNN-title-with-dashes.md
```

For example:
- `0001-use-xml-for-tool-calls.md`
- `0002-implement-streaming-responses.md`

## Creating a New ADR

1. Copy the template: `cp template.md NNNN-your-decision.md`
2. Increment the number from the last ADR
3. Fill in all sections
4. Submit a PR for review
5. Update status after decision is made

## ADR Lifecycle

```
Proposed → Accepted → [Deprecated or Superseded]
```

- **Proposed**: Under discussion
- **Accepted**: Decision has been made and implemented
- **Deprecated**: No longer recommended but not replaced
- **Superseded**: Replaced by another ADR (link to new ADR)

## Index

<!-- Keep this list updated when adding new ADRs -->

| Number | Title | Status |
|--------|-------|--------|
| [0001](0001-record-architecture-decisions.md) | Record Architecture Decisions | Accepted |
| [0002](0002-xml-format-for-tool-calls.md) | Use XML Format for Tool Calls | Accepted |
| [0003](0003-provider-abstraction-layer.md) | Provider Abstraction Layer | Accepted |
| [0004](0004-agent-content-processing.md) | Agent-Level Content Processing | Accepted |
| [0005](0005-channel-based-agent-communication.md) | Channel-Based Agent Communication | Accepted |
| [0006](0006-self-healing-error-recovery.md) | Self-Healing Error Recovery with Circuit Breaker | Accepted |
| [0007](0007-memory-system-design.md) | Memory System Design | Accepted |
| [0008](0008-agent-controlled-loop-termination.md) | Agent-Controlled Loop Termination | Accepted |
| [0009](0009-tui-executor-design.md) | TUI Executor Design | Accepted |
| [0010](0010-tool-approval-mechanism.md) | Tool Approval Mechanism | Accepted |
| [0011](0011-coding-tools-architecture.md) | Coding Tools Architecture | Accepted |
| [0012](0012-enhanced-tui-executor.md) | Enhanced TUI Executor | Implemented |
| [0013](0013-streaming-command-execution.md) | Streaming Command Execution with Interactive Overlay | Accepted |
| [0014](0014-composable-context-management.md) | Composable Context Management with Strategy Pattern | Proposed |
| [0015](0015-buffered-tool-call-summarization.md) | Buffered Tool Call Summarization with Parallel Processing | Accepted |
| [0016](0016-file-ignore-system.md) | File Ignore System for Coding Tools | Proposed |
| [0017](0017-auto-approval-and-settings-system.md) | Auto-Approval and Interactive Settings System | Accepted |
| [0018](0018-selective-tool-call-summarization.md) | Selective Tool Call Summarization | Accepted |
| [0019](0019-xml-cdata-tool-call-format.md) | XML CDATA Tool Call Format | Accepted |
| [0020](0020-context-information-overlay.md) | Context Information Overlay | Accepted |
| [0021](0021-early-tool-call-detection.md) | Early Tool Call Detection | Accepted |
| [0022](0022-intelligent-tool-result-display.md) | Intelligent Tool Result Display | Accepted |
| [0023](0023-bash-mode-architecture.md) | Bash Mode Architecture | Accepted |
| [0024](0024-xml-escaping-primary-with-cdata-fallback.md) | XML Escaping Primary with CDATA Fallback | Accepted |
| [0025](0025-tui-package-reorganization.md) | TUI Package Reorganization | Implemented |
| [0026](0026-headless-mode-architecture.md) | Headless Mode Architecture | Accepted |
| [0027](0027-safety-constraint-system.md) | Safety Constraint System | Accepted |
| [0028](0028-quality-gate-architecture.md) | Quality Gate Architecture | Accepted |
| [0029](0029-headless-git-integration.md) | Headless Git Integration | Accepted |
| [0030](0030-automated-pr-documentation.md) | Automated PR Documentation | Accepted |
| [0031](0031-headless-git-pr-creation.md) | Headless Git PR Creation | Accepted |
| [0032](0032-agent-scratchpad-notes-system.md) | Agent Scratchpad Notes System | Accepted |
| [0033](0033-notes-viewer-tui-command.md) | Notes Viewer TUI Command | Implemented |
| [0034](0034-live-reloadable-llm-settings.md) | Live-Reloadable LLM Settings | Accepted |
| [0035](0035-auto-close-command-overlay.md) | Auto-Close Command Overlay | Accepted |
| [0036](0036-agents-md-repository-context.md) | AGENTS.md Repository Context | Accepted |
| [0037](0037-custom-tools-system.md) | Custom Tools System | Accepted |
| [0038](0038-browser-automation-architecture.md) | Browser Automation Architecture | Accepted |
| [0039](0039-llm-powered-page-analysis.md) | LLM-Powered Page Analysis | Accepted |
| [0040](0040-structured-summarization-prompt.md) | Structured Summarization Prompt | Accepted |
| [0041](0041-goal-batch-compaction-strategy.md) | Goal Batch Compaction Strategy | Accepted |
| [0042](0042-summarization-model-override.md) | Summarization Model Override | Accepted |
| [0043](0043-context-snapshot-export.md) | Context Snapshot Export | Accepted |
| [0044](0044-long-term-memory-storage.md) | Long-Term Memory Storage | Accepted |
| [0045](0045-long-term-memory-embedding-provider.md) | Long-Term Memory Embedding Provider | Accepted |
| [0046](0046-long-term-memory-capture.md) | Long-Term Memory Capture | Accepted |
| [0047](0047-long-term-memory-retrieval.md) | Long-Term Memory Retrieval | Accepted |
| [0048](0048-tui-smart-scroll-lock.md) | TUI Smart Scroll-Lock | Implemented |
| [0049](0049-tui-bracketed-paste-support.md) | TUI Bracketed Paste Support | Implemented |
| [0050](0050-tui-clipboard-copy.md) | TUI Clipboard Copy | Implemented |
| [0051](0051-tui-visual-redesign.md) | TUI Visual Redesign | Implemented |

## Resources

- [Architecture Decision Records](https://adr.github.io/)
- [Documenting Architecture Decisions by Michael Nygard](http://thinkrelevance.com/blog/2011/11/15/documenting-architecture-decisions)