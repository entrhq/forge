# AGENTS.md

## Project Overview

Forge is an open-source AI coding agent framework in Go with LLM provider abstraction, tool system, memory management, and multiple execution modes (CLI, TUI, headless).

## Quick Commands

```bash
# Development
make test            # Run tests with coverage
make lint            # Run golangci-lint
make fmt             # Format code
make all             # Run all checks

# Building
make build           # Build to .bin/forge
make install         # Install to GOPATH/bin
make run             # Run in dev mode

# Tools
make install-tools   # Install golangci-lint
```

## Code Standards

- **Go 1.24.0+** with interface-first design
- **Small, focused files** - prefer modular decomposition
- **Table-driven tests** - comprehensive coverage (>80% for new code)
- **Security-first** - workspace guard, input validation, tool approval

## Architecture

```
pkg/
├── agent/       # Core loop, prompts, memory, git, slash commands
├── executor/    # CLI and TUI execution environments
├── llm/         # Provider abstraction, XML parser, tokenizer
├── tools/       # File ops, search, diff, command execution
├── config/      # Configuration management
├── security/    # Workspace isolation
└── types/       # Shared types and events
```

**Key Concepts:**
- XML tool calls with CDATA (ADR 0019)
- Context management with summarization (ADR 0014)
- Scratchpad notes for working memory (ADR 0032)
- Git integration with /commit and /pr (ADR 0030, 0031)
- Headless mode for automation (ADR 0026)

## Common Tasks

**Add Tool:** Implement `tools.Tool` interface → Add to pkg/tools/ → Write tests → Register
**Run Tests:** `make test` (all), `go test -v ./pkg/agent/...` (package), `go test -run TestName` (single)
**Pre-commit:** `make fmt && make lint && make test` - don't introduce new linter violations

## Feature Development

Documentation-first workflow:
1. **Scratch** (docs/product/scratch/) - Rough concept with problem/solution
2. **PRD** (docs/product/features/) - Full requirements, users, metrics
3. **ADR** (docs/adr/) - Technical design following template
4. **Implementation** - Code with tests, link to ADR/PRD in PR

PRDs define "what" and "why", ADRs define "how". See existing files for examples.

## Documentation Structure

- `docs/adr/` - Architecture decisions (numbered, immutable)
- `docs/product/features/` - Product requirements (PRDs)
- `docs/product/scratch/` - Feature ideas (brainstorming)
- `docs/how-to/` - How-to guides
- `examples/` - Example code
