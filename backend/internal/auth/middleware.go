package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/platform/httpx"
)

type ctxKey int

const userIDKey ctxKey = iota

// Authenticator verifies an access token and returns the user it identifies.
// *Service satisfies this; defining the interface here keeps RequireAuth usable
// from other packages without importing the concrete service.
type Authenticator interface {
	ParseAccessToken(token string) (uuid.UUID, error)
}

// RequireAuth returns middleware that rejects requests without a valid bearer
// access token and stores the authenticated user ID in the request context.
func RequireAuth(a Authenticator) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}
			userID, err := a.ParseAccessToken(token)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired session")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext returns the authenticated user ID set by RequireAuth.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	return id, ok
}

func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(h[len(prefix):]), true
}
