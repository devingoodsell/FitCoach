package location

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/memory"
)

// Errors returned to handlers.
var (
	ErrNotFound = errors.New("location not found")
	ErrInvalid  = errors.New("invalid location")
)

// sectionStore is the Coach Memory surface this package reads/writes
// (consumer-defined; *memory.Store satisfies it).
type sectionStore interface {
	GetSection(ctx context.Context, userID uuid.UUID, section memory.Section) (memory.SectionRecord, error)
	PutSection(ctx context.Context, userID uuid.UUID, section memory.Section, data json.RawMessage) (memory.SectionRecord, error)
}

// Service manages locations and current context within the locations section.
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

// Get returns the locations document (empty if none set yet).
func (s *Service) Get(ctx context.Context, userID uuid.UUID) (Doc, error) {
	rec, err := s.store.GetSection(ctx, userID, memory.SectionLocations)
	if errors.Is(err, memory.ErrSectionNotFound) {
		return Doc{Locations: []Location{}}, nil
	}
	if err != nil {
		return Doc{}, err
	}
	var doc Doc
	if err := json.Unmarshal(rec.Data, &doc); err != nil {
		return Doc{}, fmt.Errorf("decode locations: %w", err)
	}
	if doc.Locations == nil {
		doc.Locations = []Location{}
	}
	return doc, nil
}

// Add creates a location with a server-assigned id.
func (s *Service) Add(ctx context.Context, userID uuid.UUID, name string, equipment []string) (Location, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Location{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	doc, err := s.Get(ctx, userID)
	if err != nil {
		return Location{}, err
	}
	loc := Location{ID: uuid.NewString(), Name: name, Equipment: normalizeEquipment(equipment)}
	doc.Locations = append(doc.Locations, loc)
	if err := s.save(ctx, userID, doc); err != nil {
		return Location{}, err
	}
	return loc, nil
}

// Update edits a location's name/equipment.
func (s *Service) Update(ctx context.Context, userID uuid.UUID, id, name string, equipment []string) (Location, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Location{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	doc, err := s.Get(ctx, userID)
	if err != nil {
		return Location{}, err
	}
	i := doc.find(id)
	if i < 0 {
		return Location{}, ErrNotFound
	}
	doc.Locations[i].Name = name
	doc.Locations[i].Equipment = normalizeEquipment(equipment)
	if err := s.save(ctx, userID, doc); err != nil {
		return Location{}, err
	}
	return doc.Locations[i], nil
}

// Delete removes a location, clearing the current context if it pointed there.
func (s *Service) Delete(ctx context.Context, userID uuid.UUID, id string) error {
	doc, err := s.Get(ctx, userID)
	if err != nil {
		return err
	}
	i := doc.find(id)
	if i < 0 {
		return ErrNotFound
	}
	doc.Locations = append(doc.Locations[:i], doc.Locations[i+1:]...)
	if doc.Current != nil && doc.Current.LocationID == id {
		doc.Current = nil
	}
	return s.save(ctx, userID, doc)
}

// SetCurrent switches the active context to an existing location. The change is
// timestamped so the engine can treat it as a re-planning trigger (E5-S5).
func (s *Service) SetCurrent(ctx context.Context, userID uuid.UUID, locationID, note string) (CurrentContext, error) {
	doc, err := s.Get(ctx, userID)
	if err != nil {
		return CurrentContext{}, err
	}
	if doc.find(locationID) < 0 {
		return CurrentContext{}, ErrNotFound
	}
	current := CurrentContext{LocationID: locationID, Note: strings.TrimSpace(note), ChangedAt: s.now()}
	doc.Current = &current
	if err := s.save(ctx, userID, doc); err != nil {
		return CurrentContext{}, err
	}
	return current, nil
}

func (s *Service) save(ctx context.Context, userID uuid.UUID, doc Doc) error {
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal locations: %w", err)
	}
	if _, err := s.store.PutSection(ctx, userID, memory.SectionLocations, data); err != nil {
		return fmt.Errorf("persist locations: %w", err)
	}
	return nil
}

func normalizeEquipment(equipment []string) []string {
	out := make([]string, 0, len(equipment))
	for _, e := range equipment {
		if e = strings.TrimSpace(e); e != "" {
			out = append(out, e)
		}
	}
	return out
}
