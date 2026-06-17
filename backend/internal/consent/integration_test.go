package consent_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/consent"
	"pro.d11l.fitcoach/backend/internal/platform/db"
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
	email := "consent-" + uuid.NewString() + "@example.com"
	_, err := database.Exec(
		`INSERT INTO users (id, email, email_norm, password_hash, email_verified, created_at, updated_at)
		 VALUES (?, ?, ?, 'x', 0, ?, ?)`, id[:], email, email, now, now)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = database.Exec(`DELETE FROM users WHERE id = ?`, id[:]) })
	return id
}

// Revoking health-data consent flips HasConsent to false (disabling ingestion /
// manual mode), preserves the audit row, surfaces revoked_at in List, and a fresh
// Record re-enables it.
func TestIntegrationRevokeAndReconsent(t *testing.T) {
	database := requireDB(t)
	defer database.Close()
	store := consent.NewStore(database)
	ctx := context.Background()
	uid := makeUser(t, database)

	t0 := time.Date(2026, 6, 16, 8, 0, 0, 0, time.UTC)
	if err := store.Record(ctx, uid, consent.TypeHealthData, "v1", t0); err != nil {
		t.Fatalf("record: %v", err)
	}
	if ok, err := store.HasConsent(ctx, uid, consent.TypeHealthData); err != nil || !ok {
		t.Fatalf("HasConsent after record = %v, %v; want true", ok, err)
	}

	if err := store.Revoke(ctx, uid, consent.TypeHealthData, t0.Add(time.Hour)); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if ok, err := store.HasConsent(ctx, uid, consent.TypeHealthData); err != nil || ok {
		t.Fatalf("HasConsent after revoke = %v, %v; want false", ok, err)
	}

	consents, err := store.List(ctx, uid)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(consents) != 1 || consents[0].RevokedAt == nil {
		t.Fatalf("expected revoked health_data in list, got %+v", consents)
	}

	// Re-consent (a fresh, un-revoked row) restores the gate.
	if err := store.Record(ctx, uid, consent.TypeHealthData, "v1", t0.Add(2*time.Hour)); err != nil {
		t.Fatalf("re-record: %v", err)
	}
	if ok, err := store.HasConsent(ctx, uid, consent.TypeHealthData); err != nil || !ok {
		t.Fatalf("HasConsent after re-consent = %v, %v; want true", ok, err)
	}
}

// Revoke is idempotent and scoped to one type: revoking an absent consent is a
// no-op, and revoking health_data leaves medical_disclaimer untouched.
func TestIntegrationRevokeIsScopedAndIdempotent(t *testing.T) {
	database := requireDB(t)
	defer database.Close()
	store := consent.NewStore(database)
	ctx := context.Background()
	uid := makeUser(t, database)

	now := time.Date(2026, 6, 16, 8, 0, 0, 0, time.UTC)
	_ = store.Record(ctx, uid, consent.TypeHealthData, "v1", now)
	_ = store.Record(ctx, uid, consent.TypeMedicalDisclaimer, "v1", now)

	// Idempotent: revoking twice does not error.
	if err := store.Revoke(ctx, uid, consent.TypeHealthData, now.Add(time.Hour)); err != nil {
		t.Fatalf("revoke 1: %v", err)
	}
	if err := store.Revoke(ctx, uid, consent.TypeHealthData, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("revoke 2: %v", err)
	}

	if ok, _ := store.HasConsent(ctx, uid, consent.TypeHealthData); ok {
		t.Fatalf("health_data still in force after revoke")
	}
	if ok, _ := store.HasConsent(ctx, uid, consent.TypeMedicalDisclaimer); !ok {
		t.Fatalf("medical_disclaimer should be unaffected by health_data revoke")
	}
}
