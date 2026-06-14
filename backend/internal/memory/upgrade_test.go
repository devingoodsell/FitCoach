package memory

import (
	"encoding/json"
	"testing"
)

func TestUpgradePassThroughWhenCurrent(t *testing.T) {
	u := NewUpgrader()
	in := json.RawMessage(`{"name":"Dev"}`)
	out, version, err := u.Upgrade(SectionProfile, 1, in)
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if version != 1 || string(out) != string(in) {
		t.Fatalf("got version %d data %s, want 1 unchanged", version, out)
	}
}

func TestUpgradeChainsStepsWithoutDataLoss(t *testing.T) {
	// White-box: target profile at v3 with two registered steps.
	u := &Upgrader{
		upgrades: map[Section]map[int]UpgradeFunc{},
		target:   map[Section]int{SectionProfile: 3},
	}
	// v1 -> v2: add unit, preserving existing fields.
	u.Register(SectionProfile, 1, func(data json.RawMessage) (json.RawMessage, error) {
		m := map[string]any{}
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		m["unit"] = "metric"
		return json.Marshal(m)
	})
	// v2 -> v3: add schema marker.
	u.Register(SectionProfile, 2, func(data json.RawMessage) (json.RawMessage, error) {
		m := map[string]any{}
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		m["migrated"] = true
		return json.Marshal(m)
	})

	out, version, err := u.Upgrade(SectionProfile, 1, json.RawMessage(`{"age":40}`))
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if version != 3 {
		t.Fatalf("version = %d, want 3", version)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Original data preserved across both steps.
	if got["age"] != float64(40) {
		t.Errorf("lost original field: %+v", got)
	}
	if got["unit"] != "metric" || got["migrated"] != true {
		t.Errorf("upgrade steps not applied: %+v", got)
	}
}

func TestUpgradeMissingStepIsError(t *testing.T) {
	u := &Upgrader{
		upgrades: map[Section]map[int]UpgradeFunc{},
		target:   map[Section]int{SectionGoals: 2}, // no step registered
	}
	if _, _, err := u.Upgrade(SectionGoals, 1, json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected error for missing upgrade step")
	}
}

func TestUpgradeRejectsNewerThanSupported(t *testing.T) {
	u := NewUpgrader() // profile target = 1
	if _, _, err := u.Upgrade(SectionProfile, 2, json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected error for stored version newer than target")
	}
}
