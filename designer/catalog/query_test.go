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

	if got := c.Standards(); !reflect.DeepEqual(got, []string{"DIN", "ISO"}) {
		t.Errorf("Standards() = %v, want [DIN ISO]", got)
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
	if len(all) != 3 {
		t.Errorf("Fasteners subtree = %v, want the three fastener families", ids(all))
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
	if !containsID(structural, "iso1035-round-bar") {
		t.Errorf("Structural subtree = %v, want the round bar", ids(structural))
	}
	// An empty prefix matches everything.
	if got := c.ByCategory(nil); len(got) != c.Len() {
		t.Errorf("empty prefix matched %d, want all %d", len(got), c.Len())
	}
}
