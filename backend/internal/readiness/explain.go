package readiness

import "strings"

// Driver keys (machine-readable) and their plain-language phrasing. Phrasing is
// centralized here and deliberately avoids medical claims (E4-S3).
const (
	driverHRVLow     = "hrv_low"
	driverHRVHigh    = "hrv_high"
	driverRHRHigh    = "rhr_high"
	driverRHRLow     = "rhr_low"
	driverSleepShort = "sleep_short"
	driverSleepLong  = "sleep_long"
)

var driverPhrasing = map[string]string{
	driverHRVLow:     "HRV is below your recent baseline",
	driverHRVHigh:    "HRV is above your recent baseline",
	driverRHRHigh:    "resting heart rate is higher than your baseline",
	driverRHRLow:     "resting heart rate is lower than your baseline",
	driverSleepShort: "you slept less than usual",
	driverSleepLong:  "you slept more than usual",
}

// component thresholds for flagging a metric as a notable driver.
const (
	lowThreshold  = 45
	highThreshold = 55
)

// deriveDrivers maps low/high components to driver keys, in a stable order.
func deriveDrivers(comps []component) []string {
	var drivers []string
	for _, c := range comps {
		switch c.kind {
		case KindHRV:
			if c.value < lowThreshold {
				drivers = append(drivers, driverHRVLow)
			} else if c.value > highThreshold {
				drivers = append(drivers, driverHRVHigh)
			}
		case KindRHR:
			// low component means readiness-reducing, i.e. RHR elevated.
			if c.value < lowThreshold {
				drivers = append(drivers, driverRHRHigh)
			} else if c.value > highThreshold {
				drivers = append(drivers, driverRHRLow)
			}
		case KindSleep:
			if c.value < lowThreshold {
				drivers = append(drivers, driverSleepShort)
			} else if c.value > highThreshold {
				drivers = append(drivers, driverSleepLong)
			}
		}
	}
	return drivers
}

// explain renders the drivers into a short sentence. With no notable drivers, the
// signals are in line with baseline.
func explain(drivers []string, conf string) string {
	var sentence string
	if len(drivers) == 0 {
		sentence = "Your recovery signals are in line with your baseline."
	} else {
		phrases := make([]string, 0, len(drivers))
		for _, d := range drivers {
			if p, ok := driverPhrasing[d]; ok {
				phrases = append(phrases, p)
			}
		}
		sentence = capitalize(joinPhrases(phrases)) + "."
	}
	if conf == ConfidenceLow {
		sentence += " Limited data today, so this is a rough estimate."
	}
	return sentence
}

func joinPhrases(p []string) string {
	switch len(p) {
	case 0:
		return ""
	case 1:
		return p[0]
	case 2:
		return p[0] + " and " + p[1]
	}
	return strings.Join(p[:len(p)-1], ", ") + ", and " + p[len(p)-1]
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
