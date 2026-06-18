package consent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/auth"
	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// fakeStore is an in-memory consent store.
type fakeStore struct {
	records map[uuid.UUID][]Consent
}

func newFakeStore() *fakeStore { return &fakeStore{records: map[uuid.UUID][]Consent{}} }

func (f *fakeStore) Record(_ context.Context, userID uuid.UUID, ctype, version string, now time.Time) error {
	// prepend so latest-per-type is first, mirroring the real ORDER BY DESC
	f.records[userID] = append([]Consent{{Type: ctype, Version: version, AcceptedAt: now}}, f.records[userID]...)
	return nil
}

func (f *fakeStore) List(_ context.Context, userID uuid.UUID) ([]Consent, error) {
	seen := map[string]bool{}
	var out []Consent
	for _, c := range f.records[userID] {
		if seen[c.Type] {
			continue
		}
		seen[c.Type] = true
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeStore) Revoke(_ context.Context, userID uuid.UUID, ctype string, now time.Time) error {
	for i, c := range f.records[userID] {
		if c.Type == ctype && c.RevokedAt == nil {
			ts := now
			f.records[userID][i].RevokedAt = &ts
		}
	}
	return nil
}

// fakeAuth always authenticates as a fixed user, so RequireAuth populates context.
type fakeAuth struct{ id uuid.UUID }

func (f fakeAuth) ParseAccessToken(string) (uuid.UUID, error) { return f.id, nil }

func newTestRouter(h *Handler, userID uuid.UUID) *httpx.Router {
	r := httpx.NewRouter()
	h.Register(r, auth.RequireAuth(fakeAuth{id: userID}))
	return r
}

func do(r *httpx.Router, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	r.ServeHTTP(rec, req)
	return rec
}

func TestRecordAndListConsent(t *testing.T) {
	uid, _ := uuid.NewV7()
	h := NewHandler(newFakeStore(), logging.New(io.Discard, "error"),
		func() time.Time { return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC) })
	r := newTestRouter(h, uid)

	rec := do(r, http.MethodPost, "/consent", `{"type":"health_data","version":"v1"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("record status = %d, want 201 (%s)", rec.Code, rec.Body)
	}

	rec = do(r, http.MethodGet, "/consent", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", rec.Code)
	}
	var resp listResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Consents) != 1 || resp.Consents[0].Type != "health_data" || resp.Consents[0].Version != "v1" {
		t.Fatalf("unexpected consents: %+v", resp.Consents)
	}
}

func TestListReturnsLatestVersionPerType(t *testing.T) {
	uid, _ := uuid.NewV7()
	h := NewHandler(newFakeStore(), logging.New(io.Discard, "error"), nil)
	r := newTestRouter(h, uid)

	_ = do(r, http.MethodPost, "/consent", `{"type":"medical_disclaimer","version":"v1"}`)
	_ = do(r, http.MethodPost, "/consent", `{"type":"medical_disclaimer","version":"v2"}`)

	rec := do(r, http.MethodGet, "/consent", "")
	var resp listResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Consents) != 1 || resp.Consents[0].Version != "v2" {
		t.Fatalf("expected latest v2 only, got %+v", resp.Consents)
	}
}

func TestRevokeMarksConsentRevoked(t *testing.T) {
	uid, _ := uuid.NewV7()
	revokeAt := time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC)
	h := NewHandler(newFakeStore(), logging.New(io.Discard, "error"),
		func() time.Time { return revokeAt })
	r := newTestRouter(h, uid)

	if rec := do(r, http.MethodPost, "/consent", `{"type":"health_data","version":"v1"}`); rec.Code != http.StatusCreated {
		t.Fatalf("record status = %d, want 201", rec.Code)
	}

	rec := do(r, http.MethodPost, "/consent/health_data/revoke", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke status = %d, want 200 (%s)", rec.Code, rec.Body)
	}
	var revoked Consent
	if err := json.Unmarshal(rec.Body.Bytes(), &revoked); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if revoked.RevokedAt == nil || !revoked.RevokedAt.Equal(revokeAt) {
		t.Fatalf("expected revoked_at = %v, got %+v", revokeAt, revoked)
	}

	// The current state now shows health_data as revoked.
	rec = do(r, http.MethodGet, "/consent", "")
	var resp listResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Consents) != 1 || resp.Consents[0].RevokedAt == nil {
		t.Fatalf("expected health_data revoked in list, got %+v", resp.Consents)
	}
}

func TestRevokeIsIdempotentForAbsentConsent(t *testing.T) {
	uid, _ := uuid.NewV7()
	h := NewHandler(newFakeStore(), logging.New(io.Discard, "error"), nil)
	r := newTestRouter(h, uid)

	// Never consented to health_data; revoking is still a successful no-op.
	if rec := do(r, http.MethodPost, "/consent/health_data/revoke", ""); rec.Code != http.StatusOK {
		t.Fatalf("revoke status = %d, want 200", rec.Code)
	}
}

func TestRevokeRejectsUnknownType(t *testing.T) {
	uid, _ := uuid.NewV7()
	h := NewHandler(newFakeStore(), logging.New(io.Discard, "error"), nil)
	r := newTestRouter(h, uid)

	if rec := do(r, http.MethodPost, "/consent/marketing/revoke", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestRecordRejectsUnknownType(t *testing.T) {
	uid, _ := uuid.NewV7()
	h := NewHandler(newFakeStore(), logging.New(io.Discard, "error"), nil)
	r := newTestRouter(h, uid)

	rec := do(r, http.MethodPost, "/consent", `{"type":"marketing","version":"v1"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestConsentRequiresAuth(t *testing.T) {
	uid, _ := uuid.NewV7()
	h := NewHandler(newFakeStore(), logging.New(io.Discard, "error"), nil)
	r := newTestRouter(h, uid)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/consent", nil) // no Authorization header
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
