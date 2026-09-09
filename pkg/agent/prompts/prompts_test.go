package prompts

import (
	"context"
	"strings"
	"testing"

	"github.com/entrhq/forge/pkg/agent/tools"
	"github.com/entrhq/forge/pkg/types"
)

func TestFormatToolSchema(t *testing.T) {
	tool := tools.NewTaskCompletionTool()

	formatted := FormatToolSchema(tool)

	// Check that it includes the tool name
	if !strings.Contains(formatted, "task_completion") {
		t.Error("formatted schema should contain tool name")
	}

	// Check that it includes description
	if !strings.Contains(formatted, "Signal that the task is complete") {
		t.Error("formatted schema should contain description")
	}

	// Check that it includes parameters
	if !strings.Contains(formatted, "Parameters") {
		t.Error("formatted schema should contain parameters section")
	}

	// Check that it mentions loop-breaking
	if !strings.Contains(formatted, "loop-breaking") {
		t.Error("formatted schema should indicate loop-breaking tool")
	}

	// Check that it includes example
	if !strings.Contains(formatted, "Example") {
		t.Error("formatted schema should include example")
	}
}

func TestFormatToolSchemas(t *testing.T) {
	t.Run("MultipleTools", func(t *testing.T) {
		toolsList := []tools.Tool{
			tools.NewTaskCompletionTool(),
			tools.NewAskQuestionTool(),
			tools.NewConverseTool(),
		}

		formatted := FormatToolSchemas(toolsList)

		// Check all tools are included
		if !strings.Contains(formatted, "task_completion") {
			t.Error("should contain task_completion")
		}
		if !strings.Contains(formatted, "ask_question") {
			t.Error("should contain ask_question")
		}
		if !strings.Contains(formatted, "converse") {
			t.Error("should contain converse")
		}

		// Check section header
		if !strings.Contains(formatted, "AVAILABLE TOOLS") {
			t.Error("should contain AVAILABLE TOOLS header")
		}
	})

	t.Run("NoTools", func(t *testing.T) {
		formatted := FormatToolSchemas([]tools.Tool{})

		if !strings.Contains(formatted, "No tools available") {
			t.Error("should indicate no tools available")
		}
	})
}

func TestPromptBuilder(t *testing.T) {
	t.Run("BasicBuild", func(t *testing.T) {
		toolsList := []tools.Tool{
			tools.NewTaskCompletionTool(),
		}

		builder := NewPromptBuilder().
			WithTools(toolsList)

		prompt := builder.Build()

		// Check system capabilities section
		if !strings.Contains(prompt, "<system_capabilities>") {
			t.Error("should contain system capabilities section")
		}

		// Check tools are included
		if !strings.Contains(prompt, "task_completion") {
			t.Error("should contain tool information")
		}

		// Check chain-of-thought (always included)
		if !strings.Contains(prompt, "<chain_of_thought>") {
			t.Error("should always contain chain-of-thought section")
		}
	})

	t.Run("WithCustomInstructions", func(t *testing.T) {
		customInstructions := "Be extra helpful and friendly."

		builder := NewPromptBuilder().
			WithTools([]tools.Tool{}).
			WithCustomInstructions(customInstructions)

		prompt := builder.Build()

		if !strings.Contains(prompt, customInstructions) {
			t.Error("should contain custom instructions")
		}
		if !strings.Contains(prompt, "<custom_instructions>") {
			t.Error("should contain custom instructions header")
		}
	})
}

