# Product Requirements: Notes Viewer Slash Command

**Feature:** /notes Command - Agent Scratchpad Viewer  
**Version:** 1.0  
**Status:** Planned  
**Owner:** Core Team  
**Last Updated:** January 2025

---

## Product Vision

Provide complete transparency into the agent's working memory through an intuitive, discoverable interface. The `/notes` command transforms the agent's scratchpad from an opaque black box into a clear window, letting users understand what the agent knows, what it's tracking, and how it's organizing information—building trust and enabling better collaboration.

**Strategic Alignment:** As AI agents become more autonomous, transparency becomes critical for user trust. By exposing the agent's notes system through a familiar slash command interface, we demystify agent behavior, enable debugging of agent reasoning, and create opportunities for human-AI collaboration on complex tasks.

---

## Problem Statement

Users working with AI agents on complex, multi-step tasks face a critical transparency gap that undermines trust and collaboration:

1. **Opaque Agent Memory:** Users can't see what information the agent has gathered, what it's tracking, or what decisions it's made. The agent's scratchpad is invisible.
2. **Trust Erosion:** When agents make unexpected decisions or miss information, users have no way to understand why. "Did it forget? Did it ignore my input? What does it actually know?"
3. **Session Resumption Friction:** Returning to a paused task requires users to re-explain context. They can't see what the agent already has in its notes.
4. **Debugging Black Box:** When agent behavior seems wrong, users can't inspect its working memory to diagnose the issue.
5. **Collaboration Barrier:** Users can't verify that the agent captured critical information or review what it learned from exploration.

**Current Workarounds (All Inadequate):**
- **Ask the agent "what do you know?"** → Agent must summarize from memory, wastes tokens, may be incomplete
- **Read conversation history** → Inefficient, information scattered across multiple turns
- **Repeat information** → Redundant, wastes tokens, still no visibility into agent's notes
- **Trust blindly** → Risky, especially for critical decisions
- **Abandon complex tasks** → Give up on multi-step work requiring agent memory

**Real-World Impact:**
- Developer debugging: "The agent said it found the bug, but I can't see its analysis notes"
- Session resumption: "I stopped work yesterday—what context did the agent save?"
- Decision verification: "Did the agent capture all the constraints I mentioned?"
- Learning barrier: "I don't understand how the agent organizes information"
- Trust issues: "Is the agent actually tracking this, or just pretending?"

**Cost of Opacity:**
- 40% of users don't trust agent memory for complex tasks
- Average 3-5 minutes wasted per session re-explaining context
- 25% of multi-session tasks abandoned due to context loss
- Support burden: "How do I know what the agent remembers?" questions

---

## Key Value Propositions

### For All Users (Transparency)
- **Complete Visibility:** See exactly what the agent has stored in its scratchpad
- **Trust Building:** Verify the agent captured critical information correctly
- **Mental Model:** Understand how the agent organizes and categorizes knowledge
- **Discovery:** Learn what the agent considers important enough to note
- **Confidence:** Work on complex tasks knowing nothing is lost in the black box

### For Users on Complex Tasks (Session Management)
- **Context Verification:** Review what the agent learned before proceeding
- **Session Resumption:** Quickly see what was captured in previous work
- **Progress Tracking:** View notes documenting multi-step task progress
- **Decision Review:** Audit architectural decisions and trade-offs the agent noted
- **Collaboration Checkpoint:** Verify shared understanding before major changes

### For Debugging & Troubleshooting (Diagnosis)
- **Behavior Analysis:** Understand why agent made unexpected decisions
- **Information Gaps:** Identify what the agent is missing
- **Note Quality Assessment:** Verify notes are useful and accurate
- **Learning Opportunities:** See how agent structures knowledge for future reference

### For Learning Users (Education)
- **Agent Behavior Insight:** Learn how AI agents organize working memory
- **Best Practices:** See what information is worth noting
- **Tagging Patterns:** Understand effective categorization strategies
- **Note Lifecycle:** Observe how notes progress from active to scratched

---

## Target Users & Use Cases

### Primary: Developer on Multi-Step Refactoring

**Profile:**
- Working on complex architectural changes spanning multiple sessions
- Needs to verify agent captured key constraints and decisions
- Values transparency in AI-assisted development
- Frequently pauses and resumes work

**Key Use Cases:**
- Verify agent noted architectural decision rationale
- Review what the agent learned from code exploration
- Check that constraints were captured before proceeding
- Resume work by reviewing previous session's notes
- Audit agent's understanding before major changes

**Pain Points Addressed:**
- Uncertainty about what agent remembers
- Fear of agent missing critical context
- Time wasted re-explaining previous decisions
- Risk of inconsistent implementation

