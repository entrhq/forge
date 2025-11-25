package coding

import (
	"bufio"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/security/workspace"
	"github.com/entrhq/forge/pkg/types"
)

// ExecuteCommandTool executes shell commands in the workspace directory
type ExecuteCommandTool struct {
	guard          *workspace.Guard
	defaultTimeout time.Duration
}

// NewExecuteCommandTool creates a new command execution tool
func NewExecuteCommandTool(guard *workspace.Guard) *ExecuteCommandTool {
	return &ExecuteCommandTool{
		guard:          guard,
		defaultTimeout: 30 * time.Second, // 30 second default timeout
	}
}

// Name returns the tool name
func (t *ExecuteCommandTool) Name() string {
	return "execute_command"
}

// Description returns the tool description
func (t *ExecuteCommandTool) Description() string {
	return "Execute a shell command in the workspace directory. The command runs with a timeout and returns stdout, stderr, and exit code."
}

// Schema returns the tool's JSON schema
func (t *ExecuteCommandTool) Schema() map[string]interface{} {
	return tools.BaseToolSchema(
		map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute",
			},
			"timeout": map[string]interface{}{
				"type":        "number",
				"description": "Command timeout in seconds (default: 30)",
			},
			"working_dir": map[string]interface{}{
				"type":        "string",
				"description": "Working directory relative to workspace (default: workspace root)",
			},
		},
		[]string{"command"},
	)
}

// Execute runs the command with streaming output support
//
//nolint:gocyclo // Complexity is acceptable for command execution logic
func (t *ExecuteCommandTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var input struct {
		XMLName    xml.Name `xml:"arguments"`
		Command    string   `xml:"command"`
		Timeout    float64  `xml:"timeout"`
		WorkingDir string   `xml:"working_dir"`
	}
	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return "", nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Validate required fields
	if input.Command == "" {
		return "", nil, fmt.Errorf("command cannot be empty")
	}

	// Determine timeout
	timeout := t.defaultTimeout
	if input.Timeout > 0 {
		timeout = time.Duration(input.Timeout * float64(time.Second))
	}

	// Determine working directory
	workDir := t.guard.WorkspaceDir()
	if input.WorkingDir != "" {
		// Validate working directory is within workspace
		if validateErr := t.guard.ValidatePath(input.WorkingDir); validateErr != nil {
			return "", nil, fmt.Errorf("invalid working directory: %w", validateErr)
		}

		// Resolve to absolute path
		absWorkDir, resolveErr := t.guard.ResolvePath(input.WorkingDir)
		if resolveErr != nil {
			return "", nil, fmt.Errorf("failed to resolve working directory: %w", resolveErr)
		}
		workDir = absWorkDir
	}

	// Create context with timeout from parent context
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Check if we have an event emitter in context (for streaming support)
	emitEvent := getEventEmitterFromContext(ctx)

	// Generate execution ID for tracking
	execID := fmt.Sprintf("cmd_%d", time.Now().UnixNano())

	// Register this command's cancel function if we have a command registry
	if registry, ok := ctx.Value(CommandRegistryKey).(*sync.Map); ok {
		registry.Store(execID, cancel)
		defer registry.Delete(execID)
	}

	// Emit command execution start event if emitter available
	if emitEvent != nil {
		emitEvent(types.NewCommandExecutionStartEvent(execID, input.Command, workDir))
	}

	// Execute command with streaming
	start := time.Now()
	cmd := exec.CommandContext(execCtx, "sh", "-c", input.Command)
	cmd.Dir = workDir

	var stdout, stderr string
	var exitCode int
	var execErr error

	if emitEvent != nil {
		// Execute with streaming output
		stdout, stderr, exitCode, execErr = t.runCommandStreaming(execCtx, cmd, execID, emitEvent)
	} else {
		// Fall back to non-streaming execution
		stdout, stderr, exitCode, execErr = t.runCommand(cmd)
	}

	duration := time.Since(start)

	// Emit final state event
	if emitEvent != nil {
		durationStr := duration.String()

		if execErr != nil {
			// Check if context was canceled (either by user or timeout)
			if execCtx.Err() == context.Canceled || execCtx.Err() == context.DeadlineExceeded {
				emitEvent(types.NewCommandExecutionCanceledEvent(execID, durationStr))
			} else {
				emitEvent(types.NewCommandExecutionFailedEvent(execID, exitCode, durationStr, execErr))
			}
		} else {
			emitEvent(types.NewCommandExecutionCompleteEvent(execID, exitCode, durationStr))
		}
	}

	// Format response
	var result string
	if execErr != nil {
		// Check if context was canceled or timed out
		switch execCtx.Err() {
		case context.Canceled:
			result = fmt.Sprintf("Command was canceled by user after %s\n\nThe command execution was interrupted before completion.\n\nStdout:\n%s\n\nStderr:\n%s",
				duration.String(), stdout, stderr)
		case context.DeadlineExceeded:
			result = fmt.Sprintf("Command timed out after %s\n\nStdout:\n%s\n\nStderr:\n%s",
				duration.String(), stdout, stderr)
		default:
			result = fmt.Sprintf("Command failed with exit code %d\n\nStdout:\n%s\n\nStderr:\n%s",
				exitCode, stdout, stderr)
		}
	} else {
		result = fmt.Sprintf("Command completed successfully in %s\n\nStdout:\n%s",
			duration.String(), stdout)
		if stderr != "" {
			result += fmt.Sprintf("\n\nStderr:\n%s", stderr)
		}
	}

	// Add exit code info
	result += fmt.Sprintf("\n\nExit code: %d", exitCode)

	// Build metadata
	metadata := map[string]interface{}{
		"command":     input.Command,
		"exit_code":   exitCode,
		"duration_ms": duration.Milliseconds(),
		"working_dir": workDir,
	}

	return result, metadata, nil
}

