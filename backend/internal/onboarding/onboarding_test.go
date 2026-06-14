package onboarding

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

func fixedNow() time.Time { return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC) }

func strptr(s string) *string   { return &s }
func f64ptr(f float64) *float64 { return &f }

func TestProfileDerivedAgeFromDOB(t *testing.T) {
	p := Profile{Dob: strptr("1986-01-01")}
	age, ok := p.DerivedAge(fixedNow())
	if !ok || age != 40 {
		t.Fatalf("age = %d ok=%v, want 40 true", age, ok)
	}
}

func TestProfileValidation(t *testing.T) {
	now := fixedNow()
	// Missing sex and age.
	if err := (Profile{Experience: Experience{Level: LevelNovice}}).Validate(now); err == nil {
		t.Error("expected validation error for missing sex/age")
	}
	// Valid.
	ok := Profile{Sex: SexMale, Age: intptr(30), Experience: Experience{Level: LevelIntermediate}}
	if err := ok.Validate(now); err != nil {
		t.Errorf("valid profile rejected: %v", err)
	}
	// Bad level.
	bad := Profile{Sex: SexMale, Age: intptr(30), Experience: Experience{Level: "pro"}}
	if err := bad.Validate(now); err == nil {
		t.Error("expected error for invalid level")
	}
}

func intptr(i int) *int { return &i }

func TestGoalWeightsNormalize(t *testing.T) {
	g := GoalWeights{Strength: 3, Healthspan: 1}
	if err := g.Validate(); err != nil {
		t.Fatalf("valid goals rejected: %v", err)
	}
	n := g.Normalized()
	if n.Strength != 0.75 || n.Healthspan != 0.25 {
		t.Fatalf("normalized = %+v, want 0.75/0.25", n)
	}
	if err := (GoalWeights{}).Validate(); err == nil {
		t.Error("expected error for all-zero goals")
	}
}

func TestSaveProfileAppendsWeightHistoryAndDefaultsAging(t *testing.T) {
	svc := NewService(newFakeStore(), fixedNow)
	uid, _ := uuid.NewV7()
	ctx := context.Background()

	// First save with a weight.
	p1, err := svc.SaveProfile(ctx, uid, Profile{
		Sex: SexFemale, Age: intptr(65), WeightKg: f64ptr(70), Experience: Experience{Level: LevelNovice},
	})
	if err != nil {
		t.Fatalf("save1: %v", err)
	}
	if len(p1.WeightHistory) != 1 || p1.WeightHistory[0].Kg != 70 {
		t.Fatalf("weight history not recorded: %+v", p1.WeightHistory)
	}
	if p1.AgingEmphases == nil || p1.AgingEmphases.BoneBalance != 0.35 {
		t.Fatalf("aging emphases not defaulted for age 65: %+v", p1.AgingEmphases)
	}

	// Second save with a new weight appends, preserving history.
	p2, err := svc.SaveProfile(ctx, uid, Profile{
		Sex: SexFemale, Age: intptr(65), WeightKg: f64ptr(69), Experience: Experience{Level: LevelNovice},
	})
	if err != nil {
		t.Fatalf("save2: %v", err)
	}
	if len(p2.WeightHistory) != 2 || p2.WeightHistory[1].Kg != 69 {
		t.Fatalf("weight history not appended: %+v", p2.WeightHistory)
	}
}

func TestSaveGoalsPersistsNormalized(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, fixedNow)
	uid, _ := uuid.NewV7()

	if _, err := svc.SaveGoals(context.Background(), uid, GoalWeights{Strength: 1, Performance: 1}); err != nil {
		t.Fatalf("save goals: %v", err)
	}
	var stored GoalWeights
	_ = json.Unmarshal(store.data[uid][memory.SectionGoals], &stored)
	if stored.Strength != 0.5 || stored.Performance != 0.5 {
		t.Fatalf("stored goals not normalized: %+v", stored)
	}
}

func TestSaveScheduleValidation(t *testing.T) {
	svc := NewService(newFakeStore(), fixedNow)
	uid, _ := uuid.NewV7()
	_, err := svc.SaveSchedule(context.Background(), uid, Schedule{DaysPerWeek: 0, SessionLengthMin: 60})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}
