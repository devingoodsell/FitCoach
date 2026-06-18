package injury

import (
	"context"
	"errors"
	"testing"

	"pro.d11l.fitcoach/backend/internal/disclaimer"
)

// stubGen and equalStrings are shared test helpers (see helpers_test.go).

func TestAssistAsksThenProducesConfirmableDraft(t *testing.T) {
	tests := []struct {
		name         string
		answers      []QA
		modelJSON    string
		wantDone     bool
		wantQuestion string
		wantChoices  []string
		wantRegion   string
		wantStatus   Status
		wantSeverity Severity
		wantMoves    []string
	}{
		{
			name:         "first turn asks where it hurts",
			answers:      nil,
			modelJSON:    `{"done":false,"question":"Where do you feel it?","choices":["knee","shoulder","lower back"],"note":"","draft":{"region":"","status":"","severity":"","aggravating_movements":[],"notes":""}}`,
			wantDone:     false,
			wantQuestion: "Where do you feel it?",
			wantChoices:  []string{"knee", "shoulder", "lower back"},
		},
		{
			name: "final turn yields a structured draft",
			answers: []QA{
				{Question: "Where do you feel it?", Answer: "my right knee"},
				{Question: "When is it worst?", Answer: "when I squat"},
			},
			modelJSON:    `{"done":true,"question":"","choices":[],"note":"Recorded what you described.","draft":{"region":"right_knee","status":"active_flare","severity":"moderate","aggravating_movements":["squat"],"notes":"Right knee pain when squatting."}}`,
			wantDone:     true,
			wantRegion:   "right_knee",
			wantStatus:   StatusActiveFlare,
			wantSeverity: SeverityModerate,
			wantMoves:    []string{"squat"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotSchema map[string]any
			svc := NewAssistService(stubGen(tt.modelJSON, nil, &gotSchema), nil)

			resp, err := svc.Assist(context.Background(), AssistRequest{Answers: tt.answers})
			if err != nil {
				t.Fatalf("Assist: %v", err)
			}

			// The central E13 disclaimer is present on EVERY turn.
			if resp.Disclaimer != disclaimer.Medical {
				t.Errorf("disclaimer = %q, want the central medical disclaimer", resp.Disclaimer)
			}
			if len(gotSchema) == 0 {
				t.Error("assist should request structured output (schema)")
			}
			if resp.Done != tt.wantDone {
				t.Fatalf("done = %v, want %v", resp.Done, tt.wantDone)
			}

			if !tt.wantDone {
				if resp.Question != tt.wantQuestion {
					t.Errorf("question = %q, want %q", resp.Question, tt.wantQuestion)
				}
				if !equalStrings(resp.Choices, tt.wantChoices) {
					t.Errorf("choices = %v, want %v", resp.Choices, tt.wantChoices)
				}
				if resp.Draft != nil {
					t.Errorf("draft should be nil while still asking, got %+v", resp.Draft)
				}
				return
			}

			// Done turns must yield a draft the user can confirm and save.
			if resp.Draft == nil {
				t.Fatal("expected a draft on the final turn")
			}
			if resp.Draft.Injury.Region != tt.wantRegion {
				t.Errorf("region = %q, want %q", resp.Draft.Injury.Region, tt.wantRegion)
			}
			if resp.Draft.Injury.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", resp.Draft.Injury.Status, tt.wantStatus)
			}
			if resp.Draft.Injury.Severity != tt.wantSeverity {
				t.Errorf("severity = %q, want %q", resp.Draft.Injury.Severity, tt.wantSeverity)
			}
			if !equalStrings(resp.Draft.Injury.AggravatingMovements, tt.wantMoves) {
				t.Errorf("movements = %v, want %v", resp.Draft.Injury.AggravatingMovements, tt.wantMoves)
			}
		})
	}
}

func TestAssistDisclaimerPresentEvenWhenModelOmitsIt(t *testing.T) {
	// The model output carries no disclaimer; the server must stamp it anyway.
	out := `{"done":false,"question":"How long has it been bothering you?","choices":[],"note":"","draft":{"region":"","status":"","severity":"","aggravating_movements":[],"notes":""}}`
	svc := NewAssistService(stubGen(out, nil, nil), nil)

	resp, err := svc.Assist(context.Background(), AssistRequest{})
	if err != nil {
		t.Fatalf("Assist: %v", err)
	}
	if resp.Disclaimer != disclaimer.Medical {
		t.Errorf("disclaimer = %q, want it stamped server-side", resp.Disclaimer)
	}
}

func TestAssistForcesCompletionAtQuestionCap(t *testing.T) {
	// Model keeps wanting to ask more, but we are at the cap → force a draft so the
	// flow always converges to a confirmable entry.
	out := `{"done":false,"question":"And another thing?","choices":[],"note":"","draft":{"region":"lower_back","status":"managed","severity":"mild","aggravating_movements":["deadlift"],"notes":"ongoing"}}`
	svc := NewAssistService(stubGen(out, nil, nil), nil)

	answers := make([]QA, maxAssistQuestions)
	for i := range answers {
		answers[i] = QA{Question: "q", Answer: "a"}
	}

	resp, err := svc.Assist(context.Background(), AssistRequest{Answers: answers})
	if err != nil {
		t.Fatalf("Assist: %v", err)
	}
	if !resp.Done {
		t.Fatal("expected forced completion at the question cap")
	}
	if resp.Draft == nil || resp.Draft.Injury.Region != "lower_back" {
		t.Errorf("expected the best-effort draft, got %+v", resp.Draft)
	}
}

func TestAssistInvalidEnumsFlaggedLowConfidence(t *testing.T) {
	out := `{"done":true,"question":"","choices":[],"note":"","draft":{"region":"","status":"nope","severity":"awful","aggravating_movements":[],"notes":"unclear"}}`
	svc := NewAssistService(stubGen(out, nil, nil), nil)

	resp, err := svc.Assist(context.Background(), AssistRequest{Answers: []QA{{Question: "q", Answer: "a"}}})
	if err != nil {
		t.Fatalf("Assist: %v", err)
	}
	if resp.Draft == nil {
		t.Fatal("expected a draft")
	}
	low := map[string]bool{}
	for _, f := range resp.Draft.LowConfidenceFields {
		low[f] = true
	}
	for _, want := range []string{"region", "status", "severity"} {
		if !low[want] {
			t.Errorf("expected %q flagged low-confidence, got %v", want, resp.Draft.LowConfidenceFields)
		}
	}
	if resp.Draft.Injury.Status != StatusActiveFlare || resp.Draft.Injury.Severity != SeverityModerate {
		t.Errorf("invalid enums should default, got status=%q severity=%q", resp.Draft.Injury.Status, resp.Draft.Injury.Severity)
	}
}

func TestAssistModelErrorSurfaces(t *testing.T) {
	svc := NewAssistService(stubGen("", errors.New("model down"), nil), nil)

	if _, err := svc.Assist(context.Background(), AssistRequest{}); err == nil {
		t.Fatal("expected an error when the model call fails")
	}
}

func TestAssistWithoutSeamErrors(t *testing.T) {
	svc := NewAssistService(nil, nil)
	if _, err := svc.Assist(context.Background(), AssistRequest{}); err == nil {
		t.Fatal("expected an error with no model seam")
	}
}
