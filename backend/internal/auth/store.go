package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/platform/db"
)

// mysqlErrDup is the MySQL error number for a duplicate-key violation.
const mysqlErrDup = 1062

// ErrEmailTaken is returned when a signup collides with an existing account.
// Handlers must translate this to a non-enumerating response.
var ErrEmailTaken = errors.New("email already registered")

// ErrUserNotFound is returned when a lookup matches no row.
var ErrUserNotFound = errors.New("user not found")

// User is a persisted account.
type User struct {
	ID            uuid.UUID
	Email         string
	PasswordHash  string
	EmailVerified bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// RefreshToken is a persisted session record (hash only, never the plaintext).
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt sql.NullTime
}

// Store provides account and session persistence over MySQL. The narrow db.DBTX
// dependency lets callers run within or outside a transaction.
type Store struct {
	db *sql.DB
}

// NewStore returns a Store backed by the given pool.
func NewStore(database *sql.DB) *Store { return &Store{db: database} }

// CreateUser inserts a new account. A duplicate email yields ErrEmailTaken.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash string, now time.Time) (User, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return User{}, fmt.Errorf("generate user id: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO users (id, email, email_norm, password_hash, email_verified, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 0, ?, ?)`,
		id[:], email, normalizeEmail(email), passwordHash, now, now)
	if err != nil {
		var myErr *mysql.MySQLError
		if errors.As(err, &myErr) && myErr.Number == mysqlErrDup {
			return User{}, ErrEmailTaken
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}
	return User{ID: id, Email: email, PasswordHash: passwordHash, CreatedAt: now, UpdatedAt: now}, nil
}

// GetUserByEmail looks up an account by normalized email.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	var idBytes []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT id, email, password_hash, email_verified, created_at, updated_at
		 FROM users WHERE email_norm = ?`, normalizeEmail(email)).
		Scan(&idBytes, &u.Email, &u.PasswordHash, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("query user: %w", err)
	}
	if u.ID, err = uuid.FromBytes(idBytes); err != nil {
		return User{}, fmt.Errorf("parse user id: %w", err)
	}
	return u, nil
}

// CreateRefreshToken persists a session for the user.
func (s *Store) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash, deviceLabel string, expiresAt, now time.Time) error {
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate token id: %w", err)
	}
	var label any
	if deviceLabel != "" {
		label = deviceLabel
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, device_label, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id[:], userID[:], tokenHash, label, expiresAt, now)
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

// GetRefreshToken loads a session by token hash.
func (s *Store) GetRefreshToken(ctx context.Context, tokenHash string) (RefreshToken, error) {
	var rt RefreshToken
	var idBytes, userBytes []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT id, user_id, expires_at, revoked_at FROM refresh_tokens WHERE token_hash = ?`, tokenHash).
		Scan(&idBytes, &userBytes, &rt.ExpiresAt, &rt.RevokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return RefreshToken{}, ErrUserNotFound
	}
	if err != nil {
		return RefreshToken{}, fmt.Errorf("query refresh token: %w", err)
	}
	rt.ID, _ = uuid.FromBytes(idBytes)
	rt.UserID, _ = uuid.FromBytes(userBytes)
	return rt, nil
}

// RevokeRefreshToken marks a single session revoked. Idempotent.
func (s *Store) RevokeRefreshToken(ctx context.Context, tokenHash string, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ? WHERE token_hash = ? AND revoked_at IS NULL`,
		now, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

// RevokeAllUserTokens revokes every active session for a user (password reset).
func (s *Store) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, now time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`,
		now, userID[:])
	if err != nil {
		return fmt.Errorf("revoke user tokens: %w", err)
	}
	return nil
}

// compile-time assertion that *sql.DB satisfies the tx-capable surface we use.
var _ db.DBTX = (*sql.DB)(nil)
