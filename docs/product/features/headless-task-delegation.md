# Headless Task Delegation

## Product Vision

Headless Task Delegation transforms Forge from a single-context agent into an intelligent orchestrator that can spawn focused, autonomous sub-executions for well-scoped tasks. This feature enables the interactive agent to delegate focused work to fresh headless instances with built-in quality validation, dramatically improving token efficiency, task decomposition, and autonomous quality assurance.

**Strategic Purpose**: Position Forge as an AI coding assistant that not only executes tasks but intelligently decomposes and delegates work—maintaining clean context while ensuring quality through autonomous retry loops. This bridges the gap between interactive development and autonomous execution, giving users the best of both worlds.

## Key Value Propositions

- **For Software Engineers**: Get focused, quality-validated work on subtasks without cluttering your interactive conversation context. Each delegation starts fresh and completes autonomously with quality gates.

- **For Technical Leads**: Trust that delegated subtasks have explicit quality requirements (tests, linting) and will retry until they pass or clearly fail—no "hope it works" commits.

- **For Power Users**: Unlock advanced task decomposition patterns where complex work is broken into focused, validated chunks that execute in clean context slates.

- **Competitive Advantage**: No other AI coding assistant combines interactive planning with autonomous, quality-gated sub-execution. This is unique to Forge's headless mode architecture.

## Target Users & Use Cases

### Primary Personas

- **Experienced Developer**: Familiar with Forge's capabilities, wants efficient token usage and clean context management for complex multi-step tasks.

- **Power User/Team Lead**: Uses Forge extensively, values explicit quality gates and decomposition strategies for maintaining code quality at scale.

### Core Use Cases

1. **Context-Heavy Refactoring**: Agent is deep into a complex task with 100+ messages. Delegate a focused subtask (e.g., "refactor auth module") to a fresh headless instance with test gates, keeping main context clean.

2. **Quality-Critical Changes**: User wants a subtask (e.g., "add input validation") completed with guaranteed test coverage. Delegate with quality gates that retry until tests pass.

3. **Token-Efficient Workflows**: Large tasks benefit from decomposition into smaller subtasks, each executed in fresh context without accumulating token costs in the main conversation.

4. **Autonomous Retry Loops**: Subtasks that might need multiple attempts (e.g., "fix linting errors") delegate to headless instance that autonomously retries until quality gates pass.

## Product Requirements

### Must Have (P0)

- **Settings Toggle**: Experimental feature flag (`experimental.enable_headless_delegation`) that defaults to `false`
- **System Prompt Injection**: When enabled, add delegation guidance to system prompt
- **Pattern Documentation**: Clear guidance on when/how to delegate, config structure, result processing
- **Example Workflows**: Show complete delegation patterns (write config → execute → read results → cleanup)
- **Clean Failure Handling**: Agent understands exit codes and artifact structure for success/failure handling
- **Workspace Containment**: All delegation artifacts stay within workspace (`.forge/` directory)

### Should Have (P1)

- **Default Constraint Templates**: Sensible defaults in settings for timeout, max_retries, max_files
- **Result Integration Guidance**: Best practices for processing artifacts and integrating changes
- **Progress Visibility**: Agent logs what it's doing ("Delegating auth refactor to headless instance...")
- **Error Recovery**: Guidance on what to do when delegation fails (read logs, adjust approach, retry)

### Could Have (P2)

- **Delegation Metrics**: Track delegation usage, success rates, common patterns
- **Template Library**: Pre-built config templates for common delegation scenarios
- **Nested Delegation Prevention**: Explicit guidance that headless instances should NOT spawn more headless instances
- **Cost Tracking**: Estimate/track token costs of delegated tasks
- **Parallel Delegation**: Future enhancement for independent parallel subtasks (v2.0+)

## User Experience Flow

### Entry Points

**For Users:**
1. Enable via settings: `forge settings set experimental.enable_headless_delegation true`
2. Or edit `~/.config/forge/settings.yaml` directly

**For Agent:**
- System prompt includes delegation guidance when user has feature enabled
- Agent recognizes delegation opportunities during task planning

### Core User Journey

