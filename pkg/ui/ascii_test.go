package ui

import (
	"strings"
	"testing"
)

func TestGenerateASCIIArt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "simple word",
			input:    "FORGE",
			contains: []string{"███", "╗", "║", "╚"},
		},
		{
			name:     "lowercase converted to uppercase",
			input:    "forge",
			contains: []string{"███", "╗", "║", "╚"},
		},
		{
			name:     "single character",
			input:    "A",
			contains: []string{"█████╗", "██╔══██╗"},
		},
		{
			name:     "with spaces",
			input:    "MY AGENT",
			contains: []string{"███", "   "}, // Should have content and spaces
		},
		{
			name:     "numbers",
			input:    "2024",
			contains: []string{"██", "╗", "║"},
		},
		{
			name:     "empty string",
			input:    "",
			contains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateASCIIArt(tt.input)

			if tt.input == "" {
				if result != "" {
					t.Errorf("Expected empty result for empty input, got: %q", result)
				}
				return
			}

			// Check that result is not empty for non-empty input
			if result == "" {
				t.Errorf("Expected non-empty result for input %q", tt.input)
				return
			}

			// Check for expected content
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain %q, but it didn't.\nResult:\n%s", expected, result)
				}
			}

			// Verify it has 6 lines (plus leading newline)
			lines := strings.Split(strings.TrimPrefix(result, "\n"), "\n")
			if len(lines) != 6 {
				t.Errorf("Expected 6 lines of ASCII art, got %d\nResult:\n%s", len(lines), result)
			}

			// Verify each line starts with a tab
			for i, line := range lines {
				if !strings.HasPrefix(line, "\t") {
					t.Errorf("Line %d should start with tab, got: %q", i, line)
				}
			}
		})
	}
}

func TestGenerateASCIIArt_AllCharacters(t *testing.T) {
	// Test that all supported characters render without panic
	supportedChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 -_"

	result := GenerateASCIIArt(supportedChars)

	if result == "" {
		t.Error("Expected non-empty result for supported characters")
	}

	// Should contain box drawing characters
	if !strings.Contains(result, "█") {
		t.Error("Expected result to contain block characters")
	}
}

func TestGenerateASCIIArt_UnsupportedCharacters(t *testing.T) {
	// Unsupported characters should be skipped gracefully
	input := "A@B#C"
	result := GenerateASCIIArt(input)

	// Should still render A, B, C (skipping @ and #)
	if !strings.Contains(result, "█") {
		t.Error("Expected supported characters to be rendered")
	}
}
