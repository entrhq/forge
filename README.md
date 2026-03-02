# Forge

[![CI](https://github.com/entrhq/forge/workflows/CI/badge.svg)](https://github.com/entrhq/forge/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/entrhq/forge)](https://goreportcard.com/report/github.com/entrhq/forge)
[![GoDoc](https://pkg.go.dev/badge/github.com/entrhq/forge)](https://pkg.go.dev/github.com/entrhq/forge)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**Forge** is an open-source, lightweight agent framework for building AI agents with pluggable components. It provides a clean, modular architecture that makes it easy to create agents with different LLM providers and execution environments.

## Features

- 🔌 **Pluggable Architecture**: Interface-based design for maximum flexibility
- 🤖 **LLM Provider Abstraction**: Support for OpenAI-compatible APIs with extensibility for custom providers
- 🛠️ **Tool System**: Agent loop with tool execution and custom tool registration
- 🧠 **Chain-of-Thought**: Built-in thinking/reasoning capabilities for transparent agent behavior
- 💾 **Memory Management**: Conversation history and context management
- 🔄 **Event-Driven**: Real-time streaming of thinking, tool calls, and messages
- 🔁 **Self-Healing Error Recovery**: Automatic error recovery with circuit breaker pattern
- 🚀 **Execution Plane Abstraction**: Run agents in different environments (CLI, API, custom)
- 🤖 **Headless Mode**: Automated CI/CD workflows with git integration and PR creation
- 📚 **Automated Documentation**: AI-powered documentation updates on every PR
- 📦 **Library-First Design**: Import as a Go module in your own applications
- 🧪 **Well-Tested**: Comprehensive test coverage (196+ tests passing)
- 📖 **Well-Documented**: Clear, comprehensive documentation

## Quick Start

```bash
go get github.com/entrhq/forge
```

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/entrhq/forge/pkg/agent"
    "github.com/entrhq/forge/pkg/executor/tui"
    "github.com/entrhq/forge/pkg/llm/openai"
    "github.com/entrhq/forge/pkg/tools/coding"
)

func main() {
    // Create LLM provider
    provider, err := openai.NewProvider(
        os.Getenv("OPENAI_API_KEY"),
        openai.WithModel("gpt-4o"),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Create coding agent with built-in tools
    ag := agent.NewDefaultAgent(provider,
        agent.WithCustomInstructions("You are an expert coding assistant."),
    )
    
    // Register coding tools
    coding.RegisterTools(ag, ".")
    
    // Launch TUI executor
    executor := tui.NewExecutor(ag)
    if err := executor.Run(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## 🎯 Key Features

### 🖥️ Advanced Terminal UI

- **Real-time Streaming**: See agent thinking, tool calls, and responses stream in as they happen
- **Compact Header Bar**: Workspace path and active model shown at a glance; context-aware hints update with TUI state
- **Smart Scroll-Lock**: Scroll up to review history while the agent works; a banner appears when new content arrives, press `G` to jump back and resume auto-follow
- **Command Palette**: Instant slash command launcher via `Ctrl+K` / `Ctrl+P` or typing `/`; `Enter` executes immediately
- **Diff Viewer**: Interactive unified diff viewer with syntax highlighting for file changes
- **Tool Result History**: Browse all tool results from the session with `Ctrl+L`; inspect the latest result with `Ctrl+V`
- **Clipboard Copy**: Copy the full conversation as plain text with `Ctrl+Y`
- **Agent Thinking Blocks**: Extended reasoning shown inline with elapsed time; toggle visibility in settings
- **Interactive Settings**: Live configuration of LLM parameters, auto-approval rules, and UI preferences
- **Slash Commands**: Built-in commands for common workflows (`/commit`, `/pr`, `/bash`, `/notes`, `/help`, `/snapshot`)

### 🛠️ Complete Coding Toolkit

**File Operations:**
- `read_file` - Read files with optional line ranges
- `write_file` - Create or overwrite files with automatic directory creation
- `list_files` - List and filter files with glob patterns and recursive search
- `search_files` - Regex search across files with context lines

**Code Manipulation:**
- `apply_diff` - Surgical code edits with search/replace operations
- `execute_command` - Run shell commands with streaming output and timeout control

**Agent Control:**
- `task_completion` - Mark tasks complete and present results
- `ask_question` - Request clarifying information from users
- `converse` - Engage in natural conversation

### 🔐 Security & Control

- **Workspace Guard**: Prevent operations outside designated directories
- **File Ignore System**: Respect `.gitignore` and custom ignore patterns
- **Tool Approval**: Review and approve tool executions before running
- **Auto-Approval Whitelist**: Configure safe operations to run automatically
- **Command Timeout**: Prevent runaway processes with configurable timeouts

### 🧠 Smart Context Management

- **Token-Based Pruning**: Automatic conversation history management
- **Tool Call Summarization**: Condense tool results to preserve context
- **Composable Strategies**: Mix and match context management approaches
- **Threshold-Based Trimming**: Keep conversation within model limits

### 🔄 Git Workflow Integration

- **Automated Commits**: Review and commit changes directly from the TUI
- **Pull Request Creation**: Generate PRs with AI-written descriptions
- **Headless CI/CD**: Run Forge in GitHub Actions and other CI/CD pipelines
- **Automated PR Documentation**: Auto-update docs when PRs are opened
- **Change Tracking**: Monitor file modifications across agent sessions
- **Diff Preview**: View changes before committing

## 📚 Documentation

### Getting Started
- [Installation](docs/getting-started/installation.md) - Setup and configuration
- [Quick Start](docs/getting-started/quick-start.md) - Build your first agent
- [Your First Agent](docs/getting-started/your-first-agent.md) - Detailed tutorial
- [Understanding the Agent Loop](docs/getting-started/understanding-agent-loop.md) - Core concepts

### How-To Guides
- [Use TUI Interface](docs/how-to/use-tui-interface.md) - Full TUI guide (scroll-lock, clipboard, thinking blocks, overlays)
- [Configure Provider](docs/how-to/configure-provider.md) - LLM provider setup
- [Create Custom Tools](docs/how-to/create-custom-tool.md) - Extend agent capabilities
- [Setup PR Documentation](docs/how-to/setup-pr-documentation.md) - Automated docs workflow
- [Manage Memory](docs/how-to/manage-memory.md) - Context and history management
- [Handle Errors](docs/how-to/handle-errors.md) - Error recovery patterns
- [Test Tools](docs/how-to/test-tools.md) - Testing strategies
- [Optimize Performance](docs/how-to/optimize-performance.md) - Performance tuning
- [Deploy to Production](docs/how-to/deploy-production.md) - Production deployment

### Architecture
- [Overview](docs/architecture/overview.md) - System architecture
- [Agent Loop](docs/architecture/agent-loop.md) - Agent execution model
- [Tool System](docs/architecture/tool-system.md) - Tool architecture
- [Memory System](docs/architecture/memory-system.md) - Memory management

### Reference
- [API Reference](docs/reference/api-reference.md) - Complete API docs
- [Configuration](docs/reference/configuration.md) - All config options
- [Tool Schema](docs/reference/tool-schema.md) - Tool definition format
- [Message Format](docs/reference/message-format.md) - Message structure
- [Error Handling](docs/reference/error-handling.md) - Error types and recovery
- [Performance](docs/reference/performance.md) - Performance characteristics
- [Testing](docs/reference/testing.md) - Testing framework
- [Glossary](docs/reference/glossary.md) - Technical terms

### Architecture Decision Records
See [ADRs](docs/adr/) for detailed design decisions including:
- [TUI Visual Redesign](docs/adr/0051-tui-visual-redesign.md)
- [TUI Clipboard Copy](docs/adr/0050-tui-clipboard-copy.md)
- [TUI Bracketed Paste](docs/adr/0049-tui-bracketed-paste-support.md)
- [TUI Smart Scroll-Lock](docs/adr/0048-tui-smart-scroll-lock.md)
- [Automated PR Documentation](docs/adr/0030-automated-pr-documentation.md)
- [XML Tool Call Format](docs/adr/0019-xml-cdata-tool-call-format.md)
- [Auto-Approval System](docs/adr/0017-auto-approval-and-settings-system.md)
- [Context Management](docs/adr/0014-composable-context-management.md)
- [Streaming Commands](docs/adr/0013-streaming-command-execution.md)
- [TUI Design](docs/adr/0012-enhanced-tui-executor.md)

## 🏗️ Architecture

Forge is built with a clean, modular architecture designed for extensibility:

```
pkg/
├── agent/          # Agent core, loop, prompts, memory
│   ├── context/    # Context management strategies
│   ├── git/        # Git integration
│   ├── memory/     # Conversation history
│   ├── prompts/    # Dynamic prompt assembly
│   ├── slash/      # Slash command system
│   └── tools/      # Tool interface and built-ins
├── executor/       # Execution environments
│   ├── cli/        # Command-line executor
│   └── tui/        # Terminal UI executor
├── llm/            # LLM provider abstraction
│   ├── openai/     # OpenAI implementation
│   ├── parser/     # Response parsing
│   └── tokenizer/  # Token counting
├── tools/          # Tool implementations
│   └── coding/     # File and command tools
├── config/         # Configuration management
├── security/       # Security features
│   └── workspace/  # Workspace isolation
└── types/          # Shared types and events
```

### Key Design Principles

- **Interface-First**: Clean abstractions for maximum flexibility
- **Event-Driven**: Real-time streaming of agent activities
- **Composable**: Mix and match components as needed
- **Testable**: Comprehensive test coverage (200+ tests)
- **Type-Safe**: Strong typing with Go's type system

## 💡 Examples

### Basic Coding Agent

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/entrhq/forge/pkg/agent"
    "github.com/entrhq/forge/pkg/executor/cli"
    "github.com/entrhq/forge/pkg/llm/openai"
    "github.com/entrhq/forge/pkg/tools/coding"
)

func main() {
    provider, _ := openai.NewProvider(os.Getenv("OPENAI_API_KEY"))
    
    ag := agent.NewDefaultAgent(provider,
        agent.WithCustomInstructions("You are a helpful coding assistant."),
        agent.WithMaxIterations(50),
    )
    
    // Register all coding tools
    coding.RegisterTools(ag, ".")
    
    executor := cli.NewExecutor(ag)
    executor.Run(context.Background())
}
```

### With Custom Tools

```go
type WeatherTool struct{}

func (t *WeatherTool) Name() string { return "get_weather" }
func (t *WeatherTool) Description() string {
    return "Get current weather for a location"
}
func (t *WeatherTool) Parameters() map[string]interface{} {
    return map[string]interface{}{
        "location": map[string]interface{}{
            "type":        "string",
            "description": "City name",
            "required":    true,
        },
    }
}
func (t *WeatherTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
    location := args["location"].(string)
    // Call weather API...
    return fmt.Sprintf("Weather in %s: Sunny, 72°F", location), nil
}

// Register custom tool
ag.RegisterTool(&WeatherTool{})
```

See [examples/](examples/) for complete working examples.

## 🔧 Development

### Prerequisites

- Go 1.21 or higher
- Make (optional, recommended)
- Git

### Setup

```bash
# Clone repository
git clone https://github.com/entrhq/forge.git
cd forge

# Install development tools
make install-tools

# Run tests
make test

# Run linter
make lint

# Build CLI
make build
```

### Make Targets

- `make test` - Run all tests with coverage
- `make lint` - Run linters (golangci-lint)
- `make fmt` - Format code
- `make build` - Build Forge CLI
- `make install` - Install Forge CLI
- `make clean` - Clean build artifacts
- `make all` - Run all checks and build

### Running Tests

```bash
# All tests
make test

# Specific package
go test ./pkg/agent/...

# With coverage
go test -cover ./...

# Verbose
go test -v ./pkg/tools/coding/
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Areas for Contribution

- 🔌 Additional LLM provider implementations (Anthropic, Google, etc.)
- 🛠️ New tool implementations
- 📝 Documentation improvements
- 🐛 Bug fixes and performance improvements
- ✨ New features and enhancements

### Code of Conduct

By participating, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

### Security

For security issues, see our [Security Policy](SECURITY.md).

## 🗺️ Roadmap

### ✅ Completed

- [x] Core agent loop with tool execution
- [x] OpenAI provider with streaming support
- [x] Advanced TUI with syntax highlighting
- [x] Complete coding toolkit (read, write, diff, search, execute)
- [x] Context management with multiple strategies
- [x] Auto-approval and settings system
- [x] Git integration (commits and PRs)
- [x] Headless mode for CI/CD automation
- [x] Automated PR documentation workflow
- [x] File ignore system for security
- [x] Slash command system
- [x] Self-healing error recovery
- [x] XML/CDATA tool call format
- [x] Comprehensive test suite (200+ tests)

### 🚧 In Progress

- [ ] Additional LLM providers (Anthropic Claude, Google Gemini)
- [ ] Multi-agent coordination and handoffs
- [ ] Advanced executor implementations (HTTP API, Slack bot)
- [ ] Plugin system for third-party tools
- [ ] Enhanced memory with vector storage

### 🔮 Planned

- [ ] Web-based UI
- [ ] Agent marketplace and sharing
- [ ] Observability and monitoring
- [ ] Advanced debugging tools
- [ ] Multi-modal support (vision, audio)

See [ROADMAP.md](ROADMAP.md) for detailed plans.

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Built with ❤️ as part of the Entr Agent Platform.
## 🔗 Links

- 📖 [Documentation](docs/) - Complete documentation
- 💻 [Examples](examples/) - Working code examples  
- 🐛 [Issues](https://github.com/entrhq/forge/issues) - Report bugs or request features
- 💬 [Discussions](https://github.com/entrhq/forge/discussions) - Ask questions and share ideas
- 📝 [Changelog](CHANGELOG.md) - Version history
- 🤝 [Contributing](CONTRIBUTING.md) - Contribution guidelines
