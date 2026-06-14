// Package disclaimer is the single source of truth for FitCoach's medical and
// health-data disclaimer copy (E13-S1). The version ties to the consent version
// recorded in E1, so we can prove which language a user accepted. The client
// fetches this rather than hardcoding copy, and Settings surfaces it (E14-S2).
package disclaimer

// Version is the current disclaimer/policy version. Bump when copy changes; this
// is the value stored against a user's consent acceptance (E1-S4).
const Version = "v1"

// Medical is the "guidance, not medical advice" language shown wherever the
// body/health is involved (onboarding, injury, readiness, diet).
const Medical = "FitCoach provides general fitness guidance, not medical advice. It is not a " +
	"substitute for professional diagnosis or treatment. Consult a qualified clinician before " +
	"starting a program or if you have pain, an injury, or a health condition."

// HealthData explains what wearable data is read and why, shown at the consent step.
const HealthData = "With your permission, FitCoach reads sleep, resting heart rate, and heart-rate " +
	"variability from Health Connect to estimate daily readiness and tailor your training. This data " +
	"is stored with your account and used only for coaching. You can decline and use manual mode, and " +
	"revoke access at any time."

// Document is the served disclaimer payload.
type Document struct {
	Version    string `json:"version"`
	Medical    string `json:"medical"`
	HealthData string `json:"health_data"`
}

// Current returns the current disclaimer document.
func Current() Document {
	return Document{Version: Version, Medical: Medical, HealthData: HealthData}
}
