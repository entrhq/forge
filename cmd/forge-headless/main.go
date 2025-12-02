// Package main provides the Forge headless executor for CI/CD automation.
// This enables Forge to run autonomously in non-interactive environments
// with safety constraints, quality gates, and automatic git operations.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/entrhq/forge/pkg/agent"
	agentcontext "github.com/entrhq/forge/pkg/agent/context"
	"github.com/entrhq/forge/pkg/agent/memory/notes"
	"github.com/entrhq/forge/pkg/agent/tools"
	appconfig "github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/executor/headless"
	"github.com/entrhq/forge/pkg/llm/openai"
	"github.com/entrhq/forge/pkg/security/workspace"
	"github.com/entrhq/forge/pkg/tools/coding"
	"github.com/entrhq/forge/pkg/tools/scratchpad"
	"gopkg.in/yaml.v3"
)

const (
	version      = "0.1.0"
	defaultModel = "anthropic/claude-sonnet-4.5"

	// Context management defaults for headless execution
	defaultMaxTokens        = 100000 // Conservative limit with headroom for 128K context
	defaultThresholdPercent = 80.0   // Start summarizing at 80% (80K tokens)
	defaultToolCallAge      = 20     // Tool calls must be 20+ messages old to enter buffer
	defaultMinToolCalls     = 10     // Minimum 10 tool calls in buffer before summarizing
	defaultMaxToolCallDist  = 40     // Force summarization if any tool call is 40+ messages old
	defaultSummaryBatchSize = 10     // Summarize 10 messages at a time
)

// CLIConfig holds command-line configuration
type CLIConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	ConfigFile  string
	Task        string
	Workspace   string
	Mode        string
	Timeout     time.Duration
	OutputFile  string
	ShowVersion bool
}

func main() {
	// Parse command line flags
	config := parseFlags()

	// Show version if requested
	if config.ShowVersion {
		fmt.Printf("Forge Headless v%s\n", version)
		return
	}

	// Create context with signal handling
	ctx, cancel := context.WithCancel(context.Background())

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\nShutting down gracefully...")
		cancel()
	}()

	// Run the headless executor
	if err := run(ctx, config); err != nil {
		cancel() // Cancel context before exiting
		log.Printf("Execution failed: %v", err)
		os.Exit(1)
	}
	cancel() // Clean up context on success
}

// parseFlags parses command line flags
func parseFlags() *CLIConfig {
	config := &CLIConfig{}

	flag.StringVar(&config.APIKey, "api-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key")
	flag.StringVar(&config.BaseURL, "base-url", os.Getenv("OPENAI_BASE_URL"), "OpenAI API base URL")
	flag.StringVar(&config.Model, "model", defaultModel, "LLM model to use")
	flag.StringVar(&config.ConfigFile, "config", "", "Path to configuration file (YAML)")
	flag.StringVar(&config.Task, "task", "", "Task description (required if no config file)")
	flag.StringVar(&config.Workspace, "workspace", ".", "Workspace directory")
	flag.StringVar(&config.Mode, "mode", "write", "Execution mode: read-only or write")
	flag.DurationVar(&config.Timeout, "timeout", 5*time.Minute, "Execution timeout")
	flag.StringVar(&config.OutputFile, "output", "execution-summary.json", "Output file for execution summary")
	flag.BoolVar(&config.ShowVersion, "version", false, "Show version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Forge Headless - Autonomous Coding Agent for CI/CD\n\n")
		fmt.Fprintf(os.Stderr, "Usage: forge-headless [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Run with inline task\n")
		fmt.Fprintf(os.Stderr, "  forge-headless -task \"Fix all linting errors\"\n\n")
		fmt.Fprintf(os.Stderr, "  # Run with config file\n")
		fmt.Fprintf(os.Stderr, "  forge-headless -config forge-headless.yaml\n\n")
		fmt.Fprintf(os.Stderr, "  # Read-only mode\n")
		fmt.Fprintf(os.Stderr, "  forge-headless -task \"Analyze code coverage\" -mode read-only\n\n")
	}

	flag.Parse()
	return config
}

