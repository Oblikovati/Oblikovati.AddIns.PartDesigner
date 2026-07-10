// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/part-designer/designer/catalog"
)

func TestRollerCageParams(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rollerMember("NU206", 30, 62, 16, 13)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	assertParam(t, h.added, "roller_subtend", "asin(roller_dia / pitch_dia)")
	assertParam(t, h.added, "cage_half_pitch", "180 deg / roller_count")
	assertParam(t, h.added, "bar_half_angle", "0.4 * (cage_half_pitch - roller_subtend)")
	assertParam(t, h.added, "bar_id", "pitch_dia - bar_radial_thick")
	assertParam(t, h.added, "bar_od", "pitch_dia + bar_radial_thick")
	assertParam(t, h.added, "bar_axial_len", "roller_length * 0.7")
}

// The cage bar is a two-sided revolve about Z; with a bar there are 2 pattern-source bodies.
func TestRollerCageBarRevolvedBeforePattern(t *testing.T) {
	h := &fakeHost{dof: 0}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rollerMember("NU206", 30, 62, 16, 13)); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if len(h.patterns) != 1 || h.patterns[0].CountExpr != "roller_count" {
		t.Fatalf("patterns = %+v, want one roller_count pattern over roller+bar", h.patterns)
	}
	// A bar_half_angle two-sided revolve about Z exists among the revolves (roller+bar+rings).
	var found bool
	for _, rv := range h.revolves {
		if rv.Angle == "bar_half_angle" || rv.Angle == "bar_half_angle * 2" {
			found = true
		}
	}
	if !found {
		t.Errorf("no cage-bar revolve found in %+v", h.revolves)
	}
}

// rollerCageBarsFit is named distinctly from tapered_cage.go's cageBarsFit (same package, unrelated
// cone-angle guard) — see roller_cage.go for the rationale.
func TestCageBarsFitFamily(t *testing.T) {
	// NU204 is the tightest (free_half_gap 1.96deg) but still > floor → gets bars.
	if !rollerCageBarsFit(rollerMember("x", 20, 47, 14, 12)) {
		t.Error("rollerCageBarsFit false for NU204; it should still get bars")
	}
	// A dense synthetic complement (many fat rollers) → no room → no bars.
	if rollerCageBarsFit(rollerMember("x", 30, 62, 16, 40)) {
		t.Error("rollerCageBarsFit true for a dense complement; want no-bars fallback")
	}
}

// TestRollerCageFallbackSkipsBar is the Build-level regression for rollerCageBarsFit's false
// branch: TestCageBarsFitFamily only checks the predicate. With a dense roller complement (the
// free half-gap under the floor), patternRollers must skip buildRollerCageBar entirely — no
// bar_half_angle revolve appears anywhere — while the pattern still arrays exactly the one roller
// feature (roller-only, nothing to co-pattern). This member's rollerChamferFits and flangesFit are
// both still true (verified by hand), so it isolates the cage fallback: it would fail if
// rollerCageBarsFit were removed or inverted, since a bar_half_angle revolve would then appear.
func TestRollerCageFallbackSkipsBar(t *testing.T) {
	h := &fakeHost{dof: 0}
	rm := rollerMember("x", 30, 62, 16, 40) // dense complement: free half-gap < floor
	if rollerCageBarsFit(rm) {
		t.Fatal("test fixture unexpectedly passes rollerCageBarsFit; no longer degenerate")
	}
	if err := (RollerBearing{}).Build(newBuilder(h, catalog.UnitsMillimetre), rm); err != nil {
		t.Fatalf("Build error = %v", err)
	}
	if len(h.patterns) != 1 {
		t.Fatalf("patterns = %d, want 1 (roller-only, no bar to co-pattern)", len(h.patterns))
	}
	for _, rv := range h.revolves {
		if rv.Angle == "bar_half_angle" {
			t.Errorf("found bar_half_angle revolve %+v; want no cage bar in the fallback path", rv)
		}
	}
}

// TestRollerCageBarErrorsPropagate walks buildRollerCageBar's own build beyond its
// AngledOrientedSketch call (its workPlanes.create failure is already reached by
// TestRollerBearingBuildErrorsPropagate's sweep): the ring section and its constraint check. The
// roller itself has already revolved by this point (buildRoller runs first in patternRollers),
// but the bar and the roller_count pattern have not, so revolves must be exactly 1 and the
// pattern empty in both cases.
func TestRollerCageBarErrorsPropagate(t *testing.T) {
	cases := []struct {
		name      string
		method    string
		failAfter int
	}{
		{"GroundedRingSection", wire.MethodSketchAddEntity, 3},
		{"AssertFullyConstrained", wire.MethodSketchConstraintStatus, 1},
	}
	for _, c := range cases {
		h, err := buildRollerWithFailure(t, c.method, c.failAfter)
		if err == nil {
			t.Errorf("%s: Build succeeded, want an error", c.name)
		}
		if len(h.revolves) != 1 || len(h.patterns) != 0 {
			t.Errorf("%s: revolves=%d patterns=%d, want 1/0 (roller done, bar+pattern not)",
				c.name, len(h.revolves), len(h.patterns))
		}
	}
}
