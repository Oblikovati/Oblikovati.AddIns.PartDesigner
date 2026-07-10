// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"testing"

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
