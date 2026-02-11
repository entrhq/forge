# Auto-Close Command Overlay

## Product Vision
Reduce friction in the command execution workflow by allowing users to configure automatic closure of command execution overlays. This feature advances Forge's mission of providing a streamlined, efficient AI coding assistant experience by eliminating unnecessary manual interactions and respecting user preferences for automation vs. control.

## Key Value Propositions
- **For power users**: Faster workflow with fewer interruptions - commands complete and overlay auto-closes, returning focus to the main interface immediately
- **For automation-focused users**: Seamless background command execution without requiring manual dismissal of overlays
- **Competitive advantage**: Configurable automation that respects user preference between control and speed, rather than forcing one approach

## Target Users & Use Cases

### Primary Personas
- **Agent-Driven Developer**: Trusts the AI agent to handle command errors automatically, wants minimal UI friction for all commands
- **Fast-Iteration Developer**: Needs rapid feedback cycles, wants minimal UI friction, prefers automation for both successes and failures
- **Cautious Reviewer**: Wants to review command errors before continuing, enables "keep open on errors" setting
- **CI/Automation User**: Running Forge in semi-automated contexts where manual overlay dismissal is impractical

### Core Use Cases
1. **Agent-Driven Workflow**: Developer lets agent handle all command execution and error recovery - all commands auto-close, agent reads output and resolves issues without manual intervention
2. **Rapid Development Workflow**: Developer iteratively runs commands (tests, builds, lints) and wants all executions to complete without manual overlay dismissal - focus returns to agent conversation immediately
3. **Selective Error Review Workflow**: Developer enables "keep open on errors" to manually review failed commands while letting successful commands auto-close
4. **Background Task Execution**: Long-running commands complete while user is focused elsewhere, overlay auto-closes so user doesn't return to a blocked interface

## Product Requirements

### Must Have (P0)
- Settings toggle to enable/disable auto-close of command execution overlays
- Auto-close behavior applies to all completed commands by default (both success and errors)
- Separate toggle to keep overlay open on errors (disabled by default - errors also auto-close)
- Canceled commands continue to auto-close immediately (existing behavior)
- Setting persists across sessions via configuration file
- Clear visual indication when command completes before auto-close (brief status message)
- Configurable delay before auto-close (e.g., 500ms, 1s, 2s) to allow users to glimpse completion status

### Should Have (P1)
- Settings accessible via /settings slash command in TUI
- Toast notification when command completes and overlay auto-closes, showing exit code

### Could Have (P2)
- Per-command pattern configuration (e.g., auto-close for "git status" but not "npm test")
- Keyboard shortcut to temporarily override auto-close behavior for a single command
- Analytics/telemetry on auto-close usage patterns

## User Experience Flow

### Entry Points
- **/settings command**: Opens settings overlay, navigate to "UI Settings" section, toggle "Auto-close command overlay"
- **Configuration file**: Manually edit forge.json to set `ui.auto_close_command_overlay: true`

### Core User Journey
```
[Command Execution Starts] → [Overlay Opens] → [Command Runs]
     ↓
[Command Completes]
     ↓
[Auto-close Enabled?]
     ↓
[Yes] → [Keep Open on Error Enabled?]
    ↓           ↓
  [No]        [Yes] → [Exit Code = 0?]
    ↓                     ↓           ↓
[Brief delay]          [Yes]       [No]
    ↓                     ↓           ↓
[Overlay Auto-Closes] [Brief delay] [Keep Open]
    ↓                     ↓
[Toast with exit code] [Overlay Auto-Closes]
    ↓                     ↓
[User Returns]         [Toast: success]
                          ↓
                       [User Returns]

[No] → [Overlay Stays Open] → [User Presses Esc to Close]
```

### Success States
- Command completes (any exit code)
- Auto-close setting is enabled
- Overlay closes after brief delay (500ms-1s)
- User can immediately continue interacting with Forge
- Toast notification shows completion status and exit code

