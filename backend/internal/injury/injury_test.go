package injury

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/memory"
)

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
	return NewService(newFakeStore(), nil, func() time.Time { return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC) }), uid
}

func TestAddValidatesRegionAndStatus(t *testing.T) {
	svc, uid := newService()
	ctx := context.Background()
	if _, err := svc.Add(ctx, uid, Injury{Status: StatusActiveFlare}); !errors.Is(err, ErrInvalid) {
		t.Errorf("missing region should be invalid, got %v", err)
	}
	if _, err := svc.Add(ctx, uid, Injury{Region: "knee", Status: "bogus"}); !errors.Is(err, ErrInvalid) {
		t.Errorf("bad status should be invalid, got %v", err)
	}
}

func TestAddUpdateDeleteLifecycleSetsTrigger(t *testing.T) {
	svc, uid := newService()
	ctx := context.Background()

	inj, err := svc.Add(ctx, uid, Injury{Region: "left_knee", Status: StatusActiveFlare})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if inj.ID == "" || inj.Severity != SeverityModerate {
		t.Fatalf("defaults not applied: %+v", inj)
	}
	doc, _ := svc.Get(ctx, uid)
	if doc.ChangedAt == nil {
		t.Error("ChangedAt should be set as a re-plan trigger")
	}

	inj.Status = StatusManaged
	if _, err := svc.Update(ctx, uid, inj.ID, inj); err != nil {
		t.Fatalf("Update: %v", err)
	}
	doc, _ = svc.Get(ctx, uid)
	if doc.Injuries[0].Status != StatusManaged {
		t.Errorf("status not updated: %+v", doc.Injuries[0])
	}

	if err := svc.Delete(ctx, uid, inj.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	doc, _ = svc.Get(ctx, uid)
	if len(doc.Injuries) != 0 {
		t.Errorf("expected empty after delete")
	}
}

func TestContraindicationsOnlyActiveOrManaged(t *testing.T) {
	svc, uid := newService()
	ctx := context.Background()
	_, _ = svc.Add(ctx, uid, Injury{Region: "left_knee", Status: StatusActiveFlare})
	_, _ = svc.Add(ctx, uid, Injury{Region: "shoulder", Status: StatusResolved})
	_, _ = svc.Add(ctx, uid, Injury{Region: "lower_back", Status: StatusManaged, AggravatingMovements: []string{"twist"}})

	cs, err := svc.Contraindications(ctx, uid)
	if err != nil {
		t.Fatalf("Contraindications: %v", err)
	}
	if len(cs) != 2 {
		t.Fatalf("expected 2 constraints (active+managed), got %d", len(cs))
	}
	// knee → squat avoided (region default), regardless of side.
	var kneeFound, backTwist bool
	for _, c := range cs {
		if c.Region == "left_knee" {
			for _, m := range c.AvoidMovements {
				if m == "squat" {
					kneeFound = true
				}
			}
		}
		if c.Region == "lower_back" {
			for _, m := range c.AvoidMovements {
				if m == "twist" {
					backTwist = true
				}
			}
		}
	}
	if !kneeFound {
		t.Error("knee injury should contraindicate squat")
	}
	if !backTwist {
		t.Error("injury-specific aggravating movement should be included")
	}
}

func TestSafetyViewFeedsSafetyLayer(t *testing.T) {
	svc, uid := newService()
	ctx := context.Background()
	_, _ = svc.Add(ctx, uid, Injury{Region: "knee", Status: StatusActiveFlare})
	_, _ = svc.Add(ctx, uid, Injury{Region: "elbow", Status: StatusRecurring}) // not constraining

	view, err := svc.SafetyView(ctx, uid)
	if err != nil {
		t.Fatalf("SafetyView: %v", err)
	}
	if len(view.Injuries) != 1 || view.Injuries[0].Region != "knee" {
		t.Fatalf("expected only the active knee injury, got %+v", view.Injuries)
	}
	if len(view.Injuries[0].AvoidMovements) == 0 {
		t.Error("expected derived avoid movements")
	}
}

func TestParseDraftHeuristics(t *testing.T) {
	svc, _ := newService()
	d := svc.ParseDraft("My left knee has a sharp pain when I squat")
	if d.Injury.Region != "left_knee" {
		t.Errorf("region = %q, want left_knee", d.Injury.Region)
	}
	if d.Injury.Severity != SeveritySevere {
		t.Errorf("severity = %q, want severe", d.Injury.Severity)
	}
	found := false
	for _, m := range d.Injury.AggravatingMovements {
		if m == "squat" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected squat as aggravating movement: %+v", d.Injury.AggravatingMovements)
	}
}

func TestParseDraftFlagsLowConfidence(t *testing.T) {
	svc, _ := newService()
	d := svc.ParseDraft("something feels off")
	if len(d.LowConfidenceFields) == 0 {
		t.Error("expected low-confidence fields for a vague description")
	}
}
