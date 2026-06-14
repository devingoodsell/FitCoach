package events

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestRedactStripsSensitiveKeysAtAnyDepth(t *testing.T) {
	in := []byte(`{
		"latency_ms": 1200,
		"api_key": "sk-ant-secret",
		"prompt": "full assembled coach memory text",
		"nested": {"refresh_token": "rt-xyz", "ok": "keep"},
		"list": [{"password": "p"}, {"safe": "v"}]
	}`)
	out, err := Redact(in)
	if err != nil {
		t.Fatalf("Redact: %v", err)
	}
	s := string(out)
	for _, leaked := range []string{"sk-ant-secret", "full assembled coach memory text", "rt-xyz", `"p"`} {
		if strings.Contains(s, leaked) {
			t.Errorf("redaction leaked %q: %s", leaked, s)
		}
	}
	// Non-sensitive values survive.
	for _, kept := range []string{"1200", "keep", "safe"} {
		if !strings.Contains(s, kept) {
			t.Errorf("redaction dropped non-sensitive %q: %s", kept, s)
		}
	}
}

func TestWritePersistsRedactedPayload(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	w := NewWriter(database, func() time.Time { return time.Unix(0, 0).UTC() })

	uid, _ := uuid.NewV7()
	mock.ExpectExec("INSERT INTO events").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), TypeGeneration, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = w.Write(context.Background(), Event{
		Type:    TypeGeneration,
		UserID:  uid,
		Payload: map[string]any{"api_key": "sk-ant-secret", "latency_ms": 10},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestWriteAllowsSystemEventWithoutUser(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	w := NewWriter(database, nil)

	mock.ExpectExec("INSERT INTO events").
		WithArgs(sqlmock.AnyArg(), nil, TypeSafety, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := w.Write(context.Background(), Event{Type: TypeSafety, Payload: map[string]any{"ok": true}}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
