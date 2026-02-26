# TUI Input Box Alternatives

Focused exploration of input area styles. Each mockup shows only the bottom portion of the terminal (the last ~6 lines) where the input box lives, plus the line of key hints. All shown at 80 columns.

`[salmon]` = coral/brand colour · `[muted]` = low-contrast grey · `[dim]` = terminal dim · `[bold]` = bold weight

---

## A — Underline Only

No box at all. A single bottom underline marks the input boundary. Clean and airy. The cursor is the only real indicator that this is an input field.

```
    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added


  ❯ _
  ──────────────────────────────────────────────────────────────────────────────
  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

**Colour key:**  
`❯` → [salmon]  `──` underline → [muted dim]  
Input text → terminal default  
Hint line → [muted]

---

## B — Top Rule Only

A single hairline above the input zone. Nothing below — the cursor just floats in clear space. Very minimal. The rule is the only affordance.

```
    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added

  ──────────────────────────────────────────────────────────────────────────────
  ❯ _

  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

**Colour key:**  
`──` rule → [muted dim]  `❯` → [salmon]  
Hint line → [muted]

---

## C — Left Bar (Active Indicator)

A vertical bar on the left edge of the input area, coloured in brand colour. No horizontal lines. The bar signals "this is where you type" and has the same visual rhythm as the `│` used in diff views and borders elsewhere.

```
    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added


  ▎  _
  ▎
  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

When multiline:
```
  ▎  Here is a longer message that wraps
  ▎  across two lines because it has enough
  ▎  content to require it
  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

**Colour key:**  
`▎` left bar → [salmon]  
Input text → terminal default  
Hint line → [muted]

---

## D — Prompt Line

No box, no border. A `❯` prompt prefix on the same line as the input, like a shell prompt. The distinction between "conversation" and "input" is purely positional — you type right after the prompt. Feels native to terminal users.

```
    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added


──────────────────────────────────────────────────────────────────────────────
❯  _
```

**Colour key:**  
`──` separator → [muted dim]  `❯` → [salmon bold]  
No hint line — hints available via `?` command instead

---

## E — Labelled Pill

A small `you ›` label badge floats to the left of the input area. Clear role attribution without a full box. The label is the only decoration.

```
    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added


  you ›  _


  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

**Colour key:**  
`you` → [salmon]  `›` → [muted]  
Input text → terminal default  
Hint line → [muted]

---

## F — Inline Rule with Label

The separator line contains the label text inline, like a section heading. The input sits in clear space below it. Low-chrome but clearly demarcated. Works well combined with Option 10 (Zen Minimal) from the main mockups.

```
    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added

  ─────────────────────────────────── message ────────────────────────────────
  _

  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

When the agent is working, the label text changes:
```
  ────────────────────────────────── working ─────────────────────────────────
  _    (input disabled, dimmed)
```

**Colour key:**  
`────` rule → [muted dim]  `message` label → [muted]  
Label changes to `working` → [salmon dim] when agent is busy  
Hint line → [muted]

---

## G — Corner-Only Box

Like a rounded box but only the four corners are drawn — the sides are empty space. Gives the visual suggestion of a container without the heavy weight of full borders. Rare in TUIs, distinctive.

```
    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added

  ╭                                                                          ╮
    _

  ╰                                                                          ╯
  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

**Colour key:**  
`╭ ╮ ╰ ╯` corners → [salmon]  
Input text → terminal default  
Hint line → [muted]

---

## H — Bottom Bar with Embedded Input

The input and the hints are merged into a single band at the very bottom. The hints wrap to the right of the prompt on the same line when input is short, or drop below when it's long. Saves vertical space.

Single-line input (short message):
```
    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added


  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░
  ❯  _                                     Enter · send   / · cmds   Ctrl+C
```

Multiline input:
```
  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░
  ❯  Here is a longer message that goes across
     multiple lines and takes up more space
  ──────────────────────────────────────────────
  Enter · send   Alt+Enter · new line   / · cmds
```

**Colour key:**  
`░░░` input band → very subtle background tint (`#111827`)  
`❯` → [salmon]  Hint text → [muted dim]

---

## I — Focus Ring (Accent Bottom Border Only)

A full box — but with three neutral sides and one strongly accented bottom border, like a "focus ring" in web UI. The bottom edge is the only brand-coloured element. Everything else is either invisible or very subtle.

```
    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added

  ┌──────────────────────────────────────────────────────────────────────────┐
  │  _                                                                       │
  ┝━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┥
  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

Unfocused (when an overlay is open):
```
  ┌──────────────────────────────────────────────────────────────────────────┐
  │  _                                                                       │
  └──────────────────────────────────────────────────────────────────────────┘
  (all borders [dim], bottom loses accent)
```

**Colour key:**  
Top and side borders `┌─┐│` → [muted dim]  
Bottom accent `┝━━━━┥` → [salmon]  
Input text → terminal default

---

## J — Ghost / Invisible

No visible container at all — not even a rule. The placeholder text "What would you like to do?" is the only affordance. Once you start typing it disappears. The hierarchy comes entirely from vertical rhythm and spacing. Works best with a clear visual separator (header or rule) above the conversation to distinguish content zones.

```
    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added



  What would you like to do?


  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

When typing:
```
    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added



  Now add the same error handling to the auth client_



  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

**Colour key:**  
Placeholder text → [muted italic]  
Input text → terminal default (no decoration)  
Hint line → [muted dim]

---

## K — Double Rule Trough

Two thin rules form a trough. The input sits between them. No side borders. Creates a defined channel without the enclosed feeling of a box.

```
    The client has no sentinel errors. I'll add fmt.Errorf wrapping
    to all return paths and export ErrNotFound, ErrUnauthorized.

    ✎  write_file  pkg/api/client.go  ·  3 sentinel errors added

  ──────────────────────────────────────────────────────────────────────────────
   ❯  _
  ──────────────────────────────────────────────────────────────────────────────
  Enter · send   Alt+Enter · new line   / · commands   Ctrl+C
```

**Colour key:**  
Top rule → [muted dim]  Bottom rule → [salmon dim] (slightly more visible = focused)  
`❯` → [salmon]

---

## Quick Comparison

| Option | Has border | Feels like | Vertical cost | Distinctiveness |
|---|:---:|---|:---:|:---:|
| A · Underline only | partial | Markdown editor | Low | Low |
| B · Top rule only | partial | Shell / prose editor | Low | Low |
| C · Left bar | none | Code editor gutter | Very low | High |
| D · Prompt line | none | Shell (zsh/fish) | Very low | Medium |
| E · Labelled pill | none | Chat app | Low | Medium |
| F · Inline rule label | partial | Document section | Low | Medium |
| G · Corner only | none (corners) | Unique / decorative | Medium | Very high |
| H · Embedded hint bar | band | IDE status bar | Very low | High |
| I · Focus ring | full (accent bottom) | Web form input | Medium | High |
| J · Ghost / invisible | none | Notion / prose | Very low | Low |
| K · Double rule trough | partial | Classic dialog | Low | Medium |

---

## Notes on Multiline Behaviour

Whichever style is chosen, when input grows to 2+ lines:

- **B, C, F, K** expand naturally — the rules just move further apart
- **A** the underline scrolls with the bottom of the text
- **D** the prompt stays on the first line; subsequent lines are indented 3 chars to align
- **E** the `you ›` label stays on the first line; overflow wraps flush
- **G** corners stay fixed; the interior text wraps freely
- **H** the band expands upward and the hint line drops below when space is needed
- **I** the box expands; the accent bottom border always stays at the true bottom
- **J** no constraints; text just expands upward into the whitespace buffer
