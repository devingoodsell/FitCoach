package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

// migrationsTable tracks which migrations have been applied. The runner creates
// it itself (idempotently) before applying any numbered migration, so a fresh
// database and a re-run both behave correctly.
const migrationsTable = "schema_migrations"

// Migration is a single ordered SQL migration parsed from a file named
// "<version>_<name>.sql", e.g. "0001_users.sql".
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// LoadMigrations reads, parses, and sorts migrations from fsys. It is pure (no
// DB) so ordering and filename validation can be unit-tested in isolation.
func LoadMigrations(fsys fs.FS) ([]Migration, error) {
	entries, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return nil, fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(entries)

	seen := make(map[int]string)
	migrations := make([]Migration, 0, len(entries))
	for _, name := range entries {
		version, err := parseVersion(name)
		if err != nil {
			return nil, err
		}
		if prev, dup := seen[version]; dup {
			return nil, fmt.Errorf("duplicate migration version %d: %s and %s", version, prev, name)
		}
		seen[version] = name

		content, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			SQL:     string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

func parseVersion(filename string) (int, error) {
	base := filename
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	idx := strings.IndexByte(base, '_')
	if idx <= 0 {
		return 0, fmt.Errorf("migration %q must be named <version>_<name>.sql", filename)
	}
	version, err := strconv.Atoi(base[:idx])
	if err != nil {
		return 0, fmt.Errorf("migration %q has non-numeric version: %w", filename, err)
	}
	return version, nil
}

// Migrate applies every pending migration from fsys in version order, recording
// each in the migrations table within the same transaction. Re-running is a
// no-op: already-applied versions are skipped. Safe to call on every boot.
func Migrate(ctx context.Context, database *sql.DB, fsys fs.FS) error {
	migrations, err := LoadMigrations(fsys)
	if err != nil {
		return err
	}
	if err := ensureMigrationsTable(ctx, database); err != nil {
		return err
	}
	applied, err := appliedVersions(ctx, database)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}
		if err := applyOne(ctx, database, m); err != nil {
			return fmt.Errorf("apply migration %d (%s): %w", m.Version, m.Name, err)
		}
	}
	return nil
}

func ensureMigrationsTable(ctx context.Context, database *sql.DB) error {
	_, err := database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS `+migrationsTable+` (
			version    BIGINT      NOT NULL PRIMARY KEY,
			name       VARCHAR(255) NOT NULL,
			applied_at DATETIME(6)  NOT NULL
		) ENGINE=InnoDB`)
	if err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}
	return nil
}

func appliedVersions(ctx context.Context, database *sql.DB) (map[int]bool, error) {
	rows, err := database.QueryContext(ctx, `SELECT version FROM `+migrationsTable)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func applyOne(ctx context.Context, database *sql.DB, m Migration) error {
	return WithTx(ctx, database, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
			return fmt.Errorf("exec sql: %w", err)
		}
		_, err := tx.ExecContext(ctx,
			`INSERT INTO `+migrationsTable+` (version, name, applied_at) VALUES (?, ?, UTC_TIMESTAMP(6))`,
			m.Version, m.Name)
		if err != nil {
			return fmt.Errorf("record migration: %w", err)
		}
		return nil
	})
}
