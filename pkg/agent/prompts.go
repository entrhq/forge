package agent

import (
	"github.com/entrhq/forge/pkg/agent/prompts"
)

// buildSystemPrompt constructs the system prompt with tool schemas and custom instructions
func (a *DefaultAgent) buildSystemPrompt() string {
	builder := prompts.NewPromptBuilder().
		WithTools(a.getToolsList())

	// Add user's custom instructions if provided
	if a.customInstructions != "" {
		builder.WithCustomInstructions(a.customInstructions)
	}

	// Add repository context if provided
	if a.repositoryContext != "" {
		builder.WithRepositoryContext(a.repositoryContext)
	}

	return builder.Build()
}
