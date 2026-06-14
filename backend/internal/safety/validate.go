package safety

import (
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// Validator composes deterministic rules into one safety entrypoint for E5.
type Validator struct {
	rules  []Rule
	logger *logging.Logger
}

// NewValidator builds a Validator from rules applied in order. logger may be nil.
func NewValidator(logger *logging.Logger, rules ...Rule) *Validator {
	return &Validator{rules: rules, logger: logger}
}

// Validate runs every rule in order against the plan and returns the verdict and
// the (possibly corrected) plan. Any block finding yields Regenerate; otherwise
// any cap yields Corrected; otherwise Pass. Findings are logged for audit (E15-S3).
// Deterministic: same inputs always produce the same result.
func (v *Validator) Validate(plan Plan, mem MemoryView) Result {
	current := plan
	var findings []Finding
	for _, rule := range v.rules {
		corrected, ruleFindings := rule.Apply(current, mem)
		current = corrected
		findings = append(findings, ruleFindings...)
	}

	outcome := Pass
	for _, f := range findings {
		if f.Action == ActionBlock {
			outcome = Regenerate
			break
		}
	}
	if outcome == Pass {
		for _, f := range findings {
			if f.Action == ActionCap {
				outcome = Corrected
				break
			}
		}
	}

	result := Result{Outcome: outcome, Plan: current, Findings: findings}
	v.log(result)
	return result
}

func (v *Validator) log(r Result) {
	if v.logger == nil || len(r.Findings) == 0 {
		return
	}
	for _, f := range r.Findings {
		v.logger.Warn("safety finding",
			"outcome", string(r.Outcome),
			"rule", f.Rule,
			"action", string(f.Action),
			"detail", f.Detail,
		)
	}
}
