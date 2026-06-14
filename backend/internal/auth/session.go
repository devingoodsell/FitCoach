package auth

import (
	"context"
	"errors"
)

// Session-related sentinel errors. Both render as generic 401s so callers can't
// distinguish "no such account" from "wrong password" or "bad token".
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidRefresh     = errors.New("invalid refresh token")
)

// dummyHash is verified against when an account is not found, so login takes
// roughly constant time regardless of whether the email exists.
var dummyHash, _ = HashPassword("timing-equalization-placeholder")

// Login authenticates by email + password and issues a session. Any failure
// (unknown account or wrong password) returns ErrInvalidCredentials.
func (s *Service) Login(ctx context.Context, email, password, deviceLabel string) (TokenPair, error) {
	user, err := s.store.GetUserByEmail(ctx, email)
	if errors.Is(err, ErrUserNotFound) {
		_, _ = VerifyPassword(dummyHash, password) // equalize timing
		return TokenPair{}, ErrInvalidCredentials
	}
	if err != nil {
		return TokenPair{}, err
	}
	ok, err := VerifyPassword(user.PasswordHash, password)
	if err != nil {
		return TokenPair{}, err
	}
	if !ok {
		return TokenPair{}, ErrInvalidCredentials
	}
	return s.issueSession(ctx, user.ID, deviceLabel)
}

// Logout revokes the supplied refresh token. Idempotent: revoking an unknown or
// already-revoked token is not an error.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.store.RevokeRefreshToken(ctx, hashRefreshToken(refreshToken), s.now())
}

// Refresh validates a refresh token and rotates it: the presented token is
// revoked and a fresh access+refresh pair is issued. Revoked/expired/unknown
// tokens return ErrInvalidRefresh.
func (s *Service) Refresh(ctx context.Context, refreshToken, deviceLabel string) (TokenPair, error) {
	hash := hashRefreshToken(refreshToken)
	rt, err := s.store.GetRefreshToken(ctx, hash)
	if errors.Is(err, ErrUserNotFound) {
		return TokenPair{}, ErrInvalidRefresh
	}
	if err != nil {
		return TokenPair{}, err
	}
	now := s.now()
	if rt.RevokedAt.Valid || !rt.ExpiresAt.After(now) {
		return TokenPair{}, ErrInvalidRefresh
	}
	if err := s.store.RevokeRefreshToken(ctx, hash, now); err != nil {
		return TokenPair{}, err
	}
	return s.issueSession(ctx, rt.UserID, deviceLabel)
}
