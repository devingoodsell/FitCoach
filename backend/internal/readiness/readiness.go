// Package readiness computes FitCoach's OWN recovery readiness from raw Health
// Connect signals (E4-S2) — never a vendor score. It compares today's overnight
// HRV, resting HR, and sleep against the user's rolling baseline, handles
// missing/partial data by flagging low confidence, and produces a plain-language
// explanation with no medical claims (E4-S3).
package readiness

import "math"

// Signal kinds stored in health_signals and read for computation.
const (
	KindHRV   = "hrv_ms"
	KindRHR   = "rhr_bpm"
	KindSleep = "sleep_minutes"
)

// minBaseline is the number of prior samples required before a metric counts.
const minBaseline = 3

// Confidence levels.
const (
	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"
)

// Metric carries today's value (if any) and the baseline history for one signal.
type Metric struct {
	Today    float64
	HasToday bool
	Baseline []float64
}

func (m Metric) usable() bool { return m.HasToday && len(m.Baseline) >= minBaseline }

// Inputs is the per-user data the computation needs.
type Inputs struct {
	HRV   Metric
	RHR   Metric
	Sleep Metric
}

// Score is the computed readiness with drivers and a plain-language explanation.
type Score struct {
	Value       int      `json:"value"`      // 0..100 (50 = neutral)
	Confidence  string   `json:"confidence"` // high | medium | low
	Drivers     []string `json:"drivers"`    // machine keys, e.g. "hrv_low"
	Explanation string   `json:"explanation"`
}

// metric weights when all are present (renormalized over present metrics).
var weights = map[string]float64{KindHRV: 0.4, KindRHR: 0.3, KindSleep: 0.3}

// component is one metric's contribution: a 0..100 value and its weight.
type component struct {
	kind   string
	value  float64
	weight float64
}

// Compute derives the readiness score deterministically. With no usable metric it
// returns a neutral 50 at low confidence.
func Compute(in Inputs) Score {
	var comps []component
	usable := 0

	add := func(kind string, m Metric, higherIsBetter bool) {
		if !m.usable() {
			return
		}
		usable++
		mean, sd := stats(m.Baseline)
		z := 0.0
		if sd > 0 {
			z = (m.Today - mean) / sd
		}
		if !higherIsBetter {
			z = -z
		}
		comps = append(comps, component{kind: kind, value: clamp(50+12.5*z, 0, 100), weight: weights[kind]})
	}
	add(KindHRV, in.HRV, true)     // higher HRV is better
	add(KindRHR, in.RHR, false)    // higher resting HR is worse
	add(KindSleep, in.Sleep, true) // more sleep (vs your norm) is better

	if usable == 0 {
		return Score{
			Value:       50,
			Confidence:  ConfidenceLow,
			Drivers:     nil,
			Explanation: "Not enough recent data to estimate readiness today.",
		}
	}

	var weighted, totalWeight float64
	for _, c := range comps {
		weighted += c.value * c.weight
		totalWeight += c.weight
	}
	value := int(math.Round(weighted / totalWeight))

	drivers := deriveDrivers(comps)
	return Score{
		Value:       value,
		Confidence:  confidence(usable),
		Drivers:     drivers,
		Explanation: explain(drivers, confidence(usable)),
	}
}

func confidence(usable int) string {
	switch usable {
	case 3:
		return ConfidenceHigh
	case 2:
		return ConfidenceMedium
	default:
		return ConfidenceLow
	}
}

func stats(xs []float64) (mean, sd float64) {
	if len(xs) == 0 {
		return 0, 0
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	mean = sum / float64(len(xs))
	var ss float64
	for _, x := range xs {
		d := x - mean
		ss += d * d
	}
	sd = math.Sqrt(ss / float64(len(xs)))
	return mean, sd
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
