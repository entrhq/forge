// Package main provides the Forge TUI coding agent application.
// This is a flagship coding assistant that runs entirely in the terminal,
// providing file operations, code editing, and command execution with an
// intuitive chat-first interface.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/entrhq/forge/pkg/agent"
	agentcontext "github.com/entrhq/forge/pkg/agent/context"
	"github.com/entrhq/forge/pkg/agent/memory/notes"
	"github.com/entrhq/forge/pkg/agent/tools"
	appconfig "github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/executor/tui"

	"github.com/entrhq/forge/pkg/security/workspace"
	"github.com/entrhq/forge/pkg/tools/browser"
	"github.com/entrhq/forge/pkg/tools/coding"
	"github.com/entrhq/forge/pkg/tools/custom"
	"github.com/entrhq/forge/pkg/tools/scratchpad"
)

const (
	version      = "0.1.0"                       // Version of the Forge coding agent
	defaultModel = "anthropic/claude-sonnet-4.5" // Default model to use

	// Context management defaults for coding sessions
	// These are tuned for long coding sessions with many file operations
	defaultMaxTokens       = 100000 // Conservative limit with headroom for 128K context
	defaultToolCallAge     = 20     // Tool calls must be 20+ messages old to enter buffer
	defaultMinToolCalls    = 10     // Minimum 10 tool calls in buffer before summarizing
	defaultMaxToolCallDist = 40     // Force summarization if any tool call is 40+ messages old

	// Goal-batch compaction defaults (strategy 3: compact old completed turns into goal-batch blocks)
	defaultGoalBatchTurnsOld = 20 // Turns must be 20+ messages old to be eligible for compaction
	defaultGoalBatchMinTurns = 3  // Minimum 3 complete turns before triggering compaction
	defaultGoalBatchMaxTurns = 6  // Compact at most 6 turns per LLM call

	// Threshold summarization defaults (strategy 2: half-compaction when context is near full)
	// Summarizes the older half of the conversation into a single summary, keeping the recent half verbatim.
	defaultThresholdTrigger = 80.0 // Fire when context usage reaches 80% of the token limit
)

// Config holds the application configuration
type Config struct {
	APIKey         *string // Pointer to distinguish "not set" from "set to empty"
	BaseURL        *string // Pointer to distinguish "not set" from "set to empty"
	Model          *string // Pointer to distinguish "not set" from "set to default"
	WorkspaceDir   string
	SystemPrompt   string
	ShowVersion    bool
	Headless       bool
	HeadlessConfig string
}

func main() {
	// Parse command line flags
	config := parseFlags()

	// Show version if requested
	if config.ShowVersion {
		fmt.Printf("Forge v%s\n", version)
		return
	}

	// Validate configuration
	if err := config.validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Create context with signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nShutting down gracefully...")
		cancel()
	}()

	// Run the application
	if runErr := run(ctx, config); runErr != nil {
		cancel()
		log.Fatalf("Application error: %v", runErr)
	}
}

// parseFlags parses command line flags and environment variables
func parseFlags() *Config {
	config := &Config{}

	// Use temporary variables for flags
	var apiKey, baseURL, model string

	flag.StringVar(&apiKey, "api-key", "", "OpenAI API key (or set OPENAI_API_KEY env var)")
	flag.StringVar(&baseURL, "base-url", "", "OpenAI API base URL (or set OPENAI_BASE_URL env var)")
	flag.StringVar(&model, "model", "", "LLM model to use")
	flag.StringVar(&config.WorkspaceDir, "workspace", ".", "Workspace directory (default: current directory)")
	flag.StringVar(&config.SystemPrompt, "prompt", "", "Custom instructions for the agent (optional, overrides default)")
	flag.BoolVar(&config.ShowVersion, "version", false, "Show version and exit")
	flag.BoolVar(&config.Headless, "headless", false, "Run in headless mode (non-interactive)")
	flag.StringVar(&config.HeadlessConfig, "headless-config", "", "Path to headless mode configuration file (YAML)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Forge - A TUI coding agent\n\n")
		fmt.Fprintf(os.Stderr, "Usage: forge [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  OPENAI_API_KEY     OpenAI API key\n")
		fmt.Fprintf(os.Stderr, "  OPENAI_BASE_URL    OpenAI API base URL (for compatible APIs)\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # TUI Mode (default)\n")
		fmt.Fprintf(os.Stderr, "  forge                                    # Start in current directory\n")
		fmt.Fprintf(os.Stderr, "  forge -workspace /path/to/project\n")
		fmt.Fprintf(os.Stderr, "  forge -model gpt-4-turbo\n")
		fmt.Fprintf(os.Stderr, "  forge -base-url https://api.openrouter.ai/api/v1\n")
		fmt.Fprintf(os.Stderr, "\n  # Headless Mode (CI/CD)\n")
		fmt.Fprintf(os.Stderr, "  forge -headless -headless-config config.yaml\n")
		fmt.Fprintf(os.Stderr, "  forge -headless -headless-config config.yaml -workspace /path/to/project\n")
	}

	flag.Parse()

	// Convert flag values to pointers only if they were explicitly set
	// Check if flag was visited (explicitly set by user)
	flagWasSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		flagWasSet[f.Name] = true
	})

	if flagWasSet["api-key"] {
		config.APIKey = &apiKey
	}
	if flagWasSet["base-url"] {
		config.BaseURL = &baseURL
	}
	if flagWasSet["model"] {
		config.Model = &model
	}

	return config
}

