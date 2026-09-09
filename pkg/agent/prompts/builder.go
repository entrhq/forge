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

// Build constructs the complete system prompt as a single string.
// It is the concatenation of BuildSegments and is used where one string is
// needed: display, snapshots, and token accounting.
func (pb *PromptBuilder) Build() string {
	return pb.staticSection() + pb.sessionSection()
}

// BuildSegments constructs the system prompt as ordered system messages tagged
// by stability, for providers that place cache breakpoints. Content and order
// are identical to Build; the split is at the first section that can change
// during a session, the tool listing.
func (pb *PromptBuilder) BuildSegments() []*types.Message {
	return []*types.Message{
		types.NewSystemMessage(pb.staticSection()).WithStability(types.StabilityStatic),
		types.NewSystemMessage(pb.sessionSection()).WithStability(types.StabilitySession),
	}
}

// staticSection holds everything that is fixed for the life of the process:
// user instructions, repository context, and the base behavioral prompts.
func (pb *PromptBuilder) staticSection() string {
	var builder strings.Builder

	if pb.customInstructions != "" {
		builder.WriteString("<custom_instructions>\n")
		builder.WriteString(pb.customInstructions)
		builder.WriteString("\n</custom_instructions>\n\n")
	}

	if pb.repositoryContext != "" {
		builder.WriteString("<repository_context>\n")
		builder.WriteString(pb.repositoryContext)
		builder.WriteString("\n</repository_context>\n\n")
	}

	builder.WriteString(SystemCapabilitiesPrompt)
	builder.WriteString("\n\n")
	builder.WriteString(AgentLoopPrompt)
	builder.WriteString("\n\n")
	builder.WriteString(ChainOfThoughtPrompt)
	builder.WriteString("\n\n")
	builder.WriteString(ToolCallingPrompt)
	builder.WriteString("\n\n")

	return builder.String()
}

// sessionSection holds the tool listing and everything after it. The rules
// and guidance text here is itself fixed, but it follows the tool listing in
// the prompt, so it can only be as stable as the section before it.
func (pb *PromptBuilder) sessionSection() string {
	var builder strings.Builder

	if len(pb.tools) > 0 {
		builder.WriteString("<available_tools>\n")
		builder.WriteString(FormatToolSchemas(pb.tools))
		builder.WriteString("</available_tools>\n\n")
	}

	builder.WriteString(ToolUseRulesPrompt)
	builder.WriteString("\n\n")
	builder.WriteString(ScratchpadGuidancePrompt)
	builder.WriteString("\n\n")
	builder.WriteString(CustomToolsGuidancePrompt)

	if pb.customToolsList != "" {
		builder.WriteString("\n\n")
		builder.WriteString(pb.customToolsList)
	}

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

// BuildMessages creates the complete message list sent to the provider.
//
// System messages always come first and are contiguous, followed only by
// conversation messages. Providers rely on that ordering to tell the system
// segment apart from history when placing cache breakpoints.
//
// ephemeralContext entries are appended as user messages for this request only
// and are never stored in memory: retrieved long-term memories and
// error-recovery context. They sit after history so that history itself stays
// byte-identical from one request to the next.
func BuildMessages(systemMessages, history []*types.Message, userMessage string, ephemeralContext ...string) []*types.Message {
	messages := make([]*types.Message, 0, len(systemMessages)+len(history)+len(ephemeralContext)+1)
	messages = append(messages, systemMessages...)

	// Stale system messages in history are dropped so the builder's segments are
	// the only system content. RoleTool is remapped to RoleUser so XML-mode
	// providers receive the expected format while memory keeps the semantic role.
	for _, msg := range history {
		if msg.Role != types.RoleSystem {
			messages = append(messages, normalizeRoleForLLM(msg))
		}
	}

	for _, content := range ephemeralContext {
		if content != "" {
			messages = append(messages, types.NewUserMessage(content))
		}
	}

	if userMessage != "" {
		messages = append(messages, types.NewUserMessage(userMessage))
	}

	return messages
}