**Success Story:**
"I'm refactoring a payment service across 3 days. Before implementing database changes, I type `/notes` to review what the agent captured. I see notes tagged 'architecture' and 'constraints' documenting our decision to use eventual consistency and the foreign key relationships we need to preserve. Perfect—the agent got it all. I can proceed confidently without re-explaining the context. Total verification time: 30 seconds."

**Workflow:**
```
Working on complex refactoring
    ↓
Reach decision point (architectural change)
    ↓
Type /notes to verify agent's understanding
    ↓
Review notes tagged 'architecture', 'constraints'
    ↓
Confirm agent captured key decisions
    ↓
Proceed confidently
    ↓
Total time: <1 minute, high confidence
```

---

### Secondary: User Debugging Unexpected Agent Behavior

**Profile:**
- Agent made surprising decision or missed information
- Needs to diagnose why agent acted unexpectedly
- Values understanding over just getting results
- Wants to improve future interactions

**Key Use Cases:**
- Check if agent noted important information mentioned
- Verify agent's interpretation of requirements
- Identify what agent considers relevant vs. noise
- Understand agent's reasoning process

**Success Story:**
"The agent suggested a fix that doesn't match my requirements. I open `/notes` and search for notes about error handling. I see it created a note tagged 'bug' but missed my constraint about backward compatibility. Now I understand the gap—I can clarify that requirement and the agent can update its notes."

---

### Tertiary: Team Lead Reviewing Agent Work

**Profile:**
- Manages developers using AI-assisted coding
- Reviews agent-assisted PRs and architecture decisions
- Needs to verify agent captured team standards
- Wants transparency in AI decision-making

**Key Use Cases:**
- Audit agent's decision documentation
- Verify compliance with team standards
- Review agent's understanding of requirements
- Assess quality of agent memory

---

## Product Requirements

### Must Have (P0)

**1. Slash Command Access**
- **Requirement:** `/notes` command opens notes viewer overlay
- **Success Criteria:** Command appears in autocomplete, executes instantly
- **Rationale:** Consistent with existing slash command pattern

**2. Two-View Display System**

**List View (Compact Overview):**
- **Requirement:** Show all notes in chronological order (newest first)
- **Display Elements:**
  - Note ID (e.g., "note_1704825600000")
  - Created timestamp (relative: "2 hours ago" or absolute: "Jan 2, 2025 3:30 PM")
  - Content snippet (first 60 characters, truncated with "...")
  - Tags displayed as colored badges/pills
  - Status indicator (icon or color for active/scratched)
- **Success Criteria:** User can scan 10 notes in <5 seconds, identify key notes

**Detail View (Full Note Display):**
- **Requirement:** Show complete note information
- **Display Elements:**
  - Full note content (all text, no truncation)
  - All tags with clear visual separation
  - Created timestamp (full format with date & time)
  - Last updated timestamp (if note was modified)
  - Scratched status (clearly marked if note is scratched)
  - When scratched (timestamp when note was marked scratched)
- **Success Criteria:** All information about note visible without scrolling

**3. Scratched/Active Filter Toggle**
- **Requirement:** Three-state filter: All Notes / Active Only / Scratched Only
- **Default State:** Show all notes (both active and scratched)
- **Interaction:** Keyboard shortcut (e.g., 'f' for filter) cycles through states
- **Visual Feedback:** Clear indicator of current filter state
- **Success Criteria:** User can filter to active-only in 1 keystroke
- **Rationale:** Showing all notes by default provides complete transparency; users can filter to active-only if needed

**4. Navigation Consistency**
- **Requirement:** Follow existing overlay navigation patterns
- **Controls:**
  - Arrow up/down: Navigate between notes in list
  - Enter/Space: Open selected note in detail view
  - Esc/Backspace: Return to list view (or close overlay if in list)
  - q: Close overlay entirely
- **Success Criteria:** Users familiar with other overlays can navigate notes without new learning

**5. Empty State Handling**
- **Requirement:** Clear messaging when no notes exist or filter excludes all
- **States:**
  - No notes at all: "No notes yet. The agent will create notes as it works."
  - All notes filtered out: "No [active/scratched] notes. Press 'f' to change filter."
- **Success Criteria:** User understands why view is empty and how to change it

**6. Visual Design**
- **Requirement:** Consistent with Forge TUI design language
- **Elements:**
  - Color-coded tags (consistent with tag categories: type, domain, status)
  - Clear visual hierarchy (list → detail)
  - Status indicators distinguishable at a glance
  - Readable fonts and spacing
- **Success Criteria:** Notes are easily scannable, information hierarchy clear

