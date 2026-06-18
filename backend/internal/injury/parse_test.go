package injury

import (
	"encoding/json"
	"errors"
	"testing"
)

func parseModelJSON(m parseModel) string {
	b, _ := json.Marshal(m)
	return string(b)
}

// TestLLMParserMapsRepresentativeInputsToSlots covers the E7-PR2 contract: a
// freeform description plus the model's structured output map onto the reviewable
// Draft slots, with uncertain/invalid fields flagged for the user to verify.
func TestLLMParserMapsRepresentativeInputsToSlots(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		model     parseModel
		region    string
		status    Status
		severity  Severity
		moves     []string
		lowFields []string
	}{
		{
			name:     "full structured parse",
			text:     "my left knee has a sharp pain when I squat and lunge",
			model:    parseModel{Region: "left_knee", Status: "active_flare", Severity: "severe", AggravatingMovements: []string{"Squat", "lunge"}},
			region:   "left_knee",
			status:   StatusActiveFlare,
			severity: SeveritySevere,
			moves:    []string{"squat", "lunge"},
		},
		{
			name:     "managed shoulder, no movements",
			text:     "rehabbing my right shoulder, mostly under control",
			model:    parseModel{Region: "right_shoulder", Status: "managed", Severity: "mild"},
			region:   "right_shoulder",
			status:   StatusManaged,
			severity: SeverityMild,
		},
		{
			name:      "blank region is defaulted and flagged",
			text:      "something just feels off",
			model:     parseModel{Region: "", Status: "active_flare", Severity: "moderate", LowConfidenceFields: []string{"severity"}},
			region:    "",
			status:    StatusActiveFlare,
			severity:  SeverityModerate,
			lowFields: []string{"region", "severity"},
		},
		{
			name:      "invalid enums are corrected and flagged",
			text:      "tweaked my back deadlifting",
			model:     parseModel{Region: "lower_back", Status: "bogus", Severity: "weird", AggravatingMovements: []string{"deadlift"}},
			region:    "lower_back",
			status:    StatusActiveFlare,
			severity:  SeverityModerate,
			moves:     []string{"deadlift"},
			lowFields: []string{"status", "severity"},
		},
		{
			name:     "movements de-duplicated and aliased",
			text:     "knee hurts when running",
			model:    parseModel{Region: "knee", Status: "managed", Severity: "moderate", AggravatingMovements: []string{"running", "Running", "run"}},
			region:   "knee",
			status:   StatusManaged,
			severity: SeverityModerate,
			moves:    []string{"run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotSchema map[string]any
			p := NewLLMParser(stubGen(parseModelJSON(tt.model), nil, &gotSchema), nil, nil)
			d := p.Parse(tt.text)

			if len(gotSchema) == 0 {
				t.Error("parse should request structured output (schema)")
			}
			if d.Injury.Region != tt.region {
				t.Errorf("region = %q, want %q", d.Injury.Region, tt.region)
			}
			if d.Injury.Status != tt.status {
				t.Errorf("status = %q, want %q", d.Injury.Status, tt.status)
			}
			if d.Injury.Severity != tt.severity {
				t.Errorf("severity = %q, want %q", d.Injury.Severity, tt.severity)
			}
			if !equalStrings(d.Injury.AggravatingMovements, tt.moves) {
				t.Errorf("movements = %v, want %v", d.Injury.AggravatingMovements, tt.moves)
			}
			// The user's own words are kept as the editable notes.
			if d.Injury.Notes != tt.text {
				t.Errorf("notes = %q, want the original text", d.Injury.Notes)
			}
			low := map[string]bool{}
			for _, f := range d.LowConfidenceFields {
				low[f] = true
			}
			for _, want := range tt.lowFields {
				if !low[want] {
					t.Errorf("expected %q flagged low-confidence, got %v", want, d.LowConfidenceFields)
				}
			}
		})
	}
}

func TestLLMParserFallsBackToHeuristicOnError(t *testing.T) {
	p := NewLLMParser(stubGen("", errors.New("claude down"), nil), NewHeuristicParser(), nil)
	d := p.Parse("my left knee has a sharp pain when I squat")
	if d.Injury.Region != "left_knee" {
		t.Errorf("region = %q, want left_knee from heuristic fallback", d.Injury.Region)
	}
	if d.Injury.Severity != SeveritySevere {
		t.Errorf("severity = %q, want severe from heuristic fallback", d.Injury.Severity)
	}
}

func TestLLMParserFallsBackOnMalformedOutput(t *testing.T) {
	p := NewLLMParser(stubGen("not json at all", nil, nil), NewHeuristicParser(), nil)
	d := p.Parse("right shoulder is sore overhead")
	if d.Injury.Region != "right_shoulder" {
		t.Errorf("region = %q, want right_shoulder from heuristic fallback", d.Injury.Region)
	}
}

func TestLLMParserNilSeamUsesHeuristic(t *testing.T) {
	p := NewLLMParser(nil, nil, nil)
	d := p.Parse("mild ankle pain")
	if d.Injury.Region != "ankle" {
		t.Errorf("region = %q, want ankle from heuristic", d.Injury.Region)
	}
}
