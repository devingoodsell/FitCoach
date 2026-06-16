package coaching

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/disclaimer"
	"pro.d11l.fitcoach/backend/internal/injury"
	"pro.d11l.fitcoach/backend/internal/location"
	"pro.d11l.fitcoach/backend/internal/memory"
	"pro.d11l.fitcoach/backend/internal/onboarding"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
	"pro.d11l.fitcoach/backend/internal/readiness"
)

// The engine reads these seams (consumer-defined; the concrete services from
// memory/readiness/injury/location satisfy them). Keeping them small and local
// lets the engine be unit-tested with fakes and never touch a DB or Claude.
type (
	promptAssembler interface {
		AssemblePrompt(ctx context.Context, userID uuid.UUID, state memory.CurrentState) (memory.PromptPayload, error)
	}
	readinessProvider interface {
		Today(ctx context.Context, userID uuid.UUID) (readiness.Score, error)
	}
	injuryProvider interface {
		Contraindications(ctx context.Context, userID uuid.UUID) ([]injury.Contraindication, error)
	}
	locationProvider interface {
		Get(ctx context.Context, userID uuid.UUID) (location.Doc, error)
	}
)

// Engine generates a full session at "Start workout": it assembles Coach Memory +
// today's readiness + active injuries + current equipment + age/aging emphases,
// makes ONE server-side Claude call, and parses a structured session. Safety
// validation (E5-PR4) and re-plan triggers (E5-PR5) layer on top of this.
type Engine struct {
	assembler promptAssembler
	readiness readinessProvider
	injuries  injuryProvider
	locations locationProvider
	generator Generator
	model     string
	logger    *logging.Logger
	now       func() time.Time
	newID     func() string
}

// NewEngine wires an Engine. model labels the session for provenance; now/newID
// default to UTC wall-clock and UUIDv7.
func NewEngine(
	assembler promptAssembler,
	readinessSvc readinessProvider,
	injuries injuryProvider,
	locations locationProvider,
	generator Generator,
	model string,
	logger *logging.Logger,
) *Engine {
	if model == "" {
		model = "claude-opus-4-8"
	}
	return &Engine{
		assembler: assembler,
		readiness: readinessSvc,
		injuries:  injuries,
		locations: locations,
		generator: generator,
		model:     model,
		logger:    logger,
		now:       func() time.Time { return time.Now().UTC() },
		newID:     defaultNewID,
	}
}

// Generate produces one validated-shape session grounded in the user's current
// state. Readiness and current location degrade gracefully when unavailable;
// contraindications are safety-critical, so a failure to read them aborts.
func (e *Engine) Generate(ctx context.Context, userID uuid.UUID) (Session, error) {
	now := e.now()

	score, err := e.readiness.Today(ctx, userID)
	if err != nil {
		e.warn(ctx, "readiness unavailable, using neutral", "error", err.Error())
		score = readiness.Score{Value: 50, Confidence: readiness.ConfidenceLow}
	}

	locName, equipment := e.currentLocation(ctx, userID)

	readinessRaw, err := json.Marshal(score)
	if err != nil {
		return Session{}, fmt.Errorf("marshal readiness: %w", err)
	}
	payload, err := e.assembler.AssemblePrompt(ctx, userID, memory.CurrentState{
		RequestedAt:     now,
		Readiness:       readinessRaw,
		CurrentLocation: locName,
	})
	if err != nil {
		return Session{}, fmt.Errorf("assemble prompt: %w", err)
	}

	contra, err := e.injuries.Contraindications(ctx, userID)
	if err != nil {
		return Session{}, fmt.Errorf("load contraindications: %w", err)
	}

	age, hasAge := profileAge(payload.Profile, now)
	emphases := profileEmphases(payload.Profile, age, hasAge)

	prompt, err := buildPrompt(payload, age, hasAge, emphases, contra, equipment)
	if err != nil {
		return Session{}, fmt.Errorf("build prompt: %w", err)
	}

	res, err := e.generator.Generate(ctx, GenerationRequest{
		System: systemPrompt,
		Prompt: prompt,
		Schema: generationSchema(),
	})
	if err != nil {
		return Session{}, fmt.Errorf("generate session: %w", err)
	}

	var s Session
	if err := json.Unmarshal(res.SessionJSON, &s); err != nil {
		return Session{}, fmt.Errorf("parse session: %w", err)
	}

	// Server-authored fields: the model only fills the workout blocks + reasoning.
	s.ID = e.newID()
	s.GeneratedAt = now
	s.SchemaVersion = SchemaVersion
	s.Model = e.model
	s.Disclaimer = disclaimer.Medical
	s.InputsSummary = &InputsSummary{
		ReadinessValue:        score.Value,
		ReadinessConfidence:   score.Confidence,
		ContraindicationCount: len(contra),
		LocationName:          locName,
		AgingEmphases:         dominantEmphases(emphases),
	}

	if err := s.Validate(); err != nil {
		return Session{}, fmt.Errorf("invalid generated session: %w", err)
	}
	return s, nil
}

// currentLocation resolves the active location's name and equipment, best-effort.
func (e *Engine) currentLocation(ctx context.Context, userID uuid.UUID) (string, []string) {
	doc, err := e.locations.Get(ctx, userID)
	if err != nil {
		e.warn(ctx, "current location unavailable", "error", err.Error())
		return "", nil
	}
	if doc.Current == nil {
		return "", nil
	}
	for _, l := range doc.Locations {
		if l.ID == doc.Current.LocationID {
			return l.Name, l.Equipment
		}
	}
	return "", nil
}

func (e *Engine) warn(ctx context.Context, msg string, args ...any) {
	if e.logger != nil {
		e.logger.WarnContext(ctx, msg, args...)
	}
}

func defaultNewID() string {
	if id, err := uuid.NewV7(); err == nil {
		return id.String()
	}
	return uuid.NewString()
}

func profileAge(raw json.RawMessage, now time.Time) (int, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var p onboarding.Profile
	if err := json.Unmarshal(raw, &p); err != nil {
		return 0, false
	}
	return p.DerivedAge(now)
}

// profileEmphases prefers the user's saved emphases, falling back to the
// age-derived default (E2-S8) so age-appropriate programming still applies when
// the user hasn't tuned them.
func profileEmphases(raw json.RawMessage, age int, hasAge bool) *onboarding.AgingEmphases {
	if len(raw) > 0 {
		var p onboarding.Profile
		if err := json.Unmarshal(raw, &p); err == nil && p.AgingEmphases != nil {
			return p.AgingEmphases
		}
	}
	if hasAge {
		d := onboarding.DefaultAgingEmphases(age)
		return &d
	}
	return nil
}

// dominantEmphases lists the emphasis keys by descending weight (audit summary).
func dominantEmphases(e *onboarding.AgingEmphases) []string {
	if e == nil {
		return nil
	}
	type kv struct {
		key    string
		weight float64
	}
	all := []kv{
		{"bone_balance", e.BoneBalance},
		{"joint_tendon", e.JointTendon},
		{"vo2max", e.Vo2Max},
		{"cardio_base", e.CardioBase},
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].weight > all[j].weight })
	out := make([]string, 0, len(all))
	for _, e := range all {
		if e.weight > 0 {
			out = append(out, e.key)
		}
	}
	return out
}
