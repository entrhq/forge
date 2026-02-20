package prompts

import (
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

		messages := BuildMessages(systemPrompt, history, userMessage, "")

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

		messages := BuildMessages(systemPrompt, history, "", "")

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

	messages := BuildMessages(systemPrompt, history, "", "")

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
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
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
