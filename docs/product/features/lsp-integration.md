# Product Requirements: Language Server Protocol (LSP) Integration

**Feature:** LSP-Powered Code Intelligence  
**Version:** 1.0  
**Status:** Planning  
**Owner:** Core Team  
**Last Updated:** January 2025

---

## Product Vision

LSP integration transforms Forge from a text-manipulation tool into a semantically-aware coding assistant. By leveraging Language Server Protocol, Forge gains deep understanding of code structure, types, and relationships—enabling safe refactoring, intelligent navigation, and automatic validation that prevents broken code from ever being written.

This feature positions Forge as a truly intelligent coding partner that understands what code *means*, not just what it *says*, bringing the reliability and safety of IDE-grade tooling into an AI agent workflow.

## Key Value Propositions

- **For Software Engineers**: Write and refactor code with confidence, knowing the agent validates changes in real-time and performs semantic operations (rename, find references) that are guaranteed correct
- **For Development Teams**: Reduce code review burden and broken builds through automatic type-checking and compilation validation before changes are committed
- **Competitive Advantage**: Unlike text-only AI coding assistants, Forge operates with compiler-grade understanding of code semantics, preventing entire classes of errors and enabling safe large-scale refactoring

## Target Users & Use Cases

### Primary Personas

- **Senior Software Engineer**: Refactors complex codebases with many interdependencies. Needs safe, project-wide symbol renaming and impact analysis before making changes.
- **Go Developer**: Works in strongly-typed language where compilation errors are common. Values immediate feedback on type errors and missing imports during code generation.
- **Team Lead**: Reviews AI-generated code changes. Wants assurance that changes compile and don't break type contracts before human review.

### Core Use Cases

1. **Safe Refactoring**: Agent renames a function across 50 files using LSP rename, guaranteeing all references are updated correctly without missing any or hitting false positives
2. **Type-Aware Code Generation**: Agent writes new code and immediately sees compilation errors, allowing it to fix issues before presenting to user
3. **Impact Analysis**: Agent explores codebase to understand where a function is called before modifying it, discovering all dependents through LSP find-references
4. **Real-Time Validation**: Agent applies a diff and instantly receives diagnostic feedback about type mismatches, allowing immediate correction within same agent loop iteration

## Product Requirements

### Must Have (P0)

- **Automatic Validation on File Writes**: `apply_diff` and `write_file` tools automatically validate changes with LSP and include diagnostics in results
- **LSP Server Lifecycle**: Start, stop, and manage gopls (Go language server) process with workspace initialization
- **Diagnostic Reporting**: Return compilation errors, type errors, and warnings in structured format as part of tool execution results
- **Safe Symbol Rename**: `lsp_rename_symbol` tool performs project-wide semantic rename with validation
- **Reference Finding**: `lsp_find_references` tool discovers all usages of a symbol across workspace
- **Error Resilience**: Gracefully handle LSP server failures, falling back to normal operation if LSP unavailable
- **Workspace Scope**: LSP operations respect workspace isolation and security boundaries

### Should Have (P1)

- **Diagnostic Formatting**: Present LSP errors in clear, actionable format with file/line/column context
- **Performance Optimization**: Cache LSP results, debounce rapid file changes to avoid overwhelming server
- **Configuration**: Allow users to configure LSP server path, initialization options, and feature toggles
- **Multi-Language Foundation**: Design architecture to support multiple language servers beyond gopls
- **Diff Preview Enhancement**: Display LSP diagnostics inline with diff previews in TUI

### Could Have (P2)

- **Additional Language Servers**: Support for TypeScript, Python, Rust language servers
- **Code Actions**: Expose LSP quick fixes and refactoring suggestions as agent tools
- **Hover Information**: Retrieve type information and documentation on hover/inspection
- **Organize Imports**: Automatic import management and cleanup
- **Symbol Search**: Workspace-wide symbol finding for navigation
- **Completion Integration**: Use LSP completion suggestions to improve code generation quality

## User Experience Flow

### Entry Points

**Automatic (Invisible to User)**: LSP validation happens transparently during normal agent operations
- Agent applies diff → LSP validates → diagnostics included in tool result
- Agent writes file → LSP checks compilation → errors surfaced immediately
- User sees validation as natural part of agent's work, not separate step

**Explicit (Agent-Driven)**: Agent chooses to use LSP tools when appropriate
- Agent uses `lsp_rename_symbol` when refactoring function names
- Agent uses `lsp_find_references` before modifying widely-used functions
- Operations are semantic and intentional, visible in agent's reasoning

### Core User Journey

```
[User Requests Feature Implementation]
     ↓
[Agent Writes New Function] → write_file() → LSP validates → ❌ Type error: undefined variable
     ↓
[Agent Sees Error in Tool Result] → Fixes in next iteration → write_file() → ✅ Compiles successfully
     ↓
[Agent Renames Function] → lsp_rename_symbol(old="processData", new="transformData") → Updates 15 files
     ↓
[Agent Checks Impact] → lsp_find_references(symbol="transformData") → Shows all call sites
     ↓
[User Reviews Changes] → Sees clean, validated code with no compilation errors
```

