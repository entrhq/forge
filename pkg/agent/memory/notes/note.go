package notes

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// idCounter is used alongside the timestamp in GenerateID to guarantee
// uniqueness when multiple notes are created within the same nanosecond.
var idCounter atomic.Int64

const (
	// MaxContentLength is the maximum number of characters allowed in note content
	MaxContentLength = 800

	// MinTags is the minimum number of tags required per note
	MinTags = 1

	// MaxTags is the maximum number of tags allowed per note
	MaxTags = 5

	// IDPrefix is the prefix used for all note IDs
	IDPrefix = "note_"
)

// Note represents a single scratchpad entry with content, tags, and metadata
type Note struct {
	ID        string    // Unique identifier: "note_" + Unix milliseconds
	Content   string    // Note content (max 800 chars)
	Tags      []string  // Associated tags (1-5 required)
	Scratched bool      // Whether note is marked as addressed/obsolete
	CreatedAt time.Time // Creation timestamp
	UpdatedAt time.Time // Last update timestamp
}

// NewNote creates a new note with the given content and tags.
// Returns an error if validation fails.
func NewNote(content string, tags []string) (*Note, error) {
	// Validate content
	if err := ValidateContent(content); err != nil {
		return nil, err
	}

	// Validate tags
	if err := ValidateTags(tags); err != nil {
		return nil, err
	}

	now := time.Now()
	return &Note{
		ID:        GenerateID(),
		Content:   content,
		Tags:      normalizeTags(tags),
		Scratched: false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// ValidateContent checks if the content meets the requirements
func ValidateContent(content string) error {
	if content == "" {
		return fmt.Errorf("note content cannot be empty")
	}

	if len(content) > MaxContentLength {
		return fmt.Errorf(
			"note content exceeds maximum length of %d characters (got %d). "+
				"Please shorten the content or split into multiple notes",
			MaxContentLength, len(content),
		)
	}

	return nil
}

// ValidateTags checks if the tags meet the requirements
func ValidateTags(tags []string) error {
	if len(tags) < MinTags {
		return fmt.Errorf(
			"note requires at least %d tag (got %d). "+
				"Please provide at least %d tag for organization",
			MinTags, len(tags), MinTags,
		)
	}

	if len(tags) > MaxTags {
		return fmt.Errorf(
			"note requires 1-%d tags (got %d). "+
				"Please provide at least 1 tag and no more than %d tags for organization",
			MaxTags, len(tags), MaxTags,
		)
	}

	// Check for empty tags
	for i, tag := range tags {
		if strings.TrimSpace(tag) == "" {
			return fmt.Errorf("tag at position %d is empty", i)
		}
	}

	return nil
}

// GenerateID creates a unique note ID using the current timestamp combined with
// a monotonically increasing counter to guarantee uniqueness even when multiple
// notes are created within the same nanosecond (e.g. under concurrent load).
func GenerateID() string {
	seq := idCounter.Add(1)
	return fmt.Sprintf("%s%d_%d", IDPrefix, time.Now().UnixNano(), seq)
}

// normalizeTags trims whitespace and converts tags to lowercase for consistency
func normalizeTags(tags []string) []string {
	normalized := make([]string, len(tags))
	for i, tag := range tags {
		normalized[i] = strings.ToLower(strings.TrimSpace(tag))
	}
	return normalized
}

// Update modifies the note's content and/or tags.
// At least one parameter must be non-nil.
func (n *Note) Update(content *string, tags []string) error {
	if content == nil && tags == nil {
		return fmt.Errorf("at least one of content or tags must be provided for update")
	}

	// Validate new content if provided
	if content != nil {
		if err := ValidateContent(*content); err != nil {
			return err
		}
	}

	// Validate new tags if provided
	if tags != nil {
		if err := ValidateTags(tags); err != nil {
			return err
		}
	}

	// Apply updates
	if content != nil {
		n.Content = *content
	}

	if tags != nil {
		n.Tags = normalizeTags(tags)
	}

	n.UpdatedAt = time.Now()
	return nil
}

// Scratch marks the note as addressed/obsolete
func (n *Note) Scratch() {
	n.Scratched = true
	n.UpdatedAt = time.Now()
}

// Unscratched marks the note as active again
func (n *Note) Unscratched() {
	n.Scratched = false
	n.UpdatedAt = time.Now()
}

// HasTag checks if the note has a specific tag (case-insensitive)
func (n *Note) HasTag(tag string) bool {
	normalized := strings.ToLower(strings.TrimSpace(tag))
	for _, t := range n.Tags {
		if t == normalized {
			return true
		}
	}
	return false
}

// MatchesAllTags checks if the note has all specified tags (case-insensitive, AND logic)
func (n *Note) MatchesAllTags(tags []string) bool {
	if len(tags) == 0 {
		return true
	}

	for _, tag := range tags {
		if !n.HasTag(tag) {
			return false
		}
	}
	return true
}

// ContainsText checks if the note content contains the query string (case-insensitive)
func (n *Note) ContainsText(query string) bool {
	if query == "" {
		return true
	}

	return strings.Contains(
		strings.ToLower(n.Content),
		strings.ToLower(query),
	)
}
