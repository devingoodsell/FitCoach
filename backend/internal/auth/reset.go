package auth

import (
	"context"
	"errors"
)

// RequestPasswordReset issues a reset token for the account if it exists and
// hands it to the mailer. It never reveals whether the email was found: callers
// always treat the result as success (no enumeration).
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.store.GetUserByEmail(ctx, email)
	if errors.Is(err, ErrUserNotFound) {
		return nil // silently succeed for unknown emails
	}
	if err != nil {
		return err
	}

	plaintext, hash, err := newRefreshToken() // reuse the opaque-token generator
	if err != nil {
		return err
	}
	now := s.now()
	if err := s.store.CreatePasswordResetToken(ctx, user.ID, hash, now.Add(resetTokenTTL), now); err != nil {
		return err
	}
	return s.mailer.SendPasswordReset(ctx, user.Email, plaintext)
}

// ConfirmPasswordReset validates a reset token, sets the new password (single-use,
// time-limited), and revokes all of the account's existing sessions so other
// devices are signed out. Invalid/expired/used tokens return ErrInvalidResetToken.
func (s *Service) ConfirmPasswordReset(ctx context.Context, token, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	hash := hashRefreshToken(token)
	prt, err := s.store.GetPasswordResetToken(ctx, hash)
	if errors.Is(err, ErrUserNotFound) {
		return ErrInvalidResetToken
	}
	if err != nil {
		return err
	}
	now := s.now()
	if prt.UsedAt.Valid || !prt.ExpiresAt.After(now) {
		return ErrInvalidResetToken
	}

	newHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.store.ConsumeResetAndSetPassword(ctx, prt.UserID, hash, newHash, now); err != nil {
		return err
	}
	// Sign out other devices; a reset implies the old sessions may be compromised.
	return s.store.RevokeAllUserTokens(ctx, prt.UserID, now)
}
