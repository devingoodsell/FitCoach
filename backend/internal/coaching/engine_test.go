package coaching

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/disclaimer"
	"pro.d11l.fitcoach/backend/internal/injury"
	"pro.d11l.fitcoach/backend/internal/location"
	"pro.d11l.fitcoach/backend/internal/memory"
	"pro.d11l.fitcoach/backend/internal/readiness"
	"pro.d11l.fitcoach/backend/internal/safety"
)

// --- fakes -------------------------------------------------------------------

type fakeAssembler struct {
	payload  memory.PromptPayload
	gotState memory.CurrentState
}

func (f *fakeAssembler) AssemblePrompt(_ context.Context, _ uuid.UUID, state memory.CurrentState) (memory.PromptPayload, error) {
	f.gotState = state
	p := f.payload
	p.RequestedAt = state.RequestedAt
	p.Readiness = state.Readiness
	p.CurrentLocation = state.CurrentLocation
	return p, nil
}

type fakeReadiness struct{ score readiness.Score }

func (f fakeReadiness) Today(context.Context, uuid.UUID) (readiness.Score, error) {
	return f.score, nil
}

type fakeInjuries struct {
	contra []injury.Contraindication
	view   safety.MemoryView
}

func (f fakeInjuries) Contraindications(context.Context, uuid.UUID) ([]injury.Contraindication, error) {
	return f.contra, nil
}

func (f fakeInjuries) SafetyView(context.Context, uuid.UUID) (safety.MemoryView, error) {
	return f.view, nil
}

type fakeLocations struct{ doc location.Doc }

func (f fakeLocations) Get(context.Context, uuid.UUID) (location.Doc, error) { return f.doc, nil }

// captureGenerator records the request and returns a preset model-authored body.
type captureGenerator struct {
	got  GenerationRequest
	body string
	err  error
}

func (g *captureGenerator) Generate(_ context.Context, req GenerationRequest) (GenerationResult, error) {
	g.got = req
	if g.err != nil {
		return GenerationResult{}, g.err
	}
	return GenerationResult{SessionJSON: []byte(g.body), InputTokens: 5, OutputTokens: 9}, nil
}

// modelBody is a valid MODEL-authored session slice (no server fields).
const modelBody = `{
  "warmup": [{"name":"Cat-cow","movement":"spinal_flexion_extension","sets":[{"type":"reps","reps":8,"rest_sec":0}]}],
  "main_work": [{"name":"Goblet box squat","movement":"box_squat","region":"quad","sets":[{"type":"reps","reps":8,"load_kg":24,"rpe_target":7,"rest_sec":120}]}],
  "accessory": [{"name":"Row","movement":"row","sets":[{"type":"reps","reps":12,"rest_sec":60}]}],
  "aging_block": {"emphases":["bone_balance"],"items":[{"name":"Pogo hops","movement":"low_amplitude_jump","sets":[{"type":"reps","reps":15,"rest_sec":45}]}]},
  "reasoning": [{"text":"Held RPE 7 with full rest.","tag":"intensity"},{"text":"At 45, added bone-loading hops.","tag":"age_aware"}]
}`

func profilePayload(age int) memory.PromptPayload {
	return memory.PromptPayload{Profile: json.RawMessage(`{"age":` + itoa(age) + `}`)}
}

func itoa(n int) string       { return strings.TrimSpace(jsonNumber(n)) }
func jsonNumber(n int) string { b, _ := json.Marshal(n); return string(b) }

func newTestEngine(t *testing.T, asm *fakeAssembler, gen Generator, contra []injury.Contraindication, loc location.Doc) *Engine {
	t.Helper()
	return newTestEngineFull(t, asm, gen, fakeInjuries{contra: contra}, loc, nil)
}

func newTestEngineFull(t *testing.T, asm *fakeAssembler, gen Generator, inj fakeInjuries, loc location.Doc, ev eventWriter) *Engine {
	t.Helper()
	e := NewEngine(asm, fakeReadiness{score: readiness.Score{Value: 72, Confidence: readiness.ConfidenceHigh}},
		inj, fakeLocations{doc: loc}, gen, ev, "claude-opus-4-8", nil)
	e.now = func() time.Time { return time.Date(2026, 6, 16, 13, 0, 0, 0, time.UTC) }
	e.newID = func() string { return "sess-fixed" }
	return e
}

// --- tests -------------------------------------------------------------------

