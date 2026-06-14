package auth

import (
	"context"

	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// Mailer delivers transactional emails. This is the seam where a real provider
// (SES/SendGrid/etc.) plugs in for production.
type Mailer interface {
	// SendPasswordReset delivers a reset token to the account email. Implementations
	// build the user-facing link/flow around the opaque token.
	SendPasswordReset(ctx context.Context, email, token string) error
}

// LogMailer is the MVP stub: it logs that a reset was requested instead of
// sending mail. The token is logged ONLY because this is a developer stub; the
// production Mailer must never log it.
type LogMailer struct {
	logger *logging.Logger
}

// NewLogMailer returns a LogMailer.
func NewLogMailer(logger *logging.Logger) *LogMailer {
	return &LogMailer{logger: logger}
}

// SendPasswordReset logs the reset details for local development.
func (m *LogMailer) SendPasswordReset(_ context.Context, email, token string) error {
	m.logger.Info("password reset requested (stub delivery)",
		"email", email,
		"dev_reset_token", token, // stub only — replace LogMailer in production
	)
	return nil
}

// noopMailer drops messages; used as a safe default when none is provided.
type noopMailer struct{}

func (noopMailer) SendPasswordReset(context.Context, string, string) error { return nil }
