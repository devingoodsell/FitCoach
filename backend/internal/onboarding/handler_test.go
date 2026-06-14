package onboarding

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/auth"
	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

type fakeAuth struct{ id uuid.UUID }

func (f fakeAuth) ParseAccessToken(string) (uuid.UUID, error) { return f.id, nil }

func testRouter() *httpx.Router {
	svc := NewService(newFakeStore(), fixedNow)
	h := NewHandler(svc, logging.New(io.Discard, "error"))
	r := httpx.NewRouter()
	uid, _ := uuid.NewV7()
	h.Register(r, auth.RequireAuth(fakeAuth{id: uid}))
	return r
}

func put(r *httpx.Router, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer t")
	r.ServeHTTP(rec, req)
	return rec
}

func TestProfileEndpointValidAndInvalid(t *testing.T) {
	r := testRouter()

	ok := put(r, "/onboarding/profile", `{"sex":"male","age":30,"experience":{"level":"novice"}}`)
	if ok.Code != http.StatusOK {
		t.Fatalf("valid profile status = %d (%s)", ok.Code, ok.Body)
	}

	bad := put(r, "/onboarding/profile", `{"age":30,"experience":{"level":"novice"}}`) // missing sex
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("invalid profile status = %d, want 400", bad.Code)
	}
	var resp validationResponse
	_ = json.Unmarshal(bad.Body.Bytes(), &resp)
	if resp.Error != "validation_failed" || resp.Fields["sex"] == "" {
		t.Errorf("expected sex field error, got %+v", resp)
	}
}

func TestGoalsEndpointNormalizes(t *testing.T) {
	r := testRouter()
	rec := put(r, "/onboarding/goals", `{"strength":2,"healthspan":2}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body)
	}
	var g GoalWeights
	_ = json.Unmarshal(rec.Body.Bytes(), &g)
	if g.Strength != 0.5 || g.Healthspan != 0.5 {
		t.Errorf("response not normalized: %+v", g)
	}
}

func TestScheduleEndpointRejectsBad(t *testing.T) {
	r := testRouter()
	rec := put(r, "/onboarding/schedule", `{"days_per_week":9,"session_length_min":60}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