### Error/Edge States
- **Command fails with keep_open_on_error enabled**: Overlay remains open, user must manually close to review error
- **Command fails with keep_open_on_error disabled**: Overlay auto-closes after delay, toast shows error exit code
- **Command canceled**: Overlay auto-closes immediately (existing behavior, unchanged)
- **Auto-close disabled**: Overlay always requires manual Esc to close (existing behavior, unchanged)
- **Rapid consecutive commands**: Each overlay instance respects auto-close setting independently

## User Interface & Interaction Design

### Key Interactions
- **Settings Toggle**: Simple on/off checkbox in UI Settings section for auto-close
- **Keep Open on Error Toggle**: Optional checkbox to keep errors open for manual review
- **Visual Feedback**: Command status line shows "Completed (exit: X) - closing..." during delay period
- **Toast Notification**: Brief notification showing completion status and exit code when overlay auto-closes

### Information Architecture
**Settings Overlay → UI Settings Section:**
```
UI Settings
-----------
☑ Auto-close command overlay
  Commands will automatically close the overlay after completion.
  A brief delay allows you to see the completion status.

☐ Keep open on errors
  When enabled, failed commands (non-zero exit code) will keep the
  overlay open for review. Otherwise, all commands auto-close.

Delay before auto-close: [500ms ▼]
  How long to wait before closing (allows you to see completion status)
```

### Progressive Disclosure
- Basic auto-close toggle is immediately visible and understandable
- Advanced options (delay duration, keep-open-on-error) are visible when auto-close is enabled
- Inline help text explains behavior without requiring documentation lookup
- Default behavior (auto-close all commands) is clearly stated in the help text

## Feature Metrics & Success Criteria

### Key Performance Indicators
- **Adoption**: % of users who enable auto-close setting
- **Engagement**: Commands executed per session (should increase with reduced friction)
- **Success Rate**: % of command executions that benefit from auto-close (exit code 0)
- **User Satisfaction**: Qualitative feedback on workflow improvement

