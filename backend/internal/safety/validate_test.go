package safety

import (
	"bytes"
	"strings"
	"testing"

	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// blockingRule is a test-only Rule that always flags a regenerate (simulates an
// E7 hard contraindication) so the pipeline's block path is covered before E7 lands.
type blockingRule struct{}

func (blockingRule) Name() string { return "test_block" }
func (blockingRule) Apply(plan Plan, _ MemoryView) (Plan, []Finding) {
	return plan, []Finding{{Rule: "test_block", Action: ActionBlock, Detail: "contraindicated"}}
}

func TestValidatePassesCleanPlan(t *testing.T) {
	v := NewValidator(nil, NewBoundsRule(DefaultBounds()))
	plan := Plan{Exercises: []Exercise{{Name: "Squat", Sets: 3, Reps: 5, LoadKg: 100}}}

	res := v.Validate(plan, MemoryView{})
	if res.Outcome != Pass {
		t.Fatalf("outcome = %s, want pass", res.Outcome)
	}
	if len(res.Findings) != 0 {
		t.Errorf("expected no findings, got %v", res.Findings)
	}
}

func TestValidateCorrectsOverBoundPlan(t *testing.T) {
	v := NewValidator(nil, NewBoundsRule(tightBounds()))
	plan := Plan{Exercises: []Exercise{{Name: "Deadlift", Sets: 99, Reps: 99, LoadKg: 999}}}

	res := v.Validate(plan, MemoryView{})
	if res.Outcome != Corrected {
		t.Fatalf("outcome = %s, want corrected", res.Outcome)
	}
	if res.Plan.Exercises[0].Sets != 5 {
		t.Errorf("plan not corrected: %+v", res.Plan.Exercises[0])
	}
}

func TestValidateRegeneratesOnBlock(t *testing.T) {
	v := NewValidator(nil, NewBoundsRule(DefaultBounds()), blockingRule{})
	plan := Plan{Exercises: []Exercise{{Name: "OHP", Movement: "overhead_press", Sets: 3, Reps: 5, LoadKg: 40}}}

	res := v.Validate(plan, MemoryView{})
	if res.Outcome != Regenerate {
		t.Fatalf("outcome = %s, want regenerate", res.Outcome)
	}
}

func TestValidateIsDeterministic(t *testing.T) {
	v := NewValidator(nil, NewBoundsRule(tightBounds()))
	plan := Plan{Exercises: []Exercise{
		{Name: "A", Sets: 8, Reps: 20, LoadKg: 200},
		{Name: "B", Sets: 7, Reps: 6, LoadKg: 50},
	}}
	a := v.Validate(plan, MemoryView{})
	b := v.Validate(plan, MemoryView{})

	if a.Outcome != b.Outcome || len(a.Findings) != len(b.Findings) {
		t.Fatalf("non-deterministic: %+v vs %+v", a, b)
	}
	for i := range a.Findings {
		if a.Findings[i] != b.Findings[i] {
			t.Errorf("finding %d differs: %v vs %v", i, a.Findings[i], b.Findings[i])
		}
	}
}

func TestValidateLogsFindings(t *testing.T) {
	var buf bytes.Buffer
	v := NewValidator(logging.New(&buf, "warn"), NewBoundsRule(tightBounds()))
	plan := Plan{Exercises: []Exercise{{Name: "Deadlift", Sets: 99, Reps: 5, LoadKg: 50}}}

	v.Validate(plan, MemoryView{})
	out := buf.String()
	if !strings.Contains(out, "safety finding") || !strings.Contains(out, "load_volume_bounds") {
		t.Errorf("expected safety findings logged, got: %s", out)
	}
}
