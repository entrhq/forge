// Package openai provides an OpenAI-compatible LLM provider implementation.
//
// Example usage:
//
//	package main
//
//	import (
//	    "context"
//	    "fmt"
//	    "os"
//
//	    "github.com/entrhq/forge/pkg/llm/openai"
//	    "github.com/entrhq/forge/pkg/types"
//	)
//
//	func main() {
//	    provider, err := openai.NewProvider(
//	        os.Getenv("OPENAI_API_KEY"),
//	        openai.WithModel("gpt-4o"),
//	    )
//	    if err != nil {
//	        panic(err)
//	    }
//
//	    messages := []types.Message{
//	        {Role: "user", Content: "Hello!"},
//	    }
//
//	    stream, err := provider.Complete(context.Background(), messages)
//	    if err != nil {
//	        panic(err)
//	    }
//
//	    for chunk := range stream {
//	        if chunk.Type == types.ChunkTypeMessageContent {
//	            fmt.Print(chunk.Content)
//	        }
//	    }
//	}
package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/llm/parser"
	"github.com/entrhq/forge/pkg/types"
	"github.com/openai/openai-go"
)

const (
	// DefaultBaseURL is the default OpenAI API base URL
	DefaultBaseURL = "https://api.openai.com/v1"
)

// Provider implements the LLM provider interface for OpenAI-compatible APIs.
type Provider struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	model      string
	modelInfo  *types.ModelInfo
}

// ProviderOption is a function that configures a Provider.
type ProviderOption func(*Provider)

// WithModel sets the model to use for completions.
func WithModel(model string) ProviderOption {
	return func(p *Provider) {
		p.model = model
	}
}

// WithBaseURL sets a custom base URL for OpenAI-compatible APIs.
// This enables using Azure OpenAI, local models, or other compatible services.
func WithBaseURL(baseURL string) ProviderOption {
	return func(p *Provider) {
		p.baseURL = baseURL
	}
}

// NewProvider creates a new OpenAI provider with the given API key.
//
// If apiKey is empty, it will attempt to read from the OPENAI_API_KEY environment variable.
// If baseURL is not provided via WithBaseURL option, it will check OPENAI_BASE_URL environment variable.
//
// The default model is "gpt-4".
//
// Example:
//
//	// Standard OpenAI
//	provider, _ := openai.NewProvider("sk-...", openai.WithModel("gpt-4"))
//
//	// Azure OpenAI
//	provider, _ := openai.NewProvider("your-key",
//	    openai.WithBaseURL("https://your-resource.openai.azure.com"),
//	    openai.WithModel("gpt-4o"))
//
//	// Local OpenAI-compatible API
//	provider, _ := openai.NewProvider("local",
//	    openai.WithBaseURL("http://localhost:8080/v1"))
func NewProvider(apiKey string, opts ...ProviderOption) (*Provider, error) {
	// Use environment variable if no API key provided
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required (provide via parameter or OPENAI_API_KEY environment variable)")
	}

	// Create provider with defaults
	p := &Provider{
		model:      "gpt-4o", // Default model
		apiKey:     apiKey,
		httpClient: &http.Client{},
		baseURL:    DefaultBaseURL,
	}

	// Apply options (may override baseURL via WithBaseURL)
	for _, opt := range opts {
		opt(p)
	}

	// If baseURL wasn't set by options, check environment variable
	if p.baseURL == DefaultBaseURL {
		if envBaseURL := os.Getenv("OPENAI_BASE_URL"); envBaseURL != "" {
			p.baseURL = envBaseURL
		}
	}

	// Initialize model info (if not already set by options)
	if p.modelInfo == nil {
		p.modelInfo = &types.ModelInfo{
			Metadata: make(map[string]interface{}),
		}
	}

	p.modelInfo.Provider = "openai"
	p.modelInfo.Name = p.model
	p.modelInfo.SupportsStreaming = true
	p.modelInfo.MaxTokens = 8192 // Default, varies by model

	// Store base URL in metadata if not default
	if p.baseURL != DefaultBaseURL {
		p.modelInfo.Metadata["base_url"] = p.baseURL
	}

	return p, nil
}

