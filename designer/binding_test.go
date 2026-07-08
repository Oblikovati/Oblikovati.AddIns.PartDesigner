// SPDX-License-Identifier: GPL-2.0-only

package designer

import "testing"

// TestBindActiveStampedPart is the F2 acceptance check: activating a placed part binds the panel
// to it — the selection switches to the part's stamped family + size and the panel enters
// Change-Size mode, so the panel shows the active part's current size.
func TestBindActiveStampedPart(t *testing.T) {
	h := newFakeHost()
	e, _ := engineWith(t, h, "hex_bolt")
	if _, err := e.Place("iso4017-hex-bolt", "d=12,l=60"); err != nil {
		t.Fatalf("Place error = %v", err)
	}
	// Move the browse selection elsewhere, then activate the placed part.
	e.applySelection(searchControlID, "circlip")

	e.bindActiveDocument()

	if !e.bound {
		t.Fatal("panel did not bind to the active stamped part")
	}
	if e.sel.familyID != "iso4017-hex-bolt" || e.sel.memberKey != "d=12,l=60" {
		t.Errorf("bound selection = %q/%q, want the placed hex bolt d=12,l=60", e.sel.familyID, e.sel.memberKey)
	}
	if sizeLabelOf(mustFamily(t, e), e.sel.memberKey) != "12x60" {
		t.Errorf("shown size = %q, want 12x60", sizeLabelOf(mustFamily(t, e), e.sel.memberKey))
	}
}

// TestPlaceBindsPanel checks that placing a part leaves the panel bound to it directly — the
// activation event fires before the stamp, so placeSelection reconciles after Place so an
// immediate Size change re-drives the new part.
func TestPlaceBindsPanel(t *testing.T) {
	h := newFakeHost()
	e, _ := engineWith(t, h, "hex_bolt")
	e.mu.Lock()
	e.sel = panelState{familyID: "iso4017-hex-bolt", memberKey: "d=8,l=40"}
	e.mu.Unlock()

	e.placeSelection()

	if !e.bound {
		t.Fatal("panel not bound after placing a part")
	}
	if e.sel.familyID != "iso4017-hex-bolt" || e.sel.memberKey != "d=8,l=40" {
		t.Errorf("bound selection = %q/%q, want the placed part", e.sel.familyID, e.sel.memberKey)
	}
}

// TestBindUnstampedPartUnbinds checks activating a part this add-in did not place leaves the panel
// in browse mode (not bound), so a Size change would place a new part rather than re-drive it.
func TestBindUnstampedPartUnbinds(t *testing.T) {
	h := newFakeHost()
	e, _ := engineWith(t, h, "hex_bolt")
	e.bound = true // pretend a previous part was bound
	h.seedActivePart("Some Other Part")

	e.bindActiveDocument()

	if e.bound {
		t.Error("panel stayed bound to an unstamped part")
	}
}

// TestBindNonPartUnbinds checks activating an assembly (not a part) unbinds the panel.
func TestBindNonPartUnbinds(t *testing.T) {
	h := newFakeHost()
	e, _ := engineWith(t, h, "hex_bolt")
	e.bound = true
	h.seedActiveAssembly("Assembly1")

	e.bindActiveDocument()

	if e.bound {
		t.Error("panel stayed bound when the active document is an assembly")
	}
}

// TestBoundSizeChangeRedrives checks that, once bound, changing the Size re-drives the same
// document in place (Change-Size) — the new member's parameters are set and the document is
// recomputed, and no second part document is created.
func TestBoundSizeChangeRedrives(t *testing.T) {
	h := newFakeHost()
	e, _ := engineWith(t, h, "hex_bolt")
	if _, err := e.Place("iso4017-hex-bolt", "d=8,l=40"); err != nil {
		t.Fatalf("Place error = %v", err)
	}
	docsBefore := len(h.docs)
	e.bindActiveDocument()

	// Emulate the bound Size-dropdown edit: apply the selection, then re-drive because bound.
	e.applySelection(sizeControlID, "12x60")
	if err := e.ChangeSize(e.sel.memberKey); err != nil {
		t.Fatalf("ChangeSize error = %v", err)
	}

	if len(h.docs) != docsBefore {
		t.Errorf("documents = %d, want %d (Change-Size must not create a new document)", len(h.docs), docsBefore)
	}
	if !hasParam(h.set, "length", "60 mm") {
		t.Errorf("re-driven params = %+v, want length=60 mm", h.set)
	}
	if h.updates == 0 {
		t.Error("document was not recomputed after the bound size change")
	}
}
