# TUI Design Mockups

Ten distinct design options for the Forge TUI. Each mockup uses an 80-column × 24-line reference terminal. All use the same scenario: a user asked to add error handling to the API client; the agent read a file, then wrote a fix.

**Input box decision:** All options use **Option B — top rule only**. A single hairline rule above the input; the cursor floats in clear space below it. No enclosing box.

**Color annotations** used throughout:
- `[salmon]` — coral/salmon pink, the Forge brand colour (#FFB3BA)  
- `[mint]` — mint green, success/secondary (#A8E6CF)
- `[muted]` — muted grey, secondary text (#6B7280)
- `[dim]` — terminal dim attribute, low contrast
- `[bold]` — bold weight

---

## Option 1 — Minimal Chrome

**Philosophy:** Get out of the way entirely. One thin header line, one thin footer line. Maximum vertical space for conversation. The brand is present but quiet.

- Single-line header: glyph + name left, cwd centre, version right
- Flat message rendering — no boxes around messages, just indentation
- Tool calls shown as compact single-line entries with status icons
- Hint line below input — key hints only

```
⬡ forge  ·  ~/projects/myapp  ·  claude-3-5-sonnet                     v0.4.2
────────────────────────────────────────────────────────────────────────────────

 ❯  Add error handling to the API client

    I'll start by reading the API client to understand the existing
    structure before adding error handling.

    ▶  read_file  pkg/api/client.go
    ✓  read_file  234 lines

    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added


  ──────────────────────────────────────────────────────────────────────────────
  ❯ _

  Enter · send   Alt+Enter · new line   / · commands   Ctrl+Y · copy   Ctrl+C
```

**Colour key:**  
`⬡ forge` → [salmon bold]  `·` separators → [muted]  `v0.4.2` → [muted]  
`❯` prompt → [salmon]  `▶` → [muted]  `✓` → [mint]  `✎` → [muted]  
`────` header divider → [dim]  `──` input rule → [muted dim]  
Hint line → [muted]

---

## Option 2 — Two-Line Brand

**Philosophy:** Give the brand a proper home without wasting vertical space. Two dedicated header lines, then a hard separator. The second header line shows live session stats so the hint line stays minimal.

- Line 1: logo + product name prominent in brand colour
- Line 2: model name, context usage bar, session duration, cwd — all [muted]
- Double-line border header creates a clear visual zone
- Hint line: key hints only

```
╔══════════════════════════════════════════════════════════════════════════════╗
║  ⬡ FORGE                                                              v0.4.2 ║
║  claude-3-5-sonnet  ·  ctx ████░░░░ 12k/128k  ·  00:04:32  ·  ~/projects/app ║
╚══════════════════════════════════════════════════════════════════════════════╝

 ❯  Add error handling to the API client

    I'll start by reading the API client to understand the existing
    structure before adding error handling.

    ▶  read_file  pkg/api/client.go
    ✓  read_file  234 lines

    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added

  ──────────────────────────────────────────────────────────────────────────────
  ❯ _

  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

**Colour key:**  
`║  ⬡ FORGE` → [salmon bold]  border `╔═╗║╚╝` → [salmon dim]  
Stats line → [muted]  `████░░░░` → green→yellow→red gradient by %  
`──` input rule → [muted dim]  `❯` → [salmon]  Hint line → [muted]

---

## Option 3 — Invisible Chrome (Zero Header)

**Philosophy:** Remove the header completely. The product name lives as a tiny floating badge in the top-right corner. Every single line is conversation. Best for people who live in the TUI all day and already know what they're using.

- No header bar at all — conversation starts on line 1
- Floating `forge v0.4.2` badge top-right (overlaid, not part of layout)
- Role prefixes in the left margin provide orientation
- Stats embedded in the hint line at the bottom

```
                                                           ⬡ forge v0.4.2  ·

 ❯  Add error handling to the API client

    I'll start by reading the API client to understand the existing
    structure before adding error handling.

    ▶  read_file  pkg/api/client.go
    ✓  read_file  234 lines

    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added


  ──────────────────────────────────────────────────────────────────────────────
  ❯ _

  claude-3-5-sonnet  ctx 12k   Enter · send   / · commands   Ctrl+C
```

**Colour key:**  
Badge `⬡ forge v0.4.2` → [muted dim] (very subtle)  
`❯` conversation prefix → [salmon]  `▶` → [dim]  `✓` → [mint]  
`──` input rule → [muted dim]  `❯` input prompt → [salmon]  
Hint line → [muted]

---

## Option 4 — Retro Terminal

**Philosophy:** Lean into the terminal aesthetic. Box-drawing borders everywhere, explicit role labels, a classic "BBS" or "ncurses" feel. No rounded corners. Everything is structured and deliberate.

- Full outer border around the entire UI
- Role labels `[you]` `[forge]` `[tool]` as explicit text labels
- Tool calls in their own bordered inline box
- Input sits after an inner `├──┤` rule; hints appear outside the outer border

```
┌─ ⬡ forge v0.4.2 ──────────────────────────── ~/projects/myapp ─ sonnet ─────┐
│                                                                               │
│  [you]   Add error handling to the API client                                 │
│                                                                               │
│  [forge] I'll start by reading the API client to understand the               │
│          existing structure before adding error handling.                     │
│                                                                               │
│  [tool]  ┌─ read_file ───────────────────────────────────────────────────┐   │
│          │  path: pkg/api/client.go                                       │   │
│          │  result: 234 lines read ✓                                      │   │
│          └───────────────────────────────────────────────────────────────┘   │
│                                                                               │
│  [forge] The client has no sentinel errors. I'll add fmt.Errorf               │
│          wrapping to all return paths.                                        │
│                                                                               │
├───────────────────────────────────────────────────────────────────────────────┤
│  ❯ _                                                                          │
│                                                                               │
└───────────────────────────────────────────────────────────────────────────────┘
Enter: send  Alt+Enter: newline  /: commands  Ctrl+C: exit
```

**Colour key:**  
Outer border → [dim]  Header bar text → [salmon dim]  
`[you]` → [salmon]  `[forge]` → [mint]  `[tool]` → [muted]  
Tool inner border → [muted dim]  `├──┤` input rule → [dim]  
`❯` → [salmon]  Hint line below box → [muted]

---

## Option 5 — Right Sidebar Stats

**Philosophy:** Move all session metadata to a right sidebar column. The left conversation area is clean prose. The right column is a live dashboard. Best for power users who want to monitor token usage continuously.

- Conversation occupies left ~75% of width
- Right ~25% is a fixed stats sidebar separated by `│`
- Sidebar: model, context bar, token counts, session duration, working dir
- Input rule spans only the left (conversation) column width

```
                                              │ ⬡ forge v0.4.2
                                              │ claude-3-5-sonnet
 ❯  Add error handling to the API client      │
                                              │ Context
    I'll start by reading the API client      │ ████░░░░  12k/128k
    to understand the existing structure.     │
                                              │ Tokens
    ▶  read_file  pkg/api/client.go           │  ↑ in   45,231
    ✓  read_file  234 lines                   │  ↓ out  12,844
                                              │  Σ tot  58,075
    The client has no sentinel errors.        │
    I'll add fmt.Errorf wrapping to all       │ Session
    return paths and export ErrNotFound.      │ 00:04:32
                                              │ 8 messages
    ✎  write_file  pkg/api/client.go          │
       3 sentinel errors added                │ ~/projects/myapp
                                              │
  ────────────────────────────────────────────
  ❯ _

  Enter · send   / · commands   Ctrl+C
```

**Colour key:**  
`│` sidebar border → [muted dim]  Sidebar labels → [muted]  Values → [bold]  
Context bar `████` → green→yellow→red  `⬡ forge` → [salmon]  
`──` input rule → [muted dim]  `❯` → [salmon]  Hint line → [muted]

---

## Option 6 — Zone Bands

**Philosophy:** Strong horizontal colour bands create instant visual structure. The header band and the input zone are distinct. Content area is default terminal background — the bands do the heavy lifting.

- Header band: full-width, salmon background, white text — unmistakable brand
- Content area: default terminal background  
- Input rule and floating input below — no band around the input
- Footer line: stats + key hints

```
▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓
  ⬡  FORGE                                                              v 0.4.2
▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓

 ❯  Add error handling to the API client

    I'll start by reading the API client to understand the existing
    structure before adding error handling.

    ▶  read_file  pkg/api/client.go
    ✓  read_file  234 lines

    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added

  ──────────────────────────────────────────────────────────────────────────────
  ❯ _

  claude-3-5-sonnet  ctx 12k   Enter · send   / · commands   Ctrl+C
```

**Colour key:**  
`▓▓▓` header band → salmon/coral background, white foreground  
`⬡ FORGE` → white bold on salmon  `❯` conversation prefix → salmon  
`──` input rule → [muted dim]  `❯` input prompt → [salmon]  
Hint line → [muted]

---

## Option 7 — Chat Bubbles

**Philosophy:** Borrow visual language from chat applications. Each message is in its own rounded box, clearly attributed. Comfortable and familiar to anyone who uses Slack or iMessage. The input is deliberately open and unboxed — a deliberate contrast to the message bubbles above.

- User messages: rounded box with `─ You ─` label
- Agent messages: rounded box with `─ Forge ─` label
- Tool calls: compact inline entry within the agent bubble
- Input: rule + open cursor — contrast with the enclosed message style above

```
⬡ forge  ·  claude-3-5-sonnet                               ctx 12k  ·  v0.4.2
────────────────────────────────────────────────────────────────────────────────

  ╭─ You ────────────────────────────────────────────────────────────────────╮
  │  Add error handling to the API client                                    │
  ╰──────────────────────────────────────────────────────────────────────────╯

  ╭─ Forge ──────────────────────────────────────────────────────────────────╮
  │  I'll start by reading the API client to understand the existing         │
  │  structure before adding error handling.                                 │
  │                                                                          │
  │  ▸ read_file  pkg/api/client.go  ✓ 234 lines                            │
  │                                                                          │
  │  The client has no sentinel errors. I'll add fmt.Errorf wrapping to      │
  │  all return paths and export ErrNotFound, ErrUnauthorized.               │
  │                                                                          │
  │  ✎ write_file  pkg/api/client.go  ·  3 sentinel errors added            │
  ╰──────────────────────────────────────────────────────────────────────────╯

  ──────────────────────────────────────────────────────────────────────────────
  ❯ _

  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

**Colour key:**  
`─ You ─` border → [muted]  `─ Forge ─` border → [salmon]  
`▸` inline tool call → [muted]  `✓` → [mint]  `✎` → [muted]  
`──` input rule → [muted dim]  `❯` → [salmon]  Hint line → [muted]

---

## Option 8 — Split Tool Panel

**Philosophy:** Separate "what the agent said" from "what the agent did". Conversation prose lives in the main viewport. Tool activity lives in a dedicated strip above the input. The open input (no enclosing box) mirrors the open, unframed prose in the conversation.

- Main viewport: prose only (user messages + agent text)
- Tool panel: a slim strip showing tool call history, newest at bottom
- Tool panel framed by labelled dividers
- Input: rule + open cursor below the tool panel

```
⬡ forge  ·  claude-3-5-sonnet                               ctx 12k  ·  v0.4.2
────────────────────────────────────────────────────────────────────────────────

 ❯  Add error handling to the API client

    I'll start by reading the API client to understand the existing
    structure before adding error handling.

    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.


──────────────────────────────── Tool Activity ──────────────────────────────────
  ✓  read_file      pkg/api/client.go                            234 lines
  ✓  write_file     pkg/api/client.go                  3 errors added  ·  done
  ──────────────────────────────────────────────────────────────────────────────
  ❯ _

  Enter · send   / · commands   Ctrl+C
```

**Colour key:**  
`── Tool Activity ──` divider → [muted]  `✓` → [mint]  
Tool names → [bold]  Arguments → [muted]  Results → normal  
Main viewport `❯` → [salmon]  `──` input rule → [muted dim]  
`❯` input prompt → [salmon]  Hint line → [muted]

---

## Option 9 — Dashboard Header

**Philosophy:** Give power users a dense information dashboard in the header. Three clearly delineated columns show identity, agent status, and resource usage at a glance. Below that, clean minimal conversation. The open input is a deliberate contrast to the structured header.

- 3-column bordered header: `[identity] | [agent status] | [resource usage]`
- Below: clean conversation, no clutter
- Hint line: key hints only
- Compact tool call rendering inline

```
┌──────────────────┬─────────────────────────────┬───────────────────────────┐
│  ⬡ forge  v0.4.2 │  claude-3-5-sonnet           │  ctx  ████░░  12k / 128k  │
│  ~/projects/app  │  ●  Agent active              │  ↑ 45k   ↓ 12k   Σ 58k   │
└──────────────────┴─────────────────────────────┴───────────────────────────┘

 ❯  Add error handling to the API client

    I'll start by reading the API client to understand the existing
    structure before adding error handling.

    ▶  read_file  pkg/api/client.go
    ✓  read_file  234 lines

    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added

  ──────────────────────────────────────────────────────────────────────────────
  ❯ _

  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

**Colour key:**  
Header border → [muted dim]  `⬡ forge` → [salmon bold]  
`●  Agent active` → [salmon] when busy, [mint] when idle  
`████░░` context bar → green→yellow→red gradient  `↑ ↓ Σ` → [muted]  
`──` input rule → [muted dim]  `❯` → [salmon]  Hint line → [muted]

---

## Option 10 — Zen Minimal

**Philosophy:** The absolute minimum. No header. No bottom bar. No icons. No borders. Role is indicated by a right-aligned label tag. The rule before the input is the only structural element in the entire UI — making it land with real weight.

- Zero header
- Messages full-width with right-aligned `[you]` / `[forge]` role tags
- Tool calls as single-line indented entries, no icons, just `→` and `✓`
- Stats embedded in the hint line below the input

```


  Add error handling to the API client                                    [you]

  I'll start by reading the API client to understand the existing       [forge]
  structure before adding error handling.

    → read_file  pkg/api/client.go
    ✓  234 lines read

  The client has no sentinel errors. I'll add fmt.Errorf wrapping
  to all return paths and export ErrNotFound, ErrUnauthorized.

    → write_file  pkg/api/client.go
    ✓  3 sentinel errors added



  ──────────────────────────────────────────────────────────────────────────────
  ❯ _

  sonnet · ctx 12k   Enter · send   / · commands   Ctrl+C
```

**Colour key:**  
`[you]` tag → [salmon muted]  `[forge]` tag → [mint muted]  
`→` → [dim]  `✓` → [mint]  
`──` input rule → [muted dim]  `❯` → [salmon]  
Hint line → [muted dim]  Everything else → terminal default

---

## Quick Comparison

| Option | Header lines | Tool display | Message style | Stats location |
|--------|:---:|---|---|---|
| 1 · Minimal Chrome | 1 | Icons inline | Flat indented | Hint line |
| 2 · Two-Line Brand | 2 | Icons inline | Flat indented | Header line 2 |
| 3 · Invisible Chrome | 0 | Icons inline | Flat indented | Hint line + floating badge |
| 4 · Retro Terminal | 1 | Bordered box | Role-labelled | Header |
| 5 · Right Sidebar | 0 | Icons inline | Flat indented | Right sidebar column |
| 6 · Zone Bands | 1 + bands | Icons inline | Flat indented | Hint line |
| 7 · Chat Bubbles | 1 | Inline in bubble | Rounded boxes | Header right |
| 8 · Split Tool Panel | 1 | Dedicated strip | Flat (prose only) | Header right |
| 9 · Dashboard Header | 2 | Icons inline | Flat indented | Header boxes |
| 10 · Zen Minimal | 0 | Indented arrows | Role tags | Hint line |

**Input box:** All options use **Option B — top rule only** (`──` hairline above, cursor floats below).

---

## Mixing and Matching

These aren't mutually exclusive. Some combinations worth noting:

- **Option 1 header + Option 8 tool panel** — clean header, dedicated tool strip, maximum prose clarity
- **Option 2 header + Option 7 bubbles** — branded + chat-app feel, polished consumer aesthetic
- **Option 9 header + Option 10 messages** — power-user stats + distraction-free reading

The core decisions still to make:
1. **Header: none / 1-line / 2-line?** (affects available viewport height most)
2. **Tool calls: inline / separate panel / card?** (affects information density)
3. **Messages: flat / bubbled?** (affects personality / approachability)
