package safety

import (
	"fmt"
	"strings"
)

// ContraindicationRule blocks exercises that conflict with an active/managed
// injury (E7-S4). It is the injury half of the safety layer, sharing the Rule
// interface with the load/volume bounds rule (E13). The caller (E7) supplies the
// injuries in MemoryView; this rule is pure and stateless.
//
// An exercise is contraindicated when its movement is in an injury's avoid-list,
// or it targets the injured body region. Contraindications are hard: the offending
// exercise is removed (so a safe corrected plan exists) AND a block finding is
// emitted so the pipeline returns Regenerate rather than silently shipping a
// plan with gaps.
type ContraindicationRule struct{}

// NewContraindicationRule returns a ContraindicationRule.
func NewContraindicationRule() ContraindicationRule { return ContraindicationRule{} }

// Name identifies the rule in findings/logs.
func (ContraindicationRule) Name() string { return "injury_contraindication" }

// Apply removes contraindicated exercises and reports each as a block finding.
func (r ContraindicationRule) Apply(plan Plan, mem MemoryView) (Plan, []Finding) {
	if len(mem.Injuries) == 0 {
		return plan, nil
	}
	var kept []Exercise
	var findings []Finding
	for _, ex := range plan.Exercises {
		if inj, why, bad := r.contraindicated(ex, mem.Injuries); bad {
			findings = append(findings, Finding{
				Rule:   r.Name(),
				Action: ActionBlock,
				Detail: fmt.Sprintf("%q removed: %s (injury: %s)", ex.Name, why, inj.Region),
			})
			continue
		}
		kept = append(kept, ex)
	}
	return Plan{Exercises: kept}, findings
}

// contraindicated reports whether ex conflicts with any injury, and which.
func (r ContraindicationRule) contraindicated(ex Exercise, injuries []Injury) (Injury, string, bool) {
	for _, inj := range injuries {
		if ex.Movement != "" {
			for _, avoid := range inj.AvoidMovements {
				if strings.EqualFold(ex.Movement, avoid) {
					return inj, "movement " + ex.Movement + " is contraindicated", true
				}
			}
		}
		if ex.Region != "" && inj.Region != "" && strings.EqualFold(ex.Region, inj.Region) {
			return inj, "loads the injured region", true
		}
	}
	return Injury{}, "", false
}
