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
// the ball count are published, the pitch/ball/groove/shoulder diameters are derived, two grooved
// rings are revolved about the axis and one ball about the pitch-circle radius, then the ball is
// circular-patterned ball_count times. The 6205 has axial slack for the 2Z shields (#53 task 4), so
// they are revolved too — two more axis/360deg/new revolves after the rings.
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
	assertParam(t, h.added, "ball_dia", "(outer_dia - bore) * 0.28")
	// The ground race groove: its radius is a conformity multiple of the ball diameter, and the two
	// raceway shoulders flank it 2k·r_g off the pitch circle, so the ball nests in the groove.
	assertParam(t, h.added, "groove_radius", "ball_dia * 0.52")
	assertParam(t, h.added, "inner_shoulder_dia", "pitch_dia - 1.1 * groove_radius")
	assertParam(t, h.added, "outer_shoulder_dia", "pitch_dia + 1.1 * groove_radius")

	if len(h.revolves) != 5 {
		t.Fatalf("revolves = %d, want 5 (one ball + two rings + two shields)", len(h.revolves))
	}
	for i, want := range []struct{ axis, angle string }{
		{"origin/axis/x", "360 deg"}, // ball first, revolved about the pitch-circle radius
		{"origin/axis/z", "360 deg"}, // inner ring
		{"origin/axis/z", "360 deg"}, // outer ring
		{"origin/axis/z", "360 deg"}, // 2Z shield, +Z face
		{"origin/axis/z", "360 deg"}, // 2Z shield, −Z face
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

// TestBallBearingGrooveFitsRaceway guards the ground-race-groove geometry against a vanishing
// shoulder land: with groove_radius = 0.52·ball_dia and ball_dia = 0.28·gap, the groove's axial
// half-span z_s = groove_radius·sqrt(1−0.55²) ≈ 0.835·groove_radius must stay inside width/2, or the
// groove would swallow the raceway shoulder and the ring section would self-intersect on revolve.
// This is the static, per-member check the fakeHost cannot make (it does not evaluate expressions).
func TestBallBearingGrooveFitsRaceway(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatalf("catalog.Load() error = %v", err)
	}
	const zsFactor = 0.835 // sqrt(1 − 0.55²), the groove half-axial-span as a fraction of groove_radius
	for _, fam := range cat.Families() {
		if fam.Generator != "ball_bearing" {
			continue
		}
		cols := map[string]string{}
		for _, c := range fam.Columns {
			cols[c.Param] = c.Name
		}
		for _, m := range fam.Members {
			bore, outer, width := m.Values[cols["bore"]], m.Values[cols["outer_dia"]], m.Values[cols["width"]]
			grooveRadius := 0.52 * (0.28 * (outer - bore))
			if land := width/2 - zsFactor*grooveRadius; land <= 0 {
				t.Errorf("family %q member %q: groove swallows the raceway shoulder "+
					"(width/2=%.2f ≤ 0.835·groove_radius=%.2f, land=%.2f)",
					fam.ID, m.Key, width/2, zsFactor*grooveRadius, land)
			}
		}
	}
}

// TestBallShieldParams checks the 2Z shield band's derived parameters: the near face just outboard
// of the ball equator, the thickness capped by the axial slack, and the radial span a hair inside
// the two raceway shoulders.
func TestBallShieldParams(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (BallBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), bearingMember("6206", 30, 62, 16, 9)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "shield_near_z", "ball_dia / 2 + 0.2")
	assertParam(t, h.added, "shield_thick", "min(width * 0.12, width / 2 - ball_dia / 2 - 0.4)")
	assertParam(t, h.added, "shield_far_z", "shield_near_z + shield_thick")
	assertParam(t, h.added, "shield_id", "inner_shoulder_dia + 0.3")
	assertParam(t, h.added, "shield_od", "outer_shoulder_dia - 0.3")
}

// TestBallShieldsRevolvedBothFaces checks the two shields (both faces) are revolved after the two
// grooved rings: >=4 revolves, the last two z/360 deg/new.
func TestBallShieldsRevolvedBothFaces(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (BallBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), bearingMember("6206", 30, 62, 16, 9)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if len(h.revolves) < 4 { // ball + 2 rings + 2 shields (ball is 1 revolve, rings 2, shields 2 => 5)
		t.Fatalf("revolves = %d, want >=4 with two shields", len(h.revolves))
	}
	shields := h.revolves[len(h.revolves)-2:]
	for i, rv := range shields {
		if rv.AxisRef != "origin/axis/z" || rv.Angle != "360 deg" || rv.Operation != "new" {
			t.Errorf("shield revolve[%d] = %+v, want z/360 deg/new", i, rv)
		}
	}
}

// TestBallShieldsMirrorZSign is the regression guard for the −Z shield's mirroring: revolveOneShield
// steers GroundedShieldSection's rectangle seed onto the +Z or −Z face purely via the negZ sign flip
// (section_shield.go), and nothing else in the build distinguishes the two shields — same dimension
// expressions, same revolve axis/angle/op. A defect that dropped the sign (e.g. negZ ignored, or
// `sign := 1.0` unconditional) would still pass every other shield test, so this asserts the actual
// mechanism directly: the two shield rectangles' seed corners sit on opposite sides of Z=0.
func TestBallShieldsMirrorZSign(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (BallBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), bearingMember("6206", 30, 62, 16, 9)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if len(h.rectangleSeeds) != 2 {
		t.Fatalf("rectangleSeeds = %d, want 2 (one rectangle per shield)", len(h.rectangleSeeds))
	}
	faceZ := make([]float64, len(h.rectangleSeeds))
	for i, seed := range h.rectangleSeeds {
		if len(seed) != 2 || len(seed[0]) != 2 || len(seed[1]) != 2 {
			t.Fatalf("shield[%d] seed = %v, want two [x,y] corners", i, seed)
		}
		faceZ[i] = seed[0][1] // corner's Z (sketch-plane Y on XZ); opposite corner shares its sign
	}
	if faceZ[0] == 0 || faceZ[1] == 0 {
		t.Fatalf("shield seed Z = %v, want both nonzero", faceZ)
	}
	if (faceZ[0] > 0) == (faceZ[1] > 0) {
		t.Errorf("shield seed Z signs = %v, want opposite signs (one +Z face, one −Z face)", faceZ)
	}
}

// TestShieldsFit checks the axial-slack guard: every 60/62/63-series member (worst case 6200, slack
// 1.70mm) gets shields, but a synthetic fat-ball/thin-ring member (no room) falls back to none.
func TestShieldsFit(t *testing.T) {
	if !shieldsFit(bearingMember("6200", 10, 30, 9, 8)) { // worst slack 1.70mm
		t.Error("shieldsFit false for 6200; every 60/62/63 member should get shields")
	}
	if shieldsFit(bearingMember("x", 10, 40, 2, 8)) { // fat ball, thin ring → no room
		t.Error("shieldsFit true for a fat-ball/thin-ring member; want no-shield fallback")
	}
}

// TestDefaultRegistryHasBallBearing checks the generator is wired into the built-in set.
func TestDefaultRegistryHasBallBearing(t *testing.T) {
	g, ok := DefaultRegistry().Get("ball_bearing")
	if !ok || g.Kind() != "ball_bearing" {
		t.Fatalf("DefaultRegistry ball_bearing = (%v,%v), want the BallBearing generator", g, ok)
	}
}
