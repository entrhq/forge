package headless

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// QualityGate represents a validation step to run before committing changes
type QualityGate interface {
	// Name returns the name of the quality gate
	Name() string

	// Required returns true if failure should abort execution
	Required() bool

	// Execute runs the quality gate and returns an error if it fails
	Execute(ctx context.Context, workspaceDir string) error
}

// CommandQualityGate executes a shell command as a quality gate
type CommandQualityGate struct {
	name     string
	command  string
	required bool
	timeout  time.Duration
}

// NewCommandQualityGate creates a new command-based quality gate
func NewCommandQualityGate(name, command string, required bool) *CommandQualityGate {
	return &CommandQualityGate{
		name:     name,
		command:  command,
		required: required,
		timeout:  5 * time.Minute, // Default timeout
	}
}

// Name returns the name of the quality gate
func (g *CommandQualityGate) Name() string {
	return g.name
}

// Required returns true if failure should abort execution
func (g *CommandQualityGate) Required() bool {
	return g.required
}

// Execute runs the quality gate command
func (g *CommandQualityGate) Execute(ctx context.Context, workspaceDir string) error {
	// Create context with timeout
	execCtx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()

	// Parse command into parts
	parts := strings.Fields(g.command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	// Create command
	cmd := exec.CommandContext(execCtx, parts[0], parts[1:]...)
	cmd.Dir = workspaceDir

	// Execute command and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &QualityGateError{
			GateName: g.name,
			Command:  g.command,
			Output:   string(output),
			Err:      err,
		}
	}

	return nil
}

// QualityGateError represents a quality gate execution failure
type QualityGateError struct {
	GateName string
	Command  string
	Output   string
	Err      error
}

func (e *QualityGateError) Error() string {
	return fmt.Sprintf("quality gate '%s' failed: %v", e.GateName, e.Err)
}

// Unwrap returns the underlying error
func (e *QualityGateError) Unwrap() error {
	return e.Err
}

// QualityGateRunner manages execution of multiple quality gates
type QualityGateRunner struct {
	gates []QualityGate
}

// NewQualityGateRunner creates a new quality gate runner
func NewQualityGateRunner(gates []QualityGate) *QualityGateRunner {
	return &QualityGateRunner{gates: gates}
}

// RunAll executes all quality gates and returns results
func (r *QualityGateRunner) RunAll(ctx context.Context, workspaceDir string) *QualityGateResults {
	results := &QualityGateResults{
		Results: make([]QualityGateResult, 0, len(r.gates)),
	}

	for _, gate := range r.gates {
		result := QualityGateResult{
			Name:     gate.Name(),
			Required: gate.Required(),
		}

		// Execute gate
		err := gate.Execute(ctx, workspaceDir)
		if err != nil {
			result.Passed = false
			result.Error = err.Error()

			// If gate is required and failed, mark overall failure
			if gate.Required() {
				results.AllPassed = false
			}
		} else {
			result.Passed = true
		}

		results.Results = append(results.Results, result)
	}

	// If no required gates failed, mark as passed
	switch {
	case !results.AllPassed && len(results.Results) > 0:
		// Check if any required gate exists
		hasRequired := false
		for _, result := range results.Results {
			if result.Required {
				hasRequired = true
				if !result.Passed {
					results.AllPassed = false
					return results
				}
			}
		}
		if hasRequired {
			results.AllPassed = true
		}
	case len(results.Results) == 0:
		results.AllPassed = true
	default:
		// Check if all gates passed
		allPassed := true
		for _, result := range results.Results {
			if result.Required && !result.Passed {
				allPassed = false
				break
			}
		}
		results.AllPassed = allPassed
	}

	return results
}

// QualityGateResults contains results from running quality gates
type QualityGateResults struct {
	AllPassed bool
	Results   []QualityGateResult
}

// QualityGateResult represents the result of a single quality gate
type QualityGateResult struct {
	Name     string
	Required bool
	Passed   bool
	Error    string
}

// GetFailedGates returns a list of failed required gates
func (r *QualityGateResults) GetFailedGates() []QualityGateResult {
	failed := make([]QualityGateResult, 0)
	for _, result := range r.Results {
		if result.Required && !result.Passed {
			failed = append(failed, result)
		}
	}
	return failed
}

// FormatErrorMessage creates a formatted error message for failed gates
func (r *QualityGateResults) FormatErrorMessage() string {
	failed := r.GetFailedGates()
	if len(failed) == 0 {
		return ""
	}

	var msg strings.Builder
	msg.WriteString("Quality gate failures:\n\n")
	for _, result := range failed {
		msg.WriteString(fmt.Sprintf("❌ %s\n", result.Name))
		if result.Error != "" {
			msg.WriteString(fmt.Sprintf("   Error: %s\n\n", result.Error))
		}
	}

	return msg.String()
}

// FormatFeedbackMessage creates a feedback message for the agent to fix quality gate failures
func (r *QualityGateResults) FormatFeedbackMessage(retryCount, maxRetries int) string {
	failed := r.GetFailedGates()
	if len(failed) == 0 {
		return ""
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("Your previous task completion failed quality gates (attempt %d/%d).\n\n", retryCount, maxRetries))
	msg.WriteString("Please review and fix the following issues:\n\n")

	for _, result := range failed {
		msg.WriteString(fmt.Sprintf("❌ %s\n", result.Name))
		if result.Error != "" {
			// Extract useful error information from QualityGateError
			msg.WriteString(fmt.Sprintf("   %s\n\n", result.Error))
		}
	}

	msg.WriteString("\nAfter fixing these issues, use task_completion again to revalidate your changes.")

	log.Printf("Quality gate failed, sending message to agent:\n%s", msg.String())
	return msg.String()
}

// CreateQualityGates creates quality gates from configuration
func CreateQualityGates(configs []QualityGateConfig) []QualityGate {
	gates := make([]QualityGate, 0, len(configs))
	for _, config := range configs {
		gate := NewCommandQualityGate(config.Name, config.Command, config.Required)
		gates = append(gates, gate)
	}
	return gates
}
