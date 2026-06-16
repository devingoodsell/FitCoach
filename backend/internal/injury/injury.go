// Package injury manages injuries as first-class, structured Coach Memory objects
// (E7). Injuries live in the versioned `injuries` memory section (E3) as a JSON
// document. Active/managed injuries derive contraindications that constrain
// planning (E7-S3) and feed the deterministic safety layer (E7-S4 via E13).
package injury

import (
	"strings"
	"time"

	"pro.d11l.fitcoach/backend/internal/safety"
)

// Status is where an injury is in its lifecycle (E7-S2).
type Status string

const (
	StatusActiveFlare Status = "active_flare"
	StatusManaged     Status = "managed"
	StatusRecurring   Status = "recurring_but_fine"
	StatusResolved    Status = "resolved"
)

func validStatus(s Status) bool {
	switch s {
	case StatusActiveFlare, StatusManaged, StatusRecurring, StatusResolved:
		return true
	}
	return false
}

// constrains reports whether a status injects contraindications into planning.
func (s Status) constrains() bool { return s == StatusActiveFlare || s == StatusManaged }

// Severity is a coarse self-rated severity.
type Severity string

const (
	SeverityMild     Severity = "mild"
	SeverityModerate Severity = "moderate"
	SeveritySevere   Severity = "severe"
)

func validSeverity(s Severity) bool {
	return s == SeverityMild || s == SeverityModerate || s == SeveritySevere
}

// Injury is one structured condition.
type Injury struct {
	ID                   string    `json:"id"`
	Region               string    `json:"region"`
	Status               Status    `json:"status"`
	Severity             Severity  `json:"severity"`
	AggravatingMovements []string  `json:"aggravating_movements,omitempty"`
	OnsetDate            string    `json:"onset_date,omitempty"` // YYYY-MM-DD
	Notes                string    `json:"notes,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// Doc is the JSON shape stored in the `injuries` memory section. ChangedAt marks
// the last lifecycle change so the engine can treat it as a re-plan trigger.
type Doc struct {
	Injuries  []Injury   `json:"injuries"`
	ChangedAt *time.Time `json:"changed_at,omitempty"`
}

func (d Doc) find(id string) int {
	for i, inj := range d.Injuries {
		if inj.ID == id {
			return i
		}
	}
	return -1
}

// Contraindication is the planning-facing constraint derived from an injury.
type Contraindication struct {
	Region         string   `json:"region"`
	Status         Status   `json:"status"`
	AvoidMovements []string `json:"avoid_movements"`
}

// regionAvoidMovements maps a base body region to commonly contraindicated
// movements. Side prefixes (left_/right_) are stripped before lookup.
var regionAvoidMovements = map[string][]string{
	"knee":       {"squat", "lunge", "leg_extension", "leg_press"},
	"lower_back": {"deadlift", "barbell_row", "good_morning", "back_squat"},
	"back":       {"deadlift", "barbell_row", "good_morning"},
	"shoulder":   {"overhead_press", "bench_press", "lateral_raise"},
	"elbow":      {"bench_press", "dip", "curl", "pushup"},
	"wrist":      {"pushup", "front_squat", "clean"},
	"hip":        {"squat", "deadlift", "lunge"},
	"ankle":      {"lunge", "calf_raise", "jump"},
	"neck":       {"overhead_press", "shrug"},
	"hamstring":  {"deadlift", "good_morning", "sprint"},
}

func baseRegion(region string) string {
	r := strings.ToLower(strings.TrimSpace(region))
	r = strings.ReplaceAll(r, " ", "_")
	r = strings.TrimPrefix(r, "left_")
	r = strings.TrimPrefix(r, "right_")
	return r
}

// avoidedMovements returns the union of region-default and injury-specific
// movements to avoid, de-duplicated and stable.
func avoidedMovements(inj Injury) []string {
	seen := map[string]bool{}
	var out []string
	add := func(m string) {
		m = strings.ToLower(strings.TrimSpace(m))
		if m != "" && !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	for _, m := range regionAvoidMovements[baseRegion(inj.Region)] {
		add(m)
	}
	for _, m := range inj.AggravatingMovements {
		add(m)
	}
	return out
}

// Contraindications derives planning constraints from active/managed injuries.
func (d Doc) Contraindications() []Contraindication {
	var out []Contraindication
	for _, inj := range d.Injuries {
		if !inj.Status.constrains() {
			continue
		}
		out = append(out, Contraindication{
			Region:         inj.Region,
			Status:         inj.Status,
			AvoidMovements: avoidedMovements(inj),
		})
	}
	return out
}

// SafetyView builds the safety-layer MemoryView from active/managed injuries, so
// the deterministic contraindication rule (E7-PR5/E13) can validate plans.
func (d Doc) SafetyView() safety.MemoryView {
	var injuries []safety.Injury
	for _, inj := range d.Injuries {
		if !inj.Status.constrains() {
			continue
		}
		injuries = append(injuries, safety.Injury{
			Region:         inj.Region,
			Status:         string(inj.Status),
			AvoidMovements: avoidedMovements(inj),
		})
	}
	return safety.MemoryView{Injuries: injuries}
}
