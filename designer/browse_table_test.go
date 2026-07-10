// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"testing"
)

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
