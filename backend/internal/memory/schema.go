// Package memory is the Coach Memory store: a durable, account-scoped, versioned
// record the model reads on every decision (E3). Singleton sections are stored
// as versioned JSON documents; workout logs and coach notes are append-many.
package memory

// Section identifies a singleton Coach Memory domain. The set is fixed; adding a
// section is a deliberate schema change.
type Section string

const (
	SectionProfile     Section = "profile"
	SectionGoals       Section = "goals"
	SectionSchedule    Section = "schedule"
	SectionPreferences Section = "preferences"
	SectionLocations   Section = "locations"
	SectionInjuries    Section = "injuries"
	SectionDiet        Section = "diet"
)

// AllSections is the canonical, ordered list of singleton sections. Order is
// stable so prompt assembly (E3-S2) is deterministic.
var AllSections = []Section{
	SectionProfile,
	SectionGoals,
	SectionSchedule,
	SectionPreferences,
	SectionLocations,
	SectionInjuries,
	SectionDiet,
}

// CurrentVersions records the current schema version of each section. A stored
// record at a lower version is migrated forward on read by the Upgrader (E3-S1).
// Bump a section's version here and register an UpgradeFunc when its shape changes.
var CurrentVersions = map[Section]int{
	SectionProfile:     1,
	SectionGoals:       1,
	SectionSchedule:    1,
	SectionPreferences: 1,
	SectionLocations:   1,
	SectionInjuries:    1,
	SectionDiet:        1,
}

// WorkoutLogVersion is the current schema version for recorded sessions.
const WorkoutLogVersion = 1

// IsValidSection reports whether s names a known singleton section.
func IsValidSection(s Section) bool {
	_, ok := CurrentVersions[s]
	return ok
}
