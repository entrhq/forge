package context

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/entrhq/forge/pkg/agent/memory"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/llm/tokenizer"
	"github.com/entrhq/forge/pkg/logging"
	"github.com/entrhq/forge/pkg/types"
)

var debugLog *logging.Logger

func init() {
	var err error
	debugLog, err = logging.NewLogger("context")
	if err != nil {
		// Logger fell back to stderr due to initialization failure
		debugLog.Warnf("Failed to initialize context logger, using stderr fallback: %v", err)
	}
}

// Manager orchestrates multiple context summarization strategies,
// evaluating them in order and emitting events for TUI feedback.
type Manager struct {
	strategies         []Strategy
	llm                llm.Provider
	summarizationModel string // optional model override for summarization calls
	tokenizer          *tokenizer.Tokenizer
	maxTokens          int
	eventChannel       chan<- *types.AgentEvent
	mu                 sync.RWMutex // protects llm and summarizationModel
}

// NewManager creates a new context manager with the given strategies.
// Strategies are evaluated in the order provided.
// The event channel should be set later via SetEventChannel() once the agent creates it.
func NewManager(llm llm.Provider, maxTokens int, strategies ...Strategy) (*Manager, error) {
	// Create tokenizer for accurate token counting
	tok, err := tokenizer.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create tokenizer: %w", err)
	}

	return &Manager{
		strategies:   strategies,
		llm:          llm,
		tokenizer:    tok,
		maxTokens:    maxTokens,
		eventChannel: nil, // Will be set by agent during initialization
	}, nil
}

// SetEventChannel sets the event channel for emitting summarization events.
// This is called by the agent during initialization after channels are created.
// It also propagates the event channel to strategies that support progress events.
func (m *Manager) SetEventChannel(eventChan chan<- *types.AgentEvent) {
	m.eventChannel = eventChan

	// Propagate to strategies that support event emission
	for _, strategy := range m.strategies {
		// Type assert to check if strategy supports SetEventChannel
		if setter, ok := strategy.(interface {
			SetEventChannel(chan<- *types.AgentEvent)
		}); ok {
			setter.SetEventChannel(eventChan)
		}
	}
}

// SetProvider updates the LLM provider used by this context manager.
// This is called when the agent's provider is hot-reloaded.
func (m *Manager) SetProvider(provider llm.Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.llm = provider
}

// SetSummarizationModel sets the model name to use for summarization LLM calls.
// If empty, summarization uses the same model as the main provider (m.llm).
// The provider must implement llm.ModelCloner for this to take effect.
func (m *Manager) SetSummarizationModel(model string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.summarizationModel = model
}

// GetSummarizationModel returns the currently configured summarization model override.
// An empty string means the main provider model is used.
func (m *Manager) GetSummarizationModel() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.summarizationModel
}

// providerForSummarization returns the provider to use for summarization calls.
// If a summarization model override is configured and the provider implements
// llm.ModelCloner, returns a lightweight clone with the override model.
// Otherwise returns m.llm unchanged.
// The caller must not hold m.mu.
func (m *Manager) providerForSummarization() llm.Provider {
	m.mu.RLock()
	provider := m.llm
	model := m.summarizationModel
	m.mu.RUnlock()

	if model == "" {
		return provider
	}
	if cloner, ok := provider.(llm.ModelCloner); ok {
		return cloner.CloneWithModel(model)
	}
	return provider
}

// EvaluateAndSummarize evaluates all strategies and performs summarization if needed.
// This operation blocks the agent loop but emits events to keep the TUI responsive.
// Returns the total number of messages summarized across all strategies.
func (m *Manager) EvaluateAndSummarize(ctx context.Context, conv *memory.ConversationMemory, currentTokens int) (int, error) {
	totalSummarized := 0

	// Evaluate each strategy in order
	for _, strategy := range m.strategies {
		// Check if strategy should run
		if !strategy.ShouldRun(conv, currentTokens, m.maxTokens) {
			continue
		}

		// Emit start event
		if m.eventChannel != nil {
			debugLog.Printf("Emitting start event for strategy %s", strategy.Name())
			m.eventChannel <- types.NewContextSummarizationStartEvent(
				strategy.Name(),
				currentTokens,
				m.maxTokens,
			)
		}

		startTime := time.Now()

		// Execute summarization (blocking operation)
		debugLog.Printf("Executing Summarize() for strategy %s", strategy.Name())
		summarizedCount, err := strategy.Summarize(ctx, conv, m.providerForSummarization())
		if err != nil {
			debugLog.Printf("Strategy %s failed with error: %v", strategy.Name(), err)
			// Emit error event
			if m.eventChannel != nil {
				m.eventChannel <- types.NewContextSummarizationErrorEvent(
					strategy.Name(),
					err,
				)
			}
			return totalSummarized, fmt.Errorf("strategy %s failed: %w", strategy.Name(), err)
		}

		duration := time.Since(startTime)
		totalSummarized += summarizedCount
		debugLog.Printf("Strategy %s summarized %d messages in %s", strategy.Name(), summarizedCount, duration)

		// Recalculate current tokens after summarization using accurate tokenizer
		messages := conv.GetAll()
		newTokenCount := m.tokenizer.CountMessagesTokens(messages)

		// Calculate tokens saved
		tokensSaved := currentTokens - newTokenCount
		debugLog.Printf("Tokens saved: %d (before: %d, after: %d)", tokensSaved, currentTokens, newTokenCount)

		// Format duration as string
		durationStr := duration.String()

		// Emit complete event
		if m.eventChannel != nil {
			debugLog.Printf("Emitting complete event for strategy %s", strategy.Name())
			m.eventChannel <- types.NewContextSummarizationCompleteEvent(
				strategy.Name(),
				tokensSaved,
				newTokenCount,
				summarizedCount,
				durationStr,
			)
		}

		// Update current tokens for next strategy
		currentTokens = newTokenCount
	}

	return totalSummarized, nil
}

// AddStrategy adds a new strategy to the manager.
// The strategy will be evaluated after existing strategies.
func (m *Manager) AddStrategy(strategy Strategy) {
	m.strategies = append(m.strategies, strategy)
}

// GetStrategies returns the list of registered strategies.
func (m *Manager) GetStrategies() []Strategy {
	return m.strategies
}

// SetMaxTokens updates the maximum token limit.
func (m *Manager) SetMaxTokens(maxTokens int) {
	m.maxTokens = maxTokens
}

// GetMaxTokens returns the current maximum token limit.
func (m *Manager) GetMaxTokens() int {
	return m.maxTokens
}
