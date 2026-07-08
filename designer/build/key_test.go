// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// keyMember builds a synthetic resolved parallel-key member: width b, height h, length l.
func keyMember(width, height, length float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-key", Generator: "key", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"b", "h", "l"},
		Columns: []catalog.Column{
			{Name: "b", Param: "width", Type: catalog.ColumnLength},
			{Name: "h", Param: "height", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{Key: "b=12,h=8,l=40", Values: map[string]float64{"b": width, "h": height, "l": length}}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestKeyBuildsCrossSectionExtrude is the D1 acceptance check: the key's width×height cross-section
// is published, centred and extruded to `length` as one new solid.
func TestKeyBuildsCrossSectionExtrude(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (Key{}).Build(newBuilder(h, catalog.UnitsMillimetre), keyMember(12, 8, 40)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "width", "12 mm")
	assertParam(t, h.added, "height", "8 mm")
	assertParam(t, h.added, "length", "40 mm")
	if len(h.extrudes) != 1 || h.extrudes[0].Distance != "length" || h.extrudes[0].Operation != "new" {
		t.Errorf("extrudes = %+v, want one length/new key", h.extrudes)
	}
	if !hasDimension(h.dimensions, "width") || !hasDimension(h.dimensions, "height") {
		t.Errorf("section dimensions missing; have %+v", h.dimensions)
	}
}

// TestKeyUnderConstrainedFails ensures a non-zero DOF aborts the build.
func TestKeyUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 2}
	if err := (Key{}).Build(newBuilder(h, catalog.UnitsMillimetre), keyMember(12, 8, 40)); err == nil {
		t.Fatal("Build accepted an under-constrained key; want an error")
	}
}

// TestKeyBuildErrorsPropagate injects a host failure at each wire method the build uses.
func TestKeyBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodSketchCreate, wire.MethodSketchAddEntity,
		wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	for _, m := range methods {
		h := &fakeHost{dof: 0, failMethod: m}
		if err := (Key{}).Build(newBuilder(h, catalog.UnitsMillimetre), keyMember(12, 8, 40)); err == nil {
			t.Errorf("failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// TestDefaultRegistryHasKey checks the generator is wired into the built-in set.
func TestDefaultRegistryHasKey(t *testing.T) {
	g, ok := DefaultRegistry().Get("key")
	if !ok || g.Kind() != "key" {
		t.Fatalf("DefaultRegistry key = (%v,%v), want the Key generator", g, ok)
	}
}