```
[User gives complex task to interactive Forge]
     ↓
[Agent analyzes: "This has 3 independent subtasks"]
     ↓
[Agent: "I'll handle subtask 1 interactively, delegate subtask 2 to headless"]
     ↓
[Agent writes .forge/subtask-auth-20250121.yaml]
     ↓
[Agent executes: forge -headless -headless-config .forge/subtask-auth-20250121.yaml]
     ↓
[Headless instance starts with fresh context]
     ↓
[Headless makes changes, runs quality gates]
     ↓
     ├─→ [Quality gates fail] → [Retry with fixes] → [Loop up to max retries]
     └─→ [Quality gates pass] → [Write artifacts] → [Exit 0]
     ↓
[Agent reads ./headless-output/execution.json]
     ↓
     ├─→ [Success] → [Integrate changes] → [Continue to subtask 3]
     └─→ [Failure] → [Read logs] → [Adjust approach] → [Handle manually or retry]
     ↓
[Agent cleans up: rm .forge/subtask-auth-20250121.yaml, rm -rf ./headless-output/]
     ↓
[User sees clean outcome without context pollution]
```

### Success States

**Successful Delegation:**
- Config file written to `.forge/subtask-{id}.yaml`
- Headless execution completes (exit code 0)
- Quality gates passed (after 0-N retries)
- Artifacts generated in `./headless-output/`
- Agent integrates changes seamlessly
- Cleanup completes (config and artifacts removed)
- Main conversation context remains clean

**Partial Success:**
- Headless execution completes but quality gates fail after max retries
- Agent reads failure artifacts, understands what went wrong
- Agent either handles manually or adjusts approach and retries

### Error/Edge States

**Delegation Failure:**
- Headless execution fails (exit code != 0)
- Agent reads `./headless-output/execution.json` for error details
- Agent communicates failure to user with actionable information
- Recovery: Manual handling or adjusted retry

**Configuration Error:**
- Invalid YAML config written
- Agent sees validation error in headless output
- Recovery: Fix config and retry

**Workspace Conflicts:**
- Headless instance modifies files the main agent is working on
- Git state changes unexpectedly
- Recovery: Agent detects changes, adapts or reports conflict

**Quality Gate Timeout:**
- Headless instance hits max retries without passing gates
- Agent receives failure status with attempt logs
- Recovery: Agent analyzes failure patterns, adjusts constraints or approach

## User Interface & Interaction Design

### Key Interactions

**User Enablement:**
```bash
# Via CLI
forge settings set experimental.enable_headless_delegation true

# Via config file
echo "experimental:
  enable_headless_delegation: true" >> ~/.config/forge/settings.yaml
```

**Agent Delegation Pattern:**
```
User: "Refactor the entire authentication system, add tests, and update docs"

Agent: "I'll break this into focused subtasks:
1. Refactor auth module (delegate to headless with test gates)
2. Update documentation (I'll handle interactively)
3. Add integration tests (delegate to headless)"

[Agent writes .forge/auth-refactor-20250121.yaml]
[Agent: "🔄 Delegating auth refactor to headless instance..."]
[Agent executes: forge -headless -headless-config .forge/auth-refactor-20250121.yaml]
[Agent: "⏳ Waiting for completion..."]
[Headless completes after 2 quality gate retries]
[Agent: "✅ Auth refactor complete - tests passed on retry 2"]
[Agent reads artifacts and continues]
```

### Information Architecture

**Agent Knows:**
- When to delegate (focused scope, quality gates needed, context pollution risk)
- How to write valid headless config YAML
- Command syntax: `forge -headless -headless-config {path}`
- Where to find results (`./headless-output/`)
- How to interpret exit codes and artifacts
- Cleanup responsibilities

**Agent Sees (in artifacts):**
```json
{
  "task": "Refactor authentication module",
  "status": "success",
  "files_modified": 5,
  "quality_gates": [
    {
      "name": "tests",
      "status": "passed",
      "attempts": 2,
      "output": "..."
    }
  ],
  "execution_time": "45s"
}
```

### Progressive Disclosure

