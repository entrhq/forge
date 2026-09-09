// Package anthropic provides an LLM provider for the native Anthropic Messages API.
//
// It is built on the official Anthropic Go SDK so request, response, and
// streaming handling follow the vendor's wire format rather than a hand-rolled
// client. A custom base URL targets any endpoint that implements the same API.
package anthropic

import (
	"context"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"os"
	"strings"

	sdk "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/entrhq/forge/pkg/llm"
	"github.com/entrhq/forge/pkg/llm/parser"
	"github.com/entrhq/forge/pkg/logging"
	"github.com/entrhq/forge/pkg/types"
)

const (
	// DefaultBaseURL is the public Anthropic API base URL.
	DefaultBaseURL = "https://api.anthropic.com"

	defaultModel = "claude-sonnet-4-5"

	// The Messages API requires max_tokens on every request. Current models
	// allow up to 128K output tokens; hitting the ceiling truncates the
	// response mid-thought, so the defaults are generous. Streaming has no
	// HTTP timeout concern, non-streaming does, hence the lower document limit.
	defaultStreamMaxTokens   = 64000
	defaultDocumentMaxTokens = 16000
)

// debugLog records per-request token usage, including cache reads and writes,
// so cache effectiveness is visible in the session log.
var debugLog *logging.Logger

func init() {
	var err error
	debugLog, err = logging.NewLogger("anthropic")
	if err != nil {
		debugLog.Warnf("Failed to initialize anthropic logger, using stderr fallback: %v", err)
	}
}

// Provider implements llm.Provider against the Anthropic Messages API.
type Provider struct {
	client    sdk.Client
	apiKey    string
	baseURL   string
	model     string
	maxTokens int
	modelInfo *types.ModelInfo
}

// ProviderOption configures a Provider.
type ProviderOption func(*Provider)

// WithModel sets the model used for completions.
func WithModel(model string) ProviderOption {
	return func(p *Provider) {
		p.model = model
	}
}

// WithMaxTokens sets the response token ceiling for every request. Use it for
// models with a lower output cap than the default, or to bound cost. Zero or
// negative values are ignored and the defaults apply.
func WithMaxTokens(maxTokens int) ProviderOption {
	return func(p *Provider) {
		if maxTokens > 0 {
			p.maxTokens = maxTokens
		}
	}
}

// WithBaseURL points the provider at a custom Anthropic-compatible endpoint.
func WithBaseURL(baseURL string) ProviderOption {
	return func(p *Provider) {
		p.baseURL = baseURL
	}
}

// NewProvider creates a Provider with the given API key.
//
// An empty apiKey falls back to ANTHROPIC_API_KEY. If WithBaseURL is not
// given, ANTHROPIC_BASE_URL is used when set, otherwise DefaultBaseURL.
func NewProvider(apiKey string, opts ...ProviderOption) (*Provider, error) {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic API key is required (provide via parameter or ANTHROPIC_API_KEY environment variable)")
	}

	p := &Provider{
		apiKey:  apiKey,
		model:   defaultModel,
		baseURL: DefaultBaseURL,
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.baseURL == DefaultBaseURL {
		if envBaseURL := os.Getenv("ANTHROPIC_BASE_URL"); envBaseURL != "" {
			p.baseURL = envBaseURL
		}
	}

	// Environment defaults are disabled so the resolved values above are the
	// only source of truth, matching how the OpenAI provider is configured.
	p.client = sdk.NewClient(
		option.WithoutEnvironmentDefaults(),
		option.WithAPIKey(p.apiKey),
		option.WithBaseURL(p.baseURL),
	)

	p.modelInfo = &types.ModelInfo{
		Metadata:          make(map[string]any),
		Name:              p.model,
		Provider:          "anthropic",
		MaxTokens:         int(p.streamMaxTokens()),
		SupportsStreaming: true,
	}
	if p.baseURL != DefaultBaseURL {
		p.modelInfo.Metadata["base_url"] = p.baseURL
	}

	return p, nil
}

// CloneWithModel returns a shallow copy of p targeting the given model.
// The clone shares the underlying HTTP client and credentials. It implements llm.ModelCloner.
func (p *Provider) CloneWithModel(model string) llm.Provider {
	clone := *p
	clone.model = model
	mi := *p.modelInfo
	mi.Name = model
	clone.modelInfo = &mi
	return &clone
}

// StreamCompletion sends messages to the Messages API and streams back response chunks.
func (p *Provider) StreamCompletion(ctx context.Context, messages []*types.Message) (<-chan *llm.StreamChunk, error) {
	params := p.buildParams(messages)
	debugLog.Debugf("request %s", fingerprint(params))
	stream := p.client.Messages.NewStreaming(ctx, params)

	chunks := make(chan *llm.StreamChunk, 10)
	go p.processStream(ctx, stream, chunks)
	return chunks, nil
}

