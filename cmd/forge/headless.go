package main

import (
	"context"
	"fmt"
	"log"
	"os"
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

// runHeadless executes the headless mode
//
//nolint:gocyclo
func runHeadless(ctx context.Context, config *Config) error {
	// Load and validate configuration
	execConfig, err := loadAndValidateConfig(config)
	if err != nil {
		return err
	}

	// Initialize global configuration
	if initErr := appconfig.Initialize(""); initErr != nil {
		return fmt.Errorf("failed to initialize configuration: %w", initErr)
	}

	// Determine final LLM configuration (CLI args override config file)
	finalModel := config.Model
	finalBaseURL := config.BaseURL
	finalAPIKey := config.APIKey

	// Override with config file values if CLI args are not provided
	if llmConfig := appconfig.GetLLM(); llmConfig != nil {
		if config.Model == defaultModel {
			// CLI used default, check config
			if configModel := llmConfig.GetModel(); configModel != "" {
				finalModel = configModel
			}
		}
		if config.BaseURL == "" {
			// CLI didn't specify base URL, check config
			if configBaseURL := llmConfig.GetBaseURL(); configBaseURL != "" {
				finalBaseURL = configBaseURL
			}
		}
		if config.APIKey == "" {
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

	// Create context manager for headless execution
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

	// Override with custom prompt if provided
	if config.SystemPrompt != "" {
		systemPrompt = config.SystemPrompt
	}

	// Create notes manager for scratchpad
	notesManager := notes.NewManager()

	// Create agent with headless system prompt, disabled interactive tools, and context management
	ag := agent.NewDefaultAgent(
		provider,
		agent.WithCustomInstructions(systemPrompt),
		agent.WithDisabledTools("ask_question", "converse"),
		agent.WithContextManager(contextManager),
	)

	// Register coding tools with workspace guard
	codingTools := []tools.Tool{
		coding.NewReadFileTool(guard),
		coding.NewWriteFileTool(guard),
		coding.NewListFilesTool(guard),
		coding.NewSearchFilesTool(guard),
		coding.NewApplyDiffTool(guard),
		coding.NewExecuteCommandTool(guard),
	}

	for _, tool := range codingTools {
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

	// Create and run executor
	return runExecutor(ctx, ag, execConfig)
}

// loadAndValidateConfig loads and validates headless configuration
func loadAndValidateConfig(config *Config) (*headless.Config, error) {
	var execConfig *headless.Config
	var err error

	if config.HeadlessConfig != "" {
		execConfig, err = loadHeadlessConfigFromFile(config.HeadlessConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to load headless config: %w", err)
		}
	} else {
		execConfig = headless.DefaultConfig()
		execConfig.WorkspaceDir = config.WorkspaceDir
		return nil, fmt.Errorf("headless mode requires a configuration file (use -headless-config flag)")
	}

	// Override workspace from CLI if provided
	if config.WorkspaceDir != "." {
		execConfig.WorkspaceDir = config.WorkspaceDir
	}

	// Validate configuration
	if validationErr := execConfig.Validate(); validationErr != nil {
		return nil, fmt.Errorf("invalid headless configuration: %w", validationErr)
	}

	return execConfig, nil
}

// loadHeadlessConfigFromFile loads headless configuration from a YAML file
func loadHeadlessConfigFromFile(path string) (*headless.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := headless.DefaultConfig()
	if unmarshalErr := yaml.Unmarshal(data, config); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", unmarshalErr)
	}

	// Store the config file path as-is (relative or absolute)
	// If it's in the workspace, it will be excluded from commits
	config.ConfigFilePath = path

	return config, nil
}

// runExecutor creates and runs the headless executor
func runExecutor(ctx context.Context, ag *agent.DefaultAgent, execConfig *headless.Config) error {
	executor, err := headless.NewExecutor(ag, execConfig)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Apply timeout if configured
	if execConfig.Constraints.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, execConfig.Constraints.Timeout)
		defer cancel()
	}

	// Run execution
	log.Printf("Starting headless execution...")
	log.Printf("Task: %s", execConfig.Task)
	log.Printf("Mode: %s", execConfig.Mode)
	log.Printf("Workspace: %s", execConfig.WorkspaceDir)

	startTime := time.Now()
	if runErr := executor.Run(ctx); runErr != nil {
		return fmt.Errorf("execution failed: %w", runErr)
	}

	duration := time.Since(startTime)
	log.Printf("Execution completed successfully in %s", duration)
	return nil
}

// composeHeadlessSystemPrompt creates a system prompt for headless execution
func composeHeadlessSystemPrompt(mode headless.ExecutionMode) string {
	basePrompt := composeSystemPrompt()

	modeGuidance := ""
	switch mode {
	case headless.ModeReadOnly:
		modeGuidance = `

# HEADLESS MODE: READ-ONLY

You are operating in READ-ONLY headless mode. Your mission is to analyze and report, NOT to modify code.

**CRITICAL CONSTRAINTS:**
- You MUST NOT modify any files (no write_file, no apply_diff)
- You MUST NOT execute commands that modify the workspace
- You MUST only use read operations: read_file, list_files, search_files
- Focus on analysis, documentation, and recommendations

**Your Task:**
Provide a comprehensive analysis report including your findings, observations, and recommendations.
Use task_completion to present your final report when analysis is complete.`

	case headless.ModeWrite:
		modeGuidance = `

# HEADLESS MODE: WRITE (Autonomous)

You are operating in autonomous WRITE mode. You have full coding permissions but must work within safety constraints.

**CRITICAL INSTRUCTIONS:**
- Work autonomously - no user interaction available (ask_question and converse are disabled)
- Make incremental, well-tested changes
- Verify your changes work before completing
- Use task_completion when you've successfully completed the task
- If you encounter blockers you cannot resolve autonomously, document them in task_completion

**Safety Constraints:**
You are subject to file modification limits and other safety constraints. Work efficiently within these bounds.`
	}

	return basePrompt + modeGuidance
}
