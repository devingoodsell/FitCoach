package memory

import "testing"

func TestAllSectionsHaveCurrentVersions(t *testing.T) {
	if len(AllSections) != len(CurrentVersions) {
		t.Fatalf("AllSections (%d) and CurrentVersions (%d) disagree", len(AllSections), len(CurrentVersions))
	}
	for _, s := range AllSections {
		if _, ok := CurrentVersions[s]; !ok {
			t.Errorf("section %q missing from CurrentVersions", s)
		}
		if !IsValidSection(s) {
			t.Errorf("section %q not reported valid", s)
		}
	}
}

func TestIsValidSectionRejectsUnknown(t *testing.T) {
	if IsValidSection("bogus") {
		t.Error("unknown section reported valid")
	}
}
