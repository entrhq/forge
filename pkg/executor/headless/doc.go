// Package headless implements the headless executor for autonomous CI/CD execution.
//
// The headless executor enables Forge to run completely autonomously in non-interactive
// environments such as CI/CD pipelines, cron jobs, and webhooks. It provides:
//
// - Safety constraints to prevent runaway execution
// - Quality gates for validation before committing changes
// - Git integration for automatic commits
// - Artifact generation for debugging and auditing
//
// Architecture:
//
// The headless executor follows the ADR-0026 architecture, using a mode-aware agent
// with a shared core. Interactive tools (ask_question, converse) are disabled, and
// the executor provides auto-approval with constraint enforcement.
//
//	┌─────────────────────────────────────────────────────────┐
//	│                 Headless Executor                        │
//	│  - Auto-Approval with Constraints                       │
//	│  - Quality Gates                                        │
//	│  - Git Operations                                       │
//	│  - Artifact Generation                                  │
//	└──────────────────┬──────────────────────────────────────┘
//	                   │
//	                   ▼
//	        ┌──────────────────────┐
//	        │   DefaultAgent       │
//	        │ (Interactive tools   │
//	        │  disabled)           │
//	        └──────────────────────┘
//
// Example usage:
//
//	config := headless.DefaultConfig()
//	config.Task = "Fix linting errors in all Go files"
//	config.WorkspaceDir = "/path/to/project"
//
//	provider, _ := openai.NewProvider(apiKey)
//	executor, _ := headless.NewExecutor(provider, config)
//
//	if err := executor.Run(context.Background()); err != nil {
//	    log.Fatal(err)
//	}
//
// Safety Constraints:
//
// The constraint manager enforces safety limits:
// - Maximum number of files that can be modified
// - Maximum total lines changed
// - File pattern allowlists/denylists
// - Tool restrictions
// - Token usage limits
// - Execution timeout
//
// Quality Gates:
//
// Quality gates run after the agent completes its task and before committing changes:
// - Command-based gates (run any validation tool)
// - Required vs. optional gates
// - Automatic rollback on required gate failures
//
// Git Integration:
//
// The git manager handles version control operations:
// - Workspace validation
// - Automatic commits with proper attribution
// - Branch creation (for future PR workflows)
// - Rollback on failures
//
// Artifacts:
//
// The artifact writer generates execution reports:
// - execution.json: Full execution summary
// - summary.md: Human-readable markdown summary
// - metrics.json: Execution metrics
//
// For more details, see:
// - docs/adr/0026-headless-mode-architecture.md
// - docs/adr/0027-safety-constraint-system.md
// - docs/adr/0028-quality-gate-architecture.md
// - docs/adr/0029-headless-git-integration.md
package headless
