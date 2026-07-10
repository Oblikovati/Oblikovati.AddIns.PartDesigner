// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/wire"
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
// roller revolved about its own centerline (chamfered ends) and patterned by roller_count, then the
// two rings revolved — with the rollers built BEFORE the rings so the pattern does not replicate
// the rings.
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

	// The chamfered roller is now a REVOLVE about its own centerline, not an extrude: the roller
	// complement is revolves[0], followed by the inner + outer ring revolves about Z.
	if len(h.revolves) < 3 {
		t.Fatalf("revolves = %d, want >=3 (roller-about-centerline + inner + outer ring)", len(h.revolves))
	}
	for i, rv := range h.revolves[len(h.revolves)-2:] {
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
	// The clearance floor carries "mm" units: a bare unitless "0.1" cannot combine with the length
	// roller_length in max(), which collapsed flange_inner_z to 0 and filled the ⊐ channel (#53).
	assertParam(t, h.added, "flange_axial_clr", "max(0.1 mm, roller_length * 0.02)")
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

// The chamfered roller is revolved about its own centerline; assert the chamfer param + that the
// roller feature is a 360deg/new revolve (about the centerline, not the Z axis).
func TestRollerChamferParams(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rollerMember("NU206", 30, 62, 16, 13)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "roller_chamfer", "roller_dia * 0.1")
}

func TestRollerChamferFits(t *testing.T) {
	if !rollerChamferFits(rollerMember("x", 15, 35, 11, 11)) { // NU202, min leg 0.56mm
		t.Error("rollerChamferFits false for NU202; every member should chamfer")
	}
	if rollerChamferFits(rollerMember("x", 100, 100.2, 11, 11)) { // ~0 gap → sub-floor leg
		t.Error("rollerChamferFits true for a sub-floor chamfer; want plain-roller fallback")
	}
}

// TestRollerChamferFallbackExtrudesPlainRoller is the Build-level regression for rollerChamferFits'
// false branch: TestRollerChamferFits only checks the predicate, not that Build actually takes the
// fallback. With a sub-floor chamfer leg, buildRoller must call buildPlainRoller — an EXTRUDE of a
// plain cylinder — instead of RevolveAboutCenterline-ing a chamfered meridian. This member's
// flangesFit and rollerCageBarsFit are both still true (verified by hand), so it isolates the
// chamfer fallback: it would fail if rollerChamferFits were removed or inverted, since the roller
// would then be revolved (not extruded) and a centerline revolve would appear.
func TestRollerChamferFallbackExtrudesPlainRoller(t *testing.T) {
	h := &fakeHost{dof: 0}
	rm := rollerMember("x", 30, 34, 16, 13) // D-d=4mm -> roller_dia 1.12mm -> leg 0.112mm < 0.15mm floor
	if rollerChamferFits(rm) {
		t.Fatal("test fixture unexpectedly passes rollerChamferFits; no longer degenerate")
	}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if len(h.extrudes) != 1 {
		t.Fatalf("extrudes = %d, want 1 (the plain roller)", len(h.extrudes))
	}
	if h.extrudes[0].Distance != "roller_length" || h.extrudes[0].Direction != "symmetric" {
		t.Errorf("plain roller extrude = %+v, want distance roller_length / direction symmetric", h.extrudes[0])
	}
	for _, rv := range h.revolves {
		if rv.AboutCenterline {
			t.Errorf("found centerline revolve %+v; want no chamfered-roller revolve in the fallback path", rv)
		}
	}
}

// TestFlangeFallbackUsesPlainOuterRing is the Build-level regression for flangesFit's false branch:
// TestFlangesFitAcrossFamily only checks the predicate. With no axial overhang band,
// revolveFlangedOuterRing must fall back to the plain revolveRing (GroundedRingSection, a
// "rectangle" sketch entity) instead of the flanged ⊐ channel (GroundedFlangedRingSection, a
// "polyline" via closedPolyline). revolveFlangedOuterRing is unconditionally the LAST section Build
// adds (nothing runs after it), and every section's constrain step ends by grounding a "point"
// origin (groundedOrigin), so the last non-point recorded kind pins down which outline ran;
// counting polylines confirms the only one came from the (unrelated, still-fitting) chamfered
// roller, not a second one from a wrongly-taken flanged path.
func TestFlangeFallbackUsesPlainOuterRing(t *testing.T) {
	h := &fakeHost{dof: 0}
	rm := rollerMember("x", 30, 62, 1.0, 13) // width=1.0mm leaves no axial overhang band
	if flangesFit(rm) {
		t.Fatal("test fixture unexpectedly passes flangesFit; no longer degenerate")
	}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	var last string
	var polylines int
	for _, k := range h.entityKinds {
		if k != "point" {
			last = k
		}
		if k == "polyline" {
			polylines++
		}
	}
	if last != "rectangle" {
		t.Errorf("outer-ring entity kind = %q, want rectangle (plain fallback, not the flanged polyline)", last)
	}
	if polylines != 1 {
		t.Errorf("polyline entities = %d, want 1 (only the chamfered-roller section)", polylines)
	}
}

