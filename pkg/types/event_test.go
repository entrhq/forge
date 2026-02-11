package types

import (
	"errors"
	"testing"
)

func TestAgentEventType(t *testing.T) {
	tests := []struct {
		eventType AgentEventType
		name      string
		expected  string
	}{
		{
			name:      "thinking_start",
			eventType: EventTypeThinkingStart,
			expected:  "thinking_start",
		},
		{
			name:      "thinking_content",
			eventType: EventTypeThinkingContent,
			expected:  "thinking_content",
		},
		{
			name:      "thinking_end",
			eventType: EventTypeThinkingEnd,
			expected:  "thinking_end",
		},
		{
			name:      "message_start",
			eventType: EventTypeMessageStart,
			expected:  "message_start",
		},
		{
			name:      "message_content",
			eventType: EventTypeMessageContent,
			expected:  "message_content",
		},
		{
			name:      "message_end",
			eventType: EventTypeMessageEnd,
			expected:  "message_end",
		},
		{
			name:      "tool_call",
			eventType: EventTypeToolCall,
			expected:  "tool_call",
		},
		{
			name:      "tool_result",
			eventType: EventTypeToolResult,
			expected:  "tool_result",
		},
		{
			name:      "tool_result_error",
			eventType: EventTypeToolResultError,
			expected:  "tool_result_error",
		},
		{
			name:      "no_tool_call",
			eventType: EventTypeNoToolCall,
			expected:  "no_tool_call",
		},
		{
			name:      "api_call_start",
			eventType: EventTypeAPICallStart,
			expected:  "api_call_start",
		},
		{
			name:      "api_call_end",
			eventType: EventTypeAPICallEnd,
			expected:  "api_call_end",
		},
		{
			name:      "tools_update",
			eventType: EventTypeToolsUpdate,
			expected:  "tools_update",
		},
		{
			name:      "update_busy",
			eventType: EventTypeUpdateBusy,
			expected:  "update_busy",
		},
		{
			name:      "turn_end",
			eventType: EventTypeTurnEnd,
			expected:  "turn_end",
		},
		{
			name:      "error",
			eventType: EventTypeError,
			expected:  "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.eventType) != tt.expected {
				t.Errorf("EventType = %v, want %v", tt.eventType, tt.expected)
			}
		})
	}
}

func TestNewThinkingEvents(t *testing.T) {
	start := NewThinkingStartEvent()
	if start.Type != EventTypeThinkingStart {
		t.Errorf("ThinkingStart type = %v, want %v", start.Type, EventTypeThinkingStart)
	}

	content := NewThinkingContentEvent("analyzing the problem")
	if content.Type != EventTypeThinkingContent {
		t.Errorf("ThinkingContent type = %v, want %v", content.Type, EventTypeThinkingContent)
	}
	if content.Content != "analyzing the problem" {
		t.Errorf("ThinkingContent content = %v, want 'analyzing the problem'", content.Content)
	}

	end := NewThinkingEndEvent()
	if end.Type != EventTypeThinkingEnd {
		t.Errorf("ThinkingEnd type = %v, want %v", end.Type, EventTypeThinkingEnd)
	}
}

func TestNewMessageEvents(t *testing.T) {
	start := NewMessageStartEvent()
	if start.Type != EventTypeMessageStart {
		t.Errorf("MessageStart type = %v, want %v", start.Type, EventTypeMessageStart)
	}

	content := NewMessageContentEvent("Hello, world!")
	if content.Type != EventTypeMessageContent {
		t.Errorf("MessageContent type = %v, want %v", content.Type, EventTypeMessageContent)
	}
	if content.Content != "Hello, world!" {
		t.Errorf("MessageContent content = %v, want 'Hello, world!'", content.Content)
	}

	end := NewMessageEndEvent()
	if end.Type != EventTypeMessageEnd {
		t.Errorf("MessageEnd type = %v, want %v", end.Type, EventTypeMessageEnd)
	}
}

func TestNewToolEvents(t *testing.T) {
	toolInput := map[string]interface{}{
		"query": "weather in San Francisco",
	}

	call := NewToolCallEvent("weather_api", toolInput)
	if call.Type != EventTypeToolCall {
		t.Errorf("ToolCall type = %v, want %v", call.Type, EventTypeToolCall)
	}
	if call.ToolName != "weather_api" {
		t.Errorf("ToolCall tool name = %v, want 'weather_api'", call.ToolName)
	}
	if call.ToolInput["query"] != "weather in San Francisco" {
		t.Error("ToolCall input not set correctly")
	}

	result := NewToolResultEvent("weather_api", "Sunny, 72°F")
	if result.Type != EventTypeToolResult {
		t.Errorf("ToolResult type = %v, want %v", result.Type, EventTypeToolResult)
	}
	if result.ToolName != "weather_api" {
		t.Errorf("ToolResult tool name = %v, want 'weather_api'", result.ToolName)
	}
	if result.ToolOutput != "Sunny, 72°F" {
		t.Errorf("ToolResult output = %v, want 'Sunny, 72°F'", result.ToolOutput)
	}

	err := errors.New("API timeout")
	errEvent := NewToolResultErrorEvent("weather_api", err)
	if errEvent.Type != EventTypeToolResultError {
		t.Errorf("ToolResultError type = %v, want %v", errEvent.Type, EventTypeToolResultError)
	}
	if errEvent.Error != err {
		t.Error("ToolResultError error not set correctly")
	}

	noCall := NewNoToolCallEvent()
	if noCall.Type != EventTypeNoToolCall {
		t.Errorf("NoToolCall type = %v, want %v", noCall.Type, EventTypeNoToolCall)
	}
}

