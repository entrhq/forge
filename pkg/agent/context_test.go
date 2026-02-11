package agent

import (
	"testing"

	"github.com/entrhq/forge/pkg/llm/tokenizer"
)

// mustNewTokenizer creates a tokenizer or skips the test if initialization fails
func mustNewTokenizer() *tokenizer.Tokenizer {
	tok, err := tokenizer.New()
	if err != nil {
		return nil
	}
	return tok
}

// TestGetContextInfo_RepositoryContext verifies that repository context tokens are counted correctly
func TestGetContextInfo_RepositoryContext(t *testing.T) {
	// Create a test agent using the constructor
	provider := &mockProvider{}
	agent := NewDefaultAgent(provider)

	// Set test-specific values
	agent.customInstructions = "Test custom instructions"
	agent.repositoryContext = "This is test repository context that should be counted"

	if agent.tokenizer == nil {
		t.Skip("Tokenizer initialization failed, skipping test")
	}

	// Get context info
	info := agent.GetContextInfo()

	// Verify repository context tokens are counted
	if info.RepositoryContextTokens == 0 {
		t.Error("Expected RepositoryContextTokens to be > 0, got 0")
	}

	// Verify the token count is reasonable (should be around 10-15 tokens for the test string)
	if info.RepositoryContextTokens < 5 || info.RepositoryContextTokens > 30 {
		t.Errorf("Expected RepositoryContextTokens to be between 5 and 30, got %d", info.RepositoryContextTokens)
	}

	t.Logf("Repository context tokens: %d", info.RepositoryContextTokens)
}

// TestGetContextInfo_NoRepositoryContext verifies that when no repository context is set, tokens are 0
func TestGetContextInfo_NoRepositoryContext(t *testing.T) {
	// Create a test agent using the constructor
	provider := &mockProvider{}
	agent := NewDefaultAgent(provider)

	// Set test-specific values
	agent.customInstructions = "Test custom instructions"
	agent.repositoryContext = "" // Empty repository context

	if agent.tokenizer == nil {
		t.Skip("Tokenizer initialization failed, skipping test")
	}

	// Get context info
	info := agent.GetContextInfo()

	// Verify repository context tokens are 0
	if info.RepositoryContextTokens != 0 {
		t.Errorf("Expected RepositoryContextTokens to be 0 for empty context, got %d", info.RepositoryContextTokens)
	}
}
