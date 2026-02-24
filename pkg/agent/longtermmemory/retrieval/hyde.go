package retrieval

import (
	"context"
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/types"
)

const hydeSystemPrompt = `You are a memory retrieval assistant. Given the recent conversation context, write short, self-contained sentences that could plausibly exist as stored memory entries relevant to this conversation. Each sentence should be phrased as a concrete fact or decision — not a description of what the user is asking, but an example of the kind of fact that would answer them. Output one sentence per line, no preamble.`

// generateHypotheses uses a flash-class LLM to produce N hypothetical memory
// sentences for the given conversation window. These sentences are then embedded
// and used as the query vector against the VectorMap.
func generateHypotheses(ctx context.Context, provider llm.Provider, model string, window string, n int) ([]string, error) {
	userPrompt := fmt.Sprintf(
		"Conversation context:\n%s\n\nWrite %d example memory entries — concrete facts or decisions — that would be relevant to retrieve for this conversation. Each should read like a memory, not like a description of the question.",
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
	if max <= 0 {
		return []string{}
	}
	raw := strings.Split(content, "\n")
	out := make([]string, 0, max)
	for _, l := range raw {
		l = strings.TrimSpace(l)
		// Strip "- " dash-list prefix. A bare "-" (from "- " after trimming)
		// is treated as an empty list item and discarded.
		if len(l) >= 1 && l[0] == '-' && (len(l) == 1 || l[1] == ' ') {
			if len(l) > 2 {
				l = strings.TrimSpace(l[2:])
			} else {
				l = ""
			}
		} else if len(l) > 2 && l[0] >= '0' && l[0] <= '9' && l[1] == '.' {
			// Strip "N." numbered list prefix.
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
