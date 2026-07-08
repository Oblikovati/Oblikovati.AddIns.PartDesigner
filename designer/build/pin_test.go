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

// TestPinBuildsCylinder is the D2 acceptance check: the diameter is published and extruded to
// `length` as one new cylinder.
func TestPinBuildsCylinder(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (Pin{}).Build(newBuilder(h, catalog.UnitsMillimetre), pinMember(8, 40)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "diameter", "8 mm")
	assertParam(t, h.added, "length", "40 mm")
	if len(h.extrudes) != 1 || h.extrudes[0].Distance != "length" || h.extrudes[0].Operation != "new" {
		t.Errorf("extrudes = %+v, want one length/new cylinder", h.extrudes)
	}
	if h.circleRadius != "(diameter)/2" {
		t.Errorf("circle radius = %q, want the diameter half", h.circleRadius)
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