**7. Search/Filter Notes**
- **Requirement:** Quick search within notes (press '/' in list view)
- **Behavior:** Filter notes by content or tag as user types
- **Success Criteria:** Find specific note in 3-5 keystrokes
- **Rationale:** Essential for usability with 20+ notes; low implementation complexity, high user value

### Should Have (P1)

**8. Note Count Summary**
- **Requirement:** Header shows total count and breakdown
- **Display:** "Notes: 12 active, 5 scratched, 17 total"
- **Success Criteria:** User knows scope before scrolling

**9. Sort Options**
- **Requirement:** Toggle sort order (newest first ↔ oldest first)
- **Shortcut:** 's' to toggle sort
- **Success Criteria:** View notes in chronological or reverse order

### Could Have (P2)

**10. Tag-Based Filtering**
- **Requirement:** Filter notes by specific tag
- **Interaction:** Press 't' to show tag picker, select tag to filter
- **Success Criteria:** View all notes for a specific category

**11. Copy Note Content**
- **Requirement:** Copy note to clipboard (press 'c' in detail view)
- **Success Criteria:** Note content copied for pasting elsewhere

**12. Export Notes**
- **Requirement:** Export all visible notes to markdown file
- **Success Criteria:** Save notes for external review or archival

---

## User Experience Flow

### Entry Points

**Primary Entry Point:**
- Type `/notes` in chat input
- Appears in slash command autocomplete
- Works from any chat state (idle, mid-conversation, during agent execution)

**Discovery:**
- Slash command autocomplete when user types `/`
- Help command (`/help`) lists `/notes` with description
- Agent mentions notes system in appropriate contexts

### Core User Journey: Verify Agent Context

```
User working on task → wants to verify agent's understanding
    ↓
Type /notes (muscle memory from slash commands)
    ↓
Overlay opens → List View
    ↓
See notes in chronological order (newest at top)
    ↓
Scan overview: timestamps, snippets, tags, status
    ↓
[Decision Point]
    ↓
    ├─→ Found relevant note → Press Enter
    │       ↓
    │   Detail view opens → read full content
    │       ↓
    │   Verify information captured correctly
    │       ↓
    │   Press Esc → back to list or close
    │
    ├─→ Need active notes only → Press 'f' to filter
    │       ↓
    │   View updates to active notes only
    │       ↓
    │   Scan filtered list
    │
    └─→ Information verified → Press 'q' to close
            ↓
        Return to chat
            ↓
        Continue task with confidence
```

### Alternative Flow: Session Resumption

```
User returns to paused task
    ↓
Open /notes to review previous session
    ↓
Filter to active notes (press 'f')
    ↓
Scan notes tagged 'progress', 'decision', 'architecture'
    ↓
Read detail view for key notes
    ↓
Understand current state from agent's notes
    ↓
Close overlay, resume work with context
    ↓
Total time: 1-2 minutes, full context restored
```

### Alternative Flow: Debugging Agent Behavior

```
Agent makes unexpected suggestion
    ↓
Open /notes to diagnose
    ↓
Scan notes for relevant tags ('bug', 'requirements', 'constraints')
    ↓
Open suspicious note in detail view
    ↓
Discover agent missed key constraint
    ↓
Close overlay, clarify requirement in chat
    ↓
Agent updates note, proceeds correctly
```

### Success States

**1. Verification Success**
- User finds note confirming agent captured critical information
- User closes overlay with confidence to proceed
- Outcome: Trust in agent increased, task continues smoothly

**2. Gap Discovery**
- User identifies missing information in notes
- User clarifies requirement in chat
- Agent adds/updates note
- Outcome: Misalignment caught early, corrected before error

**3. Context Restoration**
- User reviews notes from previous session
- User understands current state without re-explaining
- Outcome: Seamless session resumption, no wasted tokens

**4. Learning Achieved**
- User observes how agent organizes information
- User understands effective tagging strategies
- Outcome: Improved collaboration in future sessions

### Error/Edge States

**1. No Notes Exist**
- **State:** Agent hasn't created any notes yet
- **Display:** Empty state message with guidance
- **Message:** "No notes yet. The agent will create notes as it works on tasks."
- **Recovery:** Close overlay, continue working—notes will appear as agent works

**2. All Notes Filtered Out**
- **State:** User filters to "Active Only" but all notes are scratched
- **Display:** Empty state with filter hint
- **Message:** "No active notes. All notes have been scratched. Press 'f' to show all notes."
- **Recovery:** Press 'f' to change filter state

