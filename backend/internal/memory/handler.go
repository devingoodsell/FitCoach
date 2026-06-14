package memory

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/auth"
	"pro.d11l.fitcoach/backend/internal/platform/httpx"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// memoryStore is the persistence surface the handler needs (consumer-defined for
// testability).
type memoryStore interface {
	GetAll(ctx context.Context, userID uuid.UUID) ([]SectionRecord, error)
	GetSection(ctx context.Context, userID uuid.UUID, section Section) (SectionRecord, error)
	PutSection(ctx context.Context, userID uuid.UUID, section Section, data json.RawMessage) (SectionRecord, error)
	RecordWorkout(ctx context.Context, userID uuid.UUID, clientSessionID string, data json.RawMessage, performedAt time.Time) (WorkoutLog, error)
	RecentWorkouts(ctx context.Context, userID uuid.UUID, limit int) ([]WorkoutLog, error)
}

// Handler exposes the Coach Memory HTTP surface. All routes require auth.
type Handler struct {
	store  memoryStore
	logger *logging.Logger
}

// NewHandler wires a Handler.
func NewHandler(s memoryStore, logger *logging.Logger) *Handler {
	return &Handler{store: s, logger: logger}
}

// Register attaches memory routes, each wrapped in the auth middleware.
func (h *Handler) Register(r *httpx.Router, requireAuth httpx.Middleware) {
	r.Handle("GET /memory", requireAuth(http.HandlerFunc(h.getAll)))
	r.Handle("GET /memory/{section}", requireAuth(http.HandlerFunc(h.getSection)))
	r.Handle("PUT /memory/{section}", requireAuth(http.HandlerFunc(h.putSection)))
	r.Handle("GET /workouts", requireAuth(http.HandlerFunc(h.listWorkouts)))
	r.Handle("POST /workouts", requireAuth(http.HandlerFunc(h.recordWorkout)))
}

func userID(r *http.Request) (uuid.UUID, bool) {
	return auth.UserIDFromContext(r.Context())
}

type sectionsResponse struct {
	Sections []SectionRecord `json:"sections"`
}

func (h *Handler) getAll(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	sections, err := h.store.GetAll(r.Context(), uid)
	if err != nil {
		h.logger.Error("get all memory failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	if sections == nil {
		sections = []SectionRecord{}
	}
	httpx.WriteJSON(w, http.StatusOK, sectionsResponse{Sections: sections})
}

func (h *Handler) getSection(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	section := Section(r.PathValue("section"))
	rec, err := h.store.GetSection(r.Context(), uid, section)
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusOK, rec)
	case errors.Is(err, ErrUnknownSection):
		httpx.WriteError(w, http.StatusBadRequest, "unknown_section", "unknown memory section")
	case errors.Is(err, ErrSectionNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "section not set")
	default:
		h.logger.Error("get section failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
	}
}

type putSectionBody struct {
	Data json.RawMessage `json:"data"`
}

func (h *Handler) putSection(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	section := Section(r.PathValue("section"))
	var body putSectionBody
	if err := httpx.DecodeJSON(r, &body); err != nil || len(body.Data) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "data object is required")
		return
	}
	rec, err := h.store.PutSection(r.Context(), uid, section, body.Data)
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusOK, rec)
	case errors.Is(err, ErrUnknownSection):
		httpx.WriteError(w, http.StatusBadRequest, "unknown_section", "unknown memory section")
	default:
		h.logger.Error("put section failed", "error", err.Error())
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "could not store section")
	}
}

type workoutsResponse struct {
	Workouts []WorkoutLog `json:"workouts"`
}

func (h *Handler) listWorkouts(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	limit := 0
	if q := r.URL.Query().Get("limit"); q != "" {
		limit, _ = strconv.Atoi(q)
	}
	workouts, err := h.store.RecentWorkouts(r.Context(), uid, limit)
	if err != nil {
		h.logger.Error("list workouts failed", "error", err.Error())
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "something went wrong")
		return
	}
	if workouts == nil {
		workouts = []WorkoutLog{}
	}
	httpx.WriteJSON(w, http.StatusOK, workoutsResponse{Workouts: workouts})
}

type recordWorkoutBody struct {
	ClientSessionID string          `json:"client_session_id"`
	PerformedAt     time.Time       `json:"performed_at"`
	Data            json.RawMessage `json:"data"`
}

func (h *Handler) recordWorkout(w http.ResponseWriter, r *http.Request) {
	uid, ok := userID(r)
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	var body recordWorkoutBody
	if err := httpx.DecodeJSON(r, &body); err != nil || body.ClientSessionID == "" || len(body.Data) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "client_session_id and data are required")
		return
	}
	if body.PerformedAt.IsZero() {
		body.PerformedAt = time.Now().UTC()
	}
	log, err := h.store.RecordWorkout(r.Context(), uid, body.ClientSessionID, body.Data, body.PerformedAt)
	if err != nil {
		h.logger.Error("record workout failed", "error", err.Error())
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "could not record session")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, log)
}
