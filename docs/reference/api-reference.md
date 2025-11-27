# API Reference

Complete API reference for the Forge agent framework.

## Table of Contents

- [Core Package (`pkg/core`)](#core-package-pkgcore)
- [Agent Package (`pkg/agent`)](#agent-package-pkgagent)
- [Provider Package (`pkg/provider`)](#provider-package-pkgprovider)
- [Tool Package (`pkg/tool`)](#tool-package-pkgtool)
- [Memory Package (`pkg/memory`)](#memory-package-pkgmemory)
- [Executor Package (`pkg/executor`)](#executor-package-pkgexecutor)
- [Message Package (`pkg/message`)](#message-package-pkgmessage)

---

## Core Package (`pkg/core`)

### Types

#### `Agent`

Primary interface for creating and running AI agents.

```go
type Agent interface {
    Run(ctx context.Context, executor Executor) error
}
```

**Methods:**

- `Run(ctx context.Context, executor Executor) error`
  - Starts the agent loop with the given executor
  - **Parameters:**
    - `ctx`: Context for cancellation and timeouts
    - `executor`: Execution environment (e.g., CLI)
  - **Returns:** Error if agent loop fails
  - **Thread-safe:** Yes

**Example:**

```go
agent, err := core.NewAgent(provider, memory, tools)
if err != nil {
    log.Fatal(err)
}

if err := agent.Run(ctx, executor); err != nil {
    log.Fatal(err)
}
```

---

#### `Provider`

Interface for LLM service providers.

```go
type Provider interface {
    Complete(ctx context.Context, messages []Message) (*Response, error)
    Stream(ctx context.Context, messages []Message) (<-chan StreamChunk, error)
}
```

**Methods:**

- `Complete(ctx context.Context, messages []Message) (*Response, error)`
  - Sends messages and receives complete response
  - **Parameters:**
    - `ctx`: Context for cancellation and timeouts
    - `messages`: Conversation history
  - **Returns:** Complete response or error
  - **Thread-safe:** Yes

- `Stream(ctx context.Context, messages []Message) (<-chan StreamChunk, error)`
  - Sends messages and streams response chunks
  - **Parameters:**
    - `ctx`: Context for cancellation and timeouts
    - `messages`: Conversation history
  - **Returns:** Channel of response chunks or error
  - **Thread-safe:** Yes

**Example:**

```go
provider := openai.NewProvider("gpt-4", apiKey)

response, err := provider.Complete(ctx, messages)
if err != nil {
    log.Fatal(err)
}
fmt.Println(response.Content)
```

---

#### `Tool`

Interface for tools that agents can use.

```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, args map[string]interface{}) (string, error)
    IsLoopBreaking() bool
}
```

**Methods:**

- `Name() string`
  - Returns the tool's unique identifier
  - **Thread-safe:** Yes

- `Description() string`
  - Returns human-readable description for LLM
  - **Thread-safe:** Yes

- `Parameters() map[string]interface{}`
  - Returns JSON Schema for tool arguments
  - **Thread-safe:** Yes

- `Execute(ctx context.Context, args map[string]interface{}) (string, error)`
  - Executes the tool with given arguments
  - **Parameters:**
    - `ctx`: Context for cancellation and timeouts
    - `args`: Tool arguments matching schema
  - **Returns:** Result string or error
  - **Thread-safe:** Implementation-dependent

- `IsLoopBreaking() bool`
  - Returns true if tool ends the agent loop
  - **Thread-safe:** Yes

**Example:**

```go
type Calculator struct{}

func (c *Calculator) Name() string {
    return "calculator"
}

func (c *Calculator) Description() string {
    return "Performs basic arithmetic operations"
}

func (c *Calculator) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "operation": map[string]interface{}{
                "type": "string",
                "enum": []string{"add", "subtract", "multiply", "divide"},
            },
            "a": map[string]interface{}{"type": "number"},
            "b": map[string]interface{}{"type": "number"},
        },
        "required": []string{"operation", "a", "b"},
    }
}

func (c *Calculator) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    // Implementation
}

func (c *Calculator) IsLoopBreaking() bool {
    return false
}
```

---

#### `Memory`

Interface for conversation memory management.

```go
type Memory interface {
    Add(message Message) error
    GetMessages() []Message
    Clear() error
    Prune(maxTokens int) error
}
```

**Methods:**

- `Add(message Message) error`
  - Adds a message to memory
  - **Parameters:**
    - `message`: Message to add
  - **Returns:** Error if operation fails
  - **Thread-safe:** Implementation-dependent

- `GetMessages() []Message`
  - Retrieves all messages
  - **Returns:** Slice of all messages
  - **Thread-safe:** Implementation-dependent

- `Clear() error`
  - Removes all messages
  - **Returns:** Error if operation fails
  - **Thread-safe:** Implementation-dependent

- `Prune(maxTokens int) error`
  - Removes oldest messages to stay under token limit
  - **Parameters:**
    - `maxTokens`: Maximum tokens to keep
  - **Returns:** Error if operation fails
  - **Thread-safe:** Implementation-dependent

**Example:**

```go
mem := memory.NewConversationMemory(8000)

mem.Add(message.User("Hello"))
mem.Add(message.Assistant("Hi there!"))

messages := mem.GetMessages()
fmt.Printf("Total messages: %d\n", len(messages))
```

---

#### `Executor`

Interface for execution environments.

```go
type Executor interface {
    GetInput(ctx context.Context, prompt string) (string, error)
    DisplayOutput(ctx context.Context, content string) error
    DisplayThinking(ctx context.Context, thinking string) error
}
```

**Methods:**

- `GetInput(ctx context.Context, prompt string) (string, error)`
  - Gets user input with optional prompt
  - **Parameters:**
    - `ctx`: Context for cancellation
    - `prompt`: Prompt to display (can be empty)
  - **Returns:** User input or error
  - **Thread-safe:** Yes

- `DisplayOutput(ctx context.Context, content string) error`
  - Displays agent output to user
  - **Parameters:**
    - `ctx`: Context for cancellation
    - `content`: Content to display
  - **Returns:** Error if display fails
  - **Thread-safe:** Yes

- `DisplayThinking(ctx context.Context, thinking string) error`
  - Displays agent thinking/reasoning
  - **Parameters:**
    - `ctx`: Context for cancellation
    - `thinking`: Thinking content to display
  - **Returns:** Error if display fails
  - **Thread-safe:** Yes

**Example:**

```go
executor := cli.NewExecutor()

input, err := executor.GetInput(ctx, "What would you like to do?")
if err != nil {
    log.Fatal(err)
}

executor.DisplayOutput(ctx, "Processing your request...")
```

---

### Functions

#### `NewAgent`

Creates a new agent instance.

```go
func NewAgent(provider Provider, memory Memory, tools []Tool, opts ...AgentOption) (Agent, error)
```

**Parameters:**
- `provider`: LLM provider implementation
- `memory`: Memory implementation
- `tools`: Slice of available tools
- `opts`: Optional configuration

**Returns:**
- Configured agent instance or error

**Options:**
- `WithMaxIterations(n int)` - Set max iterations (default: 10)
- `WithSystemPrompt(prompt string)` - Custom system prompt
- `WithThinkingEnabled(enabled bool)` - Enable/disable thinking display

**Example:**

```go
agent, err := core.NewAgent(
    provider,
    memory,
    tools,
    core.WithMaxIterations(15),
    core.WithSystemPrompt("You are a helpful assistant"),
)
```

---

## Agent Package (`pkg/agent`)

### Types

#### `DefaultAgent`

Default implementation of the Agent interface.

```go
type DefaultAgent struct {
    provider      Provider
    memory        Memory
    tools         map[string]Tool
    maxIterations int
    systemPrompt  string
}
```

**Fields:**
- `provider`: LLM provider
- `memory`: Conversation memory
- `tools`: Map of tool name to tool
- `maxIterations`: Maximum iterations per turn
- `systemPrompt`: System message content

**Example:**

```go
agent := &agent.DefaultAgent{
    provider:      openai.NewProvider("gpt-4", key),
    memory:        memory.NewConversationMemory(8000),
    tools:         toolMap,
    maxIterations: 10,
    systemPrompt:  "You are a helpful assistant",
}
```

---

## Provider Package (`pkg/provider`)

### OpenAI Provider

#### `OpenAIProvider`

OpenAI and OpenAI-compatible LLM provider.

```go
type OpenAIProvider struct {
    client   *openai.Client
    model    string
    baseURL  string
    apiKey   string
}
```

**Methods:**

- `NewProvider(model, apiKey string, opts ...ProviderOption) *OpenAIProvider`
  - Creates new OpenAI provider
  - **Parameters:**
    - `model`: Model name (e.g., "gpt-4")
    - `apiKey`: OpenAI API key
    - `opts`: Optional configuration
  - **Returns:** Provider instance

**Options:**
- `WithBaseURL(url string)` - Use custom API endpoint
- `WithTemperature(temp float64)` - Set temperature (0.0-2.0)
- `WithMaxTokens(max int)` - Set max response tokens

**Example:**

```go
// Standard OpenAI
provider := openai.NewProvider("gpt-4", apiKey)

// OpenAI-compatible (e.g., Anyscale)
provider := openai.NewProvider(
    "meta-llama/Llama-2-70b-chat-hf",
    apiKey,
    openai.WithBaseURL("https://api.endpoints.anyscale.com/v1"),
)
```

---

## Tool Package (`pkg/tool`)

### Built-in Tools

#### `TaskCompletion`

Completes the current task and returns results to user.

```go
type TaskCompletion struct{}

func NewTaskCompletion() *TaskCompletion
```

**Parameters:**
- `result`: (string, required) The final result or answer

**Example:**

```go
tool := tool.NewTaskCompletion()
result, err := tool.Execute(ctx, map[string]interface{}{
    "result": "The calculation result is 42",
})
```

**Loop-breaking:** Yes

---

#### `AskQuestion`

Asks the user a question and waits for response.

```go
type AskQuestion struct {
    executor Executor
}

func NewAskQuestion(executor Executor) *AskQuestion
```

**Parameters:**
- `question`: (string, required) Question to ask user

**Example:**

```go
tool := tool.NewAskQuestion(executor)
result, err := tool.Execute(ctx, map[string]interface{}{
    "question": "What is your preferred color?",
})
```

**Loop-breaking:** No (continues after receiving answer)

---

#### `Converse`

Engages in multi-turn conversation with user.

```go
type Converse struct {
    executor Executor
}

func NewConverse(executor Executor) *Converse
```

**Parameters:**
- `message`: (string, required) Message to user

**Example:**

```go
tool := tool.NewConverse(executor)
result, err := tool.Execute(ctx, map[string]interface{}{
    "message": "Let me explain the next steps...",
})
```

**Loop-breaking:** No

---

## Memory Package (`pkg/memory`)

### Types

#### `ConversationMemory`

In-memory conversation history storage.

```go
type ConversationMemory struct {
    messages   []Message
    maxTokens  int
    mu         sync.RWMutex
}

func NewConversationMemory(maxTokens int) *ConversationMemory
```

**Parameters:**
- `maxTokens`: Maximum tokens before auto-pruning

**Methods:**

All methods from Memory interface, plus:

- `Size() int`
  - Returns number of messages
  - **Thread-safe:** Yes

- `EstimateTokens() int`
  - Estimates current token count
  - **Thread-safe:** Yes

**Example:**

```go
mem := memory.NewConversationMemory(8000)

mem.Add(message.User("Hello"))
mem.Add(message.Assistant("Hi there!"))

fmt.Printf("Messages: %d, Tokens: ~%d\n", mem.Size(), mem.EstimateTokens())
```

**Thread-safety:** All methods are thread-safe using RWMutex.

---

## Executor Package (`pkg/executor`)

### TUI Executor

#### `tui.NewExecutor`

Creates a new TUI (Terminal User Interface) executor.

```go
func NewExecutor(opts ...Option) (*Executor, error)
```

**Options:**

- `WithHeaderText(headerText string)`: Sets a custom string to be rendered as an ASCII art header on the welcome screen.
- `WithInitialPrompt(prompt string)`: Sets the initial prompt to be sent to the agent on startup.

**Example:**

```go
// TUI executor with a custom header
tuiExecutor, err := tui.NewExecutor(
    tui.WithHeaderText("My App"),
)
if err != nil {
    log.Fatalf("failed to create TUI executor: %v", err)
}

// Run the agent with the TUI
if err := agent.Run(context.Background(), tuiExecutor); err != nil {
    log.Fatalf("agent run failed: %v", err)
}
```

### CLI Executor

#### `CLIExecutor`

Command-line interface executor.

```go
type CLIExecutor struct {
    reader *bufio.Reader
    writer io.Writer
}

func NewExecutor() *CLIExecutor
```

**Example:**

```go
executor := cli.NewExecutor()

// Get user input
input, err := executor.GetInput(ctx, "Enter command: ")

// Display output
executor.DisplayOutput(ctx, "Processing complete")

// Display thinking
executor.DisplayThinking(ctx, "Analyzing the input...")
```

**Features:**
- Color-coded output
- Formatted thinking display
- Clean input prompts

---

## Message Package (`pkg/message`)

### Types

#### `Message`

Represents a conversation message.

```go
type Message struct {
    Role    string
    Content string
}
```

**Fields:**
- `Role`: Message role (user/assistant/system/tool)
- `Content`: Message content

### Functions

#### Message Constructors

```go
func User(content string) Message
func Assistant(content string) Message  
func System(content string) Message
func Tool(name, result string) Message
```

**Example:**

```go
messages := []message.Message{
    message.System("You are a helpful assistant"),
    message.User("What is 2+2?"),
    message.Assistant("2+2 equals 4"),
    message.Tool("calculator", "Result: 4"),
}
```

---

## Common Patterns

### Creating a Basic Agent

```go
// 1. Create provider
provider := openai.NewProvider("gpt-4", apiKey)

// 2. Create memory
memory := memory.NewConversationMemory(8000)

// 3. Create tools
tools := []core.Tool{
    tool.NewTaskCompletion(),
    tool.NewAskQuestion(executor),
    tool.NewConverse(executor),
}

// 4. Create agent
agent, err := core.NewAgent(provider, memory, tools)
if err != nil {
    log.Fatal(err)
}

// 5. Run agent
executor := cli.NewExecutor()
if err := agent.Run(ctx, executor); err != nil {
    log.Fatal(err)
}
```

### Custom Tool Implementation

```go
type WeatherTool struct{}

func (w *WeatherTool) Name() string {
    return "get_weather"
}

func (w *WeatherTool) Description() string {
    return "Gets current weather for a location"
}

func (w *WeatherTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "location": map[string]interface{}{
                "type":        "string",
                "description": "City name",
            },
        },
        "required": []string{"location"},
    }
}

func (w *WeatherTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    location := args["location"].(string)
    // Call weather API
    return fmt.Sprintf("Weather in %s: Sunny, 72°F", location), nil
}

func (w *WeatherTool) IsLoopBreaking() bool {
    return false
}
```

### Error Handling

```go
// Context timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()

// Run with error handling
if err := agent.Run(ctx, executor); err != nil {
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        log.Println("Agent timed out")
    case errors.Is(err, context.Canceled):
        log.Println("Agent was canceled")
    default:
        log.Printf("Agent error: %v", err)
    }
}
```

### Streaming Responses

```go
stream, err := provider.Stream(ctx, messages)
if err != nil {
    log.Fatal(err)
}

var fullResponse strings.Builder
for chunk := range stream {
    if chunk.Error != nil {
        log.Printf("Stream error: %v", chunk.Error)
        break
    }
    fullResponse.WriteString(chunk.Content)
    fmt.Print(chunk.Content) // Real-time display
}
```

## See Also

- [Getting Started](../getting-started/quick-start.md) - Basic usage
- [Architecture Overview](../architecture/overview.md) - System design
- [How-To Guides](../how-to/create-custom-tool.md) - Practical guides
- [Examples](../examples/calculator-agent.md) - Complete examples