**3. Very Long Note Content**
- **State:** Note content exceeds screen height in detail view
- **Behavior:** Detail view scrollable (arrow keys or mouse)
- **Visual Indicator:** Scroll position indicator (e.g., "45% ↓")
- **Recovery:** Scroll to read full content

**4. Many Notes (Performance)**
- **State:** 100+ notes created in long-running session
- **Behavior:** Virtual scrolling, only render visible notes
- **Performance Target:** Smooth scrolling even with 500+ notes
- **Recovery:** Filter or search to reduce visible set

**5. Note Update During Viewing**
- **State:** Agent updates note while user viewing notes overlay
- **Behavior:** No auto-refresh; updates only visible after closing and reopening overlay
- **Rationale:** Keeps implementation simple; avoids UI flicker or distraction during viewing
- **Recovery:** Close and reopen overlay to see latest notes

---

## User Interface & Interaction Design

### Key Interactions

**List View Navigation:**
- `↑/↓` or `j/k`: Move selection between notes
- `Enter` or `→`: Open selected note in detail view
- `f`: Cycle filter (All → Active Only → Scratched Only → All)
- `s`: Toggle sort (Newest First ↔ Oldest First)
- `/`: Search/filter notes by text or tag (P1)
- `q` or `Esc`: Close overlay
- `?`: Show keyboard shortcuts help

**Detail View Navigation:**
- `↑/↓` or `j/k`: Scroll note content (if exceeds screen)
- `Esc` or `←` or `Backspace`: Return to list view
- `c`: Copy note content to clipboard (P2)
- `n`/`p`: Jump to next/previous note in list (P1)

**Mouse Support (Optional):**
- Click note to select/open
- Scroll wheel to navigate list/scroll content
- Click filter indicator to cycle filter state

### Information Architecture

**Header Section (Fixed):**
```
┌─────────────────────────────────────────────────────────┐
│ Notes (17 total: 12 active, 5 scratched) [Filter: All] │
│ Press 'f' to filter • 's' to sort • '?' for help       │
└─────────────────────────────────────────────────────────┘
```

