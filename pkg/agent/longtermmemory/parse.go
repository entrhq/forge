package longtermmemory

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const frontMatterDelimiter = "---"

// Parse deserializes a raw memory file byte slice into a MemoryFile.
func Parse(raw []byte) (*MemoryFile, error) {
	s := string(raw)
	if !strings.HasPrefix(s, frontMatterDelimiter) {
		return nil, fmt.Errorf("longtermmemory: missing front-matter delimiter")
	}
	rest := s[len(frontMatterDelimiter):]
	idx := strings.Index(rest, "\n"+frontMatterDelimiter)
	if idx == -1 {
		return nil, fmt.Errorf("longtermmemory: unclosed front-matter block")
	}
	yamlBlock := rest[:idx]
	// Remove the closing delimiter and up to two newlines (if separated by a blank line)
	bodyRaw := rest[idx+len("\n"+frontMatterDelimiter):]
	body := bodyRaw
	if strings.HasPrefix(bodyRaw, "\n\n") {
		body = bodyRaw[2:]
	} else if strings.HasPrefix(bodyRaw, "\n") {
		body = bodyRaw[1:]
	}

	var meta MemoryMeta
	if err := yaml.Unmarshal([]byte(yamlBlock), &meta); err != nil {
		return nil, fmt.Errorf("longtermmemory: front-matter parse error: %w", err)
	}
	return &MemoryFile{Meta: meta, Content: body}, nil
}

// Serialize renders a MemoryFile back to its on-disk byte representation.
func Serialize(m *MemoryFile) ([]byte, error) {
	yamlBytes, err := yaml.Marshal(&m.Meta)
	if err != nil {
		return nil, fmt.Errorf("longtermmemory: serialize error: %w", err)
	}
	var sb strings.Builder
	sb.WriteString(frontMatterDelimiter + "\n")
	sb.Write(yamlBytes)
	sb.WriteString(frontMatterDelimiter + "\n\n")
	sb.WriteString(m.Content)
	return []byte(sb.String()), nil
}
