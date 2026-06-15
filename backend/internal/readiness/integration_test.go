package readiness_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/platform/db"
	"pro.d11l.fitcoach/backend/internal/readiness"
	"pro.d11l.fitcoach/backend/migrations"
)

func requireDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("FITCOACH_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set FITCOACH_TEST_MYSQL_DSN to run MySQL integration tests")
	}
	database, err := db.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(context.Background(), database, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}

func makeUser(t *testing.T, database *sql.DB) uuid.UUID {
	t.Helper()
	id, _ := uuid.NewV7()
	now := time.Now().UTC()
	email := "readiness-" + uuid.NewString() + "@example.com"
	_, err := database.Exec(
		`INSERT INTO users (id, email, email_norm, password_hash, email_verified, created_at, updated_at)
		 VALUES (?, ?, ?, 'x', 0, ?, ?)`, id[:], email, email, now, now)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = database.Exec(`DELETE FROM users WHERE id = ?`, id[:]) })
	return id
}

func TestIntegrationSignalsRoundTripAndCascade(t *testing.T) {
	database := requireDB(t)
	defer database.Close()
	store := readiness.NewStore(database)
	ctx := context.Background()
	uid := makeUser(t, database)

	now := time.Date(2026, 6, 14, 7, 0, 0, 0, time.UTC)
	samples := []readiness.Sample{
		{Kind: readiness.KindHRV, Value: 60, Day: "2026-06-13"},
		{Kind: readiness.KindHRV, Value: 58, Day: "2026-06-14"},
	}
	if err := store.Upsert(ctx, uid, samples, now); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	// Idempotent re-upsert of the same day updates, not duplicates.
	if err := store.Upsert(ctx, uid, []readiness.Sample{{Kind: readiness.KindHRV, Value: 59, Day: "2026-06-14"}}, now); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}

	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM health_signals WHERE user_id = ?`, uid[:]).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows after idempotent upsert, got %d", count)
	}

	// Cascade on user deletion.
	if _, err := database.Exec(`DELETE FROM users WHERE id = ?`, uid[:]); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	_ = database.QueryRow(`SELECT COUNT(*) FROM health_signals WHERE user_id = ?`, uid[:]).Scan(&count)
	if count != 0 {
		t.Fatalf("expected signals removed by cascade, found %d", count)
	}
}
