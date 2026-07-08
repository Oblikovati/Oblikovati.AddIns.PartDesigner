// SPDX-License-Identifier: GPL-2.0-only

package catalog

import (
	"reflect"
	"testing"
)

// containsID reports whether fams includes a family with the given id.
func containsID(fams []*Family, id string) bool {
	for _, f := range fams {
		if f.ID == id {
			return true
		}
	}
	return false
}

func TestStandardsAndFilters(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := c.Standards(); !reflect.DeepEqual(got, []string{"AISC", "DIN", "EN", "ISO"}) {
		t.Errorf("Standards() = %v, want [AISC DIN EN ISO]", got)
	}
	aisc := c.ByStandardBody("aisc")
	if !containsID(aisc, "w-aisc") || !containsID(aisc, "c-aisc") {
		t.Errorf("ByStandardBody(aisc) = %v, want the AISC W and C shapes", ids(aisc))
	}

	iso := c.ByStandardBody("iso") // case-insensitive
	if !containsID(iso, "iso4017-hex-bolt") || !containsID(iso, "iso1035-round-bar") {
		t.Errorf("ByStandardBody(iso) = %v, want the ISO families", ids(iso))
	}
	din := c.ByStandardBody("DIN")
	if !containsID(din, "din934-hex-nut") || !containsID(din, "din933-hex-bolt") {
		t.Errorf("ByStandardBody(DIN) = %v, want the DIN families", ids(din))
	}
}

func TestByCategoryPrefix(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	all := c.ByCategory(CategoryPath{"Fasteners"})
	if len(all) != 13 {
		t.Errorf("Fasteners subtree = %v, want the thirteen fastener families", ids(all))
	}
	studs := c.ByCategory(CategoryPath{"Fasteners", "Studs"})
	if !containsID(studs, "din976-threaded-rod") || !containsID(studs, "din939-stud") {
		t.Errorf("Fasteners/Studs = %v, want the threaded rod and double-end stud", ids(studs))
	}
	bolts := c.ByCategory(CategoryPath{"Fasteners", "Bolts"})
	if !containsID(bolts, "iso4017-hex-bolt") || !containsID(bolts, "din933-hex-bolt") {
		t.Errorf("Fasteners/Bolts = %v, want both hex bolts", ids(bolts))
	}
	none := c.ByCategory(CategoryPath{"Bearings"})
	if len(none) != 0 {
		t.Errorf("Bearings subtree = %v, want empty", ids(none))
	}
	structural := c.ByCategory(CategoryPath{"Structural"})
	if !containsID(structural, "iso1035-round-bar") || !containsID(structural, "en10058-flat-bar") {
		t.Errorf("Structural subtree = %v, want the round bar and flat bar", ids(structural))
	}
	bars := c.ByCategory(CategoryPath{"Structural", "Bars", "Flat"})
	if len(bars) != 1 || !containsID(bars, "en10058-flat-bar") {
		t.Errorf("Structural/Bars/Flat = %v, want just the EN 10058 flat bar", ids(bars))
	}
	beams := c.ByCategory(CategoryPath{"Structural", "Beams"})
	for _, id := range []string{"ipe-en10365", "hea-en10365", "heb-en10365", "w-aisc"} {
		if !containsID(beams, id) {
			t.Errorf("Structural/Beams = %v, want the IPE/HE A/HE B and AISC W series (missing %s)", ids(beams), id)
		}
	}
	channels := c.ByCategory(CategoryPath{"Structural", "Channels"})
	if !containsID(channels, "upn-en10279") || !containsID(channels, "c-aisc") {
		t.Errorf("Structural/Channels = %v, want the UPN and AISC C channels", ids(channels))
	}
	angles := c.ByCategory(CategoryPath{"Structural", "Angles"})
	if !containsID(angles, "angle-equal-en10056") || !containsID(angles, "angle-unequal-en10056") {
		t.Errorf("Structural/Angles = %v, want the equal and unequal EN 10056 angles", ids(angles))
	}
	tees := c.ByCategory(CategoryPath{"Structural", "Tees"})
	if len(tees) != 1 || !containsID(tees, "tee-en10055") {
		t.Errorf("Structural/Tees = %v, want just the EN 10055 tee", ids(tees))
	}
	hollow := c.ByCategory(CategoryPath{"Structural", "Hollow Sections"})
	for _, id := range []string{"shs-en10219", "rhs-en10219", "chs-en10219"} {
		if !containsID(hollow, id) {
			t.Errorf("Structural/Hollow Sections = %v, want SHS/RHS/CHS (missing %s)", ids(hollow), id)
		}
	}
	// An empty prefix matches everything.
	if got := c.ByCategory(nil); len(got) != c.Len() {
		t.Errorf("empty prefix matched %d, want all %d", len(got), c.Len())
	}
}
