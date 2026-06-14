// Package safety is the DETERMINISTIC safety layer (no LLM). It validates a
// generated plan between the model and the user: rules either correct the plan
// (cap out-of-bounds work) or flag it for regeneration (hard contraindication).
// E13 owns load/volume bounds; E7 plugs its injury-contraindication rule into the
// same Rule interface. E5 calls Validator.Validate as its single safety entrypoint.
package safety

// Plan is the generated session the safety layer validates. Minimal MVP shape;
// E5 produces richer plans. Kept decoupled from other packages on purpose.
type Plan struct {
	Exercises []Exercise
}

// Exercise is one prescribed movement within a plan.
type Exercise struct {
	Name     string
	Movement string // canonical movement key for contraindication matching (E7)
	Region   string // body region targeted, for injury matching (E7)
	Sets     int
	Reps     int
	LoadKg   float64
}

// TotalSets returns the session's total working sets (volume proxy).
func (p Plan) TotalSets() int {
	total := 0
	for _, e := range p.Exercises {
		total += e.Sets
	}
	return total
}

// MemoryView is the read-only slice of Coach Memory the rules consult. E7 fills
// Injuries; defining it here is the agreed E7<->E13 interface.
type MemoryView struct {
	Injuries []Injury
}

// Injury is an active/managed condition the contraindication rule (E7-PR5) reads.
type Injury struct {
	Region         string
	Status         string   // active-flare | managed | recurring-but-fine | resolved
	AvoidMovements []string // canonical movement keys to avoid
}

// Action is what a rule did about a finding.
type Action string

const (
	// ActionCap means the plan was corrected in place (value clamped).
	ActionCap Action = "cap"
	// ActionBlock means the plan must be regenerated (cannot be safely corrected).
	ActionBlock Action = "block"
)

// Finding records one safety adjustment or violation, for audit (E15-S3).
type Finding struct {
	Rule   string `json:"rule"`
	Action Action `json:"action"`
	Detail string `json:"detail"`
}

// Outcome is the pipeline verdict.
type Outcome string

const (
	Pass       Outcome = "pass"
	Corrected  Outcome = "corrected"
	Regenerate Outcome = "regenerate"
)

// Result is the outcome of validating a plan.
type Result struct {
	Outcome  Outcome   `json:"outcome"`
	Plan     Plan      `json:"-"` // possibly-corrected plan
	Findings []Finding `json:"findings"`
}

// Rule is a deterministic safety rule. Apply returns a (possibly corrected) plan
// and any findings. Both E13's bounds rule and E7's contraindication rule
// implement this; the Validator composes them.
type Rule interface {
	Name() string
	Apply(plan Plan, mem MemoryView) (Plan, []Finding)
}
