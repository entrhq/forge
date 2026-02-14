package browser

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

// CleanedHTML represents cleaned HTML content with metadata
type CleanedHTML struct {
	HTML        string
	Title       string
	Description string
	Truncated   bool
}

// cleanHTML extracts and cleans HTML content, preserving semantic structure
// while removing scripts, styles, and other noise.
func cleanHTML(rawHTML string, maxLength int) (*CleanedHTML, error) {
	doc, err := html.Parse(strings.NewReader(rawHTML))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	result := &CleanedHTML{}

	// Extract metadata
	result.Title = extractTitle(doc)
	result.Description = extractMetaDescription(doc)

	// Clean and build HTML
	var builder strings.Builder
	var currentLength int
	result.Truncated = cleanNode(doc, &builder, &currentLength, maxLength, 0)

	result.HTML = builder.String()
	return result, nil
}

// cleanNode recursively processes HTML nodes, removing unwanted elements
// and preserving semantic structure with key attributes.
func cleanNode(n *html.Node, builder *strings.Builder, currentLength *int, maxLength int, depth int) bool {
	if *currentLength >= maxLength {
		return true // Truncated
	}

	// Skip unwanted node types
	if n.Type == html.CommentNode {
		return false
	}

	// Skip script, style, noscript, and other noise elements
	if n.Type == html.ElementNode && isSkippedElement(strings.ToLower(n.Data)) {
		return false
	}

	// Process text nodes
	if n.Type == html.TextNode {
		return processTextNode(n, builder, currentLength, maxLength)
	}

	// Process element nodes
	if n.Type == html.ElementNode {
		return processElementNode(n, builder, currentLength, maxLength, depth)
	}

	// Process children for document/fragment nodes
	return processChildren(n, builder, currentLength, maxLength, depth)
}

// processTextNode handles text node processing with truncation
func processTextNode(n *html.Node, builder *strings.Builder, currentLength *int, maxLength int) bool {
	text := strings.TrimSpace(n.Data)
	if text == "" {
		return false
	}

	if *currentLength+len(text) > maxLength {
		remaining := maxLength - *currentLength
		text = text[:remaining] + "..."
		builder.WriteString(text)
		*currentLength = maxLength
		return true
	}

	builder.WriteString(text)
	*currentLength += len(text)
	return false
}

// processElementNode handles element node processing with attributes and children
func processElementNode(n *html.Node, builder *strings.Builder, currentLength *int, maxLength int, depth int) bool {
	tagName := strings.ToLower(n.Data)

	// Add indentation for readability
	if depth > 0 && isBlockElement(tagName) {
		builder.WriteString("\n")
		builder.WriteString(strings.Repeat("  ", depth))
	}

	// Write opening tag with preserved attributes
	builder.WriteString("<")
	builder.WriteString(tagName)

	// Preserve important attributes
	for _, attr := range n.Attr {
		if shouldPreserveAttribute(tagName, attr.Key) {
			fmt.Fprintf(builder, ` %s="%s"`, attr.Key, html.EscapeString(attr.Val))
		}
	}

	builder.WriteString(">")
	*currentLength += len(tagName) + 2

	// Process children
	truncated := processChildren(n, builder, currentLength, maxLength, depth+1)

	// Write closing tag for non-void elements
	if !isVoidElement(tagName) {
		if isBlockElement(tagName) {
			builder.WriteString("\n")
			builder.WriteString(strings.Repeat("  ", depth))
		}
		builder.WriteString("</")
		builder.WriteString(tagName)
		builder.WriteString(">")
		*currentLength += len(tagName) + 3
	}

	return truncated
}

// processChildren recursively processes child nodes
func processChildren(n *html.Node, builder *strings.Builder, currentLength *int, maxLength int, depth int) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if cleanNode(c, builder, currentLength, maxLength, depth) {
			return true
		}
	}
	return false
}

// isSkippedElement returns true for elements that should be completely removed
func isSkippedElement(tagName string) bool {
	skipped := map[string]bool{
		"script":   true,
		"style":    true,
		"noscript": true,
		"iframe":   true,
		"embed":    true,
		"object":   true,
		"svg":      true, // Can be added back if needed for analysis
	}
	return skipped[tagName]
}

// isBlockElement returns true for block-level elements (for formatting)
func isBlockElement(tagName string) bool {
	blocks := map[string]bool{
		"div":        true,
		"p":          true,
		"section":    true,
		"article":    true,
		"header":     true,
		"footer":     true,
		"nav":        true,
		"main":       true,
		"aside":      true,
		"h1":         true,
		"h2":         true,
		"h3":         true,
		"h4":         true,
		"h5":         true,
		"h6":         true,
		"ul":         true,
		"ol":         true,
		"li":         true,
		"table":      true,
		"tr":         true,
		"td":         true,
		"th":         true,
		"form":       true,
		"fieldset":   true,
		"blockquote": true,
		"pre":        true,
	}
	return blocks[tagName]
}

// isVoidElement returns true for self-closing elements
func isVoidElement(tagName string) bool {
	voids := map[string]bool{
		"area":   true,
		"base":   true,
		"br":     true,
		"col":    true,
		"embed":  true,
		"hr":     true,
		"img":    true,
		"input":  true,
		"link":   true,
		"meta":   true,
		"param":  true,
		"source": true,
		"track":  true,
		"wbr":    true,
	}
	return voids[tagName]
}

// shouldPreserveAttribute returns true for attributes that are useful for analysis/targeting
func shouldPreserveAttribute(tagName, attrName string) bool {
	attrName = strings.ToLower(attrName)

	// Always preserve global attributes
	if isGlobalAttribute(attrName) {
		return true
	}

	// Preserve data-* attributes (often used for JS targeting)
	if strings.HasPrefix(attrName, "data-") {
		return true
	}

	// Tag-specific attributes
	return isTagSpecificAttribute(tagName, attrName)
}

// isGlobalAttribute returns true for globally preserved attributes
func isGlobalAttribute(attrName string) bool {
	globalAttrs := map[string]bool{
		"id":               true,
		"class":            true,
		"role":             true,
		"aria-label":       true,
		"aria-describedby": true,
	}
	return globalAttrs[attrName]
}

// isTagSpecificAttribute returns true for tag-specific preserved attributes
func isTagSpecificAttribute(tagName, attrName string) bool {
	switch tagName {
	case "a":
		return attrName == "href" || attrName == "target"
	case "img":
		return attrName == "src" || attrName == "alt"
	case "input", "textarea", "select":
		return attrName == "name" || attrName == "type" || attrName == "placeholder" || attrName == "value"
	case "button":
		return attrName == "type" || attrName == "name"
	case "form":
		return attrName == "action" || attrName == "method"
	case "table":
		return attrName == "summary"
	}
	return false
}

// extractTitle extracts the page title from the document
func extractTitle(doc *html.Node) string {
	var title string
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" {
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				title = strings.TrimSpace(n.FirstChild.Data)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
			if title != "" {
				return
			}
		}
	}
	traverse(doc)
	return title
}

// extractMetaDescription extracts the meta description from the document
func extractMetaDescription(doc *html.Node) string {
	var description string
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var isDescription bool
			var content string
			for _, attr := range n.Attr {
				if attr.Key == "name" && attr.Val == "description" {
					isDescription = true
				}
				if attr.Key == "content" {
					content = attr.Val
				}
			}
			if isDescription && content != "" {
				description = strings.TrimSpace(content)
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
			if description != "" {
				return
			}
		}
	}
	traverse(doc)
	return description
}
