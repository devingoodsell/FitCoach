package coaching

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// sampleSessionPath is the committed fixed sample the Android session-preview UI
// and E6 build against. Keeping the strict decode here means the contract, the
// Go types, and the sample can't drift apart silently.
var sampleSessionPath = filepath.Join("..", "..", "api", "examples", "session-sample.json")

func loadSampleSession(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(sampleSessionPath)
	if err != nil {
		t.Fatalf("read sample session: %v", err)
	}
	return data
}

func decodeStrict(t *testing.T, data []byte) Session {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields() // any key not on the Go types fails — catches schema drift
	var s Session
	if err := dec.Decode(&s); err != nil {
		t.Fatalf("strict decode sample session: %v", err)
	}
	return s
}

func TestSampleSessionDecodesStrictlyAndValidates(t *testing.T) {
	s := decodeStrict(t, loadSampleSession(t))
	if err := s.Validate(); err != nil {
		t.Fatalf("sample session failed Validate: %v", err)
	}
}

func TestSampleSessionShapeInvariants(t *testing.T) {
	s := decodeStrict(t, loadSampleSession(t))

	if s.ID == "" || s.GeneratedAt.IsZero() {
		t.Errorf("expected id and generated_at to be set")
	}
	if s.SchemaVersion != SchemaVersion {
		t.Errorf("schema_version = %d, want %d", s.SchemaVersion, SchemaVersion)
	}
	if s.Model == "" {
		t.Errorf("expected model id to be recorded for provenance")
	}
	// Aging block is present in EVERY session and declares its emphases (E8-S1/S2).
	if len(s.AgingBlock.Items) == 0 || len(s.AgingBlock.Emphases) == 0 {
		t.Errorf("aging_block must carry items and emphases")
	}
	// Reasoning surfaces an age-aware choice (E8-S2)...
	if !hasReasoningTag(s, TagAgeAware) {
		t.Errorf("expected an age-aware reasoning note")
	}
	// ...and the medical disclaimer appears wherever the body is involved (E13).
	if s.Disclaimer == "" {
		t.Errorf("expected the medical disclaimer to be present")
	}
	// A delivered session only carries cap findings (blocks force regenerate/reject).
	for _, f := range s.SafetyFindings {
		if f.Action == ActionBlock {
			t.Errorf("delivered session must not carry a block finding: %v", f)
		}
	}

	// Per-set prescriptions: main work ramps load across sets and carries kg;
	// the bodyweight aging move omits load_kg; a time-based warmup uses duration.
	first := s.MainWork[0]
	if len(first.Sets) < 2 || first.Sets[0].LoadKg == nil {
		t.Errorf("expected main lift to carry multiple loaded sets")
	}
	if s.AgingBlock.Items[0].Sets[0].LoadKg != nil {
		t.Errorf("expected the aging-block move to be bodyweight (nil load_kg)")
	}
	if !hasTimedSet(s.Warmup) {
		t.Errorf("expected a time-based warmup set exercising the time set type")
	}
}

func TestSessionRoundTripsStable(t *testing.T) {
	s := decodeStrict(t, loadSampleSession(t))
	out, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal session: %v", err)
	}
	var again Session
	if err := json.Unmarshal(out, &again); err != nil {
		t.Fatalf("re-decode session: %v", err)
	}
	if err := again.Validate(); err != nil {
		t.Fatalf("round-tripped session failed Validate: %v", err)
	}
}

func hasReasoningTag(s Session, tag ReasoningTag) bool {
	for _, n := range s.Reasoning {
		if n.Tag == tag {
			return true
		}
	}
	return false
}

func hasTimedSet(exercises []Exercise) bool {
	for _, e := range exercises {
		for _, set := range e.Sets {
			if set.Type == SetTime {
				return true
			}
		}
	}
	return false
}