**Level 1 - Basic Delegation:**
Agent learns simple pattern: write config → execute → read result

**Level 2 - Quality Gate Customization:**
Agent learns to tune retry counts, timeouts, gate requirements per subtask

**Level 3 - Advanced Orchestration:**
Agent learns complex decomposition strategies, when to delegate vs. handle directly

## Feature Metrics & Success Criteria

### Key Performance Indicators

**Adoption:**
- % of active users with `enable_headless_delegation: true`
- Number of delegation invocations per user per week
- Retention: Do users keep it enabled after trying?

**Engagement:**
- Delegations per interactive session
- Average subtask complexity (files modified, tokens saved)
- Quality gate pass rate vs. retry rate

**Success Rate:**
- % of delegations that complete successfully (exit code 0)
- % that pass quality gates within max retries
- % where agent successfully integrates results

**User Satisfaction:**
- Feedback on token efficiency improvement
- Reports of cleaner conversation context
- Quality improvement from explicit gates

### Success Thresholds

**3 Month Success (Experimental Phase):**
- 20% of power users enable the feature
- 80%+ delegation success rate
- 90%+ quality gate eventual pass rate (within max retries)
- Positive qualitative feedback ("game changer for complex tasks")

**6 Month Success (Promotion to Stable):**
- 40% of active users have tried it
- 85%+ delegation success rate
- Clear usage patterns emerge (common delegation scenarios)
- Zero critical incidents from delegation bugs

## User Enablement

### Discoverability

**In-Product:**
- Mention in experimental features documentation
- Settings UI shows toggle with description
- Interactive agent can suggest enabling it when pattern fits

**External:**
- Blog post: "Forge's Secret Weapon: Headless Task Delegation"
- Twitter thread showing before/after context efficiency
- YouTube demo of complex task decomposition

### Onboarding

**First-Time Flow:**
1. User enables setting: `experimental.enable_headless_delegation: true`
2. Next interactive session, agent has delegation capability
3. When appropriate task arises, agent explains: "I can delegate this focused subtask to a headless instance with quality gates. This keeps our context clean and ensures tests pass. Proceeding..."
4. User sees delegation in action, understands pattern

**Documentation:**
- Quick start guide: "Enable and Use Headless Delegation"
- Config template examples
- Common patterns library

### Mastery Path

**Novice:** Agent handles delegation autonomously, user just observes benefits (cleaner context, quality gates)

**Intermediate:** User understands when delegation happens, sees artifacts, trusts the pattern

**Advanced:** User explicitly requests delegation ("delegate this to headless with strict test gates"), tunes settings, understands trade-offs

## Risk & Mitigation

### User Risks

**Risk: Unexpected Cost**
- Delegated headless instances consume additional LLM tokens
- **Mitigation:** Experimental flag defaults off, clear documentation about token costs, guidance on when NOT to delegate

**Risk: Slower Execution**
- Delegation has overhead (spawn process, fresh context)
- **Mitigation:** Agent only delegates when benefits outweigh overhead (focused scope, quality needs, context pollution)

**Risk: Confusing Failures**
- Headless instance fails, user doesn't understand why
- **Mitigation:** Agent reads failure artifacts, explains clearly what went wrong, suggests fixes

**Risk: Workspace Conflicts**
- Headless instance modifies files main agent is using
- **Mitigation:** Agent tracks delegation scope, avoids overlapping file sets, detects unexpected changes

### Adoption Risks

**Risk: Users Don't Discover It**
- Experimental flag is off by default
- **Mitigation:** Documentation, blog posts, in-product suggestions when pattern fits

**Risk: Agent Over-Delegates**
- Agent delegates too aggressively, creates overhead
- **Mitigation:** Clear guidance in system prompt on when delegation is appropriate vs. overkill

**Risk: Quality Gates Too Strict**
- Delegated tasks fail because gates are unrealistic
- **Mitigation:** Agent learns to set appropriate gates, can retry with adjusted constraints

**Risk: Users Distrust Autonomous Sub-Execution**
- "What's it doing in the background?"
- **Mitigation:** Agent logs delegation clearly, shows artifacts, emphasizes it's the same Forge they trust

