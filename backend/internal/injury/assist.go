package injury

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"pro.d11l.fitcoach/backend/internal/disclaimer"
	"pro.d11l.fitcoach/backend/internal/platform/logging"
)

// maxAssistQuestions bounds the guided Q&A so the assist always converges to a
// reviewable draft rather than interrogating the user indefinitely. Once the
// transcript reaches this many answered questions, the model is told to finish,
// and the server forces completion if it still wouldn't.
const maxAssistQuestions = 6

// assistTimeout bounds a single guided-assist model call (the user is waiting on
// the next question).
const assistTimeout = 25 * time.Second

// QA is one answered step of the identification assist transcript.
type QA struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// AssistRequest is the full transcript so far. The flow is stateless: the client
// replays every prior question/answer each turn, so nothing is persisted until
// the user confirms the resulting draft through the normal review-before-save
// injury entry.
type AssistRequest struct {
	Answers []QA `json:"answers"`
}

// AssistResponse is one turn of the guided Q&A. Disclaimer carries the central
// E13 "not a diagnosis, consult a clinician" language and is present on EVERY
// turn. When Done is true, Draft holds the structured result for the user to
// review/correct and save via the existing injury entry; otherwise Question (and
// optional Choices) is the next thing to ask.
type AssistResponse struct {
	Disclaimer string   `json:"disclaimer"`
	Done       bool     `json:"done"`
	Question   string   `json:"question,omitempty"`
	Choices    []string `json:"choices,omitempty"`
	Note       string   `json:"note,omitempty"`
	Draft      *Draft   `json:"draft,omitempty"`
}

// AssistService runs the LLM-driven identification assist (E7-PR7/E7-S5): a
// careful, non-diagnostic intake conversation that characterizes an unknown
// issue and ends in a structured draft. It reuses the SAME server-side Claude
// seam as the parser (the injury.Generate func) and never persists anything
// itself.
type AssistService struct {
	gen     Generate
	logger  *logging.Logger
	timeout time.Duration
	maxQ    int
}

// NewAssistService wires an AssistService. A nil gen makes Assist return an error
// (the flow is LLM-driven and has no heuristic fallback, unlike parsing).
func NewAssistService(gen Generate, logger *logging.Logger) *AssistService {
	return &AssistService{gen: gen, logger: logger, timeout: assistTimeout, maxQ: maxAssistQuestions}
}

// assistSystemPrompt frames the assistant as a careful intake helper that never
// diagnoses and always defers to a clinician — matching the disclaimer the server
// stamps on every response.
const assistSystemPrompt = `You are FitCoach's injury identification assistant. A person is unsure how to describe a physical issue. You ask short, plain-language questions ONE AT A TIME to characterize it well enough to record as a structured entry they will review.

Output a SINGLE JSON object matching the provided schema. No prose, no markdown.

You are NOT a clinician. You must NOT diagnose, name a medical condition, suggest what is wrong, or give treatment advice. You only gather and structure what the person tells you. If a question would require medical judgement, ask about observable facts (location, when it hurts, what movements aggravate it, how long it has been going on) instead.

Each turn, decide:
- If you need more information, set done=false and ask exactly one question. Provide a few choices when it helps (e.g. body areas, severity wording); leave choices empty for open questions.
- If you have enough to record a draft, set done=true and fill draft with: region (snake_case, with left_/right_ only if stated), status (active_flare|managed|recurring_but_fine|resolved), severity (mild|moderate|severe), aggravating_movements (snake_case), and notes summarizing what they said in their own framing. Leave question empty when done.

Keep it brief: aim to finish within a handful of questions. Use the schema's enum values exactly.`

// assistTurn is the model's structured output for one turn.
type assistTurn struct {
	Done     bool     `json:"done"`
	Question string   `json:"question"`
	Choices  []string `json:"choices"`
	Note     string   `json:"note"`
	Draft    struct {
		Region               string   `json:"region"`
		Status               string   `json:"status"`
		Severity             string   `json:"severity"`
		AggravatingMovements []string `json:"aggravating_movements"`
		Notes                string   `json:"notes"`
	} `json:"draft"`
}

// assistPromptPayload is the user-turn the model sees: the transcript so far and
// how many questions remain before it must finish.
type assistPromptPayload struct {
	Answers            []QA `json:"answers"`
	QuestionsAsked     int  `json:"questions_asked"`
	MustFinishNow      bool `json:"must_finish_now"`
	QuestionsRemaining int  `json:"questions_remaining"`
}

