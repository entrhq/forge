# Repository Context for Forge AI Agent

This file provides repository-specific context for the Forge AI coding assistant.

## Project Overview

Forge is an AI-powered coding assistant that uses LLM providers to help with software development tasks. It operates in both interactive and headless modes, providing tools for file manipulation, code editing, command execution, and more.

## Architecture

- **Agent System**: Core agent loop in `pkg/agent/` handles LLM interactions, tool execution, and memory management
- **Tools**: Modular tool system in `pkg/agent/tools/` provides capabilities like file operations, search, and execution
- **Prompts**: Dynamic prompt building in `pkg/agent/prompts/` constructs context-aware system prompts
- **Context Management**: Automatic context summarization to handle large conversations
- **Memory System**: Scratchpad notes for persistent working memory across agent iterations

## Key Components

### Agent Loop
The agent operates in an iterative loop:
1. Analyze events and user input
2. Think through the problem
3. Select and execute tools
4. Present results or ask questions

### Tool System
Tools are the primary way the agent interacts with the codebase:
- **File Operations**: read_file, write_file, list_files
- **Code Editing**: apply_diff for surgical edits
- **Search**: search_files for pattern matching
- **Execution**: execute_command for running shell commands
- **Memory**: Scratchpad notes for working memory

### Modes
- **Interactive**: Full conversational mode with user interaction
- **Headless**: Automated execution mode with disabled interactive tools

## Development Guidelines

1. **Modularity**: Keep files focused and single-responsibility
2. **Testing**: Maintain comprehensive test coverage
3. **Error Handling**: Use robust error handling with clear messages
4. **Security**: Workspace guard prevents path traversal attacks
5. **Performance**: Efficient context management and token usage

## Current Focus

This repository context feature itself - allowing agents to understand project-specific information from this file.
