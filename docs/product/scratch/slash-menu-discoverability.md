# Slash Command Palette: Discoverability Redesign

## The Problem
Currently, the `/` command palette hardcodes `maxVisible = 5`. If there are 15 available commands, 10 are hidden behind a message saying `... and more. Keep typing to filter.` This forces users to guess or memorize commands, which hurts UX and feature discoverability.

## Goals
1. Increase the number of commands visible without overwhelming the terminal.
2. Provide a clear indication when there are more commands off-screen.
3. Allow users to seamlessly navigate through all commands using the arrow keys (scrolling the list).

---

## Option A: Dynamic Height with Viewport Scrolling (Recommended)

Instead of a hard limit of 5, the palette expands its height to fit the available commands, up to a maximum (e.g., 40% of the terminal height or max 12 items). If the list exceeds this, it becomes a scrollable viewport with a sleek scroll indicator track on the right margin.

### Mockup

```text
┌─ Slash Commands ─────────────────────────────────────────────────────────┐
│  ❯ /clear       Clear the current chat session and start fresh         █ │
│    /commit      Generate a commit message and commit changes           ║ │
│    /help        Show detailed help and keyboard shortcuts              ║ │
│    /history     Show recent chat history                               ║ │
│    /pin         Pin a file to context permanently                      ║ │
│    /unpin       Remove a file from pinned context                      ║ │
│    /pr          Review branch and create a Pull Request                ║ │
│    /model       Switch the active LLM model                            ║ │
│    /settings    Open the settings and configuration menu               ║ │
│    /notes       Open the scratchpad notes workspace                    ░ │
└──────────────────────────────────────────────────────────────────────────┘
```

**How it works:**
*   The `maxVisible` becomes dynamic: `min(len(commands), MaxAllowedHeight)`.
*   Pressing `Down` at the bottom of the visible list increments a `scrollOffset`, pushing the list up like a standard viewport list.
*   A minimalistic scrollbar `█`/`║`/`░` on the right edge indicates relative position.
*   **Pros:** Familiar modern command palette feel (like VSCode/Raycast). Fits any screen size.
*   **Cons:** Requires slightly more logic to track `scrollOffset`.

---

## Option B: Categorized / Grouped View

If we plan to add many more commands in the future, a flat list might get too long. We can group commands by domain (e.g., General, Git, Context) and show categorical headers. This breaks the list into easily skimmable chunks.

### Mockup

```text
┌─ Slash Commands ─────────────────────────────────────────────────────────┐
│                                                                          │
│  CORE ────────────────────────────────────────────────────────────────── │
│  ❯ /clear       Clear the current chat session and start fresh           │
│    /help        Show detailed help and keyboard shortcuts                │
│    /settings    Open the settings and configuration menu                 │
│                                                                          │
│  CONTEXT ─────────────────────────────────────────────────────────────── │
│    /pin         Pin a file to context permanently                        │
│    /unpin       Remove a file from pinned context                        │
│    /notes       Open the scratchpad notes workspace                      │
│                                                                          │
│  GIT ─────────────────────────────────────────────────────────────────── │
│    /commit      Generate a commit message and commit changes             │
│    /pr          Review branch and create a Pull Request                  │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**How it works:**
*   Slash commands are augmented with a `Category` field in their registry.
*   The UI groups and sorts them automatically.
*   **Pros:** High discoverability, visually structured and extremely organized.
*   **Cons:** Taller UI footprint.

---

## Technical Implementation Path

If we proceed with **Option A (Scrollable)**, the technical changes needed in `pkg/executor/tui/overlay/palette.go` are:
1.  Add `scrollOffset int` to the `CommandPalette` struct.
2.  Update `navigateUp()` and `navigateDown()` to adjust `scrollOffset` when `selectedIndex` moves beyond the visible bounds (similar to how we fixed the Settings overlay!).
3.  Dynamically calculate `maxVisible` based on `m.height` (the terminal height) rather than hardcoding `5`.
4.  Optionally render a scroll track along the right-hand padding using standard block characters (e.g., `█` and `│` or `║`).
