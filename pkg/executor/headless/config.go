package headless

import (
	"fmt"
	"time"
)

// Config represents the configuration for headless mode execution
type Config struct {
	// Task description
	Task string `yaml:"task" json:"task"`

	// Execution mode
	Mode ExecutionMode `yaml:"mode" json:"mode"`

	// Safety constraints
	Constraints ConstraintConfig `yaml:"constraints" json:"constraints"`

	// Quality gates
	QualityGates          []QualityGateConfig `yaml:"quality_gates" json:"quality_gates"`
	QualityGateMaxRetries int                 `yaml:"quality_gate_max_retries" json:"quality_gate_max_retries"` // Global max retries for quality gates (default: 3)

	// Git configuration
	Git GitConfig `yaml:"git" json:"git"`

	// Artifacts configuration
	Artifacts ArtifactConfig `yaml:"artifacts" json:"artifacts"`

	// Workspace directory
	WorkspaceDir string `yaml:"workspace_dir" json:"workspace_dir"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging" json:"logging"`
}

// ExecutionMode defines the execution mode for headless runs
type ExecutionMode string

const (
	// ModeReadOnly allows only read operations
	ModeReadOnly ExecutionMode = "read-only"
	// ModeWrite allows both read and write operations
	ModeWrite ExecutionMode = "write"
)

// ConstraintConfig defines safety constraints for headless execution
type ConstraintConfig struct {
	// File modification limits
	MaxFiles        int      `yaml:"max_files" json:"max_files"`
	MaxLinesChanged int      `yaml:"max_lines_changed" json:"max_lines_changed"`
	AllowedPatterns []string `yaml:"allowed_patterns" json:"allowed_patterns"`
	DeniedPatterns  []string `yaml:"denied_patterns" json:"denied_patterns"`

	// Tool restrictions
	AllowedTools []string `yaml:"allowed_tools" json:"allowed_tools"`

	// Resource limits
	MaxTokens int           `yaml:"max_tokens" json:"max_tokens"`
	Timeout   time.Duration `yaml:"timeout" json:"timeout"`
}

// QualityGateConfig defines a quality gate to run before committing changes
type QualityGateConfig struct {
	Name       string `yaml:"name" json:"name"`
	Command    string `yaml:"command" json:"command"`
	Required   bool   `yaml:"required" json:"required"`
	MaxRetries int    `yaml:"max_retries" json:"max_retries"` // Maximum number of retry attempts (default: 3)
}

// GitConfig defines git operation configuration
type GitConfig struct {
	AutoCommit          bool   `yaml:"auto_commit" json:"auto_commit"`
	AutoPush            bool   `yaml:"auto_push" json:"auto_push"`
	CommitOnQualityFail bool   `yaml:"commit_on_quality_fail" json:"commit_on_quality_fail"` // Whether to commit partial work when quality gates fail
	CommitMessage       string `yaml:"commit_message" json:"commit_message"`
	Branch              string `yaml:"branch" json:"branch"`
	AuthorName          string `yaml:"author_name" json:"author_name"`
	AuthorEmail         string `yaml:"author_email" json:"author_email"`

	// PR creation configuration (ADR-0031)
	CreatePR  bool   `yaml:"create_pr" json:"create_pr"`   // If true, create PR instead of direct push
	PRTitle   string `yaml:"pr_title" json:"pr_title"`     // PR title (optional, auto-generated if empty)
	PRBody    string `yaml:"pr_body" json:"pr_body"`       // PR description (optional, auto-generated if empty)
	PRBase    string `yaml:"pr_base" json:"pr_base"`       // Target branch (default: auto-detected)
	PRDraft   bool   `yaml:"pr_draft" json:"pr_draft"`     // Create as draft PR
	RequirePR bool   `yaml:"require_pr" json:"require_pr"` // Fail if PR creation is not possible (no fallback)
}

// LoggingConfig defines logging configuration
type LoggingConfig struct {
	// Verbosity controls logging level: quiet, normal, verbose, debug
	Verbosity string `yaml:"verbosity" json:"verbosity"`
}

// ArtifactConfig defines artifact generation configuration
type ArtifactConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	OutputDir string `yaml:"output_dir" json:"output_dir"`

	// Individual format flags
	JSON     bool `yaml:"json" json:"json"`
	Markdown bool `yaml:"markdown" json:"markdown"`
	Metrics  bool `yaml:"metrics" json:"metrics"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Task == "" {
		return fmt.Errorf("task description is required")
	}

	if c.Mode != ModeReadOnly && c.Mode != ModeWrite {
		return fmt.Errorf("invalid mode: %s (must be 'read-only' or 'write')", c.Mode)
	}

	if c.WorkspaceDir == "" {
		return fmt.Errorf("workspace directory is required")
	}

	// Validate constraints
	if c.Constraints.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	if c.Constraints.MaxFiles < 0 {
		return fmt.Errorf("max_files cannot be negative")
	}

	if c.Constraints.MaxLinesChanged < 0 {
		return fmt.Errorf("max_lines_changed cannot be negative")
	}

	if c.Constraints.MaxTokens < 0 {
		return fmt.Errorf("max_tokens cannot be negative")
	}

	// Validate PR configuration
	if c.Git.CreatePR {
		if !c.Git.AutoCommit {
			return fmt.Errorf("create_pr requires auto_commit to be enabled")
		}
		if c.Git.Branch == "" {
			return fmt.Errorf("create_pr requires a branch to be specified")
		}
	}

	// Set default verbosity if not specified
	if c.Logging.Verbosity == "" {
		c.Logging.Verbosity = "normal"
	}

	// Validate log level
	validLevels := map[string]bool{
		"quiet":   true,
		"normal":  true,
		"verbose": true,
		"debug":   true,
	}
	if !validLevels[c.Logging.Verbosity] {
		return fmt.Errorf("invalid logging verbosity: %s (must be 'quiet', 'normal', 'verbose', or 'debug')", c.Logging.Verbosity)
	}

	return nil
}

// DefaultConfig returns a default configuration suitable for most use cases
func DefaultConfig() *Config {
	return &Config{
		Mode: ModeWrite,
		Constraints: ConstraintConfig{
			MaxFiles:        10,
			MaxLinesChanged: 500,
			Timeout:         5 * time.Minute,
			MaxTokens:       50000,
			AllowedTools: []string{
				"task_completion",
				"read_file",
				"write_file",
				"apply_diff",
				"search_files",
				"list_files",
				"execute_command",
			},
		},
		Git: GitConfig{
			AutoCommit:    false,
			CommitMessage: "chore: automated changes via Forge",
			AuthorName:    "anvxl[bot]",
			AuthorEmail:   "2365524+anvxl[bot]@users.noreply.github.com",
		},
		Artifacts: ArtifactConfig{
			Enabled:   true,
			OutputDir: ".forge/artifacts",
			JSON:      true,
			Markdown:  true,
			Metrics:   true,
		},
	}
}