## Dependencies & Integration Points

### Feature Dependencies

**Required:**
- Headless mode (already exists) - ADR-0026
- Quality gate retry mechanism (already exists) - ADR-0028
- `write_file` tool (already exists)
- `execute_command` tool (already exists)
- Settings system (already exists) - ADR-0017

**Optional:**
- Git integration for subtask isolation (future enhancement)
- Metrics tracking for delegation analytics

### System Integration

**Settings System:**
```yaml
experimental:
  enable_headless_delegation: false
  headless_delegation_defaults:
    timeout: 600
    max_retries: 3
    max_files: 10
```

**System Prompt:**
- Conditional injection of delegation guidance when setting enabled
- Existing headless mode prompts remain unchanged

**Workspace:**
- `.forge/` directory for config files
- `./headless-output/` for artifacts (temporary)
- Workspace guard ensures containment

### External Dependencies

**None** - This is purely internal orchestration using existing capabilities.

## Constraints & Trade-offs

### Design Decisions

**Decision: No Dedicated Tool**
- **Rationale:** Agent already has write_file and execute_command. Teaching the pattern is more flexible and natural than a specialized tool.
- **Trade-off:** Slightly more complex guidance, but much more flexible and extensible.

**Decision: Experimental Flag (Off by Default)**
- **Rationale:** This is advanced functionality with cost implications. Let power users opt in.
- **Trade-off:** Slower initial adoption, but safer and builds trust.

**Decision: Sequential Only (No Parallel)**
- **Rationale:** V1.0 keeps it simple - one delegation at a time (blocking). Parallel is future enhancement.
- **Trade-off:** Can't speed up independent subtasks via parallelism, but avoids complex coordination.

**Decision: Agent Decides When to Delegate**
- **Rationale:** Agent understands context, scope, and benefits better than most users.
- **Trade-off:** Users have less explicit control, but better UX and fewer mistakes.

**Decision: .forge/ for Config Files**
- **Rationale:** Keeps configs workspace-contained, follows existing convention.
- **Trade-off:** Requires cleanup, but prevents workspace pollution.

### Known Limitations

