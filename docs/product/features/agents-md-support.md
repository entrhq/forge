# AGENTS.md Support

## Product Vision

AGENTS.md is an emerging open standard for providing context and instructions to AI coding agents, adopted by over 60,000 open-source projects and supported by major AI coding tools (Cursor, GitHub Copilot, Devin, Windsurf, etc.). By supporting this standard, Forge enables users to leverage a single, predictable file format to provide project-specific context across their entire toolchain.

**Strategic Purpose:**
- **Ecosystem Alignment**: Join the growing ecosystem of tools supporting the AGENTS.md standard
- **Reduce Onboarding Friction**: Allow projects to define context once for all AI tools
- **Enhance Project Context**: Automatically incorporate project-specific build steps, conventions, and workflows
- **Preserve Forge Identity**: AGENTS.md provides repository context, NOT behavioral overrides - Forge's agent loop and identity remain unchanged
- **Enable Monorepo Support**: Support nested AGENTS.md files for complex project structures

## Key Value Propositions

- **For Open Source Maintainers**: Define agent behavior once in AGENTS.md, works across Forge, Cursor, GitHub Copilot, and 60+ other tools
- **For Enterprise Teams**: Standardize coding conventions and workflows across AI tools without vendor lock-in
- **For Individual Developers**: Zero configuration - if project has AGENTS.md, Forge automatically picks it up
- **Competitive Advantage**: Forge becomes drop-in compatible with existing AGENTS.md infrastructure while maintaining its superior agent loop and TUI

## Target Users & Use Cases

### Primary Personas

**1. Multi-Tool Developer**
- **Role**: Developer using multiple AI coding assistants (Cursor, Copilot, Forge)
- **Goals**: Consistent agent behavior across all tools
- **Pain Points**: Having to configure each tool separately with same project conventions

**2. OSS Project Maintainer**
- **Role**: Maintainer of open-source project with existing AGENTS.md
- **Goals**: Make it easy for AI assistants to contribute correctly
- **Pain Points**: Contributors using AI tools that don't follow project conventions

**3. Monorepo Engineer**
- **Role**: Engineer working in large monorepo with multiple subprojects
- **Goals**: Different agent behavior per subproject (frontend vs backend vs infra)
- **Pain Points**: One-size-fits-all configuration doesn't work for diverse monorepos

### Core Use Cases

1. **Existing Project with AGENTS.md**: Developer runs `forge` in a project that already has AGENTS.md. Forge automatically reads it and incorporates project-specific context (test commands, code style, PR conventions) while maintaining Forge's core behavioral identity.

2. **Monorepo Nested Context**: Developer working in `packages/frontend` of a monorepo. Forge reads both root AGENTS.md (org-wide conventions) and `packages/frontend/AGENTS.md` (frontend-specific), with the closer file taking precedence for conflicts.

3. **Migration from Other Tools**: Developer switching from Cursor to Forge. Their project's existing AGENTS.md works immediately with no configuration needed.

4. **Creating New AGENTS.md**: Developer in project without AGENTS.md asks Forge to create one. Forge analyzes project structure and generates a starter AGENTS.md with detected conventions.

## Product Requirements

### Must Have (P0)

- **Auto-Detection**: Automatically detect and read AGENTS.md in workspace root on startup
- **Hybrid Prompt Composition**: Combine Forge's core identity with AGENTS.md project context
- **Graceful Fallback**: If no AGENTS.md exists, use default Forge prompt (backward compatible)
- **Markdown Parsing**: Read and parse AGENTS.md as plain markdown (no special processing)
- **CLI Override**: Support `-prompt` flag to override AGENTS.md (user preference wins)
- **Error Handling**: Handle missing, malformed, or inaccessible AGENTS.md gracefully

### Should Have (P1)

- **Nested File Support**: In monorepos, read closest AGENTS.md relative to current working file
- **Create Command**: `/agents create` slash command to generate starter AGENTS.md
- **Edit Command**: `/agents edit` to open AGENTS.md in user's editor
- **Reload Command**: `/agents reload` to re-read AGENTS.md without restarting
- **Validation**: Warn if AGENTS.md is very large (>10KB) as it consumes context budget

### Could Have (P2)

- **Smart Merging**: When both root and nested AGENTS.md exist, intelligently merge sections
- **Template Library**: Built-in templates for common project types (Go, TypeScript, Python, etc.)
- **Version Detection**: Detect if AGENTS.md references outdated commands/tools
- **Export Command**: Generate AGENTS.md from current Forge configuration
- **Diff View**: Show what changed when AGENTS.md is modified

## User Experience Flow

### Entry Points

