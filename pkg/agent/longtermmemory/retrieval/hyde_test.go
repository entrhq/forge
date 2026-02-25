package retrieval

import (
	"testing"
)

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		max   int
		want  []string
	}{
		{
			name:  "plain lines",
			input: "fact one\nfact two\nfact three",
			max:   5,
			want:  []string{"fact one", "fact two", "fact three"},
		},
		{
			name:  "capped at max",
			input: "a\nb\nc\nd",
			max:   2,
			want:  []string{"a", "b"},
		},
		{
			name:  "strips dash prefix",
			input: "- alpha\n- beta",
			max:   5,
			want:  []string{"alpha", "beta"},
		},
		{
			name:  "strips numbered list prefix",
			input: "1. first\n2. second\n3. third",
			max:   5,
			want:  []string{"first", "second", "third"},
		},
		{
			name:  "skips empty lines",
			input: "line one\n\n\nline two",
			max:   5,
			want:  []string{"line one", "line two"},
		},
		{
			name:  "trims surrounding whitespace",
			input: "  trimmed  \n  also trimmed  ",
			max:   5,
			want:  []string{"trimmed", "also trimmed"},
		},
		{
			name:  "single character line is not stripped",
			input: "a\nb",
			max:   5,
			want:  []string{"a", "b"},
		},
		{
			name:  "two character line with dash is not stripped (len not > 2)",
			input: "- \n-x",
			max:   5,
			want:  []string{"-x"},
		},
		{
			name:  "digit dot without space is still stripped",
			input: "1.fact",
			max:   5,
			want:  []string{"fact"},
		},
		{
			name:  "non-digit followed by dot is not stripped",
			input: "a.fact",
			max:   5,
			want:  []string{"a.fact"},
		},
		{
			name:  "empty input",
			input: "",
			max:   5,
			want:  []string{},
		},
		{
			name:  "max zero returns empty",
			input: "something",
			max:   0,
			want:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := splitLines(tc.input, tc.max)
			if len(got) != len(tc.want) {
				t.Fatalf("len mismatch: got %d %v, want %d %v", len(got), got, len(tc.want), tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
