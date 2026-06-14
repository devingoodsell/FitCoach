package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// setEnv sets the full set of env vars for a valid config, then applies
// overrides. t.Setenv restores the environment after the test.
func setValidEnv(t *testing.T, overrides map[string]string) {
	t.Helper()
	base := map[string]string{
		"MYSQL_DSN":         "user:pass@tcp(127.0.0.1:3306)/fitcoach",
		"ANTHROPIC_API_KEY": "sk-ant-secret-value",
		"JWT_SIGNING_KEY":   "0123456789abcdef0123456789abcdef",
	}
	for k, v := range overrides {
		base[k] = v
	}
	// Clear optional vars that could leak from the host environment.
	for _, k := range []string{"HTTP_ADDR", "LOG_LEVEL", "CLAUDE_MODEL", "ACCESS_TOKEN_TTL", "REFRESH_TOKEN_TTL"} {
		if _, ok := base[k]; !ok {
			t.Setenv(k, "")
		}
	}
	for k, v := range base {
		t.Setenv(k, v)
	}
}

func TestLoadDefaults(t *testing.T) {
	setValidEnv(t, nil)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.AccessTokenTTL != 15*time.Minute {
		t.Errorf("AccessTokenTTL = %v, want 15m", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 720*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want 720h", cfg.RefreshTokenTTL)
	}
	if cfg.ClaudeModel != "claude-opus-4-8" {
		t.Errorf("ClaudeModel = %q", cfg.ClaudeModel)
	}
	if cfg.AnthropicAPIKey.Reveal() != "sk-ant-secret-value" {
		t.Errorf("api key not loaded")
	}
}

func TestLoadRequiredVars(t *testing.T) {
	tests := []struct {
		name    string
		unset   string
		wantSub string
	}{
		{"missing dsn", "MYSQL_DSN", "MYSQL_DSN is required"},
		{"missing api key", "ANTHROPIC_API_KEY", "ANTHROPIC_API_KEY is required"},
		{"missing signing key", "JWT_SIGNING_KEY", "JWT_SIGNING_KEY is required"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setValidEnv(t, map[string]string{tc.unset: ""})
			_, err := Load()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error = %q, want substring %q", err, tc.wantSub)
			}
		})
	}
}

func TestLoadRejectsShortSigningKey(t *testing.T) {
	setValidEnv(t, map[string]string{"JWT_SIGNING_KEY": "too-short"})
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "at least 32 bytes") {
		t.Fatalf("error = %v, want signing-key length complaint", err)
	}
}

func TestLoadRejectsBadDuration(t *testing.T) {
	setValidEnv(t, map[string]string{"ACCESS_TOKEN_TTL": "fifteen-minutes"})
	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "ACCESS_TOKEN_TTL") {
		t.Fatalf("error = %v, want duration complaint", err)
	}
}

// TestSecretNeverLeaks is the core redaction guarantee: a Secret must never
// render its value through any of the formatting paths a logger might use.
func TestSecretNeverLeaks(t *testing.T) {
	const raw = "sk-ant-super-secret"
	s := NewSecret(raw)

	checks := map[string]string{
		"%s":     fmt.Sprintf("%s", s),
		"%v":     fmt.Sprintf("%v", s),
		"%#v":    fmt.Sprintf("%#v", s),
		"%+v":    fmt.Sprintf("%+v", s),
		"Sprint": fmt.Sprint(s),
	}
	for fmtName, out := range checks {
		if strings.Contains(out, raw) {
			t.Errorf("%s leaked secret: %q", fmtName, out)
		}
	}

	// JSON marshaling (how structured logs serialize) must also redact.
	b, err := json.Marshal(struct{ Key Secret }{s})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), raw) {
		t.Errorf("json leaked secret: %s", b)
	}

	// Marshaling a whole Config (as a logger might) must not leak either.
	setValidEnv(t, nil)
	cfg, _ := Load()
	cb, _ := json.Marshal(cfg)
	if strings.Contains(string(cb), "sk-ant-secret-value") {
		t.Errorf("config json leaked api key: %s", cb)
	}

	// Reveal still works for legitimate use.
	if s.Reveal() != raw {
		t.Errorf("Reveal = %q, want %q", s.Reveal(), raw)
	}
}
