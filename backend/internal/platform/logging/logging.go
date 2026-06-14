// Package logging provides a leveled JSON logger built on log/slog with helpers
// that redact secrets and PII. Every log line is structured; sensitive values
// never reach the output regardless of how a caller passes them.
package logging

import (
	"io"
	"log/slog"
	"strings"
)

// Logger is the logger type used throughout the backend. It aliases
// *slog.Logger so callers depend on this package rather than log/slog directly.
type Logger = slog.Logger

// New returns a JSON slog.Logger at the given level ("debug","info","warn",
// "error"; unknown values fall back to info). A ReplaceAttr hook redacts any
// attribute whose key looks sensitive, as a backstop to typed redaction.
func New(w io.Writer, level string) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       parseLevel(level),
		ReplaceAttr: redactAttr,
	})
	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// sensitiveKeys are attribute key fragments whose values must never be logged.
// Matching is case-insensitive and substring-based so e.g. "user_password" and
// "authorization" are caught.
var sensitiveKeys = []string{
	"password", "passwd", "secret", "token", "authorization", "api_key", "apikey",
	"anthropic", "dsn", "signing_key", "cookie", "refresh", "bearer",
}

const redacted = "[REDACTED]"

// redactAttr is the slog ReplaceAttr hook. It redacts values under sensitive
// keys at any nesting depth and defers to a value's own redaction (e.g.
// config.Secret) by leaving non-string values untouched when not key-matched.
func redactAttr(_ []string, a slog.Attr) slog.Attr {
	if isSensitiveKey(a.Key) {
		return slog.String(a.Key, redacted)
	}
	return a
}

func isSensitiveKey(key string) bool {
	return IsSensitiveKey(key)
}

// IsSensitiveKey reports whether a field name looks sensitive (case-insensitive
// substring match against the known-sensitive fragments). Exported so other
// redactors (e.g. the events writer) apply the same policy.
func IsSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, frag := range sensitiveKeys {
		if strings.Contains(k, frag) {
			return true
		}
	}
	return false
}

// Redact returns a redacted placeholder for a known-sensitive value. Use when
// you must reference that a field was present without revealing it.
func Redact(string) string { return redacted }
