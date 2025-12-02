.PHONY: test lint fmt clean examples run-example help install-tools tidy install uninstall build validate-workflows

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Directories
EXAMPLES_DIR=./examples
AGENT_CHAT=$(EXAMPLES_DIR)/agent-chat

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

test: ## Run tests with coverage
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "\nCoverage report:"
	$(GOCMD) tool cover -func=coverage.out

test-coverage: test ## Run tests and generate HTML coverage report
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linters
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run 'make install-tools' to install."; \
		exit 1; \
	fi

validate-workflows: ## Validate GitHub Actions workflows
	@echo "Validating GitHub Actions workflows..."
	@if command -v actionlint > /dev/null; then \
		actionlint .github/workflows/*.yml; \
	elif command -v gh > /dev/null; then \
		for workflow in .github/workflows/*.yml; do \
			echo "Validating $${workflow}..."; \
			gh workflow view "$$(basename "$${workflow}")" --yaml > /dev/null 2>&1 && echo "✓ $$(basename "$${workflow}")" || (echo "✗ $$(basename "$${workflow}")"; exit 1); \
		done; \
	else \
		echo "Neither actionlint nor gh CLI found. Install one of:"; \
		echo "  - actionlint: brew install actionlint"; \
		echo "  - gh: brew install gh"; \
		exit 1; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	$(GOCMD) fmt ./...

tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	$(GOMOD) tidy

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f coverage.out coverage.html
	rm -rf .bin/ dist/ build/

examples: ## Build all example applications
	@echo "Building examples..."
	@mkdir -p .bin
	$(GOBUILD) -o .bin/agent-chat $(AGENT_CHAT)
	@echo "Examples built in .bin/"

run-example: ## Run the agent chat example
	@echo "Running agent chat example..."
	@echo "Note: Make sure OPENAI_API_KEY is set in your environment"
	$(GOCMD) run $(AGENT_CHAT)/main.go

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin; \
	else \
		echo "golangci-lint already installed"; \
	fi

run: ## Run the forge TUI coding agent
	@echo "Running forge..."
	@echo "Note: Make sure OPENAI_API_KEY is set in your environment"
	$(GOCMD) run ./cmd/forge

build: ## Build forge binary to .bin directory
	@echo "Building forge..."
	@mkdir -p .bin
	$(GOBUILD) -o .bin/forge ./cmd/forge
	@echo "✓ forge built successfully at .bin/forge"

install: ## Install forge binary to GOPATH/bin
	@echo "Installing forge to $(shell go env GOPATH)/bin..."
	$(GOCMD) install ./cmd/forge
	@echo "✓ forge installed successfully"
	@echo "  Run 'forge' from anywhere to start the TUI coding agent"

uninstall: ## Remove forge binary from GOPATH/bin
	@echo "Uninstalling forge..."
	rm -f $(shell go env GOPATH)/bin/forge
	@echo "✓ forge uninstalled"

dev: run-example ## Run in development mode (alias for run-example)

all: tidy fmt lint validate-workflows test examples ## Run all checks and build examples

.DEFAULT_GOAL := help