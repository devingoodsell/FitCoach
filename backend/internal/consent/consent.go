// Package consent records a user's acceptance of health-data and medical
// disclaimer versions (E1-S4). It stores an append-only log; the latest entry
// per type is the current state, queried to gate Health Connect reads (E4).
package consent

import "time"

// Allowed consent types. health_data gates wearable ingestion; medical_disclaimer
// records acknowledgement of the "guidance, not medical advice" language (E13).
const (
	TypeHealthData        = "health_data"
	TypeMedicalDisclaimer = "medical_disclaimer"
)

// IsValidType reports whether t is a recognized consent type.
func IsValidType(t string) bool {
	return t == TypeHealthData || t == TypeMedicalDisclaimer
}

// Consent is a single recorded acceptance.
type Consent struct {
	Type       string    `json:"type"`
	Version    string    `json:"version"`
	AcceptedAt time.Time `json:"accepted_at"`
}
