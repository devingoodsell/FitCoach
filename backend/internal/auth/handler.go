package auth

import (
	"errors"
	"net/http"

	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// deviceLabelHeader lets a client tag its session for the multi-device list.
const deviceLabelHeader = "X-Device-Label"

// Handler exposes the auth HTTP endpoints.
type Handler struct {
	svc    *Service
	logger *logging.Logger
}

// NewHandler wires a Handler.
func NewHandler(svc *Service, logger *logging.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// Register attaches auth routes to the router.
func (h *Handler) Register(r *httpx.Router) {
	r.HandleFunc("POST /auth/signup", h.signup)
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) signup(w http.ResponseWriter, r *http.Request) {
	var req credentials
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "malformed request body")
		return
	}

	tokens, err := h.svc.Signup(r.Context(), req.Email, req.Password, r.Header.Get(deviceLabelHeader))
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusCreated, tokens)
	case errors.Is(err, ErrInvalidEmail):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_email", "enter a valid email address")
	case errors.Is(err, ErrWeakPassword):
		httpx.WriteError(w, http.StatusBadRequest, "weak_password",
			"password must be at least 10 characters and mix letters with numbers or symbols")
	case errors.Is(err, ErrEmailTaken):
		// Non-enumerating: same generic copy regardless of which email it was.
		httpx.WriteError(w, http.StatusConflict, "signup_failed", "could not complete signup")
	default:
		h.logger.Error("signup failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}