**List View Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ ▶ note_1704825600000    2 hours ago                    │
│   Decision to use JWT for auth scaling                 │
│   🏷 decision  🏷 auth  🏷 security                     │
├─────────────────────────────────────────────────────────┤
│   note_1704821000000    6 hours ago                    │
│   Payment service depends on user service for...       │
│   🏷 dependency  🏷 api  🏷 auth                        │
├─────────────────────────────────────────────────────────┤
│   note_1704817400000    10 hours ago   ✓ scratched     │
│   Test suite requires DB migration before running      │
│   🏷 pattern  🏷 test  🏷 database                      │
└─────────────────────────────────────────────────────────┘
```

**Detail View Layout:**
```
┌─────────────────────────────────────────────────────────┐
│ Note: note_1704825600000                                │
│ Created: Jan 2, 2025 1:30 PM                            │
│ Updated: Jan 2, 2025 3:45 PM                            │
│ Status: Active                                          │
│ Tags: 🏷 decision  🏷 auth  🏷 security                 │
├─────────────────────────────────────────────────────────┤
│ Decision to use JWT with refresh tokens for auth       │
│ scaling.                                                │
│                                                         │
│ Rationale:                                              │
│ - Stateless auth reduces DB load                       │
│ - Refresh tokens enable token rotation                 │
│ - Industry standard, well-supported libraries          │
│                                                         │
│ Trade-offs:                                             │
│ - Must handle token expiration gracefully              │
│ - Refresh token storage requires secure DB             │
│                                                         │
│ Next steps:                                             │
│ - Implement JWT middleware                             │
│ - Add refresh token endpoint                           │
│ - Update API documentation                             │
└─────────────────────────────────────────────────────────┘
```

### Progressive Disclosure

**Level 1: List View (Overview)**
- Shows essential information: ID, time, snippet, tags, status
- User can quickly scan to find relevant notes
- Minimal cognitive load, fast scanning

**Level 2: Detail View (Full Content)**
- Reveals complete note with all metadata
- User reads full content when specific note is relevant
- Rich information when needed, hidden when not

**Level 3: Filtering (Focus)**
- User can reduce noise by filtering to active/scratched
- Reveals only what's currently relevant
- Gradual complexity as user needs more control

---

## Feature Metrics & Success Criteria

### Key Performance Indicators

**Adoption Metrics:**
- **Usage Rate:** % of users who use `/notes` at least once per week
- **Target:** 40% within 1 month, 60% within 3 months
- **Session Integration:** Average uses of `/notes` per 10 agent turns
- **Target:** 1.5 uses per session for complex tasks

**Engagement Metrics:**
- **View Duration:** Average time spent in notes viewer
- **Target:** 30-60 seconds (long enough to review, short enough to be efficient)
- **Note Inspection Rate:** % of notes opened in detail view
- **Target:** 30% of notes (indicates finding relevant information)
- **Filter Usage:** % of sessions using filter toggle
- **Target:** 50% (indicates users finding feature useful)

**Success Rate Metrics:**
- **Task Confidence:** Post-feature survey: "I felt confident in agent's understanding"
- **Target:** 80% agree/strongly agree
- **Context Loss:** % of users reporting lost context between sessions
- **Target:** Reduce by 50% from baseline
- **Re-explanation Rate:** Instances of users repeating information
- **Target:** Reduce by 40% from baseline

**User Satisfaction:**
- **Transparency Rating:** "I understand what the agent knows"
- **Target:** 4.5/5.0 average rating
- **Trust Improvement:** "I trust the agent with complex tasks"
- **Target:** Increase from 3.2/5 to 4.0/5
- **NPS Impact:** Net Promoter Score increase
- **Target:** +10 points among users who use `/notes` regularly

### Success Thresholds

**Minimum Viable Success (Month 1):**
- 30% of users try `/notes` at least once
- Average 2+ uses per complex task session
- 70% satisfaction rating

**Healthy Adoption (Month 3):**
- 50% weekly active users
- 80% of multi-session tasks include notes review
- 4.0/5 satisfaction rating

**Exceptional Performance (Month 6):**
- 70% weekly active users
- `/notes` becomes second most-used slash command
- 4.5/5 satisfaction rating
- Featured in user testimonials and case studies

---

## User Enablement

### Discoverability

**In-Product Discovery:**
- Slash command autocomplete shows `/notes` with description
- `/help` command lists `/notes` in command reference
- Agent mentions notes when appropriate (e.g., "I've noted this decision" → hint to use `/notes`)
- Welcome message for new users mentions transparency features

**Documentation:**
- Dedicated "Notes Viewer" section in user guide
- Screenshots showing list and detail views
- Video walkthrough demonstrating verification workflow
- FAQ: "How do I see what the agent remembers?"

**Community Sharing:**
- Blog post: "Trust Through Transparency: Understanding Agent Memory"
- Social media examples showing notes review workflow
- User testimonials highlighting trust improvement

### Onboarding

**First-Time User Experience:**

**Step 1: Natural Discovery (Session 1-3)**
- User works with agent on task
- Agent creates notes (visible in agent loop, but opaque)
- User wonders: "What is the agent tracking?"

**Step 2: Command Discovery (Session 3-5)**
- User types `/` to explore commands
- Sees `/notes` with description: "View agent's scratchpad notes"
- Tries command out of curiosity

**Step 3: First Success (First Use)**
- Opens `/notes` overlay
- Sees notes agent created during their session
- Realizes: "Oh, this is what the agent is tracking!"
- Closes overlay, continues work

**Step 4: Habitual Use (Session 5-10)**
- Before major decisions, opens `/notes` to verify understanding
- At session start, reviews notes from previous work
- Becomes part of workflow for complex tasks

**Educational Tooltip (First Use):**
```
┌─────────────────────────────────────────────────────────┐
│ 💡 Quick Tour: Notes Viewer                            │
│                                                         │
│ This shows all the notes your agent has created:       │
│ • ↑/↓ to navigate between notes                        │
│ • Enter to view full note details                      │
│ • 'f' to filter active/scratched notes                 │
│ • 'q' to close                                         │
│                                                         │
│ [ Got it! ] (Esc to dismiss)                           │
└─────────────────────────────────────────────────────────┘
```

### Mastery Path

**Novice (Sessions 1-5):**
- Use `/notes` to browse agent's notes
- Understand what information agent tracks
- Verify agent captured key points
- **Key Skill:** Basic navigation (open, scroll, close)

**Competent (Sessions 5-15):**
- Use filter toggle to focus on active notes
- Review notes before major decisions
- Check notes when resuming previous work
- **Key Skill:** Efficient verification workflow

**Proficient (Sessions 15-30):**
- Habitually verify context before complex operations
- Use notes to debug unexpected agent behavior
- Understand agent's note organization patterns
- **Key Skill:** Diagnostic use of notes viewer

**Expert (Sessions 30+):**
- Preemptively guide agent's note-taking through prompts
- Use notes to audit agent decision-making
- Leverage notes for team collaboration and review
- **Key Skill:** Strategic use for transparency and collaboration

---

## Risk & Mitigation

### User Risks

**Risk 1: Information Overload**
- **Scenario:** User overwhelmed by many notes, gives up
- **Impact:** Medium - discourages feature use
- **Mitigation:**
  - Default to active-only filter (hide scratched)
  - Summary count in header provides context
  - Search/filter (P1) helps narrow focus
  - Good visual hierarchy prioritizes scanning
- **Validation:** User testing with 50+ note sessions

**Risk 2: Misinterpretation**
- **Scenario:** User misunderstands note content or status
- **Impact:** Medium - leads to incorrect assumptions
- **Mitigation:**
  - Clear labeling of scratched vs. active
  - Timestamps provide temporal context
  - Full detail view shows all metadata
  - Agent-generated notes use clear language
- **Validation:** User interviews about note clarity

**Risk 3: Performance Degradation**
- **Scenario:** 500+ notes causes UI lag
- **Impact:** Low - only in extreme cases
- **Mitigation:**
  - Virtual scrolling for large lists
  - Lazy rendering of note content
  - Performance testing with 1000+ notes
  - Pagination fallback if needed
- **Validation:** Load testing, performance benchmarks

**Risk 4: Privacy Concerns**
- **Scenario:** Sensitive information visible in notes
- **Impact:** Low - notes are local, not shared
- **Mitigation:**
  - Clear documentation: notes are local-only
  - No network transmission of notes
  - User controls what information they share with agent
- **Validation:** Security review, privacy documentation

### Adoption Risks

**Risk 1: Feature Goes Undiscovered**
- **Impact:** High - no value if not used
- **Likelihood:** Medium
- **Mitigation:**
  - Prominent in slash command autocomplete
  - Agent hints when notes might be useful
  - Onboarding tutorial highlights transparency
  - Blog post and documentation
- **Validation:** Track discovery rate in first 30 days

**Risk 2: Perceived as Advanced/Expert Feature**
- **Impact:** Medium - limits adoption to power users
- **Likelihood:** Medium
- **Mitigation:**
  - Simple, familiar interface (no learning curve)
  - First-use tooltip explains basics
  - Position as "transparency" not "advanced debugging"
  - User testimonials from beginners
- **Validation:** Adoption rate across user skill levels

**Risk 3: Low Perceived Value**
- **Impact:** High - feature seen as "nice to have"
- **Likelihood:** Low (given scratchpad ADR rationale)
- **Mitigation:**
  - Demonstrate value in onboarding
  - Case studies showing trust improvement
  - Metrics showing context loss reduction
  - Integration into recommended workflows
- **Validation:** User satisfaction surveys, retention analysis

---

## Dependencies & Integration Points

### Feature Dependencies

**Required (Blocking):**
- **Agent Scratchpad System (ADR-0032):** Must be implemented and stable
  - Notes data structure with tags, timestamps, content
  - CRUD operations via notes manager
  - Scratched status support
- **Slash Command System:** Must support `/notes` registration
- **TUI Overlay Framework:** Provides base overlay component

**Optional (Enhancing):**
- **Syntax Highlighting:** For code snippets in note content (P2)
- **Markdown Rendering:** For formatted note content (P2)

### System Integration

**Notes Manager Integration:**
- `/notes` command queries `pkg/memory/notes.Manager`
- Read-only access via `ListNotes()` with filter parameters
- Real-time updates when notes change during viewing

**TUI Integration:**
- Extends `pkg/ui/overlay.Overlay` base component
- Uses `pkg/ui/components` for list and detail rendering
- Follows `pkg/ui/theme` for consistent styling

**Event System:**
- Subscribe to note creation/update events for live refresh
- Emit overlay state changes for logging/metrics

### External Dependencies

**None:** Feature is entirely self-contained within Forge.

---

## Constraints & Trade-offs

### Design Decisions

**Decision 1: Read-Only in MVP**
- **Trade-off:** Simplicity vs. power
- **Rationale:**
  - Primary value is transparency (viewing), not editing
  - Editing raises questions: What should users edit? How does it affect agent?
  - Read-only reduces scope, ships faster
  - Can add editing in v1.1 based on user feedback
- **Alternative Considered:** Full CRUD in MVP
  - **Rejected:** Too much scope, uncertain value
  - **Future:** P2 feature for copy, P3 for edit/delete

**Decision 2: Chronological Default Sort**
- **Trade-off:** Recency bias vs. relevance
- **Rationale:**
  - Most recent notes often most relevant to current task
  - Matches mental model: "What did agent just learn?"
  - Consistent with chat history (recent first)
- **Alternative Considered:** Relevance sort (by tags, content match)
  - **Rejected:** Requires complex ranking algorithm
  - **Future:** P2 feature with search integration

**Decision 3: Three-State Filter (All/Active/Scratched)**
- **Trade-off:** Simplicity vs. granular control
- **Rationale:**
  - Covers core use cases: see everything, see active work, audit scratched
  - Simple toggle interaction (one key)
  - Avoids complex filter UI
- **Alternative Considered:** Multi-criteria filtering (tags, date range, status)
  - **Rejected:** Too complex for MVP, unclear demand
  - **Future:** P2 tag-based filtering, P3 advanced queries

**Decision 4: No Inline Editing**
- **Trade-off:** User control vs. agent authority
- **Rationale:**
  - Scratchpad is agent's working memory—editing raises ownership questions
  - Users can influence notes via conversation (clearer intent)
  - Avoids sync issues between user edits and agent updates
- **Alternative Considered:** Allow editing with "user-modified" flag
  - **Rejected:** Complex, unclear semantics for agent consumption
  - **Future:** P3 user annotations (separate from agent notes)

### Known Limitations

**Limitation 1: No Cross-Session Persistence UI**
- **Scope:** Notes viewer shows current session only
- **Rationale:** Notes manager handles persistence; UI just displays current state
- **Workaround:** Notes persist automatically; users see them on session resume
- **Future:** v1.1 could add session history viewer

**Limitation 2: No Real-Time Updates**
- **Scope:** Notes don't auto-refresh while viewing overlay
- **Rationale:** Keeps implementation simple; avoids UI flicker during rapid note creation
- **Workaround:** Close and reopen overlay to see latest notes
- **Future:** v1.1 could add live update mode with toggle if demand exists

**Limitation 3: No Export/Share**
- **Scope:** Cannot export notes to file or share with others
- **Rationale:** MVP focuses on individual transparency, not collaboration
- **Workaround:** Copy note content manually (P2 feature)
- **Future:** P2 export to markdown, P3 team note sharing

**Limitation 4: No Inline Editing or Deletion**
- **Scope:** Read-only viewing; cannot edit or delete notes from overlay
- **Rationale:** Maintains clear ownership (agent's working memory); users influence via conversation
- **Workaround:** Ask agent to update or scratch notes via chat
- **Future:** v1.2 could add user annotations separate from agent notes

### Future Considerations

**Phase 2 (v1.1 - Enhanced Viewing):**
- In-overlay search with text matching
- Note-to-note navigation (jump to related notes)
- Copy note content to clipboard
- Export notes to markdown file
- Live updates during agent execution

**Phase 3 (v1.2 - User Interaction):**
- User annotations on agent notes
- Pin important notes to top
- Archive notes (beyond scratched)
- Note templates for common patterns

**Phase 4 (v2.0 - Collaboration):**
- Share notes with team members
- Merge notes from multiple agents
- Team note libraries
- Note diff view (track changes over time)

---

## Competitive Analysis

**Cursor AI:**
- **Approach:** No visible notes system; agent memory is opaque
- **Limitation:** Users can't verify what Cursor remembers
- **Our Advantage:** Transparency builds trust, enables debugging

**GitHub Copilot:**
- **Approach:** Context window shown in sidebar (files, symbols)
- **Strength:** Users see input context
- **Limitation:** No working memory or decision tracking
- **Our Advantage:** We show agent's synthesized knowledge, not just input

**ChatGPT Code Interpreter:**
- **Approach:** Shows execution logs, but no persistent notes
- **Limitation:** Context lost between sessions
- **Our Advantage:** Notes persist, support multi-session work

**Slack/Discord Slash Commands:**
- **Approach:** Similar pattern for feature access
- **Strength:** Familiar, discoverable
- **Alignment:** We adopt same UX pattern for consistency

**VS Code Command Palette:**
- **Approach:** Fuzzy search for commands
- **Strength:** Fast, keyboard-driven
- **Future Inspiration:** P1 search feature mirrors this pattern

---

## Go-to-Market Considerations

### Positioning

**Primary Message:**
"See what your AI agent knows. Complete transparency into agent memory builds trust and enables better collaboration on complex tasks."

**Key Benefits (Headline Level):**
- **Verify** agent captured your requirements correctly
- **Resume** work seamlessly by reviewing previous session notes
- **Debug** unexpected behavior by inspecting agent's understanding
- **Learn** how AI agents organize and prioritize information

**Target Messaging by Persona:**
- **Developers:** "Trust but verify—see exactly what the agent learned from your code."
- **Teams:** "Audit agent decisions with full transparency into working memory."
- **Learners:** "Understand how AI thinks by watching its note-taking in action."

### Documentation Needs

**User Guide Section: "Notes Viewer"**
- What are scratchpad notes?
- When to use `/notes`
- Understanding note structure (tags, status, timestamps)
- Interpreting agent notes
- Common workflows (verification, debugging, resumption)

**Quick Start Guide:**
- One-page PDF: "Verify Agent Context with /notes"
- Screenshots of list and detail views
- 3-minute video walkthrough

**FAQ:**
- Q: "How do I see what the agent remembers?"
  A: "Type `/notes` to open the notes viewer..."
- Q: "Can I edit or delete notes?"
  A: "Currently read-only (MVP). Future versions will support editing..."
- Q: "Do notes persist between sessions?"
  A: "Yes, notes are saved automatically..."

**Developer Documentation:**
- Notes data model reference
- How agents create effective notes
- Best practices for note organization

### Support Requirements

**Support Team Training:**
- How to guide users to `/notes` for troubleshooting
- Common scenarios: verification, debugging, session resumption
- How to interpret note content for support diagnosis
- Escalation path if notes reveal agent bugs

**Support Macros:**
- "Please check `/notes` to verify agent captured your requirement..."
- "Open `/notes` and filter to active notes to see current context..."
- "If you see a note tagged 'bug', please share the content for review..."

**Troubleshooting Guide:**
- Notes viewer won't open → Check for keybinding conflicts
- No notes shown → Verify agent has created notes (check conversation)
- Performance issues → Report number of notes, session duration

---

## Evolution & Roadmap

### Version History

**v1.0 (MVP - This PRD):**
- Slash command `/notes` integration
- List view (compact overview) with chronological sort
- Detail view (full note display)
- Three-state filter (All/Active/Scratched)
- Read-only viewing
- Keyboard navigation consistent with other overlays

**v1.1 (Enhanced Viewing):**
- In-overlay search/filter by text or tags
- Copy note content to clipboard
- Export notes to markdown
- Sort toggle (newest/oldest first)
- Live updates during agent execution
- Note count summary in header

**v1.2 (User Interaction):**
- User annotations (separate from agent notes)
- Pin important notes
- Archive functionality
- Manual note creation (user-authored notes)

**v2.0 (Collaboration):**
- Share notes with team members
- Team note libraries
- Note templates for common patterns
- Cross-session note analytics
- Note diff view (track changes)

### Future Vision

**AI-Powered Enhancements:**
- Automatic note summarization for long notes
- Related note suggestions
- Anomaly detection (notes that seem incorrect)
- Note quality scoring

**Advanced Organization:**
- Custom tag taxonomies
- Hierarchical note structure
- Note relationships (dependencies, references)
- Smart filtering based on task context

**Collaboration Features:**
- Real-time note sharing during pair programming
- Team note templates and standards
- Note review and approval workflows
- Cross-project note search

### Deprecation Strategy

**Not Applicable:** Core feature with no planned deprecation.

**If Future Replacement:**
- Provide migration path from notes to new system
- Maintain read-only access to archived notes
- Clear communication timeline (6+ months notice)
- Export functionality to preserve data

---

## Technical References

- **Architecture:** [ADR-0032: Agent Scratchpad & Notes System](../../adr/0032-agent-scratchpad-notes-system.md)
- **Implementation:** Notes Manager (`pkg/memory/notes/manager.go`)
- **Tool Integration:** Scratchpad tools (`pkg/tools/scratchpad/`)
- **Related Features:** [Slash Commands PRD](./slash-commands.md), [TUI Executor PRD](./tui-executor.md)

---

## Appendix

### Research & Validation

**User Interviews (Pre-Development):**
- 12 developers interviewed about AI agent transparency needs
- 83% expressed desire to "see what the agent knows"
- 67% reported trust issues with opaque agent behavior
- 92% familiar with slash command pattern from Slack/Discord

**Competitive Analysis:**
- Reviewed 8 AI coding assistants
- None provide scratchpad visibility
- Opportunity for differentiation on transparency

**User Testing (Prototype):**
- 6 users tested mockup with 20 sample notes
- Average time to find specific note: 8 seconds
- 100% successfully navigated list → detail → filter
- Feedback: "This is exactly what I needed to trust the agent"

### Design Artifacts

**Mockups:**
- [List View Mockup](../mockups/notes-list-view.png)
- [Detail View Mockup](../mockups/notes-detail-view.png)
- [Filter States Mockup](../mockups/notes-filter-states.png)

**Prototype:**
- Interactive Figma prototype: [Link]

**User Flows:**
- Verification workflow diagram
- Debugging workflow diagram
- Session resumption workflow diagram

---

**Document Version:** 1.1  
**Last Updated:** January 2, 2025  
**Change Summary:** Promoted search to P0 feature, clarified default filter state (show all), simplified note update behavior (no real-time refresh in MVP)  
**Next Review:** After MVP implementation and first user feedback cycle
