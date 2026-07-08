// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// flatBarMember builds a synthetic resolved member with the flat-bar columns the generator
// drives: nominal width b → width, thickness a → thickness, and the stock length l → length.
func flatBarMember(width, thickness, length float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-flat", Generator: "flat_bar", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"b", "a"},
		Columns: []catalog.Column{
			{Name: "b", Param: "width", Type: catalog.ColumnLength},
			{Name: "a", Param: "thickness", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{Key: "b=40,a=8", Values: map[string]float64{"b": width, "a": thickness, "l": length}}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestFlatBarBuildsCenteredProfileExtrude is the C1 acceptance check: every dimension is
// parameter-driven, the section is a centred rectangle (four axis constraints + a fixed origin,
// sized by two side dimensions and centred by two half-size offset dimensions), and it extrudes
// to the length parameter as a single new solid.
func TestFlatBarBuildsCenteredProfileExtrude(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (FlatBar{}).Build(newBuilder(h, catalog.UnitsMillimetre), flatBarMember(40, 8, 6000)); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	assertParam(t, h.added, "width", "40 mm")
	assertParam(t, h.added, "thickness", "8 mm")
	assertParam(t, h.added, "length", "6000 mm")

	if len(h.extrudes) != 1 || h.extrudes[0].Distance != "length" || h.extrudes[0].Operation != "new" {
		t.Errorf("extrudes = %+v, want one length/new prism", h.extrudes)
	}
	if len(h.threads) != 0 {
		t.Errorf("threads = %d, want 0 (a flat bar carries no thread)", len(h.threads))
	}

	// The rectangle is a rigid, centred, parameter-driven section: two horizontals + two
	// verticals make it a rectangle, a fix grounds the centre point.
	wantKinds := map[string]int{"horizontal": 2, "vertical": 2, "fix": 1}
	got := map[string]int{}
	for _, k := range h.constraints {
		got[k]++
	}
	for k, n := range wantKinds {
		if got[k] != n {
			t.Errorf("constraint %q count = %d, want %d (all: %v)", k, got[k], n, h.constraints)
		}
	}

	// Size comes from the width/thickness dimensions; centring from the two half-size offsets —
	// so editing width or thickness re-drives the section symmetrically about the origin.
	for _, expr := range []string{"width", "thickness", "(thickness) / 2", "(width) / 2"} {
		if !hasDimension(h.dimensions, expr) {
			t.Errorf("dimension %q not applied; have %+v", expr, h.dimensions)
		}
	}
}

// TestFlatBarUnderConstrainedFails ensures a non-zero DOF aborts the build before it extrudes a
// floppy section.
func TestFlatBarUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 3}
	if err := (FlatBar{}).Build(newBuilder(h, catalog.UnitsMillimetre), flatBarMember(40, 8, 6000)); err == nil {
		t.Fatal("Build accepted an under-constrained profile; want an error")
	}
	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 when the profile is under-constrained", len(h.extrudes))
	}
}

// TestFlatBarBuildErrorsPropagate injects a host failure at each wire method the build uses and
// asserts the error surfaces rather than a half-built part.
func TestFlatBarBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodSketchCreate, wire.MethodSketchAddEntity,
		wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	for _, m := range methods {
		h := &fakeHost{dof: 0, failMethod: m}
		if err := (FlatBar{}).Build(newBuilder(h, catalog.UnitsMillimetre), flatBarMember(40, 8, 6000)); err == nil {
			t.Errorf("failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// hasDimension reports whether any recorded dimension carries the given expression.
func hasDimension(dims []wire.AddDimensionArgs, expr string) bool {
	for _, d := range dims {
		if d.Expression == expr {
			return true
		}
	}
	return false
}

// TestDefaultRegistryHasFlatBar checks the generator is wired into the built-in set so a flat-bar
// family resolves at placement.
func TestDefaultRegistryHasFlatBar(t *testing.T) {
	g, ok := DefaultRegistry().Get("flat_bar")
	if !ok || g.Kind() != "flat_bar" {
		t.Fatalf("DefaultRegistry flat_bar = (%v,%v), want the FlatBar generator", g, ok)
	}
}
