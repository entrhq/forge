# Agent Chat Example

This example demonstrates the complete capabilities of the Forge agent framework, including:

- **Agent loop with tool execution** - The agent runs in a loop, using tools until a task is complete
- **Chain-of-thought reasoning** - See the agent's thinking process in `[brackets]`
- **Custom tools** - Register and use custom tools (like the calculator)
- **Built-in tools** - Use `task_completion`, `ask_question`, and `converse` tools

## Features Demonstrated

### 1. Interactive TUI
This example uses the Forge TUI Executor to provide a rich, interactive terminal interface. It features:
- A welcome screen with a customizable ASCII art header.
- A chat-style view for agent conversation.
- A status bar and input handling.

### 2. Agent Loop
The agent continuously reasons and uses tools until it decides the task is complete by calling a loop-breaking tool (`task_completion`, `ask_question`, or `converse`).

### 3. Custom Tool Registration
The example includes a custom `CalculatorTool` that performs basic arithmetic. This shows how to create and register your own tools.

### 4. Chain-of-Thought
All responses include thinking blocks shown in brackets, e.g., `[Analyzing the problem...]`, which reveal the agent's reasoning process.

### 5. Built-in Tools
Three loop-breaking tools are always available:
- `task_completion` - Signal that a task is complete with a final result
- `ask_question` - Request additional information from the user
- `converse` - Continue the conversation naturally

## Running the Example

```bash
# Ensure you are in the root of the forge repository
export OPENAI_API_KEY="your-api-key"
go run examples/agent-chat/main.go
```

## Example Interactions

### Math Calculation
```
You: What is 15 * 23?

[I need to calculate 15 multiplied by 23...]
<Executing: calculator(operation=multiply, a=15, b=23)>
Tool result: 345.00

[The calculation is complete...]
The result of 15 × 23 is 345.
```

### Multi-Step Problem
```
You: Calculate (100 + 50) / 3

[I need to break this down into steps: first add 100 and 50, then divide by 3...]
<Executing: calculator(operation=add, a=100, b=50)>
Tool result: 150.00

[Now I need to divide 150 by 3...]
<Executing: calculator(operation=divide, a=150, b=3)>
Tool result: 50.00

[The calculation is complete...]
The result of (100 + 50) / 3 is 50.
```

## Creating Custom Tools

To create a custom tool, implement the `tools.Tool` interface:

```go
type MyTool struct{}

func (t *MyTool) Name() string {
    return "my_tool"
}

func (t *MyTool) Description() string {
    return "A brief description of what your tool does"
}

func (t *MyTool) Schema() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param1": map[string]interface{}{
                "type":        "string",
                "description": "Description of param1",
            },
        },
        "required": []string{"param1"},
    }
}

func (t *MyTool) Execute(ctx context.Context, arguments json.RawMessage) (string, error) {
    // Parse arguments
    var args struct {
        Param1 string `json:"param1"`
    }
    if err := json.Unmarshal(arguments, &args); err != nil {
        return "", err
    }
    
    // Do work
    result := "result of work"
    
    return result, nil
}

func (t *MyTool) IsLoopBreaking() bool {
    return false // Set to true if this tool should end the agent loop
}
```

Then register it with the agent:

```go
myTool := &MyTool{}
if err := ag.RegisterTool(myTool); err != nil {
    log.Fatal(err)
}
```

## Architecture

```
User Input
    ↓
Agent Loop ←──────────┐
    ↓                 │
LLM (with thinking)   │
    ↓                 │
Tool Call?            │
    ├─ Yes ───────────┤
    │   ↓             │
    │ Execute Tool    │
    │   ↓             │
    │ Add Result ─────┘
    │
    └─ No (or loop-breaking tool)
        ↓
    Return Result
```

## Notes

- **Thinking is always shown**: Chain-of-thought reasoning is mandatory and displayed in brackets
- **No iteration limits**: The agent loop continues until a loop-breaking tool is called
- **Tool results**: Are automatically added to conversation memory for context
- **Error handling**: Tool execution errors are caught and reported gracefully