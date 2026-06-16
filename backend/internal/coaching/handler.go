package coaching

import (
	"errors"
	"net/http"
	"time"

	"pro.d11l.fitcoach/backend/internal/auth"
	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// Handler exposes session generation and the deterministic re-plan check. All
// routes require auth; generation is the only path that calls Claude, and it is
// server-side only.
type Handler struct {
	engine    *Engine
	replanner *Replanner
	logger    *logging.Logger
}

// NewHandler wires a Handler.
func NewHandler(engine *Engine, replanner *Replanner, logger *logging.Logger) *Handler {
	return &Handler{engine: engine, replanner: replanner, logger: logger}
}

// Register attaches routes wrapped in the auth middleware.
func (h *Handler) Register(r *httpx.Router, requireAuth httpx.Middleware) {
	r.Handle("POST /sessions/generate", requireAuth(http.HandlerFunc(h.generate)))
	r.Handle("GET /sessions/replan-check", requireAuth(http.HandlerFunc(h.replanCheck)))
}

func (h *Handler) generate(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	session, err := h.engine.Generate(r.Context(), uid)
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusOK, session)
	case errors.Is(err, ErrUnsafe):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "unsafe_session", "a safe session could not be produced; please review your injuries")
	default:
		h.logger.Error("session generation failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}

func (h *Handler) replanCheck(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	raw := r.URL.Query().Get("since")
	if raw == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "query parameter 'since' is required (RFC 3339)")
		return
	}
	since, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "'since' must be an RFC 3339 timestamp")
		return
	}
	decision, err := h.replanner.Check(r.Context(), uid, since)
	if err != nil {
		h.logger.Error("replan check failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, decision)
}
