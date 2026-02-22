package longtermmemory

import (
	"fmt"
	"time"
)

// Scope determines where a memory file is stored.
type Scope string

const (
	ScopeRepo Scope = "repo"
	ScopeUser Scope = "user"
)

// Category classifies the kind of information a memory encodes.
type Category string

const (
	CategoryCodingPreferences      Category = "coding-preferences"
	CategoryProjectConventions     Category = "project-conventions"
	CategoryArchitecturalDecisions Category = "architectural-decisions"
	CategoryUserFacts              Category = "user-facts"
	CategoryCorrections            Category = "corrections"
	CategoryPatterns               Category = "patterns"
)

// Relationship describes the directed edge type between two memories.
type Relationship string

const (
	RelationshipSupersedes  Relationship = "supersedes"
	RelationshipRefines     Relationship = "refines"
	RelationshipContradicts Relationship = "contradicts"
	RelationshipRelatesTo   Relationship = "relates-to"
)

// Trigger records what caused the capture pass that created this memory.
type Trigger string

const (
	TriggerCadence    Trigger = "cadence"
	TriggerCompaction Trigger = "compaction"
)

// RelatedMemory is a typed edge to another memory node.
type RelatedMemory struct {
	ID           string       `yaml:"id"`
	Relationship Relationship `yaml:"relationship"`
}

// MemoryMeta holds all YAML front-matter fields.
type MemoryMeta struct {
	ID         string          `yaml:"id"`
	CreatedAt  time.Time       `yaml:"created_at"`
	UpdatedAt  time.Time       `yaml:"updated_at"`
	Version    int             `yaml:"version"`
	Scope      Scope           `yaml:"scope"`
	Category   Category        `yaml:"category"`
	Supersedes *string         `yaml:"supersedes,omitempty"`
	Related    []RelatedMemory `yaml:"related,omitempty"`
	SessionID  string          `yaml:"session_id"`
	Trigger    Trigger         `yaml:"trigger"`
}

// Validate ensures all required memory metadata fields are populated.
func (m *MemoryMeta) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("longtermmemory: missing ID")
	}
	if m.Scope == "" {
		return fmt.Errorf("longtermmemory: missing Scope")
	}
	if m.Category == "" {
		return fmt.Errorf("longtermmemory: missing Category")
	}
	if m.SessionID == "" {
		return fmt.Errorf("longtermmemory: missing SessionID")
	}
	if m.Trigger == "" {
		return fmt.Errorf("longtermmemory: missing Trigger")
	}
	if m.Version <= 0 {
		return fmt.Errorf("longtermmemory: invalid Version")
	}
	return nil
}

// MemoryFile is the fully parsed in-memory representation of a memory file.
type MemoryFile struct {
	Meta    MemoryMeta
	Content string
}
