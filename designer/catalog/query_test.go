// SPDX-License-Identifier: GPL-2.0-only

package catalog

import (
	"reflect"
	"testing"
)

func TestStandardsAndFilters(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := c.Standards(); !reflect.DeepEqual(got, []string{"DIN", "ISO"}) {
		t.Errorf("Standards() = %v, want [DIN ISO]", got)
	}

	iso := c.ByStandardBody("iso") // case-insensitive
	if len(iso) != 1 || iso[0].ID != "iso4017-hex-bolt" {
		t.Errorf("ByStandardBody(iso) = %v, want [iso4017-hex-bolt]", ids(iso))
	}
	din := c.ByStandardBody("DIN")
	if len(din) != 1 || din[0].ID != "din934-hex-nut" {
		t.Errorf("ByStandardBody(DIN) = %v, want [din934-hex-nut]", ids(din))
	}
}

func TestByCategoryPrefix(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	all := c.ByCategory(CategoryPath{"Fasteners"})
	if len(all) != 2 {
		t.Errorf("Fasteners subtree = %v, want both seed families", ids(all))
	}
	bolts := c.ByCategory(CategoryPath{"Fasteners", "Bolts"})
	if len(bolts) != 1 || bolts[0].ID != "iso4017-hex-bolt" {
		t.Errorf("Fasteners/Bolts = %v, want [iso4017-hex-bolt]", ids(bolts))
	}
	none := c.ByCategory(CategoryPath{"Bearings"})
	if len(none) != 0 {
		t.Errorf("Bearings subtree = %v, want empty", ids(none))
	}
	// An empty prefix matches everything.
	if got := c.ByCategory(nil); len(got) != c.Len() {
		t.Errorf("empty prefix matched %d, want all %d", len(got), c.Len())
	}
}