// runCommand executes the command and captures output
func (t *ExecuteCommandTool) runCommand(cmd *exec.Cmd) (stdout, stderr string, exitCode int, err error) {
	stdoutBytes, stderrBytes, err := t.captureOutput(cmd)
	stdout = string(stdoutBytes)
	stderr = string(stderrBytes)

	// Get exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start or other error
			exitCode = -1
		}
		return stdout, stderr, exitCode, err
	}

	exitCode = 0
	return stdout, stderr, exitCode, nil
}

// captureOutput runs the command and captures stdout/stderr
func (t *ExecuteCommandTool) captureOutput(cmd *exec.Cmd) (stdout, stderr []byte, err error) {
	stdout, stdoutErr := cmd.Output()
	if stdoutErr != nil {
		// Output() returns stderr in the error if command fails
		if exitErr, ok := stdoutErr.(*exec.ExitError); ok {
			stderr = exitErr.Stderr
		}
		return stdout, stderr, stdoutErr
	}
	return stdout, nil, nil
}

// IsLoopBreaking indicates this tool should not break the agent loop
func (t *ExecuteCommandTool) IsLoopBreaking() bool {
	return false
}

// GeneratePreview implements the Previewable interface to show command details before execution.
func (t *ExecuteCommandTool) GeneratePreview(ctx context.Context, argsXML []byte) (*tools.ToolPreview, error) {
	var input struct {
		XMLName    xml.Name `xml:"arguments"`
		Command    string   `xml:"command"`
		Timeout    float64  `xml:"timeout"`
		WorkingDir string   `xml:"working_dir"`
	}

	if err := tools.UnmarshalXMLWithFallback(argsXML, &input); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	if input.Command == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	// Determine working directory
	workDir := t.guard.WorkspaceDir()
	if input.WorkingDir != "" {
		if validateErr := t.guard.ValidatePath(input.WorkingDir); validateErr != nil {
			return nil, fmt.Errorf("invalid working directory: %w", validateErr)
		}

		absWorkDir, resolveErr := t.guard.ResolvePath(input.WorkingDir)
		if resolveErr != nil {
			return nil, fmt.Errorf("failed to resolve working directory: %w", resolveErr)
		}
		workDir = absWorkDir
	}

	// Determine timeout
	timeout := t.defaultTimeout
	if input.Timeout > 0 {
		timeout = time.Duration(input.Timeout * float64(time.Second))
	}

	// Build preview content
	var preview strings.Builder
	preview.WriteString("Command: ")
	preview.WriteString(input.Command)
	preview.WriteString("\n\n")
	preview.WriteString("Working Directory: ")
	preview.WriteString(workDir)
	preview.WriteString("\n\n")
	preview.WriteString(fmt.Sprintf("Timeout: %s\n", timeout))

	return &tools.ToolPreview{
		Type:        tools.PreviewTypeCommand,
		Title:       "Execute Command",
		Description: fmt.Sprintf("This will execute the command: %s", input.Command),
		Content:     preview.String(),
		Metadata: map[string]interface{}{
			"command":     input.Command,
			"working_dir": workDir,
			"timeout":     timeout.Seconds(),
		},
	}, nil
}

