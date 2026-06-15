package location

import (
	"errors"
	"net/http"

	"pro.d11l.fitcoach/backend/internal/auth"
	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// Handler exposes the location endpoints. All routes require auth.
type Handler struct {
	svc    *Service
	logger *logging.Logger
}

// NewHandler wires a Handler.
func NewHandler(svc *Service, logger *logging.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// Register attaches location routes wrapped in the auth middleware.
func (h *Handler) Register(r *httpx.Router, requireAuth httpx.Middleware) {
	r.Handle("GET /locations", requireAuth(http.HandlerFunc(h.list)))
	r.Handle("POST /locations", requireAuth(http.HandlerFunc(h.create)))
	r.Handle("PUT /locations/{id}", requireAuth(http.HandlerFunc(h.update)))
	r.Handle("DELETE /locations/{id}", requireAuth(http.HandlerFunc(h.delete)))
	r.Handle("PUT /locations/current", requireAuth(http.HandlerFunc(h.setCurrent)))
}

type locationBody struct {
	Name      string   `json:"name"`
	Equipment []string `json:"equipment"`
}

type currentBody struct {
	LocationID string `json:"location_id"`
	Note       string `json:"note"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	doc, err := h.svc.Get(r.Context(), uid)
	if err != nil {
		h.fail(w, "list locations", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, doc)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	var body locationBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		badRequest(w, "malformed request body")
		return
	}
	loc, err := h.svc.Add(r.Context(), uid, body.Name, body.Equipment)
	if errors.Is(err, ErrInvalid) {
		badRequest(w, "name is required")
		return
	}
	if err != nil {
		h.fail(w, "create location", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, loc)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	var body locationBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		badRequest(w, "malformed request body")
		return
	}
	loc, err := h.svc.Update(r.Context(), uid, r.PathValue("id"), body.Name, body.Equipment)
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusOK, loc)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "location not found")
	case errors.Is(err, ErrInvalid):
		badRequest(w, "name is required")
	default:
		h.fail(w, "update location", err)
	}
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	err := h.svc.Delete(r.Context(), uid, r.PathValue("id"))
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "location not found")
	default:
		h.fail(w, "delete location", err)
	}
}

func (h *Handler) setCurrent(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	var body currentBody
	if err := httpx.DecodeJSON(r, &body); err != nil || body.LocationID == "" {
		badRequest(w, "location_id is required")
		return
	}
	current, err := h.svc.SetCurrent(r.Context(), uid, body.LocationID, body.Note)
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusOK, current)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "location not found")
	default:
		h.fail(w, "set current context", err)
	}
}

func (h *Handler) fail(w http.ResponseWriter, what string, err error) {
	h.logger.Error(what+" failed", "error", err.Error())
	httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
}

func unauthorized(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
}

func badRequest(w http.ResponseWriter, msg string) {
	httpx.WriteError(w, http.StatusBadRequest, "invalid_request", msg)
}
