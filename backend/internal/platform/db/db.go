// Package db owns the MySQL connection pool, a small transaction helper, and an
// ordered file-based migration runner. Domain packages depend on *sql.DB (or the
// narrower interfaces they define), not on this package's internals.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Open returns a configured, pooled *sql.DB for the given DSN and verifies the
// connection with a ping. parseTime/UTC are enforced so DATETIME(6) round-trips
// to time.Time consistently regardless of the DSN supplied.
func Open(ctx context.Context, dsn string) (*sql.DB, error) {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse mysql dsn: %w", err)
	}
	cfg.ParseTime = true
	cfg.Loc = time.UTC

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, fmt.Errorf("mysql connector: %w", err)
	}

	pool := sql.OpenDB(connector)
	pool.SetConnMaxLifetime(5 * time.Minute)
	pool.SetMaxOpenConns(25)
	pool.SetMaxIdleConns(25)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.PingContext(pingCtx); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return pool, nil
}

// DBTX is the subset of *sql.DB and *sql.Tx that repositories need. Defining it
// here lets handlers run the same query code inside or outside a transaction.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// WithTx runs fn inside a transaction, committing on success and rolling back on
// error or panic. The error from fn is wrapped; rollback errors are joined.
func WithTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("%w; rollback: %v", err, rbErr)
			}
			return
		}
		err = tx.Commit()
	}()
	return fn(tx)
}