// CloneWithModel returns a shallow copy of p configured to use the given model.
// The clone shares the same HTTP client, API key, and base URL as the original,
// making it very cheap to create. It implements llm.ModelCloner.
func (p *Provider) CloneWithModel(model string) llm.Provider {
	clone := *p // shallow copy — shares httpClient (connection pool), apiKey, baseURL
	clone.model = model
	if p.modelInfo != nil {
		mi := *p.modelInfo // copy modelInfo so Name mutation doesn't affect original
		mi.Name = model
		clone.modelInfo = &mi
	}
	return &clone
}

// StreamCompletion sends messages to the OpenAI API and streams back response chunks.
//
// The returned channel emits StreamChunk instances as the response is generated.
// The channel is closed when streaming completes or an error occurs.
//
// This implementation uses raw HTTP streaming to handle SSE events directly,
// which provides better compatibility with OpenAI-compatible APIs that may
// include SSE comments or have slight format variations.
func (p *Provider) StreamCompletion(ctx context.Context, messages []*types.Message) (<-chan *llm.StreamChunk, error) {
	resp, err := p.sendStreamRequest(ctx, messages)
	if err != nil {
		return nil, err
	}

	chunks := make(chan *llm.StreamChunk, 10)
	go p.processStreamResponse(ctx, resp, chunks)
	return chunks, nil
}