func TestBuildMessages(t *testing.T) {
	t.Run("WithHistory", func(t *testing.T) {
		systemPrompt := "You are helpful"
		history := []*types.Message{
			types.NewUserMessage("Hello"),
			types.NewAssistantMessage("Hi there!"),
		}
		userMessage := "How are you?"

		messages := BuildMessages(systemMessagesFor(systemPrompt), history, userMessage)

		// Should have: system + 2 history + new user = 4 messages
		if len(messages) != 4 {
			t.Errorf("expected 4 messages, got %d", len(messages))
		}

		// First should be system
		if messages[0].Role != types.RoleSystem {
			t.Error("first message should be system")
		}
		if messages[0].Content != systemPrompt {
			t.Error("system message content mismatch")
		}

		// Last should be new user message
		if messages[len(messages)-1].Role != types.RoleUser {
			t.Error("last message should be user")
		}
		if messages[len(messages)-1].Content != userMessage {
			t.Error("user message content mismatch")
		}
	})

	t.Run("SkipsSystemInHistory", func(t *testing.T) {
		systemPrompt := "You are helpful"
		history := []*types.Message{
			types.NewSystemMessage("Old system prompt"),
			types.NewUserMessage("Hello"),
		}

		messages := BuildMessages(systemMessagesFor(systemPrompt), history, "")

		// Should have: new system + 1 user (old system skipped) = 2 messages
		if len(messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(messages))
		}

		// First should be new system prompt
		if messages[0].Content != systemPrompt {
			t.Error("should use new system prompt, not old one from history")
		}
	})
}

func TestNormalizeRoleForLLM(t *testing.T) {
	t.Run("RoleToolRemappedToRoleUser", func(t *testing.T) {
		original := types.NewToolMessage("Tool result content")
		normalized := normalizeRoleForLLM(original)

		if normalized.Role != types.RoleUser {
			t.Errorf("expected RoleUser after normalization, got %s", normalized.Role)
		}
		if normalized.Content != original.Content {
			t.Error("content must be preserved during normalization")
		}
		// Original must not be mutated — normalizeRoleForLLM must return a copy.
		if original.Role != types.RoleTool {
			t.Error("normalizeRoleForLLM must not mutate the original message")
		}
	})

	t.Run("OtherRolesPassThrough", func(t *testing.T) {
		msgs := []*types.Message{
			types.NewUserMessage("user msg"),
			types.NewAssistantMessage("assistant msg"),
			types.NewSystemMessage("system msg"),
		}
		for _, msg := range msgs {
			result := normalizeRoleForLLM(msg)
			// Non-tool messages must be returned as-is (same pointer, no copy).
			if result != msg {
				t.Errorf("expected same pointer for role %s, got a copy", msg.Role)
			}
		}
	})
}

func TestBuildMessages_ToolRoleRemapping(t *testing.T) {
	// Tool results stored in memory as RoleTool must arrive at the LLM as RoleUser.
	systemPrompt := "You are helpful"
	history := []*types.Message{
		types.NewUserMessage("Run the tests"),
		types.NewAssistantMessage("<tool>...</tool>"),
		types.NewToolMessage("Tool 'execute_command' result:\nok"),
	}

	messages := BuildMessages(systemMessagesFor(systemPrompt), history, "")

	// system + 3 history = 4 messages
	if len(messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(messages))
	}

	// The tool result (index 3) must appear as RoleUser for XML-mode LLMs.
	toolEntry := messages[3]
	if toolEntry.Role != types.RoleUser {
		t.Errorf("tool result must be remapped to RoleUser in LLM payload, got %s", toolEntry.Role)
	}
	if toolEntry.Content != history[2].Content {
		t.Error("tool result content must be preserved after role remapping")
	}
}

func TestFormatToolForLLM(t *testing.T) {
	tool := tools.NewTaskCompletionTool()

	formatted := FormatToolForLLM(tool)

	if formatted["name"] != "task_completion" {
		t.Error("should include tool name")
	}

	if _, ok := formatted["description"]; !ok {
		t.Error("should include description")
	}

	if _, ok := formatted["parameters"]; !ok {
		t.Error("should include parameters")
	}
}

func TestSchemaToJSON(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type": "string",
			},
		},
	}

	jsonStr, err := SchemaToJSON(schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(jsonStr, "object") {
		t.Error("JSON should contain schema type")
	}
	if !strings.Contains(jsonStr, "name") {
		t.Error("JSON should contain properties")
	}
}

// systemMessagesFor wraps a plain system prompt in the slice form BuildMessages takes.
func systemMessagesFor(systemPrompt string) []*types.Message {
	return []*types.Message{types.NewSystemMessage(systemPrompt)}
}

