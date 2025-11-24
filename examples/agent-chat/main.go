package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"os"

	"github.com/entrhq/forge/pkg/agent"
	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/executor/tui"
	"github.com/entrhq/forge/pkg/llm/openai"
)

// CalculatorTool is a custom tool that performs basic arithmetic
type CalculatorTool struct{}

func (t *CalculatorTool) Name() string {
	return "calculator"
}

func (t *CalculatorTool) Description() string {
	return "Performs basic arithmetic operations (add, subtract, multiply, divide)"
}

func (t *CalculatorTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"description": "The operation to perform: add, subtract, multiply, or divide",
				"enum":        []string{"add", "subtract", "multiply", "divide"},
			},
			"a": map[string]interface{}{
				"type":        "number",
				"description": "First number",
			},
			"b": map[string]interface{}{
				"type":        "number",
				"description": "Second number",
			},
		},
		"required": []string{"operation", "a", "b"},
	}
}

func (t *CalculatorTool) Execute(ctx context.Context, argsXML []byte) (string, map[string]interface{}, error) {
	var args struct {
		XMLName   xml.Name `xml:"arguments"`
		Operation string   `xml:"operation"`
		A         float64  `xml:"a"`
		B         float64  `xml:"b"`
	}

	if err := tools.UnmarshalXMLWithFallback(argsXML, &args); err != nil {
		return "", nil, fmt.Errorf("invalid arguments: %w", err)
	}

	var result float64
	switch args.Operation {
	case "add":
		result = args.A + args.B
	case "subtract":
		result = args.A - args.B
	case "multiply":
		result = args.A * args.B
	case "divide":
		if args.B == 0 {
			return "", nil, fmt.Errorf("division by zero")
		}
		result = args.A / args.B
	default:
		return "", nil, fmt.Errorf("unknown operation: %s", args.Operation)
	}

	return fmt.Sprintf("%.2f", result), nil, nil
}

func (t *CalculatorTool) IsLoopBreaking() bool {
	return false // This tool doesn't end the conversation
}

func main() {
	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Error: OPENAI_API_KEY environment variable is required.")
		log.Println("Please set it and try again.")
		os.Exit(1)
	}

	// Create OpenAI provider
	provider, err := openai.NewProvider(apiKey,
		openai.WithModel("openai/gpt-4o"),
	)
	if err != nil {
		log.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	// Create agent with custom instructions
	ag := agent.NewDefaultAgent(provider,
		agent.WithCustomInstructions("You are a helpful AI assistant with access to tools. You can help with calculations and conversations."),
	)

	// Register custom calculator tool
	calculator := &CalculatorTool{}
	if err := ag.RegisterTool(calculator); err != nil {
		log.Fatalf("Failed to register calculator tool: %v", err)
	}

	// Get current working directory for git operations
	workspaceDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Create TUI executor with provider and workspace for git operations
	executor := tui.NewExecutor(ag, provider, workspaceDir)

	// Run the conversation
	ctx := context.Background()
	if err := executor.Run(ctx); err != nil {
		log.Fatalf("Executor error: %v", err)
	}

	fmt.Println("\nGoodbye!")
}
