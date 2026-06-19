package injury

import (
	"net/http"

	"pro.d11l.fitcoach/backend/internal/auth"
	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// AssistHandler exposes the identification-assist guided Q&A (E7-PR7). It is a
// separate handler so the assist's LLM dependency stays out of the core injury
// CRUD handler. The route requires auth; the model call is server-side only.
type AssistHandler struct {
	svc    *AssistService
	logger *logging.Logger
}

// NewAssistHandler wires an AssistHandler.
func NewAssistHandler(svc *AssistService, logger *logging.Logger) *AssistHandler {
	return &AssistHandler{svc: svc, logger: logger}
}

// Register attaches the assist route wrapped in the auth middleware.
func (h *AssistHandler) Register(r *httpx.Router, requireAuth httpx.Middleware) {
	r.Handle("POST /injuries/assist", requireAuth(http.HandlerFunc(h.assist)))
}

func (h *AssistHandler) assist(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.UserIDFromContext(r.Context()); !ok {
		unauthorized(w)
		return
	}
	var req AssistRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		badRequest(w, "malformed request body")
		return
	}
	resp, err := h.svc.Assist(r.Context(), req)
	if err != nil {
		h.logger.Error("injury assist failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}
