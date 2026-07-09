// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// gibKeyMember builds a synthetic resolved gib-head-key member: width b, body height h, gib-nose
// height h2 and length l (the DIN 6887 16×10, h2=16, 56 long).
func gibKeyMember(b, h, h2, l float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-gib-key", Generator: "gib_head_key", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"b", "h", "l"},
		Columns: []catalog.Column{
			{Name: "b", Param: "width", Type: catalog.ColumnLength},
			{Name: "h", Param: "height", Type: catalog.ColumnLength},
			{Name: "h2", Param: "head_height", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{
			Key:    "b=16,h=10,l=56",
			Values: map[string]float64{"b": b, "h": h, "h2": h2, "l": l},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestGibHeadKeyBuildsSilhouetteExtrude is the acceptance check: the width/height/head-height/length
// are published, the nose length is derived, the five-point gib silhouette is fully constrained
// (fix + two horizontal + two vertical edges + four distances) and extruded across the width.
func TestGibHeadKeyBuildsSilhouetteExtrude(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (GibHeadKey{}).Build(newBuilder(h, catalog.UnitsMillimetre), gibKeyMember(16, 10, 16, 56)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "width", "16 mm")
	assertParam(t, h.added, "height", "10 mm")
	assertParam(t, h.added, "head_height", "16 mm")
	assertParam(t, h.added, "length", "56 mm")
	// The gib nose length is a derived proportion of the grounded nose height.
	assertParam(t, h.added, "nose_length", "head_height * 0.9")

	// The silhouette is extruded ACROSS the width (not to length — length lives in the sketch).
	if len(h.extrudes) != 1 || h.extrudes[0].Distance != "width" || h.extrudes[0].Operation != "new" {
		t.Fatalf("extrudes = %+v, want one width/new gib silhouette", h.extrudes)
	}
	// Body length, body height, nose height, and the body-top run that foots the nose.
	for _, expr := range []string{"length", "height", "head_height", "(length) - (nose_length)"} {
		if !hasDimension(h.dimensions, expr) {
			t.Errorf("gib dimension %q not applied; have %+v", expr, h.dimensions)
		}
	}
	// Two horizontal edges (bottom, body-top) and two vertical edges (front, nose-back).
	if got := countKind(h.constraints, "horizontal"); got != 2 {
		t.Errorf("horizontal constraints = %d, want 2", got)
	}
	if got := countKind(h.constraints, "vertical"); got != 2 {
		t.Errorf("vertical constraints = %d, want 2", got)
	}
}

// TestGibHeadKeyUnderConstrainedFails ensures a non-zero DOF aborts before the solid is extruded.
func TestGibHeadKeyUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 3}
	if err := (GibHeadKey{}).Build(newBuilder(h, catalog.UnitsMillimetre), gibKeyMember(16, 10, 16, 56)); err == nil {
		t.Fatal("Build accepted an under-constrained gib key; want an error")
	}
	if len(h.extrudes) != 0 {
		t.Errorf("extruded despite bad DOF: extrudes=%d", len(h.extrudes))
	}
}

// TestDefaultRegistryHasGibHeadKey checks the generator is wired into the built-in set.
func TestDefaultRegistryHasGibHeadKey(t *testing.T) {
	g, ok := DefaultRegistry().Get("gib_head_key")
	if !ok || g.Kind() != "gib_head_key" {
		t.Fatalf("DefaultRegistry gib_head_key = (%v,%v), want the GibHeadKey generator", g, ok)
	}
}
