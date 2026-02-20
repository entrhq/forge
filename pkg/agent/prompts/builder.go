package prompts

import (
	"strings"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/types"
)

// PromptBuilder constructs dynamic system prompts for the agent loop
type PromptBuilder struct {
	tools              []tools.Tool
	customInstructions string
	repositoryContext  string
	customToolsList    string
	browserGuidance    string
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

// WithCustomToolsList adds the formatted list of available custom tools
func (pb *PromptBuilder) WithCustomToolsList(customTools string) *PromptBuilder {
	pb.customToolsList = customTools
	return pb
}

// WithBrowserGuidance adds browser automation workflow guidance when browser tools are enabled
func (pb *PromptBuilder) WithBrowserGuidance(guidance string) *PromptBuilder {
	pb.browserGuidance = guidance
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
	builder.WriteString("\n\n")

	// Add custom tools guidance
	builder.WriteString(CustomToolsGuidancePrompt)

	// Add available custom tools list if provided
	if pb.customToolsList != "" {
		builder.WriteString("\n\n")
		builder.WriteString(pb.customToolsList)
	}

	// Add browser automation guidance if provided
	if pb.browserGuidance != "" {
		builder.WriteString("\n\n")
		builder.WriteString(pb.browserGuidance)
	}

	return builder.String()
}

// normalizeRoleForLLM returns a copy of msg with RoleTool remapped to RoleUser.
// Tool results are stored in memory as RoleTool so the context summarization
// strategies can identify and group call/result pairs correctly. XML-mode LLM
// providers don't have a native tool role, so the mapping happens here at the
// boundary — just before the payload is sent to the provider.
// The original pointer is reused when no copy is needed, avoiding allocations.
func normalizeRoleForLLM(msg *types.Message) *types.Message {
	if msg.Role != types.RoleTool {
		return msg
	}
	// Shallow-copy the message, remapping only the role.
	normalized := *msg
	normalized.Role = types.RoleUser
	return &normalized
}

// BuildMessages creates a complete message list including system prompt and conversation history
// The errorContext parameter allows passing ephemeral error messages to the agent without
// storing them in permanent memory - useful for self-healing error recovery
func BuildMessages(systemPrompt string, history []*types.Message, userMessage string, errorContext string) []*types.Message {
	messages := make([]*types.Message, 0, len(history)+3)

	// Add system message
	messages = append(messages, types.NewSystemMessage(systemPrompt))

	// Add conversation history (skip any existing system messages to avoid duplicates).
	// RoleTool messages are remapped to RoleUser so XML-mode providers receive the
	// expected format while memory retains the semantic role.
	for _, msg := range history {
		if msg.Role != types.RoleSystem {
			messages = append(messages, normalizeRoleForLLM(msg))
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


