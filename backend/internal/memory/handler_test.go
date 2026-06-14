package memory

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

// fakeStore is an in-memory memoryStore for handler tests.
type fakeStore struct {
	sections map[uuid.UUID]map[Section]SectionRecord
	workouts map[uuid.UUID]map[string]WorkoutLog
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		sections: map[uuid.UUID]map[Section]SectionRecord{},
		workouts: map[uuid.UUID]map[string]WorkoutLog{},
	}
}

func (f *fakeStore) GetAll(_ context.Context, userID uuid.UUID) ([]SectionRecord, error) {
	var out []SectionRecord
	for _, s := range AllSections {
		if rec, ok := f.sections[userID][s]; ok {
			out = append(out, rec)
		}
	}
	return out, nil
}

func (f *fakeStore) GetSection(_ context.Context, userID uuid.UUID, section Section) (SectionRecord, error) {
	if !IsValidSection(section) {
		return SectionRecord{}, ErrUnknownSection
	}
	rec, ok := f.sections[userID][section]
	if !ok {
		return SectionRecord{}, ErrSectionNotFound
	}
	return rec, nil
}

func (f *fakeStore) PutSection(_ context.Context, userID uuid.UUID, section Section, data json.RawMessage) (SectionRecord, error) {
	if !IsValidSection(section) {
		return SectionRecord{}, ErrUnknownSection
	}
	if f.sections[userID] == nil {
		f.sections[userID] = map[Section]SectionRecord{}
	}
	rec := SectionRecord{Section: section, Version: CurrentVersions[section], Data: data, UpdatedAt: time.Now()}
	f.sections[userID][section] = rec
	return rec, nil
}

func (f *fakeStore) RecordWorkout(_ context.Context, userID uuid.UUID, clientSessionID string, data json.RawMessage, performedAt time.Time) (WorkoutLog, error) {
	if f.workouts[userID] == nil {
		f.workouts[userID] = map[string]WorkoutLog{}
	}
	if existing, ok := f.workouts[userID][clientSessionID]; ok {
		existing.Data = data // idempotent: keep id, update payload
		f.workouts[userID][clientSessionID] = existing
		return existing, nil
	}
	id, _ := uuid.NewV7()
	w := WorkoutLog{ID: id, ClientSessionID: clientSessionID, Version: WorkoutLogVersion, Data: data, PerformedAt: performedAt}
	f.workouts[userID][clientSessionID] = w
	return w, nil
}

func (f *fakeStore) RecentWorkouts(_ context.Context, userID uuid.UUID, _ int) ([]WorkoutLog, error) {
	var out []WorkoutLog
	for _, w := range f.workouts[userID] {
		out = append(out, w)
	}
	return out, nil
}

type fakeAuth struct{ id uuid.UUID }

func (f fakeAuth) ParseAccessToken(string) (uuid.UUID, error) { return f.id, nil }

func testRouter(uid uuid.UUID) (*httpx.Router, *fakeStore) {
	store := newFakeStore()
	h := NewHandler(store, logging.New(io.Discard, "error"))
	r := httpx.NewRouter()
	h.Register(r, auth.RequireAuth(fakeAuth{id: uid}))
	return r, store
}

func req(r *httpx.Router, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	rq := httptest.NewRequest(method, path, strings.NewReader(body))
	rq.Header.Set("Authorization", "Bearer t")
	r.ServeHTTP(rec, rq)
	return rec
}

func TestPutThenGetSection(t *testing.T) {
	uid, _ := uuid.NewV7()
	r, _ := testRouter(uid)

	rec := req(r, http.MethodPut, "/memory/profile", `{"data":{"age":40}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("put status = %d (%s)", rec.Code, rec.Body)
	}
	rec = req(r, http.MethodGet, "/memory/profile", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d", rec.Code)
	}
	var got SectionRecord
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Section != SectionProfile || string(got.Data) != `{"age":40}` {
		t.Fatalf("record = %+v", got)
	}
}

func TestGetSectionUnknownAndMissing(t *testing.T) {
	uid, _ := uuid.NewV7()
	r, _ := testRouter(uid)

	if rec := req(r, http.MethodGet, "/memory/bogus", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown section status = %d, want 400", rec.Code)
	}
	if rec := req(r, http.MethodGet, "/memory/goals", ""); rec.Code != http.StatusNotFound {
		t.Errorf("missing section status = %d, want 404", rec.Code)
	}
}

func TestRecordWorkoutIsIdempotent(t *testing.T) {
	uid, _ := uuid.NewV7()
	r, _ := testRouter(uid)

	body := `{"client_session_id":"sess-1","performed_at":"2026-06-14T12:00:00Z","data":{"sets":5}}`
	rec := req(r, http.MethodPost, "/workouts", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("first record status = %d (%s)", rec.Code, rec.Body)
	}
	var first WorkoutLog
	_ = json.Unmarshal(rec.Body.Bytes(), &first)

	rec = req(r, http.MethodPost, "/workouts", body)
	var second WorkoutLog
	_ = json.Unmarshal(rec.Body.Bytes(), &second)
	if first.ID != second.ID {
		t.Errorf("idempotency broken: ids %s != %s", first.ID, second.ID)
	}

	// Only one workout exists.
	rec = req(r, http.MethodGet, "/workouts", "")
	var list workoutsResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list.Workouts) != 1 {
		t.Errorf("expected 1 workout, got %d", len(list.Workouts))
	}
}

func TestRecordWorkoutRequiresSessionID(t *testing.T) {
	uid, _ := uuid.NewV7()
	r, _ := testRouter(uid)
	rec := req(r, http.MethodPost, "/workouts", `{"data":{"sets":5}}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestMemoryRequiresAuth(t *testing.T) {
	uid, _ := uuid.NewV7()
	r, _ := testRouter(uid)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/memory", nil)) // no auth header
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
