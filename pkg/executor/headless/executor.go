package headless

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/entrhq/forge/pkg/agent"
	"github.com/entrhq/forge/pkg/types"
)

const (
	statusSuccess = "success"
)

// Executor implements the headless mode executor
type Executor struct {
	agent          agent.Agent
	config         *Config
	constraintMgr  *ConstraintManager
	qualityGates   *QualityGateRunner
	artifactWriter *ArtifactWriter
	gitManager     *GitManager

	// Execution state
	startTime time.Time
	summary   *ExecutionSummary
}

// NewExecutor creates a new headless executor with a pre-configured agent
func NewExecutor(ag agent.Agent, config *Config) (*Executor, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create constraint manager
	constraintMgr, err := NewConstraintManager(config.Constraints)
	if err != nil {
		return nil, fmt.Errorf("failed to create constraint manager: %w", err)
	}

	// Create quality gate runner
	gates := CreateQualityGates(config.QualityGates)
	qualityGateRunner := NewQualityGateRunner(gates)

	// Create artifact writer with workspace-relative path
	artifactOutputDir := filepath.Join(config.WorkspaceDir, config.Artifacts.OutputDir)
	artifactWriter := NewArtifactWriter(artifactOutputDir)

	// Create git manager
	gitManager := NewGitManager(config.WorkspaceDir, config.Git)

	return &Executor{
		agent:          ag,
		config:         config,
		constraintMgr:  constraintMgr,
		qualityGates:   qualityGateRunner,
		artifactWriter: artifactWriter,
		gitManager:     gitManager,
		summary: &ExecutionSummary{
			Task:   config.Task,
			Status: "running",
		},
	}, nil
}

// Run executes the headless task
//
//nolint:gocyclo // TODO: refactor to reduce complexity
func (e *Executor) Run(ctx context.Context) error {
	e.startTime = time.Now()
	e.summary.StartTime = e.startTime

	log.Printf("[Headless] Starting execution: %s", e.config.Task)

	// Validate workspace state
	e.validateWorkspace()

	// Start agent
	if err := e.agent.Start(ctx); err != nil {
		return e.fail(fmt.Errorf("failed to start agent: %w", err))
	}

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, e.config.Constraints.Timeout)
	defer cancel()

	// Get agent channels
	channels := e.agent.GetChannels()

	// Start event consumer in goroutine
	eventDone := make(chan struct{})
	turnEndReceived := false
	fileTracker := NewFileModificationTracker(e.config.Verbose)
	go func() {
		defer close(eventDone)
		for event := range channels.Event {
			// Log all events
			log.Printf("[Headless] Event received: Type=%s", event.Type)

			// Handle approval requests - validate against constraints and auto-approve
			if event.Type == types.EventTypeToolApprovalRequest {
				e.handleApprovalRequest(channels.Approval, event)
			}

			// Track tool calls for metrics and file modifications
			if event.Type == types.EventTypeToolCall {
				e.summary.ToolCallCount++
				log.Printf("[Headless] Tool call event - Name: %s, Count: %d", event.ToolName, e.summary.ToolCallCount)
				log.Printf("[Headless] Tool call input type: %T, value: %+v", event.ToolInput, event.ToolInput)

				// Track potential file modification
				if err := fileTracker.TrackToolCall(event); err != nil {
					log.Printf("[Headless] Error tracking file modification: %v", err)
				}
			}

			// Confirm successful file modifications
			if event.Type == types.EventTypeToolResult {
				log.Printf("[Headless] Tool result event - ToolName: %s", event.ToolName)
				fileTracker.ConfirmModification(event)

				// Sync file modification to constraint manager for metrics tracking
				if event.Metadata != nil {
					if path, ok := event.Metadata["file_path"].(string); ok {
						linesAdded := 0
						linesRemoved := 0
						if la, ok := event.Metadata["lines_added"].(int); ok {
							linesAdded = la
						}
						if lr, ok := event.Metadata["lines_removed"].(int); ok {
							linesRemoved = lr
						}

						if err := e.constraintMgr.RecordFileModification(path, linesAdded, linesRemoved); err != nil {
							log.Printf("[Headless] Constraint violation: %v", err)
							// Don't fail execution, just log the violation
						}
					}
				}
			}

			// Cancel failed file modifications
			if event.Type == types.EventTypeToolResultError {
				fileTracker.CancelModification(event)
			}

			// Track token usage
			if event.Type == types.EventTypeTokenUsage && event.TokenUsage != nil {
				if err := e.constraintMgr.RecordTokenUsage(event.TokenUsage.TotalTokens); err != nil {
					log.Printf("[Headless] Token limit exceeded: %v", err)
				}
			}

			// Track turn end - this signals task completion
			if event.Type == types.EventTypeTurnEnd {
				turnEndReceived = true
				log.Printf("[Headless] Turn end received")
				// Close input channel to signal agent to shut down
				// This causes the agent's event loop to terminate and close Done channel
				close(channels.Input)
			}

			// Log event details if verbose
			if e.config.Verbose {
				log.Printf("[Headless] Event details: %+v", event)
			}
		}
		// Update summary with confirmed file modifications
		e.summary.FilesModified = fileTracker.GetModifiedFiles()
		log.Printf("[Headless] Event consumer finished. Total tool calls: %d, Files modified: %d, Turn end received: %v", e.summary.ToolCallCount, len(e.summary.FilesModified), turnEndReceived)
	}()

	// Send task to agent
	channels.Input <- types.NewUserInput(e.config.Task)

	// Wait for completion or timeout
	select {
	case <-channels.Done:
		log.Printf("[Headless] Agent completed - Done channel closed")
	case <-execCtx.Done():
		if execCtx.Err() == context.DeadlineExceeded {
			return e.fail(fmt.Errorf("execution timeout exceeded"))
		}
		return e.fail(fmt.Errorf("execution canceled: %w", execCtx.Err()))
	}

	log.Printf("[Headless] Waiting for event consumer to finish...")
	// Wait for event consumer to finish processing all events
	<-eventDone
	log.Printf("[Headless] Event consumer finished")

	// Finalize execution
	return e.finalize(ctx)
}

