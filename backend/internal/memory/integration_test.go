package memory_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/memory"
	"pro.d11l.fitcoach/backend/internal/platform/db"
	"pro.d11l.fitcoach/backend/migrations"
)

// requireDB skips the test unless FITCOACH_TEST_MYSQL_DSN points at a usable
// MySQL. CI sets this; local runs without MySQL skip these tests.
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

// makeUser inserts a bare account row and returns its id.
func makeUser(t *testing.T, database *sql.DB, email string) uuid.UUID {
	t.Helper()
	id, _ := uuid.NewV7()
	now := time.Now().UTC()
	_, err := database.Exec(
		`INSERT INTO users (id, email, email_norm, password_hash, email_verified, created_at, updated_at)
		 VALUES (?, ?, ?, 'x', 0, ?, ?)`,
		id[:], email, email, now, now)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

func TestIntegrationSectionRoundTripAndIsolation(t *testing.T) {
	database := requireDB(t)
	defer database.Close()
	store := memory.NewStore(database, memory.NewUpgrader(), nil)
	ctx := context.Background()

	alice := makeUser(t, database, "alice-"+uuid.NewString()+"@example.com")
	bob := makeUser(t, database, "bob-"+uuid.NewString()+"@example.com")
	t.Cleanup(func() {
		_, _ = database.Exec(`DELETE FROM users WHERE id IN (?, ?)`, alice[:], bob[:])
	})

	// Round-trip a section for Alice.
	if _, err := store.PutSection(ctx, alice, memory.SectionProfile, json.RawMessage(`{"age":40}`)); err != nil {
		t.Fatalf("put: %v", err)
	}
	got, err := store.GetSection(ctx, alice, memory.SectionProfile)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got.Data) != `{"age": 40}` && string(got.Data) != `{"age":40}` {
		t.Fatalf("round-trip data = %s", got.Data)
	}

	// Upsert replaces, not duplicates.
	if _, err := store.PutSection(ctx, alice, memory.SectionProfile, json.RawMessage(`{"age":41}`)); err != nil {
		t.Fatalf("re-put: %v", err)
	}
	all, err := store.GetAll(ctx, alice)
	if err != nil {
		t.Fatalf("getall: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 section after upsert, got %d", len(all))
	}

	// Cross-user isolation: Bob sees nothing of Alice's.
	if _, err := store.GetSection(ctx, bob, memory.SectionProfile); !errors.Is(err, memory.ErrSectionNotFound) {
		t.Fatalf("bob read alice's data: err = %v", err)
	}
}

func TestIntegrationCascadeDeleteRemovesMemory(t *testing.T) {
	database := requireDB(t)
	defer database.Close()
	store := memory.NewStore(database, memory.NewUpgrader(), nil)
	ctx := context.Background()

	uid := makeUser(t, database, "carol-"+uuid.NewString()+"@example.com")
	if _, err := store.PutSection(ctx, uid, memory.SectionGoals, json.RawMessage(`{"strength":0.5}`)); err != nil {
		t.Fatalf("put: %v", err)
	}

	// Deleting the user cascades to coach_memory_sections.
	if _, err := database.Exec(`DELETE FROM users WHERE id = ?`, uid[:]); err != nil {
		t.Fatalf("delete user: %v", err)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM coach_memory_sections WHERE user_id = ?`, uid[:]).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected memory removed by cascade, found %d rows", count)
	}
}
