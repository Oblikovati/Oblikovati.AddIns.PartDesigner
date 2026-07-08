// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

// bearingMember builds a synthetic resolved deep-groove ball-bearing member: designation, bore d,
// outer diameter D, width B, ball count Z (the 6205: 25×52×15, 9 balls).
func bearingMember(designation string, bore, outerDia, width, balls float64) ResolvedMember {
	fam := &catalog.Family{
		ID: "t-ball-bearing", Generator: "ball_bearing", Units: catalog.UnitsMillimetre,
		KeyColumns: []string{"designation"},
		Columns: []catalog.Column{
			{Name: "designation", Param: "designation", Type: catalog.ColumnText},
			{Name: "d", Param: "bore", Type: catalog.ColumnLength},
			{Name: "D", Param: "outer_dia", Type: catalog.ColumnLength},
			{Name: "B", Param: "width", Type: catalog.ColumnLength},
			{Name: "Z", Param: "ball_count", Type: catalog.ColumnCount},
		},
		Members: []catalog.Member{{
			Key:    "designation=6205",
			Values: map[string]float64{"d": bore, "D": outerDia, "B": width, "Z": balls},
			Labels: map[string]string{"designation": designation},
		}},
	}
	return ResolvedMember{Family: fam, Member: fam.Members[0]}
}

// TestBallBearingBuildsRingsAndBalls is the E1 acceptance check: the three tabulated dimensions and
// the ball count are published, the pitch/ball/race diameters are derived, two rings are revolved
// about the axis and one ball about the pitch-circle radius, then the ball is circular-patterned
// ball_count times.
func TestBallBearingBuildsRingsAndBalls(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (BallBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), bearingMember("6205", 25, 52, 15, 9)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "bore", "25 mm")
	assertParam(t, h.added, "outer_dia", "52 mm")
	assertParam(t, h.added, "width", "15 mm")
	assertParam(t, h.added, "ball_count", "9")
	assertParam(t, h.added, "pitch_dia", "(bore + outer_dia) / 2")
	assertParam(t, h.added, "ball_dia", "(outer_dia - bore) * 0.3")

	if len(h.revolves) != 3 {
		t.Fatalf("revolves = %d, want 3 (one ball + two rings)", len(h.revolves))
	}
	for i, want := range []struct{ axis, angle string }{
		{"origin/axis/x", "360 deg"}, // ball first, revolved about the pitch-circle radius
		{"origin/axis/z", "360 deg"}, // inner ring
		{"origin/axis/z", "360 deg"}, // outer ring
	} {
		if h.revolves[i].AxisRef != want.axis || h.revolves[i].Angle != want.angle || h.revolves[i].Operation != "new" {
			t.Errorf("revolve[%d] = %+v, want %s / %s / new", i, h.revolves[i], want.axis, want.angle)
		}
	}
}

// TestBallBearingPatternsByCount checks the ball complement is a circular pattern driven by the
// ball_count parameter (never a literal count) about the world Z axis, referencing the ball feature.
func TestBallBearingPatternsByCount(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (BallBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), bearingMember("6205", 25, 52, 15, 9)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if len(h.patterns) != 1 {
		t.Fatalf("patterns = %d, want 1 (the ball complement)", len(h.patterns))
	}
	p := h.patterns[0]
	if p.CountExpr != "ball_count" {
		t.Errorf("pattern count = %q, want the ball_count parameter", p.CountExpr)
	}
	if len(p.SourceFeatures) != 1 || p.SourceFeatures[0] == "" {
		t.Errorf("pattern source = %v, want the single ball feature", p.SourceFeatures)
	}
	if len(p.AxisDir) != 3 || p.AxisDir[2] != 1 {
		t.Errorf("pattern axis = %v, want the world Z axis", p.AxisDir)
	}
}

// TestBallBearingUnderConstrainedFails ensures a non-zero DOF aborts before any solid is made.
func TestBallBearingUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 2}
	if err := (BallBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), bearingMember("6205", 25, 52, 15, 9)); err == nil {
		t.Fatal("Build accepted an under-constrained bearing; want an error")
	}
	if len(h.revolves) != 0 || len(h.patterns) != 0 {
		t.Errorf("made geometry despite bad DOF: revolves=%d patterns=%d", len(h.revolves), len(h.patterns))
	}
}

// TestBallBearingBuildErrorsPropagate injects a host failure at each wire method the build uses.
func TestBallBearingBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodParametersAdd, wire.MethodSketchCreate,
		wire.MethodSketchAddEntity, wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	for _, m := range methods {
		h := &fakeHost{dof: 0, failMethod: m}
		if err := (BallBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), bearingMember("6205", 25, 52, 15, 9)); err == nil {
			t.Errorf("failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// TestDefaultRegistryHasBallBearing checks the generator is wired into the built-in set.
func TestDefaultRegistryHasBallBearing(t *testing.T) {
	g, ok := DefaultRegistry().Get("ball_bearing")
	if !ok || g.Kind() != "ball_bearing" {
		t.Fatalf("DefaultRegistry ball_bearing = (%v,%v), want the BallBearing generator", g, ok)
	}
}
