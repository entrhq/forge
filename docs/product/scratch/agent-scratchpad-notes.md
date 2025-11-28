# Agent Scratchpad & Notes System

**Status:** Draft / Ideas Stage  
**Category:** Agent Core Capability  
**Priority:** High Impact, Near-Term  
**Last Updated:** December 2024

---

## 🎯 Overview

Provide Forge with a working memory system (scratchpad) within a session to maintain context across multiple tool calls, especially when context gets compressed or summarized. This helps the agent track discoveries, decisions, and important information without repeatedly re-reading files or re-searching.

---

## 🔥 Problem Statement

Currently, when the agent makes multiple tool calls during a complex task:
- Important discoveries (file locations, function names, patterns) can be lost between calls
- No way to track decisions made during exploration
- Context compression/summarization may lose critical details
- Agent must re-search or re-read to find previously discovered information
- No mechanism to maintain a "working set" of relevant information

This leads to inefficiency, repeated work, and potential inconsistency in decision-making.

---

## 💡 Key Capabilities

### Core Note Operations
1. **Add Note** - Store a discrete piece of information with tags
2. **Search Notes** - Find notes by content substring/keyword or tags
3. **List Notes** - View recent notes (limited to prevent context pollution)
4. **Update Note** - Modify existing note content or tags
5. **Delete Note** - Remove notes that are no longer relevant
6. **Scratch Note** - Mark notes as scratched out/resolved when no longer needed
7. **List Tags** - See all tags in use across notes

### Design Constraints
- **Concept-First**: Each note should represent ONE discrete idea/fact/decision
- **Character Limit**: Maximum 800 characters per note (backup constraint)
- **Required Tags**: Minimum 1, maximum 5 tags per note
- **Unlimited Notes**: No hard cap on total notes (for now)
- **Session-Scoped**: Notes only exist for current session, not persisted across sessions

---

## 📊 Data Structure

### Note Object
```json
{
  "id": "note_1234567890",
  "content": "Login function located at src/auth/login.go:45-67. Uses JWT tokens with 24hr expiry. TODO: Check token refresh logic.",
  "tags": ["location", "auth", "todo"],
  "created_at": "2024-12-19T10:30:00Z",
  "updated_at": "2024-12-19T10:30:00Z",
  "scratched": false
}
```

### Field Specifications

**id** (string, auto-generated)
- Unique identifier for the note
- Format: `note_` + timestamp in milliseconds
- Immutable after creation

**content** (string, required)
- The actual note text
- Maximum 800 characters
- Should represent ONE concept/idea/fact
- Can include file paths, line numbers, decisions, observations

**tags** (array of strings, required)
- Minimum: 1 tag
- Maximum: 5 tags
- Suggested tag categories:
  - **Type**: `location`, `decision`, `todo`, `bug`, `pattern`, `dependency`
  - **Domain**: `auth`, `api`, `database`, `ui`, `config`, `test`
  - **Priority**: `critical`, `important`, `minor`
  - **Status**: `active`, `blocked`, `pending`

**created_at** (ISO 8601 timestamp, auto-generated)
- When the note was first created
- Immutable after creation

**updated_at** (ISO 8601 timestamp, auto-generated)
- Last modification time
- Updated on content or tag changes

**scratched** (boolean, default: false)
- Whether the note has been scratched out (no longer needed/resolved)
- Can be toggled via scratch_note
- Scratched notes still searchable but can be filtered out

---

## 🛠 Tool Specifications

### 1. add_note
Create a new note in the scratchpad.

**Parameters:**
- `content` (string, required): Note content (max 800 chars)
- `tags` (array of strings, required): 1-5 tags

**Returns:**
- Note ID and confirmation

**Example:**
```xml
<tool>
<tool_name>add_note</tool_name>
<arguments>
  <content>Found authentication middleware in src/middleware/auth.go:23. Uses custom JWT validation. Decision: Keep existing approach for now.</content>
  <tags>
    <tag>location</tag>
    <tag>auth</tag>
    <tag>decision</tag>
  </tags>
</arguments>
</tool>
```

