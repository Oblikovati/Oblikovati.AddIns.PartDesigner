// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// columnHeader turns the descriptive Param into a full-word header (the cryptic standards symbol
// Name is not shown), expands known abbreviations, and falls back to Name only when Param is empty.
func TestColumnHeader(t *testing.T) {
	cases := []struct {
		name, param, want string
	}{
		{"s", "across_flats", "Across Flats"},
		{"k", "head_height", "Head Height"},
		{"D", "outer_dia", "Outer Diameter"}, // abbreviation expanded
		{"e", "hole_end_dist", "Hole End Distance"},
		{"l", "length", "Length"}, // single word
		{"a", "leg_a", "Leg A"},   // trailing single letter title-cased
		{"Size", "", "Size"},      // no param -> fall back to the symbol Name
	}
	for _, c := range cases {
		got := columnHeader(catalog.Column{Name: c.name, Param: c.param})
		if got != c.want {
			t.Errorf("columnHeader(name=%q,param=%q) = %q, want %q", c.name, c.param, got, c.want)
		}
	}
}

// tableRows must produce one row per member, keyed by the member Key, with a cell per declared
// column in column order — the full family-table view.
func TestTableModelForFamily(t *testing.T) {
	e, _ := engineWith(t, newFakeHost()) // real helper: placement_test.go:31
	e.sel = e.defaultSelection()         // engineWith (unlike NewEngine) does not pre-select a family
	fam := mustFamily(t, e)              // real helper: panel_test.go:126
	cols := tableColumns(fam)
	rows := tableRows(fam)
	if len(cols) != len(fam.Columns) {
		t.Fatalf("cols = %d, want %d", len(cols), len(fam.Columns))
	}
	if len(rows) != len(fam.Members) {
		t.Fatalf("rows = %d, want %d", len(rows), len(fam.Members))
	}
	if rows[0].Key != fam.Members[0].Key {
		t.Fatalf("row0 key = %q, want %q", rows[0].Key, fam.Members[0].Key)
	}
	if len(rows[0].Cells) != len(cols) {
		t.Fatalf("row0 has %d cells, want %d (one per column)", len(rows[0].Cells), len(cols))
	}
}
