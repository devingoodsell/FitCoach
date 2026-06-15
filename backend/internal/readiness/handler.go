package readiness

import (
	"errors"
	"net/http"

	"pro.d11l.fitcoach/backend/internal/auth"
	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// Handler exposes signal upload and readiness read. All routes require auth.
type Handler struct {
	svc    *Service
	logger *logging.Logger
}

// NewHandler wires a Handler.
func NewHandler(svc *Service, logger *logging.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// Register attaches routes wrapped in the auth middleware.
func (h *Handler) Register(r *httpx.Router, requireAuth httpx.Middleware) {
	r.Handle("POST /health/signals", requireAuth(http.HandlerFunc(h.ingest)))
	r.Handle("GET /readiness", requireAuth(http.HandlerFunc(h.today)))
}

type ingestBody struct {
	Samples []Sample `json:"samples"`
}

func (h *Handler) ingest(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var body ingestBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "malformed request body")
		return
	}
	err := h.svc.Ingest(r.Context(), uid, body.Samples)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ErrConsentRequired):
		httpx.WriteError(w, http.StatusForbidden, "consent_required", "health-data consent is required to upload signals")
	default:
		h.logger.Error("ingest signals failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}

func (h *Handler) today(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	score, err := h.svc.Today(r.Context(), uid)
	if err != nil {
		h.logger.Error("readiness today failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, score)
}
