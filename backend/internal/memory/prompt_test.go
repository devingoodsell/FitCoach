package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// promptFakeStore returns fixed data for deterministic assembly tests.
type promptFakeStore struct {
	sections []SectionRecord
	workouts []WorkoutLog
}

func (f *promptFakeStore) GetAll(context.Context, uuid.UUID) ([]SectionRecord, error) {
	return f.sections, nil
}
func (f *promptFakeStore) RecentWorkouts(context.Context, uuid.UUID, int) ([]WorkoutLog, error) {
	return f.workouts, nil
}

func fixedState() CurrentState {
	return CurrentState{
		RequestedAt:     time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC),
		CurrentLocation: "home gym",
	}
}

func TestAssembleIsDeterministic(t *testing.T) {
	store := &promptFakeStore{
		sections: []SectionRecord{
			{Section: SectionProfile, Version: 1, Data: json.RawMessage(`{"age":40}`)},
			{Section: SectionGoals, Version: 1, Data: json.RawMessage(`{"strength":0.6}`)},
		},
		workouts: []WorkoutLog{{ClientSessionID: "s1", Version: 1, Data: json.RawMessage(`{"sets":5}`)}},
	}
	a := NewAssembler(store, logging.New(bytes.NewBuffer(nil), "debug"))

	p1, err := a.AssemblePrompt(context.Background(), uuid.Nil, fixedState())
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	p2, _ := a.AssemblePrompt(context.Background(), uuid.Nil, fixedState())

	b1, _ := p1.Marshal()
	b2, _ := p2.Marshal()
	if !bytes.Equal(b1, b2) {
		t.Fatalf("assembly not deterministic:\n%s\n%s", b1, b2)
	}
	// Sanity: known fields present in stable order.
	if !strings.Contains(string(b1), `"profile":{"age":40}`) {
		t.Errorf("payload missing profile: %s", b1)
	}
}

func TestAssembleDegradesGracefullyWithMissingSections(t *testing.T) {
	store := &promptFakeStore{
		sections: []SectionRecord{
			{Section: SectionProfile, Version: 1, Data: json.RawMessage(`{"age":40}`)},
		},
	}
	a := NewAssembler(store, logging.New(bytes.NewBuffer(nil), "debug"))

	p, err := a.AssemblePrompt(context.Background(), uuid.Nil, fixedState())
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}
	if len(p.Goals) != 0 || len(p.Diet) != 0 {
		t.Errorf("missing sections should be empty, got goals=%s diet=%s", p.Goals, p.Diet)
	}
	// Omitted sections do not appear in the marshaled payload.
	b, _ := p.Marshal()
	if strings.Contains(string(b), `"goals"`) {
		t.Errorf("empty goals should be omitted: %s", b)
	}
}

func TestAssembleDebugLogDoesNotLeakContent(t *testing.T) {
	var buf bytes.Buffer
	store := &promptFakeStore{
		sections: []SectionRecord{
			// Place identifiable PII in the section data.
			{Section: SectionProfile, Version: 1, Data: json.RawMessage(`{"name":"Top Secret Name","age":40}`)},
		},
	}
	a := NewAssembler(store, logging.New(&buf, "debug"))

	if _, err := a.AssemblePrompt(context.Background(), uuid.Nil, fixedState()); err != nil {
		t.Fatalf("assemble: %v", err)
	}
	out := buf.String()
	if out == "" {
		t.Fatal("expected a debug log line")
	}
	if strings.Contains(out, "Top Secret Name") || strings.Contains(out, "age") {
		t.Errorf("debug log leaked section content: %s", out)
	}
	// It should log presence, not values.
	if !strings.Contains(out, "has_profile") {
		t.Errorf("debug log missing summary fields: %s", out)
	}
}
