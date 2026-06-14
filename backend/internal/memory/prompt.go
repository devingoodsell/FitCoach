package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// defaultRecentWorkouts is how many recent sessions assembly includes.
const defaultRecentWorkouts = 10

// CurrentState is the non-memory context for a planning call, supplied by the
// caller (E5). Readiness (E4) and current location/equipment (E9) arrive here as
// those epics land; they are optional and degrade gracefully when absent.
type CurrentState struct {
	RequestedAt     time.Time       `json:"requested_at"`
	Readiness       json.RawMessage `json:"readiness,omitempty"`
	CurrentLocation string          `json:"current_location,omitempty"`
}

// PromptPayload is the deterministic, ordered planning payload assembled from
// Coach Memory plus current state. Field order is fixed, so marshaling the same
// inputs always yields identical bytes. Domain section shapes are opaque here.
type PromptPayload struct {
	RequestedAt     time.Time       `json:"requested_at"`
	Profile         json.RawMessage `json:"profile,omitempty"`
	Goals           json.RawMessage `json:"goals,omitempty"`
	Schedule        json.RawMessage `json:"schedule,omitempty"`
	Preferences     json.RawMessage `json:"preferences,omitempty"`
	Locations       json.RawMessage `json:"locations,omitempty"`
	Injuries        json.RawMessage `json:"injuries,omitempty"`
	Diet            json.RawMessage `json:"diet,omitempty"`
	Readiness       json.RawMessage `json:"readiness,omitempty"`
	CurrentLocation string          `json:"current_location,omitempty"`
	RecentWorkouts  []WorkoutLog    `json:"recent_workouts,omitempty"`
}

// promptStore is the read surface assembly needs (consumer-defined).
type promptStore interface {
	GetAll(ctx context.Context, userID uuid.UUID) ([]SectionRecord, error)
	RecentWorkouts(ctx context.Context, userID uuid.UUID, limit int) ([]WorkoutLog, error)
}

// Assembler builds planning payloads deterministically.
type Assembler struct {
	store       promptStore
	logger      *logging.Logger
	recentLimit int
}

// NewAssembler wires an Assembler.
func NewAssembler(store promptStore, logger *logging.Logger) *Assembler {
	return &Assembler{store: store, logger: logger, recentLimit: defaultRecentWorkouts}
}

// AssemblePrompt gathers profile, goals, schedule, preferences, locations,
// injuries, diet, recent history, and current state into a stable payload. The
// model call is stubbed here (it lands in E5). Assembly is deterministic and logs
// a redacted summary (section presence and counts, never values) for debugging.
func (a *Assembler) AssemblePrompt(ctx context.Context, userID uuid.UUID, state CurrentState) (PromptPayload, error) {
	sections, err := a.store.GetAll(ctx, userID)
	if err != nil {
		return PromptPayload{}, fmt.Errorf("load memory: %w", err)
	}
	bySection := make(map[Section]json.RawMessage, len(sections))
	for _, rec := range sections {
		bySection[rec.Section] = rec.Data
	}

	recent, err := a.store.RecentWorkouts(ctx, userID, a.recentLimit)
	if err != nil {
		return PromptPayload{}, fmt.Errorf("load history: %w", err)
	}

	payload := PromptPayload{
		RequestedAt:     state.RequestedAt,
		Profile:         bySection[SectionProfile],
		Goals:           bySection[SectionGoals],
		Schedule:        bySection[SectionSchedule],
		Preferences:     bySection[SectionPreferences],
		Locations:       bySection[SectionLocations],
		Injuries:        bySection[SectionInjuries],
		Diet:            bySection[SectionDiet],
		Readiness:       state.Readiness,
		CurrentLocation: state.CurrentLocation,
		RecentWorkouts:  recent,
	}

	a.logSummary(ctx, userID, payload)
	return payload, nil
}

// Marshal returns the canonical, deterministic JSON encoding of the payload —
// the stable bytes other epics feed to the model.
func (p PromptPayload) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

// logSummary logs which sections are present and how many workouts were included,
// never the values themselves, so no PII or secrets leak into debug logs.
func (a *Assembler) logSummary(ctx context.Context, userID uuid.UUID, p PromptPayload) {
	if a.logger == nil {
		return
	}
	present := func(raw json.RawMessage) bool { return len(raw) > 0 }
	a.logger.DebugContext(ctx, "prompt assembled",
		"user_id", userID.String(),
		"has_profile", present(p.Profile),
		"has_goals", present(p.Goals),
		"has_schedule", present(p.Schedule),
		"has_preferences", present(p.Preferences),
		"has_locations", present(p.Locations),
		"has_injuries", present(p.Injuries),
		"has_diet", present(p.Diet),
		"has_readiness", present(p.Readiness),
		"recent_workouts", len(p.RecentWorkouts),
	)
}