// processStream drains the SDK event stream into chunks.
//
// Text deltas pass through the shared thinking parser so <thinking> tags are
// split out exactly as they are for the OpenAI provider. The accumulated
// message supplies token usage on the final chunk.
func (p *Provider) processStream(ctx context.Context, stream *ssestream.Stream[sdk.MessageStreamEventUnion], chunks chan<- *llm.StreamChunk) {
	defer close(chunks)
	defer stream.Close()

	thinkingParser := parser.NewThinkingParser()
	var accumulated sdk.Message
	roleSent := false

	for stream.Next() {
		event := stream.Current()
		if err := accumulated.Accumulate(event); err != nil {
			chunks <- &llm.StreamChunk{Error: fmt.Errorf("failed to accumulate stream event: %w", err)}
			return
		}

		delta, ok := event.AsAny().(sdk.ContentBlockDeltaEvent)
		if !ok || delta.Delta.Type != "text_delta" || delta.Delta.Text == "" {
			continue
		}

		role := ""
		if !roleSent {
			role = string(types.RoleAssistant)
			roleSent = true
		}
		thinking, message := thinkingParser.Parse(delta.Delta.Text)
		if !p.sendParsed(ctx, thinking, message, role, chunks) {
			return
		}
	}

	if err := stream.Err(); err != nil {
		chunks <- &llm.StreamChunk{Error: fmt.Errorf("stream error: %w", err)}
		return
	}

	thinking, message := thinkingParser.Flush()
	if !p.sendParsed(ctx, thinking, message, "", chunks) {
		return
	}

	u := accumulated.Usage
	debugLog.Debugf("model=%s input=%d output=%d cache_read=%d cache_write=%d",
		p.model, u.InputTokens, u.OutputTokens, u.CacheReadInputTokens, u.CacheCreationInputTokens)
	p.send(ctx, &llm.StreamChunk{Finished: true, Usage: usageInfo(u)}, chunks)
}

// sendParsed sends the thinking and message chunks produced by the parser.
// The role is attached to whichever chunk goes out first.
func (p *Provider) sendParsed(ctx context.Context, thinking, message *llm.StreamChunk, role string, chunks chan<- *llm.StreamChunk) bool {
	for _, chunk := range []*llm.StreamChunk{thinking, message} {
		if chunk == nil {
			continue
		}
		chunk.Role = role
		role = ""
		if !p.send(ctx, chunk, chunks) {
			return false
		}
	}
	return true
}

// send delivers a chunk, reporting cancellation as an error chunk.
func (p *Provider) send(ctx context.Context, chunk *llm.StreamChunk, chunks chan<- *llm.StreamChunk) bool {
	select {
	case chunks <- chunk:
		return true
	case <-ctx.Done():
		chunks <- &llm.StreamChunk{Error: ctx.Err()}
		return false
	}
}

// Complete sends messages and returns the full response by draining StreamCompletion.
func (p *Provider) Complete(ctx context.Context, messages []*types.Message) (*types.Message, error) {
	stream, err := p.StreamCompletion(ctx, messages)
	if err != nil {
		return nil, err
	}

	var content strings.Builder
	for chunk := range stream {
		if chunk.IsError() {
			return nil, chunk.Error
		}
		content.WriteString(chunk.Content)
	}

	return &types.Message{
		Role:    types.RoleAssistant,
		Content: content.String(),
	}, nil
}

// AnalyzeDocument sends an image or PDF inline with the prompt and returns the model's text.
//
// Supported media types are image/* and application/pdf.
func (p *Provider) AnalyzeDocument(ctx context.Context, fileData []byte, mediaType string, prompt string) (string, error) {
	if prompt == "" {
		prompt = "Analyze this document and provide a detailed description of its contents."
	}

	encoded := base64.StdEncoding.EncodeToString(fileData)
	var document sdk.ContentBlockParamUnion
	switch {
	case strings.HasPrefix(mediaType, "image/"):
		document = sdk.NewImageBlockBase64(mediaType, encoded)
	case mediaType == "application/pdf":
		document = sdk.NewDocumentBlock(sdk.Base64PDFSourceParam{Data: encoded})
	default:
		return "", fmt.Errorf("unsupported media type %q (expected image/* or application/pdf)", mediaType)
	}

	result, err := p.client.Messages.New(ctx, sdk.MessageNewParams{
		Model:     p.model,
		MaxTokens: p.documentMaxTokens(),
		Messages: []sdk.MessageParam{
			sdk.NewUserMessage(sdk.NewTextBlock(prompt), document),
		},
	})
	if err != nil {
		return "", fmt.Errorf("document analysis request failed: %w", err)
	}

	var text strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	if text.Len() == 0 {
		return "", fmt.Errorf("no response from API")
	}
	return text.String(), nil
}

// GetModelInfo returns information about the configured model.
func (p *Provider) GetModelInfo() *types.ModelInfo {
	return p.modelInfo
}

// GetModel returns the model name.
func (p *Provider) GetModel() string {
	return p.model
}

// GetBaseURL returns the API base URL.
func (p *Provider) GetBaseURL() string {
	return p.baseURL
}

// GetAPIKey returns the API key.
func (p *Provider) GetAPIKey() string {
	return p.apiKey
}

