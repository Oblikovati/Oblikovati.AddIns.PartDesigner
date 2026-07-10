// SPDX-License-Identifier: GPL-2.0-only

package designer

import "testing"

// A tree-node click (catalog control) selects that family by ID; a table-row click (members
// control) selects that member by Key. This replaces the old label-based Part/Size dropdowns.
func TestApplySelectionTreeAndTable(t *testing.T) {
	e, _ := engineWith(t, newFakeHost())
	fams := e.catalog.Families()
	fam := fams[0]

	e.applySelection(catalogControlID, fam.ID)
	if e.sel.familyID != fam.ID {
		t.Fatalf("after tree click, familyID = %q, want %q", e.sel.familyID, fam.ID)
	}
	key := fam.Members[0].Key
	e.applySelection(membersControlID, key)
	if e.sel.memberKey != key {
		t.Fatalf("after table click, memberKey = %q, want %q", e.sel.memberKey, key)
	}
}