---

### 2. search_notes
Search notes by content substring/keyword or filter by tags.

**Parameters:**
- `query` (string, optional): Substring to search in note content
- `tags` (array of strings, optional): Filter by specific tags (AND logic)
- `include_scratched` (boolean, default: true): Whether to include scratched notes

**Returns:**
- List of matching notes with full details
- Ordered by relevance, then recency

**Example:**
```xml
<tool>
<tool_name>search_notes</tool_name>
<arguments>
  <query>authentication</query>
  <tags>
    <tag>location</tag>
  </tags>
</arguments>
</tool>
```

---

### 3. list_notes
List recent notes, with limiting to prevent context pollution.

**Parameters:**
- `limit` (integer, default: 10): Maximum number of notes to return
- `tags` (array of strings, optional): Filter by specific tags
- `include_scratched` (boolean, default: false): Whether to include scratched notes

**Returns:**
- List of notes, most recent first
- Limited to specified count

**Behavior:**
- Always favors most recent notes
- Encourages fine-grained searching rather than dumping all notes
- Default limit of 10 keeps context manageable

**Example:**
```xml
<tool>
<tool_name>list_notes</tool_name>
<arguments>
  <limit>5</limit>
  <tags>
    <tag>todo</tag>
  </tags>
</arguments>
</tool>
```

---

### 4. update_note
Modify an existing note's content or tags.

**Parameters:**
- `id` (string, required): Note ID to update
- `content` (string, optional): New content (max 800 chars)
- `tags` (array of strings, optional): New tags (1-5)

**Returns:**
- Updated note object

**Note:** At least one of content or tags must be provided

**Example:**
```xml
<tool>
<tool_name>update_note</tool_name>
<arguments>
  <id>note_1234567890</id>
  <content>Authentication middleware updated to use refresh tokens. Implementation complete.</content>
</arguments>
</tool>
```

---

### 5. delete_note
Remove a note from the scratchpad.

**Parameters:**
- `id` (string, required): Note ID to delete

**Returns:**
- Confirmation of deletion

**Example:**
```xml
<tool>
<tool_name>delete_note</tool_name>
<arguments>
  <id>note_1234567890</id>
</arguments>
</tool>
```

---

### 6. scratch_note
Scratch out a note (mark as no longer needed/resolved).

**Parameters:**
- `id` (string, required): Note ID to scratch
- `scratched` (boolean, required): New scratched status

**Returns:**
- Updated note object

**Example:**
```xml
<tool>
<tool_name>scratch_note</tool_name>
<arguments>
  <id>note_1234567890</id>
  <scratched>true</scratched>
</arguments>
</tool>
```

---

### 7. list_tags
Get all unique tags currently in use across all notes.

**Parameters:**
- None

**Returns:**
- Array of unique tag strings with usage counts

**Example:**
```xml
<tool>
<tool_name>list_tags</tool_name>
<arguments>
</arguments>
</tool>
```

**Example Response:**
```json
{
  "tags": [
    {"tag": "location", "count": 15},
    {"tag": "todo", "count": 8},
    {"tag": "auth", "count": 6},
    {"tag": "decision", "count": 4}
  ]
}
```

---

## 💼 Use Cases

### Use Case 1: Understanding Cross-Component Dependencies
**Scenario:** Agent is exploring a large codebase to understand authentication flow

**Notes Created:**
```
Note 1: "Auth flow pattern: Login validates credentials, then JWT middleware checks 
all protected routes, then permission service authorizes specific actions. Breaking 
this chain causes security bypass - all 3 layers must be present."
Tags: [pattern, auth, security]

Note 2: "Decision: Keep JWT approach, but add refresh token support. Access tokens: 
15min, refresh: 7 days. Chose over sessions for microservice scaling."
Tags: [decision, auth, architecture]
```

