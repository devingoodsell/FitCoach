package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// fakeStore is an in-memory accountStore for service/handler tests. It mirrors
// the real Store's behavior for the cases the tests exercise.
type fakeStore struct {
	usersByEmail map[string]User
	usersByID    map[uuid.UUID]User
	tokens       map[string]RefreshToken  // keyed by token hash
	resetTokens  map[string]resetTokenRow // keyed by token hash
	mailer       *fakeMailer
}

type resetTokenRow struct {
	userID    uuid.UUID
	expiresAt time.Time
	used      bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		usersByEmail: map[string]User{},
		usersByID:    map[uuid.UUID]User{},
		tokens:       map[string]RefreshToken{},
		resetTokens:  map[string]resetTokenRow{},
	}
}

func (f *fakeStore) CreateUser(_ context.Context, email, passwordHash string, now time.Time) (User, error) {
	norm := normalizeEmail(email)
	if _, ok := f.usersByEmail[norm]; ok {
		return User{}, ErrEmailTaken
	}
	id, _ := uuid.NewV7()
	u := User{ID: id, Email: norm, PasswordHash: passwordHash, CreatedAt: now, UpdatedAt: now}
	f.usersByEmail[norm] = u
	f.usersByID[id] = u
	return u, nil
}

func (f *fakeStore) GetUserByEmail(_ context.Context, email string) (User, error) {
	u, ok := f.usersByEmail[normalizeEmail(email)]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return u, nil
}

func (f *fakeStore) CreateRefreshToken(_ context.Context, userID uuid.UUID, tokenHash, _ string, expiresAt, _ time.Time) error {
	id, _ := uuid.NewV7()
	f.tokens[tokenHash] = RefreshToken{ID: id, UserID: userID, ExpiresAt: expiresAt}
	return nil
}

func (f *fakeStore) GetRefreshToken(_ context.Context, tokenHash string) (RefreshToken, error) {
	rt, ok := f.tokens[tokenHash]
	if !ok {
		return RefreshToken{}, ErrUserNotFound
	}
	return rt, nil
}

func (f *fakeStore) RevokeRefreshToken(_ context.Context, tokenHash string, now time.Time) error {
	if rt, ok := f.tokens[tokenHash]; ok && !rt.RevokedAt.Valid {
		rt.RevokedAt.Time = now
		rt.RevokedAt.Valid = true
		f.tokens[tokenHash] = rt
	}
	return nil
}

func (f *fakeStore) RevokeAllUserTokens(_ context.Context, userID uuid.UUID, now time.Time) error {
	for k, rt := range f.tokens {
		if rt.UserID == userID && !rt.RevokedAt.Valid {
			rt.RevokedAt.Time = now
			rt.RevokedAt.Valid = true
			f.tokens[k] = rt
		}
	}
	return nil
}

func (f *fakeStore) CreatePasswordResetToken(_ context.Context, userID uuid.UUID, tokenHash string, expiresAt, _ time.Time) error {
	f.resetTokens[tokenHash] = resetTokenRow{userID: userID, expiresAt: expiresAt}
	return nil
}

func (f *fakeStore) GetPasswordResetToken(_ context.Context, tokenHash string) (PasswordResetToken, error) {
	row, ok := f.resetTokens[tokenHash]
	if !ok {
		return PasswordResetToken{}, ErrUserNotFound
	}
	prt := PasswordResetToken{UserID: row.userID, ExpiresAt: row.expiresAt}
	prt.UsedAt.Valid = row.used
	return prt, nil
}

func (f *fakeStore) ConsumeResetAndSetPassword(_ context.Context, userID uuid.UUID, tokenHash, newHash string, _ time.Time) error {
	row, ok := f.resetTokens[tokenHash]
	if !ok || row.used {
		return ErrInvalidResetToken
	}
	row.used = true
	f.resetTokens[tokenHash] = row
	if u, ok := f.usersByID[userID]; ok {
		u.PasswordHash = newHash
		f.usersByID[userID] = u
		f.usersByEmail[u.Email] = u
	}
	return nil
}

// fakeMailer records the last password-reset delivery for assertions.
type fakeMailer struct {
	lastEmail string
	lastToken string
	calls     int
}

func (m *fakeMailer) SendPasswordReset(_ context.Context, email, token string) error {
	m.lastEmail = email
	m.lastToken = token
	m.calls++
	return nil
}

// testService builds a Service over a fresh fakeStore with a fixed clock and a
// capturing mailer (reachable via store.mailer).
func testService() (*Service, *fakeStore) {
	store := newFakeStore()
	store.mailer = &fakeMailer{}
	cfg := Config{
		JWTKey:     []byte("test-signing-key-at-least-32-bytes!!"),
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 720 * time.Hour,
	}
	fixed := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	return NewService(store, cfg, store.mailer, func() time.Time { return fixed }), store
}
