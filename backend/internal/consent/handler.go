package consent

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/auth"
	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// store is the persistence surface the handler needs (consumer-defined so it is
// testable with a fake).
type store interface {
	Record(ctx context.Context, userID uuid.UUID, ctype, version string, now time.Time) error
	List(ctx context.Context, userID uuid.UUID) ([]Consent, error)
	Revoke(ctx context.Context, userID uuid.UUID, ctype string, now time.Time) error
}

// Handler exposes the consent endpoints. Routes are protected by RequireAuth.
type Handler struct {
	store  store
	logger *logging.Logger
	now    func() time.Time
}

// NewHandler wires a Handler. now defaults to time.Now (UTC) when nil.
func NewHandler(s store, logger *logging.Logger, now func() time.Time) *Handler {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Handler{store: s, logger: logger, now: now}
}

// Register attaches consent routes, each wrapped in the auth middleware.
func (h *Handler) Register(r *httpx.Router, requireAuth httpx.Middleware) {
	r.Handle("GET /consent", requireAuth(http.HandlerFunc(h.list)))
	r.Handle("POST /consent", requireAuth(http.HandlerFunc(h.record)))
	r.Handle("POST /consent/{type}/revoke", requireAuth(http.HandlerFunc(h.revoke)))
}

type listResponse struct {
	Consents []Consent `json:"consents"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	consents, err := h.store.List(r.Context(), userID)
	if err != nil {
		h.logger.Error("list consents failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	if consents == nil {
		consents = []Consent{}
	}
	httpx.WriteJSON(w, http.StatusOK, listResponse{Consents: consents})
}

type recordRequest struct {
	Type    string `json:"type"`
	Version string `json:"version"`
}

func (h *Handler) record(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var req recordRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "malformed request body")
		return
	}
	if !IsValidType(req.Type) || req.Version == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "unknown consent type or missing version")
		return
	}

	now := h.now()
	if err := h.store.Record(r.Context(), userID, req.Type, req.Version, now); err != nil {
		h.logger.Error("record consent failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, Consent{Type: req.Type, Version: req.Version, AcceptedAt: now})
}

// revoke withdraws an in-force consent (E14-S2). Idempotent: revoking an absent or
// already-revoked consent still returns 200 with the current state. For health_data
// this flips HasConsent to false, so readiness ingestion (E4) reverts to manual mode.
func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	ctype := r.PathValue("type")
	if !IsValidType(ctype) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "unknown consent type")
		return
	}

	now := h.now()
	if err := h.store.Revoke(r.Context(), userID, ctype, now); err != nil {
		h.logger.Error("revoke consent failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}

	// Report the resulting state so the client reflects it without a refetch.
	consents, err := h.store.List(r.Context(), userID)
	if err != nil {
		h.logger.Error("list after revoke failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	for _, c := range consents {
		if c.Type == ctype {
			httpx.WriteJSON(w, http.StatusOK, c)
			return
		}
	}
	// Never consented to this type: nothing to revoke, report it as revoked-now.
	httpx.WriteJSON(w, http.StatusOK, Consent{Type: ctype, RevokedAt: &now})
}
