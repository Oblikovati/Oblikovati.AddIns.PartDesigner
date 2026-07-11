// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// controlByID finds a panel control by id.
func controlByID(controls []wire.PanelControlSpec, id string) (wire.PanelControlSpec, bool) {
	for _, c := range controls {
		if c.ID == id {
			return c, true
		}
	}
	return wire.PanelControlSpec{}, false
}

// TestPanelShowsBrowseSurface checks the panel renders the Category/Standard/Search filters
// (fed by the catalogue) plus the catalog tree + members table, and a Place button wired to the
// Place command. Supersedes the old dropdown-based TestPanelShowsCascadingBrowser (issue #48):
// the Part/Size dropdowns are gone, replaced by a PanelTree over a PanelTable.
func TestPanelShowsBrowseSurface(t *testing.T) {
	host := newFakeHost()
	e := NewEngine(host)
	if _, err := e.ShowPanel(); err != nil {
		t.Fatalf("ShowPanel error = %v", err)
	}
	controls := host.windows[0].Controls

	std, ok := controlByID(controls, standardControlID)
	if !ok || !contains(std.Options, "ISO") || !contains(std.Options, "DIN") {
		t.Errorf("standard options = %v, want All + ISO + DIN", std.Options)
	}
	tree, ok := controlByID(controls, catalogControlID)
	if !ok || len(tree.Nodes) == 0 {
		t.Fatal("catalog tree control missing or empty")
	}
	// A default family + first size are always selected (the first family id-sorted, whichever
	// it is) so the panel opens ready to place.
	if tree.Value == "" {
		t.Errorf("default tree selection = %q, want a family ID", tree.Value)
	}
	table, ok := controlByID(controls, membersControlID)
	if !ok || len(table.TableRows) == 0 || table.Value == "" {
		t.Errorf("members table empty or unselected: rows=%d value=%q", len(table.TableRows), table.Value)
	}
	place, ok := controlByID(controls, "place")
	if !ok || place.CommandID != PlaceCommandID {
		t.Errorf("place button command = %q, want %q", place.CommandID, PlaceCommandID)
	}
}

// TestWasherTextSizeColumn checks a family keyed on a text "size" column renders its sizes as
// nominal designations (M6..M12) in the members table rather than numeric labels — selecting it
// explicitly (via the catalog tree control) so the check does not depend on which family sorts
// first as the default.
func TestWasherTextSizeColumn(t *testing.T) {
	e := NewEngine(newFakeHost())
	e.applySelection(catalogControlID, "din125-washer")
	if e.sel.familyID != "din125-washer" {
		t.Fatalf("selected family = %q, want din125-washer", e.sel.familyID)
	}
	fam := mustFamily(t, e)
	rows := tableRows(fam)
	// Sizes render as nominal "M…" designations, not numeric labels. The table spans the full
	// preferred series, so assert the designation format plus that both M6 and M12 rows exist
	// (rather than pinning the first row, which is now the smallest size M1.6).
	if len(rows) == 0 || !strings.HasPrefix(rows[0].Cells[0], "M") {
		t.Errorf("first washer row = %+v, want an \"M…\" size designation", rows)
	}
	if !anyRowCellEquals(rows, "M6") || !anyRowCellEquals(rows, "M12") {
		t.Errorf("washer table rows = %+v, want both M6 and M12 rows", rows)
	}
}

// anyRowCellEquals reports whether any row has a cell equal to want.
func anyRowCellEquals(rows []wire.TableRow, want string) bool {
	for _, r := range rows {
		if contains(r.Cells, want) {
			return true
		}
	}
	return false
}

// familyIDs extracts family ids for readable assertions.
func familyIDs(fams []*catalog.Family) []string {
	out := make([]string, len(fams))
	for i, f := range fams {
		out[i] = f.ID
	}
	return out
}

// TestSelectionCascade checks that a standard filter narrows the filtered family set, choosing a
// family (via the catalog tree control) switches the members table, and choosing a member (via
// the members table control) updates the selected member.
func TestSelectionCascade(t *testing.T) {
	e := NewEngine(newFakeHost())

	e.applySelection(standardControlID, "ISO") // narrows the browse tree to ISO families
	if e.sel.standard != "ISO" {
		t.Fatalf("standard = %q, want ISO", e.sel.standard)
	}
	if !contains(familyIDs(e.filteredFamilies(e.sel)), "iso4017-hex-bolt") {
		t.Errorf("ISO filtered families = %v, want to include the hex bolt", familyIDs(e.filteredFamilies(e.sel)))
	}

	e.applySelection(catalogControlID, "iso4017-hex-bolt")
	e.applySelection(membersControlID, "d=12,l=60") // ISO 4017 M12x60
	if e.sel.familyID != "iso4017-hex-bolt" || e.sel.memberKey != "d=12,l=60" {
		t.Errorf("selection = %q/%q, want iso4017-hex-bolt / d=12,l=60", e.sel.familyID, e.sel.memberKey)
	}

	// Clearing the standard filter restores every family.
	e.applySelection(standardControlID, allOption)
	if got := len(e.filteredFamilies(e.sel)); got != e.catalog.Len() {
		t.Errorf("filtered families after clearing filter = %d, want all %d", got, e.catalog.Len())
	}
}

