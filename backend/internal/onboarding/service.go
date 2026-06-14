package onboarding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/memory"
)

// sectionStore is the Coach Memory surface onboarding writes through
// (consumer-defined for testability; *memory.Store satisfies it).
type sectionStore interface {
	GetSection(ctx context.Context, userID uuid.UUID, section memory.Section) (memory.SectionRecord, error)
	PutSection(ctx context.Context, userID uuid.UUID, section memory.Section, data json.RawMessage) (memory.SectionRecord, error)
}

// Service validates user-model input and persists it into Coach Memory.
type Service struct {
	store sectionStore
	now   func() time.Time
}

// NewService wires a Service. now defaults to time.Now (UTC) when nil.
func NewService(store sectionStore, now func() time.Time) *Service {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, now: now}
}

// SaveProfile validates and persists the profile section. It maintains the
// timestamped weight history (E2-S2) and defaults aging emphases from age (E2-S8)
// when not supplied.
func (s *Service) SaveProfile(ctx context.Context, userID uuid.UUID, p Profile) (Profile, error) {
	now := s.now()
	if err := p.Validate(now); err != nil {
		return Profile{}, err
	}

	history, err := s.existingWeightHistory(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	if p.WeightKg != nil {
		history = append(history, WeightEntry{Kg: *p.WeightKg, RecordedAt: now})
	}
	p.WeightHistory = history

	if p.AgingEmphases == nil {
		if age, ok := p.DerivedAge(now); ok {
			def := DefaultAgingEmphases(age)
			p.AgingEmphases = &def
		}
	}
	return p, s.put(ctx, userID, memory.SectionProfile, p)
}

func (s *Service) existingWeightHistory(ctx context.Context, userID uuid.UUID) ([]WeightEntry, error) {
	rec, err := s.store.GetSection(ctx, userID, memory.SectionProfile)
	if errors.Is(err, memory.ErrSectionNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var prev Profile
	if err := json.Unmarshal(rec.Data, &prev); err != nil {
		return nil, nil // corrupt prior data shouldn't block a fresh save
	}
	return prev.WeightHistory, nil
}

// SaveGoals validates and persists the goal distribution, normalized to sum 1.
func (s *Service) SaveGoals(ctx context.Context, userID uuid.UUID, g GoalWeights) (GoalWeights, error) {
	if err := g.Validate(); err != nil {
		return GoalWeights{}, err
	}
	norm := g.Normalized()
	return norm, s.put(ctx, userID, memory.SectionGoals, norm)
}

// SaveSchedule validates and persists the schedule.
func (s *Service) SaveSchedule(ctx context.Context, userID uuid.UUID, sc Schedule) (Schedule, error) {
	if err := sc.Validate(); err != nil {
		return Schedule{}, err
	}
	return sc, s.put(ctx, userID, memory.SectionSchedule, sc)
}

// SaveDiet validates and persists dietary preferences (read by E11).
func (s *Service) SaveDiet(ctx context.Context, userID uuid.UUID, d DietPrefs) (DietPrefs, error) {
	if err := d.Validate(); err != nil {
		return DietPrefs{}, err
	}
	return d, s.put(ctx, userID, memory.SectionDiet, d)
}

// SavePreferences persists exercise/equipment preferences (no hard validation).
func (s *Service) SavePreferences(ctx context.Context, userID uuid.UUID, p Preferences) (Preferences, error) {
	return p, s.put(ctx, userID, memory.SectionPreferences, p)
}

func (s *Service) put(ctx context.Context, userID uuid.UUID, section memory.Section, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", section, err)
	}
	if _, err := s.store.PutSection(ctx, userID, section, data); err != nil {
		return fmt.Errorf("persist %s: %w", section, err)
	}
	return nil
}
