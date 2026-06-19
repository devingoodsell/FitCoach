package injury

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

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

// Generate performs ONE server-side model call and returns the model's raw JSON
// output. injury defines this seam itself rather than importing
// coaching.Generator, because coaching already imports injury (importing it back
// would be a cycle). cmd/server adapts coaching's Generator to it at wiring time,
// so the Anthropic API key stays server-side (CLAUDE.md §2).
type Generate func(ctx context.Context, system string, prompt []byte, schema map[string]any) ([]byte, error)

// parseTimeout bounds the single server-side parse call so a slow model can't hang
// the request; on timeout (or any failure) the heuristic takes over.
const parseTimeout = 30 * time.Second

// parseSystemPrompt instructs the model to map a freeform description onto the
// structured slots — never to diagnose. It mirrors the heuristic's slot vocabulary
// so an LLM draft and a fallback draft are interchangeable for review.
const parseSystemPrompt = `You convert a user's freeform description of a physical issue into structured fields a fitness app will show the user to review before saving.

Output a SINGLE JSON object matching the provided schema. No prose, no markdown.

- region: the affected body part in lower_snake_case, prefixed left_ or right_ ONLY if the user states a side (e.g. "left_knee", "lower_back", "shoulder"). Use "" if no body part is identifiable.
- status: where the issue sits in its lifecycle — active_flare (currently hurting / acute), managed (rehabbing / under control), recurring_but_fine (comes and goes, not limiting now), resolved (past / recovered).
- severity: mild, moderate, or severe, inferred from how the user describes it.
- aggravating_movements: movements or exercises the user says make it worse, lower_snake_case (e.g. "squat", "overhead_press"); empty if none mentioned.
- low_confidence_fields: list every field above you are NOT confident about, so the user is prompted to verify it.

Do NOT diagnose, name conditions, or give medical or treatment advice — only structure what the user said. When unsure about a field, still fill your best guess AND add that field name to low_confidence_fields.`

// llmParser turns a freeform description into a structured Draft via one
// server-side Claude call (E7-PR2). When the model is unavailable or returns
// something unusable it falls back to the deterministic HeuristicParser, so the
// review-before-save flow always has a draft to show.
type llmParser struct {
	gen      Generate
	fallback Parser
	logger   *logging.Logger
}

// NewLLMParser wires an LLM-backed Parser. fallback defaults to the heuristic; a
// nil gen makes it behave exactly like the heuristic (useful when no key is set).
func NewLLMParser(gen Generate, fallback Parser, logger *logging.Logger) Parser {
	if fallback == nil {
		fallback = NewHeuristicParser()
	}
	return &llmParser{gen: gen, fallback: fallback, logger: logger}
}

// Parse asks the model for a structured draft, degrading to the heuristic on any
// failure. It satisfies Parser, so no caller changes (the seam is unchanged).
func (p *llmParser) Parse(text string) Draft {
	if p.gen == nil {
		return p.fallback.Parse(text)
	}
	ctx, cancel := context.WithTimeout(context.Background(), parseTimeout)
	defer cancel()

	raw, err := p.gen(ctx, parseSystemPrompt, []byte(text), parseSchema())
	if err != nil {
		p.warn("injury parse: model unavailable, using heuristic", "error", err.Error())
		return p.fallback.Parse(text)
	}
	draft, err := decodeParse(raw, text)
	if err != nil {
		p.warn("injury parse: unusable model output, using heuristic", "error", err.Error())
		return p.fallback.Parse(text)
	}
	return draft
}

func (p *llmParser) warn(msg string, args ...any) {
	if p.logger != nil {
		p.logger.Warn(msg, args...)
	}
}

// parseModel is the model-authored slice of a Draft. Server-set fields (notes from
// the original text) are added afterward and so are not in the schema.
type parseModel struct {
	Region               string   `json:"region"`
	Status               string   `json:"status"`
	Severity             string   `json:"severity"`
	AggravatingMovements []string `json:"aggravating_movements"`
	LowConfidenceFields  []string `json:"low_confidence_fields"`
}

// parseSchema constrains the model to the structured slots. It stays within the
// structured-output subset (additionalProperties:false, enums for closed sets).
func parseSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"region", "status", "severity", "aggravating_movements", "low_confidence_fields"},
		"properties": map[string]any{
			"region":                map[string]any{"type": "string"},
			"status":                map[string]any{"type": "string", "enum": []any{"active_flare", "managed", "recurring_but_fine", "resolved"}},
			"severity":              map[string]any{"type": "string", "enum": []any{"mild", "moderate", "severe"}},
			"aggravating_movements": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"low_confidence_fields": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
	}
}

// decodeParse maps model output onto a reviewable Draft. It guards every closed
// slot: a blank region or an out-of-enum status/severity is defaulted AND flagged
// low-confidence, so a weak parse surfaces for review rather than saving silently.
func decodeParse(raw []byte, original string) (Draft, error) {
	var m parseModel
	if err := json.Unmarshal(raw, &m); err != nil {
		return Draft{}, fmt.Errorf("decode parse: %w", err)
	}
	inj := Injury{
		Region:               strings.TrimSpace(m.Region),
		Status:               Status(strings.TrimSpace(m.Status)),
		Severity:             Severity(strings.TrimSpace(m.Severity)),
		AggravatingMovements: normalizeMovements(m.AggravatingMovements),
		Notes:                strings.TrimSpace(original),
	}
	low := append([]string(nil), m.LowConfidenceFields...)
	if inj.Region == "" {
		low = appendUnique(low, "region")
	}
	if !validStatus(inj.Status) {
		inj.Status = StatusActiveFlare
		low = appendUnique(low, "status")
	}
	if !validSeverity(inj.Severity) {
		inj.Severity = SeverityModerate
		low = appendUnique(low, "severity")
	}
	return Draft{Injury: inj, LowConfidenceFields: low}, nil
}

// normalizeMovements lowercases, snake-cases, de-duplicates, and canonicalizes a
// movement list (reusing normalizeMovement for the heuristic's aliases).
func normalizeMovements(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range in {
		m = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(m)), " ", "_")
		if m == "" {
			continue
		}
		m = normalizeMovement(m)
		if seen[m] {
			continue
		}
		seen[m] = true
		out = append(out, m)
	}
	return out
}

func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

// HeuristicParser is a deterministic keyword-based stand-in for the LLM parser.
// It remains the offline / no-key fallback for NewLLMParser, and keeps the
// review-before-save flow functional and testable without a model call.
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
