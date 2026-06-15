package readiness

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// baselineDays is how far back the rolling baseline reaches.
const baselineDays = 14

// ErrConsentRequired is returned when ingesting signals without health-data consent.
var ErrConsentRequired = errors.New("health-data consent required")

// signalStore is the persistence surface (consumer-defined for testing).
type signalStore interface {
	Upsert(ctx context.Context, userID uuid.UUID, samples []Sample, now time.Time) error
	recent(ctx context.Context, userID uuid.UUID, kind string, days int) ([]dayValue, error)
}

// consentChecker gates ingestion on health-data consent (E1-S4).
type consentChecker interface {
	HasConsent(ctx context.Context, userID uuid.UUID, ctype string) (bool, error)
}

// Service ingests raw signals and computes today's readiness.
type Service struct {
	store   signalStore
	consent consentChecker
	now     func() time.Time
}

// NewService wires a Service. now defaults to time.Now (UTC) when nil.
func NewService(store signalStore, consent consentChecker, now func() time.Time) *Service {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, consent: consent, now: now}
}

// Ingest stores raw signals, gated by health-data consent.
func (s *Service) Ingest(ctx context.Context, userID uuid.UUID, samples []Sample) error {
	ok, err := s.consent.HasConsent(ctx, userID, "health_data")
	if err != nil {
		return err
	}
	if !ok {
		return ErrConsentRequired
	}
	return s.store.Upsert(ctx, userID, samples, s.now())
}

// Today computes the current readiness from stored signals.
func (s *Service) Today(ctx context.Context, userID uuid.UUID) (Score, error) {
	today := s.now().Format("2006-01-02")
	hrv, err := s.metric(ctx, userID, KindHRV, today)
	if err != nil {
		return Score{}, err
	}
	rhr, err := s.metric(ctx, userID, KindRHR, today)
	if err != nil {
		return Score{}, err
	}
	sleep, err := s.metric(ctx, userID, KindSleep, today)
	if err != nil {
		return Score{}, err
	}
	return Compute(Inputs{HRV: hrv, RHR: rhr, Sleep: sleep}), nil
}

// metric splits stored samples into today's value and the prior baseline.
func (s *Service) metric(ctx context.Context, userID uuid.UUID, kind, today string) (Metric, error) {
	rows, err := s.store.recent(ctx, userID, kind, baselineDays+1)
	if err != nil {
		return Metric{}, err
	}
	var m Metric
	for _, dv := range rows {
		if dv.Day == today {
			m.Today = dv.Value
			m.HasToday = true
			continue
		}
		m.Baseline = append(m.Baseline, dv.Value)
	}
	return m, nil
}

// PoorRecovery reports whether the score is a re-planning trigger (E4-S4 / E5-S5):
// a confident, clearly-low reading.
func PoorRecovery(s Score) bool {
	return s.Confidence != ConfidenceLow && s.Value < 40
}
