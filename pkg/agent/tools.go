package agent

import (
	"sort"

	"github.com/entrhq/forge/pkg/agent/tools"
)

// getToolsList returns the currently visible tools sorted by name.
// Tools implementing ConditionallyVisible are omitted when ShouldShow is false.
// The sort makes the tool listing in the prompt identical across requests when
// the set has not changed, which providers rely on for prompt caching.
func (a *DefaultAgent) getToolsList() []tools.Tool {
	a.toolsMu.RLock()
	defer a.toolsMu.RUnlock()

	toolsList := make([]tools.Tool, 0, len(a.tools))
	for _, tool := range a.tools {
		if cv, ok := tool.(tools.ConditionallyVisible); ok && !cv.ShouldShow() {
			continue
		}
		toolsList = append(toolsList, tool)
	}
	sort.Slice(toolsList, func(i, j int) bool {
		return toolsList[i].Name() < toolsList[j].Name()
	})
	return toolsList
}

// getTool retrieves a tool by name (thread-safe)
func (a *DefaultAgent) getTool(name string) (tools.Tool, bool) {
	a.toolsMu.RLock()
	defer a.toolsMu.RUnlock()

	tool, exists := a.tools[name]
	return tool, exists
}