**Primary**: Automatic on startup
- Forge checks for AGENTS.md in workspace root
- If found, reads and incorporates into system prompt
- No user action required

**Secondary**: Slash commands
- `/agents create` - Generate new AGENTS.md
- `/agents edit` - Edit existing AGENTS.md
- `/agents reload` - Reload after manual edits
- `/context` - View token breakdown including AGENTS.md contribution

### Core User Journey

```
User runs `forge` in project directory
     ↓
[AGENTS.md exists?]
     ↓
  Yes → Read AGENTS.md → Combine with Forge identity → Start agent
     ↓
   No → Use default Forge prompt → Start agent
     ↓
User issues task → Agent follows Forge behavior + AGENTS.md context
     ↓
[User types /agents create]
     ↓
Forge analyzes project → Generates AGENTS.md → Saves to workspace
     ↓
User reviews/edits → Types /agents reload → New context active
```

### Success States

**Seamless Integration**
- User with existing AGENTS.md sees no difference in Forge startup
- Agent behavior reflects project conventions without explicit configuration
- Context command shows AGENTS.md content is active

**Creation Flow**
- Generated AGENTS.md captures actual project structure (package.json scripts, Makefile targets, test commands)
- File is immediately usable and follows best practices

### Error/Edge States

**Missing AGENTS.md**
- Forge uses default prompt, no error message (backward compatible)

**Malformed AGENTS.md**
- Forge logs warning but continues with best-effort parsing
- Unparseable sections are skipped, valid sections used

**Very Large AGENTS.md**
- Forge warns user that file is consuming significant context budget
- Offers to summarize or suggests trimming

**Permission Error**
- If AGENTS.md exists but isn't readable, log warning and fall back to default

**Conflicting Instructions**
- AGENTS.md overrides Forge defaults for specific domains
- Core Forge identity (agent loop, tool usage) always preserved

## User Interface & Interaction Design

### Key Interactions

**Automatic Loading (Transparent)**
```
$ forge
Starting Forge in /home/user/myproject
✓ Loaded repository context from AGENTS.md (2.3 KB)
Ready to assist!
```

**Context Command**
```
$ /context

Context Usage: 45,231 / 128,000 tokens (35%)

Sources:
- System Prompt (Forge Identity): 8,450 tokens
- Repository Context (AGENTS.md): 3,220 tokens
- Conversation History: 33,561 tokens
```

**Create Command**
```
$ /agents create

Analyzing project structure...
✓ Detected package.json
✓ Found Makefile
✓ Discovered .github/workflows

Generated AGENTS.md with:
- Setup commands (npm install, make build)
- Test commands (npm test, make test)
- Code style (detected ESLint, Prettier)

Review and edit: /agents edit
```

**Edit Command**
```
$ /agents edit
Opening AGENTS.md in $EDITOR...
(Editor opens)
```

**Reload Command**
```
$ /agents reload
✓ Reloaded AGENTS.md (2.5 KB, +200 bytes)
Context updated!
```

### Information Architecture

System prompt structure:
1. **<custom_instructions>** - Forge Core Identity (always present, defines agent behavior)
2. **<repository_context>** - AGENTS.md content (project-specific guidance)
3. **<system_capabilities>** - Agent capabilities
4. **<agent_loop>** - Agent loop mechanics
5. **<available_tools>** - Tool schemas

The repository context is clearly separated and labeled, never mixed with Forge's identity.

### Progressive Disclosure

- Basic usage: AGENTS.md just works, no UI needed
- Intermediate: `/context` shows what's loaded
- Advanced: `/agents` commands for creation/editing
- Expert: Direct file editing, nested AGENTS.md in monorepos

## Feature Metrics & Success Criteria

### Key Performance Indicators

- **Adoption Rate**: % of Forge sessions in repos with AGENTS.md
- **Auto-Detection Success**: % of AGENTS.md files successfully loaded
- **Creation Rate**: % of users who create AGENTS.md via `/agents create`
- **File Size Distribution**: Median/p95 AGENTS.md file sizes
- **Error Rate**: % of sessions with AGENTS.md loading errors

### Success Thresholds

**V1.0 Success:**
- 95%+ of existing AGENTS.md files load without errors
- <1% of sessions encounter AGENTS.md-related errors
- Zero regression in default (no AGENTS.md) experience

**V1.1 Success:**
- 10%+ of users create AGENTS.md via Forge
- Median AGENTS.md file size under 5KB
- Nested AGENTS.md support works in 95%+ of monorepos

## User Enablement

### Discoverability

