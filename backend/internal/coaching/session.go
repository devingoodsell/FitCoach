// Package coaching is the session-generation engine (E5/E8) and the ONLY package
// that may call Claude. It assembles Coach Memory + today's readiness + active
// injuries + current equipment into one server-side model call, parses a
// structured session, runs it through the deterministic safety layer, and only
// then returns it. The Anthropic API key is read server-side from config and
// never reaches the client.
//
// This file defines the canonical session shape — the stable, versioned contract
// the Android session-preview UI (E5-S3) and the in-session experience (E6) build
// against. It mirrors the Session schema in backend/api/openapi.yaml and the
// committed sample in backend/api/examples/session-sample.json exactly.
package coaching

import (
	"fmt"
	"time"
)

// SchemaVersion is the current generated-session schema version. Bump it when the
// session shape changes so older cached clients can detect and refetch.
const SchemaVersion = 1

// Session is a full generated workout: warmup, main work, accessory, an
// always-present healthspan block (E8-S1), and brief plain-language reasoning
// (E5-S2). Every Session returned to the client has already passed the
// deterministic safety layer.
type Session struct {
	ID             string          `json:"id"`
	GeneratedAt    time.Time       `json:"generated_at"`
	SchemaVersion  int             `json:"schema_version"`
	Model          string          `json:"model,omitempty"`
	InputsSummary  *InputsSummary  `json:"inputs_summary,omitempty"`
	Warmup         []Exercise      `json:"warmup"`
	MainWork       []Exercise      `json:"main_work"`
	Accessory      []Exercise      `json:"accessory"`
	AgingBlock     AgingBlock      `json:"aging_block"`
	Reasoning      []ReasoningNote `json:"reasoning"`
	SafetyFindings []SafetyFinding `json:"safety_findings,omitempty"`
	Disclaimer     string          `json:"disclaimer"`
}

// InputsSummary is a redacted snapshot of what fed generation — an audit aid and
// enough for the client to show "planned for readiness 72 (high)" without a
// second call. It never carries raw memory values.
type InputsSummary struct {
	ReadinessValue        int      `json:"readiness_value,omitempty"`
	ReadinessConfidence   string   `json:"readiness_confidence,omitempty"`
	ContraindicationCount int      `json:"contraindication_count,omitempty"`
	LocationName          string   `json:"location_name,omitempty"`
	AgingEmphases         []string `json:"aging_emphases,omitempty"`
}

// Exercise is one prescribed movement carrying one or more sets. Modeling sets as
// a list makes ramping loads, per-set rest, and time/distance work first-class.
// Name/Movement/Region map onto safety.Exercise for contraindication matching.
type Exercise struct {
	Name     string            `json:"name"`
	Movement string            `json:"movement"` // canonical key for contraindication matching (E7)
	Region   string            `json:"region,omitempty"`
	Sets     []SetPrescription `json:"sets"`
	Notes    string            `json:"notes,omitempty"`
}

// SetType is how a set is measured.
type SetType string

const (
	SetReps     SetType = "reps"
	SetTime     SetType = "time"
	SetDistance SetType = "distance"
)

// SetPrescription is one prescribed set. Loads are canonical kilograms (load_kg);
// the client converts to the user's preferred unit for display. RestSec is the
// explicit rest after this set (E5-S1); RPETarget drives on-device
// autoregulation (E5-S4).
type SetPrescription struct {
	Type        SetType  `json:"type"`
	Reps        int      `json:"reps,omitempty"`
	LoadKg      *float64 `json:"load_kg,omitempty"` // canonical kg; nil = bodyweight
	RPETarget   *float64 `json:"rpe_target,omitempty"`
	DurationSec int      `json:"duration_sec,omitempty"`
	RestSec     int      `json:"rest_sec,omitempty"`
}

// AgingEmphasis is a healthspan focus (E8). Mirrors onboarding.AgingEmphases keys.
type AgingEmphasis string

const (
	EmphasisBoneBalance AgingEmphasis = "bone_balance"
	EmphasisJointTendon AgingEmphasis = "joint_tendon"
	EmphasisVO2Max      AgingEmphasis = "vo2max"
	EmphasisCardioBase  AgingEmphasis = "cardio_base"
)

