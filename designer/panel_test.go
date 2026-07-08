// SPDX-License-Identifier: GPL-2.0-only

package designer

import (
	"strings"
	"testing"

	"oblikovati.org/api/wire"
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
	// Default selection is the first family (id-sorted: din934-hex-nut) and its first size.
	if fam.Value != "DIN 934 Hex" {
		t.Errorf("default family = %q, want DIN 934 Hex", fam.Value)
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

// TestSelectFamilyAndCategory covers picking a family by its label and filtering by category.
func TestSelectFamilyAndCategory(t *testing.T) {
	e := NewEngine(newFakeHost())

	// Filtering by the top-level category keeps both seed families (both under Fasteners).
	e.applySelection(categoryControlID, "Fasteners")
	if e.sel.category != "Fasteners" || len(e.familyOptions(e.sel)) != 2 {
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
	e.applySelection(categoryControlID, "Bearings")
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
// than failing silently (din934-hex-nut's generator is not registered here).
func TestPlaceSelectionReportsGeneratorGap(t *testing.T) {
	host := newFakeHost()
	e := NewEngine(host) // DefaultRegistry has only round_bar, not hex_nut
	// Default selection is din934-hex-nut.
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
