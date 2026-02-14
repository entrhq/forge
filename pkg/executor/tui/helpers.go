package tui

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// getRandomLoadingMessage returns a random loading message to display while agent is thinking
func getRandomLoadingMessage() string {
	messages := []string{
		"Thinking...",
		"Processing...",
		"Analyzing...",
		"Computing...",
		"Working on it...",
		"Contemplating...",
		"Formulating response...",
		"Pondering...",
		"Crunching data...",
		"Running calculations...",
		"Evaluating options...",
		"Assembling thoughts...",
		"Parsing information...",
		"Synthesizing data...",
		"Deliberating...",
		"Examining details...",
		"Crafting solution...",
		"Reviewing possibilities...",
		"Connecting the dots...",
		"Processing request...",
		"Generating ideas...",
		"Organizing thoughts...",
		"Brewing response...",
		"Distilling essence...",
		"Channeling my inner genius...",
		"Consulting the digital crystal ball...",
		"Spinning the hamster wheel faster...",
		"Defragmenting my thoughts...",
		"Warming up the neural networks...",
		"Bribing the electrons to work harder...",
		"Teaching silicon to dream...",
		"Asking the rubber duck for advice...",
		"Translating coffee into code...",
		"Summoning the debugging spirits...",
		"Untangling the spaghetti logic...",
		"Polishing the algorithmic gems...",
		"Herding cats in binary...",
		"Negotiating with stubborn variables...",
		"Convincing the compiler to cooperate...",
		"Dancing with the data structures...",
		"Whispering sweet nothings to the CPU...",
		"Juggling ones and zeros...",
		"Playing chess with chaos theory...",
		"Feeding the code gremlins...",
		"Calibrating the flux capacitor...",
		"Adjusting the reality parameters...",
		"Downloading more RAM...",
		"Applying percussive maintenance...",
		"Sacrificing a USB cable to the tech gods...",
		"Asking ChatGPT what ChatGPT would do...",
	}
	return messages[rand.Intn(len(messages))] //nolint:gosec
}

// formatTokenCount formats a token count with K/M suffixes for readability
func formatTokenCount(count int) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
	if count >= 1000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%d", count)
}

// formatEntry formats a content entry with an icon and optional styling
func formatEntry(icon string, text string, style lipgloss.Style, width int, iconOnly bool) string {
	// Calculate wrap width (full width minus small padding)
	wrapWidth := width - 4
	if wrapWidth <= 0 {
		wrapWidth = 80
	}

	if iconOnly {
		// Style only the icon, keep text white
		styledIcon := style.Render(icon)
		fullText := icon + text // Use unstyled for wrapping calculation
		wrapped := wordWrap(fullText, wrapWidth)

		// Replace the unstyled icon with styled icon in first occurrence
		wrapped = strings.Replace(wrapped, icon, styledIcon, 1)
		return wrapped
	}

	// Style everything (default behavior)
	fullText := icon + text
	wrapped := wordWrap(fullText, wrapWidth)
	return style.Render(wrapped)
}

// wordWrap wraps text to fit within the specified width while preserving paragraph breaks
//
//nolint:gocyclo
func wordWrap(text string, width int) string {
	if width <= 0 {
		width = 80
	}

	var result strings.Builder
	// Split by newlines to preserve paragraph breaks
	paragraphs := strings.Split(text, "\n")

	firstPara := true
	for _, para := range paragraphs {
		// Trim leading/trailing spaces from paragraph but preserve structure
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		if !firstPara {
			result.WriteString("\n")
		}
		firstPara = false

		// Extract and preserve leading whitespace
		leadingSpace := ""
		trimmed := strings.TrimLeft(para, " \t")
		if len(trimmed) < len(para) {
			leadingSpace = para[:len(para)-len(trimmed)]
		}

		words := strings.Fields(trimmed)
		if len(words) == 0 {
			continue
		}

		currentLine := leadingSpace // Start first line with leading space

		for _, word := range words {
			// If a single word is longer than width, break it up
			if len(word) > width {
				// First, flush current line if it has content
				if currentLine != "" {
					result.WriteString(currentLine)
					result.WriteString("\n")
					currentLine = ""
				}

				// Break the long word into chunks
				for len(word) > 0 {
					chunkSize := width
					if len(word) < chunkSize {
						chunkSize = len(word)
					}
					result.WriteString(word[:chunkSize])
					result.WriteString("\n")
					word = word[chunkSize:]
				}
				continue
			}

			// Check if adding this word would exceed width
			switch {
			case currentLine == "" || currentLine == leadingSpace:
				currentLine = leadingSpace + word
			case len(currentLine)+1+len(word) > width:
				// Write current line and start new one
				result.WriteString(currentLine)
				result.WriteString("\n")
				currentLine = word
			default:
				// Add word to current line
				currentLine += " " + word
			}
		}

		// Write final line if there's content
		if currentLine != "" && currentLine != leadingSpace {
			result.WriteString(currentLine)
		}
	}

	return result.String()
}

// updateTextAreaHeight dynamically adjusts the textarea height based on content
// accounting for line wrapping and multi-line input
func (m *model) updateTextAreaHeight() {
	value := m.textarea.Value()
	if value == "" {
		if m.textarea.Height() != 1 {
			m.textarea.SetHeight(1)
			m.recalculateLayout()
		}
		return
	}

	// Calculate visual lines accounting for wrapping
	width := m.textarea.Width()
	if width <= 0 {
		width = 80 // default width
	}

	// Account for prompt width ("> " = 2 chars)
	effectiveWidth := width - 2
	if effectiveWidth <= 0 {
		effectiveWidth = 78
	}

	// Split by actual newlines first
	textLines := strings.Split(value, "\n")
	visualLines := 0

	for _, line := range textLines {
		if line == "" {
			visualLines++ // Empty line still counts as 1 visual line
		} else {
			// Calculate how many visual lines this logical line takes
			lineLen := len(line)
			wrappedLines := (lineLen + effectiveWidth - 1) / effectiveWidth
			if wrappedLines == 0 {
				wrappedLines = 1
			}
			visualLines += wrappedLines
		}
	}

	// Clamp between 1 and MaxHeight
	if visualLines < 1 {
		visualLines = 1
	}
	if visualLines > m.textarea.MaxHeight {
		visualLines = m.textarea.MaxHeight
	}

	// Only update if height changed to avoid unnecessary recalculation
	if visualLines != m.textarea.Height() {
		m.textarea.SetHeight(visualLines)
		m.recalculateLayout()
	}
}