func TestEngineGeneratesValidSessionWithServerFields(t *testing.T) {
	asm := &fakeAssembler{payload: profilePayload(45)}
	gen := &captureGenerator{body: modelBody}
	loc := location.Doc{
		Locations: []location.Location{{ID: "loc1", Name: "Home gym", Equipment: []string{"dumbbells", "bench"}}},
		Current:   &location.CurrentContext{LocationID: "loc1"},
	}
	e := newTestEngine(t, asm, gen, nil, loc)

	s, err := e.Generate(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Server-authored fields are filled and the session is valid.
	if s.ID != "sess-fixed" || s.SchemaVersion != SchemaVersion || s.Model != "claude-opus-4-8" {
		t.Errorf("server fields not set: %+v", s)
	}
	if s.GeneratedAt.IsZero() {
		t.Errorf("generated_at not set")
	}
	if s.Disclaimer != disclaimer.Medical {
		t.Errorf("disclaimer = %q, want the central medical disclaimer", s.Disclaimer)
	}
	if err := s.Validate(); err != nil {
		t.Errorf("generated session invalid: %v", err)
	}
	// Aging block present (E8-S1) and an age-aware reasoning note (E8-S2).
	if len(s.AgingBlock.Items) == 0 {
		t.Errorf("expected an aging block")
	}
	if !hasReasoningTag(s, TagAgeAware) {
		t.Errorf("expected an age-aware reasoning note")
	}
	// Inputs summary reflects what fed generation.
	if s.InputsSummary == nil || s.InputsSummary.ReadinessValue != 72 || s.InputsSummary.LocationName != "Home gym" {
		t.Errorf("inputs_summary = %+v", s.InputsSummary)
	}
}

func TestEnginePromptCarriesContraindicationsEquipmentAndAge(t *testing.T) {
	asm := &fakeAssembler{payload: profilePayload(68)}
	gen := &captureGenerator{body: modelBody}
	contra := []injury.Contraindication{{Region: "left_knee", Status: injury.StatusManaged, AvoidMovements: []string{"deep_squat"}}}
	loc := location.Doc{
		Locations: []location.Location{{ID: "g", Name: "Hotel gym", Equipment: []string{"treadmill", "kettlebell"}}},
		Current:   &location.CurrentContext{LocationID: "g"},
	}
	e := newTestEngine(t, asm, gen, contra, loc)

	if _, err := e.Generate(context.Background(), uuid.New()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	prompt := string(gen.got.Prompt)
	for _, want := range []string{"left_knee", "deep_squat", "kettlebell", "\"age\":68", "bone_balance"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q\nprompt: %s", want, prompt)
		}
	}
	// The structured-output schema is attached.
	if gen.got.Schema == nil {
		t.Errorf("expected a structured-output schema on the request")
	}
}

// E8-S2: older vs younger profiles feed measurably different aging emphases into
// generation (older skews bone/balance, younger skews vo2max).
func TestEngineAgeDrivesDifferentEmphasisInputs(t *testing.T) {
	run := func(age int) string {
		asm := &fakeAssembler{payload: profilePayload(age)}
		gen := &captureGenerator{body: modelBody}
		e := newTestEngine(t, asm, gen, nil, location.Doc{})
		if _, err := e.Generate(context.Background(), uuid.New()); err != nil {
			t.Fatalf("Generate(age=%d): %v", age, err)
		}
		return string(gen.got.Prompt)
	}
	young, old := run(30), run(68)
	if young == old {
		t.Fatalf("expected different prompts for young vs old profiles")
	}

	var dy, do promptInput
	mustUnmarshalDerived(t, young, &dy)
	mustUnmarshalDerived(t, old, &do)
	// Older user weights bone_balance more heavily than the younger user.
	if !(do.Derived.AgingEmphases.BoneBalance > dy.Derived.AgingEmphases.BoneBalance) {
		t.Errorf("expected older bone_balance (%v) > younger (%v)",
			do.Derived.AgingEmphases.BoneBalance, dy.Derived.AgingEmphases.BoneBalance)
	}
}

func TestEngineRejectsSessionMissingAgingBlock(t *testing.T) {
	// Model omits the aging block — Validate must reject it, never return it.
	bad := `{"warmup":[{"name":"w","movement":"m","sets":[{"type":"reps","reps":5,"rest_sec":0}]}],
	         "main_work":[{"name":"x","movement":"m","sets":[{"type":"reps","reps":5,"rest_sec":0}]}],
	         "accessory":[{"name":"y","movement":"m","sets":[{"type":"reps","reps":5,"rest_sec":0}]}],
	         "aging_block":{"emphases":[],"items":[]},
	         "reasoning":[{"text":"note"}]}`
	asm := &fakeAssembler{payload: profilePayload(40)}
	e := newTestEngine(t, asm, &captureGenerator{body: bad}, nil, location.Doc{})

	if _, err := e.Generate(context.Background(), uuid.New()); err == nil {
		t.Fatalf("expected an error for a session with an empty aging block")
	}
}

func TestGenerationSchemaIsWellFormed(t *testing.T) {
	sch := generationSchema()
	raw, err := json.Marshal(sch)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	for _, want := range []string{"warmup", "main_work", "accessory", "aging_block", "reasoning", "additionalProperties"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("schema missing %q", want)
		}
	}
}

func mustUnmarshalDerived(t *testing.T, prompt string, out *promptInput) {
	t.Helper()
	if err := json.Unmarshal([]byte(prompt), out); err != nil {
		t.Fatalf("unmarshal prompt: %v", err)
	}
	if out.Derived.AgingEmphases == nil {
		t.Fatalf("expected aging emphases in derived context")
	}
}
