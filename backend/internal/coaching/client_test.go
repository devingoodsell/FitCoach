package coaching

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"

	"pro.d11l.fitcoach/backend/internal/platform/config"
)

// captureTransport is a fake option.HTTPClient that records the outbound request
// and returns a canned Messages response, so we exercise real SDK request
// building without touching the network.
type captureTransport struct {
	gotAPIKey string
	gotBody   map[string]any
	respText  string
}

func (c *captureTransport) Do(r *http.Request) (*http.Response, error) {
	c.gotAPIKey = r.Header.Get("x-api-key")
	if r.Body != nil {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &c.gotBody)
	}
	body := `{
      "id": "msg_test",
      "type": "message",
      "role": "assistant",
      "model": "claude-opus-4-8",
      "content": [{"type": "text", "text": ` + jsonString(c.respText) + `}],
      "stop_reason": "end_turn",
      "usage": {"input_tokens": 11, "output_tokens": 22,
                "cache_creation_input_tokens": 0, "cache_read_input_tokens": 0}
    }`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func testConfig() config.Config {
	return config.Config{
		AnthropicAPIKey: config.NewSecret("sk-test-from-config"),
		ClaudeModel:     "claude-opus-4-8",
	}
}

func TestAnthropicGeneratorBuildsRequestAndSourcesKeyFromConfig(t *testing.T) {
	tr := &captureTransport{respText: `{"hello":"world"}`}
	gen := NewGenerator(testConfig(), option.WithHTTPClient(tr), option.WithMaxRetries(0))

	res, err := gen.Generate(context.Background(), GenerationRequest{
		System: "You are FitCoach. Reply with JSON only.",
		Prompt: []byte(`{"requested_at":"2026-06-16T13:00:00Z"}`),
		Schema: map[string]any{"type": "object"},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Key comes ONLY from config, not the environment.
	if tr.gotAPIKey != "sk-test-from-config" {
		t.Errorf("x-api-key = %q, want the config secret", tr.gotAPIKey)
	}
	// Request is built correctly: configured model, our system prompt, the prompt
	// as the user turn, and the structured-output schema.
	if tr.gotBody["model"] != "claude-opus-4-8" {
		t.Errorf("model = %v, want claude-opus-4-8", tr.gotBody["model"])
	}
	if sys, _ := tr.gotBody["system"].([]any); len(sys) == 0 {
		t.Errorf("expected a system prompt in the request, got %v", tr.gotBody["system"])
	}
	if _, ok := tr.gotBody["output_config"]; !ok {
		t.Errorf("expected output_config (structured outputs) in the request")
	}
	if _, ok := tr.gotBody["messages"]; !ok {
		t.Errorf("expected messages in the request")
	}

	// Response text is returned verbatim with usage.
	if string(res.SessionJSON) != `{"hello":"world"}` {
		t.Errorf("SessionJSON = %s, want the model text", res.SessionJSON)
	}
	if res.InputTokens != 11 || res.OutputTokens != 22 {
		t.Errorf("usage = %d/%d, want 11/22", res.InputTokens, res.OutputTokens)
	}
}

func TestAnthropicGeneratorDefaultsModelWhenUnset(t *testing.T) {
	tr := &captureTransport{respText: `{}`}
	cfg := config.Config{AnthropicAPIKey: config.NewSecret("sk-test")} // no ClaudeModel
	gen := NewGenerator(cfg, option.WithHTTPClient(tr), option.WithMaxRetries(0))

	if _, err := gen.Generate(context.Background(), GenerationRequest{Prompt: []byte("{}")}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if tr.gotBody["model"] != "claude-opus-4-8" {
		t.Errorf("model = %v, want default claude-opus-4-8", tr.gotBody["model"])
	}
}

func TestAnthropicGeneratorEmptyResponseIsError(t *testing.T) {
	tr := &captureTransport{respText: "   "} // whitespace only
	gen := NewGenerator(testConfig(), option.WithHTTPClient(tr), option.WithMaxRetries(0))

	if _, err := gen.Generate(context.Background(), GenerationRequest{Prompt: []byte("{}")}); err == nil {
		t.Fatalf("expected an error on empty model response")
	}
}
