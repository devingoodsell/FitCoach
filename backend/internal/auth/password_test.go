package auth

import (
	"strings"
	"testing"
)

func TestHashPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("unexpected hash format: %q", hash)
	}
	if strings.Contains(hash, "correct horse battery") {
		t.Fatal("hash must not contain plaintext")
	}

	ok, err := VerifyPassword(hash, "correct horse battery")
	if err != nil || !ok {
		t.Fatalf("verify correct = (%v, %v), want (true, nil)", ok, err)
	}

	ok, err = VerifyPassword(hash, "wrong password")
	if err != nil || ok {
		t.Fatalf("verify wrong = (%v, %v), want (false, nil)", ok, err)
	}
}

func TestHashPasswordUniqueSalts(t *testing.T) {
	a, _ := HashPassword("same-password-1")
	b, _ := HashPassword("same-password-1")
	if a == b {
		t.Fatal("identical passwords must hash differently (random salt)")
	}
}

func TestVerifyPasswordRejectsMalformedHash(t *testing.T) {
	for _, bad := range []string{"", "notahash", "$argon2id$bad", "$bcrypt$x$y$z$a$b"} {
		if _, err := VerifyPassword(bad, "x"); err == nil {
			t.Errorf("VerifyPassword(%q) error = nil, want error", bad)
		}
	}
}
