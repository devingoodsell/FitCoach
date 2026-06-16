package coaching

import (
	"strings"

	"pro.d11l.fitcoach/backend/internal/safety"
)

// exRef points back at the Session exercise a flattened safety.Exercise came
// from, so corrections (caps) can be applied in place.
type exRef struct {
	block *[]Exercise
	idx   int
}

// collectExercises flattens every block into a safety.Plan (volume + movement
// checks span the whole session) while keeping a parallel ref list to map
// corrections back. Order is stable: warmup, main work, accessory, aging items.
func collectExercises(s *Session) (safety.Plan, []exRef) {
	var plan safety.Plan
	var refs []exRef
	add := func(block *[]Exercise) {
		for i := range *block {
			plan.Exercises = append(plan.Exercises, toSafetyExercise((*block)[i]))
			refs = append(refs, exRef{block: block, idx: i})
		}
	}
	add(&s.Warmup)
	add(&s.MainWork)
	add(&s.Accessory)
	add(&s.AgingBlock.Items)
	return plan, refs
}

// toSafetyExercise collapses an exercise's sets into the single-set shape the
// deterministic rules check: total set count, and the heaviest reps/load across
// sets (conservative — bounds clamp the worst case).
func toSafetyExercise(e Exercise) safety.Exercise {
	maxReps := 0
	var maxLoad float64
	for _, s := range e.Sets {
		if s.Reps > maxReps {
			maxReps = s.Reps
		}
		if s.LoadKg != nil && *s.LoadKg > maxLoad {
			maxLoad = *s.LoadKg
		}
	}
	return safety.Exercise{
		Name:     e.Name,
		Movement: e.Movement,
		Region:   e.Region,
		Sets:     len(e.Sets),
		Reps:     maxReps,
		LoadKg:   maxLoad,
	}
}

// applyCaps maps a corrected (capped) plan back onto the Session: it clamps each
// set's reps/load to the corrected ceiling and truncates the set count. It is
// only used on the Corrected outcome, where the bounds rule preserves exercise
// order and count (the contraindication rule, which removes exercises, forces
// Regenerate instead). After capping, empty-set exercises are pruned.
func applyCaps(refs []exRef, corrected safety.Plan) {
	for i, ref := range refs {
		if i >= len(corrected.Exercises) {
			break
		}
		c := corrected.Exercises[i]
		ex := &(*ref.block)[ref.idx]
		for j := range ex.Sets {
			if c.Reps > 0 && ex.Sets[j].Reps > c.Reps {
				ex.Sets[j].Reps = c.Reps
			}
			if c.LoadKg > 0 && ex.Sets[j].LoadKg != nil && *ex.Sets[j].LoadKg > c.LoadKg {
				v := c.LoadKg
				ex.Sets[j].LoadKg = &v
			}
		}
		if c.Sets >= 0 && c.Sets < len(ex.Sets) {
			ex.Sets = ex.Sets[:c.Sets]
		}
	}
	pruneEmpty(refsBlocks(refs))
}

// refsBlocks returns the distinct block pointers touched, so pruning runs once
// per block.
func refsBlocks(refs []exRef) []*[]Exercise {
	seen := map[*[]Exercise]bool{}
	var out []*[]Exercise
	for _, r := range refs {
		if !seen[r.block] {
			seen[r.block] = true
			out = append(out, r.block)
		}
	}
	return out
}

func pruneEmpty(blocks []*[]Exercise) {
	for _, block := range blocks {
		kept := (*block)[:0]
		for _, e := range *block {
			if len(e.Sets) > 0 {
				kept = append(kept, e)
			}
		}
		*block = kept
	}
}

// toFindings maps safety findings to the transport shape.
func toFindings(in []safety.Finding) []SafetyFinding {
	if len(in) == 0 {
		return nil
	}
	out := make([]SafetyFinding, 0, len(in))
	for _, f := range in {
		out = append(out, SafetyFinding{Rule: f.Rule, Action: string(f.Action), Detail: f.Detail})
	}
	return out
}

// describeFindings renders findings into a short note fed back to the model on a
// regeneration attempt.
func describeFindings(in []safety.Finding) string {
	parts := make([]string, 0, len(in))
	for _, f := range in {
		parts = append(parts, f.Detail)
	}
	return strings.Join(parts, "; ")
}
