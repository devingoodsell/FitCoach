package coaching

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/auth"
	"pro.d11l.fitcoach/backend/internal/injury"
	"pro.d11l.fitcoach/backend/internal/location"
	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

type fakeAuth struct{ uid uuid.UUID }

func (f fakeAuth) ParseAccessToken(string) (uuid.UUID, error) { return f.uid, nil }

func newHandlerServer(t *testing.T, engine *Engine, replanner *Replanner) *httptest.Server {
	t.Helper()
	h := NewHandler(engine, replanner, logging.New(io.Discard, "error"))
	router := httpx.NewRouter()
	h.Register(router, auth.RequireAuth(fakeAuth{uid: uuid.New()}))
	return httptest.NewServer(router)
}

func authedGet(t *testing.T, url string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return resp
}

func authedPost(t *testing.T, url string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("Authorization", "Bearer token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return resp
}

func TestHandlerGenerateHappyPath(t *testing.T) {
	asm := &fakeAssembler{payload: profilePayload(40)}
	engine := newTestEngineFull(t, asm, &captureGenerator{body: cleanBody}, fakeInjuries{}, location.Doc{}, nil)
	srv := newHandlerServer(t, engine, nil)
	defer srv.Close()

	resp := authedPost(t, srv.URL+"/sessions/generate")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var s Session
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := s.Validate(); err != nil {
		t.Errorf("returned session invalid: %v", err)
	}
}

func TestHandlerGenerateUnsafeReturns422(t *testing.T) {
	asm := &fakeAssembler{payload: profilePayload(40)}
	// Model keeps returning a contraindicated plan -> ErrUnsafe -> 422.
	engine := newTestEngineFull(t, asm, &captureGenerator{body: modelBody}, fakeInjuries{view: kneeView()}, location.Doc{}, nil)
	srv := newHandlerServer(t, engine, nil)
	defer srv.Close()

	resp := authedPost(t, srv.URL+"/sessions/generate")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestHandlerReplanCheckRequiresSince(t *testing.T) {
	replanner := NewReplanner(fakeInjuryDoc{}, fakeLocations{}, goodReadiness(), nil)
	srv := newHandlerServer(t, nil, replanner)
	defer srv.Close()

	resp := authedGet(t, srv.URL+"/sessions/replan-check")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 when 'since' is missing", resp.StatusCode)
	}
}

func TestHandlerReplanCheckReturnsDecision(t *testing.T) {
	replanner := NewReplanner(
		fakeInjuryDoc{doc: injury.Doc{ChangedAt: ptr(replanSince.Add(time.Hour))}},
		fakeLocations{}, goodReadiness(), nil)
	srv := newHandlerServer(t, nil, replanner)
	defer srv.Close()

	resp := authedGet(t, srv.URL+"/sessions/replan-check?since="+replanSince.Format(time.RFC3339))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var dec ReplanDecision
	if err := json.NewDecoder(resp.Body).Decode(&dec); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !dec.ReplanNeeded || !hasReason(dec, ReasonInjuryChanged) {
		t.Errorf("decision = %+v, want injury_changed", dec)
	}
}