Later, when implementing changes, agent searches: `search_notes(query="auth", tags=["decision"])` to recall what was decided and why.

---

### Use Case 2: Multi-Step Refactoring
**Scenario:** Agent needs to refactor error handling across multiple files

**Workflow:**
1. Add note for refactoring pattern/approach
2. Track progress through complex migration
3. Search incomplete notes to see what's left
4. Delete notes once entire refactoring is verified

**Notes:**
```
Note 1: "Error refactoring approach: Replacing string errors with structured Error 
type (code, message, context). Pattern: wrap existing errors with errors.Wrap(), 
add error codes at API boundary only. Affects all API handlers - ~15 files."
Tags: [pattern, refactor, approach]

Note 2: "Error migration progress: Completed users & posts APIs. Still need: 
comments, media, notifications. Each needs code + wrap pattern + test updates."
Tags: [todo, refactor, progress]
Scratched: true (after all migrations complete)
```

---

### Use Case 3: Bug Investigation
**Scenario:** Agent is debugging a complex issue across multiple subsystems

**Notes:**
```
Note 1: "Bug root cause: Password reset doesn't invalidate existing sessions, allowing 
old session tokens to remain valid after reset. This is a security issue - attacker 
with stolen session can maintain access even after victim resets password."
Tags: [bug, auth, security, root-cause]

Note 2: "Fix approach: Add session.InvalidateAllForUser() call in password reset flow. 
Also need to add integration test covering this scenario. Trade-off: logs out user 
from all devices, but that's correct security behavior."
Tags: [decision, auth, security, fix]
```

---

## 🎨 Example Session Flow

```
Agent: Receives task to add 2FA support

[Exploration Phase - Understanding Current State]
→ search_files for "authentication" patterns
→ read_file on key auth files to understand flow
→ add_note: "Current auth uses JWT-only flow. Login → JWT generation → middleware 
  validation. No MFA support. Decision point: Add 2FA as optional or mandatory?"
  Tags: [baseline, auth, architecture]

[Planning Phase - Decisions & Approach]
→ add_note: "2FA Implementation Plan: 1) Add TOTP support with google/go-totp library,
  2) Make it optional per-user with enrollment flow, 3) Add backup codes for recovery,
  4) Update login flow to check 2FA status after password validation. Trade-off: 
  Complexity vs security - going with optional to avoid breaking existing users."
  Tags: [decision, plan, 2fa, architecture]

→ add_note: "Schema changes needed: users table gets 'totp_secret' and 'totp_enabled' 
  fields, new 'backup_codes' table for recovery codes. Migration must handle existing 
  users (default: 2FA disabled)."
  Tags: [decision, database, migration]

[Implementation Phase - Executing Plan]
→ Implements database migration
→ Implements TOTP enrollment endpoint
→ Updates login flow with 2FA check
→ list_notes with tags=["decision"] to verify implementation matches planned approach

[Completion Phase]
→ scratch_note for implementation plan (work complete)
→ Keep architecture decision notes for future reference
→ add_note: "2FA implementation complete. Note for future: Consider adding WebAuthn 
  as alternative to TOTP for better UX. Current approach works but TOTP apps can be 
  friction point for non-technical users."
  Tags: [tech-debt, future, 2fa, ux]
```

---

## ✅ Value Propositions

### For Agent Efficiency
- **Decision Tracking**: Remember why certain approaches were chosen, preventing contradictory changes
- **Better Planning**: Track progress through complex multi-step tasks
- **Context Preservation**: Keep critical insights even when context is compressed
- **Pattern Recognition**: Document cross-component relationships and non-obvious dependencies

### For Code Quality
- **Consistency**: Refer back to architectural decisions and trade-offs made earlier
- **Completeness**: Track complex workflows and migration steps across multiple files
- **Reduced Rework**: Avoid re-analyzing problems already solved in the session