**For Existing AGENTS.md Users:**
- Startup message confirms AGENTS.md was loaded
- Documentation mentions AGENTS.md support prominently
- GitHub README includes badge/note about standard support

**For New Users:**
- `/help` mentions `/agents` commands
- Documentation tutorial on creating AGENTS.md
- Example AGENTS.md files in Forge repo examples/

### Onboarding

**First-Time AGENTS.md Creation:**
1. User types `/agents create`
2. Forge analyzes project, shows preview
3. User confirms or edits sections
4. File saved, user prompted to review
5. Immediate feedback on what changed in agent behavior

### Mastery Path

**Novice**: Uses auto-detected AGENTS.md, doesn't think about it
**Intermediate**: Checks `/context` to see what's loaded, uses `/agents edit`
**Advanced**: Creates optimized AGENTS.md with project conventions
**Expert**: Uses nested AGENTS.md in monorepos, fine-tunes context budget

## Risk & Mitigation

### User Risks

**Risk: Users expect AGENTS.md to override Forge behavior**
- Mitigation: Clear documentation that AGENTS.md is repository context, not agent configuration
- AGENTS.md provides project-specific information (setup, testing, conventions)
- Forge's agent identity and loop mechanics are never modified by AGENTS.md

**Risk: Large AGENTS.md files consume too much context**
- Mitigation: Warning at 10KB, error at 20KB, with guidance to trim
- Consider summarization for very large files

**Risk: Malformed AGENTS.md breaks agent startup**
- Mitigation: Robust parsing with fallback to default prompt
- Clear error messages pointing to problematic lines

**Risk: Nested AGENTS.md causes confusing behavior**
- Mitigation: `/context` clearly shows which AGENTS.md is active
- Precedence rules documented and intuitive (closest wins)

### Adoption Risks

**Risk: Users don't discover AGENTS.md support**
- Mitigation: Prominent documentation, blog post, GitHub badge
- Startup message when AGENTS.md is detected

**Risk: Generated AGENTS.md is low quality**
- Mitigation: Template-based generation with human review step
- Start simple, iterate based on feedback

**Risk: Incompatibility with other tools' AGENTS.md expectations**
- Mitigation: Follow standard strictly, don't add proprietary extensions
- Test with AGENTS.md files from popular projects

## Dependencies & Integration Points

### Feature Dependencies

**Existing:**
- Context management system (for displaying AGENTS.md content)
- Workspace guard (for validating AGENTS.md location)
- Prompt builder (for combining Forge identity + AGENTS.md)

**New:**
- File watcher for AGENTS.md changes (for `/agents reload`)
- Markdown parser (simple, no need for full renderer)
- Project analyzer (for `/agents create`)

### System Integration

- **Prompt System**: AGENTS.md content injected after Forge identity, before tools
- **Context Manager**: AGENTS.md tracked separately for token accounting
- **Slash Commands**: New `/agents` command family
- **Settings System**: Config option to disable AGENTS.md auto-loading

### External Dependencies

- Standard library file I/O (no external deps needed)
- Existing workspace security model
- Optional: External editor for `/agents edit`

## Constraints & Trade-offs

### Design Decisions

**Decision: Additive approach (Forge identity + AGENTS.md repository context)**
- Rationale: AGENTS.md is repository-specific guidance, not agent behavior override
- AGENTS.md appears as separate <repository_context> section in system prompt
- Forge's core identity, agent loop, and tool usage patterns are never overridden
- Trade-off: Clear separation of concerns, no confusion about precedence

**Decision: Closest-wins for nested files**
- Rationale: Matches user intuition, aligns with other standards
- Trade-off: No merging means some duplication in nested files

**Decision: Auto-load by default**
- Rationale: Zero-config experience, matches user expectations
- Trade-off: Adds small startup overhead, but negligible

**Decision: Plain markdown parsing**
- Rationale: Keep it simple, avoid over-engineering
- Trade-off: Miss some structure, but standard doesn't require it

### Known Limitations

- No semantic understanding of AGENTS.md sections
- Nested file merging not supported (closest file wins entirely)
- Large AGENTS.md files can consume significant context budget
- No validation of command correctness (e.g., "npm test" actually works)

### Future Considerations

**V2.0 Features:**
- Semantic section understanding (setup vs testing vs style)
- Smart merging of nested AGENTS.md files
- Template marketplace for popular frameworks
- Integration with project scaffolding tools
- AI-powered AGENTS.md optimization suggestions

## Competitive Analysis

### How Other Tools Handle AGENTS.md

**Cursor**
- Auto-loads AGENTS.md
- Shows in context window
- No special commands for creation/editing

**GitHub Copilot**
- Supports AGENTS.md via workspace context
- Simple integration, minimal UI

