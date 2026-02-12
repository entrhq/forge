package custom

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToolMetadata_Validate(t *testing.T) {
	tests := []struct {
		name    string
		meta    ToolMetadata
		wantErr bool
	}{
		{
			name: "valid metadata",
			meta: ToolMetadata{
				Name:        "test-tool",
				Description: "A test tool",
				Version:     "1.0.0",
				Entrypoint:  "test-tool",
				Usage:       "Usage instructions",
				Parameters: []Parameter{
					{
						Name:        "param1",
						Type:        "string",
						Required:    true,
						Description: "First parameter",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			meta: ToolMetadata{
				Description: "A test tool",
				Version:     "1.0.0",
				Entrypoint:  "test-tool",
			},
			wantErr: true,
		},
		{
			name: "missing description",
			meta: ToolMetadata{
				Name:       "test-tool",
				Version:    "1.0.0",
				Entrypoint: "test-tool",
			},
			wantErr: true,
		},
		{
			name: "invalid parameter type",
			meta: ToolMetadata{
				Name:        "test-tool",
				Description: "A test tool",
				Version:     "1.0.0",
				Entrypoint:  "test-tool",
				Parameters: []Parameter{
					{
						Name:        "param1",
						Type:        "invalid",
						Description: "Test param",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "parameter missing name",
			meta: ToolMetadata{
				Name:        "test-tool",
				Description: "A test tool",
				Version:     "1.0.0",
				Entrypoint:  "test-tool",
				Parameters: []Parameter{
					{
						Type:        "string",
						Description: "Test param",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadSaveMetadata(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	metadataPath := filepath.Join(tmpDir, "tool.yaml")

	// Create test metadata
	original := &ToolMetadata{
		Name:        "test-tool",
		Description: "A test tool",
		Version:     "1.0.0",
		Entrypoint:  "test-tool",
		Usage:       "Multi-line\nusage\ninstructions",
		Parameters: []Parameter{
			{
				Name:        "input",
				Type:        "string",
				Required:    true,
				Description: "Input parameter",
			},
			{
				Name:        "count",
				Type:        "number",
				Required:    false,
				Description: "Count parameter",
			},
		},
	}

	// Save metadata
	if err := SaveMetadata(metadataPath, original); err != nil {
		t.Fatalf("SaveMetadata() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Fatal("Metadata file was not created")
	}

	// Load metadata
	loaded, err := LoadMetadata(metadataPath)
	if err != nil {
		t.Fatalf("LoadMetadata() error = %v", err)
	}

	// Compare fields
	if loaded.Name != original.Name {
		t.Errorf("Name = %v, want %v", loaded.Name, original.Name)
	}
	if loaded.Description != original.Description {
		t.Errorf("Description = %v, want %v", loaded.Description, original.Description)
	}
	if loaded.Version != original.Version {
		t.Errorf("Version = %v, want %v", loaded.Version, original.Version)
	}
	if loaded.Entrypoint != original.Entrypoint {
		t.Errorf("Entrypoint = %v, want %v", loaded.Entrypoint, original.Entrypoint)
	}
	if loaded.Usage != original.Usage {
		t.Errorf("Usage = %v, want %v", loaded.Usage, original.Usage)
	}
	if len(loaded.Parameters) != len(original.Parameters) {
		t.Errorf("Parameters length = %v, want %v", len(loaded.Parameters), len(original.Parameters))
	}
}

func TestGetToolsDir(t *testing.T) {
	dir, err := GetToolsDir()
	if err != nil {
		t.Fatalf("GetToolsDir() error = %v", err)
	}

	// Verify it contains .forge/tools
	if !filepath.IsAbs(dir) {
		t.Errorf("GetToolsDir() returned relative path: %v", dir)
	}
	if filepath.Base(dir) != "tools" {
		t.Errorf("GetToolsDir() base = %v, want tools", filepath.Base(dir))
	}
}

func TestGetToolDir(t *testing.T) {
	dir, err := GetToolDir("test-tool")
	if err != nil {
		t.Fatalf("GetToolDir() error = %v", err)
	}

	if filepath.Base(dir) != "test-tool" {
		t.Errorf("GetToolDir() base = %v, want test-tool", filepath.Base(dir))
	}
}

func TestGetToolMetadataPath(t *testing.T) {
	path, err := GetToolMetadataPath("test-tool")
	if err != nil {
		t.Fatalf("GetToolMetadataPath() error = %v", err)
	}

	if filepath.Base(path) != "tool.yaml" {
		t.Errorf("GetToolMetadataPath() base = %v, want tool.yaml", filepath.Base(path))
	}
}
