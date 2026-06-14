package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// defaultWorkoutLimit / maxWorkoutLimit bound how many recent sessions are read.
const (
	defaultWorkoutLimit = 30
	maxWorkoutLimit     = 200
)

// WorkoutLog is a recorded session outcome.
type WorkoutLog struct {
	ID              uuid.UUID       `json:"id"`
	ClientSessionID string          `json:"client_session_id"`
	Version         int             `json:"schema_version"`
	Data            json.RawMessage `json:"data"`
	PerformedAt     time.Time       `json:"performed_at"`
}

// RecordWorkout persists a completed session. It is idempotent on
// (user_id, client_session_id): re-recording the same session (e.g. an offline
// client re-syncing) updates the payload rather than creating a duplicate.
func (s *Store) RecordWorkout(ctx context.Context, userID uuid.UUID, clientSessionID string, data json.RawMessage, performedAt time.Time) (WorkoutLog, error) {
	if clientSessionID == "" {
		return WorkoutLog{}, fmt.Errorf("client_session_id is required")
	}
	if !json.Valid(data) {
		return WorkoutLog{}, fmt.Errorf("workout data is not valid JSON")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return WorkoutLog{}, fmt.Errorf("generate workout id: %w", err)
	}
	now := s.now()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO workout_logs (id, user_id, client_session_id, schema_version, data, performed_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE schema_version = VALUES(schema_version), data = VALUES(data), performed_at = VALUES(performed_at)`,
		id[:], userID[:], clientSessionID, WorkoutLogVersion, []byte(data), performedAt, now)
	if err != nil {
		return WorkoutLog{}, fmt.Errorf("upsert workout: %w", err)
	}

	// Read back to return the canonical row (id is the original on conflict).
	var idBytes []byte
	var version int
	var stored time.Time
	err = s.db.QueryRowContext(ctx,
		`SELECT id, schema_version, performed_at FROM workout_logs WHERE user_id = ? AND client_session_id = ?`,
		userID[:], clientSessionID).Scan(&idBytes, &version, &stored)
	if err != nil {
		return WorkoutLog{}, fmt.Errorf("read back workout: %w", err)
	}
	storedID, _ := uuid.FromBytes(idBytes)
	return WorkoutLog{ID: storedID, ClientSessionID: clientSessionID, Version: version, Data: data, PerformedAt: stored}, nil
}

// RecentWorkouts returns the user's most recent sessions, newest first. limit is
// clamped to a sane range.
func (s *Store) RecentWorkouts(ctx context.Context, userID uuid.UUID, limit int) ([]WorkoutLog, error) {
	if limit <= 0 {
		limit = defaultWorkoutLimit
	}
	if limit > maxWorkoutLimit {
		limit = maxWorkoutLimit
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, client_session_id, schema_version, data, performed_at
		 FROM workout_logs WHERE user_id = ? ORDER BY performed_at DESC LIMIT ?`,
		userID[:], limit)
	if err != nil {
		return nil, fmt.Errorf("query workouts: %w", err)
	}
	defer rows.Close()

	var out []WorkoutLog
	for rows.Next() {
		var w WorkoutLog
		var idBytes, raw []byte
		if err := rows.Scan(&idBytes, &w.ClientSessionID, &w.Version, &raw, &w.PerformedAt); err != nil {
			return nil, fmt.Errorf("scan workout: %w", err)
		}
		w.ID, _ = uuid.FromBytes(idBytes)
		w.Data = json.RawMessage(raw)
		out = append(out, w)
	}
	return out, rows.Err()
}
