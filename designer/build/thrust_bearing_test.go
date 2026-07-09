// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"math"
	"testing"

	"oblikovati.org/api/wire"
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

// TestThrustBearingBuildsGroovedWashers is the acceptance check: the tabulated dimensions and ball
// count are published, the pitch/ball diameters, groove radius and land offset derived, the balls
// patterned FIRST (one ball revolve about the pitch radius + a pattern by ball_count), then the two
// grooved washers each revolved about the Z axis.
func TestThrustBearingBuildsGroovedWashers(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (ThrustBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), thrustMember("51105", 25, 42, 11, 15)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "bore", "25 mm")
	assertParam(t, h.added, "height", "11 mm")
	assertParam(t, h.added, "ball_count", "15")
	assertParam(t, h.added, "ball_dia", "height * 0.45")
	// The ground groove: a conformity multiple of the ball diameter, with the land face a fraction of
	// the groove radius off the mid-plane so the groove is a channel the ball nests in.
	assertParam(t, h.added, "groove_radius", "ball_dia * 0.53")
	assertParam(t, h.added, "land_offset", "groove_radius * 0.7")

	if len(h.extrudes) != 0 {
		t.Errorf("extrudes = %d, want 0 (washers are revolved, not extruded)", len(h.extrudes))
	}
	// One ball revolve about X, plus two grooved-washer revolves about Z.
	if len(h.revolves) != 3 {
		t.Fatalf("revolves = %d, want 3 (one ball + two grooved washers)", len(h.revolves))
	}
	if h.revolves[0].AxisRef != "origin/axis/x" {
		t.Errorf("first revolve axis = %q, want origin/axis/x (the ball)", h.revolves[0].AxisRef)
	}
	for i := 1; i < 3; i++ {
		if h.revolves[i].AxisRef != "origin/axis/z" || h.revolves[i].Angle != "360 deg" {
			t.Errorf("washer revolve[%d] = %+v, want origin/axis/z / 360 deg", i, h.revolves[i])
		}
	}
	if len(h.patterns) != 1 || h.patterns[0].CountExpr != "ball_count" {
		t.Errorf("patterns = %+v, want one ball_count pattern", h.patterns)
	}
}

// TestThrustBearingUnderConstrainedFails ensures a non-zero DOF aborts before any solid is made.
func TestThrustBearingUnderConstrainedFails(t *testing.T) {
	h := &fakeHost{dof: 2}
	if err := (ThrustBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), thrustMember("51105", 25, 42, 11, 15)); err == nil {
		t.Fatal("Build accepted an under-constrained thrust bearing; want an error")
	}
}

// TestThrustBearingBuildErrorsPropagate injects a host failure at each wire method the build uses.
func TestThrustBearingBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodParametersAdd, wire.MethodSketchCreate,
		wire.MethodSketchAddEntity, wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd,
	}
	for _, m := range methods {
		h := &fakeHost{dof: 0, failMethod: m}
		if err := (ThrustBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), thrustMember("51105", 25, 42, 11, 15)); err == nil {
			t.Errorf("failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// TestThrustGrooveFitsWasher guards the grooved-washer geometry against a groove that would swallow a
// land or breach the back face. With ball_dia = 0.45·H, r_g = 0.53·ball_dia and the land at 0.7·r_g,
// the groove half-width is w = r_g·sqrt(1 − 0.7²); the inner land (pitch_r − w − bore/2) and outer
// land (OD/2 − pitch_r − w) must stay positive, the groove floor must clear the back face (r_g <
// H/2), and the balls must not overlap on the pitch circle (2π·pitch_r/Z > ball_dia). This is the
// static, per-member check the fakeHost cannot make (it does not evaluate expressions).
func TestThrustGrooveFitsWasher(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("catalog.Load() error = %v", err)
	}
	for _, fam := range cat.Families() {
		if fam.Generator != "thrust_bearing" {
			continue
		}
		cols := map[string]string{}
		for _, c := range fam.Columns {
			cols[c.Param] = c.Name
		}
		for _, m := range fam.Members {
			checkThrustMember(t, fam.ID, m, cols)
		}
	}
}

// checkThrustMember validates one thrust member's derived groove geometry stays non-degenerate.
func checkThrustMember(t *testing.T, famID string, m catalog.Member, cols map[string]string) {
	t.Helper()
	bore, outer, height := m.Values[cols["bore"]], m.Values[cols["outer_dia"]], m.Values[cols["height"]]
	balls := m.Values[cols["ball_count"]]
	pitchR := (bore + outer) / 4
	ballDia := 0.45 * height
	rg := 0.53 * ballDia
	w := rg * math.Sqrt(1-0.7*0.7) // groove half-width; land at 0.7·r_g off the mid-plane
	innerLand := (pitchR - w) - bore/2
	outerLand := outer/2 - (pitchR + w)
	spacing := 2 * math.Pi * pitchR / balls
	switch {
	case innerLand <= 0 || outerLand <= 0:
		t.Errorf("family %q member %q: groove swallows a land (inner=%.2f outer=%.2f)",
			famID, m.Key, innerLand, outerLand)
	case rg >= height/2:
		t.Errorf("family %q member %q: groove floor breaches the back face (r_g=%.2f ≥ H/2=%.2f)",
			famID, m.Key, rg, height/2)
	case spacing <= ballDia:
		t.Errorf("family %q member %q: balls overlap on the pitch circle (spacing=%.2f ≤ ball_dia=%.2f)",
			famID, m.Key, spacing, ballDia)
	}
}

// TestDefaultRegistryHasThrustBearing checks the generator is wired into the built-in set.
func TestDefaultRegistryHasThrustBearing(t *testing.T) {
	g, ok := DefaultRegistry().Get("thrust_bearing")
	if !ok || g.Kind() != "thrust_bearing" {
		t.Fatalf("DefaultRegistry thrust_bearing = (%v,%v), want the ThrustBearing generator", g, ok)
	}
}
