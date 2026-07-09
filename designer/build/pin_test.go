// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// pinMember builds a synthetic resolved dowel-pin member: diameter d, length l.
func pinMember(diameter, length float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-pin", Generator: "pin", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"d", "l"},
		Columns: []catalog.Column{
			{Name: "d", Param: "diameter", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{Key: "d=8,l=40", Values: map[string]float64{"d": diameter, "l": length}}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestPinBuildsChamferedCylinder is the acceptance check: the diameter/length are published, the
// end chamfer is derived, and a chamfered rod half-section is revolved about the Z axis as one new
// solid (the two end chamfers give the dowel its lead-in).
func TestPinBuildsChamferedCylinder(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (Pin{}).Build(newBuilder(h, catalog.UnitsMillimetre), pinMember(8, 40)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "diameter", "8 mm")
	assertParam(t, h.added, "length", "40 mm")
	assertParam(t, h.added, "end_chamfer", "diameter * 0.1")
	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %+v, want none (the pin is revolved)", h.extrudes)
	}
	if len(h.revolves) != 1 || h.revolves[0].AxisRef != "origin/axis/z" ||
		h.revolves[0].Angle != "360 deg" || h.revolves[0].Operation != "new" {
		t.Errorf("revolves = %+v, want one z / 360 deg / new", h.revolves)
	}
	// The chamfer feet and hypotenuses are parameter expressions, not literal coordinates.
	for _, expr := range []string{"length", "(diameter) / 2 - (end_chamfer)", "(end_chamfer) * sqrt(2)"} {
		if !hasDimension(h.dimensions, expr) {
			t.Errorf("rod dimension %q not applied; have %+v", expr, h.dimensions)
		}
	}
}

// TestPinUnderConstrainedFails ensures a non-zero DOF aborts the build.
func TestPinUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 1}
	if err := (Pin{}).Build(newBuilder(h, catalog.UnitsMillimetre), pinMember(8, 40)); err == nil {
		t.Fatal("Build accepted an under-constrained pin; want an error")
	}
}

// TestPinBuildErrorsPropagate injects a host failure at each wire method the build uses.
func TestPinBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodSketchCreate, wire.MethodSketchAddEntity,
		wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	for _, m := range methods {
		h := &fakeHost{dof: 0, failMethod: m}
		if err := (Pin{}).Build(newBuilder(h, catalog.UnitsMillimetre), pinMember(8, 40)); err == nil {
			t.Errorf("failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// TestDefaultRegistryHasPin checks the generator is wired into the built-in set.
func TestDefaultRegistryHasPin(t *testing.T) {
	g, ok := DefaultRegistry().Get("pin")
	if !ok || g.Kind() != "pin" {
		t.Fatalf("DefaultRegistry pin = (%v,%v), want the Pin generator", g, ok)
	}
}