**V1.0 Scope Boundaries:**
- No parallel delegation (sequential only)
- No nested delegation (headless can't spawn more headless)
- No cross-workspace delegation
- No delegation from headless mode (interactive → headless only)
- No automatic cost tracking/budgets

**Explicit Non-Goals:**
- Multi-agent coordination
- Real-time progress streaming from delegated tasks
- Delegation to different models/providers
- Shared state between delegated instances

### Future Considerations

**V1.1 - Enhancements:**
- Parallel delegation for independent subtasks
- Cost estimation before delegation
- Delegation templates library
- Better artifact cleanup automation

**V2.0 - Advanced Features:**
- Nested delegation with depth limits
- Cross-workspace delegation for monorepo scenarios
- Delegation to specialized models (cheap model for tests, expensive for complex logic)
- Shared scratchpad/notes between instances

## Competitive Analysis

**GitHub Copilot Workspace:**
- No equivalent to headless delegation
- All work happens in single context
- No autonomous retry loops
- **Advantage:** Forge uniquely combines interactive planning with autonomous sub-execution

**Cursor Agent Mode:**
- Single-context execution only
- No task decomposition with quality gates
- **Advantage:** Forge's headless architecture enables this pattern; Cursor would need to rebuild from scratch

**Cody (Sourcegraph):**
- Context management via embeddings, not fresh execution contexts
- No autonomous sub-agents
- **Advantage:** Forge's delegation is actual fresh execution, not just context tricks

**Aider:**
- CLI-only, no interactive → autonomous delegation
- **Advantage:** Forge combines best of both (interactive planning, autonomous execution)

## Go-to-Market Considerations

### Positioning

**Primary Message:**
"Forge now thinks in layers: plan interactively, execute autonomously. Break complex tasks into focused, quality-validated subtasks that run in fresh contexts with automatic retry—keeping your conversation clean and your code quality high."

**Key Benefits:**
- Token efficiency through fresh context delegation
- Quality assurance through autonomous retry loops
- Cleaner interactive conversations
- No context pollution from subtask noise

### Documentation Needs

**Required:**
- Feature guide: "Headless Task Delegation"
- Configuration reference: Settings and defaults
- Pattern library: Common delegation scenarios
- Troubleshooting: Common issues and solutions

**Nice to Have:**
- Video tutorial showing complex task decomposition
- Blog post with real-world examples
- Case studies from early adopters

### Support Requirements

**Support Team Training:**
- How delegation works (high-level flow)
- How to enable/disable the feature
- Common issues: delegation failures, quality gate problems
- Reading delegation artifacts for debugging

**Self-Service Resources:**
- FAQ: "When should I use delegation?"
- Troubleshooting guide: "Delegation failed, now what?"
- Cost calculator: "How much does delegation cost?"

## Evolution & Roadmap

### Version History

**v1.0 (Experimental):**
- Sequential delegation (one at a time)
- Basic system prompt guidance
- Manual cleanup required
- Settings-based enablement

**v1.1 (Refinement):**
- Automatic cleanup patterns
- Default constraint templates
- Usage metrics tracking
- Improved failure handling

**v1.2 (Stability):**
- Promotion from experimental to stable
- Parallel delegation (opt-in)
- Cost estimation and budgets
- Template library

### Future Vision

**V2.0 - Advanced Orchestration:**
- Nested delegation with depth limits (2-3 levels)
- Shared scratchpad across delegated instances
- Cross-workspace delegation for monorepos
- Specialized model routing (cheap for tests, expensive for complex)

**V3.0 - Multi-Agent Coordination:**
- Coordinated parallel delegation with dependency graphs
- Real-time progress aggregation
- Dynamic constraint adjustment based on results
- Learning from delegation patterns (auto-suggest decomposition)

### Deprecation Strategy

**N/A** - This feature builds on stable headless architecture and won't be deprecated. Worst case: disable experimental flag if major issues emerge, but core pattern remains valid.

## Technical References

- **Architecture**: [ADR-0026: Headless Mode Architecture](../../adr/0026-headless-mode-architecture.md)
- **Quality Gates**: [ADR-0028: Quality Gate Architecture](../../adr/0028-quality-gate-architecture.md)
- **Settings System**: [ADR-0017: Auto-Approval and Settings System](../../adr/0017-auto-approval-and-settings-system.md)
- **Implementation**: [ADR-0035: Headless Task Delegation](../../adr/0035-headless-task-delegation.md)

## Appendix

### Research & Validation

**User Research Insights:**
- Power users report token exhaustion in long sessions
- Context pollution is a top complaint ("agent forgot what we were doing")
- Users want quality assurance without manual intervention
- Task decomposition is a learned skill, agent should help

**Early Testing:**
- Dogfooding shows 40% token savings on complex refactors
- Quality gate retry successfully fixes 85% of test failures
- Clean context improves agent focus on main task

### Design Artifacts

**System Prompt Section:**
```markdown
# Headless Task Delegation (Experimental)

For focused subtasks that benefit from clean context and quality validation, you can spawn a headless Forge instance to execute them autonomously.

## When to Delegate

- Well-scoped, independent subtasks
- Work requiring quality gates (tests, linting)
- Tasks that might need multiple retry attempts
- Focused execution without polluting current context

## How to Delegate

1. Write headless config to `.forge/subtask-{unique-id}.yaml`
2. Execute: `forge -headless -headless-config .forge/subtask-{id}.yaml`
3. Read artifacts from `./headless-output/`
4. Clean up config and artifacts

[See full implementation details in system prompt guidance]
```

**Example Config Template:**
```yaml
task: "Add comprehensive error handling to HTTP handlers"
mode: write
workspace_dir: "."

constraints:
  max_files: 10
  timeout: 600s
  
quality_gates:
  - name: "tests"
    command: "go test ./..."
    required: true
  - name: "lint"
    command: "golangci-lint run"
    required: false

quality_gate_max_retries: 3

artifacts:
  enabled: true
  output_dir: "./headless-output"
  json: true
  markdown: true

git:
  auto_commit: false
```
