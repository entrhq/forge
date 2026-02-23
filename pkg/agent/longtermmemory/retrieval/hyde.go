package retrieval

import (
	"context"
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/types"
)

const hydeSystemPrompt = `You are a memory retrieval assistant. Given the recent conversation context, generate short, self-contained sentences that describe facts, decisions, or knowledge that would be useful to retrieve from long-term memory. Output one sentence per line.`

// generateHypotheses uses a flash-class LLM to produce N hypothetical memory
// sentences for the given conversation window. These sentences are then embedded
// and used as the query vector against the VectorMap.
func generateHypotheses(ctx context.Context, provider llm.Provider, model string, window string, n int) ([]string, error) {
	userPrompt := fmt.Sprintf(
		"Conversation context:\n%s\n\nGenerate %d short sentences describing facts or context that might appear in long-term memory relevant to this conversation.",
		window, n,
	)

	messages := []*types.Message{
		types.NewSystemMessage(hydeSystemPrompt),
		types.NewUserMessage(userPrompt),
	}

	// Clone the provider with a different model if needed.
	var resp *types.Message
	var err error

	if mc, ok := provider.(llm.ModelCloner); ok && model != "" {
		p := mc.CloneWithModel(model)
		resp, err = p.Complete(ctx, messages)
	} else {
		resp, err = provider.Complete(ctx, messages)
	}
	if err != nil {
		return nil, fmt.Errorf("hyde: LLM call failed: %w", err)
	}

	lines := splitLines(resp.Content, n)
	return lines, nil
}

// splitLines splits content into non-empty trimmed lines, capped at max.
func splitLines(content string, max int) []string {
	raw := strings.Split(content, "\n")
	out := make([]string, 0, max)
	for _, l := range raw {
		l = strings.TrimSpace(l)
		// Strip common list prefixes: "- ", "1. ", etc.
		if len(l) > 2 && (l[0] == '-' || l[1] == '.') {
			l = strings.TrimSpace(l[2:])
		}
		if l != "" {
			out = append(out, l)
		}
		if len(out) >= max {
			break
		}
	}
	return out
}
