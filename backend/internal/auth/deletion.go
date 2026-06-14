package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// DeleteAccount permanently removes the account after confirming the caller's
// password. Deletion cascades to all user-owned data via FK constraints. It is
// idempotent: if the account is already gone, it returns nil.
func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID, password string) error {
	user, err := s.store.GetUserByID(ctx, userID)
	if errors.Is(err, ErrUserNotFound) {
		return nil // already deleted
	}
	if err != nil {
		return err
	}
	ok, err := VerifyPassword(user.PasswordHash, password)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidCredentials
	}
	return s.store.DeleteUser(ctx, userID)
}
