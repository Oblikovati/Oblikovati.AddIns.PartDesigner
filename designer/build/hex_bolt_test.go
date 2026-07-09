// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"strings"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// hexBoltMember builds a synthetic resolved member with the ISO 4017 columns the HexBolt
// generator drives geometry against.
func hexBoltMember(d, pitch, s, k, l, b float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-hex", Generator: "hex_bolt", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"d", "l"},
		Columns: []catalog.Column{
			{Name: "d", Param: "nominal_dia", Type: catalog.ColumnLength},
			{Name: "P", Param: "thread_pitch", Type: catalog.ColumnLength},
			{Name: "s", Param: "across_flats", Type: catalog.ColumnLength},
			{Name: "k", Param: "head_height", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
			{Name: "b", Param: "thread_length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{Key: "d=8,l=40", Values: map[string]float64{
			"d": d, "P": pitch, "s": s, "k": k, "l": l, "b": b,
		}}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestHexBoltBuildsParametricBolt is the B1 acceptance check: an M8 hex bolt is realized as a
// DOF-0 hexagonal head (extruded up) + cylindrical shank (extruded down, joined) + cosmetic
// thread, with every dimension bound to a published parameter (never a literal), so editing
// the size re-drives the part.
func TestHexBoltBuildsParametricBolt(t *testing.T) {
	h := &fakeHost{dof: 0}
	rm := hexBoltMember(8, 1.25, 13, 5.3, 40, 40)
	if err := (HexBolt{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	assertParam(t, h.added, "across_flats", "13 mm")
	assertParam(t, h.added, "head_height", "5.3 mm")
	assertParam(t, h.added, "nominal_dia", "8 mm")
	assertParam(t, h.added, "length", "40 mm")

	// Hexagon: centre fixed, +X flat vertical (rotation), across-corners dimensioned; then the
	// shank circle's centre fixed. Both dimensions are parameter expressions, not coordinates.
	wantCon := []string{"fix", "vertical", "fix"}
	if strings.Join(h.constraints, ",") != strings.Join(wantCon, ",") {
		t.Errorf("constraints = %v, want %v", h.constraints, wantCon)
	}
	if len(h.dimensions) != 2 ||
		h.dimensions[0].Kind != "distance" || h.dimensions[0].Expression != "(across_flats) / cos(30 deg)" ||
		h.dimensions[1].Kind != "diameter" || h.dimensions[1].Expression != "nominal_dia" {
		t.Errorf("dimensions = %+v, want across-corners then diameter, both parameter-driven", h.dimensions)
	}

	// Head extrudes up (new), shank extrudes down (join, negative) — parameter-driven depths.
	if len(h.extrudes) != 2 {
		t.Fatalf("extrudes = %d, want 2 (head + shank)", len(h.extrudes))
	}
	if h.extrudes[0].Distance != "head_height" || h.extrudes[0].Operation != "new" || h.extrudes[0].Direction != "" {
		t.Errorf("head extrude = %+v, want head_height/new/positive", h.extrudes[0])
	}
	if h.extrudes[1].Distance != "length" || h.extrudes[1].Operation != "join" || h.extrudes[1].Direction != "negative" {
		t.Errorf("shank extrude = %+v, want length/join/negative", h.extrudes[1])
	}

	// Cosmetic thread on the shank's cylindrical face, sized from d and P.
	if len(h.threads) != 1 || h.threads[0].FaceRef != "shank-cyl" ||
		h.threads[0].Designation != "M8x1.25" || h.threads[0].Cut {
		t.Errorf("threads = %+v, want one cosmetic M8x1.25 on shank-cyl", h.threads)
	}
}

// TestHexBoltThreadNeedsCylinder covers the guard: with no cylindrical face reported, the
// thread step fails loudly instead of threading the wrong surface.
func TestHexBoltThreadNeedsCylinder(t *testing.T) {
	h := &fakeHost{dof: 0, noCylinder: true}
	err := (HexBolt{}).Build(newBuilder(h, catalog.UnitsMillimetre), hexBoltMember(8, 1.25, 13, 5.3, 40, 40))
	if err == nil || !strings.Contains(err.Error(), "cylindrical face") {
		t.Fatalf("Build error = %v, want it to mention the missing cylindrical face", err)
	}
	if len(h.threads) != 0 {
		t.Errorf("threads = %+v, want none when the shank face is missing", h.threads)
	}
}

// TestHexBoltUnderConstrainedFails ensures a non-zero DOF aborts the build before any solid is
// extruded, so a floppy head is caught.
func TestHexBoltUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 3}
	err := (HexBolt{}).Build(newBuilder(h, catalog.UnitsMillimetre), hexBoltMember(8, 1.25, 13, 5.3, 40, 40))
	if err == nil {
		t.Fatal("Build accepted an under-constrained hexagon; want an error")
	}
	if h.hasMethod(wire.MethodFeaturesAdd) {
		t.Error("a feature ran on an under-constrained sketch; the DOF check must gate it")
	}
}

// TestHexBoltHeadAddFails propagates a host failure while adding the head polygon (rather than
// building a headless bolt).
func TestHexBoltHeadAddFails(t *testing.T) {
	h := &fakeHost{dof: 0, failMethod: wire.MethodSketchAddEntity}
	err := (HexBolt{}).Build(newBuilder(h, catalog.UnitsMillimetre), hexBoltMember(8, 1.25, 13, 5.3, 40, 40))
	if err == nil || !strings.Contains(err.Error(), "add hexagon") {
		t.Fatalf("Build error = %v, want it to mention adding the hexagon", err)
	}
}

// TestHexBoltRejectsMalformedPolygon covers the guard that a polygon reply must carry the six
// corners plus the centre; a short reply is caught rather than indexing out of range.
func TestHexBoltRejectsMalformedPolygon(t *testing.T) {
	h := &fakeHost{dof: 0, shortPolygon: true}
	err := (HexBolt{}).Build(newBuilder(h, catalog.UnitsMillimetre), hexBoltMember(8, 1.25, 13, 5.3, 40, 40))
	if err == nil || !strings.Contains(err.Error(), "want 6 corners") {
		t.Fatalf("Build error = %v, want it to reject the malformed polygon", err)
	}
}

// TestHexBoltReferenceKeysError propagates a host failure while reading the reference keys the
// thread step needs to find the shank face.
func TestHexBoltReferenceKeysError(t *testing.T) {
	h := &fakeHost{dof: 0, failMethod: wire.MethodModelReferenceKeys}
	err := (HexBolt{}).Build(newBuilder(h, catalog.UnitsMillimetre), hexBoltMember(8, 1.25, 13, 5.3, 40, 40))
	if err == nil || !strings.Contains(err.Error(), "reference keys") {
		t.Fatalf("Build error = %v, want it to mention reading reference keys", err)
	}
}

// TestMetricThreadDesignation covers the ISO metric designation rendering across coarse sizes.
func TestMetricThreadDesignation(t *testing.T) {
	for _, tc := range []struct {
		d, p float64
		want string
	}{
		{6, 1.0, "M6x1"},
		{8, 1.25, "M8x1.25"},
		{10, 1.5, "M10x1.5"},
		{12, 1.75, "M12x1.75"},
	} {
		got := metricThreadDesignation(hexBoltMember(tc.d, tc.p, 0, 0, 0, 0))
		if got != tc.want {
			t.Errorf("designation(d=%g,P=%g) = %q, want %q", tc.d, tc.p, got, tc.want)
		}
	}
}

// TestDefaultRegistryHasHexBolt checks the generator is wired into the built-in set so a
// hex-bolt family resolves at placement.
func TestDefaultRegistryHasHexBolt(t *testing.T) {
	g, ok := DefaultRegistry().Get("hex_bolt")
	if !ok || g.Kind() != "hex_bolt" {
		t.Fatalf("DefaultRegistry hex_bolt = (%v,%v), want the HexBolt generator", g, ok)
	}
}

// assertParam fails unless params contains name=expr exactly once.
func assertParam(t *testing.T, params []wire.ParameterSetArgs, name, expr string) {
	t.Helper()
	for _, p := range params {
		if p.Name == name {
			if p.Expression != expr {
				t.Errorf("param %q = %q, want %q", name, p.Expression, expr)
			}
			return
		}
	}
	t.Errorf("param %q not published; have %+v", name, params)
}

// TestThreadDesignationPrefersLabel checks an explicit `thread` label (an ANSI inch designation)
// wins over the metric d/P fallback, so one fastener generator serves both metric and inch families.
func TestThreadDesignationPrefersLabel(t *testing.T) {
	metric := ResolvedMember{Member: catalog.Member{Values: map[string]float64{"d": 8, "P": 1.25}}}
	if got := threadDesignation(metric); got != "M8x1.25" {
		t.Errorf("metric threadDesignation = %q, want M8x1.25", got)
	}
	inch := ResolvedMember{Member: catalog.Member{Labels: map[string]string{"thread": "1/4-20"}}}
	if got := threadDesignation(inch); got != "1/4-20" {
		t.Errorf("inch threadDesignation = %q, want 1/4-20", got)
	}
}
