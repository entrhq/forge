# 35. Auto-Close Command Overlay

**Status:** Accepted
**Date:** 2024-01-09
**Deciders:** Product Team, Engineering Team
**Technical Story:** User request to reduce manual interactions with command execution overlays

---

## Context

Forge uses command execution overlays to show real-time streaming output when the agent runs shell commands via `execute_command`. Currently, these overlays remain open after command completion, requiring users to manually press Esc to dismiss them. This creates friction in agent-driven workflows where commands execute frequently and errors are handled automatically by the AI agent.

### Background

The current command execution flow:
1. Agent calls `execute_command` tool
2. Overlay opens and displays streaming output
3. Command completes with exit code
4. Overlay shows completion status but stays open
5. User must press Esc to close overlay and return to conversation

Exception: Canceled commands already auto-close immediately, and this behavior is well-received.

### Problem Statement

Manual overlay dismissal creates unnecessary friction in the workflow, particularly for:
- **Agent-driven iterations**: Agent executes multiple commands and handles errors automatically
- **Rapid development cycles**: Developers running frequent commands (tests, builds, lints)
- **Background tasks**: Long-running commands complete while user is focused elsewhere

### Goals

- Reduce manual interactions required to complete command execution workflows
- Support agent-driven error handling where AI reads output and resolves issues automatically
- Maintain error visibility for users who prefer manual error review
- Provide configurable automation that respects different workflow preferences

### Non-Goals

- Per-command auto-close rules (future enhancement)
- Context-aware auto-close based on workflow state (future enhancement)
- Automatic error recovery (separate feature)

---

## Decision Drivers

* **Agent-First Design**: Forge is an AI coding assistant - agent should be able to handle command errors without requiring manual user intervention
* **User Control**: Different users have different preferences - some want automation, others want manual control
* **Error Visibility**: Critical errors must not be silently dismissed
* **Workflow Efficiency**: Minimize unnecessary interactions that slow down development
* **Backward Compatibility**: Existing users should not experience unexpected behavior changes

---

## Considered Options

### Option 1: Never Auto-Close on Errors

**Description:** Auto-close only on successful commands (exit code 0), always keep errors open for manual review.

**Pros:**
- Safest approach - no risk of missing error messages
- Clear separation between success and failure cases
- Easy to understand and explain

**Cons:**
- Creates friction in agent-driven workflows where AI handles errors
- Assumes user always needs to manually review errors
- Doesn't align with agent-first philosophy
- Still requires manual Esc on every failed command

### Option 2: Auto-Close Everything by Default

**Description:** Auto-close all commands (success and errors) after a brief delay, no configuration options.

**Pros:**
- Simplest implementation
- Least manual interaction required
- Best for fully automated agent-driven workflows

**Cons:**
- No escape hatch for users who want manual error review
- Risky if toast notifications are missed
- Violates principle of user control
- Breaking change for existing users

### Option 3: Auto-Close All with Keep-Open-on-Error Toggle (CHOSEN)

**Description:** Auto-close all commands by default after a brief delay with toast notification showing exit code. Provide optional "keep open on error" toggle for users who prefer manual error review.

**Pros:**
- Supports agent-driven workflows (default behavior)
- Provides escape hatch for manual error review (toggle)
- Toast notifications ensure error visibility
- Configurable to match user preference
- Aligns with agent-first design philosophy

**Cons:**
- More complex than binary on/off
- Requires careful UI design to explain two-toggle system
- Initial rollout should default to disabled for safety

---

## Decision

**Chosen Option:** Option 3 - Auto-Close All with Keep-Open-on-Error Toggle

### Rationale

This option best balances agent-first design with user control:

1. **Agent-Driven by Default**: In an AI coding assistant, the agent should be able to execute commands and handle errors automatically without manual intervention. Auto-closing all commands supports this workflow.

2. **Error Visibility Maintained**: Toast notifications showing exit codes ensure users are aware of command results even when overlays auto-close. Critical information is not lost.

3. **User Choice Preserved**: Users who prefer manual error inspection can enable "keep open on error" toggle. This respects different workflow preferences without forcing one approach.

4. **Safe Rollout**: Feature defaults to disabled initially, allowing existing users to opt-in. May become default-enabled in future releases once proven stable.

