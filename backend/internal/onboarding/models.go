// Package onboarding captures and validates the user model (E2), persisting each
// section into versioned Coach Memory (E3). It owns typed shapes + validation for
// the minimum-to-generate fields and the optional ones; editing UI is E14.
package onboarding

import "time"

// Sex is biological sex, used for programming (not identity).
type Sex string

const (
	SexMale   Sex = "male"
	SexFemale Sex = "female"
	SexOther  Sex = "other"
)

func validSex(s Sex) bool { return s == SexMale || s == SexFemale || s == SexOther }

// Level is a self-rated experience level.
type Level string

const (
	LevelNovice       Level = "novice"
	LevelIntermediate Level = "intermediate"
	LevelAdvanced     Level = "advanced"
)

func validLevel(l Level) bool {
	return l == LevelNovice || l == LevelIntermediate || l == LevelAdvanced
}

// DietPattern is a dietary preference consumed by diet guidance (E11).
type DietPattern string

const (
	DietOmnivore    DietPattern = "omnivore"
	DietVegan       DietPattern = "vegan"
	DietVegetarian  DietPattern = "vegetarian"
	DietPescatarian DietPattern = "pescatarian"
	DietKosher      DietPattern = "kosher"
	DietHalal       DietPattern = "halal"
)

func validDietPattern(p DietPattern) bool {
	switch p {
	case DietOmnivore, DietVegan, DietVegetarian, DietPescatarian, DietKosher, DietHalal:
		return true
	}
	return false
}

// BenchmarkLift is an optional strength benchmark for calibration.
type BenchmarkLift struct {
	Name        string  `json:"name"`
	OneRepMaxKg float64 `json:"one_rep_max_kg"`
}

// Experience calibrates difficulty (E2-S3).
type Experience struct {
	TrainingAgeYears *float64        `json:"training_age_years,omitempty"`
	Level            Level           `json:"level"`
	BenchmarkLifts   []BenchmarkLift `json:"benchmark_lifts,omitempty"`
}

// AgingEmphases weight healthspan focuses (E2-S8); defaulted from age, adjustable.
type AgingEmphases struct {
	BoneBalance float64 `json:"bone_balance"`
	JointTendon float64 `json:"joint_tendon"`
	Vo2Max      float64 `json:"vo2max"`
	CardioBase  float64 `json:"cardio_base"`
}

// WeightEntry is a timestamped body-weight reading (E2-S2 trend tracking).
type WeightEntry struct {
	Kg         float64   `json:"kg"`
	RecordedAt time.Time `json:"recorded_at"`
}

// Profile is the profile/physiology memory section (E2-S2/S3/S8).
type Profile struct {
	Dob           *string        `json:"dob,omitempty"` // YYYY-MM-DD (preferred)
	Age           *int           `json:"age,omitempty"` // fallback when no DOB
	Sex           Sex            `json:"sex"`
	HeightCm      *float64       `json:"height_cm,omitempty"`
	WeightKg      *float64       `json:"weight_kg,omitempty"` // input; appended to history server-side
	WeightHistory []WeightEntry  `json:"weight_history,omitempty"`
	Experience    Experience     `json:"experience"`
	AgingEmphases *AgingEmphases `json:"aging_emphases,omitempty"`
}

// GoalWeights is the goal distribution (E2-S4); persisted normalized to sum 1.
type GoalWeights struct {
	Strength        float64 `json:"strength"`
	Healthspan      float64 `json:"healthspan"`
	BodyComposition float64 `json:"body_composition"`
	Performance     float64 `json:"performance"`
}

// Schedule is the training schedule (E2-S5).
type Schedule struct {
	DaysPerWeek      int      `json:"days_per_week"`
	SessionLengthMin int      `json:"session_length_min"`
	PreferredDays    []string `json:"preferred_days,omitempty"`
	PreferredTimes   []string `json:"preferred_times,omitempty"`
}

// DietPrefs is the dietary-preferences section (E2-S7), read by E11.
type DietPrefs struct {
	Pattern     DietPattern `json:"pattern"`
	Supplements string      `json:"supplements,omitempty"`
	Medications string      `json:"medications,omitempty"`
}

// Preferences captures exercise/equipment preferences (E2-S6). HardAvoids are
// constraints in planning, distinct from soft Dislikes.
type Preferences struct {
	Likes      []string `json:"likes,omitempty"`
	Dislikes   []string `json:"dislikes,omitempty"`
	HardAvoids []string `json:"hard_avoids,omitempty"`
}
