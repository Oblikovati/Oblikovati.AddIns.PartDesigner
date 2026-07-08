// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"strings"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// hexNutMember builds a synthetic resolved member with the ISO 4032 / DIN 934 columns the HexNut
// generator drives geometry against.
func hexNutMember(d, pitch, s, m float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-hex-nut", Generator: "hex_nut", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"d"},
		Columns: []catalog.Column{
			{Name: "d", Param: "nominal_dia", Type: catalog.ColumnLength},
			{Name: "P", Param: "thread_pitch", Type: catalog.ColumnLength},
			{Name: "s", Param: "across_flats", Type: catalog.ColumnLength},
			{Name: "m", Param: "nut_height", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{Key: "d=8", Values: map[string]float64{
			"d": d, "P": pitch, "s": s, "m": m,
		}}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestHexNutBuildsParametricNut is the B3 acceptance check: an M8 hex nut is realized as a DOF-0
// hexagonal prism (extruded up) with a through-bore cut down its axis and a cosmetic internal
// thread on the bore wall, every dimension bound to a published parameter (never a literal), so
// editing the size re-drives the part.
func TestHexNutBuildsParametricNut(t *testing.T) {
	h := &fakeHost{dof: 0}
	rm := hexNutMember(8, 1.25, 13, 6.8)
	if err := (HexNut{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	assertParam(t, h.added, "across_flats", "13 mm")
	assertParam(t, h.added, "nut_height", "6.8 mm")
	assertParam(t, h.added, "nominal_dia", "8 mm")

	// Hexagon: centre fixed, +X flat vertical (rotation), across-corners dimensioned; then the
	// bore circle's centre fixed. Both dimensions are parameter expressions, not coordinates.
	wantCon := []string{"fix", "vertical", "fix"}
	if strings.Join(h.constraints, ",") != strings.Join(wantCon, ",") {
		t.Errorf("constraints = %v, want %v", h.constraints, wantCon)
	}
	if len(h.dimensions) != 2 ||
		h.dimensions[0].Kind != "distance" || h.dimensions[0].Expression != "(across_flats) / cos(30 deg)" ||
		h.dimensions[1].Kind != "diameter" || h.dimensions[1].Expression != "nominal_dia" {
		t.Errorf("dimensions = %+v, want across-corners then bore diameter, both parameter-driven", h.dimensions)
	}

	// Prism extrudes up (new); the bore cuts up through the full height (cut). Both are
	// parameter-driven by nut_height, so the six flats and the through-hole re-derive with size.
	if len(h.extrudes) != 2 {
		t.Fatalf("extrudes = %d, want 2 (prism + bore)", len(h.extrudes))
	}
	if h.extrudes[0].Distance != "nut_height" || h.extrudes[0].Operation != "new" || h.extrudes[0].Direction != "" {
		t.Errorf("prism extrude = %+v, want nut_height/new/positive", h.extrudes[0])
	}
	if h.extrudes[1].Distance != "nut_height" || h.extrudes[1].Operation != "cut" || h.extrudes[1].Direction != "" {
		t.Errorf("bore extrude = %+v, want nut_height/cut/positive", h.extrudes[1])
	}

	// Cosmetic internal thread on the bore's cylindrical face, sized from d and P, added LAST so
	// no boolean follows it (a cut after a cosmetic thread silently no-ops).
	if len(h.threads) != 1 || h.threads[0].Designation != "M8x1.25" || h.threads[0].Cut {
		t.Errorf("threads = %+v, want one cosmetic M8x1.25 on the bore", h.threads)
	}
}

// TestHexNutThreadNeedsCylinder covers the guard: with no cylindrical face reported (the bore cut
// failed to leave an analytic cylinder), the thread step fails loudly instead of threading the
// wrong surface.
func TestHexNutThreadNeedsCylinder(t *testing.T) {
	h := &fakeHost{dof: 0, noCylinder: true}
	err := (HexNut{}).Build(newBuilder(h, catalog.UnitsMillimetre), hexNutMember(8, 1.25, 13, 6.8))
	if err == nil || !strings.Contains(err.Error(), "cylindrical face") {
		t.Fatalf("Build error = %v, want it to mention the missing cylindrical face", err)
	}
	if len(h.threads) != 0 {
		t.Errorf("threads = %+v, want none when the bore face is missing", h.threads)
	}
}

// TestHexNutUnderConstrainedFails ensures a non-zero DOF aborts the build before any solid is
// extruded, so a floppy prism is caught.
func TestHexNutUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 3}
	err := (HexNut{}).Build(newBuilder(h, catalog.UnitsMillimetre), hexNutMember(8, 1.25, 13, 6.8))
	if err == nil {
		t.Fatal("Build accepted an under-constrained hexagon; want an error")
	}
	if h.hasMethod(wire.MethodFeaturesAdd) {
		t.Error("a feature ran on an under-constrained sketch; the DOF check must gate it")
	}
}

// TestDefaultRegistryHasHexNut checks the generator is wired into the built-in set so a hex-nut
// family resolves at placement.
func TestDefaultRegistryHasHexNut(t *testing.T) {
	g, ok := DefaultRegistry().Get("hex_nut")
	if !ok || g.Kind() != "hex_nut" {
		t.Fatalf("DefaultRegistry hex_nut = (%v,%v), want the HexNut generator", g, ok)
	}
}
