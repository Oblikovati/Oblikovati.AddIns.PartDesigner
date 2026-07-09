// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"math"
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// splitMember builds a synthetic resolved split-pin member: nominal d, end offset a, eye height b,
// eye width c, length l (the ISO 1234 3.2×32).
func splitMember(d, a, b, c, l float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-split", Generator: "split_pin", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"d", "l"},
		Columns: []catalog.Column{
			{Name: "d", Param: "d", Type: catalog.ColumnLength},
			{Name: "a", Param: "end_offset", Type: catalog.ColumnLength},
			{Name: "b", Param: "eye_height", Type: catalog.ColumnLength},
			{Name: "c", Param: "eye_width", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{
			Key:    "d=3.2,l=32",
			Values: map[string]float64{"d": d, "a": a, "b": b, "c": c, "l": l},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestSplitPinBuildsHairpinSweep is the acceptance check: parameters published, the wire diameter
// derived as d/2, and a single sweep of the wire circle along a hairpin path — leg 1 up from the
// origin, a semicircular eye over the top, leg 2 back down — grounded in the ISO 1234 dims.
func TestSplitPinBuildsHairpinSweep(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (SplitPin{}).Build(newBuilder(h, catalog.UnitsMillimetre), splitMember(3.2, 3.2, 6.4, 5.8, 32)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "d", "3.2 mm")
	assertParam(t, h.added, "eye_width", "5.8 mm")
	assertParam(t, h.added, "length", "32 mm")
	// The wire is the round-wire approximation of the two half-round prongs (each ~d/2).
	assertParam(t, h.added, "wire_dia", "d / 2")

	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 (the pin is swept, not extruded)", len(h.extrudes))
	}
	if len(h.sweeps) != 1 {
		t.Fatalf("sweeps = %d, want 1 (the folded wire)", len(h.sweeps))
	}
	sw := h.sweeps[0]
	if sw.Operation != "new" {
		t.Errorf("sweep operation = %q, want new", sw.Operation)
	}
	p := sw.PathPoints
	if len(p) < 4 {
		t.Fatalf("sweep path has %d points, want a leg+eye+leg polyline", len(p))
	}
	// The path starts at the origin (= the profile circle's centre) and ends at leg 2's foot, the
	// eye width c apart (cm) and extended the offset end `a` below leg 1. The eye bulges above l.
	const cCm, lCm, aCm = 0.58, 3.2, 0.32
	if !pointNear(p[0], 0, 0, 0) {
		t.Errorf("path start = %v, want the origin", p[0])
	}
	if !pointNear(p[len(p)-1], cCm, 0, -aCm) {
		t.Errorf("path end = %v, want leg 2 foot at x=c=%.2f, z=-a=%.2f cm", p[len(p)-1], cCm, aCm)
	}
	if zMax := maxZ(p); zMax <= lCm {
		t.Errorf("eye apex z = %.3f cm, want > leg length %.2f (the eye must rise above the legs)", zMax, lCm)
	}
}

// TestSplitPinUnderConstrainedFails ensures a non-zero DOF on the profile aborts before any solid.
func TestSplitPinUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 2}
	if err := (SplitPin{}).Build(newBuilder(h, catalog.UnitsMillimetre), splitMember(3.2, 3.2, 6.4, 5.8, 32)); err == nil {
		t.Fatal("Build accepted an under-constrained split pin; want an error")
	}
	if len(h.sweeps) != 0 {
		t.Errorf("swept despite bad DOF: sweeps=%d", len(h.sweeps))
	}
}

// TestDefaultRegistryHasSplitPin checks the generator is wired into the built-in set.
func TestDefaultRegistryHasSplitPin(t *testing.T) {
	g, ok := DefaultRegistry().Get("split_pin")
	if !ok || g.Kind() != "split_pin" {
		t.Fatalf("DefaultRegistry split_pin = (%v,%v), want the SplitPin generator", g, ok)
	}
}

func pointNear(p []float64, x, y, z float64) bool {
	return len(p) == 3 && math.Abs(p[0]-x) < 1e-6 && math.Abs(p[1]-y) < 1e-6 && math.Abs(p[2]-z) < 1e-6
}

func maxZ(pts [][]float64) float64 {
	m := math.Inf(-1)
	for _, p := range pts {
		if len(p) == 3 && p[2] > m {
			m = p[2]
		}
	}
	return m
}
