// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"strings"
	"testing"

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

// TestPanelShowsCascadingBrowser checks the panel renders the Category/Standard/Part/Size
// dropdowns (fed by the catalogue) plus a Place button wired to the Place command.
func TestPanelShowsCascadingBrowser(t *testing.T) {
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
	fam, ok := controlByID(controls, familyControlID)
	if !ok || !contains(fam.Options, "ISO 4017 Hex Head") || !contains(fam.Options, "DIN 934 Hex") {
		t.Errorf("family options = %v, want both seed families' labels", fam.Options)
	}
	// A default family + first size are always selected (the first family id-sorted, whichever
	// it is) so the panel opens ready to place.
	if fam.Value == "" || !contains(fam.Options, fam.Value) {
		t.Errorf("default family = %q, want one of the family options %v", fam.Value, fam.Options)
	}
	size, _ := controlByID(controls, sizeControlID)
	if size.Value == "" || len(size.Options) == 0 {
		t.Errorf("size dropdown empty; value=%q options=%v", size.Value, size.Options)
	}
	place, ok := controlByID(controls, "place")
	if !ok || place.CommandID != PlaceCommandID {
		t.Errorf("place button command = %q, want %q", place.CommandID, PlaceCommandID)
	}
}

// TestWasherTextSizeColumn checks a family keyed on a text "size" column renders its sizes as
// nominal designations (M6..M12) rather than numeric labels — selecting it explicitly so the
// check does not depend on which family sorts first as the default.
func TestWasherTextSizeColumn(t *testing.T) {
	e := NewEngine(newFakeHost())
	e.applySelection(familyControlID, "DIN 125 Plain")
	if e.sel.familyID != "din125-washer" {
		t.Fatalf("selected family = %q, want din125-washer", e.sel.familyID)
	}
	fam, _ := e.family(e.sel.familyID)
	sizes := sizeOptions(fam)
	if sizeLabelOf(fam, e.sel.memberKey) != "M6" || !contains(sizes, "M12") {
		t.Errorf("washer sizes = %q %v, want nominal M6..M12 labels", sizeLabelOf(fam, e.sel.memberKey), sizes)
	}
}

// TestSelectionCascade checks that a standard filter narrows the Part list, choosing a family
// switches the sizes, and choosing a size updates the member.
func TestSelectionCascade(t *testing.T) {
	e := NewEngine(newFakeHost())

	e.applySelection(standardControlID, "ISO") // narrows the Part list to ISO families
	if e.sel.standard != "ISO" {
		t.Fatalf("standard = %q, want ISO", e.sel.standard)
	}
	if !contains(e.familyOptions(e.sel), "ISO 4017 Hex Head") {
		t.Errorf("ISO family options = %v, want to include the hex bolt", e.familyOptions(e.sel))
	}

	e.applySelection(familyControlID, "ISO 4017 Hex Head")
	e.applySelection(sizeControlID, "12x60") // ISO 4017 M12x60
	if e.sel.familyID != "iso4017-hex-bolt" || e.sel.memberKey != "d=12,l=60" {
		t.Errorf("selection = %q/%q, want iso4017-hex-bolt / d=12,l=60", e.sel.familyID, e.sel.memberKey)
	}

	// Clearing the standard filter restores every family.
	e.applySelection(standardControlID, allOption)
	if got := len(e.familyOptions(e.sel)); got != e.catalog.Len() {
		t.Errorf("family options after clearing filter = %d, want all %d", got, e.catalog.Len())
	}
}

// TestSearchNarrowsAndSelects is the F1 acceptance check: typing a search query narrows the Part
// list to matching families and re-picks a valid family, and clearing the search restores all.
func TestSearchNarrowsAndSelects(t *testing.T) {
	e := NewEngine(newFakeHost())

	// Searching a bearing designation narrows the Part list to that family and selects it.
	e.applySelection(searchControlID, "6205")
	opts := e.familyOptions(e.sel)
	if len(opts) != 1 || !contains(opts, "ISO 15 Deep Groove") {
		t.Fatalf("search 6205 family options = %v, want just the ISO 15 ball bearing", opts)
	}
	if e.sel.familyID != "iso15-deep-groove-ball-bearing" {
		t.Errorf("family after search = %q, want the ball bearing", e.sel.familyID)
	}

	// The searched size is offered in the now-selected family's Size dropdown.
	if !contains(sizeOptions(mustFamily(t, e)), "6205") {
		t.Errorf("size options after 6205 search = %v, want to include 6205", sizeOptions(mustFamily(t, e)))
	}

	// Clearing the search restores every family.
	e.applySelection(searchControlID, "")
	if got := len(e.familyOptions(e.sel)); got != e.catalog.Len() {
		t.Errorf("family options after clearing search = %d, want all %d", got, e.catalog.Len())
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

// TestSelectFamilyAndCategory covers picking a family by its label and filtering by category.
func TestSelectFamilyAndCategory(t *testing.T) {
	e := NewEngine(newFakeHost())

	// Filtering by the top-level category keeps the fastener families: 13 metric (2 hex bolts,
	// 3 socket screws, 3 hex nuts, 3 washers, 2 studs) plus 4 ANSI inch (hex bolt, hex nut,
	// washer, socket-head cap screw).
	e.applySelection(categoryControlID, "Fasteners")
	if e.sel.category != "Fasteners" || len(e.familyOptions(e.sel)) != 17 {
		t.Fatalf("category=Fasteners: cat=%q families=%v", e.sel.category, e.familyOptions(e.sel))
	}

	// Choosing a family by its label switches to it and re-picks its first size.
	e.applySelection(familyControlID, "ISO 4017 Hex Head")
	if e.sel.familyID != "iso4017-hex-bolt" {
		t.Errorf("family after label select = %q, want iso4017-hex-bolt", e.sel.familyID)
	}
	if e.sel.memberKey != "d=6,l=30" {
		t.Errorf("member after family switch = %q, want the first size d=6,l=30", e.sel.memberKey)
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
	e.applySelection(familyControlID, "ISO 4017 Hex Head") // pick the hex bolt + its first size

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
	e.applySelection(familyControlID, "DIN 934 Hex")
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
