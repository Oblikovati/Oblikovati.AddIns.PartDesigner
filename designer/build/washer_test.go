// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"strings"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// washerMember builds a synthetic resolved member with the washer columns the generator drives
// geometry against, for the given variant ("plain"/"spring"). A spring washer additionally needs
// the free_height column for its coil.
func washerMember(variant string, d1, d2, s, freeHeight float64) ResolvedMember {
	cols := []catalog.Column{
		{Name: "size", Param: "designation", Type: catalog.ColumnText},
		{Name: "d1", Param: "inner_dia", Type: catalog.ColumnLength},
		{Name: "d2", Param: "outer_dia", Type: catalog.ColumnLength},
		{Name: "s", Param: "thickness", Type: catalog.ColumnLength},
	}
	values := map[string]float64{"d1": d1, "d2": d2, "s": s}
	if variant == "spring" {
		cols = append(cols, catalog.Column{Name: "H", Param: "free_height", Type: catalog.ColumnLength})
		values["H"] = freeHeight
	}
	fam := &catalog.Family{
		ID: "t-washer", Generator: "washer", Variant: variant, Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"size"}, Columns: cols,
		Members: []catalog.Member{{Key: "size=M8", Values: values, Labels: map[string]string{"size": "M8"}}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestWasherBuildsPlainAnnulus is the B4 acceptance check for the plain washer: an M8 washer is
// realized as a DOF-0 outer disk with a concentric bore cut, every dimension bound to a published
// parameter (never a literal), and NO coil.
func TestWasherBuildsPlainAnnulus(t *testing.T) {
	h := &fakeHost{dof: 0}
	rm := washerMember("plain", 8.4, 16, 1.6, 0)
	if err := (Washer{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	// The text "size" column is a label, not a published parameter; only the lengths are.
	assertParam(t, h.added, "inner_dia", "8.4 mm")
	assertParam(t, h.added, "outer_dia", "16 mm")
	assertParam(t, h.added, "thickness", "1.6 mm")
	for _, p := range h.added {
		if p.Name == "designation" {
			t.Errorf("text column published as parameter %+v; text is a label", p)
		}
	}

	// Two circles, each centre-fixed and diameter-dimensioned to a parameter (outer then inner).
	if len(h.dimensions) != 2 ||
		h.dimensions[0].Kind != "diameter" || h.dimensions[0].Expression != "outer_dia" ||
		h.dimensions[1].Kind != "diameter" || h.dimensions[1].Expression != "inner_dia" {
		t.Errorf("dimensions = %+v, want outer then inner diameter, parameter-driven", h.dimensions)
	}

	// Outer disk (new) then bore (cut), both by thickness; no coil on a plain washer.
	if len(h.extrudes) != 2 {
		t.Fatalf("extrudes = %d, want 2 (disk + bore) for a plain washer", len(h.extrudes))
	}
	if h.extrudes[0].Distance != "thickness" || h.extrudes[0].Operation != "new" {
		t.Errorf("disk extrude = %+v, want thickness/new", h.extrudes[0])
	}
	if h.extrudes[1].Distance != "thickness" || h.extrudes[1].Operation != "cut" {
		t.Errorf("bore extrude = %+v, want thickness/cut", h.extrudes[1])
	}
	if len(h.coils) != 0 {
		t.Errorf("coils = %d, want 0 (a plain washer is a flat ring)", len(h.coils))
	}
}

// TestWasherSpringBuildsCoil covers the DIN 127 spring-lock variant: a helical coil of the ring's
// cross-section, not a flat ring. The cross-section is a DOF-0 rectangle positioned at the mean
// radius, and the coil sweeps it just under a full turn with its height driven by free_height.
func TestWasherSpringBuildsCoil(t *testing.T) {
	h := &fakeHost{dof: 0}
	rm := washerMember("spring", 12.2, 21.1, 2.5, 5.45)
	if err := (Washer{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	// A spring washer is one coil, no disk/bore extrudes.
	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 (a spring washer is a coil, not extruded)", len(h.extrudes))
	}
	if len(h.coils) != 1 {
		t.Fatalf("coils = %d, want 1", len(h.coils))
	}
	c := h.coils[0]
	if c.AxisRef != "origin/axis/z" || c.Height != "free_height - thickness" || c.Revolutions != coilRevolutions {
		t.Errorf("coil = %+v, want axis z, height free_height-thickness, revolutions %s", c, coilRevolutions)
	}

	// The cross-section is a DOF-0 rectangle: a grounded origin, axis-aligned edges (fix via the
	// grounded point + 5 horizontal/vertical), and three parameter-driven placement/size distances.
	wantCon := "ground,horizontal,horizontal,vertical,vertical,horizontal"
	if strings.Join(h.constraints, ",") != wantCon {
		t.Errorf("section constraints = %v, want %s", h.constraints, wantCon)
	}
	if len(h.dimensions) != 3 ||
		h.dimensions[0].Expression != "(inner_dia) / 2" ||
		h.dimensions[1].Expression != "((outer_dia) - (inner_dia)) / 2" ||
		h.dimensions[2].Expression != "thickness" {
		t.Errorf("section dimensions = %+v, want inner-radius, radial-width, thickness", h.dimensions)
	}
}

// TestWasherRejectsUnknownVariant ensures a mis-typed variant fails loudly rather than silently
// building a plain ring.
func TestWasherRejectsUnknownVariant(t *testing.T) {
	h := &fakeHost{dof: 0}
	err := (Washer{}).Build(newBuilder(h, catalog.UnitsMillimetre), washerMember("wavy", 8.4, 16, 1.6, 0))
	if err == nil || !strings.Contains(err.Error(), "variant") {
		t.Fatalf("Build error = %v, want it to reject the unknown variant", err)
	}
	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 when the variant is rejected before building", len(h.extrudes))
	}
}

// TestWasherUnderConstrainedFails ensures a non-zero DOF aborts the build before any solid is
// created, so a floppy profile is caught.
func TestWasherUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 2}
	err := (Washer{}).Build(newBuilder(h, catalog.UnitsMillimetre), washerMember("plain", 8.4, 16, 1.6, 0))
	if err == nil {
		t.Fatal("Build accepted an under-constrained profile; want an error")
	}
}

// TestWasherSpringUnderConstrainedFails ensures the coil's cross-section is gated on DOF-0 too.
func TestWasherSpringUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 3}
	err := (Washer{}).Build(newBuilder(h, catalog.UnitsMillimetre), washerMember("spring", 12.2, 21.1, 2.5, 5.45))
	if err == nil {
		t.Fatal("Build accepted an under-constrained coil section; want an error")
	}
	if len(h.coils) != 0 {
		t.Errorf("coils = %d, want 0 when the section is under-constrained", len(h.coils))
	}
}

