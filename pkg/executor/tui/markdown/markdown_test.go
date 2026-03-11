package markdown_test

import (
	"strings"
	"testing"

	"github.com/entrhq/forge/pkg/executor/tui/markdown"
)

func TestNew_DefaultStyle(t *testing.T) {
	r := markdown.New("")
	// Should not panic and should produce output for basic input.
	out := r.Render("hello", 80)
	if out == "" {
		t.Fatal("expected non-empty output for simple text")
	}
}

func TestRender_Heading(t *testing.T) {
	r := markdown.New("dark")
	out := r.Render("# Hello World", 80)
	if out == "" {
		t.Fatal("expected non-empty output for heading")
	}
	// The heading text should appear in the output (with possible ANSI codes).
	if !strings.Contains(stripANSI(out), "Hello World") {
		t.Errorf("heading text missing from rendered output: %q", out)
	}
}

func TestRender_CodeBlock(t *testing.T) {
	r := markdown.New("dark")
	md := "```go\nfmt.Println(\"hello\")\n```"
	out := r.Render(md, 80)
	if out == "" {
		t.Fatal("expected non-empty output for code block")
	}
	if !strings.Contains(stripANSI(out), "fmt.Println") {
		t.Errorf("code block content missing from output: %q", out)
	}
}

func TestRender_BulletList(t *testing.T) {
	r := markdown.New("dark")
	md := "- item one\n- item two\n- item three"
	out := r.Render(md, 80)
	if !strings.Contains(stripANSI(out), "item one") {
		t.Errorf("list item missing from output: %q", out)
	}
	if !strings.Contains(stripANSI(out), "item two") {
		t.Errorf("list item missing from output: %q", out)
	}
}

func TestRender_PlainText(t *testing.T) {
	r := markdown.New("dark")
	// Plain text (no markdown) should pass through without error.
	out := r.Render("Just some plain text without any markdown syntax.", 80)
	if !strings.Contains(stripANSI(out), "plain text") {
		t.Errorf("plain text content missing from output: %q", out)
	}
}

func TestRender_FallbackOnNarrowWidth(t *testing.T) {
	r := markdown.New("dark")
	text := "# Heading"
	// Width below MinWidth should return raw text as-is.
	out := r.Render(text, markdown.MinWidth-1)
	if out != text {
		t.Errorf("expected raw text fallback for narrow width, got: %q", out)
	}
}

func TestRender_NoTrailingNewlines(t *testing.T) {
	r := markdown.New("dark")
	out := r.Render("Some text", 80)
	if strings.HasSuffix(out, "\n") {
		t.Errorf("output should have trailing newlines trimmed, got: %q", out)
	}
}

func TestRender_ZeroWidth(t *testing.T) {
	r := markdown.New("dark")
	// Zero width should default to 80 (not panic).
	out := r.Render("hello", 0)
	if out == "" {
		t.Fatal("expected non-empty output for zero-width fallback")
	}
}

func TestRenderFn_ReflowsOnWidthChange(t *testing.T) {
	r := markdown.New("dark")
	text := "## Section\n\nThis is a paragraph with enough words to potentially reflow across different terminal widths."
	fn := r.RenderFn(text)

	out80 := fn(80)
	out120 := fn(120)

	// Both should contain the text.
	if !strings.Contains(stripANSI(out80), "Section") {
		t.Errorf("80-width output missing heading: %q", out80)
	}
	if !strings.Contains(stripANSI(out120), "Section") {
		t.Errorf("120-width output missing heading: %q", out120)
	}
}

func TestRenderFn_SubtractsViewportPadding(t *testing.T) {
	r := markdown.New("dark")
	fn := r.RenderFn("hello world")

	// Calling with width=80 should render at 76 (80-4 padding).
	// We cannot directly observe the internal width, but it should not panic
	// and should produce valid output.
	out := fn(80)
	if out == "" {
		t.Fatal("expected non-empty output from RenderFn")
	}
}

// stripANSI is a minimal helper for test assertions that strips common ANSI
// escape sequences so we can check plain text content.
func stripANSI(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip until we find the terminating letter.
			i += 2
			for i < len(s) && (s[i] == ';' || (s[i] >= '0' && s[i] <= '9')) {
				i++
			}
			if i < len(s) {
				i++ // skip terminating letter
			}
			continue
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}
