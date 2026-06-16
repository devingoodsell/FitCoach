package coaching

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/injury"
	"pro.d11l.fitcoach/backend/internal/location"
	"pro.d11l.fitcoach/backend/internal/readiness"
)

type fakeInjuryDoc struct {
	doc injury.Doc
	err error
}

func (f fakeInjuryDoc) Get(context.Context, uuid.UUID) (injury.Doc, error) { return f.doc, f.err }

var replanSince = time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)

func ptr(t time.Time) *time.Time { return &t }

func goodReadiness() fakeReadiness {
	return fakeReadiness{score: readiness.Score{Value: 72, Confidence: readiness.ConfidenceHigh}}
}

func TestReplanNoTriggers(t *testing.T) {
	r := NewReplanner(
		fakeInjuryDoc{doc: injury.Doc{ChangedAt: ptr(replanSince.Add(-time.Hour))}},
		fakeLocations{doc: location.Doc{Current: &location.CurrentContext{ChangedAt: replanSince.Add(-time.Hour)}}},
		goodReadiness(), nil)

	dec, err := r.Check(context.Background(), uuid.New(), replanSince)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if dec.ReplanNeeded || len(dec.Reasons) != 0 {
		t.Errorf("expected no re-plan, got %+v", dec)
	}
}

func TestReplanInjuryChanged(t *testing.T) {
	r := NewReplanner(
		fakeInjuryDoc{doc: injury.Doc{ChangedAt: ptr(replanSince.Add(time.Hour))}},
		fakeLocations{}, goodReadiness(), nil)

	dec, _ := r.Check(context.Background(), uuid.New(), replanSince)
	if !dec.ReplanNeeded || !hasReason(dec, ReasonInjuryChanged) {
		t.Errorf("expected injury_changed, got %+v", dec)
	}
}

func TestReplanContextChanged(t *testing.T) {
	r := NewReplanner(
		fakeInjuryDoc{},
		fakeLocations{doc: location.Doc{Current: &location.CurrentContext{ChangedAt: replanSince.Add(time.Hour)}}},
		goodReadiness(), nil)

	dec, _ := r.Check(context.Background(), uuid.New(), replanSince)
	if !dec.ReplanNeeded || !hasReason(dec, ReasonContextChanged) {
		t.Errorf("expected context_changed, got %+v", dec)
	}
}

func TestReplanPoorRecovery(t *testing.T) {
	r := NewReplanner(
		fakeInjuryDoc{},
		fakeLocations{},
		fakeReadiness{score: readiness.Score{Value: 30, Confidence: readiness.ConfidenceHigh}}, nil)

	dec, _ := r.Check(context.Background(), uuid.New(), replanSince)
	if !dec.ReplanNeeded || !hasReason(dec, ReasonPoorRecovery) {
		t.Errorf("expected poor_recovery, got %+v", dec)
	}
}

func TestReplanAllTriggers(t *testing.T) {
	r := NewReplanner(
		fakeInjuryDoc{doc: injury.Doc{ChangedAt: ptr(replanSince.Add(time.Hour))}},
		fakeLocations{doc: location.Doc{Current: &location.CurrentContext{ChangedAt: replanSince.Add(time.Hour)}}},
		fakeReadiness{score: readiness.Score{Value: 25, Confidence: readiness.ConfidenceMedium}}, nil)

	dec, _ := r.Check(context.Background(), uuid.New(), replanSince)
	want := []string{ReasonInjuryChanged, ReasonContextChanged, ReasonPoorRecovery}
	if len(dec.Reasons) != len(want) {
		t.Fatalf("reasons = %v, want %v", dec.Reasons, want)
	}
	for i := range want {
		if dec.Reasons[i] != want[i] {
			t.Errorf("reasons[%d] = %q, want %q", i, dec.Reasons[i], want[i])
		}
	}
}

func TestReplanInjuryReadErrorAborts(t *testing.T) {
	r := NewReplanner(fakeInjuryDoc{err: errors.New("db down")}, fakeLocations{}, goodReadiness(), nil)
	if _, err := r.Check(context.Background(), uuid.New(), replanSince); err == nil {
		t.Fatalf("expected an error when injuries can't be read")
	}
}

func hasReason(d ReplanDecision, reason string) bool {
	for _, r := range d.Reasons {
		if r == reason {
			return true
		}
	}
	return false
}
