package readiness

import (
	"context"
	"encoding/json"
	"errors"
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

// in-memory signal store
type fakeStore struct {
	rows map[string][]dayValue // key kind
}

func (f *fakeStore) Upsert(_ context.Context, _ uuid.UUID, samples []Sample, _ time.Time) error {
	if f.rows == nil {
		f.rows = map[string][]dayValue{}
	}
	for _, s := range samples {
		f.rows[s.Kind] = append([]dayValue{{Day: s.Day, Value: s.Value}}, f.rows[s.Kind]...)
	}
	return nil
}

func (f *fakeStore) recent(_ context.Context, _ uuid.UUID, kind string, _ int) ([]dayValue, error) {
	return f.rows[kind], nil
}

type fakeConsent struct{ granted bool }

func (f fakeConsent) HasConsent(context.Context, uuid.UUID, string) (bool, error) {
	return f.granted, nil
}

func newService(granted bool) (*Service, *fakeStore) {
	store := &fakeStore{}
	now := func() time.Time { return time.Date(2026, 6, 14, 7, 0, 0, 0, time.UTC) }
	return NewService(store, fakeConsent{granted: granted}, now), store
}

func TestIngestRequiresConsent(t *testing.T) {
	svc, _ := newService(false)
	err := svc.Ingest(context.Background(), uuid.Nil, []Sample{{Kind: KindHRV, Value: 60, Day: "2026-06-14"}})
	if !errors.Is(err, ErrConsentRequired) {
		t.Fatalf("err = %v, want ErrConsentRequired", err)
	}
}

func TestIngestThenComputeToday(t *testing.T) {
	svc, _ := newService(true)
	ctx := context.Background()
	uid, _ := uuid.NewV7()

	// 5 baseline days (varied, so variance > 0) + today for each metric.
	var samples []Sample
	days := []string{"2026-06-09", "2026-06-10", "2026-06-11", "2026-06-12", "2026-06-13"}
	hrvB := []float64{58, 60, 62, 59, 61}
	rhrB := []float64{54, 55, 56, 55, 54}
	sleepB := []float64{440, 460, 450, 470, 455}
	for i, d := range days {
		samples = append(samples,
			Sample{Kind: KindHRV, Value: hrvB[i], Day: d},
			Sample{Kind: KindRHR, Value: rhrB[i], Day: d},
			Sample{Kind: KindSleep, Value: sleepB[i], Day: d},
		)
	}
	// today: poor recovery
	samples = append(samples,
		Sample{Kind: KindHRV, Value: 40, Day: "2026-06-14"},
		Sample{Kind: KindRHR, Value: 65, Day: "2026-06-14"},
		Sample{Kind: KindSleep, Value: 360, Day: "2026-06-14"},
	)
	if err := svc.Ingest(ctx, uid, samples); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	score, err := svc.Today(ctx, uid)
	if err != nil {
		t.Fatalf("today: %v", err)
	}
	if score.Confidence != ConfidenceHigh {
		t.Errorf("confidence = %s, want high", score.Confidence)
	}
	if !PoorRecovery(score) {
		t.Errorf("expected poor-recovery trigger, score = %+v", score)
	}
}

func TestTodayWithNoDataIsLowConfidence(t *testing.T) {
	svc, _ := newService(true)
	uid, _ := uuid.NewV7()
	score, err := svc.Today(context.Background(), uid)
	if err != nil {
		t.Fatalf("today: %v", err)
	}
	if score.Confidence != ConfidenceLow || score.Value != 50 {
		t.Errorf("no-data score = %+v, want neutral/low", score)
	}
}

// --- handler ---

type fakeAuth struct{ id uuid.UUID }

func (f fakeAuth) ParseAccessToken(string) (uuid.UUID, error) { return f.id, nil }

func TestIngestHandlerConsentForbidden(t *testing.T) {
	svc, _ := newService(false)
	h := NewHandler(svc, logging.New(io.Discard, "error"))
	r := httpx.NewRouter()
	uid, _ := uuid.NewV7()
	h.Register(r, auth.RequireAuth(fakeAuth{id: uid}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/health/signals",
		strings.NewReader(`{"samples":[{"kind":"hrv_ms","value":60,"day":"2026-06-14"}]}`))
	req.Header.Set("Authorization", "Bearer t")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestReadinessHandlerReturnsScore(t *testing.T) {
	svc, _ := newService(true)
	h := NewHandler(svc, logging.New(io.Discard, "error"))
	r := httpx.NewRouter()
	uid, _ := uuid.NewV7()
	h.Register(r, auth.RequireAuth(fakeAuth{id: uid}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readiness", nil)
	req.Header.Set("Authorization", "Bearer t")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var score Score
	if err := json.Unmarshal(rec.Body.Bytes(), &score); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if score.Explanation == "" {
		t.Error("expected an explanation")
	}
}
