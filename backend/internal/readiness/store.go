package readiness

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/platform/db"
)

// Sample is a raw daily signal value.
type Sample struct {
	Kind  string  `json:"kind"`
	Value float64 `json:"value"`
	Day   string  `json:"day"` // YYYY-MM-DD
}

// dayValue is a stored sample for one day.
type dayValue struct {
	Day   string
	Value float64
}

// Store persists raw health signals in MySQL, account-scoped.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store.
func NewStore(database *sql.DB) *Store { return &Store{db: database} }

// Upsert writes samples idempotently on (user, kind, day): re-uploading a day
// updates rather than duplicates it (E12 sync safety).
func (s *Store) Upsert(ctx context.Context, userID uuid.UUID, samples []Sample, now time.Time) error {
	if len(samples) == 0 {
		return nil
	}
	return db.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		for _, sample := range samples {
			id, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("generate signal id: %w", err)
			}
			_, err = tx.ExecContext(ctx,
				`INSERT INTO health_signals (id, user_id, kind, value, day, created_at)
				 VALUES (?, ?, ?, ?, ?, ?)
				 ON DUPLICATE KEY UPDATE value = VALUES(value)`,
				id[:], userID[:], sample.Kind, sample.Value, sample.Day, now)
			if err != nil {
				return fmt.Errorf("upsert signal: %w", err)
			}
		}
		return nil
	})
}

// recent returns up to `days` most-recent samples for a kind, newest first.
func (s *Store) recent(ctx context.Context, userID uuid.UUID, kind string, days int) ([]dayValue, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT day, value FROM health_signals WHERE user_id = ? AND kind = ? ORDER BY day DESC LIMIT ?`,
		userID[:], kind, days)
	if err != nil {
		return nil, fmt.Errorf("query signals: %w", err)
	}
	defer rows.Close()

	var out []dayValue
	for rows.Next() {
		var dv dayValue
		var day time.Time
		if err := rows.Scan(&day, &dv.Value); err != nil {
			return nil, fmt.Errorf("scan signal: %w", err)
		}
		dv.Day = day.Format("2006-01-02")
		out = append(out, dv)
	}
	return out, rows.Err()
}
