package db

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestLoadMigrationsSortsByVersion(t *testing.T) {
	fsys := fstest.MapFS{
		"0010_later.sql":  {Data: []byte("SELECT 10")},
		"0002_second.sql": {Data: []byte("SELECT 2")},
		"0001_first.sql":  {Data: []byte("SELECT 1")},
	}
	got, err := LoadMigrations(fsys)
	if err != nil {
		t.Fatalf("LoadMigrations: %v", err)
	}
	want := []int{1, 2, 10}
	if len(got) != len(want) {
		t.Fatalf("got %d migrations, want %d", len(got), len(want))
	}
	for i, m := range got {
		if m.Version != want[i] {
			t.Errorf("position %d version = %d, want %d", i, m.Version, want[i])
		}
	}
	if got[0].SQL != "SELECT 1" {
		t.Errorf("SQL not loaded: %q", got[0].SQL)
	}
}

func TestLoadMigrationsRejectsBadNames(t *testing.T) {
	tests := map[string]fstest.MapFS{
		"no underscore": {"0001.sql": {Data: []byte("x")}},
		"non-numeric":   {"vONE_first.sql": {Data: []byte("x")}},
		"duplicate version": {
			"0001_a.sql": {Data: []byte("x")},
			"0001_b.sql": {Data: []byte("y")},
		},
	}
	for name, fsys := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadMigrations(fsys); err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestMigrateAppliesPendingOnFreshDB(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()

	fsys := fstest.MapFS{
		"0001_a.sql": {Data: []byte("CREATE TABLE a (id INT)")},
		"0002_b.sql": {Data: []byte("CREATE TABLE b (id INT)")},
	}

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT version FROM schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"version"})) // none applied yet

	for _, v := range []int{1, 2} {
		mock.ExpectBegin()
		mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO schema_migrations").
			WithArgs(v, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()
	}

	if err := Migrate(context.Background(), database, fsys); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMigrateSkipsAlreadyApplied(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()

	fsys := fstest.MapFS{
		"0001_a.sql": {Data: []byte("CREATE TABLE a (id INT)")},
		"0002_b.sql": {Data: []byte("CREATE TABLE b (id INT)")},
	}

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))
	// Version 1 already applied; only 2 should run.
	mock.ExpectQuery("SELECT version FROM schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(1))

	mock.ExpectBegin()
	mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO schema_migrations").
		WithArgs(2, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	if err := Migrate(context.Background(), database, fsys); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestMigrateNoOpWhenAllApplied(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()

	fsys := fstest.MapFS{"0001_a.sql": {Data: []byte("CREATE TABLE a (id INT)")}}

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT version FROM schema_migrations").
		WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(1))
	// No Begin/Exec/Commit expected: nothing to apply.

	if err := Migrate(context.Background(), database, fsys); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
