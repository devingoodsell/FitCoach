package safety

import "fmt"

// Bounds are configurable, model-independent caps on a session's load and volume
// (E13-S2). They are conservative defaults; an operator can tune them.
type Bounds struct {
	MaxSetsPerExercise int
	MaxRepsPerSet      int
	MaxLoadKg          float64
	MaxTotalSets       int // session volume cap across all exercises
}

// DefaultBounds returns sane MVP limits.
func DefaultBounds() Bounds {
	return Bounds{
		MaxSetsPerExercise: 10,
		MaxRepsPerSet:      50,
		MaxLoadKg:          500,
		MaxTotalSets:       40,
	}
}

// BoundsRule caps a plan's per-exercise sets/reps/load and total session volume.
// It only ever corrects (caps); it never blocks.
type BoundsRule struct {
	b Bounds
}

// NewBoundsRule returns a BoundsRule with the given bounds.
func NewBoundsRule(b Bounds) BoundsRule { return BoundsRule{b: b} }

// Name identifies the rule in findings/logs.
func (r BoundsRule) Name() string { return "load_volume_bounds" }

// Apply clamps each exercise to the per-exercise caps, then trims total volume to
// MaxTotalSets by reducing sets from the last exercises (deterministic). Returns
// the corrected plan and a finding per cap applied.
func (r BoundsRule) Apply(plan Plan, _ MemoryView) (Plan, []Finding) {
	var findings []Finding
	exercises := make([]Exercise, len(plan.Exercises))
	copy(exercises, plan.Exercises)

	for i := range exercises {
		e := &exercises[i]
		if e.Sets > r.b.MaxSetsPerExercise {
			findings = append(findings, capFinding(e.Name, "sets", e.Sets, r.b.MaxSetsPerExercise))
			e.Sets = r.b.MaxSetsPerExercise
		}
		if e.Reps > r.b.MaxRepsPerSet {
			findings = append(findings, capFinding(e.Name, "reps", e.Reps, r.b.MaxRepsPerSet))
			e.Reps = r.b.MaxRepsPerSet
		}
		if e.LoadKg > r.b.MaxLoadKg {
			findings = append(findings, Finding{
				Rule:   r.Name(),
				Action: ActionCap,
				Detail: fmt.Sprintf("%s: load %.1fkg capped to %.1fkg", e.Name, e.LoadKg, r.b.MaxLoadKg),
			})
			e.LoadKg = r.b.MaxLoadKg
		}
	}

	// Trim total session volume deterministically: remove sets from the last
	// exercise with sets > 0 until within the cap.
	corrected := Plan{Exercises: exercises}
	if total := corrected.TotalSets(); total > r.b.MaxTotalSets {
		over := total - r.b.MaxTotalSets
		for i := len(exercises) - 1; i >= 0 && over > 0; i-- {
			reducible := exercises[i].Sets
			if reducible <= 0 {
				continue
			}
			cut := over
			if cut > reducible {
				cut = reducible
			}
			exercises[i].Sets -= cut
			over -= cut
		}
		findings = append(findings, Finding{
			Rule:   r.Name(),
			Action: ActionCap,
			Detail: fmt.Sprintf("total volume %d sets capped to %d", total, r.b.MaxTotalSets),
		})
	}
	return Plan{Exercises: exercises}, findings
}

func capFinding(name, field string, was, capped int) Finding {
	return Finding{
		Rule:   "load_volume_bounds",
		Action: ActionCap,
		Detail: fmt.Sprintf("%s: %s %d capped to %d", name, field, was, capped),
	}
}
