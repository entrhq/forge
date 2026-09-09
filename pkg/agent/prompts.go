package agent

import (
	"sort"

	"github.com/entrhq/forge/pkg/agent/prompts"
	"github.com/entrhq/forge/pkg/config"
	customtools "github.com/entrhq/forge/pkg/tools/custom"
	"github.com/entrhq/forge/pkg/types"
)

// buildSystemPrompt constructs the system prompt as a single string, for
// display and token accounting.
func (a *DefaultAgent) buildSystemPrompt() string {
	return a.newPromptBuilder().Build()
}

// buildSystemMessages constructs the system prompt as stability-tagged
// messages, the form sent to the provider on each iteration.
func (a *DefaultAgent) buildSystemMessages() []*types.Message {
	return a.newPromptBuilder().BuildSegments()
}

// newPromptBuilder assembles a prompt builder from the agent's current tools,
// instructions, repository context, custom tools, and browser state.
func (a *DefaultAgent) newPromptBuilder() *prompts.PromptBuilder {
	builder := prompts.NewPromptBuilder().
		WithTools(a.getToolsList())

	if a.customInstructions != "" {
		builder.WithCustomInstructions(a.customInstructions)
	}

	if a.repositoryContext != "" {
		builder.WithRepositoryContext(a.repositoryContext)
	}

	if customToolsList := a.getCustomToolsList(); customToolsList != "" {
		builder.WithCustomToolsList(customToolsList)
	}

	if browserGuidance := a.getBrowserGuidance(); browserGuidance != "" {
		builder.WithBrowserGuidance(browserGuidance)
	}

	return builder
}

// getCustomToolsList builds a formatted list of available custom tools
func (a *DefaultAgent) getCustomToolsList() string {
	// Get the run_custom_tool instance
	a.toolsMu.RLock()
	tool, exists := a.tools["run_custom_tool"]
	a.toolsMu.RUnlock()

	if !exists {
		return ""
	}

	// Type assert to access the registry
	type registryProvider interface {
		GetRegistry() *customtools.Registry
	}

	provider, ok := tool.(registryProvider)
	if !ok {
		return ""
	}

	// The registry lists from a map; sort so the prompt text is identical
	// across requests when the set of custom tools has not changed.
	toolsList := provider.GetRegistry().List()
	sort.Slice(toolsList, func(i, j int) bool {
		return toolsList[i].GetName() < toolsList[j].GetName()
	})

	metadataList := make([]prompts.ToolMetadata, len(toolsList))
	for i, t := range toolsList {
		metadataList[i] = t
	}

	return prompts.FormatCustomToolsList(metadataList)
}

// getBrowserGuidance returns browser automation guidance if browser tools are active
func (a *DefaultAgent) getBrowserGuidance() string {
	// Check if browser is enabled in config
	if !config.IsInitialized() {
		return ""
	}
	ui := config.GetUI()
	if ui == nil || !ui.IsBrowserEnabled() {
		return ""
	}

	// Get browser manager if available and check for active sessions
	if a.browserManager == nil || !a.browserManager.HasSessions() {
		return ""
	}

	return prompts.BrowserUsePrompt
}
