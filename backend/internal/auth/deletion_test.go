package auth

import (
	"context"
	"errors"
	"testing"
)

func TestDeleteAccountRemovesUserAndSessions(t *testing.T) {
	svc, store := testService()
	ctx := context.Background()
	tokens, _ := svc.Signup(ctx, "user@example.com", "abcd1234ef", "")
	userID := store.usersByEmail["user@example.com"].ID

	if err := svc.DeleteAccount(ctx, userID, "abcd1234ef"); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if _, ok := store.usersByID[userID]; ok {
		t.Error("user row still present after deletion")
	}
	// Cascade: sessions are gone, so the refresh token is unusable.
	if _, err := svc.Refresh(ctx, tokens.RefreshToken, ""); !errors.Is(err, ErrInvalidRefresh) {
		t.Errorf("session survived deletion: %v", err)
	}
}

func TestDeleteAccountRequiresCorrectPassword(t *testing.T) {
	svc, store := testService()
	ctx := context.Background()
	_, _ = svc.Signup(ctx, "user@example.com", "abcd1234ef", "")
	userID := store.usersByEmail["user@example.com"].ID

	if err := svc.DeleteAccount(ctx, userID, "wrongpass99"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("err = %v, want ErrInvalidCredentials", err)
	}
	if _, ok := store.usersByID[userID]; !ok {
		t.Error("user deleted despite wrong password")
	}
}

func TestDeleteAccountIsIdempotent(t *testing.T) {
	svc, store := testService()
	ctx := context.Background()
	_, _ = svc.Signup(ctx, "user@example.com", "abcd1234ef", "")
	userID := store.usersByEmail["user@example.com"].ID

	if err := svc.DeleteAccount(ctx, userID, "abcd1234ef"); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	// Second delete (account already gone) succeeds without error.
	if err := svc.DeleteAccount(ctx, userID, "anything-goes"); err != nil {
		t.Errorf("idempotent delete err = %v, want nil", err)
	}
}
