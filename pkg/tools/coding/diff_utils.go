package coding

import (
	"strings"
)

// LineChanges represents the number of lines added and removed in a modification.
type LineChanges struct {
	LinesAdded   int
	LinesRemoved int
}

// CalculateLineChanges computes the number of lines added and removed
// when transforming oldContent into newContent.
func CalculateLineChanges(oldContent, newContent string) LineChanges {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)

	return LineChanges{
		LinesAdded:   len(newLines),
		LinesRemoved: len(oldLines),
	}
}

// splitLines splits content into lines, handling different line ending styles.
// Empty content returns an empty slice (not a slice with one empty string).
func splitLines(content string) []string {
	if content == "" {
		return []string{}
	}

	// Normalize line endings to \n
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	lines := strings.Split(normalized, "\n")

	// If the content ends with a newline, Split will create an empty string
	// at the end. Remove it to get accurate line count.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}
