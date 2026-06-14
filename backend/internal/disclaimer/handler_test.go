package disclaimer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pro.d11l.fitcoach/backend/internal/platform/httpx"
)

func TestGetDisclaimers(t *testing.T) {
	r := httpx.NewRouter()
	NewHandler().Register(r)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/disclaimers", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var doc Document
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.Version != Version {
		t.Errorf("version = %q, want %q", doc.Version, Version)
	}
	if doc.Medical == "" || doc.HealthData == "" {
		t.Errorf("disclaimer text must be non-empty: %+v", doc)
	}
}

func TestVersionMatchesConsentExpectation(t *testing.T) {
	// The disclaimer version is what the client records as the accepted consent
	// version (E1-S4); guard against it accidentally going blank.
	if Version == "" {
		t.Fatal("disclaimer Version must not be empty")
	}
}
