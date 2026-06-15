package diet

import (
	"net/http"

	"pro.d11l.fitcoach/backend/internal/auth"
	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// Handler exposes the diet endpoints. All routes require auth.
type Handler struct {
	svc    *Service
	logger *logging.Logger
}

// NewHandler wires a Handler.
func NewHandler(svc *Service, logger *logging.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// Register attaches diet routes wrapped in the auth middleware.
func (h *Handler) Register(r *httpx.Router, requireAuth httpx.Middleware) {
	r.Handle("GET /diet/targets", requireAuth(http.HandlerFunc(h.targets)))
	r.Handle("GET /diet/post-workout-note", requireAuth(http.HandlerFunc(h.note)))
}

func (h *Handler) targets(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	result, err := h.svc.Targets(r.Context(), uid)
	if err != nil {
		h.logger.Error("diet targets failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) note(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	// Default to a non-heavy note unless the client marks the session heavy.
	heavy := r.URL.Query().Get("intensity") == "heavy"
	result, err := h.svc.PostWorkoutNote(r.Context(), uid, heavy)
	if err != nil {
		h.logger.Error("diet note failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}
