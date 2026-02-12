package custom

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateCustomToolTool_Execute(t *testing.T) {
	tests := []struct {
		name        string
		argsXML     string
		wantErr     bool
		errContains string
		validate    func(t *testing.T, toolsDir string)
	}{
		{
			name: "creates tool with all parameters",
			argsXML: `<arguments>
				<name>test_tool</name>
				<description>A test tool</description>
				<version>2.0.0</version>
			</arguments>`,
			wantErr: false,
			validate: func(t *testing.T, toolsDir string) {
				toolDir := filepath.Join(toolsDir, "test_tool")
				
				// Check tool directory exists
				if _, err := os.Stat(toolDir); os.IsNotExist(err) {
					t.Errorf("tool directory not created: %s", toolDir)
				}

				// Check tool.yaml exists
				metadataPath := filepath.Join(toolDir, "tool.yaml")
				if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
					t.Errorf("tool.yaml not created: %s", metadataPath)
				}

				// Check Go source file exists
				goFilePath := filepath.Join(toolDir, "test_tool.go")
				if _, err := os.Stat(goFilePath); os.IsNotExist(err) {
					t.Errorf("Go source file not created: %s", goFilePath)
				}

				// Verify metadata content
				metadata, err := LoadMetadata(metadataPath)
				if err != nil {
					t.Fatalf("failed to load metadata: %v", err)
				}

				if metadata.Name != "test_tool" {
					t.Errorf("expected name 'test_tool', got '%s'", metadata.Name)
				}
				if metadata.Description != "A test tool" {
					t.Errorf("expected description 'A test tool', got '%s'", metadata.Description)
				}
				if metadata.Version != "2.0.0" {
					t.Errorf("expected version '2.0.0', got '%s'", metadata.Version)
				}
			},
		},
		{
			name: "creates tool with default version",
			argsXML: `<arguments>
				<name>default_version_tool</name>
				<description>A tool with default version</description>
			</arguments>`,
			wantErr: false,
			validate: func(t *testing.T, toolsDir string) {
				metadataPath := filepath.Join(toolsDir, "default_version_tool", "tool.yaml")
				metadata, err := LoadMetadata(metadataPath)
				if err != nil {
					t.Fatalf("failed to load metadata: %v", err)
				}

				if metadata.Version != "1.0.0" {
					t.Errorf("expected default version '1.0.0', got '%s'", metadata.Version)
				}
			},
		},
		{
			name: "fails with missing name",
			argsXML: `<arguments>
				<description>A test tool</description>
			</arguments>`,
			wantErr:     true,
			errContains: "missing required parameter: name",
		},
		{
			name: "fails with missing description",
			argsXML: `<arguments>
				<name>test_tool</name>
			</arguments>`,
			wantErr:     true,
			errContains: "missing required parameter: description",
		},
		{
			name: "fails with empty name",
			argsXML: `<arguments>
				<name></name>
				<description>A test tool</description>
			</arguments>`,
			wantErr:     true,
			errContains: "missing required parameter: name",
		},
		{
			name: "fails with empty description",
			argsXML: `<arguments>
				<name>test_tool</name>
				<description></description>
			</arguments>`,
			wantErr:     true,
			errContains: "missing required parameter: description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary tools directory
			tmpDir := t.TempDir()

			// Create tool with custom tools directory
			tool := NewCreateCustomToolToolWithDir(tmpDir)
			ctx := context.Background()

			result, metadata, err := tool.Execute(ctx, []byte(tt.argsXML))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing '%s', got nil", tt.errContains)
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == "" {
				t.Error("expected non-empty result")
			}

			if metadata == nil {
				t.Error("expected non-nil metadata")
			}

			if tt.validate != nil {
				tt.validate(t, tmpDir)
			}
		})
	}
}

func TestCreateCustomToolTool_Schema(t *testing.T) {
	tool := NewCreateCustomToolTool()
	schema := tool.Schema()

	if schema == nil {
		t.Fatal("schema is nil")
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("schema properties is not a map")
	}

	// Check required parameters
	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("schema required is not a string array")
	}

	expectedRequired := []string{"name", "description"}
	if len(required) != len(expectedRequired) {
		t.Errorf("expected %d required fields, got %d", len(expectedRequired), len(required))
	}

	// Check properties exist
	expectedProps := []string{"name", "description", "version"}
	for _, prop := range expectedProps {
		if _, exists := properties[prop]; !exists {
			t.Errorf("expected property '%s' not found in schema", prop)
		}
	}
}

func TestCreateCustomToolTool_IsLoopBreaking(t *testing.T) {
	tool := NewCreateCustomToolTool()
	if tool.IsLoopBreaking() {
		t.Error("create_custom_tool should not be loop-breaking")
	}
}

func TestCreateCustomToolTool_Name(t *testing.T) {
	tool := NewCreateCustomToolTool()
	if tool.Name() != "create_custom_tool" {
		t.Errorf("expected name 'create_custom_tool', got '%s'", tool.Name())
	}
}

func TestCreateCustomToolTool_Description(t *testing.T) {
	tool := NewCreateCustomToolTool()
	desc := tool.Description()
	if desc == "" {
		t.Error("description should not be empty")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
