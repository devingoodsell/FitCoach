package readiness

import (
	"strings"
	"testing"
)

func baseline(v float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = v
	}
	return out
}

func TestComputeNeutralWhenOnBaseline(t *testing.T) {
	in := Inputs{
		HRV:   Metric{Today: 60, HasToday: true, Baseline: baseline(60, 7)},
		RHR:   Metric{Today: 55, HasToday: true, Baseline: baseline(55, 7)},
		Sleep: Metric{Today: 450, HasToday: true, Baseline: baseline(450, 7)},
	}
	got := Compute(in)
	if got.Value != 50 {
		t.Fatalf("value = %d, want 50 (on baseline)", got.Value)
	}
	if got.Confidence != ConfidenceHigh {
		t.Errorf("confidence = %s, want high", got.Confidence)
	}
	if len(got.Drivers) != 0 {
		t.Errorf("expected no drivers on baseline, got %v", got.Drivers)
	}
}

func TestComputePoorRecoveryLowersScore(t *testing.T) {
	// HRV well below baseline, RHR above, short sleep → low score with drivers.
	in := Inputs{
		HRV:   Metric{Today: 40, HasToday: true, Baseline: []float64{60, 62, 58, 61, 59}},
		RHR:   Metric{Today: 65, HasToday: true, Baseline: []float64{54, 55, 56, 55, 54}},
		Sleep: Metric{Today: 360, HasToday: true, Baseline: []float64{460, 470, 455, 465, 450}},
	}
	got := Compute(in)
	if got.Value >= 45 {
		t.Fatalf("value = %d, want clearly below 45", got.Value)
	}
	wantDrivers := map[string]bool{driverHRVLow: false, driverRHRHigh: false, driverSleepShort: false}
	for _, d := range got.Drivers {
		if _, ok := wantDrivers[d]; ok {
			wantDrivers[d] = true
		}
	}
	for d, seen := range wantDrivers {
		if !seen {
			t.Errorf("expected driver %q in %v", d, got.Drivers)
		}
	}
}

func TestComputeGoodRecoveryRaisesScore(t *testing.T) {
	in := Inputs{
		HRV:   Metric{Today: 80, HasToday: true, Baseline: []float64{60, 62, 58, 61, 59}},
		RHR:   Metric{Today: 48, HasToday: true, Baseline: []float64{54, 55, 56, 55, 54}},
		Sleep: Metric{Today: 520, HasToday: true, Baseline: []float64{460, 470, 455, 465, 450}},
	}
	got := Compute(in)
	if got.Value <= 55 {
		t.Fatalf("value = %d, want clearly above 55", got.Value)
	}
}

func TestComputeMissingDataFlagsLowConfidence(t *testing.T) {
	// Only HRV present → medium? No: only one usable metric → low confidence.
	in := Inputs{
		HRV: Metric{Today: 60, HasToday: true, Baseline: baseline(60, 7)},
	}
	got := Compute(in)
	if got.Confidence != ConfidenceLow {
		t.Errorf("confidence = %s, want low (one metric)", got.Confidence)
	}

	// No usable metrics at all → neutral 50, low confidence, no crash.
	empty := Compute(Inputs{})
	if empty.Value != 50 || empty.Confidence != ConfidenceLow {
		t.Errorf("empty inputs = %+v, want neutral/low", empty)
	}
}

func TestComputeInsufficientBaselineNotUsable(t *testing.T) {
	in := Inputs{
		HRV:   Metric{Today: 60, HasToday: true, Baseline: []float64{60, 60}}, // < minBaseline
		RHR:   Metric{Today: 55, HasToday: true, Baseline: baseline(55, 5)},
		Sleep: Metric{Today: 450, HasToday: true, Baseline: baseline(450, 5)},
	}
	got := Compute(in)
	if got.Confidence != ConfidenceMedium {
		t.Errorf("confidence = %s, want medium (HRV baseline too short)", got.Confidence)
	}
}

func TestExplanationHasNoMedicalClaims(t *testing.T) {
	in := Inputs{
		HRV:   Metric{Today: 40, HasToday: true, Baseline: []float64{60, 62, 58, 61, 59}},
		RHR:   Metric{Today: 65, HasToday: true, Baseline: []float64{54, 55, 56, 55, 54}},
		Sleep: Metric{Today: 360, HasToday: true, Baseline: []float64{460, 470, 455, 465, 450}},
	}
	got := Compute(in)
	banned := []string{"diagnos", "disease", "illness", "medical", "treat", "symptom", "sick", "cure"}
	low := strings.ToLower(got.Explanation)
	for _, b := range banned {
		if strings.Contains(low, b) {
			t.Errorf("explanation contains medical-claim term %q: %s", b, got.Explanation)
		}
	}
	if got.Explanation == "" {
		t.Error("explanation should be non-empty")
	}
}

func TestComputeIsDeterministic(t *testing.T) {
	in := Inputs{
		HRV:   Metric{Today: 52, HasToday: true, Baseline: []float64{60, 62, 58, 61, 59}},
		RHR:   Metric{Today: 58, HasToday: true, Baseline: []float64{54, 55, 56, 55, 54}},
		Sleep: Metric{Today: 430, HasToday: true, Baseline: []float64{460, 470, 455, 465, 450}},
	}
	a, b := Compute(in), Compute(in)
	if a.Value != b.Value || a.Explanation != b.Explanation || len(a.Drivers) != len(b.Drivers) {
		t.Fatalf("non-deterministic: %+v vs %+v", a, b)
	}
}
