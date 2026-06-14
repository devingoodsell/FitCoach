package memory

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func newMockStore(t *testing.T, u *Upgrader) (*Store, sqlmock.Sqlmock, func()) {
	t.Helper()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	now := func() time.Time { return time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC) }
	return NewStore(database, u, now), mock, func() { _ = database.Close() }
}

func TestPutSectionRejectsUnknownSectionAndBadJSON(t *testing.T) {
	s, _, done := newMockStore(t, NewUpgrader())
	defer done()
	uid, _ := uuid.NewV7()
	if _, err := s.PutSection(context.Background(), uid, "bogus", json.RawMessage(`{}`)); !errors.Is(err, ErrUnknownSection) {
		t.Errorf("unknown section err = %v", err)
	}
	if _, err := s.PutSection(context.Background(), uid, SectionProfile, json.RawMessage(`{bad`)); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestPutSectionUpserts(t *testing.T) {
	s, mock, done := newMockStore(t, NewUpgrader())
	defer done()
	uid, _ := uuid.NewV7()

	mock.ExpectExec("INSERT INTO coach_memory_sections").
		WithArgs(sqlmock.AnyArg(), "profile", 1, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	rec, err := s.PutSection(context.Background(), uid, SectionProfile, json.RawMessage(`{"age":40}`))
	if err != nil {
		t.Fatalf("PutSection: %v", err)
	}
	if rec.Version != 1 || string(rec.Data) != `{"age":40}` {
		t.Errorf("record = %+v", rec)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestGetSectionReturnsCurrent(t *testing.T) {
	s, mock, done := newMockStore(t, NewUpgrader())
	defer done()
	uid, _ := uuid.NewV7()

	mock.ExpectQuery("SELECT schema_version, data, updated_at FROM coach_memory_sections").
		WithArgs(sqlmock.AnyArg(), "profile").
		WillReturnRows(sqlmock.NewRows([]string{"schema_version", "data", "updated_at"}).
			AddRow(1, []byte(`{"age":40}`), time.Now()))

	rec, err := s.GetSection(context.Background(), uid, SectionProfile)
	if err != nil {
		t.Fatalf("GetSection: %v", err)
	}
	if rec.Version != 1 || string(rec.Data) != `{"age":40}` {
		t.Errorf("record = %+v", rec)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestGetSectionNotFound(t *testing.T) {
	s, mock, done := newMockStore(t, NewUpgrader())
	defer done()
	uid, _ := uuid.NewV7()

	mock.ExpectQuery("SELECT schema_version, data, updated_at FROM coach_memory_sections").
		WithArgs(sqlmock.AnyArg(), "goals").
		WillReturnRows(sqlmock.NewRows([]string{"schema_version", "data", "updated_at"}))

	if _, err := s.GetSection(context.Background(), uid, SectionGoals); !errors.Is(err, ErrSectionNotFound) {
		t.Fatalf("err = %v, want ErrSectionNotFound", err)
	}
}

func TestGetSectionPersistsUpgrade(t *testing.T) {
	// Upgrader targets profile v2 so a stored v1 row is migrated and written back.
	u := &Upgrader{upgrades: map[Section]map[int]UpgradeFunc{}, target: map[Section]int{SectionProfile: 2}}
	u.Register(SectionProfile, 1, func(data json.RawMessage) (json.RawMessage, error) {
		m := map[string]any{}
		_ = json.Unmarshal(data, &m)
		m["unit"] = "metric"
		return json.Marshal(m)
	})
	s, mock, done := newMockStore(t, u)
	defer done()
	uid, _ := uuid.NewV7()

	mock.ExpectQuery("SELECT schema_version, data, updated_at FROM coach_memory_sections").
		WithArgs(sqlmock.AnyArg(), "profile").
		WillReturnRows(sqlmock.NewRows([]string{"schema_version", "data", "updated_at"}).
			AddRow(1, []byte(`{"age":40}`), time.Now()))
	mock.ExpectExec("UPDATE coach_memory_sections SET schema_version").
		WithArgs(2, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), "profile").
		WillReturnResult(sqlmock.NewResult(0, 1))

	rec, err := s.GetSection(context.Background(), uid, SectionProfile)
	if err != nil {
		t.Fatalf("GetSection: %v", err)
	}
	if rec.Version != 2 {
		t.Errorf("version = %d, want 2", rec.Version)
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Data, &got)
	if got["age"] != float64(40) || got["unit"] != "metric" {
		t.Errorf("upgraded data wrong: %s", rec.Data)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
