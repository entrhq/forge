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
		"retrieval_top_k":            s.RetrievalTopK,
		"retrieval_hop_depth":        s.RetrievalHopDepth,
		"retrieval_hypothesis_count": s.RetrievalHypothesisCount,
		"injection_token_budget":     s.InjectionTokenBudget,
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

	if k, ok := data["retrieval_top_k"]; ok {
		if v, ok := k.(int); ok {
			s.RetrievalTopK = v
		} else if v, ok := k.(float64); ok {
			s.RetrievalTopK = int(v)
		}
	}

	if depth, ok := data["retrieval_hop_depth"]; ok {
		if v, ok := depth.(int); ok {
			s.RetrievalHopDepth = v
		} else if v, ok := depth.(float64); ok {
			s.RetrievalHopDepth = int(v)
		}
	}

	if hc, ok := data["retrieval_hypothesis_count"]; ok {
		if v, ok := hc.(int); ok {
			s.RetrievalHypothesisCount = v
		} else if v, ok := hc.(float64); ok {
			s.RetrievalHypothesisCount = int(v)
		}
	}

	if tb, ok := data["injection_token_budget"]; ok {
		if v, ok := tb.(int); ok {
			s.InjectionTokenBudget = v
		} else if v, ok := tb.(float64); ok {
			s.InjectionTokenBudget = int(v)
		}
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
