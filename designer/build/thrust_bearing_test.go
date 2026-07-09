// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/part-designer/designer/catalog"
)

// thrustMember builds a synthetic resolved thrust-ball-bearing member: designation, bore d, outer
// diameter D, height H, ball count Z (the 51105: 25×42×11, 15 balls).
func thrustMember(designation string, bore, outerDia, height, balls float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-thrust-bearing", Generator: "thrust_bearing", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"designation"},
		Columns: []catalog.Column{
			{Name: "designation", Param: "designation", Type: catalog.ColumnText},
			{Name: "d", Param: "bore", Type: catalog.ColumnLength},
			{Name: "D", Param: "outer_dia", Type: catalog.ColumnLength},
			{Name: "H", Param: "height", Type: catalog.ColumnLength},
			{Name: "Z", Param: "ball_count", Type: catalog.ColumnCount},
		},
		Members: []catalog.Member{{
			Key:    "designation=51105",
			Values: map[string]float64{"d": bore, "D": outerDia, "H": height, "Z": balls},
			Labels: map[string]string{"designation": designation},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestThrustBearingBuildsWashersAndBalls is the acceptance check: the tabulated dimensions and ball
// count are published, the pitch/ball diameters and washer thickness derived, the balls patterned
// FIRST (one ball revolve about the pitch radius + a pattern by ball_count), then the two washers
// each built as an outer-diameter new solid plus a bore cut on an offset plane.
func TestThrustBearingBuildsWashersAndBalls(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (ThrustBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), thrustMember("51105", 25, 42, 11, 15)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "bore", "25 mm")
	assertParam(t, h.added, "height", "11 mm")
	assertParam(t, h.added, "ball_count", "15")
	assertParam(t, h.added, "ball_dia", "height * 0.4")
	assertParam(t, h.added, "washer_thickness", "height * 0.28")

	if len(h.revolves) != 1 || h.revolves[0].AxisRef != "origin/axis/x" {
		t.Errorf("revolves = %+v, want one ball revolve about x", h.revolves)
	}
	if len(h.patterns) != 1 || h.patterns[0].CountExpr != "ball_count" {
		t.Errorf("patterns = %+v, want one ball_count pattern", h.patterns)
	}
	// Two washers, each an outer-dia new + bore cut = 4 extrudes, all to washer_thickness.
	if len(h.extrudes) != 4 {
		t.Fatalf("extrudes = %d, want 4 (2 washers × outer-new + bore-cut)", len(h.extrudes))
	}
	newOps, cutOps := 0, 0
	for _, ex := range h.extrudes {
		if ex.Distance != "washer_thickness" {
			t.Errorf("washer extrude distance = %q, want washer_thickness", ex.Distance)
		}
		switch ex.Operation {
		case "new":
			newOps++
		case "cut":
			cutOps++
		}
	}
	if newOps != 2 || cutOps != 2 {
		t.Errorf("washer ops = %d new / %d cut, want 2 each", newOps, cutOps)
	}
	if len(h.workPlanes) != 4 {
		t.Errorf("work planes = %d, want 4 (one per washer sketch)", len(h.workPlanes))
	}
}

// TestThrustBearingUnderConstrainedFails ensures a non-zero DOF aborts before any solid is made.
func TestThrustBearingUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 2}
	if err := (ThrustBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), thrustMember("51105", 25, 42, 11, 15)); err == nil {
		t.Fatal("Build accepted an under-constrained thrust bearing; want an error")
	}
}

// TestDefaultRegistryHasThrustBearing checks the generator is wired into the built-in set.
func TestDefaultRegistryHasThrustBearing(t *testing.T) {
	g, ok := DefaultRegistry().Get("thrust_bearing")
	if !ok || g.Kind() != "thrust_bearing" {
		t.Fatalf("DefaultRegistry thrust_bearing = (%v,%v), want the ThrustBearing generator", g, ok)
	}
}
