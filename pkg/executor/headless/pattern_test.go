package headless

import (
	"testing"
)

func TestPatternMatcher_IsAllowed(t *testing.T) {
	tests := []struct {
		name            string
		allowedPatterns []string
		deniedPatterns  []string
		path            string
		want            bool
	}{
		{
			name:            "no patterns - allow all",
			allowedPatterns: []string{},
			deniedPatterns:  []string{},
			path:            "src/main.go",
			want:            true,
		},
		{
			name:            "simple allowed pattern match",
			allowedPatterns: []string{"src/*.go"},
			deniedPatterns:  []string{},
			path:            "src/main.go",
			want:            true,
		},
		{
			name:            "simple allowed pattern no match",
			allowedPatterns: []string{"src/*.go"},
			deniedPatterns:  []string{},
			path:            "tests/test.go",
			want:            false,
		},
		{
			name:            "denied pattern takes precedence",
			allowedPatterns: []string{"src/**"},
			deniedPatterns:  []string{"src/internal/**"},
			path:            "src/internal/secret.go",
			want:            false,
		},
		{
			name:            "denied pattern blocks specific file",
			allowedPatterns: []string{"**/*.go"},
			deniedPatterns:  []string{"**/secret.go"},
			path:            "src/secret.go",
			want:            false,
		},
		{
			name:            "recursive pattern matches nested files",
			allowedPatterns: []string{"src/**/*.go"},
			deniedPatterns:  []string{},
			path:            "src/pkg/utils/helper.go",
			want:            true,
		},
		{
			name:            "multiple allowed patterns",
			allowedPatterns: []string{"src/*.go", "tests/*.go"},
			deniedPatterns:  []string{},
			path:            "tests/unit_test.go",
			want:            true,
		},
		{
			name:            "path normalization",
			allowedPatterns: []string{"src/*.go"},
			deniedPatterns:  []string{},
			path:            "./src/main.go",
			want:            true,
		},
		{
			name:            "allow directory pattern",
			allowedPatterns: []string{"src/**"},
			deniedPatterns:  []string{},
			path:            "src/pkg/file.go",
			want:            true,
		},
		{
			name:            "complex scenario - allow src except tests",
			allowedPatterns: []string{"src/**"},
			deniedPatterns:  []string{"src/**/*_test.go"},
			path:            "src/pkg/main_test.go",
			want:            false,
		},
		{
			name:            "complex scenario - allowed file in denied directory",
			allowedPatterns: []string{"src/**"},
			deniedPatterns:  []string{"src/vendor/**"},
			path:            "src/main.go",
			want:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm, err := NewPatternMatcher(tt.allowedPatterns, tt.deniedPatterns)
			if err != nil {
				t.Fatalf("NewPatternMatcher() error = %v", err)
			}

			got := pm.IsAllowed(tt.path)
			if got != tt.want {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestNewPatternMatcher_InvalidPatterns(t *testing.T) {
	tests := []struct {
		name            string
		allowedPatterns []string
		deniedPatterns  []string
		wantErr         bool
	}{
		{
			name:            "valid patterns",
			allowedPatterns: []string{"src/*.go", "**/*.txt"},
			deniedPatterns:  []string{"**/test_*.go"},
			wantErr:         false,
		},
		{
			name:            "invalid allowed pattern",
			allowedPatterns: []string{"[invalid"},
			deniedPatterns:  []string{},
			wantErr:         true,
		},
		{
			name:            "invalid denied pattern",
			allowedPatterns: []string{"src/*.go"},
			deniedPatterns:  []string{"[invalid"},
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPatternMatcher(tt.allowedPatterns, tt.deniedPatterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPatternMatcher() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPatternMatcher_DeniedTakesPrecedence(t *testing.T) {
	pm, err := NewPatternMatcher(
		[]string{"**/*.go"},
		[]string{"**/secret*.go"},
	)
	if err != nil {
		t.Fatalf("NewPatternMatcher() error = %v", err)
	}

	tests := []struct {
		path string
		want bool
	}{
		{"src/main.go", true},
		{"src/secret.go", false},
		{"src/secrets.go", false},
		{"tests/secret_test.go", false},
		{"src/pkg/helper.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := pm.IsAllowed(tt.path)
			if got != tt.want {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
