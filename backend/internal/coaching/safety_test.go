package coaching

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/location"
	"pro.d11l.fitcoach/backend/internal/platform/events"
	"pro.d11l.fitcoach/backend/internal/safety"
)

// scriptedGenerator returns a different body per call (last body repeats), so we
// can simulate the model fixing — or not fixing — a rejected plan.
type scriptedGenerator struct {
	bodies []string
	calls  int
}

func (g *scriptedGenerator) Generate(_ context.Context, _ GenerationRequest) (GenerationResult, error) {
	i := g.calls
	if i >= len(g.bodies) {
		i = len(g.bodies) - 1
	}
	g.calls++
	return GenerationResult{SessionJSON: []byte(g.bodies[i]), InputTokens: 3, OutputTokens: 4}, nil
}

type fakeEvents struct{ written []events.Event }

func (f *fakeEvents) Write(_ context.Context, e events.Event) error {
	f.written = append(f.written, e)
	return nil
}

// overloadedBody puts an absurd load on the main lift (600kg > DefaultBounds 500).
const overloadedBody = `{
  "warmup": [{"name":"w","movement":"hip_hinge","sets":[{"type":"reps","reps":8,"rest_sec":30}]}],
  "main_work": [{"name":"Bench","movement":"bench_press","region":"chest","sets":[{"type":"reps","reps":8,"load_kg":600,"rest_sec":120}]}],
  "accessory": [{"name":"Row","movement":"row","sets":[{"type":"reps","reps":12,"rest_sec":60}]}],
  "aging_block": {"emphases":["bone_balance"],"items":[{"name":"Hops","movement":"low_amplitude_jump","sets":[{"type":"reps","reps":15,"rest_sec":45}]}]},
  "reasoning": [{"text":"note","tag":"intensity"}]
}`

// cleanBody avoids the contraindicated movement/region (no box_squat, no quad).
const cleanBody = `{
  "warmup": [{"name":"w","movement":"hip_hinge","sets":[{"type":"reps","reps":8,"rest_sec":30}]}],
  "main_work": [{"name":"Bench","movement":"bench_press","region":"chest","sets":[{"type":"reps","reps":8,"load_kg":40,"rest_sec":120}]}],
  "accessory": [{"name":"Row","movement":"row","sets":[{"type":"reps","reps":12,"rest_sec":60}]}],
  "aging_block": {"emphases":["bone_balance"],"items":[{"name":"Hops","movement":"low_amplitude_jump","sets":[{"type":"reps","reps":15,"rest_sec":45}]}]},
  "reasoning": [{"text":"note","tag":"intensity"}]
}`

// kneeView contraindicates the box squat movement and the quad region.
func kneeView() safety.MemoryView {
	return safety.MemoryView{Injuries: []safety.Injury{
		{Region: "quad", Status: "managed", AvoidMovements: []string{"box_squat"}},
	}}
}

func TestEngineCapsOutOfBoundsLoad(t *testing.T) {
	asm := &fakeAssembler{payload: profilePayload(40)}
	gen := &captureGenerator{body: overloadedBody}
	e := newTestEngine(t, asm, gen, nil, location.Doc{})

	s, err := e.Generate(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	// The corrected session is returned with the load clamped to the bound...
	got := s.MainWork[0].Sets[0].LoadKg
	if got == nil || *got != 500 {
		t.Errorf("main load = %v, want capped to 500", got)
	}
	// ...and the correction is recorded as a cap finding (never silently shipped).
	if len(s.SafetyFindings) == 0 || s.SafetyFindings[0].Action != ActionCap {
		t.Errorf("expected a cap finding, got %+v", s.SafetyFindings)
	}
}

// A contraindicated plan is never returned as-is: it forces regeneration, and the
// fixed plan is what ships.
func TestEngineRegeneratesPastContraindication(t *testing.T) {
	asm := &fakeAssembler{payload: profilePayload(40)}
	gen := &scriptedGenerator{bodies: []string{modelBody, cleanBody}} // first contraindicated, then clean
	e := newTestEngineFull(t, asm, gen, fakeInjuries{view: kneeView()}, location.Doc{}, nil)

	s, err := e.Generate(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if gen.calls != 2 {
		t.Errorf("expected a regeneration (2 calls), got %d", gen.calls)
	}
	// The shipped plan contains no contraindicated movement/region.
	for _, ex := range s.MainWork {
		if ex.Movement == "box_squat" || ex.Region == "quad" {
			t.Errorf("contraindicated exercise was returned: %+v", ex)
		}
	}
}

// If the model keeps proposing a contraindicated plan, the engine rejects rather
// than ever returning an unsafe session.
func TestEngineRejectsWhenContraindicationPersists(t *testing.T) {
	asm := &fakeAssembler{payload: profilePayload(40)}
	gen := &captureGenerator{body: modelBody} // always the contraindicated body
	e := newTestEngineFull(t, asm, gen, fakeInjuries{view: kneeView()}, location.Doc{}, nil)

	if _, err := e.Generate(context.Background(), uuid.New()); err != ErrUnsafe {
		t.Fatalf("err = %v, want ErrUnsafe", err)
	}
}

func TestEngineWritesGenerationAndSafetyEvents(t *testing.T) {
	asm := &fakeAssembler{payload: profilePayload(40)}
	ev := &fakeEvents{}
	e := newTestEngineFull(t, asm, &captureGenerator{body: cleanBody}, fakeInjuries{}, location.Doc{}, ev)

	if _, err := e.Generate(context.Background(), uuid.New()); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var gen, saf int
	for _, e := range ev.written {
		switch e.Type {
		case events.TypeGeneration:
			gen++
		case events.TypeSafety:
			saf++
		}
	}
	if gen == 0 || saf == 0 {
		t.Errorf("expected generation and safety events, got gen=%d safety=%d", gen, saf)
	}
}
