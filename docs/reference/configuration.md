# Configuration Reference

Complete reference for configuring Forge agents.

## Table of Contents

- [Agent Configuration](#agent-configuration)
- [Provider Configuration](#provider-configuration)
- [Memory Configuration](#memory-configuration)
- [Tool Configuration](#tool-configuration)
- [Executor Configuration](#executor-configuration)
- [Environment Variables](#environment-variables)
- [Configuration Examples](#configuration-examples)

---

## Agent Configuration

### Agent Options

Configure agent behavior using functional options:

```go
agent, err := core.NewAgent(
    provider,
    memory,
    tools,
    core.WithMaxIterations(15),
    core.WithSystemPrompt("Custom system prompt"),
    core.WithThinkingEnabled(true),
)
```

### `WithMaxIterations`

Sets maximum iterations per turn to prevent infinite loops.

```go
func WithMaxIterations(max int) AgentOption
```

**Parameters:**
- `max`: Maximum number of iterations (must be > 0)

**Default:** 10

**Recommendations:**
- Simple tasks: 5-10 iterations
- Complex tasks: 15-20 iterations
- Very complex: 25-30 iterations

**Example:**

```go
// Simple chatbot
core.WithMaxIterations(5)

// Complex reasoning tasks
core.WithMaxIterations(20)
```

**Trade-offs:**
- Higher: More capable but costs more
- Lower: Cheaper but may fail on complex tasks

---

### `WithSystemPrompt`

Sets custom system prompt for agent behavior.

```go
func WithSystemPrompt(prompt string) AgentOption
```

**Parameters:**
- `prompt`: System message content

**Default:** Generic helpful assistant prompt

**Example:**

```go
systemPrompt := `You are a Python coding assistant.
- Write clean, idiomatic Python code
- Include type hints
- Follow PEP 8 style guide
- Explain your code choices`

agent, err := core.NewAgent(
    provider,
    memory,
    tools,
    core.WithSystemPrompt(systemPrompt),
)
```

**Best Practices:**
- Be specific about role and constraints
- Include formatting preferences
- Specify tone and style
- List prohibited behaviors

---

### `WithThinkingEnabled`

Controls whether agent displays thinking/reasoning.

```go
func WithThinkingEnabled(enabled bool) AgentOption
```

**Parameters:**
- `enabled`: true to show thinking, false to hide

**Default:** true

**Example:**

```go
// Show thinking (default)
core.WithThinkingEnabled(true)

// Hide thinking for cleaner output
core.WithThinkingEnabled(false)
```

**When to Disable:**
- Production user-facing applications
- When output is processed by code
- Minimal output requirements

**When to Enable:**
- Development and debugging
- Transparency to users
- Educational purposes

---

## Provider Configuration

### OpenAI Provider Options

```go
provider := openai.NewProvider(
    "gpt-4",
    apiKey,
    openai.WithBaseURL("https://custom.api.com/v1"),
    openai.WithTemperature(0.7),
    openai.WithMaxTokens(2000),
    openai.WithTimeout(60*time.Second),
)
```

### `WithBaseURL`

Sets custom API endpoint for OpenAI-compatible services.

```go
func WithBaseURL(url string) ProviderOption
```

**Parameters:**
- `url`: Base URL for API (must end with /v1)

**Default:** `https://api.openai.com/v1`

**Common Values:**

```go
// Anyscale
openai.WithBaseURL("https://api.endpoints.anyscale.com/v1")

// Together AI
openai.WithBaseURL("https://api.together.xyz/v1")

// LocalAI
openai.WithBaseURL("http://localhost:8080/v1")

// Ollama (with OpenAI compatibility)
openai.WithBaseURL("http://localhost:11434/v1")
```

---

### `WithTemperature`

Controls randomness in model responses.

```go
func WithTemperature(temp float64) ProviderOption
```

**Parameters:**
- `temp`: Temperature value (0.0 to 2.0)

**Default:** 0.7

**Values:**
- `0.0`: Deterministic, focused
- `0.3-0.5`: Consistent, slight variation
- `0.7-0.9`: Balanced creativity
- `1.0-2.0`: Very creative, unpredictable

**Examples:**

```go
// Code generation - consistent
openai.WithTemperature(0.2)

// Creative writing
openai.WithTemperature(1.2)

// Balanced assistant
openai.WithTemperature(0.7)
```

---

### `WithMaxTokens`

Sets maximum tokens in model response.

```go
func WithMaxTokens(max int) ProviderOption
```

**Parameters:**
- `max`: Maximum response tokens

**Default:** Model's maximum (e.g., 4096 for GPT-4)

**Recommendations:**
- Short answers: 500-1000 tokens
- Medium responses: 1000-2000 tokens
- Long-form content: 2000-4000 tokens

**Example:**

```go
// Concise responses
openai.WithMaxTokens(500)

// Detailed explanations
openai.WithMaxTokens(2000)
```

**Note:** This is response length, not total context length.

---

### `WithTimeout`

Sets HTTP timeout for API requests.

```go
func WithTimeout(duration time.Duration) ProviderOption
```

**Parameters:**
- `duration`: Timeout duration

**Default:** 30 seconds

**Example:**

```go
// Quick responses only
openai.WithTimeout(10 * time.Second)

// Allow long responses
openai.WithTimeout(120 * time.Second)
```

---

### Model Selection

Choose appropriate model for your use case:

```go
// GPT-4 Turbo - Best reasoning
provider := openai.NewProvider("gpt-4-turbo-preview", apiKey)

// GPT-4 - Highly capable
provider := openai.NewProvider("gpt-4", apiKey)

// GPT-3.5 Turbo - Fast and economical
provider := openai.NewProvider("gpt-3.5-turbo", apiKey)
```

**Model Comparison:**

| Model | Speed | Cost | Capability | Best For |
|-------|-------|------|------------|----------|
| gpt-4-turbo | Medium | $$$ | Excellent | Complex reasoning |
| gpt-4 | Slow | $$$$ | Excellent | Highest quality |
| gpt-3.5-turbo | Fast | $ | Good | Simple tasks |

---

### Summarization and Browser Analysis Models

In addition to the primary model, you can specify distinct models for summarization and browser analysis tasks. This is useful for cost optimization, as these tasks can often be handled by smaller, faster models.

These settings are typically managed via the `config.yaml` file or the TUI settings (`/settings`).

#### `summarization_model`
- **Type**: `string`
- **Default**: The primary `model`
- **Description**: The model to use for all context summarization and compaction tasks. If not set, it defaults to the main agent model.
- **Example**: `"anthropic/claude-haiku-3-5"`

#### `browser_analysis_model`
- **Type**: `string`
- **Default**: The primary `model`
- **Description**: The model used by the `browser/analyze_page` tool for web page analysis. If not set, it defaults to the main agent model.
- **Example**: `"anthropic/claude-haiku-3-5"`

**Example `config.yaml`:**
```yaml
llm:
  model: "anthropic/claude-sonnet-4.5"
  summarization_model: "anthropic/claude-haiku-3-5"
  browser_analysis_model: "anthropic/claude-haiku-3-5"
  base_url: "https://openrouter.ai/api/v1"
  api_key: "sk-..."
```

---

## Memory Configuration

### ConversationMemory Options

```go
memory := memory.NewConversationMemory(
    8000,  // maxTokens
    memory.WithPruningStrategy(memory.OldestFirst),
    memory.WithSystemMessageRetention(true),
)
```

### Token Limit

Maximum tokens before automatic pruning.

```go
memory := memory.NewConversationMemory(maxTokens)
```

**Parameters:**
- `maxTokens`: Maximum tokens to keep in memory

**Recommendations:**

| Model | Context Window | Recommended Limit | Reason |
|-------|---------------|-------------------|--------|
| GPT-4 | 8,192 | 6,000-7,000 | Leave room for response |
| GPT-4-32k | 32,768 | 28,000-30,000 | Leave room for response |
| GPT-3.5 | 4,096 | 3,000-3,500 | Leave room for response |

**Example:**

```go
// For GPT-4
memory := memory.NewConversationMemory(7000)

// For GPT-4-32k (long conversations)
memory := memory.NewConversationMemory(28000)

// For GPT-3.5
memory := memory.NewConversationMemory(3500)
```

---

### Pruning Strategy

How memory removes old messages:

```go
func WithPruningStrategy(strategy PruningStrategy) MemoryOption
```

**Strategies:**

1. **OldestFirst** (default)
   - Removes oldest messages first
   - Preserves recent context
   - Best for most use cases

2. **ScoreBasedPruning**
   - Removes least important messages
   - Preserves key information
   - More complex, slower

**Example:**

```go
// Simple oldest-first
memory.WithPruningStrategy(memory.OldestFirst)

// Keep important messages
memory.WithPruningStrategy(memory.ScoreBasedPruning)
```

---

### System Message Retention

Whether to always keep system message:

```go
func WithSystemMessageRetention(retain bool) MemoryOption
```

**Parameters:**
- `retain`: true to always keep system message

**Default:** true

**Example:**

```go
// Always keep system message (recommended)
memory.WithSystemMessageRetention(true)

// Allow pruning system message
memory.WithSystemMessageRetention(false)
```

**Recommendation:** Keep `true` unless you have specific reasons.

---

## Tool Configuration

### Tool Registration

Register tools with agent:

```go
tools := []core.Tool{
    tool.NewTaskCompletion(),
    tool.NewAskQuestion(executor),
    tool.NewConverse(executor),
    custom.NewWeatherTool(),
    custom.NewCalculatorTool(),
}

agent, err := core.NewAgent(provider, memory, tools)
```

### Custom Tool Parameters

Define JSON Schema for tool parameters:

```go
func (t *MyTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param1": map[string]interface{}{
                "type":        "string",
                "description": "Description for LLM",
                "enum":        []string{"option1", "option2"},
            },
            "param2": map[string]interface{}{
                "type":        "number",
                "description": "Numeric parameter",
                "minimum":     0,
                "maximum":     100,
            },
        },
        "required": []string{"param1"},
    }
}
```

**Schema Properties:**

- `type`: Data type (string, number, boolean, object, array)
- `description`: Help text for LLM
- `enum`: Allowed values
- `minimum`/`maximum`: Numeric constraints
- `pattern`: Regex pattern for strings
- `required`: Required parameter names

---

## Executor Configuration

### CLI Executor

```go
executor := cli.NewExecutor(
    cli.WithColorOutput(true),
    cli.WithPromptPrefix("> "),
    cli.WithThinkingColor(color.Cyan),
)
```

### Color Output

Enable/disable colored terminal output:

```go
func WithColorOutput(enabled bool) ExecutorOption
```

**Default:** true (if terminal supports it)

**Example:**

```go
// Colored output
cli.WithColorOutput(true)

// Plain text (for logging, pipes)
cli.WithColorOutput(false)
```

---

### Custom Prompt

Set custom input prompt:

```go
func WithPromptPrefix(prefix string) ExecutorOption
```

**Default:** `"You: "`

**Example:**

```go
cli.WithPromptPrefix("> ")
cli.WithPromptPrefix("User: ")
cli.WithPromptPrefix("🤖 ")
```

---

## Environment Variables

### Required Variables

```bash
# OpenAI API Key (required for OpenAI provider)
export OPENAI_API_KEY="sk-..."

# Optional: Custom OpenAI endpoint
export OPENAI_BASE_URL="https://custom.api.com/v1"

# Optional: Default model
export OPENAI_MODEL="gpt-4"
```

### Reading in Code

```go
import "os"

apiKey := os.Getenv("OPENAI_API_KEY")
if apiKey == "" {
    log.Fatal("OPENAI_API_KEY not set")
}

model := os.Getenv("OPENAI_MODEL")
if model == "" {
    model = "gpt-4" // default
}

provider := openai.NewProvider(model, apiKey)
```

---

## Configuration Examples

### Development Configuration

Fast iteration, verbose output:

```go
provider := openai.NewProvider(
    "gpt-3.5-turbo",
    apiKey,
    openai.WithTemperature(0.7),
    openai.WithTimeout(30*time.Second),
)

memory := memory.NewConversationMemory(4000)

agent, err := core.NewAgent(
    provider,
    memory,
    tools,
    core.WithMaxIterations(10),
    core.WithThinkingEnabled(true),
)

executor := cli.NewExecutor(
    cli.WithColorOutput(true),
)
```

---

### Production Configuration

Robust, efficient, user-facing:

```go
provider := openai.NewProvider(
    "gpt-4-turbo-preview",
    apiKey,
    openai.WithTemperature(0.5),
    openai.WithMaxTokens(2000),
    openai.WithTimeout(60*time.Second),
)

memory := memory.NewConversationMemory(
    28000,
    memory.WithPruningStrategy(memory.OldestFirst),
    memory.WithSystemMessageRetention(true),
)

systemPrompt := `You are a professional customer service agent.
- Be polite and helpful
- Provide accurate information
- If unsure, ask for clarification
- Keep responses concise`

agent, err := core.NewAgent(
    provider,
    memory,
    tools,
    core.WithMaxIterations(15),
    core.WithSystemPrompt(systemPrompt),
    core.WithThinkingEnabled(false), // Clean output
)

executor := cli.NewExecutor(
    cli.WithColorOutput(true),
    cli.WithPromptPrefix("You: "),
)
```

---

### Cost-Optimized Configuration

Minimize API costs:

```go
provider := openai.NewProvider(
    "gpt-3.5-turbo",
    apiKey,
    openai.WithTemperature(0.3),
    openai.WithMaxTokens(500),
)

memory := memory.NewConversationMemory(2000) // Smaller context

agent, err := core.NewAgent(
    provider,
    memory,
    tools,
    core.WithMaxIterations(5), // Fewer iterations
)
```

---

### High-Performance Configuration

For complex reasoning:

```go
provider := openai.NewProvider(
    "gpt-4-turbo-preview",
    apiKey,
    openai.WithTemperature(0.7),
    openai.WithMaxTokens(4000),
    openai.WithTimeout(120*time.Second),
)

memory := memory.NewConversationMemory(30000) // Large context

agent, err := core.NewAgent(
    provider,
    memory,
    tools,
    core.WithMaxIterations(30), // Many iterations for complex tasks
    core.WithThinkingEnabled(true),
)
```

---

### Local Model Configuration

Using Ollama or LocalAI:

```go
provider := openai.NewProvider(
    "llama2",
    "not-needed", // Local models may not need key
    openai.WithBaseURL("http://localhost:11434/v1"),
    openai.WithTemperature(0.7),
    openai.WithTimeout(120*time.Second), // Local can be slower
)

memory := memory.NewConversationMemory(4000)

agent, err := core.NewAgent(
    provider,
    memory,
    tools,
    core.WithMaxIterations(10),
)
```

---

## Configuration Best Practices

### 1. Use Environment Variables

```go
// ✅ Good: Configurable
apiKey := os.Getenv("OPENAI_API_KEY")

// ❌ Bad: Hardcoded
apiKey := "sk-hardcoded-key"
```

### 2. Set Appropriate Timeouts

```go
// ✅ Good: Reasonable timeout
openai.WithTimeout(60 * time.Second)

// ❌ Bad: Too short
openai.WithTimeout(1 * time.Second)
```

### 3. Match Memory to Model Context

```go
// ✅ Good: Leaves room for response
memory := memory.NewConversationMemory(7000) // GPT-4 has 8K context

// ❌ Bad: Uses full context
memory := memory.NewConversationMemory(8000) // No room for response
```

### 4. Choose Appropriate Temperature

```go
// ✅ Good: Low for factual tasks
openai.WithTemperature(0.2) // For code generation

// ✅ Good: Higher for creative tasks
openai.WithTemperature(1.0) // For story writing

// ❌ Bad: Too high for factual tasks
openai.WithTemperature(2.0) // Too random for facts
```

### 5. Limit Iterations Appropriately

```go
// ✅ Good: Reasonable limits
core.WithMaxIterations(15) // Most tasks

// ❌ Bad: Too high (costly)
core.WithMaxIterations(100)

// ❌ Bad: Too low (fails complex tasks)
core.WithMaxIterations(2)
```

---

## See Also

- [API Reference](api-reference.md) - Complete API documentation
- [Quick Start](../getting-started/quick-start.md) - Getting started
- [Architecture Overview](../architecture/overview.md) - System design
- [How-To Guides](../how-to/) - Practical guides