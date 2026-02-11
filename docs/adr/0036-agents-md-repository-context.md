# 36. AGENTS.md Repository Context Integration

**Status:** Accepted
**Date:** 2024-01-15
**Deciders:** Forge Team
**Technical Story:** Support for AGENTS.md standard to provide repository-specific context to the AI agent

---

## Context

AGENTS.md is an emerging open standard adopted by 60,000+ open-source projects for providing context and instructions to AI coding agents. It's supported by major AI coding tools including Cursor, GitHub Copilot, Devin, and Windsurf. Projects use AGENTS.md to document setup commands, testing procedures, code style conventions, and project-specific workflows in a single, standardized location.

### Background

The AGENTS.md standard allows projects to maintain a single source of truth for how AI agents should interact with their codebase. This file typically contains:
- Project setup and build commands
- Testing procedures and conventions
- Code style guidelines
- PR/commit conventions
- Architecture notes and design decisions

Currently, Forge users must manually provide this context through conversation or custom instructions, leading to:
- Repeated context provision across sessions
- Inconsistent agent behavior across different AI tools
- Missed project-specific conventions
- Friction when switching between tools

### Problem Statement

Forge needs a way to automatically incorporate repository-specific guidance without:
- Overriding Forge's core behavioral identity
- Confusing what is agent behavior vs. project context
- Breaking existing workflows or prompts
- Requiring manual configuration from users

### Goals

- Automatically detect and load AGENTS.md from workspace root
- Provide repository context to the agent without modifying Forge's identity
- Maintain backward compatibility (works with or without AGENTS.md)
- Support the AGENTS.md standard exactly as other tools do
- Enable zero-configuration experience for users

### Non-Goals

- Semantic understanding or validation of AGENTS.md content
- Merging multiple AGENTS.md files (nested support is future work)
- Custom Forge-specific extensions to the standard
- Automatic generation of AGENTS.md (future enhancement)

---

## Decision Drivers

* **Ecosystem Compatibility**: Join the growing ecosystem of tools supporting AGENTS.md
* **User Experience**: Zero-configuration automatic loading
* **Clear Separation**: Repository context must not be confused with agent behavior
* **Backward Compatibility**: Must work seamlessly with existing Forge setups
* **Prompt Architecture**: Must fit cleanly into existing prompt composition system
* **Maintainability**: Keep implementation simple and focused

---

## Considered Options

### Option 1: Append to Custom Instructions

**Description:** Read AGENTS.md and append its content to the existing custom instructions section

**Pros:**
- Simple implementation
- Reuses existing WithCustomInstructions API
- No new prompt sections needed

**Cons:**
- Confuses agent behavior with project context
- No clear separation in the system prompt
- Can't distinguish what's Forge identity vs. repo guidance
- Harder to debug context issues

### Option 2: Separate Repository Context Section

**Description:** Add a new `<repository_context>` section in the system prompt, distinct from custom instructions

**Pros:**
- Clear separation of concerns
- Easy to identify what comes from AGENTS.md
- Maintains Forge's identity integrity
- Better for debugging and token tracking
- Aligns with mental model (repo info vs. agent behavior)

**Cons:**
- Requires new PromptBuilder method
- Slightly more complex prompt structure

### Option 3: Override Entire System Prompt

**Description:** Allow AGENTS.md to completely replace Forge's system prompt

**Pros:**
- Maximum flexibility
- Simple conceptually

**Cons:**
- Loses Forge's identity and capabilities
- Breaks agent loop mechanics
- Inconsistent behavior across tools
- Not what users expect from AGENTS.md

---

## Decision

**Chosen Option:** Option 2 - Separate Repository Context Section

### Rationale

AGENTS.md is fundamentally about repository-specific information (how to build, test, style code in THIS repo), not about how the agent should behave. Mixing it with custom instructions creates confusion about what's controlling the agent's behavior vs. what's providing project context.

The separate `<repository_context>` section:
1. Makes it explicit to the LLM that this is contextual information
2. Preserves Forge's behavioral identity completely unchanged
3. Allows easy debugging (can see exactly what came from AGENTS.md)
4. Enables future enhancements (token tracking per source)
5. Aligns with user mental model (project info, not agent config)

This approach maintains the integrity of Forge's agent loop, tool usage patterns, and core capabilities while seamlessly incorporating project-specific guidance.

---

## Consequences

### Positive

- Clear separation between agent behavior and repository context
- Forge's identity and capabilities remain unchanged
- Users can see exactly what context came from AGENTS.md in `/context` command
- Token budget tracking shows AGENTS.md contribution separately
- Future-proof for nested AGENTS.md support
- Drop-in compatibility with 60k+ existing AGENTS.md files

### Negative

- Slightly more complex prompt builder implementation
- New concept to document (repository context vs. custom instructions)
- One more section in the system prompt

### Neutral

- System prompt grows when AGENTS.md is present (expected and acceptable)
- Need to educate users on what AGENTS.md is vs. isn't

