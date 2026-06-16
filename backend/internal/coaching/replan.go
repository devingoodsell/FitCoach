package coaching

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pro.d11l.fitcoach/backend/internal/injury"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
	"pro.d11l.fitcoach/backend/internal/readiness"
)

// Re-plan reason codes (E5-S5). Mirrors the ReplanCheck.reasons enum in the API.
const (
	ReasonInjuryChanged  = "injury_changed"
	ReasonContextChanged = "context_changed"
	ReasonPoorRecovery   = "poor_recovery"
)

// injuryDocProvider exposes the injuries document so the checker can read its
// last-changed timestamp (consumer-defined; injury.Service satisfies it).
type injuryDocProvider interface {
	Get(ctx context.Context, userID uuid.UUID) (injury.Doc, error)
}

// ReplanDecision tells the client whether to regenerate its cached session.
type ReplanDecision struct {
	ReplanNeeded bool     `json:"replan_needed"`
	Reasons      []string `json:"reasons"`
}

// Replanner answers, deterministically and WITHOUT a Claude call, whether the
// inputs behind a client's cached session have materially changed since it was
// generated (E5-S5): a new/changed injury, a current-context (location) change,
// or a poor-recovery morning. Otherwise the client keeps its offline cache and
// re-plans on the next session by default.
type Replanner struct {
	injuries  injuryDocProvider
	locations locationProvider
	readiness readinessProvider
	logger    *logging.Logger
}

// NewReplanner wires a Replanner.
func NewReplanner(injuries injuryDocProvider, locations locationProvider, readinessSvc readinessProvider, logger *logging.Logger) *Replanner {
	return &Replanner{injuries: injuries, locations: locations, readiness: readinessSvc, logger: logger}
}

// Check reports whether a session cached at `since` should be regenerated.
// Injuries are safety-critical so a read failure aborts; location and readiness
// degrade gracefully (a missing signal simply doesn't add its trigger).
func (r *Replanner) Check(ctx context.Context, userID uuid.UUID, since time.Time) (ReplanDecision, error) {
	var reasons []string

	doc, err := r.injuries.Get(ctx, userID)
	if err != nil {
		return ReplanDecision{}, err
	}
	if doc.ChangedAt != nil && doc.ChangedAt.After(since) {
		reasons = append(reasons, ReasonInjuryChanged)
	}

	if loc, err := r.locations.Get(ctx, userID); err != nil {
		r.warn(ctx, "replan: location unavailable", "error", err.Error())
	} else if loc.Current != nil && loc.Current.ChangedAt.After(since) {
		reasons = append(reasons, ReasonContextChanged)
	}

	if score, err := r.readiness.Today(ctx, userID); err != nil {
		r.warn(ctx, "replan: readiness unavailable", "error", err.Error())
	} else if readiness.PoorRecovery(score) {
		reasons = append(reasons, ReasonPoorRecovery)
	}

	return ReplanDecision{ReplanNeeded: len(reasons) > 0, Reasons: reasons}, nil
}

func (r *Replanner) warn(ctx context.Context, msg string, args ...any) {
	if r.logger != nil {
		r.logger.WarnContext(ctx, msg, args...)
	}
}
