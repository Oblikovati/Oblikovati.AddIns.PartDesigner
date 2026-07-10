// SPDX-License-Identifier: GPL-2.0-only

package build

import "math"

// Cylindrical-roller CAGE proportions (#53 Task 3). A continuous revolved band at the pitch
// diameter would PIERCE the rollers (they sit ON the pitch circle, axes parallel to Z, unlike the
// tapered roller which only touches the pitch cone along a line). So the representational cage is
// one thin BRIDGE BAR per inter-roller gap, at the pitch diameter, revolved a small ±angle about Z
// on a plane at the half-pitch azimuth (π/roller_count) so it sits centred between two rollers. The
// bar is built as a second body BEFORE the roller circular pattern, so the single existing pattern
// (which copies every current body) arrays the roller AND the bar together — the bar, offset half a
// pitch from the roller, lands in every gap. Unlike the tapered cage there is no cone and no
// small-end rim: the roller subtends a CONSTANT half-angle about the axis, φ = asin(roller_dia /
// pitch_dia) (no contact-angle term — the roller axis is parallel to Z, not tilted), and the
// roller-free end band is already claimed by the outer-ring guide flanges (Task 1), so bars-only.
// Constants below are prefixed rollerCage (not cage) because tapered_cage.go declares its own
// cageBar*/cageBarGapFloor constants in this same package for the unrelated tapered/cone cage —
// same short names would redeclare.
const (
	// rollerCageBarFill sets the bar's angular half-width as this fraction of the free half-gap
	// (π/roller_count − roller_subtend), leaving equal clearance to both neighbouring rollers.
	rollerCageBarFill = "0.4"
	// rollerCageBarThickFraction / rollerCageBarAxialFraction size the bar: its radial thickness as
	// a fraction of the roller diameter, and its axial length as a fraction of the roller length
	// (kept short of the guide flanges).
	rollerCageBarThickFraction = "0.25"
	rollerCageBarAxialFraction = "0.7"
	// rollerCageBarGapFloor is the free half-gap (radians) below which the inter-roller gap is too
	// tight to seat a visible bar, so no bar is built at all (a defensive guard for dense complements).
	rollerCageBarGapFloor = 0.020 // radians
)

// rollerCageDerivations lists the cage's derived parameters (appended after the roller-chamfer
// derivations, so they can reference pitch_dia / roller_dia / roller_count / roller_length). Every
// distance is a parameter expression, so the cage re-drives with bore/outer_dia/width/roller_count.
// Named distinctly from tapered_cage.go's cageDerivations (same package, unrelated cone-angle math).
func rollerCageDerivations() []struct{ name, expr string } {
	return []struct{ name, expr string }{
		// Roller half-angle subtense about the axis: φ = asin(roller_dia / pitch_dia).
		{"roller_subtend", "asin(roller_dia / pitch_dia)"},
		// Half-pitch azimuth (the bar's work-plane angle) and the bar's angular half-width.
		{"cage_half_pitch", "180 deg / roller_count"},
		{"bar_half_angle", rollerCageBarFill + " * (cage_half_pitch - roller_subtend)"},
		// Bar meridian: a thin rectangle centred on the mid-plane at the pitch diameter.
		{"bar_radial_thick", "roller_dia * " + rollerCageBarThickFraction},
		{"bar_id", "pitch_dia - bar_radial_thick"},
		{"bar_od", "pitch_dia + bar_radial_thick"},
		{"bar_axial_len", "roller_length * " + rollerCageBarAxialFraction},
	}
}

// deriveRollerCageParams publishes the cylindrical-roller cage's derived parameters in dependency
// order.
func deriveRollerCageParams(b *PartBuilder) error {
	for _, d := range rollerCageDerivations() {
		if err := b.DeriveParam(d.name, d.expr); err != nil {
			return err
		}
	}
	return nil
}

// rollerCageBarsFit reports whether the inter-roller gap can seat a bridge bar: the free half-gap
// π/roller_count − asin(roller_dia/pitch_dia) must clear the floor. Mirrors the parametric
// derivation above so the Go build decision matches the published geometry. Named distinctly from
// tapered_cage.go's cageBarsFit (same package, unrelated cone-angle guard).
func rollerCageBarsFit(rm ResolvedMember) bool {
	d, D, n := rm.Value("d"), rm.Value("D"), rm.Value("Z")
	if n < 3 {
		return false
	}
	gap := D - d
	rollerDia, pitchDia := 0.28*gap, (d+D)/2
	freeHalfGap := math.Pi/n - math.Asin(rollerDia/pitchDia)
	return freeHalfGap > rollerCageBarGapFloor
}

// buildRollerCageBar builds one bridge bar as a thin ring-section revolved a small ±angle about Z
// on a plane at the half-pitch azimuth, so it sits centred in the gap between two rollers. Built
// BEFORE the roller pattern so the single pattern arrays roller+bar together.
func (b *PartBuilder) buildRollerCageBar() error {
	sk, err := b.AngledOrientedSketch("cage_half_pitch")
	if err != nil {
		return err
	}
	if err := sk.GroundedRingSection("bar_id", "bar_od", "bar_axial_len"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.RevolveTwoSided(sk, "origin/axis/z", "bar_half_angle", "new")
	return err
}
