// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// TestFilletEmitsBlendAndRadius: the fillet helper adds a blend arc between two edges and
// dimensions its radius from the expression, returning the arc id (#69). The host consumes the
// corner and pins tangency, so this radius dimension is all the generator adds per fillet.
func TestFilletEmitsBlendAndRadius(t *testing.T) {
	h := &fakeHost{}
	b := newBuilder(h, catalog.UnitsMillimetre)
	sk, err := b.Sketch("XY")
	if err != nil {
		t.Fatalf("Sketch: %v", err)
	}
	arcID, err := sk.Fillet(70, 71, "flange_thickness")
	if err != nil {
		t.Fatalf("Fillet: %v", err)
	}
	if arcID != 85 {
		t.Errorf("fillet arc id = %d, want 85 (the host fillet reply)", arcID)
	}
	if len(h.dimensions) != 1 || h.dimensions[0].Expression != "flange_thickness" {
		t.Errorf("dimensions = %+v, want one radius dim driven by flange_thickness", h.dimensions)
	}
}