func TestNewAPIEvents(t *testing.T) {
	start := NewAPICallStartEvent("openai", 50000, 100000)
	if start.Type != EventTypeAPICallStart {
		t.Errorf("APICallStart type = %v, want %v", start.Type, EventTypeAPICallStart)
	}
	if start.Metadata["api_name"] != "openai" {
		t.Error("APICallStart api_name metadata not set")
	}
	if start.APICallInfo == nil {
		t.Error("APICallInfo not set")
	}
	if start.APICallInfo.ContextTokens != 50000 {
		t.Errorf("ContextTokens = %v, want %v", start.APICallInfo.ContextTokens, 50000)
	}
	if start.APICallInfo.MaxContextTokens != 100000 {
		t.Errorf("MaxContextTokens = %v, want %v", start.APICallInfo.MaxContextTokens, 100000)
	}

	end := NewAPICallEndEvent("openai")
	if end.Type != EventTypeAPICallEnd {
		t.Errorf("APICallEnd type = %v, want %v", end.Type, EventTypeAPICallEnd)
	}
	if end.Metadata["api_name"] != "openai" {
		t.Error("ApiCallEnd api_name metadata not set")
	}
}

func TestNewOtherEvents(t *testing.T) {
	tools := []string{"weather_api", "calculator", "search"}
	toolsUpdate := NewToolsUpdateEvent(tools)
	if toolsUpdate.Type != EventTypeToolsUpdate {
		t.Errorf("ToolsUpdate type = %v, want %v", toolsUpdate.Type, EventTypeToolsUpdate)
	}

	busyTrue := NewUpdateBusyEvent(true)
	if busyTrue.Type != EventTypeUpdateBusy {
		t.Errorf("UpdateBusy type = %v, want %v", busyTrue.Type, EventTypeUpdateBusy)
	}
	if !busyTrue.IsBusy {
		t.Error("UpdateBusy should be busy")
	}

	busyFalse := NewUpdateBusyEvent(false)
	if busyFalse.IsBusy {
		t.Error("UpdateBusy should not be busy")
	}

	turnEnd := NewTurnEndEvent()
	if turnEnd.Type != EventTypeTurnEnd {
		t.Errorf("TurnEnd type = %v, want %v", turnEnd.Type, EventTypeTurnEnd)
	}

	err := errors.New("test error")
	errorEvent := NewErrorEvent(err)
	if errorEvent.Type != EventTypeError {
		t.Errorf("Error type = %v, want %v", errorEvent.Type, EventTypeError)
	}
	if errorEvent.Error != err {
		t.Error("Error event error not set correctly")
	}
}

func TestAgentEventWithMetadata(t *testing.T) {
	event := NewMessageContentEvent("test")
	key := "test_key"
	value := "test_value"

	result := event.WithMetadata(key, value)

	if result != event {
		t.Error("WithMetadata should return the same event for chaining")
	}
	if event.Metadata[key] != value {
		t.Errorf("WithMetadata did not set metadata correctly, got %v, want %v", event.Metadata[key], value)
	}
}

func TestAgentEventHelpers(t *testing.T) {
	tests := []struct {
		event      *AgentEvent
		name       string
		isThinking bool
		isMessage  bool
		isTool     bool
		isApi      bool
		isContent  bool
		isError    bool
	}{
		{
			name:       "thinking_start",
			event:      NewThinkingStartEvent(),
			isThinking: true,
			isMessage:  false,
			isTool:     false,
			isApi:      false,
			isContent:  false,
			isError:    false,
		},
		{
			name:       "thinking_content",
			event:      NewThinkingContentEvent("test"),
			isThinking: true,
			isMessage:  false,
			isTool:     false,
			isApi:      false,
			isContent:  true,
			isError:    false,
		},
		{
			name:       "message_content",
			event:      NewMessageContentEvent("test"),
			isThinking: false,
			isMessage:  true,
			isTool:     false,
			isApi:      false,
			isContent:  true,
			isError:    false,
		},
		{
			name:       "tool_call",
			event:      NewToolCallEvent("test", nil),
			isThinking: false,
			isMessage:  false,
			isTool:     true,
			isApi:      false,
			isContent:  false,
			isError:    false,
		},
		{
			name:       "api_call_start",
			event:      NewAPICallStartEvent("test", 1000, 2000),
			isThinking: false,
			isMessage:  false,
			isTool:     false,
			isApi:      true,
			isContent:  false,
			isError:    false,
		},
		{
			name:       "error",
			event:      NewErrorEvent(errors.New("test")),
			isThinking: false,
			isMessage:  false,
			isTool:     false,
			isApi:      false,
			isContent:  false,
			isError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.event.IsThinkingEvent() != tt.isThinking {
				t.Errorf("IsThinkingEvent() = %v, want %v", tt.event.IsThinkingEvent(), tt.isThinking)
			}
			if tt.event.IsMessageEvent() != tt.isMessage {
				t.Errorf("IsMessageEvent() = %v, want %v", tt.event.IsMessageEvent(), tt.isMessage)
			}
			if tt.event.IsToolEvent() != tt.isTool {
				t.Errorf("IsToolEvent() = %v, want %v", tt.event.IsToolEvent(), tt.isTool)
			}
			if tt.event.IsAPIEvent() != tt.isApi {
				t.Errorf("IsAPIEvent() = %v, want %v", tt.event.IsAPIEvent(), tt.isApi)
			}
			if tt.event.IsContentEvent() != tt.isContent {
				t.Errorf("IsContentEvent() = %v, want %v", tt.event.IsContentEvent(), tt.isContent)
			}
			if tt.event.IsErrorEvent() != tt.isError {
				t.Errorf("IsErrorEvent() = %v, want %v", tt.event.IsErrorEvent(), tt.isError)
			}
		})
	}
}
