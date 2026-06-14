package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pro.d11l.fitcoach/backend/internal/platform/httpx"
)

func TestLimiterBlocksAfterMaxFailures(t *testing.T) {
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	l := NewLimiter(3, 10*time.Minute, clock)

	for i := 0; i < 2; i++ {
		l.Fail("k")
		if blocked, _ := l.Retry("k"); blocked {
			t.Fatalf("blocked too early after %d failures", i+1)
		}
	}
	l.Fail("k") // third failure trips the block
	blocked, retryAfter := l.Retry("k")
	if !blocked {
		t.Fatal("expected block after max failures")
	}
	if retryAfter <= 0 || retryAfter > 10*time.Minute {
		t.Errorf("retryAfter = %v, want (0, 10m]", retryAfter)
	}

	// After the cooldown elapses, attempts are allowed again.
	now = now.Add(11 * time.Minute)
	if blocked, _ := l.Retry("k"); blocked {
		t.Error("still blocked after cooldown elapsed")
	}
}

func TestLimiterResetClearsFailures(t *testing.T) {
	clock := func() time.Time { return time.Unix(0, 0) }
	l := NewLimiter(2, time.Minute, clock)
	l.Fail("k")
	l.Reset("k")
	l.Fail("k") // counts as the first failure again, not the second
	if blocked, _ := l.Retry("k"); blocked {
		t.Fatal("reset did not clear failure count")
	}
}

func TestLoginHandlerThrottlesRepeatedFailures(t *testing.T) {
	h := newTestHandler()
	// seed an account so failures are wrong-password, not unknown-account
	_ = postJSON(h, "/auth/signup", `{"email":"user@b.co","password":"abcd1234ef"}`)

	router := httpx.NewRouter()
	h.Register(router)

	doLogin := func() int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login",
			strings.NewReader(`{"email":"user@b.co","password":"wrongpass99"}`))
		req.RemoteAddr = "203.0.113.7:5555"
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	// First maxAuthFailures attempts are 401; the next is throttled (429).
	for i := 0; i < maxAuthFailures; i++ {
		if code := doLogin(); code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want 401", i+1, code)
		}
	}
	if code := doLogin(); code != http.StatusTooManyRequests {
		t.Fatalf("post-threshold status = %d, want 429", code)
	}
}
