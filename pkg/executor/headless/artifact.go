package headless

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ArtifactWriter handles writing execution artifacts
type ArtifactWriter struct {
	outputDir string
}

// NewArtifactWriter creates a new artifact writer
func NewArtifactWriter(outputDir string) *ArtifactWriter {
	return &ArtifactWriter{
		outputDir: outputDir,
	}
}

// WriteAll writes all configured artifact formats
func (w *ArtifactWriter) WriteAll(summary *ExecutionSummary) error {
	// Ensure output directory exists
	if err := os.MkdirAll(w.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write JSON execution report
	if err := w.WriteExecutionJSON(summary); err != nil {
		return fmt.Errorf("failed to write execution JSON: %w", err)
	}

	// Write markdown summary
	if err := w.WriteSummaryMarkdown(summary); err != nil {
		return fmt.Errorf("failed to write summary markdown: %w", err)
	}

	// Write metrics JSON
	if err := w.WriteMetricsJSON(summary); err != nil {
		return fmt.Errorf("failed to write metrics JSON: %w", err)
	}

	return nil
}

// WriteExecutionJSON writes the full execution summary as JSON
func (w *ArtifactWriter) WriteExecutionJSON(summary *ExecutionSummary) error {
	path := filepath.Join(w.outputDir, "execution.json")

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal execution summary: %w", err)
	}

	if writeErr := os.WriteFile(path, data, 0600); writeErr != nil {
		return fmt.Errorf("failed to write execution JSON: %w", writeErr)
	}

	return nil
}

// WriteSummaryMarkdown writes a human-readable markdown summary
func (w *ArtifactWriter) WriteSummaryMarkdown(summary *ExecutionSummary) error {
	path := filepath.Join(w.outputDir, "summary.md")

	var md strings.Builder

	// Header
	md.WriteString("# Forge Headless Execution Summary\n\n")
	md.WriteString(fmt.Sprintf("**Task:** %s\n\n", summary.Task))
	md.WriteString(fmt.Sprintf("**Status:** %s\n\n", summary.Status))
	md.WriteString(fmt.Sprintf("**Started:** %s\n\n", summary.StartTime.Format(time.RFC3339)))
	md.WriteString(fmt.Sprintf("**Completed:** %s\n\n", summary.EndTime.Format(time.RFC3339)))
	md.WriteString(fmt.Sprintf("**Duration:** %s\n\n", summary.Duration))

	// Result
	md.WriteString("## Result\n\n")
	if summary.Error != "" {
		md.WriteString(fmt.Sprintf("❌ **Error:** %s\n\n", summary.Error))
	} else {
		md.WriteString("✅ **Success**\n\n")
	}

	// Files Modified
	if len(summary.FilesModified) > 0 {
		md.WriteString("## Files Modified\n\n")
		for _, file := range summary.FilesModified {
			md.WriteString(fmt.Sprintf("- `%s` (+%d/-%d lines)\n",
				file.Path, file.LinesAdded, file.LinesRemoved))
		}
		md.WriteString("\n")
	}

	// Quality Gates
	if summary.QualityGateResults != nil && len(summary.QualityGateResults.Results) > 0 {
		md.WriteString("## Quality Gates\n\n")
		for _, result := range summary.QualityGateResults.Results {
			status := "✅"
			if !result.Passed {
				status = "❌"
			}
			md.WriteString(fmt.Sprintf("%s **%s**", status, result.Name))
			if result.Required {
				md.WriteString(" (required)")
			}
			md.WriteString("\n")
			if result.Error != "" {
				md.WriteString(fmt.Sprintf("   Error: %s\n", result.Error))
			}
		}
		md.WriteString("\n")
	}

	// Metrics
	md.WriteString("## Metrics\n\n")
	md.WriteString(fmt.Sprintf("- **Files Modified:** %d\n", summary.Metrics.FilesModified))
	md.WriteString(fmt.Sprintf("- **Total Lines Added:** %d\n", summary.Metrics.TotalLinesAdded))
	md.WriteString(fmt.Sprintf("- **Total Lines Removed:** %d\n", summary.Metrics.TotalLinesRemoved))
	md.WriteString(fmt.Sprintf("- **Tokens Used:** %d\n", summary.Metrics.TokensUsed))
	md.WriteString(fmt.Sprintf("- **Iterations:** %d\n", summary.Metrics.Iterations))

	// Write file
	if writeErr := os.WriteFile(path, []byte(md.String()), 0600); writeErr != nil {
		return fmt.Errorf("failed to write summary markdown: %w", writeErr)
	}

	return nil
}

// WriteMetricsJSON writes execution metrics as JSON
func (w *ArtifactWriter) WriteMetricsJSON(summary *ExecutionSummary) error {
	path := filepath.Join(w.outputDir, "metrics.json")

	data, err := json.MarshalIndent(summary.Metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if writeErr := os.WriteFile(path, data, 0600); writeErr != nil {
		return fmt.Errorf("failed to write metrics JSON: %w", writeErr)
	}

	return nil
}

// ExecutionSummary contains a complete summary of headless execution
type ExecutionSummary struct {
	Task               string              `json:"task"`
	Status             string              `json:"status"`
	Error              string              `json:"error,omitempty"`
	StartTime          time.Time           `json:"start_time"`
	EndTime            time.Time           `json:"end_time"`
	Duration           time.Duration       `json:"duration"`
	FilesModified      []FileModification  `json:"files_modified"`
	QualityGateResults *QualityGateResults `json:"quality_gate_results,omitempty"`
	Metrics            ExecutionMetrics    `json:"metrics"`
	GitInfo            *GitInfo            `json:"git_info,omitempty"`
	ToolCallCount      int                 `json:"tool_call_count"`
}

// ExecutionMetrics contains execution metrics
type ExecutionMetrics struct {
	FilesModified     int `json:"files_modified"`
	TotalLinesAdded   int `json:"total_lines_added"`
	TotalLinesRemoved int `json:"total_lines_removed"`
	TokensUsed        int `json:"tokens_used"`
	Iterations        int `json:"iterations"`
}

// GitInfo contains git-related information
type GitInfo struct {
	Branch        string `json:"branch,omitempty"`
	CommitHash    string `json:"commit_hash,omitempty"`
	CommitMessage string `json:"commit_message,omitempty"`
}
