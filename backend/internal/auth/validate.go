package auth

import (
	"errors"
	"net/mail"
	"strings"
	"unicode"
)

// Validation errors. Handlers map these to 400s with safe messages.
var (
	ErrInvalidEmail = errors.New("invalid email address")
	ErrWeakPassword = errors.New("password does not meet strength requirements")
)

const (
	minPasswordLen = 10
	maxPasswordLen = 256
	maxEmailLen    = 320
)

// normalizeEmail lowercases and trims an email for uniqueness comparisons. The
// original-cased value is preserved separately for display.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validateEmail checks RFC-5322 addr-spec shape and length.
func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" || len(email) > maxEmailLen {
		return ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return ErrInvalidEmail
	}
	return nil
}

// validatePassword enforces length and basic complexity (at least one letter and
// one non-letter) so trivially weak passwords are rejected without being so
// strict that good passphrases fail.
func validatePassword(pw string) error {
	if len(pw) < minPasswordLen || len(pw) > maxPasswordLen {
		return ErrWeakPassword
	}
	var hasLetter, hasOther bool
	for _, r := range pw {
		if unicode.IsLetter(r) {
			hasLetter = true
		} else {
			hasOther = true
		}
	}
	if !hasLetter || !hasOther {
		return ErrWeakPassword
	}
	return nil
}