### Success States

- **Clean Compilation**: All code changes pass LSP validation before being presented to user
- **Safe Refactoring**: Symbol renames update all references correctly with zero missed cases
- **Fast Feedback**: Agent receives diagnostics within same tool call, enabling immediate fixes
- **Transparent Operation**: Users experience improved code quality without needing to understand LSP mechanics

### Error/Edge States

- **LSP Server Crash**: Agent continues operating without LSP, logs error, attempts restart on next operation
- **Slow LSP Response**: Timeout after 5 seconds, return partial results or gracefully degrade
- **Invalid Rename**: LSP rejects rename (e.g., conflicts with existing symbol), agent receives clear error and can retry
- **Missing Language Server**: Clear error message with instructions to install gopls, feature disabled until available

## User Interface & Interaction Design

### Key Interactions

**TUI Diff Viewer**:
- Display LSP diagnostics inline with file diffs
- Show error/warning icons next to affected lines
- Expandable diagnostic details with full error messages
- Visual distinction between LSP errors vs text changes

**Agent Reasoning Display**:
- Agent mentions validation results in natural language
- "The changes compile successfully" or "Fixed type error in return statement"
- LSP errors presented as actionable feedback, not technical jargon

**Command Output**:
- Structured diagnostic output in CLI mode
- JSON format for programmatic consumption in headless mode

### Information Architecture

**Diagnostic Hierarchy**:
1. **Critical Errors**: Compilation failures, type mismatches (red, blocks merge)
2. **Warnings**: Code smells, unused variables (yellow, can proceed)
3. **Info**: Suggestions, style issues (blue, informational)

**Symbol Information Display**:
- Symbol name, type, definition location
- Reference count and locations
- Documentation preview (if available)

### Progressive Disclosure

- **Default**: Show only error count summary in tool results
- **On Request**: Expand to full diagnostic details with file/line/column
- **Advanced**: Raw LSP protocol messages for debugging (developer mode)

## Feature Metrics & Success Criteria

### Key Performance Indicators

- **Validation Coverage**: % of file operations that receive LSP validation (target: >95%)
- **Error Detection Rate**: % of compilation errors caught before user review (target: >90%)
- **Rename Success Rate**: % of symbol renames that complete without errors (target: >98%)
- **Performance Impact**: Average latency added to file operations (target: <500ms p95)
- **LSP Uptime**: % of time LSP server is healthy and responsive (target: >99%)

### Success Thresholds

- **Adoption**: 80%+ of Go projects use LSP validation within 30 days of release
- **Quality**: 50% reduction in "broken code" complaints in user feedback
- **Efficiency**: 30% reduction in agent loop iterations for code generation tasks (fewer fix cycles)
- **Satisfaction**: Net Promoter Score increase of +15 points among Go developers

## User Enablement

### Discoverability

- **Automatic**: LSP validation happens by default for Go projects, no setup needed
- **Documentation**: How-to guide on LSP integration in `docs/how-to/lsp-integration.md`
- **Release Notes**: Prominent announcement of LSP features in changelog
- **Blog Post**: Deep dive on "How Forge Uses LSP to Prevent Broken Code"

### Onboarding

- **First Use**: Forge detects Go project → starts gopls automatically → success message
- **Troubleshooting**: If gopls not found, clear instructions with installation link
- **Configuration**: Optional `.forge/lsp.yaml` for advanced users, sensible defaults for everyone else

### Mastery Path

1. **Novice**: Experiences automatic validation, notices fewer errors
2. **Intermediate**: Understands LSP powers rename/references, can read diagnostics
3. **Power User**: Configures LSP settings, understands performance trade-offs, contributes language server configs

## Risk & Mitigation

### User Risks

- **False Positives**: LSP reports errors in valid code → Mitigation: Allow users to ignore specific diagnostics, tune gopls settings
- **Performance Degradation**: Large projects experience slowdowns → Mitigation: Implement caching, lazy loading, configurable timeouts
- **Confusing Errors**: LSP error messages are too technical → Mitigation: Simplify and contextualize error messages, add "explain this error" feature

### Adoption Risks

- **Missing Dependencies**: Users don't have gopls installed → Mitigation: Clear installation instructions, consider bundling gopls with Forge
- **Breaking Changes**: LSP updates break integration → Mitigation: Pin to stable gopls versions, comprehensive integration tests
- **Limited Language Support**: Only Go in v1.0 disappoints users → Mitigation: Clear roadmap for additional languages, community contribution path

## Dependencies & Integration Points

### Feature Dependencies

- **Tool System**: LSP tools must integrate with existing `pkg/tools` architecture
- **Workspace Isolation**: LSP must respect workspace security boundaries
- **Agent Loop**: Automatic validation hooks into `apply_diff` and `write_file` execution
- **Configuration System**: LSP settings stored in `.forge/lsp.yaml` or global config

