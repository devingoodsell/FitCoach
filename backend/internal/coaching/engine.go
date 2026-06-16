package coaching

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/disclaimer"
	"pro.d11l.fitcoach/backend/internal/injury"
	"pro.d11l.fitcoach/backend/internal/location"
	"pro.d11l.fitcoach/backend/internal/memory"
	"pro.d11l.fitcoach/backend/internal/onboarding"
	"pro.d11l.fitcoach/backend/internal/platform/events"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
	"pro.d11l.fitcoach/backend/internal/readiness"
	"pro.d11l.fitcoach/backend/internal/safety"
)

// maxRegens is how many times the engine re-asks the model after the safety layer
// rejects a plan before giving up. One retry is enough in practice; the cap stops
// a pathological model from looping.
const maxRegens = 1

// ErrUnsafe is returned when no safe session can be produced (e.g. the model keeps
// proposing a contraindicated plan). The handler maps it to 422.
var ErrUnsafe = errors.New("could not produce a safe session")

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
		SafetyView(ctx context.Context, userID uuid.UUID) (safety.MemoryView, error)
	}
	locationProvider interface {
		Get(ctx context.Context, userID uuid.UUID) (location.Doc, error)
	}
	// eventWriter records redacted generation/safety audit events (E15-S3).
	eventWriter interface {
		Write(ctx context.Context, e events.Event) error
	}
)

// Engine generates a full session at "Start workout": it assembles Coach Memory +
// today's readiness + active injuries + current equipment + age/aging emphases,
// makes ONE server-side Claude call, runs the result through the deterministic
// safety layer (correct or regenerate), and only returns a validated session.
type Engine struct {
	assembler promptAssembler
	readiness readinessProvider
	injuries  injuryProvider
	locations locationProvider
	generator Generator
	validator *safety.Validator
	events    eventWriter
	model     string
	logger    *logging.Logger
	now       func() time.Time
	newID     func() string
}

// NewEngine wires an Engine. The safety validator is the mandatory single
// entrypoint between the model and the user (bounds + contraindication rules).
// events may be nil. model labels the session for provenance; now/newID default
// to UTC wall-clock and UUIDv7.
func NewEngine(
	assembler promptAssembler,
	readinessSvc readinessProvider,
	injuries injuryProvider,
	locations locationProvider,
	generator Generator,
	eventsWriter eventWriter,
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
		validator: safety.NewValidator(logger, safety.NewBoundsRule(safety.DefaultBounds()), safety.NewContraindicationRule()),
		events:    eventsWriter,
		model:     model,
		logger:    logger,
		now:       func() time.Time { return time.Now().UTC() },
		newID:     defaultNewID,
	}
}

// Generate produces one safety-validated session grounded in the user's current
// state. Readiness and current location degrade gracefully when unavailable;
// contraindications are safety-critical, so a failure to read them aborts. Every
// generated plan passes through the safety layer before return: caps are applied
// in place, a contraindicated plan is regenerated, and an unfixable plan is
// rejected — an unvalidated plan is never returned.
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
	memView, err := e.injuries.SafetyView(ctx, userID)
	if err != nil {
		return Session{}, fmt.Errorf("load safety view: %w", err)
	}

	age, hasAge := profileAge(payload.Profile, now)
	emphases := profileEmphases(payload.Profile, age, hasAge)
	inputs := &InputsSummary{
		ReadinessValue:        score.Value,
		ReadinessConfidence:   score.Confidence,
		ContraindicationCount: len(contra),
		LocationName:          locName,
		AgingEmphases:         dominantEmphases(emphases),
	}

	var feedback string
	for attempt := 0; attempt <= maxRegens; attempt++ {
		prompt, err := buildPrompt(payload, age, hasAge, emphases, contra, equipment, feedback)
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

		s, err := e.stamp(res, now, inputs)
		if err != nil {
			// Structurally malformed output — re-ask with feedback, don't ship.
			feedback = "Previous output was not a valid session: " + err.Error()
			continue
		}

		plan, refs := collectExercises(&s)
		result := e.validator.Validate(plan, memView)
		e.logSafety(ctx, userID, result)

		switch result.Outcome {
		case safety.Pass:
			e.logGeneration(ctx, userID, res, string(result.Outcome), attempt)
			return s, nil
		case safety.Corrected:
			applyCaps(refs, result.Plan)
			if err := s.Validate(); err != nil {
				// A correction emptied a required block — regenerate smaller.
				feedback = "Previous plan exceeded safe volume/load bounds; reduce them."
				continue
			}
			s.SafetyFindings = toFindings(result.Findings)
			e.logGeneration(ctx, userID, res, string(result.Outcome), attempt)
			return s, nil
		case safety.Regenerate:
			feedback = describeFindings(result.Findings)
		}
	}

	e.logGeneration(ctx, userID, GenerationResult{}, "rejected", maxRegens)
	return Session{}, ErrUnsafe
}

// stamp unmarshals the model output, fills server-authored fields, and applies the
// structural guard.
func (e *Engine) stamp(res GenerationResult, now time.Time, inputs *InputsSummary) (Session, error) {
	var s Session
	if err := json.Unmarshal(res.SessionJSON, &s); err != nil {
		return Session{}, fmt.Errorf("parse session: %w", err)
	}
	s.ID = e.newID()
	s.GeneratedAt = now
	s.SchemaVersion = SchemaVersion
	s.Model = e.model
	s.Disclaimer = disclaimer.Medical
	s.InputsSummary = inputs
	if err := s.Validate(); err != nil {
		return Session{}, err
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

// logGeneration records a redacted generation audit event (E15-S3).
func (e *Engine) logGeneration(ctx context.Context, userID uuid.UUID, res GenerationResult, outcome string, attempt int) {
	if e.events == nil {
		return
	}
	payload := map[string]any{
		"model":         e.model,
		"outcome":       outcome,
		"attempt":       attempt,
		"input_tokens":  res.InputTokens,
		"output_tokens": res.OutputTokens,
	}
	if err := e.events.Write(ctx, events.Event{Type: events.TypeGeneration, UserID: userID, Payload: payload}); err != nil {
		e.warn(ctx, "write generation event failed", "error", err.Error())
	}
}

// logSafety records the safety verdict and findings (redacted) for audit.
func (e *Engine) logSafety(ctx context.Context, userID uuid.UUID, result safety.Result) {
	if e.events == nil {
		return
	}
	payload := map[string]any{
		"outcome":  string(result.Outcome),
		"findings": result.Findings,
	}
	if err := e.events.Write(ctx, events.Event{Type: events.TypeSafety, UserID: userID, Payload: payload}); err != nil {
		e.warn(ctx, "write safety event failed", "error", err.Error())
	}
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
