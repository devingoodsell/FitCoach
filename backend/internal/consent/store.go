package consent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store persists consent records in MySQL.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store backed by the given pool.
func NewStore(database *sql.DB) *Store { return &Store{db: database} }

// Record appends a consent acceptance.
func (s *Store) Record(ctx context.Context, userID uuid.UUID, ctype, version string, now time.Time) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate consent id: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO consents (id, user_id, type, version, accepted_at) VALUES (?, ?, ?, ?, ?)`,
		id[:], userID[:], ctype, version, now)
	if err != nil {
		return fmt.Errorf("insert consent: %w", err)
	}
	return nil
}

// HasConsent reports whether the user has ever recorded a consent of the given
// type (used to gate health-data ingestion in E4).
func (s *Store) HasConsent(ctx context.Context, userID uuid.UUID, ctype string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM consents WHERE user_id = ? AND type = ? LIMIT 1`, userID[:], ctype).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query consent: %w", err)
	}
	return true, nil
}

// List returns the current consent state: the most recent acceptance per type.
func (s *Store) List(ctx context.Context, userID uuid.UUID) ([]Consent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT type, version, accepted_at FROM consents WHERE user_id = ? ORDER BY accepted_at DESC`,
		userID[:])
	if err != nil {
		return nil, fmt.Errorf("query consents: %w", err)
	}
	defer rows.Close()

	seen := make(map[string]bool)
	var out []Consent
	for rows.Next() {
		var c Consent
		if err := rows.Scan(&c.Type, &c.Version, &c.AcceptedAt); err != nil {
			return nil, fmt.Errorf("scan consent: %w", err)
		}
		if seen[c.Type] {
			continue // keep only the latest per type
		}
		seen[c.Type] = true
		out = append(out, c)
	}
	return out, rows.Err()
}
