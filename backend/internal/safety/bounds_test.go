package safety

import "testing"

func tightBounds() Bounds {
	return Bounds{MaxSetsPerExercise: 5, MaxRepsPerSet: 12, MaxLoadKg: 100, MaxTotalSets: 10}
}

func TestBoundsPassesWithinLimits(t *testing.T) {
	rule := NewBoundsRule(tightBounds())
	plan := Plan{Exercises: []Exercise{
		{Name: "Squat", Sets: 3, Reps: 5, LoadKg: 80},
		{Name: "Bench", Sets: 3, Reps: 8, LoadKg: 60},
	}}
	out, findings := rule.Apply(plan, MemoryView{})
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %v", findings)
	}
	if out.Exercises[0].Sets != 3 || out.Exercises[1].LoadKg != 60 {
		t.Errorf("within-bounds plan was modified: %+v", out)
	}
}

func TestBoundsCapsPerExercise(t *testing.T) {
	rule := NewBoundsRule(tightBounds())
	plan := Plan{Exercises: []Exercise{
		{Name: "Deadlift", Sets: 9, Reps: 20, LoadKg: 250},
	}}
	out, findings := rule.Apply(plan, MemoryView{})

	e := out.Exercises[0]
	if e.Sets != 5 || e.Reps != 12 || e.LoadKg != 100 {
		t.Fatalf("expected caps applied, got %+v", e)
	}
	if len(findings) != 3 {
		t.Errorf("expected 3 cap findings (sets/reps/load), got %d: %v", len(findings), findings)
	}
	for _, f := range findings {
		if f.Action != ActionCap {
			t.Errorf("expected cap action, got %s", f.Action)
		}
	}
}

func TestBoundsTrimsTotalVolumeFromEnd(t *testing.T) {
	rule := NewBoundsRule(tightBounds()) // MaxTotalSets = 10
	plan := Plan{Exercises: []Exercise{
		{Name: "A", Sets: 5, Reps: 5, LoadKg: 50},
		{Name: "B", Sets: 5, Reps: 5, LoadKg: 50},
		{Name: "C", Sets: 4, Reps: 5, LoadKg: 50}, // total 14 -> must drop 4
	}}
	out, findings := rule.Apply(plan, MemoryView{})

	if out.TotalSets() != 10 {
		t.Fatalf("total sets = %d, want 10", out.TotalSets())
	}
	// Trimmed from the end deterministically: C (4) removed first.
	if out.Exercises[2].Sets != 0 || out.Exercises[0].Sets != 5 {
		t.Errorf("unexpected trim distribution: %+v", out.Exercises)
	}
	var sawVolumeFinding bool
	for _, f := range findings {
		if f.Detail == "total volume 14 sets capped to 10" {
			sawVolumeFinding = true
		}
	}
	if !sawVolumeFinding {
		t.Errorf("expected total-volume finding, got %v", findings)
	}
}

func TestBoundsRuleNeverMutatesInput(t *testing.T) {
	rule := NewBoundsRule(tightBounds())
	plan := Plan{Exercises: []Exercise{{Name: "X", Sets: 99, Reps: 99, LoadKg: 999}}}
	_, _ = rule.Apply(plan, MemoryView{})
	if plan.Exercises[0].Sets != 99 {
		t.Errorf("Apply mutated the input plan: %+v", plan.Exercises[0])
	}
}