// sendStreamRequest creates and sends the HTTP request for streaming
func (p *Provider) sendStreamRequest(ctx context.Context, messages []*types.Message) (*http.Response, error) {
	openaiMessages := convertToOpenAIMessages(messages)

	reqBody := map[string]interface{}{
		"model":    p.model,
		"messages": openaiMessages,
		"stream":   true,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := p.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("API request failed with status %d (failed to read error body: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// processStreamResponse processes the SSE stream and sends chunks to the channel
func (p *Provider) processStreamResponse(ctx context.Context, resp *http.Response, chunks chan<- *llm.StreamChunk) {
	defer close(chunks)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	firstChunk := true
	thinkingParser := parser.NewThinkingParser()

	for scanner.Scan() {
		line := scanner.Text()

		if !p.isValidSSELine(line) {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			p.handleStreamEnd(ctx, thinkingParser, chunks)
			return
		}

		if !p.processSSEChunk(ctx, data, &firstChunk, thinkingParser, chunks) {
			return
		}
	}

	p.flushRemainingContent(ctx, thinkingParser, chunks)

	if err := scanner.Err(); err != nil {
		chunks <- &llm.StreamChunk{Error: fmt.Errorf("stream read error: %w", err)}
	}
}

// isValidSSELine checks if a line is a valid SSE data line
func (p *Provider) isValidSSELine(line string) bool {
	return line != "" && !strings.HasPrefix(line, ":") && strings.HasPrefix(line, "data: ")
}

// handleStreamEnd handles the [DONE] marker and flushes remaining content
func (p *Provider) handleStreamEnd(ctx context.Context, thinkingParser *parser.ThinkingParser, chunks chan<- *llm.StreamChunk) {
	p.flushRemainingContent(ctx, thinkingParser, chunks)
	chunks <- &llm.StreamChunk{Finished: true}
}

// flushRemainingContent flushes any buffered content from the thinking parser
func (p *Provider) flushRemainingContent(ctx context.Context, thinkingParser *parser.ThinkingParser, chunks chan<- *llm.StreamChunk) {
	thinking, message := thinkingParser.Flush()
	p.sendChunkIfPresent(ctx, thinking, chunks)
	p.sendChunkIfPresent(ctx, message, chunks)
}

// sendChunkIfPresent sends a chunk to the channel if it's not nil
func (p *Provider) sendChunkIfPresent(ctx context.Context, chunk *llm.StreamChunk, chunks chan<- *llm.StreamChunk) bool {
	if chunk == nil {
		return true
	}
	select {
	case chunks <- chunk:
		return true
	case <-ctx.Done():
		chunks <- &llm.StreamChunk{Error: ctx.Err()}
		return false
	}
}

// processSSEChunk processes a single SSE data chunk
func (p *Provider) processSSEChunk(ctx context.Context, data string, firstChunk *bool, thinkingParser *parser.ThinkingParser, chunks chan<- *llm.StreamChunk) bool {
	var chunk struct {
		Choices []struct {
			Delta struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"delta"`
			FinishReason *string `json:"finish_reason"`
		} `json:"choices"`
	}

	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return true // Skip malformed chunks silently
	}

	if len(chunk.Choices) == 0 {
		return true
	}

	delta := chunk.Choices[0].Delta
	streamChunk := &llm.StreamChunk{}

	if *firstChunk && delta.Role != "" {
		streamChunk.Role = delta.Role
		*firstChunk = false
	}

	if delta.Content != "" {
		if !p.processContent(ctx, delta.Content, streamChunk.Role, thinkingParser, chunks) {
			return false
		}
	}

	return p.handleFinishReason(ctx, chunk.Choices[0].FinishReason, streamChunk, chunks)
}

// processContent parses and sends content chunks
func (p *Provider) processContent(ctx context.Context, content, role string, thinkingParser *parser.ThinkingParser, chunks chan<- *llm.StreamChunk) bool {
	thinkingChunk, messageChunk := thinkingParser.Parse(content)

	if thinkingChunk != nil {
		thinkingChunk.Role = role
		if !p.sendChunkIfPresent(ctx, thinkingChunk, chunks) {
			return false
		}
	}

	if messageChunk != nil {
		messageChunk.Role = role
		if !p.sendChunkIfPresent(ctx, messageChunk, chunks) {
			return false
		}
	}

	return true
}

// handleFinishReason handles the finish_reason field
func (p *Provider) handleFinishReason(ctx context.Context, finishReason *string, streamChunk *llm.StreamChunk, chunks chan<- *llm.StreamChunk) bool {
	if finishReason != nil && *finishReason == "stop" {
		streamChunk.Finished = true
		return p.sendChunkIfPresent(ctx, streamChunk, chunks)
	}

	if streamChunk.Role != "" {
		return p.sendChunkIfPresent(ctx, streamChunk, chunks)
	}

	return true
}

// Complete sends messages to the OpenAI API and returns the full response.
//
// This is a convenience wrapper around StreamCompletion that accumulates
// all chunks into a single message.
func (p *Provider) Complete(ctx context.Context, messages []*types.Message) (*types.Message, error) {
	stream, err := p.StreamCompletion(ctx, messages)
	if err != nil {
		return nil, err
	}

	var content string
	var role string

	for chunk := range stream {
		if chunk.IsError() {
			return nil, chunk.Error
		}

		if chunk.Role != "" {
			role = chunk.Role
		}

		content += chunk.Content
	}

	// Default to assistant role if not set
	if role == "" {
		role = string(types.RoleAssistant)
	}

	return &types.Message{
		Role:    types.MessageRole(role),
		Content: content,
	}, nil
}

// GetModelInfo returns information about the OpenAI model being used.
func (p *Provider) GetModelInfo() *types.ModelInfo {
	return p.modelInfo
}

// GetModel returns the model name being used.
func (p *Provider) GetModel() string {
	return p.model
}

// GetBaseURL returns the base URL being used.
func (p *Provider) GetBaseURL() string {
	return p.baseURL
}

// GetAPIKey returns the API key being used.
func (p *Provider) GetAPIKey() string {
	return p.apiKey
}

// convertToOpenAIMessages converts our Message format to OpenAI's ChatCompletionMessageParamUnion format.
func convertToOpenAIMessages(messages []*types.Message) []openai.ChatCompletionMessageParamUnion {
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))

	for _, msg := range messages {
		switch msg.Role {
		case types.RoleSystem:
			openaiMessages = append(openaiMessages, openai.SystemMessage(msg.Content))
		case types.RoleUser:
			openaiMessages = append(openaiMessages, openai.UserMessage(msg.Content))
		case types.RoleAssistant:
			openaiMessages = append(openaiMessages, openai.AssistantMessage(msg.Content))
		default:
			// Default to user message for unknown roles
			openaiMessages = append(openaiMessages, openai.UserMessage(msg.Content))
		}
	}

	return openaiMessages
}
