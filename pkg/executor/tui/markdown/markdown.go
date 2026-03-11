// Package markdown provides a thin wrapper around charmbracelet/glamour for
// rendering markdown text to ANSI-styled terminal output within the TUI.
//
// Key design constraints:
//   - glamour requires complete markdown input; never call Render mid-stream.
//   - Render is called inside DisplayMessage.RenderFn closures so the output
//     is always produced at the correct terminal width (supports resize reflow).
//   - If glamour fails for any reason, raw markdown text is returned as a
//     plain-text fallback so the agent response is never silently dropped.
package markdown

import (
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	glamourstyles "github.com/charmbracelet/glamour/styles"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

const (
	// codeBlockStartMarker and codeBlockEndMarker are ASCII control characters
	// (STX/ETX) injected via BlockPrefix/BlockSuffix in darkStyleConfig.
	// They are used as sentinels in applyCodeBlockBorder to locate code block
	// regions in the rendered output without parsing ANSI escape sequences.
	codeBlockStartMarker = "\x02"
	codeBlockEndMarker   = "\x03"

	// codeBorderANSI is the ANSI sequence for the left border glyph rendered
	// on every line of a code block: a half-block bar in muted slate blue.
	codeBorderANSI = "\x1b[38;2;98;114;164m▌\x1b[0m"

	// codeBackgroundANSI sets the background color (#33353d) for code block
	// lines.  It is re-injected after every \x1b[0m reset inside the line so
	// that chroma's per-token resets do not kill the background fill.
	codeBackgroundANSI = "\x1b[48;2;51;53;61m"

	// ansiReset is the standard ANSI reset sequence.
	ansiReset = "\x1b[0m"
)

const (
	// DefaultStyle is the glamour stylesheet used when GLAMOUR_STYLE is not set.
	DefaultStyle = "dark"

	// MinWidth is the minimum render width below which glamour rendering is
	// skipped in favor of raw text (prevents layout artifacts on tiny terminals).
	MinWidth = 20
)

// Renderer renders markdown to ANSI terminal output using charmbracelet/glamour.
// It is safe to use concurrently.
//
// TermRenderer instances are cached per width so that repeated calls at the
// same terminal width (e.g. every streaming tick) skip re-initialisation.
// The cache is invalidated implicitly: a different width key produces a new
// renderer and the old one is kept alongside it (the set of distinct widths
// seen in a session is tiny — typically just one or two after a resize).
type Renderer struct {
	style     string
	mu        sync.Mutex
	renderers map[int]*glamour.TermRenderer
}

// New returns a Renderer.  If style is empty the DefaultStyle ("dark") is used.
// The GLAMOUR_STYLE environment variable always takes precedence at render time.
func New(style string) *Renderer {
	if style == "" {
		style = DefaultStyle
	}
	return &Renderer{
		style:     style,
		renderers: make(map[int]*glamour.TermRenderer),
	}
}

// Render converts markdown text to ANSI-styled output, word-wrapped at width.
//
//   - Leading/trailing whitespace is trimmed from the rendered output so callers
//     can append their own Trailing newlines via DisplayMessage.Trailing.
//   - Falls back to raw text on any glamour error.
//   - If width < MinWidth the raw text is returned to avoid glamour artifacts.
func (r *Renderer) Render(text string, width int) string {
	if width <= 0 {
		width = 80
	}
	if width < MinWidth {
		return text
	}

	style := r.style
	// Honor GLAMOUR_STYLE env var at render time so users can override without
	// restarting.  An explicit empty string means "use our default".
	if env := os.Getenv("GLAMOUR_STYLE"); env != "" {
		style = env
	}

	gr, err := r.getRenderer(style, width)
	if err != nil {
		return text
	}

	out, err := gr.Render(text)
	if err != nil {
		return text
	}

	if style == DefaultStyle {
		out = applyCodeBlockBorder(out)
	}

	return strings.TrimRight(out, "\n")
}

// darkStyleConfig returns the standard dark StyleConfig with sentinel markers
// injected via BlockPrefix/BlockSuffix.  These are picked up by
// applyCodeBlockBorder to prepend a visible left border to every code line.
//
// Background color overrides are intentionally absent: glamour's
// CodeBlockElement uses its own indent.WriterPipe that bypasses the
// MarginWriter padding path, so BackgroundColor on the outer StyleBlock
// has no visible effect.
func darkStyleConfig() ansi.StyleConfig {
	s := glamourstyles.DarkStyleConfig
	s.CodeBlock.BlockPrefix = codeBlockStartMarker + "\n"
	s.CodeBlock.BlockSuffix = "\n" + codeBlockEndMarker
	// Zero out the built-in margin/indent so glamour doesn't prepend extra
	// leading spaces before each code line.  Our applyCodeBlockBorder step
	// adds the only visible indentation (the border glyph + one space).
	zero := uint(0)
	s.CodeBlock.Margin = &zero
	s.CodeBlock.Indent = &zero
	return s
}

// applyCodeBlockBorder scans the glamour-rendered output for the sentinel
// characters injected by darkStyleConfig, buffers each code block's lines,
// then emits them with:
//   - a ▌ left-border glyph in muted slate blue
//   - a solid #1c1c1c background fill that spans the full block width
//
// Background fill works by:
//  1. Buffering all lines in a block so the maximum visible width is known.
//  2. Re-injecting codeBackgroundANSI after every \x1b[0m reset inside each
//     line — chroma emits resets between tokens, which would otherwise kill
//     the background mid-line.
//  3. Padding each line to max_width with background-colored spaces so the
//     right edge of the block is a solid filled rectangle.
//
// This is a post-processing step because glamour's CodeBlockElement
// hard-codes its indent callback and the outer StyleBlock BackgroundColor
// field is silently ignored by that render path.
func applyCodeBlockBorder(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	inCode := false
	var codeLines []string

	// contentWidth returns the visible width of a code line after stripping
	// ANSI sequences and trimming trailing whitespace that glamour pads to
	// word-wrap width.  This gives the true content width, not the padded width.
	contentWidth := func(cl string) int {
		return runewidth.StringWidth(strings.TrimRight(xansi.Strip(cl), " \t"))
	}

	emitBlock := func() {
		// Strip trailing blank/whitespace-only lines inside the block.
		for len(codeLines) > 0 && strings.TrimSpace(xansi.Strip(codeLines[len(codeLines)-1])) == "" {
			codeLines = codeLines[:len(codeLines)-1]
		}
		// Measure maximum content width (strip ANSI, then trim trailing spaces
		// so glamour's word-wrap padding does not inflate maxW to terminal width).
		maxW := 4
		for _, cl := range codeLines {
			if w := contentWidth(cl); w > maxW {
				maxW = w
			}
		}
		// Add a small right-side margin so the background rectangle breathes.
		maxW += 2
		// Helper: a blank filled line at full block width for top/bottom padding.
		blankLine := codeBorderANSI + codeBackgroundANSI + strings.Repeat(" ", maxW) + ansiReset

		// Top padding line inside the background.
		out = append(out, blankLine)

		// Emit each line: border glyph + background + content + right padding.
		for _, cl := range codeLines {
			visW := contentWidth(cl)
			// Truncate to content width — removes glamour's trailing space
			// padding even when those spaces are embedded after ANSI resets.
			cl = xansi.Truncate(cl, visW, "")
			// Re-inject background after every reset so chroma token resets
			// don't expose the terminal default background mid-line.
			cl = strings.ReplaceAll(cl, ansiReset, ansiReset+codeBackgroundANSI)
			padding := strings.Repeat(" ", maxW-visW)
			out = append(out, codeBorderANSI+codeBackgroundANSI+cl+padding+ansiReset)
		}

		// Bottom padding line inside the background.
		out = append(out, blankLine)
		out = append(out, "") // one blank line after the code block
		codeLines = nil
	}

	for _, line := range lines {
		if strings.Contains(line, codeBlockStartMarker) {
			inCode = true
			codeLines = nil
			continue // drop the sentinel line
		}
		if strings.Contains(line, codeBlockEndMarker) {
			inCode = false
			emitBlock()
			continue // drop the sentinel line
		}
		if inCode {
			codeLines = append(codeLines, line)
		} else {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// getRenderer returns a cached TermRenderer for the given style and width,
// creating one if it does not yet exist.  Callers must not hold r.mu.
func (r *Renderer) getRenderer(style string, width int) (*glamour.TermRenderer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if gr, ok := r.renderers[width]; ok {
		return gr, nil
	}

	// For the default dark style we supply our own tweaked StyleConfig
	// (darker code-block background).  For any other style (including
	// GLAMOUR_STYLE overrides) we fall back to glamour's standard lookup so
	// that user-chosen themes are rendered faithfully.
	var styleOpt glamour.TermRendererOption
	if style == DefaultStyle {
		styleOpt = glamour.WithStyles(darkStyleConfig())
	} else {
		styleOpt = glamour.WithStandardStyle(style)
	}

	gr, err := glamour.NewTermRenderer(
		styleOpt,
		glamour.WithWordWrap(width),
		glamour.WithPreservedNewLines(),
	)
	if err != nil {
		return nil, err
	}
	r.renderers[width] = gr
	return gr, nil
}

// RenderFn returns a closure compatible with DisplayMessage.RenderFn.
//
// The viewport padding (2 chars each side = 4 total) is subtracted from the
// supplied width before rendering so the output stays within the visible area.
// This matches the wrapWidth = width-4 convention used throughout helpers.go.
//
// The closure caches its last rendered output keyed on width.  During active
// streaming, renderMessages is called on every token tick at a constant width,
// so all already-committed messages return their cached string instantly rather
// than re-invoking glamour on every tick.  On a genuine terminal resize a new
// width is seen and the output is re-rendered once then cached again.
func (r *Renderer) RenderFn(text string) func(int) string {
	var (
		mu           sync.Mutex
		cachedWidth  int
		cachedOutput string
	)
	return func(width int) string {
		contentWidth := width - 4
		mu.Lock()
		if contentWidth == cachedWidth && cachedOutput != "" {
			out := cachedOutput
			mu.Unlock()
			return out
		}
		mu.Unlock()
		out := r.Render(text, contentWidth)
		mu.Lock()
		cachedWidth = contentWidth
		cachedOutput = out
		mu.Unlock()
		return out
	}
}
