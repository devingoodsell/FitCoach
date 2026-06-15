// Package diet computes lightweight, preference-aware daily nutrition targets and
// guidance (E11). No food logging: just calorie/protein ranges derived from the
// user model and training load, plus pattern-aware suggestions. All output is
// framed as guidance, not medical advice (E13).
package diet

import (
	"math"

	"pro.d11l.fitcoach/backend/internal/onboarding"
)

// Targets are daily ranges. Zero numeric values with LowConfidence=true mean we
// lacked the inputs (e.g. weight) to compute them.
type Targets struct {
	CaloriesMin   int  `json:"calories_min"`
	CaloriesMax   int  `json:"calories_max"`
	ProteinMinG   int  `json:"protein_min_g"`
	ProteinMaxG   int  `json:"protein_max_g"`
	LowConfidence bool `json:"low_confidence"`
}

// Inputs is the model slice the computation needs.
type Inputs struct {
	Age         int
	Sex         string // male | female | other
	WeightKg    float64
	HasWeight   bool
	HeightCm    float64
	HasHeight   bool
	Goals       onboarding.GoalWeights // normalized
	DaysPerWeek int
}

const (
	assumedHeightCm = 170.0
	minProteinPerKg = 1.6
	maxProteinPerKg = 2.2
)

// ComputeTargets derives calorie and protein ranges deterministically. Without a
// weight it cannot compute numbers and returns LowConfidence. A missing height is
// estimated (also LowConfidence) so calories degrade gracefully.
func ComputeTargets(in Inputs) Targets {
	if !in.HasWeight || in.WeightKg <= 0 {
		return Targets{LowConfidence: true}
	}

	// Protein: g/kg scaled by strength + body-composition emphasis.
	perKg := minProteinPerKg + 0.6*(in.Goals.Strength+in.Goals.BodyComposition)
	if perKg > maxProteinPerKg {
		perKg = maxProteinPerKg
	}
	proteinCenter := perKg * in.WeightKg

	// Calories: Mifflin-St Jeor BMR x activity x goal adjustment.
	height := in.HeightCm
	lowConfidence := false
	if !in.HasHeight || height <= 0 {
		height = assumedHeightCm
		lowConfidence = true
	}
	bmr := 10*in.WeightKg + 6.25*height - 5*float64(in.Age) + sexConstant(in.Sex)
	activity := 1.2 + 0.06*float64(clampDays(in.DaysPerWeek))
	tdee := bmr * activity
	adjustment := 1.0 - 0.15*in.Goals.BodyComposition + 0.08*in.Goals.Strength + 0.05*in.Goals.Performance
	calorieCenter := tdee * adjustment

	return Targets{
		CaloriesMin:   roundTo(calorieCenter-150, 10),
		CaloriesMax:   roundTo(calorieCenter+150, 10),
		ProteinMinG:   roundTo(proteinCenter-10, 5),
		ProteinMaxG:   roundTo(proteinCenter+10, 5),
		LowConfidence: lowConfidence,
	}
}

func sexConstant(sex string) float64 {
	switch sex {
	case "male":
		return 5
	case "female":
		return -161
	default:
		return -78 // average of male/female for "other"
	}
}

func clampDays(d int) int {
	if d < 0 {
		return 0
	}
	if d > 7 {
		return 7
	}
	return d
}

func roundTo(v float64, nearest int) int {
	if nearest <= 0 {
		return int(math.Round(v))
	}
	return int(math.Round(v/float64(nearest))) * nearest
}
