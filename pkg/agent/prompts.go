package agent

import (
	"github.com/entrhq/forge/pkg/agent/prompts"
	"github.com/entrhq/forge/pkg/config"
	customtools "github.com/entrhq/forge/pkg/tools/custom"
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

	// Add available custom tools list
	customToolsList := a.getCustomToolsList()
	if customToolsList != "" {
		builder.WithCustomToolsList(customToolsList)
	}

	// Add browser automation guidance if sessions exist
	browserGuidance := a.getBrowserGuidance()
	if browserGuidance != "" {
		builder.WithBrowserGuidance(browserGuidance)
	}

	return builder.Build()
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

	// Get the list of custom tools
	registry := provider.GetRegistry()
	toolsList := registry.List()

	// Convert to prompts.ToolMetadata interface
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