### Success Thresholds
- 40%+ adoption rate among active users within 30 days
- No increase in command re-execution rate (indicating users aren't missing error messages)
- Positive feedback from beta testers on workflow improvement

## User Enablement

### Discoverability
- Mentioned in settings overlay (users already discover /settings via help or command palette)
- Release notes highlight new feature
- Default setting is OFF to avoid surprising existing users

### Onboarding
- First-time users see tooltip/help text explaining auto-close behavior when they open settings
- Default disabled state means users must opt-in (safe default)

### Mastery Path
1. **Novice**: Discovers setting, enables basic auto-close
2. **Intermediate**: Adjusts delay duration to match their reading speed
3. **Power User**: May enable close-on-error for fully automated workflows, or configure per-pattern rules (future)

## Risk & Mitigation

### User Risks
- **Risk**: User misses important error messages because overlay auto-closed
  - **Mitigation**: Toast notification shows exit code and status, "keep open on errors" toggle available for users who want manual error review
- **Risk**: Auto-close happens too fast, user can't read completion status
  - **Mitigation**: Configurable delay (P0), default 500ms-1s is reasonable
- **Risk**: Users forget they enabled auto-close and are confused when overlay disappears
  - **Mitigation**: Toast notification provides feedback, status line shows "closing..." during delay

### Adoption Risks
- **Risk**: Users don't discover the setting
  - **Mitigation**: Prominent placement in /settings, release notes, community communication
- **Risk**: Users try it and disable it due to confusing behavior
  - **Mitigation**: Clear documentation, safe defaults (disabled initially, errors auto-close by default which matches agent-driven workflow)

## Dependencies & Integration Points

### Feature Dependencies
- Settings system must be functional (already exists in ADR-0017)
- Command execution overlay must emit completion events (already exists)
- Configuration persistence layer must handle new UI section (needs implementation)

### System Integration
- Integrates with existing command execution overlay (pkg/executor/tui/overlay/command.go)
- Leverages settings system architecture (pkg/config/)
- Uses event system for command completion detection (pkg/types/event.go)

### External Dependencies
- None - purely internal feature

## Constraints & Trade-offs

### Design Decisions
- **Decision**: Auto-close on all completions by default, with opt-in to keep errors open
  - **Rationale**: Agent-driven workflow means errors will be handled automatically by the agent. Users who want manual error review can enable "keep open on errors" toggle. Toast notification ensures error visibility either way.
- **Decision**: Default to disabled initially (safe rollout)
  - **Rationale**: Existing users expect current behavior, opt-in prevents surprise behavior change. May become default-enabled in future releases once proven stable.
- **Decision**: Brief delay before close (500ms-1s configurable)
  - **Rationale**: Allows user to glimpse completion status, feels less jarring than instant close

### Known Limitations
- No per-command pattern configuration in v1 (punted to P2/future)
- No keyboard shortcut to override auto-close for individual commands (P2/future)
- Delay duration is global, not context-aware (future could vary by command type)
- Keep-open-on-error is a global setting, not per-command customizable in v1

### Future Considerations
- Command-specific auto-close rules (e.g., always auto-close git status, never auto-close npm test)
- Per-command keep-open-on-error overrides (some commands always show errors, others never)
- Integration with command whitelist (auto-close for whitelisted safe commands only)
- Smart delay based on output length (longer output = longer delay)
- Context-aware behavior (auto-close during agent iterations, keep open during manual command entry)

## Competitive Analysis
- **VS Code**: Command output panels stay open until manually closed - no auto-close
- **Cursor**: Similar - manual close required
- **Other AI coding assistants**: Most don't have equivalent command execution UIs
- **Opportunity**: Forge can differentiate by providing configurable automation that respects user preference

## Go-to-Market Considerations

### Positioning
"Streamline your agent-driven workflow with smart overlay automation - commands complete and get out of your way automatically, letting the AI agent handle errors seamlessly. Enable 'keep open on errors' if you prefer manual error review."

### Documentation Needs
- Settings reference documentation update
- How-to guide: "Customizing command overlay behavior"
- Release notes highlighting feature

### Support Requirements
- FAQ entry on auto-close behavior
- Support team aware of new setting and where to find it
- Known issue tracking: edge cases in auto-close timing

## Evolution & Roadmap

### Version History
- **v1.0** (this feature): Auto-close toggle with keep-open-on-error option, configurable delay, toast notifications
- **v1.1** (future): Per-command pattern configuration, smart delay based on output length
- **v2.0** (future): ML-based prediction of when user wants overlay to stay open vs. close, context-aware auto-close

### Future Vision
- Intelligent auto-close that learns user preferences and adapts to workflow patterns
- Context-aware behavior: auto-close during agent iterations, keep open during manual command entry
- Integration with task/workflow context (auto-close during active coding, keep open during debugging sessions)
- Smart error handling: differentiate between recoverable errors (auto-close) and critical failures (always keep open)
- Voice/natural language control: "auto-close all commands" or "keep errors open"

### Deprecation Strategy
N/A - core feature unlikely to be deprecated

## Technical References
- **Architecture**: ADR-0017 (Auto-Approval and Settings System)
- **Implementation**: This feature document
- **Related**: ADR-0013 (Streaming Command Execution), docs/product/features/streaming-command-execution.md

## Appendix

### Research & Validation
- User request: "I want overlays to auto-close when commands complete"
- Pain point identified: Manual Esc key press required even for successful commands
- Validation: Existing auto-close behavior for canceled commands is well-received
- Design evolution: Initial approach (never auto-close errors) revised to support agent-driven workflows where errors are handled automatically
- Rationale: Toast notifications provide error visibility, keep-open-on-error toggle serves users who prefer manual error review

### Design Artifacts
- Settings UI mockup (to be created during implementation)
- Command overlay state machine diagram (to be created during implementation)
