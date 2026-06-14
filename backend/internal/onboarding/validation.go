package onboarding

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ValidationError lists per-field problems; handlers render it as a 400.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	keys := make([]string, 0, len(e.Fields))
	for k := range e.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys) // deterministic message
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", k, e.Fields[k]))
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

func newValidationError() *ValidationError { return &ValidationError{Fields: map[string]string{}} }

func (e *ValidationError) add(field, msg string) { e.Fields[field] = msg }
func (e *ValidationError) orNil() error {
	if len(e.Fields) == 0 {
		return nil
	}
	return e
}

const dobLayout = "2006-01-02"

// DerivedAge returns the user's age in whole years from DOB (preferred) or the
// explicit Age fallback, evaluated against now.
func (p Profile) DerivedAge(now time.Time) (int, bool) {
	if p.Dob != nil {
		if dob, err := time.Parse(dobLayout, *p.Dob); err == nil {
			return yearsBetween(dob, now), true
		}
	}
	if p.Age != nil {
		return *p.Age, true
	}
	return 0, false
}

func yearsBetween(from, to time.Time) int {
	years := to.Year() - from.Year()
	if to.YearDay() < from.YearDay() {
		years--
	}
	return years
}

// Validate checks the profile against the minimum-to-generate rules. now is used
// to validate DOB-derived age ranges.
func (p Profile) Validate(now time.Time) error {
	v := newValidationError()
	if !validSex(p.Sex) {
		v.add("sex", "must be male, female, or other")
	}
	if p.Dob == nil && p.Age == nil {
		v.add("age", "provide date of birth or age")
	}
	if p.Dob != nil {
		if _, err := time.Parse(dobLayout, *p.Dob); err != nil {
			v.add("dob", "must be YYYY-MM-DD")
		}
	}
	if age, ok := p.DerivedAge(now); ok && (age < 13 || age > 120) {
		v.add("age", "must be between 13 and 120")
	}
	if p.HeightCm != nil && (*p.HeightCm < 50 || *p.HeightCm > 260) {
		v.add("height_cm", "must be between 50 and 260")
	}
	if p.WeightKg != nil && (*p.WeightKg < 20 || *p.WeightKg > 400) {
		v.add("weight_kg", "must be between 20 and 400")
	}
	if !validLevel(p.Experience.Level) {
		v.add("experience.level", "must be novice, intermediate, or advanced")
	}
	for i, lift := range p.Experience.BenchmarkLifts {
		if lift.Name == "" || lift.OneRepMaxKg <= 0 {
			v.add(fmt.Sprintf("experience.benchmark_lifts[%d]", i), "name and positive one_rep_max_kg required")
		}
	}
	return v.orNil()
}

// Validate ensures the goal distribution is non-negative with at least one
// positive weight (it is normalized to sum 1 on save).
func (g GoalWeights) Validate() error {
	v := newValidationError()
	for name, w := range g.byName() {
		if w < 0 {
			v.add("goals."+name, "must be >= 0")
		}
	}
	if g.sum() <= 0 {
		v.add("goals", "at least one goal weight must be > 0")
	}
	return v.orNil()
}

func (g GoalWeights) byName() map[string]float64 {
	return map[string]float64{
		"strength":         g.Strength,
		"healthspan":       g.Healthspan,
		"body_composition": g.BodyComposition,
		"performance":      g.Performance,
	}
}

func (g GoalWeights) sum() float64 {
	return g.Strength + g.Healthspan + g.BodyComposition + g.Performance
}

// Normalized returns the weights scaled to sum 1 (assumes a positive sum).
func (g GoalWeights) Normalized() GoalWeights {
	s := g.sum()
	if s <= 0 {
		return g
	}
	return GoalWeights{
		Strength:        g.Strength / s,
		Healthspan:      g.Healthspan / s,
		BodyComposition: g.BodyComposition / s,
		Performance:     g.Performance / s,
	}
}

// Validate checks the schedule.
func (s Schedule) Validate() error {
	v := newValidationError()
	if s.DaysPerWeek < 1 || s.DaysPerWeek > 7 {
		v.add("days_per_week", "must be between 1 and 7")
	}
	if s.SessionLengthMin < 10 || s.SessionLengthMin > 240 {
		v.add("session_length_min", "must be between 10 and 240")
	}
	return v.orNil()
}

// Validate checks the dietary preferences.
func (d DietPrefs) Validate() error {
	v := newValidationError()
	if !validDietPattern(d.Pattern) {
		v.add("pattern", "unknown dietary pattern")
	}
	return v.orNil()
}

// DefaultAgingEmphases infers healthspan emphases from age (E2-S8). Older users
// get more bone/balance, joint/tendon, and cardiovascular emphasis. Adjustable.
func DefaultAgingEmphases(age int) AgingEmphases {
	switch {
	case age >= 60:
		return AgingEmphases{BoneBalance: 0.35, JointTendon: 0.3, Vo2Max: 0.15, CardioBase: 0.2}
	case age >= 45:
		return AgingEmphases{BoneBalance: 0.25, JointTendon: 0.25, Vo2Max: 0.25, CardioBase: 0.25}
	default:
		return AgingEmphases{BoneBalance: 0.15, JointTendon: 0.2, Vo2Max: 0.35, CardioBase: 0.3}
	}
}
