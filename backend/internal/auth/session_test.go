package auth

import (
	"context"
	"errors"
	"testing"
)

func TestLoginSucceedsWithCorrectCredentials(t *testing.T) {
	svc, _ := testService()
	ctx := context.Background()
	if _, err := svc.Signup(ctx, "user@example.com", "abcd1234ef", ""); err != nil {
		t.Fatalf("signup: %v", err)
	}
	tokens, err := svc.Login(ctx, "USER@example.com", "abcd1234ef", "laptop")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := svc.ParseAccessToken(tokens.AccessToken); err != nil {
		t.Errorf("access token invalid: %v", err)
	}
}

func TestLoginGenericFailure(t *testing.T) {
	svc, _ := testService()
	ctx := context.Background()
	_, _ = svc.Signup(ctx, "user@example.com", "abcd1234ef", "")

	// Wrong password and unknown account both yield the same error.
	if _, err := svc.Login(ctx, "user@example.com", "wrongpass99", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("wrong password err = %v, want ErrInvalidCredentials", err)
	}
	if _, err := svc.Login(ctx, "nobody@example.com", "abcd1234ef", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("unknown account err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	svc, _ := testService()
	ctx := context.Background()
	tokens, _ := svc.Signup(ctx, "user@example.com", "abcd1234ef", "")

	if err := svc.Logout(ctx, tokens.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	// Revoked token can no longer be refreshed.
	if _, err := svc.Refresh(ctx, tokens.RefreshToken, ""); !errors.Is(err, ErrInvalidRefresh) {
		t.Errorf("refresh after logout err = %v, want ErrInvalidRefresh", err)
	}
}

func TestRefreshRotatesAndInvalidatesOld(t *testing.T) {
	svc, _ := testService()
	ctx := context.Background()
	tokens, _ := svc.Signup(ctx, "user@example.com", "abcd1234ef", "")

	rotated, err := svc.Refresh(ctx, tokens.RefreshToken, "")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if rotated.RefreshToken == tokens.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	// Old token is now invalid (single-use rotation).
	if _, err := svc.Refresh(ctx, tokens.RefreshToken, ""); !errors.Is(err, ErrInvalidRefresh) {
		t.Errorf("reuse of old token err = %v, want ErrInvalidRefresh", err)
	}
	// New token works.
	if _, err := svc.Refresh(ctx, rotated.RefreshToken, ""); err != nil {
		t.Errorf("rotated token should work: %v", err)
	}
}

func TestMultiDeviceSessionsAreIndependent(t *testing.T) {
	svc, store := testService()
	ctx := context.Background()
	_, _ = svc.Signup(ctx, "user@example.com", "abcd1234ef", "phone")
	second, _ := svc.Login(ctx, "user@example.com", "abcd1234ef", "tablet")

	// Two active sessions exist for the user.
	active := 0
	for _, rt := range store.tokens {
		if !rt.RevokedAt.Valid {
			active++
		}
	}
	if active != 2 {
		t.Fatalf("active sessions = %d, want 2", active)
	}

	// Logging out one device leaves the other usable.
	if err := svc.Logout(ctx, second.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Refresh(ctx, second.RefreshToken, ""); !errors.Is(err, ErrInvalidRefresh) {
		t.Errorf("logged-out device still valid: %v", err)
	}
}
