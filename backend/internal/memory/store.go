package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrSectionNotFound is returned when a section has not been set for a user.
var ErrSectionNotFound = errors.New("memory section not found")

// ErrUnknownSection is returned for a section name outside the fixed set.
var ErrUnknownSection = errors.New("unknown memory section")

// SectionRecord is a stored section after read-time upgrade.
type SectionRecord struct {
	Section   Section         `json:"section"`
	Version   int             `json:"schema_version"`
	Data      json.RawMessage `json:"data"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// Store reads and writes versioned Coach Memory, account-scoped. All queries are
// keyed by user_id, so cross-user access is impossible by construction.
type Store struct {
	db       *sql.DB
	upgrader *Upgrader
	now      func() time.Time
}

// NewStore returns a Store. now defaults to time.Now (UTC) when nil.
func NewStore(database *sql.DB, upgrader *Upgrader, now func() time.Time) *Store {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Store{db: database, upgrader: upgrader, now: now}
}

// PutSection upserts a section's JSON document at the section's current schema
// version.
func (s *Store) PutSection(ctx context.Context, userID uuid.UUID, section Section, data json.RawMessage) (SectionRecord, error) {
	if !IsValidSection(section) {
		return SectionRecord{}, ErrUnknownSection
	}
	if !json.Valid(data) {
		return SectionRecord{}, fmt.Errorf("section data is not valid JSON")
	}
	version := CurrentVersions[section]
	now := s.now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO coach_memory_sections (user_id, section, schema_version, data, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE schema_version = VALUES(schema_version), data = VALUES(data), updated_at = VALUES(updated_at)`,
		userID[:], string(section), version, []byte(data), now, now)
	if err != nil {
		return SectionRecord{}, fmt.Errorf("upsert section %q: %w", section, err)
	}
	return SectionRecord{Section: section, Version: version, Data: data, UpdatedAt: now}, nil
}

// GetSection reads a section and upgrades it to the current schema version,
// persisting the upgrade if the stored version was older.
func (s *Store) GetSection(ctx context.Context, userID uuid.UUID, section Section) (SectionRecord, error) {
	if !IsValidSection(section) {
		return SectionRecord{}, ErrUnknownSection
	}
	var storedVersion int
	var raw []byte
	var updatedAt time.Time
	err := s.db.QueryRowContext(ctx,
		`SELECT schema_version, data, updated_at FROM coach_memory_sections WHERE user_id = ? AND section = ?`,
		userID[:], string(section)).Scan(&storedVersion, &raw, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return SectionRecord{}, ErrSectionNotFound
	}
	if err != nil {
		return SectionRecord{}, fmt.Errorf("query section %q: %w", section, err)
	}

	upgraded, version, err := s.upgrader.Upgrade(section, storedVersion, raw)
	if err != nil {
		return SectionRecord{}, err
	}
	if version != storedVersion {
		if _, err := s.db.ExecContext(ctx,
			`UPDATE coach_memory_sections SET schema_version = ?, data = ?, updated_at = ? WHERE user_id = ? AND section = ?`,
			version, []byte(upgraded), s.now(), userID[:], string(section)); err != nil {
			return SectionRecord{}, fmt.Errorf("persist upgrade for section %q: %w", section, err)
		}
	}
	return SectionRecord{Section: section, Version: version, Data: upgraded, UpdatedAt: updatedAt}, nil
}

// GetAll returns every set section for the user, each upgraded to current.
func (s *Store) GetAll(ctx context.Context, userID uuid.UUID) ([]SectionRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT section, schema_version, data, updated_at FROM coach_memory_sections WHERE user_id = ? ORDER BY section`,
		userID[:])
	if err != nil {
		return nil, fmt.Errorf("query sections: %w", err)
	}
	defer rows.Close()

	var out []SectionRecord
	for rows.Next() {
		var name string
		var version int
		var raw []byte
		var updatedAt time.Time
		if err := rows.Scan(&name, &version, &raw, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan section: %w", err)
		}
		upgraded, newVersion, err := s.upgrader.Upgrade(Section(name), version, raw)
		if err != nil {
			return nil, err
		}
		out = append(out, SectionRecord{Section: Section(name), Version: newVersion, Data: upgraded, UpdatedAt: updatedAt})
	}
	return out, rows.Err()
}
