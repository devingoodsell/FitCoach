package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

func TestValidateEmail(t *testing.T) {
	good := []string{"a@b.co", "user.name+tag@example.com"}
	bad := []string{"", "no-at", "a@", "@b.com", "a b@c.com", "two@@c.com"}
	for _, e := range good {
		if err := validateEmail(e); err != nil {
			t.Errorf("validateEmail(%q) = %v, want nil", e, err)
		}
	}
	for _, e := range bad {
		if err := validateEmail(e); err == nil {
			t.Errorf("validateEmail(%q) = nil, want error", e)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if err := validatePassword("abcd1234ef"); err != nil {
		t.Errorf("strong password rejected: %v", err)
	}
	for _, p := range []string{"short1", "alllettersonly", "1234567890"} {
		if err := validatePassword(p); err == nil {
			t.Errorf("validatePassword(%q) = nil, want error", p)
		}
	}
}

func TestSignupServiceCreatesUserAndSession(t *testing.T) {
	svc, store := testService()
	tokens, err := svc.Signup(context.Background(), "New.User@Example.com", "abcd1234ef", "pixel")
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("expected non-empty tokens")
	}
	if tokens.TokenType != "Bearer" || tokens.ExpiresIn != 900 {
		t.Errorf("token meta = %+v", tokens)
	}
	// User stored under normalized email.
	if _, err := store.GetUserByEmail(context.Background(), "new.user@example.com"); err != nil {
		t.Errorf("user not stored normalized: %v", err)
	}
	// Refresh token persisted by hash, not plaintext.
	if _, ok := store.tokens[hashRefreshToken(tokens.RefreshToken)]; !ok {
		t.Error("refresh token not persisted by hash")
	}
	// Access token verifies and carries the user id.
	if _, err := svc.ParseAccessToken(tokens.AccessToken); err != nil {
		t.Errorf("ParseAccessToken: %v", err)
	}
}

func TestSignupServiceRejectsDuplicate(t *testing.T) {
	svc, _ := testService()
	ctx := context.Background()
	if _, err := svc.Signup(ctx, "dup@example.com", "abcd1234ef", ""); err != nil {
		t.Fatalf("first signup: %v", err)
	}
	_, err := svc.Signup(ctx, "DUP@example.com", "abcd1234ef", "")
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("duplicate signup err = %v, want ErrEmailTaken", err)
	}
}

func TestSignupServiceRejectsWeakInputs(t *testing.T) {
	svc, _ := testService()
	ctx := context.Background()
	if _, err := svc.Signup(ctx, "bad-email", "abcd1234ef", ""); !errors.Is(err, ErrInvalidEmail) {
		t.Errorf("err = %v, want ErrInvalidEmail", err)
	}
	if _, err := svc.Signup(ctx, "ok@example.com", "weak", ""); !errors.Is(err, ErrWeakPassword) {
		t.Errorf("err = %v, want ErrWeakPassword", err)
	}
}

func newTestHandler() *Handler {
	svc, _ := testService()
	return NewHandler(svc, logging.New(io.Discard, "error"))
}

func postJSON(h *Handler, path, body string) *httptest.ResponseRecorder {
	r := httpx.NewRouter()
	h.Register(r)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.ServeHTTP(rec, req)
	return rec
}

func TestSignupHandlerStatuses(t *testing.T) {
	t.Run("created", func(t *testing.T) {
		rec := postJSON(newTestHandler(), "/auth/signup", `{"email":"a@b.co","password":"abcd1234ef"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (%s)", rec.Code, rec.Body)
		}
		var tp TokenPair
		if err := json.Unmarshal(rec.Body.Bytes(), &tp); err != nil || tp.AccessToken == "" {
			t.Fatalf("bad token body: %v %s", err, rec.Body)
		}
	})

	t.Run("weak password", func(t *testing.T) {
		rec := postJSON(newTestHandler(), "/auth/signup", `{"email":"a@b.co","password":"weak"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("duplicate is generic", func(t *testing.T) {
		h := newTestHandler()
		_ = postJSON(h, "/auth/signup", `{"email":"dup@b.co","password":"abcd1234ef"}`)
		rec := postJSON(h, "/auth/signup", `{"email":"dup@b.co","password":"abcd1234ef"}`)
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", rec.Code)
		}
		body := strings.ToLower(rec.Body.String())
		if strings.Contains(body, "exist") || strings.Contains(body, "already") || strings.Contains(body, "registered") {
			t.Errorf("duplicate response enumerates account: %s", rec.Body)
		}
	})
}
