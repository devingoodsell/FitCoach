package injury

import (
	"errors"
	"net/http"

	"pro.d11l.fitcoach/backend/internal/auth"
	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// Handler exposes the injury endpoints. All routes require auth.
type Handler struct {
	svc    *Service
	logger *logging.Logger
}

// NewHandler wires a Handler.
func NewHandler(svc *Service, logger *logging.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// Register attaches injury routes wrapped in the auth middleware.
func (h *Handler) Register(r *httpx.Router, requireAuth httpx.Middleware) {
	r.Handle("GET /injuries", requireAuth(http.HandlerFunc(h.list)))
	r.Handle("POST /injuries", requireAuth(http.HandlerFunc(h.create)))
	r.Handle("PUT /injuries/{id}", requireAuth(http.HandlerFunc(h.update)))
	r.Handle("DELETE /injuries/{id}", requireAuth(http.HandlerFunc(h.delete)))
	r.Handle("POST /injuries/parse", requireAuth(http.HandlerFunc(h.parse)))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	doc, err := h.svc.Get(r.Context(), uid)
	if err != nil {
		h.fail(w, "list injuries", err)
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
	var in Injury
	if err := httpx.DecodeJSON(r, &in); err != nil {
		badRequest(w, "malformed request body")
		return
	}
	created, err := h.svc.Add(r.Context(), uid, in)
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusCreated, created)
	case errors.Is(err, ErrInvalid):
		badRequest(w, err.Error())
	default:
		h.fail(w, "create injury", err)
	}
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		unauthorized(w)
		return
	}
	var in Injury
	if err := httpx.DecodeJSON(r, &in); err != nil {
		badRequest(w, "malformed request body")
		return
	}
	updated, err := h.svc.Update(r.Context(), uid, r.PathValue("id"), in)
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusOK, updated)
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "injury not found")
	case errors.Is(err, ErrInvalid):
		badRequest(w, err.Error())
	default:
		h.fail(w, "update injury", err)
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
		httpx.WriteError(w, http.StatusNotFound, "not_found", "injury not found")
	default:
		h.fail(w, "delete injury", err)
	}
}

type parseBody struct {
	Text string `json:"text"`
}

func (h *Handler) parse(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserIDFromContext(r.Context()); !ok {
		unauthorized(w)
		return
	}
	var body parseBody
	if err := httpx.DecodeJSON(r, &body); err != nil || body.Text == "" {
		badRequest(w, "text is required")
		return
	}
	// Returns a draft for the user to review/correct before saving (E7-S1).
	httpx.WriteJSON(w, http.StatusOK, h.svc.ParseDraft(body.Text))
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
