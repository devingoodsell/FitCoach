package safety

import "testing"

func TestContraindicationBlocksByMovement(t *testing.T) {
	rule := NewContraindicationRule()
	plan := Plan{Exercises: []Exercise{
		{Name: "Back Squat", Movement: "squat", Sets: 3, Reps: 5},
		{Name: "Bench Press", Movement: "bench_press", Sets: 3, Reps: 5},
	}}
	mem := MemoryView{Injuries: []Injury{{Region: "left_knee", Status: "active_flare", AvoidMovements: []string{"squat"}}}}

	out, findings := rule.Apply(plan, mem)
	if len(out.Exercises) != 1 || out.Exercises[0].Name != "Bench Press" {
		t.Fatalf("squat should be removed, got %+v", out.Exercises)
	}
	if len(findings) != 1 || findings[0].Action != ActionBlock {
		t.Fatalf("expected one block finding, got %+v", findings)
	}
}

func TestContraindicationBlocksByRegion(t *testing.T) {
	rule := NewContraindicationRule()
	plan := Plan{Exercises: []Exercise{
		{Name: "Leg Extension", Region: "left_knee", Sets: 3, Reps: 12},
		{Name: "Pull-up", Region: "back", Sets: 3, Reps: 8},
	}}
	mem := MemoryView{Injuries: []Injury{{Region: "left_knee", Status: "managed"}}}

	out, findings := rule.Apply(plan, mem)
	if len(out.Exercises) != 1 || out.Exercises[0].Name != "Pull-up" {
		t.Fatalf("knee exercise should be removed, got %+v", out.Exercises)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}
}

func TestContraindicationPassesCleanPlan(t *testing.T) {
	rule := NewContraindicationRule()
	plan := Plan{Exercises: []Exercise{{Name: "Bench Press", Movement: "bench_press", Region: "chest"}}}
	mem := MemoryView{Injuries: []Injury{{Region: "left_knee", AvoidMovements: []string{"squat"}}}}

	out, findings := rule.Apply(plan, mem)
	if len(out.Exercises) != 1 || len(findings) != 0 {
		t.Fatalf("clean plan should be unchanged, got %+v / %+v", out, findings)
	}
}

func TestContraindicationNoInjuriesNoop(t *testing.T) {
	rule := NewContraindicationRule()
	plan := Plan{Exercises: []Exercise{{Name: "Squat", Movement: "squat"}}}
	out, findings := rule.Apply(plan, MemoryView{})
	if len(out.Exercises) != 1 || findings != nil {
		t.Fatalf("no injuries should be a no-op")
	}
}

// Integration with the Validator: a contraindication forces Regenerate.
func TestValidatorRegeneratesOnContraindication(t *testing.T) {
	v := NewValidator(nil, NewBoundsRule(DefaultBounds()), NewContraindicationRule())
	plan := Plan{Exercises: []Exercise{{Name: "Squat", Movement: "squat", Sets: 3, Reps: 5, LoadKg: 100}}}
	mem := MemoryView{Injuries: []Injury{{Region: "knee", AvoidMovements: []string{"squat"}}}}

	res := v.Validate(plan, mem)
	if res.Outcome != Regenerate {
		t.Fatalf("outcome = %s, want regenerate", res.Outcome)
	}
	if len(res.Plan.Exercises) != 0 {
		t.Errorf("contraindicated exercise should be removed from corrected plan")
	}
}
