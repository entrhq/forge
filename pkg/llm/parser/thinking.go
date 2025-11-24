// Package parser provides utilities for parsing structured content from LLM streams.
package parser

import (
	"strings"

	"github.com/entrhq/forge/pkg/llm"
)

// ThinkingParser parses streaming content and separates <thinking> tags from regular content.
// It maintains state across multiple content chunks to handle tags that span chunks.
type ThinkingParser struct {
	buffer     strings.Builder
	tagBuffer  strings.Builder // Buffer for potential tag content between < and >
	inThinking bool
	inTag      bool // true when we're buffering a potential tag (saw '<' but not yet '>')
}

// NewThinkingParser creates a new thinking parser.
func NewThinkingParser() *ThinkingParser {
	return &ThinkingParser{}
}

// Parse processes a content chunk and returns separate chunks for thinking and message content.
// It handles <thinking> tags that may span multiple chunks by buffering potential tags.
//
// Returns:
//   - thinkingChunk: Non-nil if thinking content is found (with Type = ContentTypeThinking)
//   - messageChunk: Non-nil if message content is found (with Type = ContentTypeMessage)
func (p *ThinkingParser) Parse(content string) (thinkingChunk, messageChunk *llm.StreamChunk) {
	if content == "" {
		return nil, nil
	}

	for _, ch := range content {
		if ch == '<' {
			// If we're already in a tag, the previous < wasn't a real tag
			if p.inTag {
				// Flush the previous buffered tag as regular content
				chunk := p.flushTagBuffer()
				thinkingChunk, messageChunk = p.appendChunk(thinkingChunk, messageChunk, chunk)
			}

			// Flush any accumulated non-tag content before starting new tag
			if p.buffer.Len() > 0 {
				chunk := p.createChunk(p.buffer.String())
				p.buffer.Reset()
				thinkingChunk, messageChunk = p.appendChunk(thinkingChunk, messageChunk, chunk)
			}

			// Start buffering potential tag
			p.inTag = true
			p.tagBuffer.Reset()
			p.tagBuffer.WriteRune(ch)
			continue
		}

		if ch == '>' && p.inTag {
			// Complete the tag
			p.tagBuffer.WriteRune(ch)
			tag := p.tagBuffer.String()
			p.tagBuffer.Reset()
			p.inTag = false

			// Check if this is a thinking tag
			if tag == "<thinking>" {
				p.inThinking = true
				// Don't emit anything - just change state
				continue
			}

			if tag == "</thinking>" {
				p.inThinking = false
				// Don't emit anything - just change state
				continue
			}

			// Not a thinking tag - treat as regular content
			chunk := p.createChunk(tag)
			thinkingChunk, messageChunk = p.appendChunk(thinkingChunk, messageChunk, chunk)
			continue
		}

		// Regular character
		if p.inTag {
			// Accumulate into tag buffer
			p.tagBuffer.WriteRune(ch)
		} else {
			// Accumulate into regular buffer
			p.buffer.WriteRune(ch)
		}
	}

	// Emit any accumulated regular content at end of chunk
	if p.buffer.Len() > 0 {
		chunk := p.createChunk(p.buffer.String())
		p.buffer.Reset()
		thinkingChunk, messageChunk = p.appendChunk(thinkingChunk, messageChunk, chunk)
	}

	return
}

// flushTagBuffer flushes the current tag buffer as regular content
func (p *ThinkingParser) flushTagBuffer() *llm.StreamChunk {
	if p.tagBuffer.Len() == 0 {
		return nil
	}
	text := p.tagBuffer.String()
	p.tagBuffer.Reset()
	return p.createChunk(text)
}

// createChunk creates a chunk with appropriate type based on current mode
func (p *ThinkingParser) createChunk(text string) *llm.StreamChunk {
	if text == "" {
		return nil
	}

	if p.inThinking {
		return &llm.StreamChunk{
			Content: text,
			Type:    llm.ContentTypeThinking,
		}
	}

	return &llm.StreamChunk{
		Content: text,
		Type:    llm.ContentTypeMessage,
	}
}

// appendChunk appends a new chunk to existing chunks based on type
func (p *ThinkingParser) appendChunk(thinkingChunk, messageChunk, newChunk *llm.StreamChunk) (*llm.StreamChunk, *llm.StreamChunk) {
	if newChunk == nil {
		return thinkingChunk, messageChunk
	}

	if newChunk.Type == llm.ContentTypeThinking {
		if thinkingChunk == nil {
			return newChunk, messageChunk
		}
		thinkingChunk.Content += newChunk.Content
		return thinkingChunk, messageChunk
	}

	if messageChunk == nil {
		return thinkingChunk, newChunk
	}
	messageChunk.Content += newChunk.Content
	return thinkingChunk, messageChunk
}

// IsInThinking returns true if currently parsing thinking content.
func (p *ThinkingParser) IsInThinking() bool {
	return p.inThinking
}

// Flush returns any buffered content that hasn't been emitted yet.
// This should be called at the end of a stream to ensure all content is processed.
func (p *ThinkingParser) Flush() (thinkingChunk, messageChunk *llm.StreamChunk) {
	// Flush any incomplete tag first
	if p.inTag && p.tagBuffer.Len() > 0 {
		chunk := p.flushTagBuffer()
		thinkingChunk, messageChunk = p.appendChunk(thinkingChunk, messageChunk, chunk)
		p.inTag = false
	}

	// Flush regular buffer
	if p.buffer.Len() > 0 {
		text := p.buffer.String()
		p.buffer.Reset()
		chunk := p.createChunk(text)
		thinkingChunk, messageChunk = p.appendChunk(thinkingChunk, messageChunk, chunk)
	}

	return thinkingChunk, messageChunk
}

// Reset resets the parser state for a new stream.
func (p *ThinkingParser) Reset() {
	p.buffer.Reset()
	p.tagBuffer.Reset()
	p.inThinking = false
	p.inTag = false
}
