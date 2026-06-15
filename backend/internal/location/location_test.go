package location

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/memory"
)

// fakeStore is an in-memory sectionStore.
type fakeStore struct {
	data map[uuid.UUID]map[memory.Section]json.RawMessage
}

func newFakeStore() *fakeStore {
	return &fakeStore{data: map[uuid.UUID]map[memory.Section]json.RawMessage{}}
}

func (f *fakeStore) GetSection(_ context.Context, userID uuid.UUID, section memory.Section) (memory.SectionRecord, error) {
	raw, ok := f.data[userID][section]
	if !ok {
		return memory.SectionRecord{}, memory.ErrSectionNotFound
	}
	return memory.SectionRecord{Section: section, Version: 1, Data: raw}, nil
}

func (f *fakeStore) PutSection(_ context.Context, userID uuid.UUID, section memory.Section, data json.RawMessage) (memory.SectionRecord, error) {
	if f.data[userID] == nil {
		f.data[userID] = map[memory.Section]json.RawMessage{}
	}
	f.data[userID][section] = data
	return memory.SectionRecord{Section: section, Version: 1, Data: data}, nil
}

func newService() (*Service, uuid.UUID) {
	uid, _ := uuid.NewV7()
	return NewService(newFakeStore(), func() time.Time { return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC) }), uid
}

func TestGetEmptyWhenUnset(t *testing.T) {
	svc, uid := newService()
	doc, err := svc.Get(context.Background(), uid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(doc.Locations) != 0 || doc.Current != nil {
		t.Fatalf("expected empty doc, got %+v", doc)
	}
}

func TestAddUpdateDeleteLocation(t *testing.T) {
	svc, uid := newService()
	ctx := context.Background()

	gym, err := svc.Add(ctx, uid, "Equinox", []string{"barbell", "rack", ""})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if gym.ID == "" || len(gym.Equipment) != 2 { // empty string trimmed out
		t.Fatalf("unexpected location: %+v", gym)
	}

	if _, err := svc.Update(ctx, uid, gym.ID, "Equinox Downtown", []string{"dumbbells"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	doc, _ := svc.Get(ctx, uid)
	if doc.Locations[0].Name != "Equinox Downtown" || doc.Locations[0].Equipment[0] != "dumbbells" {
		t.Fatalf("update not applied: %+v", doc.Locations[0])
	}

	if err := svc.Delete(ctx, uid, gym.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	doc, _ = svc.Get(ctx, uid)
	if len(doc.Locations) != 0 {
		t.Fatalf("expected empty after delete, got %d", len(doc.Locations))
	}
}

func TestAddRejectsBlankName(t *testing.T) {
	svc, uid := newService()
	if _, err := svc.Add(context.Background(), uid, "   ", nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
}

func TestSetCurrentContextRequiresExistingLocation(t *testing.T) {
	svc, uid := newService()
	ctx := context.Background()
	if _, err := svc.SetCurrent(ctx, uid, "nope", "traveling"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}

	loc, _ := svc.Add(ctx, uid, "Hotel", []string{"dumbbells"})
	cur, err := svc.SetCurrent(ctx, uid, loc.ID, "traveling this week")
	if err != nil {
		t.Fatalf("SetCurrent: %v", err)
	}
	if cur.LocationID != loc.ID || cur.Note != "traveling this week" || cur.ChangedAt.IsZero() {
		t.Fatalf("unexpected current context: %+v", cur)
	}
}

func TestDeleteClearsCurrentContext(t *testing.T) {
	svc, uid := newService()
	ctx := context.Background()
	loc, _ := svc.Add(ctx, uid, "Home", nil)
	_, _ = svc.SetCurrent(ctx, uid, loc.ID, "")

	if err := svc.Delete(ctx, uid, loc.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	doc, _ := svc.Get(ctx, uid)
	if doc.Current != nil {
		t.Fatalf("current context should be cleared, got %+v", doc.Current)
	}
}