// run executes the headless mode
//
//nolint:gocyclo
func run(ctx context.Context, cliConfig *CLIConfig) error {
	// Load or create execution configuration
	execConfig, err := loadConfig(cliConfig)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if validationErr := execConfig.Validate(); validationErr != nil {
		return fmt.Errorf("invalid configuration: %w", validationErr)
	}

	// Initialize global configuration
	if initErr := appconfig.Initialize(""); initErr != nil {
		return fmt.Errorf("failed to initialize configuration: %w", initErr)
	}

	// Determine final LLM configuration (CLI args override config file)
	finalModel := cliConfig.Model
	finalBaseURL := cliConfig.BaseURL
	finalAPIKey := cliConfig.APIKey

	// Override with config file values if CLI args are not provided
	if llmConfig := appconfig.GetLLM(); llmConfig != nil {
		if cliConfig.Model == defaultModel {
			// CLI used default, check config
			if configModel := llmConfig.GetModel(); configModel != "" {
				finalModel = configModel
			}
		}
		if cliConfig.BaseURL == "" {
			// CLI didn't specify base URL, check config
			if configBaseURL := llmConfig.GetBaseURL(); configBaseURL != "" {
				finalBaseURL = configBaseURL
			}
		}
		if cliConfig.APIKey == "" {
			// CLI didn't specify API key, check config
			if configAPIKey := llmConfig.GetAPIKey(); configAPIKey != "" {
				finalAPIKey = configAPIKey
			}
		}
	}

	// Create LLM provider with final configuration
	providerOpts := []openai.ProviderOption{
		openai.WithModel(finalModel),
	}

	if finalBaseURL != "" {
		providerOpts = append(providerOpts, openai.WithBaseURL(finalBaseURL))
	}

	provider, err := openai.NewProvider(finalAPIKey, providerOpts...)
	if err != nil {
		return fmt.Errorf("failed to create LLM provider: %w", err)
	}

	// Create context manager for long-running autonomous tasks
	toolCallStrategy := agentcontext.NewToolCallSummarizationStrategy(
		defaultToolCallAge,
		defaultMinToolCalls,
		defaultMaxToolCallDist,
	)

	thresholdStrategy := agentcontext.NewThresholdSummarizationStrategy(
		defaultThresholdPercent,
		defaultSummaryBatchSize,
	)

	contextManager, err := agentcontext.NewManager(
		provider,
		defaultMaxTokens,
		toolCallStrategy,
		thresholdStrategy,
	)
	if err != nil {
		return fmt.Errorf("failed to create context manager: %w", err)
	}

	// Create workspace security guard
	guard, err := workspace.NewGuard(execConfig.WorkspaceDir)
	if err != nil {
		return fmt.Errorf("failed to create workspace guard: %w", err)
	}

	// Compose the headless system prompt with mode-specific guidance
	systemPrompt := composeHeadlessSystemPrompt(execConfig.Mode)

	// Create notes manager for scratchpad
	notesManager := notes.NewManager()

	// Create agent with headless system prompt, disabled interactive tools, context management, and shared notes manager
	ag := agent.NewDefaultAgent(
		provider,
		agent.WithCustomInstructions(systemPrompt),
		agent.WithDisabledTools("ask_question", "converse"),
		agent.WithContextManager(contextManager),
		agent.WithNotesManager(notesManager),
	)

	// Register coding tools with workspace guard, filtered by constraints
	codingTools := []tools.Tool{
		coding.NewReadFileTool(guard),
		coding.NewWriteFileTool(guard),
		coding.NewListFilesTool(guard),
		coding.NewSearchFilesTool(guard),
		coding.NewApplyDiffTool(guard),
		coding.NewExecuteCommandTool(guard),
	}

	for _, tool := range codingTools {
		// Filter tools based on allowed_tools constraint
		if !execConfig.Constraints.ShouldRegisterTool(tool.Name()) {
			continue
		}
		if regErr := ag.RegisterTool(tool); regErr != nil {
			return fmt.Errorf("failed to register tool: %w", regErr)
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
		if regErr := ag.RegisterTool(tool); regErr != nil {
			return fmt.Errorf("failed to register scratchpad tool: %w", regErr)
		}
	}

	// Create headless executor with configured agent
	executor, err := headless.NewExecutor(ag, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Apply timeout if specified
	if cliConfig.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cliConfig.Timeout)
		defer cancel()
	}

	// Run execution
	log.Printf("Starting headless execution...")
	log.Printf("Task: %s", execConfig.Task)
	log.Printf("Mode: %s", execConfig.Mode)
	log.Printf("Workspace: %s", execConfig.WorkspaceDir)

	if err := executor.Run(ctx); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	log.Printf("Execution completed successfully")
	return nil
}

// loadConfig loads execution configuration from file or CLI arguments
func loadConfig(cliConfig *CLIConfig) (*headless.Config, error) {
	// If config file is provided, load from file
	if cliConfig.ConfigFile != "" {
		return loadConfigFromFile(cliConfig.ConfigFile)
	}

	// Otherwise, create config from CLI arguments
	if cliConfig.Task == "" {
		return nil, fmt.Errorf("task is required when not using a config file")
	}

	config := headless.DefaultConfig()
	config.Task = cliConfig.Task
	config.WorkspaceDir = cliConfig.Workspace
	config.Constraints.Timeout = cliConfig.Timeout

	// Set execution mode
	switch cliConfig.Mode {
	case "read-only":
		config.Mode = headless.ModeReadOnly
	case "write":
		config.Mode = headless.ModeWrite
	default:
		return nil, fmt.Errorf("invalid mode: %s (must be 'read-only' or 'write')", cliConfig.Mode)
	}

	return config, nil
}

// loadConfigFromFile loads configuration from a YAML file
func loadConfigFromFile(path string) (*headless.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := headless.DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Store the config file path so it can be excluded from commits
	config.ConfigFilePath = path

	return config, nil
}
