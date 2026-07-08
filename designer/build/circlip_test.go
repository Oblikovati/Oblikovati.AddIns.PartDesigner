// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// circlipMember builds a synthetic resolved retaining-ring member: nominal, ring bore/outer, s.
func circlipMember(nominal, innerDia, outerDia, thickness float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-circlip", Generator: "circlip", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"d"},
		Columns: []catalog.Column{
			{Name: "d", Param: "nominal_dia", Type: catalog.ColumnLength},
			{Name: "di", Param: "inner_dia", Type: catalog.ColumnLength},
			{Name: "do", Param: "outer_dia", Type: catalog.ColumnLength},
			{Name: "s", Param: "thickness", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{
			Key:    "d=20",
			Values: map[string]float64{"d": nominal, "di": innerDia, "do": outerDia, "s": thickness},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestCirclipRevolvesSplitRing is the D3 acceptance check: the ring parameters are published and a
// radial section is revolved through the split-gap angle (under a full turn) as one new solid.
func TestCirclipRevolvesSplitRing(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), circlipMember(20, 19, 27, 1.2)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "inner_dia", "19 mm")
	assertParam(t, h.added, "outer_dia", "27 mm")
	assertParam(t, h.added, "thickness", "1.2 mm")

	if len(h.revolves) != 1 {
		t.Fatalf("revolves = %d, want 1 (the split ring)", len(h.revolves))
	}
	rv := h.revolves[0]
	if rv.Angle != splitGapAngle || rv.Operation != "new" || rv.AxisRef != "origin/axis/z" {
		t.Errorf("revolve = %+v, want %s about z / new", rv, splitGapAngle)
	}
	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 (the ring is revolved, not extruded)", len(h.extrudes))
	}
}

// TestCirclipUnderConstrainedFails ensures a non-zero DOF aborts before the revolve.
func TestCirclipUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 3}
	if err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), circlipMember(20, 19, 27, 1.2)); err == nil {
		t.Fatal("Build accepted an under-constrained ring; want an error")
	}
	if len(h.revolves) != 0 {
		t.Errorf("revolves = %d, want 0 when under-constrained", len(h.revolves))
	}
}

// TestCirclipBuildErrorsPropagate injects a host failure at each wire method the build uses.
func TestCirclipBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodSketchCreate, wire.MethodSketchAddEntity,
		wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	for _, m := range methods {
		h := &fakeHost{dof: 0, failMethod: m}
		if err := (Circlip{}).Build(newBuilder(h, catalog.UnitsMillimetre), circlipMember(20, 19, 27, 1.2)); err == nil {
			t.Errorf("failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// TestDefaultRegistryHasCirclip checks the generator is wired into the built-in set.
func TestDefaultRegistryHasCirclip(t *testing.T) {
	g, ok := DefaultRegistry().Get("circlip")
	if !ok || g.Kind() != "circlip" {
		t.Fatalf("DefaultRegistry circlip = (%v,%v), want the Circlip generator", g, ok)
	}
}