// TestPlainRollerErrorsPropagate targets buildPlainRoller's own three guards — the fallback
// buildRoller takes when rollerChamferFits is false (roller_bearing.go). Uses the same
// degenerate-chamfer fixture as TestRollerChamferFallbackExtrudesPlainRoller (D-d=4mm ⇒ leg below
// the visibility floor), so buildPlainRoller — not the chamfered-roller section — is the first
// geometry Build attempts; failAfter=0 on each method reaches its own first call in the whole
// Build, since nothing runs before the roller.
func TestPlainRollerErrorsPropagate(t *testing.T) {
	rm := rollerMember("x", 30, 34, 16, 13) // D-d=4mm -> roller_dia 1.12mm -> leg 0.112mm < 0.15mm floor
	if rollerChamferFits(rm) {
		t.Fatal("test fixture unexpectedly passes rollerChamferFits; no longer degenerate")
	}
	cases := []struct {
		name   string
		method string
	}{
		{"Sketch", wire.MethodSketchCreate},
		{"GroundedOffsetCircle", wire.MethodSketchAddEntity},
		{"AssertFullyConstrained", wire.MethodSketchConstraintStatus},
	}
	for _, c := range cases {
		h := &fakeHost{dof: 0, failMethod: c.method}
		err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm)
		if err == nil {
			t.Errorf("%s: Build succeeded, want an error", c.name)
		}
		if len(h.extrudes) != 0 || len(h.revolves) != 0 || len(h.patterns) != 0 {
			t.Errorf("%s: made geometry despite a plain-roller failure: extrudes=%d revolves=%d patterns=%d",
				c.name, len(h.extrudes), len(h.revolves), len(h.patterns))
		}
	}
}

// buildRollerWithFailure runs RollerBearing.Build for a member with flanges, a chamfered roller
// and a cage bar all fitting (NU206), with method returning an error starting on its
// (failAfter+1)th call — the first failAfter matching calls succeed normally. That is how the
// tests below reach a LATER wire call inside a multi-call derivation or section instead of always
// tripping the very first one (fakeHost's plain failMethod, used with the default failAfter of 0,
// already covers every FIRST occurrence — see TestRollerBearingBuildErrorsPropagate).
func buildRollerWithFailure(t *testing.T, method string, failAfter int) (*fakeHost, error) {
	t.Helper()
	h := &fakeHost{dof: 0, failMethod: method, failAfter: failAfter}
	rm := rollerMember("NU206", 30, 62, 16, 13)
	err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm)
	return h, err
}

// TestRollerBearingBuildErrorsPropagate injects a host failure at each wire method the #53
// roller-bearing build uses, including the two #53 introduced beyond the pre-existing generators:
// workPlanes.create (the cage bar's AngledOrientedSketch) and document.{get,set}SketchSettings
// (the chamfered-roller and flanged-ring sections' disableSketchInference). Mirrors
// TestBallBearingBuildErrorsPropagate's sweep for the sibling generator.
func TestRollerBearingBuildErrorsPropagate(t *testing.T) {
	methods := []string{
		wire.MethodParametersList, wire.MethodParametersAdd, wire.MethodSketchCreate,
		wire.MethodSketchAddEntity, wire.MethodSketchAddConstraint, wire.MethodSketchAddDimension,
		wire.MethodSketchConstraintStatus, wire.MethodFeaturesAdd, wire.MethodWorkPlanesCreate,
		wire.MethodDocumentGetSketchSettings, wire.MethodDocumentSetSketchSettings,
	}
	for _, m := range methods {
		if _, err := buildRollerWithFailure(t, m, 0); err == nil {
			t.Errorf("failMethod %q: Build succeeded, want an error", m)
		}
	}
}

// TestRollerDeriveParamsErrorsPropagate walks deriveRollerParams' own derivation chain — pitch
// diameter, roller diameter/length, the clearing races (deriveRacesClearing), the flange band
// (deriveFlangeParams), the chamfer leg (deriveRollerChamferParams) and the cage bar
// (deriveRollerCageParams) — by letting parameters.list succeed failAfter times (every earlier
// derive step) before failing the next one, reaching each step's OWN "if err != nil" guard one at
// a time instead of always tripping the first call (inside PublishParams, already covered by
// TestRollerBearingBuildErrorsPropagate). No geometry is built until every derive succeeds, so
// every case must leave the bearing with zero revolves and no roller pattern.
func TestRollerDeriveParamsErrorsPropagate(t *testing.T) {
	cases := []struct {
		name      string
		failAfter int
	}{
		{"pitch_dia (derivePitchDia)", 1},
		{"roller_dia", 2},
		{"roller_length", 3},
		{"outer_race_dia (deriveRacesClearing)", 4},
		{"flange_axial_clr (deriveFlangeParams)", 6},
		{"flange_inner_z (deriveFlangeParams)", 7},
		{"roller_chamfer (deriveRollerChamferParams)", 9},
		{"roller_subtend (deriveRollerCageParams)", 10},
	}
	for _, c := range cases {
		h, err := buildRollerWithFailure(t, wire.MethodParametersList, c.failAfter)
		if err == nil {
			t.Errorf("%s: Build succeeded, want an error", c.name)
		}
		if len(h.revolves) != 0 || len(h.patterns) != 0 {
			t.Errorf("%s: made geometry despite a derive failure: revolves=%d patterns=%d",
				c.name, len(h.revolves), len(h.patterns))
		}
	}
}

