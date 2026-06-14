// Package config loads backend configuration from the environment. The Claude
// API key and other secrets live here at runtime ONLY — never in the client,
// the repo, or any log line (see Redacted). Required values fail fast on boot.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds all runtime settings. Secret fields are wrapped in Secret so they
// cannot be accidentally printed; call Reveal() to access the value.
type Config struct {
	HTTPAddr        string
	LogLevel        string
	MySQLDSN        Secret
	AnthropicAPIKey Secret
	ClaudeModel     string
	JWTSigningKey   Secret
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// Secret wraps a sensitive string so it is redacted by fmt, JSON, and logging.
// The underlying value is reachable only via Reveal().
type Secret struct {
	value string
}

// NewSecret wraps v. Exported for tests and explicit construction.
func NewSecret(v string) Secret { return Secret{value: v} }

// Reveal returns the underlying secret value. Call this only where the value is
// genuinely needed (e.g. handing the DSN to the driver), never for logging.
func (s Secret) Reveal() string { return s.value }

// IsZero reports whether the secret is empty.
func (s Secret) IsZero() bool { return s.value == "" }

const redacted = "[REDACTED]"

// String implements fmt.Stringer so %s/%v never leak the value.
func (s Secret) String() string {
	if s.value == "" {
		return ""
	}
	return redacted
}

// GoString implements fmt.GoStringer so %#v never leaks the value.
func (s Secret) GoString() string { return s.String() }

// MarshalJSON ensures structured logs / JSON encoders never serialize the value.
func (s Secret) MarshalJSON() ([]byte, error) {
	if s.value == "" {
		return []byte(`""`), nil
	}
	return []byte(`"` + redacted + `"`), nil
}

// Load reads configuration from the process environment, applying defaults and
// validating that required secrets are present. It returns an error listing
// every missing/invalid value so the operator fixes them in one pass.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getEnvDefault("HTTP_ADDR", ":8080"),
		LogLevel:        getEnvDefault("LOG_LEVEL", "info"),
		MySQLDSN:        NewSecret(os.Getenv("MYSQL_DSN")),
		AnthropicAPIKey: NewSecret(os.Getenv("ANTHROPIC_API_KEY")),
		ClaudeModel:     getEnvDefault("CLAUDE_MODEL", "claude-opus-4-8"),
		JWTSigningKey:   NewSecret(os.Getenv("JWT_SIGNING_KEY")),
	}

	var problems []string
	require := func(name string, s Secret) {
		if s.IsZero() {
			problems = append(problems, name+" is required")
		}
	}
	require("MYSQL_DSN", cfg.MySQLDSN)
	require("ANTHROPIC_API_KEY", cfg.AnthropicAPIKey)
	require("JWT_SIGNING_KEY", cfg.JWTSigningKey)

	// The signing key must be long enough to be meaningful for HS256.
	if !cfg.JWTSigningKey.IsZero() && len(cfg.JWTSigningKey.Reveal()) < 32 {
		problems = append(problems, "JWT_SIGNING_KEY must be at least 32 bytes")
	}

	var err error
	if cfg.AccessTokenTTL, err = getEnvDuration("ACCESS_TOKEN_TTL", 15*time.Minute); err != nil {
		problems = append(problems, err.Error())
	}
	if cfg.RefreshTokenTTL, err = getEnvDuration("REFRESH_TOKEN_TTL", 720*time.Hour); err != nil {
		problems = append(problems, err.Error())
	}

	if len(problems) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %s", strings.Join(problems, "; "))
	}
	return cfg, nil
}

func getEnvDefault(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func getEnvDuration(name string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(name)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s is not a valid duration: %w", name, err)
	}
	return d, nil
}
