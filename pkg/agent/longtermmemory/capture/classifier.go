package capture

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/entrhq/forge/pkg/agent/longtermmemory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/logging"
	"github.com/entrhq/forge/pkg/types"
)

// classifiedMemory is the JSON structure returned by the classifier LLM.
// Fields mirror the memory schema but use plain strings so JSON unmarshalling
// is simple; validation and type conversion happen in Classify().
type classifiedMemory struct {
	Content    string                         `json:"content"`
	Scope      longtermmemory.Scope           `json:"scope"`
	Category   longtermmemory.Category        `json:"category"`
	Supersedes string                         `json:"supersedes,omitempty"`
	Related    []longtermmemory.RelatedMemory `json:"related,omitempty"`
}

// validScopes is the set of accepted scope values.
// Any LLM-returned value outside this set is rejected with a debug log.
var validScopes = map[longtermmemory.Scope]struct{}{
	longtermmemory.ScopeRepo: {},
	longtermmemory.ScopeUser: {},
}

// validCategories is the set of accepted category values.
var validCategories = map[longtermmemory.Category]struct{}{
	longtermmemory.CategoryCodingPreferences:      {},
	longtermmemory.CategoryProjectConventions:     {},
	longtermmemory.CategoryArchitecturalDecisions: {},
	longtermmemory.CategoryUserFacts:              {},
	longtermmemory.CategoryCorrections:            {},
	longtermmemory.CategoryPatterns:               {},
}

// validRelationships is the set of accepted relationship values for related edges.
// Any LLM-returned value outside this set is dropped with a warning.
var validRelationships = map[longtermmemory.Relationship]struct{}{
	longtermmemory.RelationshipSupersedes:  {},
	longtermmemory.RelationshipRefines:     {},
	longtermmemory.RelationshipContradicts: {},
	longtermmemory.RelationshipRelatesTo:   {},
}

// triggerToMemoryTrigger maps capture TriggerKind values to the storage-layer
// Trigger constants used in MemoryMeta. The ADR spec calls the per-turn trigger
// "turn" in the capture package, while the storage layer records it as "cadence".
func triggerToMemoryTrigger(kind TriggerKind) longtermmemory.Trigger {
	switch kind {
	case TriggerKindCompaction:
		return longtermmemory.TriggerCompaction
	default:
		// TriggerKindTurn and any unknown kind map to cadence.
		return longtermmemory.TriggerCadence
	}
}

// Classifier uses an LLM to identify memory-worthy content from a conversation window.
// It loads existing memories before each LLM call so the classifier can reference
// real memory IDs when populating supersedes and related fields.
type Classifier struct {
	provider llm.Provider
	model    string
	store    longtermmemory.MemoryStore
	log      *logging.Logger
}

// NewClassifier constructs a Classifier that uses the given LLM provider and optional
// model override. If model is non-empty and provider implements llm.ModelCloner, a
// lightweight clone is used for each call (following the pattern from ADR-0042).
// store is used to load existing memories before each LLM call.
func NewClassifier(provider llm.Provider, model string, store longtermmemory.MemoryStore, log *logging.Logger) *Classifier {
	if log == nil {
		log, _ = logging.NewLogger("memory")
	}
	return &Classifier{provider: provider, model: model, store: store, log: log}
}