// EventEmitter is a function type for emitting agent events
type EventEmitter func(*types.AgentEvent)

// ContextKey is a type for context keys to avoid collisions
type ContextKey string

// EventEmitterKey is the context key for the event emitter
const EventEmitterKey ContextKey = "event_emitter"

// CommandRegistryKey is the context key for the command registry (sync.Map)
const CommandRegistryKey ContextKey = "command_registry"

// getEventEmitterFromContext retrieves the event emitter from context if available
func getEventEmitterFromContext(ctx context.Context) EventEmitter {
	if emitter, ok := ctx.Value(EventEmitterKey).(EventEmitter); ok {
		return emitter
	}
	return nil
}

// runCommandStreaming executes a command with streaming output support
func (t *ExecuteCommandTool) runCommandStreaming(ctx context.Context, cmd *exec.Cmd, execID string, emitEvent EventEmitter) (stdout, stderr string, exitCode int, err error) {
	// Create pipes for stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", -1, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", "", -1, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", "", -1, fmt.Errorf("failed to start command: %w", err)
	}

	// Use WaitGroup to wait for both goroutines to finish
	var wg sync.WaitGroup
	var stdoutBuilder, stderrBuilder strings.Builder
	var outputMu sync.Mutex

	// Monitor context cancellation and kill process if needed
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			// Context was canceled - kill the process
			if cmd.Process != nil {
				// Use Kill() for immediate termination
				// We intentionally ignore the error here because:
				// 1. The process may have already exited
				// 2. We're in cancellation mode and want to proceed regardless
				//nolint:errcheck
				cmd.Process.Kill()
			}
		case <-done:
			// Command finished normally
		}
	}()

	// Stream stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		t.streamOutput(stdoutPipe, "stdout", execID, emitEvent, &stdoutBuilder, &outputMu)
	}()

	// Stream stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		t.streamOutput(stderrPipe, "stderr", execID, emitEvent, &stderrBuilder, &outputMu)
	}()

	// Wait for command to finish (this closes the pipes)
	execErr := cmd.Wait()

	// Signal that command is done
	close(done)

	// Wait for streaming to complete (now that pipes are closed)
	wg.Wait()

	// Get final output
	outputMu.Lock()
	stdout = stdoutBuilder.String()
	stderr = stderrBuilder.String()
	outputMu.Unlock()

	// Determine exit code
	if execErr != nil {
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
		return stdout, stderr, exitCode, execErr
	}

	return stdout, stderr, 0, nil
}

// streamOutput reads from a pipe and emits chunked output events
func (t *ExecuteCommandTool) streamOutput(pipe io.ReadCloser, streamType, execID string, emitEvent EventEmitter, builder *strings.Builder, mu *sync.Mutex) {
	scanner := bufio.NewScanner(pipe)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text() + "\n"

		// Append to full output
		mu.Lock()
		builder.WriteString(line)
		mu.Unlock()

		// Emit output event
		emitEvent(types.NewCommandOutputEvent(execID, line, streamType))
	}
}
