// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// hollowRectMember builds a synthetic resolved SHS/RHS member.
func hollowRectMember(designation string, width, height, thickness, length float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-hollowrect", Generator: "hollow_rect", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"designation"},
		Columns: []catalog.Column{
			{Name: "designation", Param: "designation", Type: catalog.ColumnText},
			{Name: "b", Param: "width", Type: catalog.ColumnLength},
			{Name: "h", Param: "height", Type: catalog.ColumnLength},
			{Name: "t", Param: "thickness", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{
			Key:    "designation=" + designation,
			Values: map[string]float64{"b": width, "h": height, "t": thickness, "l": length},
			Labels: map[string]string{"designation": designation},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// hollowRoundMember builds a synthetic resolved CHS member.
func hollowRoundMember(designation string, outerDia, thickness, length float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-hollowround", Generator: "hollow_round", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"designation"},
		Columns: []catalog.Column{
			{Name: "designation", Param: "designation", Type: catalog.ColumnText},
			{Name: "d", Param: "outer_dia", Type: catalog.ColumnLength},
			{Name: "t", Param: "thickness", Type: catalog.ColumnLength},
			{Name: "l", Param: "length", Type: catalog.ColumnLength},
		},
		Members: []catalog.Member{{
			Key:    "designation=" + designation,
			Values: map[string]float64{"d": outerDia, "t": thickness, "l": length},
			Labels: map[string]string{"designation": designation},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestHollowRectBuildsTube is the C3 SHS/RHS acceptance check: the section parameters are
// published, an outer rectangle extrudes as a new solid and a concentric inset inner rectangle
// cuts the bore — two length extrudes, new then cut.
func TestHollowRectBuildsTube(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (HollowRect{}).Build(newBuilder(h, catalog.UnitsMillimetre),
		hollowRectMember("RHS 120x60x5", 120, 60, 5, 6000)); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	assertParam(t, h.added, "width", "120 mm")
	assertParam(t, h.added, "height", "60 mm")
	assertParam(t, h.added, "thickness", "5 mm")
	assertParam(t, h.added, "length", "6000 mm")
	// The cold-formed corner radii are derived from the wall: outer ≈ 2·t, inner = outer − t = t.
	assertParam(t, h.added, "outer_radius", "2 * (thickness)")
	assertParam(t, h.added, "inner_radius", "thickness")

	if len(h.extrudes) != 2 {
		t.Fatalf("extrudes = %d, want 2 (outer new + inner cut)", len(h.extrudes))
	}
	if h.extrudes[0].Operation != "new" || h.extrudes[0].Distance != "length" {
		t.Errorf("outer extrude = %+v, want length/new", h.extrudes[0])
	}
	if h.extrudes[1].Operation != "cut" || h.extrudes[1].Distance != "length" {
		t.Errorf("inner extrude = %+v, want length/cut", h.extrudes[1])
	}
	// Each rounded rectangle centres its four edges by half-width/half-height offsets from the
	// grounded origin; the inner one is inset by the wall thickness on every side.
	for _, expr := range []string{
		"(width) / 2", "(height) / 2",
		"(width - 2 * (thickness)) / 2", "(height - 2 * (thickness)) / 2",
	} {
		if !hasDimension(h.dimensions, expr) {
			t.Errorf("edge-offset dimension %q missing; have %+v", expr, h.dimensions)
		}
	}
	// Each corner is a real quarter-round: the four arcs share one radius (EqualRadius ×3 per
	// rectangle) driven by a single radius dimension, for two rectangles.
	if got := countConstraintKind(h.constraints, "equalRadius"); got != 6 {
		t.Errorf("equalRadius constraints = %d, want 6 (3 per rounded rectangle)", got)
	}
	if !hasDimension(h.dimensions, "outer_radius") || !hasDimension(h.dimensions, "inner_radius") {
		t.Errorf("corner-radius dimensions missing; have %+v", h.dimensions)
	}
}

// countConstraintKind counts recorded geometric constraints of one kind.
func countConstraintKind(kinds []string, want string) int {
	n := 0
	for _, k := range kinds {
		if k == want {
			n++
		}
	}
	return n
}

// TestRoundedRectangleFullyConstrains checks GroundedRoundedRectangle emits a closed
// four-line/four-arc outline pinned to DOF 0: the loop is closed by eight coincidences, each arc's
// two radii are axis-aligned (four horizontal + four vertical), and the size is driven by one
// radius dimension plus four edge offsets — no free sketch-fillet arcs.
func TestRoundedRectangleFullyConstrains(t *testing.T) {
	h := &fakeHost{dof: 0}
	b := newBuilder(h, catalog.UnitsMillimetre)
	sk, err := b.Sketch("XY")
	if err != nil {
		t.Fatalf("Sketch error = %v", err)
	}
	if err := sk.GroundedRoundedRectangle("width", "height", "corner_r"); err != nil {
		t.Fatalf("GroundedRoundedRectangle error = %v", err)
	}
	// Closed loop: edge→arc→edge around the outline is eight coincident joins.
	if got := countConstraintKind(h.constraints, "coincident"); got != 8 {
		t.Errorf("coincident joins = %d, want 8 (closed four-edge/four-arc loop)", got)
	}
	// Each of the four arcs pins one horizontal and one vertical radius (centre→endpoint).
	if got := countConstraintKind(h.constraints, "horizontal"); got != 4 {
		t.Errorf("horizontal arc radii = %d, want 4", got)
	}
	if got := countConstraintKind(h.constraints, "vertical"); got != 4 {
		t.Errorf("vertical arc radii = %d, want 4", got)
	}
	if got := countConstraintKind(h.constraints, "equalRadius"); got != 3 {
		t.Errorf("equalRadius constraints = %d, want 3 (chain the four corners)", got)
	}
	if !hasDimension(h.dimensions, "corner_r") {
		t.Errorf("corner-radius dimension missing; have %+v", h.dimensions)
	}
	// One radius + four edge offsets = five dimensions total; no more (nothing over-dimensioned).
	if len(h.dimensions) != 5 {
		t.Errorf("dimensions = %d, want 5 (one radius + four edge offsets)", len(h.dimensions))
	}
}

// TestHollowRoundBuildsPipe is the C3 CHS acceptance check: an outer circle extrudes as a new
// solid and a concentric inner circle inset by the wall cuts the bore.
func TestHollowRoundBuildsPipe(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (HollowRound{}).Build(newBuilder(h, catalog.UnitsMillimetre),
		hollowRoundMember("CHS 88.9x5", 88.9, 5, 6000)); err != nil {
		t.Fatalf("Build error = %v", err)
	}

	assertParam(t, h.added, "outer_dia", "88.9 mm")
	assertParam(t, h.added, "thickness", "5 mm")

	if len(h.extrudes) != 2 || h.extrudes[0].Operation != "new" || h.extrudes[1].Operation != "cut" {
		t.Fatalf("extrudes = %+v, want outer new + inner cut", h.extrudes)
	}
	// The circle radii are half of the outer and the inset inner diameter.
	if h.circleRadius != "(outer_dia - 2 * (thickness))/2" {
		t.Errorf("last circle radius = %q, want the inset inner radius", h.circleRadius)
	}
}

// TestHollowUnderConstrainedFails ensures a non-zero DOF aborts before any cut.
func TestHollowUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 4}
	if err := (HollowRect{}).Build(newBuilder(h, catalog.UnitsMillimetre),
		hollowRectMember("SHS 100x100x5", 100, 100, 5, 6000)); err == nil {
		t.Fatal("Build accepted an under-constrained tube; want an error")
	}
	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 when the outer profile is under-constrained", len(h.extrudes))
	}
}

// TestHollowBuildErrorsPropagate injects a host failure at each wire method the builds use.
func TestHollowBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodSketchCreate, wire.MethodSketchAddEntity,
		wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	for _, m := range methods {
		hr := &fakeHost{dof: 0, failMethod: m}
		if err := (HollowRect{}).Build(newBuilder(hr, catalog.UnitsMillimetre),
			hollowRectMember("SHS 100x100x5", 100, 100, 5, 6000)); err == nil {
			t.Errorf("hollow_rect failMethod %q: Build succeeded, want an error", m)
		}
		hc := &fakeHost{dof: 0, failMethod: m}
		if err := (HollowRound{}).Build(newBuilder(hc, catalog.UnitsMillimetre),
			hollowRoundMember("CHS 88.9x5", 88.9, 5, 6000)); err == nil {
			t.Errorf("hollow_round failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// TestDefaultRegistryHasHollow checks both hollow generators are wired into the built-in set.
func TestDefaultRegistryHasHollow(t *testing.T) {
	for _, kind := range []string{"hollow_rect", "hollow_round"} {
		g, ok := DefaultRegistry().Get(kind)
		if !ok || g.Kind() != kind {
			t.Fatalf("DefaultRegistry %s = (%v,%v), want the hollow generator", kind, g, ok)
		}
	}
}
