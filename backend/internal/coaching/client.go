package coaching

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"pro.d11l.fitcoach/backend/internal/platform/config"
)

// defaultMaxTokens caps generation output. A session is a small JSON document, so
// this is generous headroom; non-streaming is fine at this size.
const defaultMaxTokens = 8192

// defaultTimeout bounds the single server-side model call.
const defaultTimeout = 60 * time.Second

// Generator performs ONE server-side Claude call and returns the raw JSON session
// body. It is defined as an interface so the engine (E5-PR3) can be unit-tested
// with a fake — tests never hit the real Anthropic API. coaching is the only
// package that may call Claude (CLAUDE.md §2).
type Generator interface {
	Generate(ctx context.Context, req GenerationRequest) (GenerationResult, error)
}

// GenerationRequest is the assembled input to one generation call.
type GenerationRequest struct {
	System string         // system prompt: role, JSON-only contract, disclaimer language
	Prompt []byte         // assembled Coach Memory + current state (the user turn)
	Schema map[string]any // structured-output JSON schema; constrains output to a valid session
}

// GenerationResult is the model output plus token usage for audit/cost (E15-S3).
type GenerationResult struct {
	SessionJSON  []byte
	InputTokens  int
	OutputTokens int
}

// anthropicGenerator is the real, server-side Claude-backed Generator. It is the
// only place the Anthropic API key is used, and the key is sourced exclusively
// from config — never the ambient ANTHROPIC_API_KEY environment variable.
type anthropicGenerator struct {
	client    anthropic.Client
	model     anthropic.Model
	maxTokens int64
}

// NewGenerator builds a Claude-backed Generator from server-side config. The API
// key comes from cfg.AnthropicAPIKey (a redacted config.Secret), so it can never
// leak via logs or reach the client. Extra opts let tests inject a mock HTTP
// transport; production passes none. The SDK retries 429/5xx with backoff.
func NewGenerator(cfg config.Config, opts ...option.RequestOption) Generator {
	model := anthropic.Model(cfg.ClaudeModel)
	if model == "" {
		model = anthropic.ModelClaudeOpus4_8
	}
	base := []option.RequestOption{
		option.WithAPIKey(cfg.AnthropicAPIKey.Reveal()),
		option.WithRequestTimeout(defaultTimeout),
	}
	base = append(base, opts...)
	return &anthropicGenerator{
		client:    anthropic.NewClient(base...),
		model:     model,
		maxTokens: defaultMaxTokens,
	}
}

// Generate makes the single Messages call and returns the model's JSON text. When
// a Schema is supplied it is sent as a structured-output format so the model is
// constrained to emit a schema-valid session (E5-S1); parsing/validation of the
// body is the engine's job (E5-PR3).
func (g *anthropicGenerator) Generate(ctx context.Context, req GenerationRequest) (GenerationResult, error) {
	params := anthropic.MessageNewParams{
		Model:     g.model,
		MaxTokens: g.maxTokens,
		// Adaptive thinking is the recommended mode for Opus 4.8 (no budget_tokens).
		Thinking: anthropic.ThinkingConfigParamUnion{OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(string(req.Prompt))),
		},
	}
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{Text: req.System}}
	}
	if len(req.Schema) > 0 {
		params.OutputConfig = anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{Schema: req.Schema},
		}
	}

	resp, err := g.client.Messages.New(ctx, params)
	if err != nil {
		return GenerationResult{}, fmt.Errorf("claude generate: %w", err)
	}

	var b strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			b.WriteString(t.Text)
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return GenerationResult{}, fmt.Errorf("claude generate: empty response")
	}
	return GenerationResult{
		SessionJSON:  []byte(out),
		InputTokens:  int(resp.Usage.InputTokens),
		OutputTokens: int(resp.Usage.OutputTokens),
	}, nil
}
