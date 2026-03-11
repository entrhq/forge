package retrieval

import (
	"strings"
	"testing"

	"github.com/entrhq/forge/pkg/agent/longtermmemory"
)

func TestFormatInjection_Empty(t *testing.T) {
	got := FormatInjection(nil, 0)
	if got != "" {
		t.Errorf("expected empty string for nil memories, got %q", got)
	}

	got = FormatInjection([]*longtermmemory.MemoryFile{}, 0)
	if got != "" {
		t.Errorf("expected empty string for empty memories, got %q", got)
	}
}

func TestFormatInjection_ContainsMemoryContent(t *testing.T) {
	memories := []*longtermmemory.MemoryFile{
		makeMemoryFile("1", "The project uses Go 1.24.", "architecture"),
		makeMemoryFile("2", "Tests are run with make test.", "workflow"),
	}
	got := FormatInjection(memories, 0)

	if !strings.Contains(got, "The project uses Go 1.24.") {
		t.Errorf("output missing first memory content: %q", got)
	}
	if !strings.Contains(got, "Tests are run with make test.") {
		t.Errorf("output missing second memory content: %q", got)
	}
}

func TestFormatInjection_HasWrapperTags(t *testing.T) {
	memories := []*longtermmemory.MemoryFile{
		makeMemoryFile("1", "some fact", ""),
	}
	got := FormatInjection(memories, 0)

	if !strings.Contains(got, "<long_term_memory>") {
		t.Errorf("output missing opening tag: %q", got)
	}
	if !strings.Contains(got, "</long_term_memory>") {
		t.Errorf("output missing closing tag: %q", got)
	}
}

func TestFormatInjection_TokenBudgetTruncates(t *testing.T) {
	// Build enough memories to exceed a small budget.
	memories := make([]*longtermmemory.MemoryFile, 0, 20)
	for range 20 {
		memories = append(memories, makeMemoryFile("id", strings.Repeat("word ", 50), ""))
	}

	// A budget of 10 tokens ≈ 40 chars — should drastically truncate.
	got := FormatInjection(memories, 10)
	full := FormatInjection(memories, 0)

	if len(got) >= len(full) {
		t.Errorf("budget-constrained output (%d) should be shorter than full (%d)", len(got), len(full))
	}
}

func TestFormatInjection_ZeroBudgetIncludesAll(t *testing.T) {
	memories := []*longtermmemory.MemoryFile{
		makeMemoryFile("a", "fact one", ""),
		makeMemoryFile("b", "fact two", ""),
		makeMemoryFile("c", "fact three", ""),
	}
	got := FormatInjection(memories, 0)

	for _, m := range memories {
		if !strings.Contains(got, m.Content) {
			t.Errorf("output missing content for memory %q", m.Meta.ID)
		}
	}
}

func TestFormatInjection_SingleEntry(t *testing.T) {
	m := makeMemoryFile("solo", "only memory fact", "decisions")
	got := FormatInjection([]*longtermmemory.MemoryFile{m}, 0)

	if !strings.Contains(got, "only memory fact") {
		t.Errorf("output missing content: %q", got)
	}
	if !strings.HasPrefix(strings.TrimSpace(got), "<long_term_memory>") {
		t.Errorf("output should start with opening tag: %q", got)
	}
}
