// Package events is the persistence seam for generation and safety-validation
// audit events (E15-S3). The coaching (E5) and safety (E7-S4/E13) packages call
// Writer.Write to record outcomes; payloads are redacted before storage so no
// secrets or prompt contents are persisted.
package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// Event types recorded by the platform.
const (
	TypeGeneration = "generation"
	TypeSafety     = "safety"
)

// Event is an audit record. UserID may be uuid.Nil for system events. Payload is
// any JSON-serializable value; sensitive fields are redacted before storage.
type Event struct {
	Type    string
	UserID  uuid.UUID
	Payload any
}

// Writer persists events to MySQL.
type Writer struct {
	db  *sql.DB
	now func() time.Time
}

// NewWriter wires a Writer. now defaults to time.Now (UTC) when nil.
func NewWriter(database *sql.DB, now func() time.Time) *Writer {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Writer{db: database, now: now}
}

// Write redacts and persists an event. The payload is marshaled, scrubbed of
// sensitive keys at any depth, and stored.
func (w *Writer) Write(ctx context.Context, e Event) error {
	raw, err := json.Marshal(e.Payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}
	redacted, err := Redact(raw)
	if err != nil {
		return fmt.Errorf("redact event payload: %w", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate event id: %w", err)
	}
	var userID any
	if e.UserID != uuid.Nil {
		userID = e.UserID[:]
	}
	_, err = w.db.ExecContext(ctx,
		`INSERT INTO events (id, user_id, type, payload, created_at) VALUES (?, ?, ?, ?, ?)`,
		id[:], userID, e.Type, []byte(redacted), w.now())
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}
	return nil
}

const redactedValue = "[REDACTED]"

// Redact walks a JSON document and replaces the value of any sensitive key
// (including "prompt") with a placeholder, at any nesting depth. It returns
// canonical JSON.
func Redact(raw []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return json.Marshal(redactValue(v))
}

func redactValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if isSensitive(k) {
				out[k] = redactedValue
				continue
			}
			out[k] = redactValue(val)
		}
		return out
	case []any:
		for i := range t {
			t[i] = redactValue(t[i])
		}
		return t
	default:
		return v
	}
}

// isSensitive applies the shared logging policy plus event-specific keys (the
// assembled prompt text must never be persisted).
func isSensitive(key string) bool {
	if logging.IsSensitiveKey(key) {
		return true
	}
	switch key {
	case "prompt", "prompt_text", "messages", "completion":
		return true
	}
	return false
}
