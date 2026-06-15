package diet

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/memory"
	"pro.d11l.fitcoach/backend/internal/onboarding"
)

func TestComputeTargetsKnownProfile(t *testing.T) {
	// age 40 male, 80kg, 180cm, strength/body-comp split, 4 days/week.
	in := Inputs{
		Age: 40, Sex: "male", WeightKg: 80, HasWeight: true, HeightCm: 180, HasHeight: true,
		Goals:       onboarding.GoalWeights{Strength: 0.5, BodyComposition: 0.5},
		DaysPerWeek: 4,
	}
	got := ComputeTargets(in)

	// protein: perKg = 1.6 + 0.6*(1.0) = 2.2 (cap) -> 176g center, ±10 rounded to 5.
	if got.ProteinMinG != 165 || got.ProteinMaxG != 185 {
		t.Errorf("protein range = %d-%d, want 165-185", got.ProteinMinG, got.ProteinMaxG)
	}
	// calories center ~2404; min/max are center∓150 rounded to 10.
	if got.CaloriesMin < 2240 || got.CaloriesMin > 2270 {
		t.Errorf("calories_min = %d, want ~2250", got.CaloriesMin)
	}
	if got.CaloriesMax-got.CaloriesMin != 300 {
		t.Errorf("calorie band = %d, want 300", got.CaloriesMax-got.CaloriesMin)
	}
	if got.LowConfidence {
		t.Error("should be high confidence with weight+height")
	}
}

func TestComputeTargetsWithoutWeightIsLowConfidence(t *testing.T) {
	got := ComputeTargets(Inputs{Age: 30, Sex: "female", HasWeight: false})
	if !got.LowConfidence || got.CaloriesMin != 0 || got.ProteinMinG != 0 {
		t.Fatalf("expected low-confidence zeros, got %+v", got)
	}
}

func TestComputeTargetsWithoutHeightDegrades(t *testing.T) {
	got := ComputeTargets(Inputs{Age: 30, Sex: "female", WeightKg: 65, HasWeight: true})
	if !got.LowConfidence {
		t.Error("missing height should flag low confidence")
	}
	if got.ProteinMinG == 0 {
		t.Error("protein should still compute from weight")
	}
}

func TestVeganGuidanceHasNoAnimalProtein(t *testing.T) {
	animal := []string{"chicken", "beef", "fish", "shrimp", "egg", "yogurt", "whey", "cottage", "pork"}
	for _, line := range Guidance(onboarding.DietVegan) {
		low := strings.ToLower(line)
		for _, a := range animal {
			if strings.Contains(low, a) {
				t.Errorf("vegan guidance contains animal source %q: %s", a, line)
			}
		}
	}
	// And it should suggest plant sources.
	joined := strings.ToLower(strings.Join(Guidance(onboarding.DietVegan), " "))
	if !strings.Contains(joined, "tofu") && !strings.Contains(joined, "lentils") {
		t.Errorf("vegan guidance missing plant proteins: %s", joined)
	}
}

func TestPostWorkoutNoteDiffersByLoad(t *testing.T) {
	heavy := PostWorkoutNote(true, onboarding.DietOmnivore)
	light := PostWorkoutNote(false, onboarding.DietOmnivore)
	if heavy == light {
		t.Fatal("heavy and light notes should differ")
	}
	if !strings.Contains(strings.ToLower(heavy), "refuel") {
		t.Errorf("heavy note should emphasize refueling: %s", heavy)
	}
}

func TestPostWorkoutNoteRespectsVegan(t *testing.T) {
	note := strings.ToLower(PostWorkoutNote(true, onboarding.DietVegan))
	for _, a := range []string{"chicken", "beef", "fish", "egg", "whey"} {
		if strings.Contains(note, a) {
			t.Errorf("vegan note contains %q: %s", a, note)
		}
	}
}

// --- service integration with a fake store ---

type fakeStore struct {
	data map[memory.Section]json.RawMessage
}

func (f fakeStore) GetSection(_ context.Context, _ uuid.UUID, section memory.Section) (memory.SectionRecord, error) {
	raw, ok := f.data[section]
	if !ok {
		return memory.SectionRecord{}, memory.ErrSectionNotFound
	}
	return memory.SectionRecord{Section: section, Version: 1, Data: raw}, nil
}

func TestServiceTargetsReadsMemory(t *testing.T) {
	store := fakeStore{data: map[memory.Section]json.RawMessage{
		memory.SectionProfile:  json.RawMessage(`{"sex":"male","age":40,"height_cm":180,"weight_history":[{"kg":80,"recorded_at":"2026-06-14T12:00:00Z"}],"experience":{"level":"intermediate"}}`),
		memory.SectionGoals:    json.RawMessage(`{"strength":0.5,"body_composition":0.5}`),
		memory.SectionDiet:     json.RawMessage(`{"pattern":"vegan"}`),
		memory.SectionSchedule: json.RawMessage(`{"days_per_week":4,"session_length_min":60}`),
	}}
	svc := NewService(store, func() time.Time { return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC) })

	res, err := svc.Targets(context.Background(), uuid.Nil)
	if err != nil {
		t.Fatalf("Targets: %v", err)
	}
	if res.Targets.ProteinMinG == 0 {
		t.Error("expected protein computed from weight history")
	}
	if res.Pattern != "vegan" {
		t.Errorf("pattern = %q, want vegan", res.Pattern)
	}
	if res.Disclaimer == "" {
		t.Error("disclaimer must be attached")
	}
	if strings.Contains(strings.ToLower(strings.Join(res.Guidance, " ")), "chicken") {
		t.Error("vegan guidance leaked animal protein")
	}
}

func TestServiceTargetsLowConfidenceWithoutProfile(t *testing.T) {
	svc := NewService(fakeStore{data: map[memory.Section]json.RawMessage{}}, nil)
	res, err := svc.Targets(context.Background(), uuid.Nil)
	if err != nil {
		t.Fatalf("Targets: %v", err)
	}
	if !res.Targets.LowConfidence {
		t.Error("expected low confidence without a profile")
	}
	if len(res.Guidance) == 0 {
		t.Error("guidance should still be present")
	}
}