### For User Experience
- **Transparency**: User can see what the agent is tracking and remembering
- **Reliability**: Less likely to forget or contradict earlier decisions
- **Explainability**: Notes provide audit trail of agent's thinking and approach

---

## 🚧 Open Questions

1. **Storage Mechanism**: In-memory only? Persist to temp file for crash recovery?
2. **Note Limits**: Should we add a max total notes limit? Auto-prune old notes?
3. **UI Display**: How should notes be displayed to users in TUI?
4. **Context Impact**: How do notes affect token usage in agent loop?
5. **Smart Suggestions**: Should system suggest tags based on content?
6. **Note Relationships**: Should notes be able to reference other notes?
7. **Export**: Should users be able to export notes at end of session?

---

## 🔗 Related Features

- **Context Management** - Notes complement existing context compression
- **Memory System** - Long-term vs short-term memory boundary
- **Agent Loop Architecture** - Notes persist across loop iterations
- **Tool Approval System** - Notes might need UI for user visibility

---

## 🎯 Success Metrics

- Reduction in repeated file reads for same information
- Fewer instances of contradictory decisions in same session
- Agent task completion with fewer total tool calls
- User reports of more consistent and logical agent behavior

---

## Next Steps

1. **Finalize Data Structure**: Review and approve note schema
2. **Create PRD**: Promote to full PRD with implementation details
3. **Prototype**: Build minimal implementation with add/search/list
4. **Test with Complex Tasks**: Validate utility in real refactoring scenarios
5. **Iterate on Limits**: Adjust character/tag limits based on usage
6. **UI Integration**: Design how notes appear in TUI

---

## 📋 Usage Guidelines

### When to Use Scratchpad Notes

The scratchpad is for **important, longer-living context** that needs to persist across multiple tool calls within a session. Use it judiciously.

### ✅ Good Use Cases

**1. Cross-File Dependencies & Patterns**
```
"Payment flow: checkout.ts calls stripe.ts which triggers webhook.ts. 
Must update all 3 if changing payment provider."
Tags: [pattern, payments, dependency]
```
- Why: Captures relationships that aren't obvious from individual files
- Not searchable: The connection between these files requires context

**2. Important Decisions & Trade-offs**
```
"Decision: Using optimistic locking instead of pessimistic for user updates.
Trade-off: Better performance but requires conflict resolution UI."
Tags: [decision, architecture, users]
```
- Why: Explains rationale that isn't in the code
- Prevents contradictory changes later in the session

**3. Multi-Step Task Tracking**
```
"Migration plan: 1) Add new column 2) Backfill data 3) Update code 4) Remove old column.
Currently on step 2."
Tags: [todo, migration, database]
```
- Why: Tracks progress on complex tasks spanning multiple tool calls
- Helps resume work if context gets compressed

**4. Known Issues & Workarounds**
```
"Bug in test suite: TestUserAuth fails intermittently due to race condition.
Workaround: Added 100ms sleep in setup. TODO: Fix properly with sync primitives."
Tags: [bug, testing, workaround]
```
- Why: Prevents wasted time re-discovering the same issue
- Tracks technical debt to address later

**5. Configuration & Environment Context**
```
"Project uses custom build script (scripts/build.sh) instead of standard 'go build'.
Must run with BUILD_ENV=production for release builds."
Tags: [config, build, environment]
```
- Why: Non-standard workflows that aren't immediately obvious
- Prevents errors from incorrect build commands

### ❌ Bad Use Cases (Don't Do This)

**1. Function Locations** ❌
```
"Login function is at src/auth/login.go:45-67"
Tags: [location, auth]
```
- Why useless: Can search for "func.*login" or "login.go" instantly
- Pollutes notes with easily discoverable information
- Creates maintenance burden if function moves

