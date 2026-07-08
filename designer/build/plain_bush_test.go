// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// bushMember builds a synthetic resolved plain-bush member: bore d, outer diameter D, length L.
func bushMember(bore, outerDia, length float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-plain-bush", Generator: "plain_bush", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"d", "L"},
		Columns: []catalog.Column{
			{Name: "d", Param: "bore", Type: catalog.ColumnLength},
			{Name: "D", Param: "outer_dia", Type: catalog.ColumnLength},
			{Name: "L", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{
			Key:    "d=20,L=20",
			Values: map[string]float64{"d": bore, "D": outerDia, "L": length},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestPlainBushBuildsSleeve is the E2 plain-bush acceptance check: id/od/length published, the
// outside diameter extruded as a new solid to `length`, the bore cut through — one hollow sleeve.
func TestPlainBushBuildsSleeve(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (PlainBush{}).Build(newBuilder(h, catalog.UnitsMillimetre), bushMember(20, 26, 20)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "bore", "20 mm")
	assertParam(t, h.added, "outer_dia", "26 mm")
	assertParam(t, h.added, "length", "20 mm")

	if len(h.extrudes) != 2 {
		t.Fatalf("extrudes = %d, want 2 (outer new + bore cut)", len(h.extrudes))
	}
	if h.extrudes[0].Distance != "length" || h.extrudes[0].Operation != "new" {
		t.Errorf("outer extrude = %+v, want length/new", h.extrudes[0])
	}
	if h.extrudes[1].Distance != "length" || h.extrudes[1].Operation != "cut" {
		t.Errorf("bore extrude = %+v, want length/cut", h.extrudes[1])
	}
	if h.circleRadius != "(bore)/2" {
		t.Errorf("last circle radius = %q, want the bore half", h.circleRadius)
	}
}

// TestPlainBushUnderConstrainedFails ensures a non-zero DOF aborts the sleeve.
func TestPlainBushUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 1}
	if err := (PlainBush{}).Build(newBuilder(h, catalog.UnitsMillimetre), bushMember(20, 26, 20)); err == nil {
		t.Fatal("Build accepted an under-constrained bush; want an error")
	}
}

// TestDefaultRegistryHasPlainBush checks the generator is wired into the built-in set.
func TestDefaultRegistryHasPlainBush(t *testing.T) {
	g, ok := DefaultRegistry().Get("plain_bush")
	if !ok || g.Kind() != "plain_bush" {
		t.Fatalf("DefaultRegistry plain_bush = (%v,%v), want the PlainBush generator", g, ok)
	}
}
