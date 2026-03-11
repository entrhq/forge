package coding

import (
	"fmt"
	"strings"
)

// GenerateUnifiedDiff creates a unified diff between original and modified content.
func GenerateUnifiedDiff(original, modified, filename string) string {
	originalLines := strings.Split(original, "\n")
	modifiedLines := strings.Split(modified, "\n")

	var diff strings.Builder
	fmt.Fprintf(&diff, "--- %s\n", filename)
	fmt.Fprintf(&diff, "+++ %s\n", filename)

	changes := findChanges(originalLines, modifiedLines)
	if len(changes) == 0 {
		return "No changes"
	}

	for _, change := range changes {
		fmt.Fprintf(&diff, "@@ -%d,%d +%d,%d @@\n",
			change.originalStart+1, change.originalCount,
			change.modifiedStart+1, change.modifiedCount)

		for _, line := range change.lines {
			diff.WriteString(line)
			diff.WriteString("\n")
		}
	}

	return diff.String()
}

type diffChange struct {
	originalStart int
	originalCount int
	modifiedStart int
	modifiedCount int
	lines         []string
}

func findChanges(original, modified []string) []diffChange {
	var changes []diffChange
	var currentChange *diffChange

	maxLen := len(original)
	if len(modified) > maxLen {
		maxLen = len(modified)
	}

	for i := 0; i < maxLen; i++ {
		origLine := ""
		modLine := ""

		if i < len(original) {
			origLine = original[i]
		}
		if i < len(modified) {
			modLine = modified[i]
		}

		if origLine != modLine {
			if currentChange == nil {
				currentChange = &diffChange{
					originalStart: i,
					modifiedStart: i,
					lines:         []string{},
				}
			}

			// Collect all deleted and added lines separately for block-style diff
			if i < len(original) {
				currentChange.originalCount++
			}
			if i < len(modified) {
				currentChange.modifiedCount++
			}
		} else if currentChange != nil {
			// Finalize the change block
			finalizeChangeBlock(currentChange, original, modified)
			changes = append(changes, *currentChange)
			currentChange = nil
		}
	}

	if currentChange != nil {
		finalizeChangeBlock(currentChange, original, modified)
		changes = append(changes, *currentChange)
	}

	return changes
}

// finalizeChangeBlock groups all deletions together followed by all additions (block-style)
func finalizeChangeBlock(change *diffChange, original, modified []string) {
	// Add all deleted lines first (red block)
	for i := 0; i < change.originalCount; i++ {
		if change.originalStart+i < len(original) {
			change.lines = append(change.lines, "-"+original[change.originalStart+i])
		}
	}

	// Then add all added lines (green block)
	for i := 0; i < change.modifiedCount; i++ {
		if change.modifiedStart+i < len(modified) {
			change.lines = append(change.lines, "+"+modified[change.modifiedStart+i])
		}
	}
}