**2. Simple Variable Names** ❌
```
"User ID field is called 'userId' in the database"
Tags: [database, schema]
```
- Why useless: Can search for "userId" or read schema directly
- Too granular for note-taking

**3. Standard Patterns** ❌
```
"Project uses Express.js for routing"
Tags: [framework, routing]
```
- Why useless: Obvious from package.json or imports
- Search for "express" or "require.*express" works fine

**4. Temporary Tool Results** ❌
```
"Test suite currently has 45 tests, 3 failing"
Tags: [testing, status]
```
- Why useless: Changes constantly, can re-run tests anytime
- Stale immediately after next code change

**5. Single-Use Information** ❌
```
"User model defined in src/models/user.ts"
Tags: [location, models]
```
- Why useless: Only needed once during current exploration
- Search finds it immediately if needed again

### Decision Framework

Before creating a note, ask:

1. **Is this easily searchable?** → Don't note it
   - File locations: Use search_files
   - Function definitions: Use search_files or read_file
   - Import statements: Search for them
   - Configuration values: Read the config file

2. **Will I need this across multiple tool calls?** → Maybe note it
   - If only for next 1-2 calls: Skip it
   - If for complex multi-step task: Note it

3. **Does this represent a relationship or decision?** → Note it
   - Dependencies between components
   - Trade-offs and rationale
   - Non-obvious patterns
   - Important constraints

4. **Would losing this context require significant re-work?** → Note it
   - Understanding built up over many tool calls
   - Decisions that prevent backtracking
   - Complex workflows or sequences

### Examples: Good vs Bad

**Scenario: Adding authentication to an API**

❌ **Bad Notes:**
```
"Login endpoint at /api/auth/login"  (searchable)
"Uses bcrypt for passwords"  (searchable in imports)
"Token expiry is 24 hours"  (in config file)
```

✅ **Good Notes:**
```
"Auth decision: Using JWT with refresh tokens. Access token: 15min, 
refresh: 7 days. Chose this over sessions for better scaling with 
microservices. Must implement token rotation to prevent security issues."
Tags: [decision, auth, security]

"Auth middleware chain: validateToken → checkPermissions → rateLimiter.
Order matters: rate limiting must be last to prevent bypass attempts."
Tags: [pattern, auth, security]
```

**Scenario: Debugging a failing test**

❌ **Bad Notes:**
```
"Test file is at tests/user.test.ts"  (searchable)
"Test uses Jest framework"  (obvious from imports)
"Currently 2 tests failing"  (temporary state)
```

✅ **Good Notes:**
```
"Test failure root cause: Mock database doesn't reset between tests,
causing state pollution. Fixed by adding beforeEach() hook. Pattern
applies to ALL database tests - need to audit rest of suite."
Tags: [bug, testing, pattern, todo]
```

### Tag Strategy

Use tags to make notes **discoverable by intent**:

- **Type tags**: `decision`, `todo`, `bug`, `pattern`, `dependency`
- **Domain tags**: `auth`, `database`, `api`, `frontend`
- **Priority tags**: `critical`, `blocking`, `tech-debt`

Avoid over-tagging:
- Don't tag obvious things (if note is about auth, it'll contain "auth")
- Don't tag file types unless meaningful (avoid `typescript`, `golang`)
- Max 3-4 tags per note is usually enough

### Maintenance Best Practices

**Scratch notes** when resolved:
```xml
<tool>
<tool_name>scratch_note</tool_name>
<arguments>
  <id>note_123</id>
  <scratched>true</scratched>
</arguments>
</tool>
```

**Delete notes** when:
- Task is completely finished
- Information is now in code/docs
- Note was incorrect or superseded

**Update notes** when:
- New information changes the context
- Decision evolves
- Task progresses

### Summary

**Remember:** The scratchpad is not a cache of file contents or search results. It's for **insights, decisions, and relationships** that you discover through work and don't want to lose.

If you can find it with a single search or file read, don't note it. If losing it means re-doing significant analysis, note it.
