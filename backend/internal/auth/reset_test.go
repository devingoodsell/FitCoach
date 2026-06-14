package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRequestResetIssuesTokenForKnownEmail(t *testing.T) {
	svc, store := testService()
	ctx := context.Background()
	_, _ = svc.Signup(ctx, "user@example.com", "abcd1234ef", "")

	if err := svc.RequestPasswordReset(ctx, "USER@example.com"); err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}
	if store.mailer.calls != 1 || store.mailer.lastToken == "" {
		t.Fatalf("expected one delivery with a token, got %+v", store.mailer)
	}
	if len(store.resetTokens) != 1 {
		t.Errorf("expected one persisted reset token, got %d", len(store.resetTokens))
	}
}

func TestRequestResetSilentForUnknownEmail(t *testing.T) {
	svc, store := testService()
	if err := svc.RequestPasswordReset(context.Background(), "nobody@example.com"); err != nil {
		t.Fatalf("should not error for unknown email: %v", err)
	}
	if store.mailer.calls != 0 {
		t.Errorf("no mail should be sent for unknown email, got %d", store.mailer.calls)
	}
}

func TestConfirmResetChangesPasswordAndRevokesSessions(t *testing.T) {
	svc, store := testService()
	ctx := context.Background()
	session, _ := svc.Signup(ctx, "user@example.com", "abcd1234ef", "")
	_ = svc.RequestPasswordReset(ctx, "user@example.com")
	token := store.mailer.lastToken

	if err := svc.ConfirmPasswordReset(ctx, token, "newpass1234"); err != nil {
		t.Fatalf("ConfirmPasswordReset: %v", err)
	}
	// Old password no longer works; new one does.
	if _, err := svc.Login(ctx, "user@example.com", "abcd1234ef", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("old password still works: %v", err)
	}
	if _, err := svc.Login(ctx, "user@example.com", "newpass1234", ""); err != nil {
		t.Errorf("new password should work: %v", err)
	}
	// Pre-existing sessions are revoked.
	if _, err := svc.Refresh(ctx, session.RefreshToken, ""); !errors.Is(err, ErrInvalidRefresh) {
		t.Errorf("existing session should be revoked after reset: %v", err)
	}
}

func TestConfirmResetTokenIsSingleUse(t *testing.T) {
	svc, store := testService()
	ctx := context.Background()
	_, _ = svc.Signup(ctx, "user@example.com", "abcd1234ef", "")
	_ = svc.RequestPasswordReset(ctx, "user@example.com")
	token := store.mailer.lastToken

	if err := svc.ConfirmPasswordReset(ctx, token, "newpass1234"); err != nil {
		t.Fatalf("first confirm: %v", err)
	}
	if err := svc.ConfirmPasswordReset(ctx, token, "another12345"); !errors.Is(err, ErrInvalidResetToken) {
		t.Errorf("second use err = %v, want ErrInvalidResetToken", err)
	}
}

func TestConfirmResetRejectsExpiredToken(t *testing.T) {
	store := newFakeStore()
	store.mailer = &fakeMailer{}
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	cfg := Config{JWTKey: []byte("test-signing-key-at-least-32-bytes!!"), AccessTTL: time.Minute, RefreshTTL: time.Hour}
	svc := NewService(store, cfg, store.mailer, func() time.Time { return now })
	ctx := context.Background()
	_, _ = svc.Signup(ctx, "user@example.com", "abcd1234ef", "")
	_ = svc.RequestPasswordReset(ctx, "user@example.com")
	token := store.mailer.lastToken

	now = now.Add(resetTokenTTL + time.Minute) // advance past expiry
	if err := svc.ConfirmPasswordReset(ctx, token, "newpass1234"); !errors.Is(err, ErrInvalidResetToken) {
		t.Errorf("expired token err = %v, want ErrInvalidResetToken", err)
	}
}

func TestConfirmResetRejectsUnknownToken(t *testing.T) {
	svc, _ := testService()
	if err := svc.ConfirmPasswordReset(context.Background(), "not-a-real-token", "newpass1234"); !errors.Is(err, ErrInvalidResetToken) {
		t.Errorf("unknown token err = %v, want ErrInvalidResetToken", err)
	}
}
