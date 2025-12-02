// Package tui provides a terminal user interface executor for Forge agents,
// offering an interactive, Gemini-style interface for conversations.
//
// The TUI codebase is split into multiple files for better organization:
// - executor.go: Main executor implementation and program lifecycle
// - model.go: Core model structure and state
// - init.go: Initialization logic
// - update.go: Bubble Tea Update function and message handling
// - view.go: Bubble Tea View function and rendering
// - events.go: Agent event processing
// - helpers.go: Utility functions
// - styles.go: Color schemes and styling
package tui

import (
	"context"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrhq/forge/pkg/agent"
	"github.com/entrhq/forge/pkg/agent/git"
	"github.com/entrhq/forge/pkg/agent/slash"
	"github.com/entrhq/forge/pkg/config"
	"github.com/entrhq/forge/pkg/llm"
)

// Executor is a TUI-based executor that provides an interactive,
// Gemini-style interface for agent interaction.
type Executor struct {
	agent        agent.Agent
	program      *tea.Program
	provider     llm.Provider
	workspaceDir string
	header       string // Custom ASCII art header (optional)
}

// NewExecutor creates a new TUI executor for the given agent.
// The headerText will be automatically converted to ASCII art for display.
func NewExecutor(agent agent.Agent, provider llm.Provider, workspaceDir string, headerText string) *Executor {
	return &Executor{
		agent:        agent,
		provider:     provider,
		workspaceDir: workspaceDir,
		header:       headerText,
	}
}

// Run starts the TUI executor and blocks until the user exits.
func (e *Executor) Run(ctx context.Context) error {
	// Initialize debug logging first
	initDebugLog()
	debugLog.Printf("TUI Executor starting...")

	// Start the agent first
	if err := e.agent.Start(ctx); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}
	debugLog.Printf("Agent started successfully")

	// Discover tools from agent and populate config
	if err := config.DiscoverToolsFromAgent(e.agent); err != nil {
		// Log error but don't fail - config system is optional
		log.Printf("Warning: failed to discover tools from agent: %v", err)
		debugLog.Printf("Warning: failed to discover tools from agent: %v", err)
	}

	m := initialModel()
	m.agent = e.agent
	m.channels = e.agent.GetChannels()
	m.provider = e.provider
	m.workspaceDir = e.workspaceDir
	m.header = e.header
	debugLog.Printf("Model initialized, workspace: %s", e.workspaceDir)

	// Initialize slash handler for git operations
	if e.provider != nil && e.workspaceDir != "" {
		llmClient := newLLMAdapter(e.provider)
		tracker := git.NewModificationTracker()
		m.commitGen = git.NewCommitMessageGenerator(llmClient)
		m.prGen = git.NewPRGenerator(llmClient)
		m.slashHandler = slash.NewHandler(e.workspaceDir, tracker, m.commitGen, m.prGen)
	}

	e.program = tea.NewProgram(
		&m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	go func() {
		// Listen for agent events and forward them to the TUI
		for event := range m.channels.Event {
			debugLog.Printf("Forwarding agent event to TUI: %T - %+v", event, event)
			e.program.Send(event)
		}
	}()

	if _, err := e.program.Run(); err != nil {
		return fmt.Errorf("failed to run TUI program: %w", err)
	}

	return nil
}
