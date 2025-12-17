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
	"syscall"

	"github.com/entrhq/forge/pkg/agent"
	agentcontext "github.com/entrhq/forge/pkg/agent/context"
	"github.com/entrhq/forge/pkg/agent/memory/notes"
	"github.com/entrhq/forge/pkg/agent/tools"
	appconfig "github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/executor/tui"

	"github.com/entrhq/forge/pkg/security/workspace"
	"github.com/entrhq/forge/pkg/tools/coding"
	"github.com/entrhq/forge/pkg/tools/scratchpad"
)

const (
	version      = "0.1.0"                       // Version of the Forge coding agent
	defaultModel = "anthropic/claude-sonnet-4.5" // Default model to use

	// Context management defaults for coding sessions
	// These are tuned for long coding sessions with many file operations
	defaultMaxTokens        = 100000 // Conservative limit with headroom for 128K context
	defaultThresholdPercent = 80.0   // Start summarizing at 80% (80K tokens)
	defaultToolCallAge      = 20     // Tool calls must be 20+ messages old to enter buffer
	defaultMinToolCalls     = 10     // Minimum 10 tool calls in buffer before summarizing
	defaultMaxToolCallDist  = 40     // Force summarization if any tool call is 40+ messages old
	defaultSummaryBatchSize = 10     // Summarize 10 messages at a time
)

// Config holds the application configuration
type Config struct {
	APIKey         string
	BaseURL        string
	Model          string
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

	flag.StringVar(&config.APIKey, "api-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key (or set OPENAI_API_KEY env var)")
	flag.StringVar(&config.BaseURL, "base-url", os.Getenv("OPENAI_BASE_URL"), "OpenAI API base URL (or set OPENAI_BASE_URL env var)")
	flag.StringVar(&config.Model, "model", defaultModel, "LLM model to use")
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
	return config
}

// validate checks that the configuration is valid
func (c *Config) validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key is required. Set OPENAI_API_KEY environment variable or use -api-key flag")
	}

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

func runTUI(ctx context.Context, config *Config) error {
	// Initialize global configuration (for auto-approval and command whitelist)
	if err := appconfig.Initialize(""); err != nil {
		return fmt.Errorf("failed to initialize configuration: %w", err)
	}

	provider, err := appconfig.BuildProvider(config.Model, config.BaseURL, config.APIKey, defaultModel)
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

	// Strategy 2: Summarize when approaching token limit to prevent exhaustion
	thresholdStrategy := agentcontext.NewThresholdSummarizationStrategy(
		defaultThresholdPercent,
		defaultSummaryBatchSize,
	)

	// Create context manager with both strategies
	// Event channel will be set by the agent during initialization
	contextManager, err := agentcontext.NewManager(
		provider,
		defaultMaxTokens,
		toolCallStrategy,
		thresholdStrategy,
	)
	if err != nil {
		return fmt.Errorf("failed to create context manager: %w", err)
	}

	// Compose the system prompt
	systemPrompt := composeSystemPrompt()
	if config.SystemPrompt != "" {
		systemPrompt = config.SystemPrompt // Override with user-provided prompt
	}

	// Create workspace security guard
	guard, err := workspace.NewGuard(config.WorkspaceDir)
	if err != nil {
		return fmt.Errorf("failed to create workspace guard: %w", err)
	}

	// Create notes manager for scratchpad
	notesManager := notes.NewManager()

	// Create agent with custom system prompt, context manager, and shared notes manager
	ag := agent.NewDefaultAgent(
		provider,
		agent.WithCustomInstructions(systemPrompt),
		agent.WithContextManager(contextManager),
		agent.WithNotesManager(notesManager),
	)

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

	// Create TUI executor with provider and workspace for git operations
	executor := tui.NewExecutor(ag, provider, config.WorkspaceDir, "forge")

	// Display welcome message
	fmt.Printf("Forge v%s - Coding Agent\n", version)
	fmt.Printf("Workspace: %s\n", config.WorkspaceDir)
	fmt.Printf("Model: %s\n", config.Model)
	fmt.Println("\nStarting TUI...")
	fmt.Println()

	// Run the executor
	if err := executor.Run(ctx); err != nil {
		return fmt.Errorf("executor error: %w", err)
	}

	return nil
}