// TestWasherBuildErrorsPropagate injects a host failure at each wire method both build paths use
// and asserts the error surfaces rather than a half-built part — covering the failure branches of
// the ring (disk/bore) and the coil (section + sweep).
func TestWasherBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodSketchCreate, wire.MethodSketchAddEntity,
		wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	cases := []struct {
		variant               string
		d1, d2, s, freeHeight float64
	}{
		{"plain", 8.4, 16, 1.6, 0},
		{"spring", 12.2, 21.1, 2.5, 5.45},
	}
	for _, tc := range cases {
		for _, m := range methods {
			h := &fakeHost{dof: 0, failMethod: m}
			rm := washerMember(tc.variant, tc.d1, tc.d2, tc.s, tc.freeHeight)
			if err := (Washer{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err == nil {
				t.Errorf("variant %s, failMethod %q: Build succeeded, want an error", tc.variant, m)
			}
		}
	}
}

// TestGroundedRadialSectionRejectsShortReply covers the guard that a section needs its origin
// point and four rectangle corners; a short reply is caught rather than indexing out of range.
func TestGroundedRadialSectionRejectsShortReply(t *testing.T) {
	h := &fakeHost{dof: 0, noPoints: true} // circles/points come back with no points
	b := newBuilder(h, catalog.UnitsMillimetre)
	sk, err := b.Sketch("XZ")
	if err != nil {
		t.Fatalf("Sketch error = %v", err)
	}
	if err := sk.GroundedRadialSection("inner_dia", "outer_dia", "thickness"); err == nil {
		t.Fatal("GroundedRadialSection accepted a section with no origin point; want an error")
	}
}

// TestDefaultRegistryHasWasher checks the generator is wired into the built-in set so a washer
// family resolves at placement.
func TestDefaultRegistryHasWasher(t *testing.T) {
	g, ok := DefaultRegistry().Get("washer")
	if !ok || g.Kind() != "washer" {
		t.Fatalf("DefaultRegistry washer = (%v,%v), want the Washer generator", g, ok)
	}
}
