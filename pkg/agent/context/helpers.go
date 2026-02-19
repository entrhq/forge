package context

import "github.com/entrhq/forge/pkg/types"

// isSummarized checks if a message has already been summarized.
// It is a shared helper used by both ThresholdSummarizationStrategy and
// ToolCallSummarizationStrategy so neither strategy depends on the other's
// implementation file.
func isSummarized(msg *types.Message) bool {
	if msg.Metadata == nil {
		return false
	}
	summarized, ok := msg.Metadata["summarized"].(bool)
	return ok && summarized
}