5. **Future-Proof**: This design accommodates future enhancements like per-command rules, context-aware behavior, and smart error categorization.

---

## Consequences

### Positive

- Faster iteration cycles - commands complete and get out of the way automatically
- Better support for agent-driven error handling and recovery
- Fewer manual interactions required to complete tasks
- Configurable to match different user preferences and workflows
- Toast notifications provide passive awareness of command results
- Consistent with existing auto-close behavior for canceled commands

### Negative

- Users may initially miss error messages if they ignore toast notifications
- Two-toggle configuration (auto-close + keep-open-on-error) is more complex than simple on/off
- Requires user education about new feature and its configuration options
- Default disabled state means users must discover and enable feature

### Neutral

- Changes existing workflow expectations (opt-in mitigates this)
- Adds new settings to configuration surface area
- Requires implementation of toast notification system for command completion

---

## Implementation

### Core Components

1. **Configuration Settings** (`pkg/config/ui.go`):
   ```go
   type UIConfig struct {
       AutoCloseCommandOverlay bool          // Enable auto-close feature
       KeepOpenOnError        bool          // Keep overlay open when exit code != 0
       AutoCloseDelay         time.Duration // Delay before closing (500ms-1s)
   }
   ```

2. **Command Overlay Logic** (`pkg/tui/overlays/command.go`):
   - Check auto-close setting on command completion
   - If enabled, check exit code and keep-open-on-error setting
   - Start configurable delay timer
   - Auto-close overlay after delay
   - Send toast notification with completion status

3. **Toast Notifications**:
   - Show exit code and basic status (success/error)
   - Brief, unobtrusive display
   - Provides passive awareness without blocking workflow

4. **Settings UI**:
   - Checkbox for "Auto-close command overlay"
   - Nested checkbox for "Keep open on errors" (visible when auto-close enabled)
   - Dropdown for delay duration (500ms, 1s, 2s options)
   - Clear help text explaining behavior

### Migration Path

1. Feature ships disabled by default (safe rollout)
2. Documentation and release notes highlight new capability
3. Users discover and opt-in via settings
4. Future release may flip default to enabled once proven stable
5. Backward compatible - no breaking changes to existing behavior

### Timeline

- **Phase 1**: Core implementation (auto-close logic, settings, toast notifications)
- **Phase 2**: UI polish and testing
- **Phase 3**: Documentation and release
- **Future**: Per-command rules, context-aware behavior, ML-based predictions

---

## Validation

### Success Metrics

- **Adoption Rate**: Percentage of users who enable auto-close feature
- **Configuration Patterns**: How many users enable keep-open-on-error vs. full auto-close
- **Workflow Efficiency**: Reduction in manual Esc key presses
- **Error Miss Rate**: How often users miss important errors (monitor support tickets)
- **Feature Satisfaction**: User feedback on auto-close behavior

### Monitoring

- Track setting enable/disable events
- Monitor toast notification display rates
- Collect user feedback on auto-close timing and behavior
- Watch for bug reports about missed errors or unexpected closures

---

## Related Decisions

- [ADR-0013](0013-streaming-command-execution.md) - Streaming Command Execution (foundation for this feature)
- [ADR-0017](0017-auto-approval-and-settings-system.md) - Auto-Approval and Settings System (settings architecture)
- [ADR-0012](0012-enhanced-tui-executor.md) - Enhanced TUI Executor (overlay architecture)

---

## References

- [Product Feature Doc](../product/features/auto-close-command-overlay.md) - Detailed product requirements
- User Request: "I want overlays to auto-close when commands complete"
- Existing behavior: Canceled commands already auto-close successfully

---

## Notes

**Design Evolution**: Initial approach was "never auto-close on errors" but revised to support agent-driven workflows where errors are handled automatically by the AI. The keep-open-on-error toggle provides escape hatch for users who prefer manual error review.

**Key Insight**: In an AI coding assistant, the default workflow should assume the agent can handle errors autonomously. Manual error review should be opt-in, not mandatory.

**Future Considerations**:
- Per-command auto-close rules (e.g., always auto-close git status, never auto-close npm test)
- Context-aware behavior (auto-close during agent iterations, keep open during manual command entry)
- Smart error categorization (recoverable vs. critical failures)
- ML-based prediction of when user wants overlay to stay open

**Last Updated:** 2024-01-09
