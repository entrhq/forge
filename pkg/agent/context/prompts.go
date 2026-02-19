package context

// episodicMemorySystemPrompt is the shared system prompt used by all summarization
// strategies that write episodic memory for the agent. It instructs the summarizing
// LLM to write in operational first-person so the output lands in the agent's context
// as recalled experience rather than a third-party report.
const episodicMemorySystemPrompt = "You are writing episodic memory for an AI coding agent. " +
	"Your output will be injected directly into the agent's context window as its own recalled experience. " +
	"Write entirely in operational first-person: declarative statements of completed actions " +
	"('I ran X, result was Y', 'I found Z at path P') — not reflective narrative ('I remember trying', 'I think I'). " +
	"Uncertainty markers are forbidden: never write 'I think', 'I believe', 'I'm not sure'. " +
	"Be dense, exact, and technical. " +
	"Preserve every concrete artifact (file paths, function names, error strings, command output, line numbers). " +
	"Omit XML markup, role labels, conversational filler, and hedging language."
