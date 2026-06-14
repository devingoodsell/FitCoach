package disclaimer

import (
	"net/http"

	"pro.d11l.fitcoach/backend/internal/platform/httpx"
)

// Handler serves the disclaimer document. The route is public: it is needed at
// onboarding and the consent step, before a session exists.
type Handler struct{}

// NewHandler returns a Handler.
func NewHandler() *Handler { return &Handler{} }

// Register attaches the disclaimer route.
func (h *Handler) Register(r *httpx.Router) {
	r.HandleFunc("GET /disclaimers", h.get)
}

func (h *Handler) get(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, Current())
}
