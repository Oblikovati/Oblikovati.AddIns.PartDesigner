// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// clevisMember builds a synthetic resolved clevis-pin member: shank d, head dk, head height k,
// cotter hole d1, hole end distance le, length l (the ISO 2341 10×40, dk=18, hole 3.2 at le=4.5).
func clevisMember(d, dk, k, d1, le, l float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-clevis", Generator: "clevis_pin", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"d", "l"},
		Columns: []catalog.Column{
			{Name: "d", Param: "shank_dia", Type: catalog.ColumnLength},
			{Name: "dk", Param: "head_dia", Type: catalog.ColumnLength},
			{Name: "k", Param: "head_height", Type: catalog.ColumnLength},
			{Name: "d1", Param: "hole_dia", Type: catalog.ColumnLength},
			{Name: "le", Param: "hole_end_dist", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{
			Key:    "d=10,l=40",
			Values: map[string]float64{"d": d, "dk": dk, "k": k, "d1": d1, "le": le, "l": l},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestClevisPinBuildsHeadShankAndHole is the acceptance check: parameters published, hole depth
// derived, head extruded up (new), shank down (join), and the cotter hole cut (cut) through the
// shank, all parameter-driven.
func TestClevisPinBuildsHeadShankAndHole(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (ClevisPin{}).Build(newBuilder(h, catalog.UnitsMillimetre), clevisMember(10, 18, 4, 3.2, 4.5, 40)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "shank_dia", "10 mm")
	assertParam(t, h.added, "head_dia", "18 mm")
	assertParam(t, h.added, "hole_dia", "3.2 mm")
	assertParam(t, h.added, "hole_end_dist", "4.5 mm")
	// The hole sits hole_end_dist from the far end → this far below the head plane.
	assertParam(t, h.added, "hole_depth", "length - hole_end_dist")

	if len(h.extrudes) != 3 {
		t.Fatalf("extrudes = %d, want 3 (head, shank, hole cut)", len(h.extrudes))
	}
	head, shank, hole := h.extrudes[0], h.extrudes[1], h.extrudes[2]
	if head.Distance != "head_height" || head.Operation != "new" {
		t.Errorf("head extrude = %+v, want head_height/new", head)
	}
	if shank.Distance != "length" || shank.Operation != "join" || shank.Direction != "negative" {
		t.Errorf("shank extrude = %+v, want length/join/negative", shank)
	}
	if hole.Distance != "head_dia" || hole.Operation != "cut" || hole.Direction != "symmetric" {
		t.Errorf("hole cut = %+v, want head_dia/cut/symmetric", hole)
	}
	// The cotter hole is placed by a parameter distance, not a literal coordinate.
	if !hasDimension(h.dimensions, "hole_depth") {
		t.Errorf("hole not placed by hole_depth; have %+v", h.dimensions)
	}
}

// TestClevisPinUnderConstrainedFails ensures a non-zero DOF aborts before any solid is made.
func TestClevisPinUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 2}
	if err := (ClevisPin{}).Build(newBuilder(h, catalog.UnitsMillimetre), clevisMember(10, 18, 4, 3.2, 4.5, 40)); err == nil {
		t.Fatal("Build accepted an under-constrained clevis pin; want an error")
	}
	if len(h.extrudes) != 0 {
		t.Errorf("extruded despite bad DOF: extrudes=%d", len(h.extrudes))
	}
}

// TestDefaultRegistryHasClevisPin checks the generator is wired into the built-in set.
func TestDefaultRegistryHasClevisPin(t *testing.T) {
	g, ok := DefaultRegistry().Get("clevis_pin")
	if !ok || g.Kind() != "clevis_pin" {
		t.Fatalf("DefaultRegistry clevis_pin = (%v,%v), want the ClevisPin generator", g, ok)
	}
}
