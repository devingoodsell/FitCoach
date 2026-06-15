package diet

import "pro.d11l.fitcoach/backend/internal/onboarding"

// proteinSources maps a dietary pattern to suggested protein sources. Plant-only
// patterns never include animal sources (E11-S2: never violate a hard constraint).
var proteinSources = map[onboarding.DietPattern][]string{
	onboarding.DietOmnivore:    {"chicken", "eggs", "fish", "Greek yogurt", "lean beef"},
	onboarding.DietVegetarian:  {"eggs", "Greek yogurt", "cottage cheese", "lentils", "tofu"},
	onboarding.DietVegan:       {"tofu", "tempeh", "lentils", "chickpeas", "seitan", "pea protein"},
	onboarding.DietPescatarian: {"fish", "shrimp", "eggs", "Greek yogurt", "lentils"},
	onboarding.DietKosher:      {"chicken", "fish", "eggs", "lentils", "Greek yogurt"},
	onboarding.DietHalal:       {"chicken", "fish", "eggs", "lentils", "Greek yogurt"},
}

// ProteinSources returns suggested sources for the pattern, defaulting to
// omnivore when the pattern is unknown/blank.
func ProteinSources(pattern onboarding.DietPattern) []string {
	if sources, ok := proteinSources[pattern]; ok {
		return sources
	}
	return proteinSources[onboarding.DietOmnivore]
}

// Guidance returns short, plain-language, preference-aware suggestions.
func Guidance(pattern onboarding.DietPattern) []string {
	sources := ProteinSources(pattern)
	return []string{
		"Spread protein across meals; good sources for you: " + join(sources) + ".",
		"Build meals around vegetables and whole-food carbohydrates to support training.",
		"Hydrate through the day, especially around your sessions.",
	}
}

// PostWorkoutNote returns a brief, non-prescriptive note for the rest of the day
// given today's load, respecting the dietary pattern (E11-S3).
func PostWorkoutNote(heavy bool, pattern onboarding.DietPattern) string {
	sources := ProteinSources(pattern)
	if heavy {
		return "You trained hard today — prioritize protein and carbohydrates to refuel. " +
			"Easy options: " + join(sources) + "."
	}
	return "Lighter session today — keep protein steady and eat to appetite. " +
		"Protein options: " + join(sources) + "."
}

func join(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	out := items[0]
	for i := 1; i < len(items)-1; i++ {
		out += ", " + items[i]
	}
	return out + ", or " + items[len(items)-1]
}