---

## Implementation

### Architecture

**Prompt Builder Enhancement:**
```go
type PromptBuilder struct {
    tools              []tools.Tool
    customInstructions string
    repositoryContext  string  // NEW
}

func (pb *PromptBuilder) WithRepositoryContext(context string) *PromptBuilder {
    pb.repositoryContext = context
    return pb
}
```

**Build Method Changes:**
```go
func (pb *PromptBuilder) Build() string {
    var builder strings.Builder
    
    // 1. Custom instructions (Forge identity)
    if pb.customInstructions != "" {
        builder.WriteString("<custom_instructions>\n")
        builder.WriteString(pb.customInstructions)
        builder.WriteString("\n</custom_instructions>\n\n")
    }
    
    // 2. Repository context (AGENTS.md) - NEW
    if pb.repositoryContext != "" {
        builder.WriteString("<repository_context>\n")
        builder.WriteString(pb.repositoryContext)
        builder.WriteString("\n</repository_context>\n\n")
    }
    
    // 3. System capabilities, agent loop, etc.
    builder.WriteString(SystemCapabilitiesPrompt)
    // ... rest of prompt
}
```

**Main Entry Point:**
```go
// In cmd/forge/main.go
func main() {
    // ... existing setup ...
    
    // Check for AGENTS.md
    agentsMdPath := filepath.Join(config.WorkspaceDir, "AGENTS.md")
    var repositoryContext string
    if content, err := os.ReadFile(agentsMdPath); err == nil {
        repositoryContext = string(content)
        // Optional: log that we loaded it
    }
    
    // Build prompt
    promptBuilder := prompts.NewPromptBuilder().
        WithTools(toolsList).
        WithCustomInstructions(composeSystemPrompt())
    
    if repositoryContext != "" {
        promptBuilder = promptBuilder.WithRepositoryContext(repositoryContext)
    }
    
    systemPrompt := promptBuilder.Build()
    // ... rest of setup ...
}
```

**Context Command Integration:**
The `/context` command should display repository context token count separately:
```
Context Usage: 45,231 / 128,000 tokens (35%)

Sources:
- System Prompt (Forge Identity): 8,450 tokens
- Repository Context (AGENTS.md): 3,220 tokens
- Conversation History: 33,561 tokens
```

Implementation requires:
- PromptBuilder to track repository context token count
- Context manager to expose this metric
- TUI command to display it in breakdown

### Migration Path

No migration needed - this is purely additive:
1. Existing Forge setups without AGENTS.md work exactly as before
2. Projects with AGENTS.md automatically get enhanced context
3. Users can continue using `-prompt` flag to override everything if needed

### Timeline

**Phase 1 (MVP):**
- Add WithRepositoryContext to PromptBuilder
- Auto-detect AGENTS.md in workspace root
- Add to system prompt as separate section
- Basic error handling (file not found is OK)
- Track repository context token count separately
- Display in `/context` command output

**Phase 2 (Enhancement):**
- Warn if AGENTS.md is very large (>10KB)
- Add `/agents` slash commands (create, edit, reload)

**Phase 3 (Advanced):**
- Support nested AGENTS.md in monorepos
- Smart context trimming for large files

---

## Validation

### Success Metrics

- 95%+ of existing AGENTS.md files load without errors
- Zero regression in sessions without AGENTS.md
- Token overhead <5KB for typical AGENTS.md files
- No reported confusion about agent behavior vs. repository context

### Monitoring

- Track AGENTS.md detection rate (% of sessions with file present)
- Measure AGENTS.md file sizes (median, p95, p99)
- Monitor for AGENTS.md-related errors in logs
- User feedback on whether agent follows project conventions better

---

## Related Decisions

- [ADR-0014](0014-composable-context-management.md) - Composable Context Management (provides foundation for context sections)
- [ADR-0032](0032-agent-scratchpad-notes-system.md) - Agent Scratchpad Notes System (another form of context)

---

## References

- [AGENTS.md Standard](https://github.com/Josh-XT/AGENTS) - Original specification
- [PRD: AGENTS.md Support](../product/features/agents-md-support.md) - Product requirements
- [Cursor AGENTS.md Docs](https://cursor.sh/agents) - How other tools implement it

---

## Notes

**Key Design Principle:** AGENTS.md provides repository-specific information (what), not agent behavior (how). Forge's identity controls the agent's approach to tasks, while AGENTS.md informs it about project conventions.

**Example System Prompt Structure:**
```
<custom_instructions>
  You are Forge, an elite coding assistant...
  [Core principles, workflow guidance, etc.]
</custom_instructions>

<repository_context>
  Setup: npm install
  Test: npm test
  Style: ESLint + Prettier
  [Project-specific info from AGENTS.md]
</repository_context>

<system_capabilities>
  [Agent capabilities and tool descriptions]
</system_capabilities>
```

**Last Updated:** 2024-01-15
