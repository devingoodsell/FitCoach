package auth

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// deviceLabelHeader lets a client tag its session for the multi-device list.
const deviceLabelHeader = "X-Device-Label"

// Auth backoff defaults: block a key after this many consecutive failures, for
// this long. Applied per account and per client IP.
const (
	maxAuthFailures = 5
	authCooldown    = 15 * time.Minute
)

// Handler exposes the auth HTTP endpoints.
type Handler struct {
	svc     *Service
	logger  *logging.Logger
	limiter *Limiter
}

// NewHandler wires a Handler with a default failure-backoff limiter.
func NewHandler(svc *Service, logger *logging.Logger) *Handler {
	return &Handler{
		svc:     svc,
		logger:  logger,
		limiter: NewLimiter(maxAuthFailures, authCooldown, nil),
	}
}

// Register attaches auth routes to the router.
func (h *Handler) Register(r *httpx.Router) {
	r.HandleFunc("POST /auth/signup", h.signup)
	r.HandleFunc("POST /auth/login", h.login)
	r.HandleFunc("POST /auth/logout", h.logout)
	r.HandleFunc("POST /auth/refresh", h.refresh)
	r.HandleFunc("POST /auth/reset/request", h.resetRequest)
	r.HandleFunc("POST /auth/reset/confirm", h.resetConfirm)
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshBody struct {
	RefreshToken string `json:"refresh_token"`
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

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req credentials
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "malformed request body")
		return
	}

	emailKey := "login:email:" + normalizeEmail(req.Email)
	ipKey := "login:ip:" + clientIP(r)
	if h.throttled(w, emailKey, ipKey) {
		return
	}

	tokens, err := h.svc.Login(r.Context(), req.Email, req.Password, r.Header.Get(deviceLabelHeader))
	switch {
	case err == nil:
		h.limiter.Reset(emailKey)
		h.limiter.Reset(ipKey)
		httpx.WriteJSON(w, http.StatusOK, tokens)
	case errors.Is(err, ErrInvalidCredentials):
		h.limiter.Fail(emailKey)
		h.limiter.Fail(ipKey)
		// Generic: do not reveal whether the email or the password was wrong.
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "incorrect email or password")
	default:
		h.logger.Error("login failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}

type resetRequestBody struct {
	Email string `json:"email"`
}

func (h *Handler) resetRequest(w http.ResponseWriter, r *http.Request) {
	var req resetRequestBody
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "malformed request body")
		return
	}

	// Throttle reset spam per IP and per email, but always respond 200 so the
	// caller cannot enumerate accounts via timing or status differences.
	ipKey := "reset:ip:" + clientIP(r)
	emailKey := "reset:email:" + normalizeEmail(req.Email)
	if blocked, _ := h.limiter.Retry(ipKey); !blocked {
		if blocked, _ := h.limiter.Retry(emailKey); !blocked {
			if err := h.svc.RequestPasswordReset(r.Context(), req.Email); err != nil {
				h.logger.Error("reset request failed", "error", err.Error())
			}
			h.limiter.Fail(ipKey)
			h.limiter.Fail(emailKey)
		}
	}
	w.WriteHeader(http.StatusOK)
}

type resetConfirmBody struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) resetConfirm(w http.ResponseWriter, r *http.Request) {
	var req resetConfirmBody
	if err := httpx.DecodeJSON(r, &req); err != nil || req.Token == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "token and new_password are required")
		return
	}

	err := h.svc.ConfirmPasswordReset(r.Context(), req.Token, req.NewPassword)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ErrWeakPassword):
		httpx.WriteError(w, http.StatusBadRequest, "weak_password",
			"password must be at least 10 characters and mix letters with numbers or symbols")
	case errors.Is(err, ErrInvalidResetToken):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_token", "this reset link is invalid or has expired")
	default:
		h.logger.Error("reset confirm failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}

// throttled writes a 429 with Retry-After and returns true if any key is blocked.
func (h *Handler) throttled(w http.ResponseWriter, keys ...string) bool {
	for _, k := range keys {
		if blocked, retryAfter := h.limiter.Retry(k); blocked {
			secs := int(retryAfter.Seconds()) + 1
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			httpx.WriteError(w, http.StatusTooManyRequests, "too_many_attempts",
				"too many attempts; please try again later")
			return true
		}
	}
	return false
}

// clientIP extracts the caller's IP, honoring a single X-Forwarded-For hop when
// present (set by our own proxy), else falling back to RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(first)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var req refreshBody
	if err := httpx.DecodeJSON(r, &req); err != nil || req.RefreshToken == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return
	}
	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		h.logger.Error("logout failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshBody
	if err := httpx.DecodeJSON(r, &req); err != nil || req.RefreshToken == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return
	}
	tokens, err := h.svc.Refresh(r.Context(), req.RefreshToken, r.Header.Get(deviceLabelHeader))
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusOK, tokens)
	case errors.Is(err, ErrInvalidRefresh):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_refresh", "session expired; please sign in again")
	default:
		h.logger.Error("refresh failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}