// Assist produces the next turn of the guided Q&A. The disclaimer is stamped
// server-side on every response (never sourced from the model), guaranteeing the
// safety framing is always present (E13-S1). A model failure surfaces as an error
// the handler maps to 5xx.
func (s *AssistService) Assist(ctx context.Context, req AssistRequest) (AssistResponse, error) {
	if s.gen == nil {
		return AssistResponse{}, fmt.Errorf("assist: no model seam configured")
	}
	cctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	asked := len(req.Answers)
	mustFinish := asked >= s.maxQ
	remaining := s.maxQ - asked
	if remaining < 0 {
		remaining = 0
	}
	prompt, err := json.Marshal(assistPromptPayload{
		Answers:            req.Answers,
		QuestionsAsked:     asked,
		MustFinishNow:      mustFinish,
		QuestionsRemaining: remaining,
	})
	if err != nil {
		return AssistResponse{}, fmt.Errorf("assist: marshal prompt: %w", err)
	}

	out, err := s.gen(cctx, assistSystemPrompt, prompt, assistSchema())
	if err != nil {
		return AssistResponse{}, fmt.Errorf("assist: model call: %w", err)
	}

	var turn assistTurn
	if err := json.Unmarshal(out, &turn); err != nil {
		return AssistResponse{}, fmt.Errorf("assist: decode model output: %w", err)
	}

	// The disclaimer is ALWAYS present, regardless of model output (E13-S1).
	resp := AssistResponse{Disclaimer: disclaimer.Medical, Note: strings.TrimSpace(turn.Note)}

	// Force completion once we have run out of question budget so the flow always
	// terminates in a reviewable draft.
	done := turn.Done || mustFinish

	if done {
		resp.Done = true
		resp.Draft = buildAssistDraft(turn)
		return resp, nil
	}

	q := strings.TrimSpace(turn.Question)
	if q == "" {
		// Model wants to continue but gave no question — finish rather than stall.
		resp.Done = true
		resp.Draft = buildAssistDraft(turn)
		return resp, nil
	}
	resp.Question = q
	resp.Choices = cleanChoices(turn.Choices)
	return resp, nil
}

// buildAssistDraft converts the model's final draft into a reviewable Draft,
// enforcing the status/severity enums and flagging anything uncertain so the
// existing review-before-save form prompts the user to check it.
func buildAssistDraft(turn assistTurn) *Draft {
	var low []string
	inj := Injury{}
	inj.Region = strings.TrimSpace(turn.Draft.Region)
	if inj.Region == "" {
		low = append(low, "region")
	}
	if validStatus(Status(turn.Draft.Status)) {
		inj.Status = Status(turn.Draft.Status)
	} else {
		inj.Status = StatusActiveFlare
		low = append(low, "status")
	}
	if validSeverity(Severity(turn.Draft.Severity)) {
		inj.Severity = Severity(turn.Draft.Severity)
	} else {
		inj.Severity = SeverityModerate
		low = append(low, "severity")
	}
	seen := map[string]bool{}
	for _, m := range turn.Draft.AggravatingMovements {
		m = strings.ToLower(strings.TrimSpace(m))
		if m != "" && !seen[m] {
			seen[m] = true
			inj.AggravatingMovements = append(inj.AggravatingMovements, m)
		}
	}
	inj.Notes = strings.TrimSpace(turn.Draft.Notes)
	return &Draft{Injury: inj, LowConfidenceFields: low}
}

func cleanChoices(in []string) []string {
	var out []string
	for _, c := range in {
		if c = strings.TrimSpace(c); c != "" {
			out = append(out, c)
		}
	}
	return out
}

// assistSchema constrains the model to one well-formed turn. It stays within the
// structured-output subset: additionalProperties:false everywhere, enum-bounded
// status/severity, no numeric bounds.
func assistSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"done", "question", "choices", "note", "draft"},
		"properties": map[string]any{
			"done":     map[string]any{"type": "boolean"},
			"question": map[string]any{"type": "string"},
			"choices":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"note":     map[string]any{"type": "string"},
			"draft": map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []any{"region", "status", "severity", "aggravating_movements", "notes"},
				"properties": map[string]any{
					"region":                map[string]any{"type": "string"},
					"status":                map[string]any{"type": "string", "enum": []any{"active_flare", "managed", "recurring_but_fine", "resolved"}},
					"severity":              map[string]any{"type": "string", "enum": []any{"mild", "moderate", "severe"}},
					"aggravating_movements": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"notes":                 map[string]any{"type": "string"},
				},
			},
		},
	}
}
