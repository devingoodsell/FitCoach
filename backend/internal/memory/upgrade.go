package memory

import (
	"encoding/json"
	"fmt"
)

// UpgradeFunc migrates a section document from version v to v+1. It must preserve
// existing data, only adding/transforming fields.
type UpgradeFunc func(data json.RawMessage) (json.RawMessage, error)

// Upgrader migrates stored section documents forward to the current schema
// version on read (E3-S1), so evolving a section never strands old records. With
// no registered upgrades and matching versions it is a pass-through.
type Upgrader struct {
	upgrades map[Section]map[int]UpgradeFunc
	target   map[Section]int
}

// NewUpgrader returns an Upgrader targeting CurrentVersions with no upgrades
// registered (the MVP state, where every section is at version 1).
func NewUpgrader() *Upgrader {
	return &Upgrader{
		upgrades: map[Section]map[int]UpgradeFunc{},
		target:   CurrentVersions,
	}
}

// Register adds the upgrade that takes a section document from `from` to from+1.
// Call once per version step when a section's shape changes.
func (u *Upgrader) Register(s Section, from int, fn UpgradeFunc) {
	if u.upgrades[s] == nil {
		u.upgrades[s] = map[int]UpgradeFunc{}
	}
	u.upgrades[s][from] = fn
}

// Upgrade migrates data from fromVersion up to the section's target version,
// applying each registered step in turn. It returns the upgraded document and
// the version it now conforms to. A stored version newer than the target, or a
// missing intermediate step, is an error.
func (u *Upgrader) Upgrade(s Section, fromVersion int, data json.RawMessage) (json.RawMessage, int, error) {
	target, ok := u.target[s]
	if !ok {
		return nil, 0, fmt.Errorf("unknown section %q", s)
	}
	if fromVersion > target {
		return nil, 0, fmt.Errorf("section %q stored version %d is newer than supported version %d", s, fromVersion, target)
	}
	out := data
	for cur := fromVersion; cur < target; cur++ {
		fn, ok := u.upgrades[s][cur]
		if !ok {
			return nil, 0, fmt.Errorf("no upgrade registered for section %q from version %d", s, cur)
		}
		next, err := fn(out)
		if err != nil {
			return nil, 0, fmt.Errorf("upgrade section %q v%d->v%d: %w", s, cur, cur+1, err)
		}
		out = next
	}
	return out, target, nil
}