// handleApprovalRequest handles tool approval requests by validating against constraints
// and auto-approving (or rejecting) the tool call
func (e *Executor) handleApprovalRequest(approvalChan chan<- *types.ApprovalResponse, event *types.AgentEvent) {
	approvalID := event.ApprovalID
	if approvalID == "" {
		log.Printf("[Headless] Warning: approval request missing approval_id")
		return
	}

	toolName := event.ToolName
	toolInput := event.ToolInput

	log.Printf("[Headless] Approval request for tool: %s (approval_id: %s)", toolName, approvalID)

	// Validate against constraints
	if err := e.constraintMgr.ValidateToolCall(toolName, toolInput); err != nil {
		log.Printf("[Headless] Tool call rejected due to constraint violation: %v", err)
		// Send rejection response
		approvalChan <- types.NewApprovalResponse(approvalID, types.ApprovalRejected)
		return
	}

	log.Printf("[Headless] Tool call approved: %s", toolName)
	// Send approval response
	approvalChan <- types.NewApprovalResponse(approvalID, types.ApprovalGranted)
}

// Stop gracefully stops the executor
func (e *Executor) Stop(ctx context.Context) error {
	return e.agent.Shutdown(ctx)
}

// validateWorkspace validates the workspace state before execution
func (e *Executor) validateWorkspace() {
	// Check if workspace directory exists
	// TODO: Add directory existence check

	// Check git status if git operations are enabled
	if e.config.Git.AutoCommit {
		// Note: We're being lenient here - workspace doesn't need to be clean
		// We'll handle uncommitted changes appropriately
		log.Printf("[Headless] Git auto-commit enabled")
	}
}

// finalize completes the execution and generates artifacts
func (e *Executor) finalize(ctx context.Context) error {
	e.summary.EndTime = time.Now()
	e.summary.Duration = e.summary.EndTime.Sub(e.summary.StartTime)

	// Get constraint state
	state := e.constraintMgr.GetCurrentState()

	// Calculate total lines from FilesModified
	totalLinesAdded := 0
	totalLinesRemoved := 0
	for _, mod := range e.summary.FilesModified {
		totalLinesAdded += mod.LinesAdded
		totalLinesRemoved += mod.LinesRemoved
	}

	// Don't overwrite FilesModified - we tracked them in the event loop
	// Calculate totals from our tracked files, use constraint manager only for tokens
	e.summary.Metrics = ExecutionMetrics{
		FilesModified:     len(e.summary.FilesModified),
		TotalLinesAdded:   totalLinesAdded,
		TotalLinesRemoved: totalLinesRemoved,
		TokensUsed:        state.TokensUsed,
		Iterations:        e.summary.ToolCallCount, // Each tool call represents one iteration of the agent loop
	}

	// Run quality gates if configured
	if len(e.qualityGates.gates) > 0 {
		log.Printf("[Headless] Running quality gates...")
		results := e.qualityGates.RunAll(ctx, e.config.WorkspaceDir)
		e.summary.QualityGateResults = results

		if !results.AllPassed {
			e.summary.Status = "failed"
			e.summary.Error = results.FormatErrorMessage()
			log.Printf("[Headless] Quality gates failed")

			// Note: Rollback is disabled to preserve work done by the agent
			// The task will fail but changes will be kept for inspection
			log.Printf("[Headless] Rollback disabled - preserving changes for inspection")
		} else {
			e.summary.Status = statusSuccess
			log.Printf("[Headless] Quality gates passed")
		}
	} else {
		e.summary.Status = statusSuccess
	}

	// Commit changes if configured and gates passed
	if e.config.Git.AutoCommit && e.summary.Status == statusSuccess {
		if err := e.commitChanges(ctx); err != nil {
			log.Printf("[Headless] Warning: failed to commit changes: %v", err)
			// Don't fail the execution, just log the warning
		}
	}

	// Generate artifacts if enabled
	if e.config.Artifacts.Enabled {
		if err := e.artifactWriter.WriteAll(e.summary); err != nil {
			log.Printf("[Headless] Warning: failed to write artifacts: %v", err)
		} else {
			log.Printf("[Headless] Artifacts written to %s", e.config.Artifacts.OutputDir)
		}
	}

	// Log final status
	log.Printf("[Headless] Execution completed: %s (duration: %s)", e.summary.Status, e.summary.Duration)

	if e.summary.Status != statusSuccess {
		return fmt.Errorf("execution failed: %s", e.summary.Error)
	}

	return nil
}

// commitChanges creates a git commit with the changes
func (e *Executor) commitChanges(ctx context.Context) error {
	// Generate commit message
	message := e.gitManager.GenerateCommitMessage(ctx, e.config.Task)

	// Create commit
	if err := e.gitManager.Commit(ctx, message); err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	log.Printf("[Headless] Created git commit")
	return nil
}

// fail marks the execution as failed and returns an error
func (e *Executor) fail(err error) error {
	e.summary.Status = "failed"
	e.summary.Error = err.Error()
	e.summary.EndTime = time.Now()
	e.summary.Duration = e.summary.EndTime.Sub(e.startTime)

	// Try to generate artifacts even on failure
	if e.config.Artifacts.Enabled {
		if artifactErr := e.artifactWriter.WriteAll(e.summary); artifactErr != nil {
			log.Printf("[Headless] Warning: failed to write failure artifacts: %v", artifactErr)
		}
	}

	return err
}
