package config

import (
	"sync"
)

const (
	// SectionIDMultimodal is the identifier for the multimodal settings section
	SectionIDMultimodal = "multimodal"
)

// MultimodalSection manages multimodal document analysis settings.
type MultimodalSection struct {
	Model        string // model to use for multimodal analysis
	PDFPageLimit int    // max pages to analyze per PDF (0 = all pages)
	mu           sync.RWMutex
}

// NewMultimodalSection creates a new multimodal section with default settings.
func NewMultimodalSection() *MultimodalSection {
	return &MultimodalSection{
		Model:        "", // empty = use main LLM model
		PDFPageLimit: 10, // default to first 10 pages
	}
}

// ID returns the section identifier.
func (s *MultimodalSection) ID() string {
	return SectionIDMultimodal
}

// Title returns the section title.
func (s *MultimodalSection) Title() string {
	return "Multimodal Settings"
}

// Description returns the section description.
func (s *MultimodalSection) Description() string {
	return "Configure multimodal document analysis. Set model to override which LLM is used for image/PDF analysis. Set pdf_page_limit to control truncation (0 = all pages)."
}

// Data returns the current configuration data.
func (s *MultimodalSection) Data() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"model":          s.Model,
		"pdf_page_limit": s.PDFPageLimit,
	}
}

// SetData updates the configuration from the provided data.
func (s *MultimodalSection) SetData(data map[string]any) error {
	if data == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if model, ok := data["model"].(string); ok {
		s.Model = model
	}

	if pdfPageLimit, ok := data["pdf_page_limit"].(int); ok {
		s.PDFPageLimit = pdfPageLimit
	}

	// Handle float64 conversion from JSON
	if pdfPageLimit, ok := data["pdf_page_limit"].(float64); ok {
		s.PDFPageLimit = int(pdfPageLimit)
	}

	return nil
}

// Validate validates the current configuration.
func (s *MultimodalSection) Validate() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// pdf_page_limit can be 0 (all pages) or any positive integer
	// No validation needed as all values are acceptable
	return nil
}

// Reset resets the section to default configuration.
func (s *MultimodalSection) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Model = ""
	s.PDFPageLimit = 10
}

// GetModel returns the configured model name for multimodal analysis.
// An empty string means use the main LLM model.
func (s *MultimodalSection) GetModel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Model
}

// SetModel sets the model name for multimodal analysis.
// Pass an empty string to revert to using the main model.
func (s *MultimodalSection) SetModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Model = model
}

// GetPDFPageLimit returns the configured PDF page limit.
// 0 means analyze all pages.
func (s *MultimodalSection) GetPDFPageLimit() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.PDFPageLimit
}

// SetPDFPageLimit sets the PDF page limit.
// Pass 0 to analyze all pages.
func (s *MultimodalSection) SetPDFPageLimit(limit int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PDFPageLimit = limit
}
