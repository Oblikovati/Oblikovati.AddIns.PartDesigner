// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// angleMember builds a synthetic resolved angle member: a text designation plus the two leg
// lengths, the thickness, and the stock length.
func angleMember(designation string, legA, legB, thickness, r1, r2, length float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-angle", Generator: "angle", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"designation"},
		Columns: []catalog.Column{
			{Name: "designation", Param: "designation", Type: catalog.ColumnText},
			{Name: "a", Param: "leg_a", Type: catalog.ColumnLength},
			{Name: "b", Param: "leg_b", Type: catalog.ColumnLength},
			{Name: "t", Param: "thickness", Type: catalog.ColumnLength},
			{Name: "r1", Param: "root_radius", Type: catalog.ColumnLength},
			{Name: "r2", Param: "toe_radius", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{
			Key:    "designation=" + designation,
			Values: map[string]float64{"a": legA, "b": legB, "t": thickness, "r1": r1, "r2": r2, "l": length},
			Labels: map[string]string{"designation": designation},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestAngleBuildsHeelAnchoredSection is the C3 angle acceptance check: the five section parameters
// are published, the heel is fixed at the origin, the six edges + three arcs (root fillet + two toe
// radii) are oriented and centre-pinned, the legs/thickness are offset-driven, and it extrudes to
// `length`.
func TestAngleBuildsHeelAnchoredSection(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (Angle{}).Build(newBuilder(h, catalog.UnitsMillimetre),
		angleMember("L 60x40x5", 60, 40, 5, 6, 3, 6000)); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	assertParam(t, h.added, "leg_a", "60 mm")
	assertParam(t, h.added, "leg_b", "40 mm")
	assertParam(t, h.added, "thickness", "5 mm")
	assertParam(t, h.added, "root_radius", "6 mm")
	assertParam(t, h.added, "toe_radius", "3 mm")
	assertParam(t, h.added, "length", "6000 mm")

	if len(h.extrudes) != 1 || h.extrudes[0].Distance != "length" || h.extrudes[0].Operation != "new" {
		t.Errorf("extrudes = %+v, want one length/new L-section", h.extrudes)
	}
	// Heel fixed at the origin; the legs and thicknesses are offset-driven.
	if got := countKind(h.constraints, "fix"); got != 1 {
		t.Errorf("fix count = %d, want 1 (the heel)", got)
	}
	// 6 edge orientations (3 H + 3 V) + 3 arc centre-pins (2 each = 6).
	if got := countKind(h.constraints, "horizontal") + countKind(h.constraints, "vertical"); got != 12 {
		t.Errorf("axis alignments = %d, want 12 (6 edges + 3 arc pins)", got)
	}
	// One shared toe radius (EqualRadius) + the root and toe radius dimensions.
	if got := countKind(h.constraints, "equalRadius"); got != 1 {
		t.Errorf("equalRadius = %d, want 1 (the two toes)", got)
	}
	for _, expr := range []string{"leg_a", "leg_b", "thickness", "root_radius", "toe_radius"} {
		if !hasDimension(h.dimensions, expr) {
			t.Errorf("dimension %q not applied; have %+v", expr, h.dimensions)
		}
	}
}

// TestAngleUnderConstrainedFails ensures a non-zero DOF aborts before extruding.
func TestAngleUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 2}
	if err := (Angle{}).Build(newBuilder(h, catalog.UnitsMillimetre),
		angleMember("L 60x40x5", 60, 40, 5, 6, 3, 6000)); err == nil {
		t.Fatal("Build accepted an under-constrained angle; want an error")
	}
	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 when under-constrained", len(h.extrudes))
	}
}

// TestAngleBuildErrorsPropagate injects a host failure at each wire method the build uses.
func TestAngleBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodSketchCreate, wire.MethodSketchAddEntity,
		wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	for _, m := range methods {
		h := &fakeHost{dof: 0, failMethod: m}
		if err := (Angle{}).Build(newBuilder(h, catalog.UnitsMillimetre),
			angleMember("L 60x40x5", 60, 40, 5, 6, 3, 6000)); err == nil {
			t.Errorf("failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// TestDefaultRegistryHasAngle checks the generator is wired into the built-in set.
func TestDefaultRegistryHasAngle(t *testing.T) {
	g, ok := DefaultRegistry().Get("angle")
	if !ok || g.Kind() != "angle" {
		t.Fatalf("DefaultRegistry angle = (%v,%v), want the Angle generator", g, ok)
	}
}