// buildParams converts Forge messages into a Messages API request.
//
// System-role messages become the top-level system field, one text block
// each. Every other role becomes a user or assistant turn with a single text
// block. Forge's tool protocol is XML inside message text and RoleTool is
// normalised to RoleUser before reaching a provider, so the SDK's native
// tool_use/tool_result blocks are intentionally not used.
func (p *Provider) buildParams(messages []*types.Message) sdk.MessageNewParams {
	params := sdk.MessageNewParams{
		Model:     p.model,
		MaxTokens: p.streamMaxTokens(),
	}
	bp := newBreakpoints()

	for _, msg := range messages {
		// The API rejects empty text blocks, so blank messages are dropped.
		if msg.Content == "" {
			continue
		}
		switch msg.Role {
		case types.RoleSystem:
			params.System = append(params.System, sdk.TextBlockParam{Text: msg.Content})
			bp.noteSystem(len(params.System)-1, msg.Stability)
		case types.RoleAssistant:
			params.Messages = append(params.Messages, sdk.NewAssistantMessage(sdk.NewTextBlock(msg.Content)))
			bp.noteMessage(len(params.Messages)-1, msg.Stability)
		default:
			params.Messages = append(params.Messages, sdk.NewUserMessage(sdk.NewTextBlock(msg.Content)))
			bp.noteMessage(len(params.Messages)-1, msg.Stability)
		}
	}

	bp.apply(&params)
	return params
}

// breakpoints tracks where prompt cache breakpoints go, using the stability
// tags set by the prompt builder. Anthropic allows four per request; the
// candidates, in prompt order, are:
//
//  1. the last static system block, so the fixed prompt text caches on its own;
//  2. the last system block, so the tool listing caches separately from history;
//  3. the last session-stable conversation message, the most recent compaction
//     summary, so compacted history caches as a unit;
//  4. the last message, so the whole request is the prefix of the next one.
//
// An index of -1 means that candidate is absent from this request.
type breakpoints struct {
	lastStaticSystem   int
	lastSystem         int
	lastSessionMessage int
	lastMessage        int
}

func newBreakpoints() breakpoints {
	return breakpoints{lastStaticSystem: -1, lastSystem: -1, lastSessionMessage: -1, lastMessage: -1}
}

func (b *breakpoints) noteSystem(index int, stability types.Stability) {
	b.lastSystem = index
	if stability == types.StabilityStatic {
		b.lastStaticSystem = index
	}
}

func (b *breakpoints) noteMessage(index int, stability types.Stability) {
	b.lastMessage = index
	if stability == types.StabilitySession {
		b.lastSessionMessage = index
	}
}

// apply marks the chosen blocks with an ephemeral (5 minute) cache control.
// Candidates that coincide share one marker.
func (b *breakpoints) apply(params *sdk.MessageNewParams) {
	cache := sdk.NewCacheControlEphemeralParam()
	for _, i := range []int{b.lastStaticSystem, b.lastSystem} {
		if i >= 0 && i < len(params.System) {
			params.System[i].CacheControl = cache
		}
	}
	for _, i := range []int{b.lastSessionMessage, b.lastMessage} {
		if i >= 0 && i < len(params.Messages) {
			params.Messages[i].Content[0].OfText.CacheControl = cache
		}
	}
}

// streamMaxTokens is the ceiling for streamed completions: the configured
// value, or the streaming default.
func (p *Provider) streamMaxTokens() int64 {
	if p.maxTokens > 0 {
		return int64(p.maxTokens)
	}
	return defaultStreamMaxTokens
}

// documentMaxTokens is the ceiling for non-streamed document analysis. A
// configured value applies here too, since it may reflect a model's hard cap.
func (p *Provider) documentMaxTokens() int64 {
	if p.maxTokens > 0 {
		return min(int64(p.maxTokens), defaultDocumentMaxTokens)
	}
	return defaultDocumentMaxTokens
}

// fingerprint summarizes a request's cacheable structure without its content:
// a short hash and length per system block, the message count, and a hash of
// every message but the last. Comparing consecutive lines shows which part of
// the prefix changed when cache reads drop.
func fingerprint(params sdk.MessageNewParams) string {
	var b strings.Builder
	b.WriteString("system=[")
	for i, block := range params.System {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%s/%d", shortHash(block.Text), len(block.Text))
	}
	b.WriteString("]")

	prefix := fnv.New64a()
	for _, msg := range params.Messages[:max(len(params.Messages)-1, 0)] {
		prefix.Write([]byte(msg.Role))
		prefix.Write([]byte(msg.Content[0].OfText.Text))
	}
	fmt.Fprintf(&b, " messages=%d history_prefix=%08x", len(params.Messages), prefix.Sum64()&0xffffffff)
	return b.String()
}

func shortHash(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return fmt.Sprintf("%08x", h.Sum64()&0xffffffff)
}

// usageInfo maps SDK usage onto the provider-neutral UsageInfo.
func usageInfo(u sdk.Usage) *llm.UsageInfo {
	return &llm.UsageInfo{
		PromptTokens:     int(u.InputTokens),
		CompletionTokens: int(u.OutputTokens),
		TotalTokens:      int(u.InputTokens + u.OutputTokens),
	}
}
