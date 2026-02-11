package prompts

import (
	"fmt"
	"strings"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/types"
)

// PromptBuilder constructs dynamic system prompts for the agent loop
type PromptBuilder struct {
	tools              []tools.Tool
	customInstructions string
	repositoryContext  string
}

// NewPromptBuilder creates a new prompt builder with default settings
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{
		tools: []tools.Tool{},
	}
}

// WithTools sets the available tools for the agent
func (pb *PromptBuilder) WithTools(toolsList []tools.Tool) *PromptBuilder {
	pb.tools = toolsList
	return pb
}

// WithCustomInstructions adds custom user-provided instructions
// These are instructions from the end user, not the base system prompt
func (pb *PromptBuilder) WithCustomInstructions(instructions string) *PromptBuilder {
	pb.customInstructions = instructions
	return pb
}

// WithRepositoryContext adds repository-specific context from AGENTS.md
// This provides project-specific information separate from custom instructions
func (pb *PromptBuilder) WithRepositoryContext(context string) *PromptBuilder {
	pb.repositoryContext = context
	return pb
}

// Build constructs the complete system prompt by assembling all sections
func (pb *PromptBuilder) Build() string {
	var builder strings.Builder

	// Add custom instructions if provided (these are user-provided instructions)
	if pb.customInstructions != "" {
		builder.WriteString("<custom_instructions>\n")
		builder.WriteString(pb.customInstructions)
		builder.WriteString("\n</custom_instructions>\n\n")
	}

	// Add repository context if provided (from AGENTS.md)
	if pb.repositoryContext != "" {
		builder.WriteString("<repository_context>\n")
		builder.WriteString(pb.repositoryContext)
		builder.WriteString("\n</repository_context>\n\n")
	}

	// Add system capabilities
	builder.WriteString(SystemCapabilitiesPrompt)
	builder.WriteString("\n\n")

	// Add agent loop explanation
	builder.WriteString(AgentLoopPrompt)
	builder.WriteString("\n\n")

	// Add chain of thought instructions
	builder.WriteString(ChainOfThoughtPrompt)
	builder.WriteString("\n\n")

	// Add tool calling instructions
	builder.WriteString(ToolCallingPrompt)
	builder.WriteString("\n\n")

	// Add available tools section
	if len(pb.tools) > 0 {
		builder.WriteString("<available_tools>\n")
		builder.WriteString(FormatToolSchemas(pb.tools))
		builder.WriteString("</available_tools>\n\n")
	}

	// Add tool use rules
	builder.WriteString(ToolUseRulesPrompt)
	builder.WriteString("\n\n")

	// Add scratchpad guidance
	builder.WriteString(ScratchpadGuidancePrompt)

	return builder.String()
}

// BuildMessages creates a complete message list including system prompt and conversation history
// The errorContext parameter allows passing ephemeral error messages to the agent without
// storing them in permanent memory - useful for self-healing error recovery
func BuildMessages(systemPrompt string, history []*types.Message, userMessage string, errorContext string) []*types.Message {
	messages := make([]*types.Message, 0, len(history)+3)

	// Add system message
	messages = append(messages, types.NewSystemMessage(systemPrompt))

	// Add conversation history (skip any existing system messages to avoid duplicates)
	for _, msg := range history {
		if msg.Role != types.RoleSystem {
			messages = append(messages, msg)
		}
	}

	// Add error context as ephemeral user message if provided
	// This is NOT stored in memory - only used for this iteration
	if errorContext != "" {
		messages = append(messages, types.NewUserMessage(errorContext))
	}

	// Add new user message if provided
	if userMessage != "" {
		messages = append(messages, types.NewUserMessage(userMessage))
	}

	return messages
}

// BuildMessagesForIteration creates messages for an agent loop iteration,
// including tool results and thinking
func BuildMessagesForIteration(
	systemPrompt string,
	history []*types.Message,
	toolResults []ToolResult,
) []*types.Message {
	messages := make([]*types.Message, 0, len(history)+1+len(toolResults))

	// Add system message
	messages = append(messages, types.NewSystemMessage(systemPrompt))

	// Add conversation history (skip system messages)
	for _, msg := range history {
		if msg.Role != types.RoleSystem {
			messages = append(messages, msg)
		}
	}

	// Add tool results as user messages
	for _, result := range toolResults {
		resultMsg := fmt.Sprintf("Tool '%s' result:\n%s", result.ToolName, result.Result)
		if result.Error != nil {
			resultMsg = fmt.Sprintf("Tool '%s' error:\n%s", result.ToolName, result.Error.Error())
		}
		messages = append(messages, types.NewUserMessage(resultMsg))
	}

	return messages
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolName string
	Result   string
	Error    error
}
