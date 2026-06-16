package injury

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/memory"
	"pro.d11l.fitcoach/backend/internal/safety"
)

// Errors returned to handlers.
var (
	ErrNotFound = errors.New("injury not found")
	ErrInvalid  = errors.New("invalid injury")
)

// sectionStore is the Coach Memory surface (consumer-defined).
type sectionStore interface {
	GetSection(ctx context.Context, userID uuid.UUID, section memory.Section) (memory.SectionRecord, error)
	PutSection(ctx context.Context, userID uuid.UUID, section memory.Section, data json.RawMessage) (memory.SectionRecord, error)
}

// Service manages injuries within the injuries memory section.
type Service struct {
	store  sectionStore
	parser Parser
	now    func() time.Time
}

// NewService wires a Service. parser defaults to the heuristic stand-in; now to
// time.Now (UTC).
func NewService(store sectionStore, parser Parser, now func() time.Time) *Service {
	if parser == nil {
		parser = NewHeuristicParser()
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{store: store, parser: parser, now: now}
}

// Get returns the injuries document (empty if unset).
func (s *Service) Get(ctx context.Context, userID uuid.UUID) (Doc, error) {
	rec, err := s.store.GetSection(ctx, userID, memory.SectionInjuries)
	if errors.Is(err, memory.ErrSectionNotFound) {
		return Doc{Injuries: []Injury{}}, nil
	}
	if err != nil {
		return Doc{}, err
	}
	var doc Doc
	if err := json.Unmarshal(rec.Data, &doc); err != nil {
		return Doc{}, fmt.Errorf("decode injuries: %w", err)
	}
	if doc.Injuries == nil {
		doc.Injuries = []Injury{}
	}
	return doc, nil
}

// ParseDraft turns freeform text into a reviewable draft (no persistence).
func (s *Service) ParseDraft(text string) Draft { return s.parser.Parse(text) }

// Add validates and stores a new injury, marking a lifecycle change.
func (s *Service) Add(ctx context.Context, userID uuid.UUID, in Injury) (Injury, error) {
	if err := validate(&in); err != nil {
		return Injury{}, err
	}
	doc, err := s.Get(ctx, userID)
	if err != nil {
		return Injury{}, err
	}
	now := s.now()
	in.ID = uuid.NewString()
	in.CreatedAt = now
	in.UpdatedAt = now
	doc.Injuries = append(doc.Injuries, in)
	if err := s.save(ctx, userID, doc, now); err != nil {
		return Injury{}, err
	}
	return in, nil
}

// Update edits an injury (status changes alter programming immediately, E7-S2).
func (s *Service) Update(ctx context.Context, userID uuid.UUID, id string, in Injury) (Injury, error) {
	if err := validate(&in); err != nil {
		return Injury{}, err
	}
	doc, err := s.Get(ctx, userID)
	if err != nil {
		return Injury{}, err
	}
	i := doc.find(id)
	if i < 0 {
		return Injury{}, ErrNotFound
	}
	now := s.now()
	in.ID = id
	in.CreatedAt = doc.Injuries[i].CreatedAt
	in.UpdatedAt = now
	doc.Injuries[i] = in
	if err := s.save(ctx, userID, doc, now); err != nil {
		return Injury{}, err
	}
	return in, nil
}

// Delete removes an injury, marking a lifecycle change.
func (s *Service) Delete(ctx context.Context, userID uuid.UUID, id string) error {
	doc, err := s.Get(ctx, userID)
	if err != nil {
		return err
	}
	i := doc.find(id)
	if i < 0 {
		return ErrNotFound
	}
	doc.Injuries = append(doc.Injuries[:i], doc.Injuries[i+1:]...)
	return s.save(ctx, userID, doc, s.now())
}

// Contraindications returns the active planning constraints for a user.
func (s *Service) Contraindications(ctx context.Context, userID uuid.UUID) ([]Contraindication, error) {
	doc, err := s.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	return doc.Contraindications(), nil
}

// SafetyView returns the safety-layer view of the user's injuries (E7-PR5).
func (s *Service) SafetyView(ctx context.Context, userID uuid.UUID) (safety.MemoryView, error) {
	doc, err := s.Get(ctx, userID)
	if err != nil {
		return safety.MemoryView{}, err
	}
	return doc.SafetyView(), nil
}

func (s *Service) save(ctx context.Context, userID uuid.UUID, doc Doc, now time.Time) error {
	doc.ChangedAt = &now // mark re-plan trigger (E5-S5 handshake)
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal injuries: %w", err)
	}
	if _, err := s.store.PutSection(ctx, userID, memory.SectionInjuries, data); err != nil {
		return fmt.Errorf("persist injuries: %w", err)
	}
	return nil
}

func validate(in *Injury) error {
	in.Region = strings.TrimSpace(in.Region)
	if in.Region == "" {
		return fmt.Errorf("%w: region is required", ErrInvalid)
	}
	if !validStatus(in.Status) {
		return fmt.Errorf("%w: invalid status", ErrInvalid)
	}
	if in.Severity == "" {
		in.Severity = SeverityModerate
	}
	if !validSeverity(in.Severity) {
		return fmt.Errorf("%w: invalid severity", ErrInvalid)
	}
	return nil
}
