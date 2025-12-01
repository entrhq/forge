package prompts

import (
	"fmt"
	"strings"
)

// XMLExampleProvider is an optional interface that tools can implement
// to provide custom XML usage examples
type XMLExampleProvider interface {
	XMLExample() string
}

// GenerateXMLExample creates a concrete XML example from a JSON Schema
func GenerateXMLExample(schema map[string]interface{}, toolName string) string {
	var builder strings.Builder

	builder.WriteString("<tool>\n")
	builder.WriteString("<server_name>local</server_name>\n")
	builder.WriteString(fmt.Sprintf("<tool_name>%s</tool_name>\n", toolName))
	builder.WriteString("<arguments>\n")

	// Extract properties
	properties, ok := schema["properties"].(map[string]interface{})
	if ok && len(properties) > 0 {
		// Get required fields
		requiredFields := make(map[string]bool)
		if req, ok := schema["required"].([]string); ok {
			for _, field := range req {
				requiredFields[field] = true
			}
		}

		// Generate example for each property
		for propName, propValue := range properties {
			propMap, ok := propValue.(map[string]interface{})
			if !ok {
				continue
			}

			// Skip optional fields in basic example
			if !requiredFields[propName] {
				continue
			}

			example := generatePropertyExample(propName, propMap, "  ")
			builder.WriteString(example)
		}
	}

	builder.WriteString("</arguments>\n")
	builder.WriteString("</tool>")

	return builder.String()
}

// generatePropertyExample creates an XML example for a single property
func generatePropertyExample(name string, propSchema map[string]interface{}, indent string) string {
	propType, _ := propSchema["type"].(string)           //nolint:errcheck
	description, _ := propSchema["description"].(string) //nolint:errcheck

	switch propType {
	case "string":
		return generateStringExample(name, propSchema, description, indent)
	case "integer", "number":
		return generateNumberExample(name, propType, indent)
	case "boolean":
		return fmt.Sprintf("%s<%s>true</%s>\n", indent, name, name)
	case "array":
		return generateArrayExample(name, propSchema, indent)
	case "object":
		return generateObjectExample(name, propSchema, indent)
	default:
		return fmt.Sprintf("%s<%s>value</%s>\n", indent, name, name)
	}
}

// generateStringExample creates example for string properties
func generateStringExample(name string, propSchema map[string]interface{}, description string, indent string) string {
	// Check if this is a code/content field that might have special characters
	isCodeField := strings.Contains(description, "code") ||
		strings.Contains(description, "content") ||
		strings.Contains(description, "diff") ||
		strings.Contains(description, "search") ||
		strings.Contains(description, "replace") ||
		strings.Contains(name, "content") ||
		strings.Contains(name, "search") ||
		strings.Contains(name, "replace")

	if isCodeField {
		// Use XML entity escaping (preferred method per ADR-0024)
		// Show example with escaped entities
		return fmt.Sprintf("%s<%s>example &amp; content</%s>\n", indent, name, name)
	}

	// Simple string
	exampleValue := "value"
	if enum, ok := propSchema["enum"].([]interface{}); ok && len(enum) > 0 {
		if str, ok := enum[0].(string); ok {
			exampleValue = str
		}
	}

	return fmt.Sprintf("%s<%s>%s</%s>\n", indent, name, exampleValue, name)
}

// generateNumberExample creates example for numeric properties
func generateNumberExample(name string, propType string, indent string) string {
	value := "42"
	if propType == "number" {
		value = "3.14"
	}
	return fmt.Sprintf("%s<%s>%s</%s>\n", indent, name, value, name)
}

// generateArrayExample creates example for array properties
func generateArrayExample(name string, propSchema map[string]interface{}, indent string) string {
	items, ok := propSchema["items"].(map[string]interface{})
	if !ok {
		// Simple array - show multiple elements
		return fmt.Sprintf("%s<%s>item1</%s>\n%s<%s>item2</%s>\n",
			indent, name, name, indent, name, name)
	}

	itemType, _ := items["type"].(string) //nolint:errcheck

	// If items are objects, use nested structure
	if itemType == "object" {
		var builder strings.Builder
		builder.WriteString(fmt.Sprintf("%s<%s>\n", indent, name))

		// Generate example item (singular form if possible)
		singularName := name
		if strings.HasSuffix(name, "s") {
			singularName = name[:len(name)-1]
		}

		builder.WriteString(fmt.Sprintf("%s  <%s>\n", indent, singularName))

		if itemProps, ok := items["properties"].(map[string]interface{}); ok {
			for propName, propValue := range itemProps {
				if propMap, ok := propValue.(map[string]interface{}); ok {
					example := generatePropertyExample(propName, propMap, indent+"    ")
					builder.WriteString(example)
				}
			}
		}

		builder.WriteString(fmt.Sprintf("%s  </%s>\n", indent, singularName))
		builder.WriteString(fmt.Sprintf("%s</%s>\n", indent, name))

		return builder.String()
	}

	// Simple array of strings - show multiple elements with the same tag name
	// This matches the XML unmarshaling pattern: Tags []string `xml:"tags>tag"`
	// which expects: <tags><tag>value1</tag><tag>value2</tag></tags>
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s<%s>\n", indent, name))
	
	// Generate 2 example items with singular form
	singularName := name
	if strings.HasSuffix(name, "s") {
		singularName = name[:len(name)-1]
	}
	
	builder.WriteString(fmt.Sprintf("%s  <%s>item1</%s>\n", indent, singularName, singularName))
	builder.WriteString(fmt.Sprintf("%s  <%s>item2</%s>\n", indent, singularName, singularName))
	builder.WriteString(fmt.Sprintf("%s</%s>\n", indent, name))
	
	return builder.String()
}

// generateObjectExample creates example for object properties
func generateObjectExample(name string, propSchema map[string]interface{}, indent string) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("%s<%s>\n", indent, name))

	if props, ok := propSchema["properties"].(map[string]interface{}); ok {
		for propName, propValue := range props {
			if propMap, ok := propValue.(map[string]interface{}); ok {
				example := generatePropertyExample(propName, propMap, indent+"  ")
				builder.WriteString(example)
			}
		}
	}

	builder.WriteString(fmt.Sprintf("%s</%s>\n", indent, name))

	return builder.String()
}