// TestSearchNarrowsAndSelects is the F1 acceptance check: typing a search query narrows the
// filtered family set to matching families and re-picks a valid family, and clearing the search
// restores all.
func TestSearchNarrowsAndSelects(t *testing.T) {
	e := NewEngine(newFakeHost())

	// Searching a bearing designation narrows the filtered families to that family and selects it.
	e.applySelection(searchControlID, "6205")
	fams := e.filteredFamilies(e.sel)
	if len(fams) != 1 || fams[0].ID != "iso15-deep-groove-ball-bearing" {
		t.Fatalf("search 6205 filtered families = %v, want just the ISO 15 ball bearing", familyIDs(fams))
	}
	if e.sel.familyID != "iso15-deep-groove-ball-bearing" {
		t.Errorf("family after search = %q, want the ball bearing", e.sel.familyID)
	}

	// The searched size is offered in the now-selected family's members table.
	if !anyRowCellEquals(tableRows(mustFamily(t, e)), "6205") {
		t.Errorf("table rows after 6205 search = %+v, want a 6205 row", tableRows(mustFamily(t, e)))
	}

	// Clearing the search restores every family.
	e.applySelection(searchControlID, "")
	if got := len(e.filteredFamilies(e.sel)); got != e.catalog.Len() {
		t.Errorf("filtered families after clearing search = %d, want all %d", got, e.catalog.Len())
	}
}

// mustFamily returns the engine's currently-selected family or fails.
func mustFamily(t *testing.T, e *Engine) *catalog.Family {
	t.Helper()
	fam, ok := e.family(e.sel.familyID)
	if !ok {
		t.Fatalf("no family selected (%q)", e.sel.familyID)
	}
	return fam
}

// TestSelectFamilyAndCategory covers picking a family by ID (via the catalog tree control) and
// filtering by category.
func TestSelectFamilyAndCategory(t *testing.T) {
	e := NewEngine(newFakeHost())

	// Filtering by the top-level category keeps the fastener families: 13 metric (2 hex bolts,
	// 3 socket screws, 3 hex nuts, 3 washers, 2 studs) plus 4 ANSI inch (hex bolt, hex nut,
	// washer, socket-head cap screw).
	e.applySelection(categoryControlID, "Fasteners")
	if e.sel.category != "Fasteners" || len(e.filteredFamilies(e.sel)) != 17 {
		t.Fatalf("category=Fasteners: cat=%q families=%v", e.sel.category, familyIDs(e.filteredFamilies(e.sel)))
	}

	// Choosing a family by ID (a tree-node click) switches to it and re-picks its first size.
	e.applySelection(catalogControlID, "iso4017-hex-bolt")
	if e.sel.familyID != "iso4017-hex-bolt" {
		t.Errorf("family after tree select = %q, want iso4017-hex-bolt", e.sel.familyID)
	}
	// Re-picks the family's first (smallest) size, now M1.6 at the head of the preferred series.
	if e.sel.memberKey != "d=1.6,l=8" {
		t.Errorf("member after family switch = %q, want the first size d=1.6,l=8", e.sel.memberKey)
	}

	// An unknown category clears the family list.
	e.applySelection(categoryControlID, "Nonexistent")
	if e.sel.familyID != "" {
		t.Errorf("family under empty category = %q, want none", e.sel.familyID)
	}
}

// TestPlaceSelection places the current selection through the same path the Place button and
// headless command use.
func TestPlaceSelection(t *testing.T) {
	host := newFakeHost()
	e, _ := engineWith(t, host, "hex_bolt")
	e.applySelection(catalogControlID, "iso4017-hex-bolt") // pick the hex bolt + its first size

	e.placeSelection()

	if len(host.docs) != 1 || host.docs[0].Type != "part" {
		t.Fatalf("docs = %+v, want one placed part", host.docs)
	}
	if got := host.attrs[attrKey(host.docs[0].ID, attrSet, familyAttr)]; got != "iso4017-hex-bolt" {
		t.Errorf("stamped family = %q, want iso4017-hex-bolt", got)
	}
	if !strings.HasPrefix(host.status, "Part Designer: placed") {
		t.Errorf("status = %q, want a placed-confirmation", host.status)
	}
}

// TestPlaceSelectionReportsGeneratorGap surfaces a missing generator on the status bar rather
// than failing silently. The engine here registers only hex_bolt, so selecting the hex-nut
// family (generator "hex_nut", absent from this registry) and placing must report the gap.
func TestPlaceSelectionReportsGeneratorGap(t *testing.T) {
	host := newFakeHost()
	e, _ := engineWith(t, host, "hex_bolt") // hex_nut deliberately NOT registered
	e.applySelection(catalogControlID, "din934-hex-nut")
	e.placeSelection()
	if !strings.Contains(host.status, "not registered") {
		t.Errorf("status = %q, want a generator-not-registered message", host.status)
	}
}

// TestCatalogErrorControls renders a visible error when the catalogue is unavailable.
func TestCatalogErrorControls(t *testing.T) {
	controls := catalogErrorControls(nil)
	if _, ok := controlByID(controls, "err"); !ok {
		t.Error("catalogErrorControls has no error label")
	}
}

// contains reports whether xs includes s.
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// browserControls must expose a PanelTree and a PanelTable (the browse surface) plus the Place
// button, and keep the Category/Standard/Search filters. The old Part/Size dropdowns are gone.
func TestBrowserControlsHasTreeAndTable(t *testing.T) {
	e, _ := engineWith(t, newFakeHost())
	controls := e.browserControls(e.defaultSelection())
	kinds := map[types.PanelControlKind]int{}
	for _, c := range controls {
		kinds[c.Kind]++
	}
	if kinds[types.PanelTree] != 1 || kinds[types.PanelTable] != 1 {
		t.Fatalf("want exactly one tree and one table, got tree=%d table=%d", kinds[types.PanelTree], kinds[types.PanelTable])
	}
	if kinds[types.PanelDropdown] < 2 { // Category + Standard filters remain
		t.Fatalf("want the Category/Standard filter dropdowns retained, got %d dropdowns", kinds[types.PanelDropdown])
	}
}
