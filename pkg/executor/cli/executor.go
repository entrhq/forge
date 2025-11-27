// Package cli provides a command-line executor for Forge agents.
//
// Example usage:
//
//	package main
//
//	import (
//	    "context"
//	    "log"
//	    "os"
//
//	    "github.com/entrhq/forge/pkg/agent"
//	    "github.com/entrhq/forge/pkg/executor/cli"
//	    "github.com/entrhq/forge/pkg/llm/openai"
//	)
//
//	func main() {
//	    provider, _ := openai.NewProvider(
//	        os.Getenv("OPENAI_API_KEY"),
//	        openai.WithModel("gpt-4o"),
//	    )
//
//	    ag := agent.NewDefaultAgent(provider,
//	        agent.WithSystemPrompt("You are helpful."),
//	    )
//
//	    executor := cli.NewExecutor(ag,
//	        cli.WithShowThinking(true),
//	    )
//
//	    if err := executor.Run(context.Background()); err != nil {
//	        log.Fatal(err)
//	    }
//	}
package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/entrhq/forge/pkg/agent"
	"github.com/entrhq/forge/pkg/types"
)

// Executor is a CLI-based executor that enables turn-by-turn conversation
// with an agent through terminal input/output.
type Executor struct {
	agent  agent.Agent
	reader *bufio.Reader
	writer io.Writer

	// Display options
	showThinking bool

	// State tracking
	messageStartPrinted bool
}

// ExecutorOption is a function that configures an Executor.
type ExecutorOption func(*Executor)

// WithShowThinking enables/disables displaying the agent's thinking process.
func WithShowThinking(show bool) ExecutorOption {
	return func(e *Executor) {
		e.showThinking = show
	}
}

// WithWriter sets a custom output writer (default is os.Stdout).
func WithWriter(w io.Writer) ExecutorOption {
	return func(e *Executor) {
		e.writer = w
	}
}

// NewExecutor creates a new CLI executor for the given agent.
func NewExecutor(agent agent.Agent, opts ...ExecutorOption) *Executor {
	e := &Executor{
		agent:        agent,
		reader:       bufio.NewReader(os.Stdin),
		writer:       os.Stdout,
		showThinking: true, // Show thinking by default
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Run starts the executor and begins the conversation loop.
// Returns when the user exits or an error occurs.
func (e *Executor) Run(ctx context.Context) error {
	// Start the agent
	if err := e.agent.Start(ctx); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	channels := e.agent.GetChannels()

	// Start event handler in background
	eventsDone := make(chan struct{})
	turnEnd := make(chan struct{}, 1)
	go e.handleEvents(channels.Event, eventsDone, turnEnd)

	// Print welcome message
	fmt.Fprintln(e.writer, "Forge Agent")
	fmt.Fprintln(e.writer, "Type your message and press Enter. Type 'exit' or 'quit' to end the conversation.")
	fmt.Fprintln(e.writer)

	// Main conversation loop
	for {
		// Check if context is canceled
		select {
		case <-ctx.Done():
			e.shutdown(ctx)
			<-eventsDone
			return ctx.Err()
		default:
		}

		// Read user input
		fmt.Fprint(e.writer, "> ")
		input, err := e.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				e.shutdown(ctx)
				<-eventsDone
				return nil
			}
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)

		// Handle exit commands
		if input == "exit" || input == "quit" {
			e.shutdown(ctx)
			<-eventsDone
			return nil
		}

		// Skip empty input
		if input == "" {
			continue
		}

		// Send input to agent
		channels.Input <- types.NewUserInput(input)

		// Wait for turn to complete
		<-turnEnd
	}
}

// handleEvents processes events from the agent and renders them to the terminal.
func (e *Executor) handleEvents(events <-chan *types.AgentEvent, done chan struct{}, turnEnd chan struct{}) {
	defer close(done)

	for event := range events {
		e.handleEvent(event, turnEnd)
	}
}

// handleEvent processes a single event based on its type
func (e *Executor) handleEvent(event *types.AgentEvent, turnEnd chan struct{}) {
	switch event.Type {
	case types.EventTypeThinkingStart:
		e.handleThinkingStart()
	case types.EventTypeThinkingContent:
		e.handleThinkingContent(event.Content)
	case types.EventTypeThinkingEnd:
		e.handleThinkingEnd()
	case types.EventTypeToolCallStart, types.EventTypeToolCallContent, types.EventTypeToolCallEnd:
		// Tool call events are captured but not displayed
	case types.EventTypeToolCall:
		e.handleToolCall(event.ToolName)
	case types.EventTypeToolResult:
		e.handleToolResult(event.ToolOutput)
	case types.EventTypeToolResultError:
		e.handleToolResultError(event.ToolName, event.Error)
	case types.EventTypeMessageStart:
		e.handleMessageStart()
	case types.EventTypeMessageContent:
		e.handleMessageContent(event.Content)
	case types.EventTypeMessageEnd:
		e.handleMessageEnd()
	case types.EventTypeError:
		e.handleError(event.Error)
	case types.EventTypeUpdateBusy:
		// Could show a spinner here in the future
	case types.EventTypeTurnEnd:
		e.handleTurnEnd(turnEnd)
	}
}

func (e *Executor) handleThinkingStart() {
	if e.showThinking {
		fmt.Fprintln(e.writer, "\n[Thinking...]")
	}
}

func (e *Executor) handleThinkingContent(content string) {
	if e.showThinking {
		fmt.Fprint(e.writer, content)
	}
}

func (e *Executor) handleThinkingEnd() {
	if e.showThinking {
		fmt.Fprintln(e.writer, "\n[Done thinking]")
	}
}

func (e *Executor) handleToolCall(toolName string) {
	fmt.Fprintf(e.writer, "\n🔧 Tool: %s\n", toolName)
}

func (e *Executor) handleToolResult(toolOutput interface{}) {
	if result, ok := toolOutput.(string); ok {
		fmt.Fprintf(e.writer, "✅ Result: %s\n", result)
	} else {
		fmt.Fprintf(e.writer, "✅ Result: %v\n", toolOutput)
	}
}

func (e *Executor) handleToolResultError(toolName string, err error) {
	fmt.Fprintf(e.writer, "❌ Tool Error (%s): %v\n", toolName, err)
}

func (e *Executor) handleMessageStart() {
	e.messageStartPrinted = false
}

func (e *Executor) handleMessageContent(content string) {
	if content != "" && !e.messageStartPrinted {
		fmt.Fprintln(e.writer, "Assistant:")
		e.messageStartPrinted = true
	}
	fmt.Fprint(e.writer, content)
}

func (e *Executor) handleMessageEnd() {
	fmt.Fprintln(e.writer) // New line after message
}

func (e *Executor) handleError(err error) {
	fmt.Fprintf(e.writer, "\n❌ Error: %v\n", err)
}

func (e *Executor) handleTurnEnd(turnEnd chan struct{}) {
	select {
	case turnEnd <- struct{}{}:
	default:
	}
}

// shutdown gracefully shuts down the agent.
func (e *Executor) shutdown(ctx context.Context) {
	fmt.Fprintln(e.writer, "\nShutting down...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*1000000000) // 5 seconds
	defer cancel()

	if err := e.agent.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(e.writer, "Warning: shutdown error: %v\n", err)
	}
}
