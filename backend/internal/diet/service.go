package diet

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/disclaimer"
	"pro.d11l.fitcoach/backend/internal/memory"
	"pro.d11l.fitcoach/backend/internal/onboarding"
)

// sectionStore is the read surface diet needs (consumer-defined).
type sectionStore interface {
	GetSection(ctx context.Context, userID uuid.UUID, section memory.Section) (memory.SectionRecord, error)
}

// Service computes targets and guidance from the user's Coach Memory.
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

// Result is the daily targets payload, framed as guidance.
type Result struct {
	Targets    Targets  `json:"targets"`
	Guidance   []string `json:"guidance"`
	Pattern    string   `json:"pattern"`
	Disclaimer string   `json:"disclaimer"`
}

// NoteResult is the post-workout note payload.
type NoteResult struct {
	Note       string `json:"note"`
	Disclaimer string `json:"disclaimer"`
}

// Targets computes daily calorie/protein ranges and preference-aware guidance.
func (s *Service) Targets(ctx context.Context, userID uuid.UUID) (Result, error) {
	profile := s.readProfile(ctx, userID)
	goals := s.readGoals(ctx, userID)
	pattern := s.readDietPattern(ctx, userID)
	days := s.readDaysPerWeek(ctx, userID)

	in := Inputs{Goals: goals.Normalized(), DaysPerWeek: days}
	if profile != nil {
		if age, ok := profile.DerivedAge(s.now()); ok {
			in.Age = age
		}
		in.Sex = string(profile.Sex)
		if w, ok := currentWeight(profile); ok {
			in.WeightKg, in.HasWeight = w, true
		}
		if profile.HeightCm != nil {
			in.HeightCm, in.HasHeight = *profile.HeightCm, true
		}
	}

	return Result{
		Targets:    ComputeTargets(in),
		Guidance:   Guidance(pattern),
		Pattern:    string(pattern),
		Disclaimer: disclaimer.Medical,
	}, nil
}

// PostWorkoutNote returns a load-aware, preference-aware note (E11-S3).
func (s *Service) PostWorkoutNote(ctx context.Context, userID uuid.UUID, heavy bool) (NoteResult, error) {
	pattern := s.readDietPattern(ctx, userID)
	return NoteResult{Note: PostWorkoutNote(heavy, pattern), Disclaimer: disclaimer.Medical}, nil
}

func (s *Service) readProfile(ctx context.Context, userID uuid.UUID) *onboarding.Profile {
	var p onboarding.Profile
	if s.decode(ctx, userID, memory.SectionProfile, &p) {
		return &p
	}
	return nil
}

func (s *Service) readGoals(ctx context.Context, userID uuid.UUID) onboarding.GoalWeights {
	var g onboarding.GoalWeights
	s.decode(ctx, userID, memory.SectionGoals, &g)
	return g
}

func (s *Service) readDietPattern(ctx context.Context, userID uuid.UUID) onboarding.DietPattern {
	var d onboarding.DietPrefs
	s.decode(ctx, userID, memory.SectionDiet, &d)
	return d.Pattern
}

func (s *Service) readDaysPerWeek(ctx context.Context, userID uuid.UUID) int {
	var sc onboarding.Schedule
	s.decode(ctx, userID, memory.SectionSchedule, &sc)
	return sc.DaysPerWeek
}

// decode loads a section into dst, returning false if absent or unreadable.
func (s *Service) decode(ctx context.Context, userID uuid.UUID, section memory.Section, dst any) bool {
	rec, err := s.store.GetSection(ctx, userID, section)
	if errors.Is(err, memory.ErrSectionNotFound) || err != nil {
		return false
	}
	return json.Unmarshal(rec.Data, dst) == nil
}

// currentWeight returns the most recent weight (history wins over the raw field).
func currentWeight(p *onboarding.Profile) (float64, bool) {
	if n := len(p.WeightHistory); n > 0 {
		return p.WeightHistory[n-1].Kg, true
	}
	if p.WeightKg != nil {
		return *p.WeightKg, true
	}
	return 0, false
}
