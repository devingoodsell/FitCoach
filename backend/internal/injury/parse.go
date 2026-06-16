package injury

import "strings"

// Draft is a best-effort structured parse of a freeform description, returned for
// the user to review/correct before saving (E7-S1). LowConfidenceFields names the
// slots the parser was unsure about.
type Draft struct {
	Injury              Injury   `json:"injury"`
	LowConfidenceFields []string `json:"low_confidence_fields,omitempty"`
}

// Parser turns a freeform injury description into a reviewable Draft.
type Parser interface {
	Parse(text string) Draft
}

// HeuristicParser is a deterministic keyword-based stand-in for the LLM parser.
// E5 will provide an LLM-backed Parser (server-side, via the E15 seam); this keeps
// the flow functional and testable until then.
type HeuristicParser struct{}

// NewHeuristicParser returns a HeuristicParser.
func NewHeuristicParser() HeuristicParser { return HeuristicParser{} }

var regionKeywords = map[string]string{
	"knee": "knee", "back": "lower_back", "lower back": "lower_back", "shoulder": "shoulder",
	"elbow": "elbow", "wrist": "wrist", "hip": "hip", "ankle": "ankle", "neck": "neck",
	"hamstring": "hamstring",
}

var movementKeywords = []string{
	"squat", "deadlift", "lunge", "press", "row", "run", "running", "jump", "curl", "pushup", "bench",
}

// Parse extracts region, side, severity, status, and aggravating movements.
func (HeuristicParser) Parse(text string) Draft {
	lower := strings.ToLower(text)
	var lowConfidence []string
	inj := Injury{}

	// region (+ side)
	region := ""
	for kw, canonical := range regionKeywords {
		if strings.Contains(lower, kw) {
			region = canonical
			break
		}
	}
	if region == "" {
		lowConfidence = append(lowConfidence, "region")
	} else {
		switch {
		case strings.Contains(lower, "left"):
			region = "left_" + region
		case strings.Contains(lower, "right"):
			region = "right_" + region
		}
	}
	inj.Region = region

	// severity
	switch {
	case containsAny(lower, "severe", "sharp", "intense", "really bad", "can't"):
		inj.Severity = SeveritySevere
	case containsAny(lower, "mild", "slight", "minor", "a little"):
		inj.Severity = SeverityMild
	default:
		inj.Severity = SeverityModerate
		lowConfidence = append(lowConfidence, "severity")
	}

	// status
	switch {
	case containsAny(lower, "flare", "acute", "just hurt", "today"):
		inj.Status = StatusActiveFlare
	case containsAny(lower, "managing", "manageable", "managed", "rehab"):
		inj.Status = StatusManaged
	case containsAny(lower, "used to", "old injury", "fine now", "recovered", "resolved"):
		inj.Status = StatusResolved
	default:
		inj.Status = StatusActiveFlare
		lowConfidence = append(lowConfidence, "status")
	}

	// aggravating movements
	seen := map[string]bool{}
	for _, m := range movementKeywords {
		if strings.Contains(lower, m) && !seen[m] {
			seen[m] = true
			inj.AggravatingMovements = append(inj.AggravatingMovements, normalizeMovement(m))
		}
	}

	inj.Notes = strings.TrimSpace(text)
	return Draft{Injury: inj, LowConfidenceFields: lowConfidence}
}

func normalizeMovement(m string) string {
	switch m {
	case "running":
		return "run"
	case "bench":
		return "bench_press"
	case "press":
		return "overhead_press"
	case "row":
		return "barbell_row"
	}
	return m
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