**Devin**
- Full AGENTS.md support
- Can generate AGENTS.md automatically

**Windsurf**
- AGENTS.md as primary context source
- Nested file support

### Forge Differentiators

- **Superior agent loop**: AGENTS.md provides context, Forge provides execution
- **Advanced TUI**: Better visualization of what's loaded
- **Creation tools**: `/agents create` analyzes project better than competitors
- **Context management**: Better token tracking and optimization

## Go-to-Market Considerations

### Positioning

**Message**: "Forge supports the AGENTS.md standard - your existing project context works out of the box."

**Key Points:**
- Drop-in compatibility with 60k+ projects
- Zero configuration needed
- Part of open ecosystem, not proprietary

### Documentation Needs

**New Docs:**
- AGENTS.md support overview page
- Tutorial: Creating your first AGENTS.md
- Reference: Nested AGENTS.md in monorepos
- FAQ: How Forge uses AGENTS.md

**Updated Docs:**
- Getting Started: Mention AGENTS.md detection
- Configuration: Document `-prompt` flag interaction
- Context Management: Include AGENTS.md in context tracking

### Support Requirements

**Common Questions:**
- "Why isn't my AGENTS.md being read?" → Check workspace root, permissions
- "How do I override AGENTS.md?" → Use `-prompt` flag
- "Can I have multiple AGENTS.md files?" → Yes, in monorepos

**Support Tools:**
- Debug command to show AGENTS.md loading status
- Validation tool to check AGENTS.md format
- Example AGENTS.md files for common setups

## Evolution & Roadmap

### Version History

**v1.0 (MVP)**
- Auto-detect and load AGENTS.md from workspace root
- Hybrid prompt composition (Forge + AGENTS.md)
- `/context` shows AGENTS.md content
- Graceful fallback if missing

**v1.1 (Enhanced)**
- `/agents create` command
- `/agents edit` command
- `/agents reload` command
- Warning for large files

**v1.2 (Monorepo)**
- Nested AGENTS.md support
- Closest-wins precedence
- Context command shows active file path

### Future Vision

**v2.0 (Intelligent)**
- AI-powered AGENTS.md optimization
- Automatic updates based on project changes
- Smart merging of nested files
- Template marketplace

**v3.0 (Ecosystem)**
- Share AGENTS.md templates community-wide
- Integration with scaffolding tools
- Cross-project AGENTS.md inheritance
- Version control and diffs

### Deprecation Strategy

N/A - AGENTS.md is additive, no deprecation planned. If standard evolves, Forge will maintain backward compatibility while adding new features.

## Technical References

- **Architecture**: [ADR-XXX: AGENTS.md Integration Architecture]
- **Implementation**: [ADR-YYY: Prompt Composition with AGENTS.md]
- **API Specification**: See workspace guard, prompt builder packages

## Appendix

### Research & Validation

**Standard Analysis:**
- Reviewed 50+ open-source AGENTS.md files
- Common sections: setup, testing, style, PR guidelines
- Size range: 500 bytes to 15KB (median ~2KB)
- Format: 95% use standard markdown headers

**User Interviews:**
- 15 developers using multiple AI tools
- Pain point: Configuring each tool separately
- Desire: "Just works" with existing AGENTS.md

### Design Artifacts

**Example AGENTS.md (Generated by Forge):**

```markdown
# AGENTS.md

## Setup Commands
- Install dependencies: `go mod download`
- Build: `make build`
- Run tests: `make test`

## Code Style
- Go formatting: `gofmt` (enforced by CI)
- Linting: `golangci-lint run`
- Import order: stdlib, external, internal

## Testing Guidelines
- All new features require tests
- Run `make test` before committing
- Minimum 80% coverage for new code

## PR Guidelines
- Title format: `[type] description`
- Types: feat, fix, docs, refactor, test
- Always run `make lint` and `make test` before creating PR
```

**Prompt Composition Example:**

```
<custom_instructions>
# Forge Coding Assistant: Core Identity
You are Forge, an elite coding assistant...

# Core Principles
1. Clarity and Simplicity...

[... Forge identity sections ...]
</custom_instructions>

<repository_context>
# Repository Guidelines (from AGENTS.md)

## Setup Commands
- Install dependencies: `go mod download`
- Build: `make build`
- Run tests: `make test`

## Code Style
- Go formatting: `gofmt` (enforced by CI)
- Linting: `golangci-lint run`

## Testing Guidelines
- All new features require tests
- Minimum 80% coverage for new code
</repository_context>

<system_capabilities>
[... rest of Forge system prompt ...]
</system_capabilities>
```
