package logging

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedactsSensitiveKeys(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "info")

	logger.Info("login",
		"email", "user@example.com",
		"password", "hunter2",
		"refresh_token", "rt-abc123",
		"authorization", "Bearer xyz",
		"anthropic_api_key", "sk-ant-leak",
	)

	out := buf.String()
	for _, leaked := range []string{"hunter2", "rt-abc123", "Bearer xyz", "sk-ant-leak"} {
		if strings.Contains(out, leaked) {
			t.Errorf("log leaked %q: %s", leaked, out)
		}
	}
	// Non-sensitive values pass through.
	if !strings.Contains(out, "user@example.com") {
		t.Errorf("expected email to pass through: %s", out)
	}
	if strings.Count(out, redacted) < 4 {
		t.Errorf("expected 4 redactions, got: %s", out)
	}
}

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "warn")

	logger.Debug("debug-msg")
	logger.Info("info-msg")
	logger.Warn("warn-msg")
	logger.Error("error-msg")

	out := buf.String()
	if strings.Contains(out, "debug-msg") || strings.Contains(out, "info-msg") {
		t.Errorf("below-threshold lines should be dropped: %s", out)
	}
	if !strings.Contains(out, "warn-msg") || !strings.Contains(out, "error-msg") {
		t.Errorf("warn/error should be logged: %s", out)
	}
}

func TestMiddlewareLogsAndSetsRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := New(&buf, "info")

	var seenID string
	h := Middleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenID = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusTeapot)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))

	if seenID == "" {
		t.Fatal("request id not present in context")
	}
	if got := rec.Header().Get(RequestIDHeader); got != seenID {
		t.Errorf("response header id = %q, want %q", got, seenID)
	}

	var line map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &line); err != nil {
		t.Fatalf("log not JSON: %v (%s)", err, buf.String())
	}
	if line["msg"] != "http_request" {
		t.Errorf("msg = %v", line["msg"])
	}
	if line["status"].(float64) != float64(http.StatusTeapot) {
		t.Errorf("status = %v, want 418", line["status"])
	}
	if line["request_id"] != seenID {
		t.Errorf("logged request_id = %v, want %q", line["request_id"], seenID)
	}
	if line["method"] != http.MethodGet || line["path"] != "/x" {
		t.Errorf("method/path = %v %v", line["method"], line["path"])
	}
}

func TestMiddlewareHonorsInboundRequestID(t *testing.T) {
	logger := New(&bytes.Buffer{}, "info")
	var seenID string
	h := Middleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenID = RequestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(RequestIDHeader, "trusted-upstream-id")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if seenID != "trusted-upstream-id" {
		t.Errorf("inbound request id not honored: %q", seenID)
	}
}
