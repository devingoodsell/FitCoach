package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// accountStore is the persistence surface the Service needs. Defining it here
// (consumer-side) keeps the Service unit-testable with a fake and lets *Store
// satisfy it implicitly.
type accountStore interface {
	CreateUser(ctx context.Context, email, passwordHash string, now time.Time) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash, deviceLabel string, expiresAt, now time.Time) error
	GetRefreshToken(ctx context.Context, tokenHash string) (RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string, now time.Time) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, now time.Time) error
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt, now time.Time) error
	GetPasswordResetToken(ctx context.Context, tokenHash string) (PasswordResetToken, error)
	ConsumeResetAndSetPassword(ctx context.Context, userID uuid.UUID, tokenHash, newHash string, now time.Time) error
}

// resetTokenTTL bounds how long a password-reset link is valid.
const resetTokenTTL = time.Hour

// Config carries the secrets and TTLs the Service needs.
type Config struct {
	JWTKey     []byte
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// Service implements the account/session use cases.
type Service struct {
	store      accountStore
	mailer     Mailer
	jwtKey     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

// NewService constructs a Service. mailer defaults to a no-op when nil; now
// defaults to time.Now (UTC) when nil. Tests inject a fixed clock and fake mailer.
func NewService(store accountStore, cfg Config, mailer Mailer, now func() time.Time) *Service {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	if mailer == nil {
		mailer = noopMailer{}
	}
	return &Service{
		store:      store,
		mailer:     mailer,
		jwtKey:     cfg.JWTKey,
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
		now:        now,
	}
}

// Signup validates input, creates the account, and issues a session. A duplicate
// email returns ErrEmailTaken, which the handler renders non-enumerably.
func (s *Service) Signup(ctx context.Context, email, password, deviceLabel string) (TokenPair, error) {
	if err := validateEmail(email); err != nil {
		return TokenPair{}, err
	}
	if err := validatePassword(password); err != nil {
		return TokenPair{}, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return TokenPair{}, err
	}
	user, err := s.store.CreateUser(ctx, normalizeEmail(email), hash, s.now())
	if err != nil {
		return TokenPair{}, err // ErrEmailTaken handled by caller
	}
	return s.issueSession(ctx, user.ID, deviceLabel)
}

// issueSession mints an access JWT and a persisted, rotating refresh token.
func (s *Service) issueSession(ctx context.Context, userID uuid.UUID, deviceLabel string) (TokenPair, error) {
	now := s.now()
	access, err := s.issueAccessToken(userID, now)
	if err != nil {
		return TokenPair{}, err
	}
	plaintext, hash, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}
	if err := s.store.CreateRefreshToken(ctx, userID, hash, deviceLabel, now.Add(s.refreshTTL), now); err != nil {
		return TokenPair{}, fmt.Errorf("persist session: %w", err)
	}
	return TokenPair{
		AccessToken:  access,
		RefreshToken: plaintext,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTTL.Seconds()),
	}, nil
}