func TestBuildSegments(t *testing.T) {
	builder := NewPromptBuilder().
		WithCustomInstructions("be terse").
		WithRepositoryContext("go module").
		WithTools([]tools.Tool{tools.NewTaskCompletionTool()}).
		WithCustomToolsList("## Custom").
		WithBrowserGuidance("## Browser")

	segments := builder.BuildSegments()
	if len(segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segments))
	}
	for _, seg := range segments {
		if seg.Role != types.RoleSystem {
			t.Errorf("segment role must be system, got %s", seg.Role)
		}
	}
	if segments[0].Stability != types.StabilityStatic {
		t.Error("first segment must be static")
	}
	if segments[1].Stability != types.StabilitySession {
		t.Error("second segment must be session")
	}

	// The split point is the tool listing: everything before it is static.
	if strings.Contains(segments[0].Content, "<available_tools>") {
		t.Error("static segment must not contain the tool listing")
	}
	if !strings.HasPrefix(segments[1].Content, "<available_tools>") {
		t.Error("session segment must start with the tool listing")
	}
	for _, want := range []string{"be terse", "go module", ToolCallingPrompt} {
		if !strings.Contains(segments[0].Content, want) {
			t.Errorf("static segment missing %q", want)
		}
	}
	for _, want := range []string{"task_completion", ToolUseRulesPrompt, "## Custom", "## Browser"} {
		if !strings.Contains(segments[1].Content, want) {
			t.Errorf("session segment missing %q", want)
		}
	}

	if joined := segments[0].Content + segments[1].Content; joined != builder.Build() {
		t.Error("BuildSegments joined must equal Build")
	}
}

func TestBuildMessages_SystemFirstThenEphemeralTrailing(t *testing.T) {
	system := []*types.Message{
		types.NewSystemMessage("static").WithStability(types.StabilityStatic),
		types.NewSystemMessage("session").WithStability(types.StabilitySession),
	}
	history := []*types.Message{
		types.NewUserMessage("hello"),
		types.NewAssistantMessage("hi"),
	}

	messages := BuildMessages(system, history, "", "memories", "", "error context")

	wantRoles := []types.MessageRole{
		types.RoleSystem, types.RoleSystem,
		types.RoleUser, types.RoleAssistant,
		types.RoleUser, types.RoleUser,
	}
	if len(messages) != len(wantRoles) {
		t.Fatalf("expected %d messages, got %d", len(wantRoles), len(messages))
	}
	for i, role := range wantRoles {
		if messages[i].Role != role {
			t.Errorf("message %d: expected role %s, got %s", i, role, messages[i].Role)
		}
	}
	if messages[0].Stability != types.StabilityStatic || messages[1].Stability != types.StabilitySession {
		t.Error("system message stability must be preserved")
	}
	if messages[4].Content != "memories" || messages[5].Content != "error context" {
		t.Error("ephemeral context must follow history in the order given, skipping empty entries")
	}
}

// manyParamsTool has enough required parameters that unsorted map iteration
// would reorder the formatted output almost every call.
type manyParamsTool struct{}

func (manyParamsTool) Name() string        { return "many_params" }
func (manyParamsTool) Description() string { return "A tool with many parameters." }
func (manyParamsTool) IsLoopBreaking() bool { return false }
func (manyParamsTool) Execute(context.Context, []byte) (string, map[string]any, error) {
	return "", nil, nil
}
func (manyParamsTool) Schema() map[string]any {
	props := map[string]any{}
	var required []string
	for _, name := range []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta"} {
		props[name] = map[string]any{"type": "string", "description": name}
		required = append(required, name)
	}
	return map[string]any{"type": "object", "properties": props, "required": required}
}

func TestFormatToolSchemas_Deterministic(t *testing.T) {
	toolsList := []tools.Tool{manyParamsTool{}}
	first := FormatToolSchemas(toolsList)
	for range 50 {
		if got := FormatToolSchemas(toolsList); got != first {
			t.Fatal("FormatToolSchemas output changed between calls with identical input")
		}
	}
	if !strings.Contains(first, "- `alpha`") || strings.Index(first, "- `alpha`") > strings.Index(first, "- `beta`") {
		t.Error("parameters must be listed in sorted order")
	}
}