// AgingBlock is the healthspan block required in every session (E8-S1). Emphases
// records which aging focuses drove it (E8-S2).
type AgingBlock struct {
	Emphases []AgingEmphasis `json:"emphases"`
	Items    []Exercise      `json:"items"`
}

// ReasoningTag classifies a reasoning note so the UI can group/surface choices.
type ReasoningTag string

const (
	TagIntensity           ReasoningTag = "intensity"
	TagExerciseChoice      ReasoningTag = "exercise_choice"
	TagInjuryAccommodation ReasoningTag = "injury_accommodation"
	TagAgeAware            ReasoningTag = "age_aware"
	TagRecovery            ReasoningTag = "recovery"
)

// ReasoningNote is one brief plain-language explanation, no medical claims
// (E5-S2). A note tagged age_aware satisfies E8-S2 when relevant.
type ReasoningNote struct {
	Text string       `json:"text"`
	Tag  ReasoningTag `json:"tag,omitempty"`
}

// Safety actions mirror safety.Action. A delivered session only ever carries cap
// findings — a block forces regeneration or rejection server-side.
const (
	ActionCap   = "cap"
	ActionBlock = "block"
)

// SafetyFinding mirrors safety.Finding for transport/audit (redacted detail).
type SafetyFinding struct {
	Rule   string `json:"rule"`
	Action string `json:"action"` // cap | block
	Detail string `json:"detail,omitempty"`
}

// Validate checks the structural invariants every generated session must hold
// before it reaches the client: a current schema version, every block populated,
// an aging block with emphases (E8-S1), reasoning and disclaimer present, and
// each exercise/set well-formed. It is the parse-time guard the engine (E5-PR3)
// applies to model output and is unit tested against the committed sample so the
// contract can't silently drift.
func (s Session) Validate() error {
	if s.SchemaVersion != SchemaVersion {
		return fmt.Errorf("schema_version = %d, want %d", s.SchemaVersion, SchemaVersion)
	}
	blocks := []struct {
		name string
		ex   []Exercise
	}{
		{"warmup", s.Warmup},
		{"main_work", s.MainWork},
		{"accessory", s.Accessory},
		{"aging_block.items", s.AgingBlock.Items},
	}
	for _, b := range blocks {
		if len(b.ex) == 0 {
			return fmt.Errorf("session %q block is empty", b.name)
		}
		for i, e := range b.ex {
			if err := e.validate(); err != nil {
				return fmt.Errorf("%s[%d]: %w", b.name, i, err)
			}
		}
	}
	if len(s.AgingBlock.Emphases) == 0 {
		return fmt.Errorf("aging_block must declare its emphases (E8-S1)")
	}
	if len(s.Reasoning) == 0 {
		return fmt.Errorf("session reasoning is required")
	}
	if s.Disclaimer == "" {
		return fmt.Errorf("session disclaimer is required")
	}
	return nil
}

func (e Exercise) validate() error {
	if e.Name == "" {
		return fmt.Errorf("name is required")
	}
	if e.Movement == "" {
		return fmt.Errorf("movement is required")
	}
	if len(e.Sets) == 0 {
		return fmt.Errorf("at least one set is required")
	}
	for i, set := range e.Sets {
		if err := set.validate(); err != nil {
			return fmt.Errorf("sets[%d]: %w", i, err)
		}
	}
	return nil
}

func (p SetPrescription) validate() error {
	switch p.Type {
	case SetReps:
		if p.Reps < 1 {
			return fmt.Errorf("reps set must have reps >= 1")
		}
	case SetTime:
		if p.DurationSec < 1 {
			return fmt.Errorf("time set must have duration_sec >= 1")
		}
	case SetDistance:
		// distance magnitude is carried in notes/movement for now
	default:
		return fmt.Errorf("invalid set type %q", p.Type)
	}
	if p.RestSec < 0 {
		return fmt.Errorf("rest_sec must be >= 0")
	}
	return nil
}
