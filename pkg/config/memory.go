package config

import (
	"sync"
)

const (
	// SectionIDMemory is the identifier for the memory settings section
	SectionIDMemory = "memory"
)

// MemorySection manages memory configuration settings.
type MemorySection struct {
	Enabled                  bool
	ClassifierModel          string
	HypothesisModel          string
	EmbeddingModel           string
	EmbeddingBaseURL         string
	EmbeddingAPIKey          string
	RetrievalTopK            int
	RetrievalHopDepth        int
	RetrievalHypothesisCount int
	InjectionTokenBudget     int
	mu                       sync.RWMutex
}

// NewMemorySection creates a new memory section with default settings.
func NewMemorySection() *MemorySection {
	return &MemorySection{
		Enabled:                  true,
		ClassifierModel:          "",
		HypothesisModel:          "",
		EmbeddingModel:           "",
		EmbeddingBaseURL:         "",
		EmbeddingAPIKey:          "",
		RetrievalTopK:            10,
		RetrievalHopDepth:        1,
		RetrievalHypothesisCount: 5,
		InjectionTokenBudget:     0,
	}
}

// ID returns the section identifier.
func (s *MemorySection) ID() string {
	return SectionIDMemory
}

// Title returns the section title.
func (s *MemorySection) Title() string {
	return "Memory Settings"
}

// Description returns the section description.
func (s *MemorySection) Description() string {
	return "Configure long-term memory capabilities including embedding and retrieval models."
}

// Data returns the current configuration data.
func (s *MemorySection) Data() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"enabled":                    s.Enabled,
		"classifier_model":           s.ClassifierModel,
		"hypothesis_model":           s.HypothesisModel,
		"embedding_model":            s.EmbeddingModel,
		"embedding_base_url":         s.EmbeddingBaseURL,
		"embedding_api_key":          s.EmbeddingAPIKey,
		"retrieval_top_k":            s.RetrievalTopK,
		"retrieval_hop_depth":        s.RetrievalHopDepth,
		"retrieval_hypothesis_count": s.RetrievalHypothesisCount,
		"injection_token_budget":     s.InjectionTokenBudget,
	}
}

// intFromAny converts a map value (int or float64) to int, returning false if
// the value is absent or not a numeric type. JSON/YAML unmarshal produces
// float64 for numbers, so both types must be handled.
func intFromAny(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

// SetData updates the configuration from the provided data.
func (s *MemorySection) SetData(data map[string]any) error {
	if data == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if enabled, ok := data["enabled"].(bool); ok {
		s.Enabled = enabled
	}
	if model, ok := data["classifier_model"].(string); ok {
		s.ClassifierModel = model
	}
	if hypothesis, ok := data["hypothesis_model"].(string); ok {
		s.HypothesisModel = hypothesis
	}
	if embedding, ok := data["embedding_model"].(string); ok {
		s.EmbeddingModel = embedding
	}
	if url, ok := data["embedding_base_url"].(string); ok {
		s.EmbeddingBaseURL = url
	}
	if key, ok := data["embedding_api_key"].(string); ok {
		s.EmbeddingAPIKey = key
	}
	if v, ok := intFromAny(data["retrieval_top_k"]); ok {
		s.RetrievalTopK = v
	}
	if v, ok := intFromAny(data["retrieval_hop_depth"]); ok {
		s.RetrievalHopDepth = v
	}
	if v, ok := intFromAny(data["retrieval_hypothesis_count"]); ok {
		s.RetrievalHypothesisCount = v
	}
	if v, ok := intFromAny(data["injection_token_budget"]); ok {
		s.InjectionTokenBudget = v
	}

	return nil
}

// Validate validates the current configuration.
func (s *MemorySection) Validate() error {
	// Memory configuration is optional - validation always passes
	return nil
}

// Reset resets the section to default configuration.
func (s *MemorySection) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Enabled = true
	s.ClassifierModel = ""
	s.HypothesisModel = ""
	s.EmbeddingModel = ""
	s.EmbeddingBaseURL = ""
	s.EmbeddingAPIKey = ""
	s.RetrievalTopK = 10
	s.RetrievalHopDepth = 1
	s.RetrievalHypothesisCount = 5
	s.InjectionTokenBudget = 0
}

func (s *MemorySection) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Enabled
}

func (s *MemorySection) SetEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Enabled = enabled
}

func (s *MemorySection) GetClassifierModel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ClassifierModel
}

func (s *MemorySection) SetClassifierModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ClassifierModel = model
}

func (s *MemorySection) GetHypothesisModel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.HypothesisModel
}

func (s *MemorySection) SetHypothesisModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.HypothesisModel = model
}

func (s *MemorySection) GetEmbeddingModel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.EmbeddingModel
}

func (s *MemorySection) SetEmbeddingModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EmbeddingModel = model
}

func (s *MemorySection) GetEmbeddingAPIKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.EmbeddingAPIKey
}

func (s *MemorySection) SetEmbeddingAPIKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EmbeddingAPIKey = key
}

func (s *MemorySection) GetEmbeddingBaseURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.EmbeddingBaseURL
}

func (s *MemorySection) SetEmbeddingBaseURL(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EmbeddingBaseURL = url
}

func (s *MemorySection) GetRetrievalTopK() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.RetrievalTopK
}

func (s *MemorySection) SetRetrievalTopK(k int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RetrievalTopK = k
}

func (s *MemorySection) GetRetrievalHopDepth() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.RetrievalHopDepth
}

func (s *MemorySection) SetRetrievalHopDepth(depth int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RetrievalHopDepth = depth
}

func (s *MemorySection) GetRetrievalHypothesisCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.RetrievalHypothesisCount
}

func (s *MemorySection) SetRetrievalHypothesisCount(count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.RetrievalHypothesisCount = count
}

func (s *MemorySection) GetInjectionTokenBudget() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.InjectionTokenBudget
}

func (s *MemorySection) SetInjectionTokenBudget(budget int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.InjectionTokenBudget = budget
}