// validate checks that the configuration is valid
func (c *Config) validate() error {
	// Note: We no longer validate API key here since it will be resolved from
	// CLI flags -> Environment variables -> Config file in BuildProvider

	// Headless mode requires config file
	if c.Headless && c.HeadlessConfig == "" {
		return fmt.Errorf("headless mode requires a configuration file (use -headless-config flag)")
	}

	// Verify workspace directory exists (unless using headless config which will be validated later)
	if !c.Headless || c.WorkspaceDir != "." {
		info, err := os.Stat(c.WorkspaceDir)
		if err != nil {
			return fmt.Errorf("workspace directory error: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("workspace path '%s' is not a directory", c.WorkspaceDir)
		}
	}

	return nil
}

// run executes the main application logic
func run(ctx context.Context, config *Config) error {
	// Check if headless mode is requested
	if config.Headless {
		return runHeadless(ctx, config)
	}

	// Run TUI mode (default)
	return runTUI(ctx, config)
}

// runTUI executes the TUI mode
//
//nolint:gocyclo
func runTUI(ctx context.Context, config *Config) error {
	// Initialize global configuration (for auto-approval and command whitelist)
	if err := appconfig.Initialize(""); err != nil {
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}

	// Resolve LLM configuration with proper precedence:
	// CLI flags -> Environment variables -> Config file -> Defaults
	var cliModel, cliBaseURL, cliAPIKey string
	if config.Model != nil {
		cliModel = *config.Model
	}
	if config.BaseURL != nil {
		cliBaseURL = *config.BaseURL
	}
	if config.APIKey != nil {
		cliAPIKey = *config.APIKey
	}

	provider, err := appconfig.BuildProvider(cliModel, cliBaseURL, cliAPIKey, defaultModel)
	if err != nil {
		return err
	}

	// Create context summarization strategies for long coding sessions
	// Strategy 1: Summarize old tool calls to compress historical operations (with buffering)
	toolCallStrategy := agentcontext.NewToolCallSummarizationStrategy(
		defaultToolCallAge,
		defaultMinToolCalls,
		defaultMaxToolCallDist,
	)

	// Strategy 2: Collapse the older half of the conversation into a single summary when
	// context usage crosses the threshold. The recent half is always kept verbatim.
	thresholdStrategy := agentcontext.NewThresholdSummarizationStrategy(
		defaultThresholdTrigger,
	)

	// Strategy 3: Compact old completed turns (user message + summaries) into goal-batch blocks
	goalBatchStrategy := agentcontext.NewGoalBatchCompactionStrategy(
		defaultGoalBatchTurnsOld,
		defaultGoalBatchMinTurns,
		defaultGoalBatchMaxTurns,
	)

	// Create context manager with active strategies
	// Event channel will be set by the agent during initialization
	contextManager, err := agentcontext.NewManager(
		provider,
		defaultMaxTokens,
		toolCallStrategy,
		thresholdStrategy,
		goalBatchStrategy,
	)
	if err != nil {
		return fmt.Errorf("failed to create context manager: %w", err)
	}

	// Apply summarization model override from config (no-op if not configured)
	if llmCfg := appconfig.GetLLM(); llmCfg != nil {
		if summarizationModel := llmCfg.GetSummarizationModel(); summarizationModel != "" {
			contextManager.SetSummarizationModel(summarizationModel)
		}
	}

	// Create workspace security guard
	guard, err := workspace.NewGuard(config.WorkspaceDir)
	if err != nil {
		return fmt.Errorf("failed to create workspace guard: %w", err)
	}

	// Whitelist custom tools directory for custom tool operations
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	customToolsDir := filepath.Join(homeDir, ".forge", "tools")
	if err := guard.AddWhitelist(customToolsDir); err != nil {
		return fmt.Errorf("failed to whitelist custom tools directory: %w", err)
	}

	// Check for AGENTS.md in workspace root
	agentsMdPath := filepath.Join(config.WorkspaceDir, "AGENTS.md")
	var repositoryContext string
	if content, err := os.ReadFile(agentsMdPath); err == nil {
		repositoryContext = string(content)
		fmt.Printf("Loaded repository context from AGENTS.md\n")
	}

	// Compose the system prompt
	systemPrompt := composeSystemPrompt()
	if config.SystemPrompt != "" {
		systemPrompt = config.SystemPrompt // Override with user-provided prompt
	}

	// Create notes manager for scratchpad
	notesManager := notes.NewManager()

	// Create browser session manager
	browserManager := browser.NewSessionManager()

	// Create agent with custom system prompt, repository context, context manager, and shared notes manager
	agentOptions := []agent.AgentOption{
		agent.WithCustomInstructions(systemPrompt),
		agent.WithContextManager(contextManager),
		agent.WithNotesManager(notesManager),
		agent.WithBrowserManager(browserManager),
	}

	// Add repository context if AGENTS.md was loaded
	if repositoryContext != "" {
		agentOptions = append(agentOptions, agent.WithRepositoryContext(repositoryContext))
	}

	ag := agent.NewDefaultAgent(provider, agentOptions...)

	// Register coding tools
	codingTools := []tools.Tool{
		coding.NewReadFileTool(guard),
		coding.NewWriteFileTool(guard),
		coding.NewListFilesTool(guard),
		coding.NewSearchFilesTool(guard),
		coding.NewApplyDiffTool(guard),
		coding.NewExecuteCommandTool(guard),
	}

	for _, tool := range codingTools {
		if err := ag.RegisterTool(tool); err != nil {
			return fmt.Errorf("failed to register tool: %w", err)
		}
	}

	// Register scratchpad tools
	scratchpadTools := []tools.Tool{
		scratchpad.NewAddNoteTool(notesManager),
		scratchpad.NewListNotesTool(notesManager),
		scratchpad.NewSearchNotesTool(notesManager),
		scratchpad.NewListTagsTool(notesManager),
		scratchpad.NewScratchNoteTool(notesManager),
		scratchpad.NewUpdateNoteTool(notesManager),
	}

	for _, tool := range scratchpadTools {
		if err := ag.RegisterTool(tool); err != nil {
			return fmt.Errorf("failed to register scratchpad tool: %w", err)
		}
	}

	// Register custom tool management tools
	customTools := []tools.Tool{
		custom.NewCreateCustomToolTool(),
		custom.NewRunCustomToolTool(guard),
	}

	for _, tool := range customTools {
		if err := ag.RegisterTool(tool); err != nil {
			return fmt.Errorf("failed to register custom tool: %w", err)
		}
	}

	// Register browser tools using the browser registry
	browserRegistry := browser.NewToolRegistry(browserManager)
	browserRegistry.SetLLMProvider(provider) // Enable AI-powered browser tools
	browserTools := browserRegistry.RegisterTools()

	for _, tool := range browserTools {
		if err := ag.RegisterTool(tool); err != nil {
			return fmt.Errorf("failed to register browser tool: %w", err)
		}
	}

	// Create TUI executor with provider and workspace for git operations
	executor := tui.NewExecutor(ag, provider, config.WorkspaceDir, "forge")

	// Display welcome message
	fmt.Printf("Forge v%s - Coding Agent\n", version)
	fmt.Printf("Workspace: %s\n", config.WorkspaceDir)
	if config.Model != nil {
		fmt.Printf("Model: %s\n", *config.Model)
	}
	fmt.Println("\nStarting TUI...")
	fmt.Println()

	// Run the executor
	if err := executor.Run(ctx); err != nil {
		return fmt.Errorf("executor error: %w", err)
	}

	return nil
}
