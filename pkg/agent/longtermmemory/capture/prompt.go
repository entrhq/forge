package capture

import (
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/agent/longtermmemory"
)

// classifierSystemPrompt instructs the classifier LLM.
// It sets the memory-worthiness threshold, scope/category rules, relationship
// edge semantics, and output format requirements.
const classifierSystemPrompt = `You are a memory classifier for an AI coding assistant called Forge.
Your job is to read a window of conversation and identify information that is
worth remembering permanently across sessions.

MEMORY-WORTHINESS CRITERIA (all must be met):
- The information represents a durable preference, convention, decision, or correction
- It is not transient (e.g. "run this command once" is not worth remembering)
- It is not already obvious from the codebase itself
- It is not sensitive (never capture credentials, tokens, passwords, or personal identifiers)

SCOPE ASSIGNMENT:
- scope: "repo" — information specific to the current project (conventions, architecture, patterns)
- scope: "user" — information about how this user works across all projects (preferences, style, habits)

CATEGORY ASSIGNMENT:
- coding-preferences — style, idiom, tooling choices
- project-conventions — repo-specific patterns and standards
- architectural-decisions — design decisions with rationale
- user-facts — facts about how the user works or thinks
- corrections — mistakes the agent made and was corrected on
- patterns — non-obvious relationships or recurring structures

RELATIONSHIP EDGES:
If a new memory refines, contradicts, or supersedes an existing memory listed in
EXISTING MEMORIES below, include the relationship in the "related" or "supersedes"
field using the exact ID from that list. Do not invent IDs.

The "relationship" field in related entries MUST be one of these exact strings:
- "refines"      — this memory adds detail or precision to the related one
- "contradicts"  — this memory conflicts with or corrects the related one
- "relates-to"   — this memory is topically connected but neither refines nor contradicts
- "supersedes"   — this memory fully replaces the related one (prefer the top-level supersedes field instead)

OUTPUT FORMAT:
Return a JSON array of memory objects. Return an empty array [] if nothing is memory-worthy.
Each object must have: content (markdown string), scope, category.
Optional fields: supersedes (memory ID string), related (array of {id, relationship}).

Example:
[
  {
    "content": "User prefers ` + "`" + `errors.As` + "`" + ` over type assertions for error handling in all Go code.",
    "scope": "user",
    "category": "coding-preferences"
  }
]`

// buildClassifierPrompt constructs the user-turn prompt for the classifier LLM.
// It formats the stripped conversation messages and, if existing memories are
// provided, appends an EXISTING MEMORIES section with compact one-line summaries
// so the classifier can reference real memory IDs in supersedes/related fields.
func buildClassifierPrompt(event TriggerEvent, existing []*longtermmemory.MemoryFile) string {
	var b strings.Builder

	b.WriteString("CONVERSATION WINDOW\n")
	b.WriteString("(tool call content has been removed; only user and assistant prose is shown)\n\n")

	for _, msg := range event.Messages {
		fmt.Fprintf(&b, "[%s]: %s\n\n", msg.Role, msg.Content)
	}

	if len(existing) > 0 {
		b.WriteString("---\n\nEXISTING MEMORIES\n")
		b.WriteString("Use these IDs verbatim when populating supersedes or related fields.\n\n")
		for _, m := range existing {
			firstLine := firstLine(m.Content)
			fmt.Fprintf(&b, "- [%s] (%s/%s) %s\n", m.Meta.ID, m.Meta.Scope, m.Meta.Category, firstLine)
		}
		b.WriteString("\n")
	}

	b.WriteString("---\n\n")
	b.WriteString("Analyze the conversation window above and return a JSON array of memory objects. ")
	b.WriteString("Return [] if nothing meets the memory-worthiness criteria.")

	return b.String()
}

// firstLine returns the first non-empty line of s, trimmed of whitespace.
// Used to produce compact one-line summaries of existing memory content for the prompt.
func firstLine(s string) string {
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