// TestRollerBearingRingStageErrorsPropagate targets Build's own "if err := b.revolveRing(...)"
// guard (line 47-49) and revolveFlangedOuterRing's three guards (the flanged
// outer_dia/outer_race_dia ring, #53). The revolveRing case fails at the plain bore ring's very
// first call (its Sketch) — the minimal way to reach Build's guard without exercising
// revolveRing's own internals, a pre-existing helper #53 did not touch. Each case lets the
// earlier sketch.create/constraintStatus/document.getSketchSettings calls through so the failure
// lands on the named later call, and asserts the ring that call belongs to never finished
// revolving (revolves stops short of the full roller+bar+inner+outer complement of 4).
func TestRollerBearingRingStageErrorsPropagate(t *testing.T) {
	cases := []struct {
		name         string
		method       string
		failAfter    int
		wantRevolves int // revolves that must have completed before this failure
	}{
		{"Build: revolveRing(bore) propagation", wire.MethodSketchCreate, 2, 2},
		{"revolveFlangedOuterRing: Sketch", wire.MethodSketchCreate, 3, 3},
		{"revolveFlangedOuterRing: GroundedFlangedRingSection", wire.MethodDocumentGetSketchSettings, 1, 3},
		{"revolveFlangedOuterRing: AssertFullyConstrained", wire.MethodSketchConstraintStatus, 3, 3},
	}
	for _, c := range cases {
		h, err := buildRollerWithFailure(t, c.method, c.failAfter)
		if err == nil {
			t.Errorf("%s: Build succeeded, want an error", c.name)
		}
		if len(h.revolves) != c.wantRevolves {
			t.Errorf("%s: revolves = %d, want %d (the ring under test must not have completed)",
				c.name, len(h.revolves), c.wantRevolves)
		}
	}
}

// TestFlangedRingSectionErrorsPropagate walks GroundedFlangedRingSection's own build: the closed
// polyline, the grounded origin, orientFlangedRing's axis alignment and sizeFlangedRing's two
// dimension passes (the three radii, then the two z-levels) — reached by letting every earlier
// sketch.addEntity/addConstraint/addDimension call (the roller, the cage bar, the bore ring, and
// the flanged ring's own disableSketchInference) through first. The flanged ring's Revolve never
// runs in any case, so revolves must stay at 3 (roller + cage bar + bore ring).
func TestFlangedRingSectionErrorsPropagate(t *testing.T) {
	cases := []struct {
		name      string
		method    string
		failAfter int
	}{
		{"closedPolyline", wire.MethodSketchAddEntity, 7},
		{"groundedOrigin", wire.MethodSketchAddEntity, 8},
		{"orientFlangedRing", wire.MethodSketchAddConstraint, 18},
		{"sizeFlangedRing radii", wire.MethodSketchAddDimension, 16},
		{"sizeFlangedRing z-levels", wire.MethodSketchAddDimension, 19},
	}
	for _, c := range cases {
		h, err := buildRollerWithFailure(t, c.method, c.failAfter)
		if err == nil {
			t.Errorf("%s: Build succeeded, want an error", c.name)
		}
		if len(h.revolves) != 3 {
			t.Errorf("%s: revolves = %d, want 3 (the flanged ring must not have completed)", c.name, len(h.revolves))
		}
	}
}

// TestRollerChamferedSectionErrorsPropagate walks GroundedChamferedRollerSection's own build
// beyond its first (disableSketchInference/groundedOrigin) calls — already reached by
// TestRollerBearingBuildErrorsPropagate's sweep — into the closed polyline, the centerline weld
// (addRollerCenterline) and the axis anchor (anchorRollerRodAxis). No roller has revolved yet at
// any of these points, so revolves and the pattern must both stay empty.
func TestRollerChamferedSectionErrorsPropagate(t *testing.T) {
	cases := []struct {
		name      string
		method    string
		failAfter int
	}{
		{"closedPolyline", wire.MethodSketchAddEntity, 1},
		{"addRollerCenterline: AddCenterline", wire.MethodSketchAddEntity, 2},
		{"addRollerCenterline: weld start", wire.MethodSketchAddConstraint, 1},
		{"addRollerCenterline: weld end", wire.MethodSketchAddConstraint, 2},
		{"anchorRollerRodAxis: alignLevels", wire.MethodSketchAddConstraint, 3},
		{"anchorRollerRodAxis: centre roller axially", wire.MethodSketchAddDimension, 1},
	}
	for _, c := range cases {
		h, err := buildRollerWithFailure(t, c.method, c.failAfter)
		if err == nil {
			t.Errorf("%s: Build succeeded, want an error", c.name)
		}
		if len(h.revolves) != 0 || len(h.patterns) != 0 {
			t.Errorf("%s: made geometry despite a roller-section failure: revolves=%d patterns=%d",
				c.name, len(h.revolves), len(h.patterns))
		}
	}
}