// Classify sends the conversation window to the classifier LLM and returns
// zero or more MemoryFiles to be written to the store. Returns nil, nil if nothing
// in the conversation window meets the memory-worthiness threshold.
//
// Before calling the LLM, Classify loads all existing memories from both scopes so
// the classifier can reference real memory IDs in supersedes and related fields.
// Failure to load existing memories is non-fatal: supersedes/related linking is
// disabled for that turn but the capture still proceeds.
func (c *Classifier) Classify(ctx context.Context, event TriggerEvent) ([]*longtermmemory.MemoryFile, error) {
	if len(event.Messages) == 0 {
		return nil, nil
	}

	// Phase 1: load existing memories so the prompt can include real IDs.
	existing, err := c.loadExistingMemories(ctx)
	if err != nil {
		c.log.Warnf("memory: failed to load existing memories for classifier, supersedes linking disabled: %v", err)
		existing = nil
	}
	c.log.Debugf("memory: classifier loaded %d existing memories from store", len(existing))

	// Phase 2: build prompt and call the classifier LLM.
	prompt := buildClassifierPrompt(event, existing)
	provider := c.providerForClassification()

	messages := []*types.Message{
		types.NewSystemMessage(classifierSystemPrompt),
		types.NewUserMessage(prompt),
	}

	response, err := provider.Complete(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("classifier: LLM call failed: %w", err)
	}

	// Phase 3: parse the JSON array response.
	// LLMs sometimes wrap JSON in markdown code fences — strip them before parsing.
	// A non-JSON or empty response means nothing was memory-worthy — treat as nil, nil.
	rawJSON := stripCodeFences(response.Content)
	var classified []classifiedMemory
	if parseErr := json.Unmarshal([]byte(rawJSON), &classified); parseErr != nil {
		c.log.Warnf("memory: classifier returned non-JSON response (treating as empty): %v — response: %q", parseErr, truncate(response.Content, 200))
		return nil, nil
	}
	if len(classified) == 0 {
		return nil, nil
	}

	// Phase 4: validate each entry, resolve predecessor versions, construct MemoryFiles.
	existingByID := make(map[string]*longtermmemory.MemoryFile, len(existing))
	for _, m := range existing {
		existingByID[m.Meta.ID] = m
	}

	now := time.Now().UTC()
	memTrigger := triggerToMemoryTrigger(event.Kind)
	out := make([]*longtermmemory.MemoryFile, 0, len(classified))

	for _, cm := range classified {
		if cm.Content == "" {
			continue
		}

		// Reject unknown scopes to prevent invalid files from reaching the store.
		if _, ok := validScopes[cm.Scope]; !ok {
			c.log.Warnf("memory: classifier returned unknown scope %q (skipping entry)", cm.Scope)
			continue
		}

		// Reject unknown categories for the same reason.
		if _, ok := validCategories[cm.Category]; !ok {
			c.log.Warnf("memory: classifier returned unknown category %q (skipping entry)", cm.Category)
			continue
		}

		// Validate and filter related edges — reject any with an unknown relationship type.
		filtered := cm.Related[:0]
		for _, rel := range cm.Related {
			if _, ok := validRelationships[rel.Relationship]; !ok {
				c.log.Warnf("memory: classifier returned unknown relationship %q on related entry %q (dropping edge)", rel.Relationship, rel.ID)
				continue
			}
			filtered = append(filtered, rel)
		}
		cm.Related = filtered

		// Resolve the supersedes pointer: nil means this is a first-version memory.
		var supersedes *string
		version := 1
		if cm.Supersedes != "" {
			if predecessor, ok := existingByID[cm.Supersedes]; ok {
				s := cm.Supersedes
				supersedes = &s
				version = predecessor.Meta.Version + 1
			} else {
				// The LLM referenced an ID that doesn't exist — clear the link
				// rather than write a dangling reference.
				c.log.Warnf("memory: classifier supersedes ID %q not found in store (clearing link)", cm.Supersedes)
			}
		}

		m := &longtermmemory.MemoryFile{
			Meta: longtermmemory.MemoryMeta{
				ID:         longtermmemory.NewMemoryID(),
				CreatedAt:  now,
				UpdatedAt:  now,
				Version:    version,
				Scope:      cm.Scope,
				Category:   cm.Category,
				Supersedes: supersedes,
				Related:    cm.Related,
				SessionID:  event.SessionID,
				Trigger:    memTrigger,
			},
			Content: cm.Content,
		}
		out = append(out, m)
	}

	return out, nil
}

// loadExistingMemories retrieves all memories from both scopes.
// Partial failures (one scope unavailable) are tolerated; both scopes must
// fail before an error is returned.
func (c *Classifier) loadExistingMemories(ctx context.Context) ([]*longtermmemory.MemoryFile, error) {
	repoMems, repoErr := c.store.ListByScope(ctx, longtermmemory.ScopeRepo)
	userMems, userErr := c.store.ListByScope(ctx, longtermmemory.ScopeUser)
	if repoErr != nil && userErr != nil {
		return nil, fmt.Errorf("classifier: could not load either scope: repo=%w user=%v", repoErr, userErr)
	}
	combined := make([]*longtermmemory.MemoryFile, 0, len(repoMems)+len(userMems))
	combined = append(combined, repoMems...)
	combined = append(combined, userMems...)
	return combined, nil
}

// truncate returns the first n characters of s, appending "..." if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// stripCodeFences removes markdown code fences that LLMs sometimes wrap JSON
// responses in (e.g. ```json ... ``` or ``` ... ```).
func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	// Remove opening fence: ```json or ```
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx != -1 {
			s = s[idx+1:]
		}
	}
	// Remove closing fence
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

// providerForClassification returns the provider to use for classifier LLM calls.
// If a model override is configured and the provider implements llm.ModelCloner,
// a lightweight clone is returned. Otherwise the original provider is used.
// This follows the same pattern as the context manager (ADR-0042).
func (c *Classifier) providerForClassification() llm.Provider {
	if c.model == "" {
		return c.provider
	}
	if cloner, ok := c.provider.(llm.ModelCloner); ok {
		return cloner.CloneWithModel(c.model)
	}
	return c.provider
}
