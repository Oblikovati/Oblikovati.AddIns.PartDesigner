// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// rollerMember builds a synthetic resolved cylindrical-roller-bearing member: designation, bore d,
// outer diameter D, width B, roller count Z (the NU205: 25×52×15, 12 rollers).
func rollerMember(designation string, bore, outerDia, width, rollers float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-roller-bearing", Generator: "roller_bearing", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"designation"},
		Columns: []catalog.Column{
			{Name: "designation", Param: "designation", Type: catalog.ColumnText},
			{Name: "d", Param: "bore", Type: catalog.ColumnLength},
			{Name: "D", Param: "outer_dia", Type: catalog.ColumnLength},
			{Name: "B", Param: "width", Type: catalog.ColumnLength},
			{Name: "Z", Param: "roller_count", Type: catalog.ColumnCount},
		},
		Members: []catalog.Member{{
			Key:    "designation=NU205",
			Values: map[string]float64{"d": bore, "D": outerDia, "B": width, "Z": rollers},
			Labels: map[string]string{"designation": designation},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestRollerBearingBuildsRollersAndRings is the E2 roller acceptance check: the tabulated
// dimensions and roller count are published, the roller/race parameters derived, one cylindrical
// roller extruded symmetric about the mid-plane and patterned by roller_count, then the two rings
// revolved — with the rollers built BEFORE the rings so the pattern does not replicate the rings.
func TestRollerBearingBuildsRollersAndRings(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rollerMember("NU205", 25, 52, 15, 12)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "bore", "25 mm")
	assertParam(t, h.added, "roller_count", "12")
	assertParam(t, h.added, "roller_dia", "(outer_dia - bore) * 0.28")
	assertParam(t, h.added, "roller_length", "width * 0.8")
	// Each race clears the roller crest (pitch_dia ± roller_dia) by a fraction of the gap.
	assertParam(t, h.added, "outer_race_dia", "pitch_dia + roller_dia + (outer_dia - bore) * 0.012")
	assertParam(t, h.added, "inner_race_dia", "pitch_dia - roller_dia - (outer_dia - bore) * 0.012")

	if len(h.extrudes) != 1 {
		t.Fatalf("extrudes = %d, want 1 (the roller)", len(h.extrudes))
	}
	if h.extrudes[0].Distance != "roller_length" || h.extrudes[0].Operation != "new" || h.extrudes[0].Direction != "symmetric" {
		t.Errorf("roller extrude = %+v, want roller_length/new/symmetric", h.extrudes[0])
	}
	if len(h.revolves) != 2 {
		t.Fatalf("revolves = %d, want 2 (inner + outer ring)", len(h.revolves))
	}
	for i, rv := range h.revolves {
		if rv.AxisRef != "origin/axis/z" || rv.Angle != "360 deg" || rv.Operation != "new" {
			t.Errorf("ring revolve[%d] = %+v, want z / 360 deg / new", i, rv)
		}
	}
}

// TestRollerBearingPatternsByCount checks the roller complement is a circular pattern driven by the
// roller_count parameter about the Z axis, referencing the single roller feature.
func TestRollerBearingPatternsByCount(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rollerMember("NU205", 25, 52, 15, 12)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if len(h.patterns) != 1 {
		t.Fatalf("patterns = %d, want 1 (the roller complement)", len(h.patterns))
	}
	if h.patterns[0].CountExpr != "roller_count" {
		t.Errorf("pattern count = %q, want the roller_count parameter", h.patterns[0].CountExpr)
	}
	if len(h.patterns[0].SourceFeatures) != 1 || h.patterns[0].SourceFeatures[0] == "" {
		t.Errorf("pattern source = %v, want the single roller feature", h.patterns[0].SourceFeatures)
	}
}

// TestRollerBearingUnderConstrainedFails ensures a non-zero DOF aborts before any solid is made.
func TestRollerBearingUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 2}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rollerMember("NU205", 25, 52, 15, 12)); err == nil {
		t.Fatal("Build accepted an under-constrained roller bearing; want an error")
	}
	if len(h.extrudes) != 0 || len(h.revolves) != 0 || len(h.patterns) != 0 {
		t.Errorf("made geometry despite bad DOF: extrudes=%d revolves=%d patterns=%d",
			len(h.extrudes), len(h.revolves), len(h.patterns))
	}
}

// TestDefaultRegistryHasRollerBearing checks the generator is wired into the built-in set.
func TestDefaultRegistryHasRollerBearing(t *testing.T) {
	g, ok := DefaultRegistry().Get("roller_bearing")
	if !ok || g.Kind() != "roller_bearing" {
		t.Fatalf("DefaultRegistry roller_bearing = (%v,%v), want the RollerBearing generator", g, ok)
	}
}

// deriveFlangeParams publishes the flange axial band; flangesFit gates on positive land/overlap/band.
func TestRollerFlangeParams(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rollerMember("NU206", 30, 62, 16, 13)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "flange_axial_clr", "max(0.1, roller_length * 0.02)")
	assertParam(t, h.added, "flange_inner_z", "roller_length / 2 + flange_axial_clr")
	assertParam(t, h.added, "flange_bore_dia", "pitch_dia")
}

func TestRollerFlangedOuterRingRevolved(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rollerMember("NU206", 30, 62, 16, 13)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	// inner ring + flanged outer ring = 2 revolves about Z, all 360deg/new.
	if len(h.revolves) < 2 {
		t.Fatalf("revolves = %d, want >=2 (inner + flanged outer)", len(h.revolves))
	}
	last := h.revolves[len(h.revolves)-1]
	if last.AxisRef != "origin/axis/z" || last.Angle != "360 deg" || last.Operation != "new" {
		t.Errorf("outer ring revolve = %+v, want z/360 deg/new", last)
	}
}

func TestFlangesFitAcrossFamily(t *testing.T) {
	members := [][4]float64{{15, 35, 11, 11}, {30, 62, 16, 13}, {50, 90, 20, 15}} // NU202, NU206, NU210
	for _, m := range members {
		if !flangesFit(rollerMember("x", m[0], m[1], m[2], m[3])) {
			t.Errorf("flangesFit false for d=%v D=%v B=%v; every NU2xx must get flanges", m[0], m[1], m[2])
		}
	}
	// Degenerate: a roller nearly as long as the ring leaves no overhang band → no flanges.
	if flangesFit(rollerMember("x", 30, 62, 1.0, 13)) {
		t.Error("flangesFit true for a member with no axial overhang; want plain-ring fallback")
	}
}