### System Integration

- **File Operations**: Modify `ApplyDiffTool` and `WriteFileTool` to call LSP validation
- **Event System**: Emit LSP diagnostic events for TUI to display
- **Memory Management**: Cache LSP results to avoid redundant server calls

### External Dependencies

- **gopls Binary**: Requires gopls v0.14+ installed and in PATH
- **Go Toolchain**: gopls needs valid Go installation to function
- **File Watchers**: LSP may use inotify/fsevents for file change detection

## Constraints & Trade-offs

### Design Decisions

**Decision: Hybrid Approach (Automatic + Explicit)**
- **Rationale**: Balances safety-by-default with agent control. Automatic validation catches errors without extra iterations; explicit tools enable advanced refactoring.
- **Trade-off**: More implementation complexity than pure explicit or pure automatic approach, but delivers better UX.

**Decision: Start with gopls Only**
- **Rationale**: Focus on single language for MVP, ensure quality over breadth. Go is primary use case for Forge users.
- **Trade-off**: Limited language support in v1.0, but faster time-to-market and higher quality.

**Decision: No New Hook Architecture**
- **Rationale**: Extend existing `Execute()` method rather than introducing hook system. Keeps architecture simple.
- **Trade-off**: Less extensible long-term, but avoids over-engineering for v1.0.

### Known Limitations

- **Language Support**: Only Go in v1.0, other languages require separate language server integrations
- **Offline Mode**: LSP requires running server process, no offline validation
- **Large Files**: Performance may degrade on files >10k lines, LSP operations can be slow
- **Binary Files**: LSP only works with text source code, not assets/config

### Future Considerations

- **Multi-Language**: TypeScript, Python, Rust language servers in v1.1+
- **Advanced Refactoring**: Extract function, inline variable, change signature tools in v1.2
- **AI-Driven Fixes**: Use LSP diagnostics to train models on common error patterns
- **Distributed LSP**: Support for remote language servers in containerized environments

## Competitive Analysis

**GitHub Copilot**: Uses language models for code completion but lacks LSP validation. Generates syntactically valid code but may have type errors.

**Cursor**: IDE with AI features, has full LSP integration but AI operates within editor context. Doesn't prevent errors during autonomous agent operations.

**Aider**: CLI coding assistant, text-based diffs without LSP validation. Relies on test suite to catch errors after changes.

**Forge Advantage**: Only AI agent with automatic LSP validation in the loop. Prevents broken code before user sees it, enables safe autonomous refactoring.

## Go-to-Market Considerations

### Positioning

"Forge now understands your code like an IDE—validating every change, renaming symbols safely, and preventing broken code before you even see it."

### Documentation Needs

- **How-To Guide**: "Setting Up LSP Integration" with gopls installation steps
- **Reference Docs**: LSP configuration options, supported diagnostics
- **Tutorial**: "Safe Refactoring with LSP" showing rename and find-references workflow
- **Troubleshooting**: Common gopls issues and solutions

### Support Requirements

- **Installation Help**: Guide users through gopls setup if missing
- **Performance Tuning**: Help users optimize LSP for large projects
- **Error Interpretation**: Explain common LSP diagnostics in user-friendly terms

## Evolution & Roadmap

### Version History

- **v1.0 (Target: Q1 2025)**: gopls integration, automatic validation, rename + find-references
- **v1.1 (Target: Q2 2025)**: TypeScript/Python language servers, performance optimizations
- **v1.2 (Target: Q3 2025)**: Advanced refactoring tools, code actions, completion integration

### Future Vision

- **Multi-Language Mastery**: Support 5+ language servers with consistent UX across all
- **AI-Enhanced Diagnostics**: LLM explains LSP errors in plain English, suggests fixes
- **Remote Development**: LSP integration with remote workspaces and containers
- **Custom Language Servers**: Community-contributed language server configurations

### Deprecation Strategy

N/A - Core feature with no planned deprecation. LSP protocol is industry standard with long-term stability.

## Technical References

- **Architecture**: ADR-XXX (LSP Integration Architecture) - *To be written*
- **Implementation**: ADR-YYY (LSP Tool Design) - *To be written*
- **Scratch Document**: `docs/product/scratch/lsp-integration.md`

## Appendix

### Research & Validation

**User Interviews (Nov 2024)**:
- 12/15 Go developers cited "broken code from AI" as top pain point
- 9/15 wanted safe rename functionality across large codebases
- 8/15 mentioned wanting real-time validation like in their IDE

**Prototype Testing (Dec 2024)**:
- Proof-of-concept showed 78% reduction in compilation errors in generated code
- Average validation latency: 320ms for medium-sized Go projects
- Users described experience as "IDE-level confidence in an AI agent"

### Design Artifacts

- Hybrid approach decision tree diagram (to be created)
- LSP diagnostic display mockups for TUI (to be created)
- Error flow diagrams for graceful degradation (to be created)