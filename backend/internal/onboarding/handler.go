package onboarding

import (
	"errors"
	"net/http"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/auth"
	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// Handler exposes the onboarding write endpoints. All routes require auth and
// persist into the caller's Coach Memory.
type Handler struct {
	svc    *Service
	logger *logging.Logger
}

// NewHandler wires a Handler.
func NewHandler(svc *Service, logger *logging.Logger) *Handler {
	return &Handler{svc: svc, logger: logger}
}

// Register attaches onboarding routes, each wrapped in the auth middleware.
func (h *Handler) Register(r *httpx.Router, requireAuth httpx.Middleware) {
	r.Handle("PUT /onboarding/profile", requireAuth(http.HandlerFunc(h.profile)))
	r.Handle("PUT /onboarding/goals", requireAuth(http.HandlerFunc(h.goals)))
	r.Handle("PUT /onboarding/schedule", requireAuth(http.HandlerFunc(h.schedule)))
	r.Handle("PUT /onboarding/diet", requireAuth(http.HandlerFunc(h.diet)))
	r.Handle("PUT /onboarding/preferences", requireAuth(http.HandlerFunc(h.preferences)))
}

// save is the shared decode -> service -> respond flow for a section.
func save[T any](h *Handler, w http.ResponseWriter, r *http.Request, fn func(userID uuid.UUID, in T) (any, error)) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var in T
	if err := httpx.DecodeJSON(r, &in); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "malformed request body")
		return
	}
	out, err := fn(userID, in)
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusOK, out)
	case isValidation(err):
		var ve *ValidationError
		errors.As(err, &ve)
		httpx.WriteJSON(w, http.StatusBadRequest, validationBody(ve))
	default:
		h.logger.Error("onboarding save failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}

func (h *Handler) profile(w http.ResponseWriter, r *http.Request) {
	save(h, w, r, func(userID uuid.UUID, in Profile) (any, error) {
		return h.svc.SaveProfile(r.Context(), userID, in)
	})
}

func (h *Handler) goals(w http.ResponseWriter, r *http.Request) {
	save(h, w, r, func(userID uuid.UUID, in GoalWeights) (any, error) {
		return h.svc.SaveGoals(r.Context(), userID, in)
	})
}

func (h *Handler) schedule(w http.ResponseWriter, r *http.Request) {
	save(h, w, r, func(userID uuid.UUID, in Schedule) (any, error) {
		return h.svc.SaveSchedule(r.Context(), userID, in)
	})
}

func (h *Handler) diet(w http.ResponseWriter, r *http.Request) {
	save(h, w, r, func(userID uuid.UUID, in DietPrefs) (any, error) {
		return h.svc.SaveDiet(r.Context(), userID, in)
	})
}

func (h *Handler) preferences(w http.ResponseWriter, r *http.Request) {
	save(h, w, r, func(userID uuid.UUID, in Preferences) (any, error) {
		return h.svc.SavePreferences(r.Context(), userID, in)
	})
}

func isValidation(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

type validationResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields"`
}

func validationBody(ve *ValidationError) validationResponse {
	return validationResponse{Error: "validation_failed", Message: "some fields are invalid", Fields: ve.Fields}
}
