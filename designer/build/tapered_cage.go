// SPDX-License-Identifier: GPL-2.0-only

package build

import "math"

// Tapered-roller CAGE proportions (geometry-math-advisor #54). A real tapered cage is a truncated
// cone with trapezoidal windows the rollers poke through; a continuous revolved band at the pitch
// cone would PIERCE the rollers (they sit ON the pitch cone). So the representational cage is two
// disjoint pieces that are provably clear of the rollers, rings and rib:
//
//   - BRIDGE BARS: one thin bar per inter-roller gap, revolved a small angle about Z from a
//     meridian on a plane through Z at the half-pitch azimuth (π/roller_count), so it sits centred
//     in the azimuthal gap between two rollers. The bar is built as a SECOND body before the roller
//     circular pattern, so the single existing pattern (which copies every current body) arrays the
//     roller AND the bar together — the bar, offset half a pitch from the roller, lands in every gap.
//     A roller subtends a constant half-angle about the axis, φ = asin(tanβ/tanδ) (ζ cancels), so
//     one scalar guard keeps the bar clear: bar_half + φ < π/roller_count.
//   - SMALL-END RIM: a continuous revolved frustum in the small-end axial overhang (beyond the
//     roller small ends, before the small face), where no roller exists so it cannot pierce them,
//     and no rib either (the rib is at the big end). Sized to sit inside the cone→cup window.
const (
	// cageBarFill sets the bar's angular half-width as this fraction of the free half-gap
	// (π/roller_count − φ_roller), so the bar fills the middle half of the gap with equal
	// clearance on both sides.
	cageBarFill = "0.5"
	// cageBarAxialFraction / cageBarThickFraction size the bar: its axial length as a fraction of
	// the roller axial span (kept short of the big-end rib) and its radial thickness as a fraction
	// of the roller small diameter.
	cageBarAxialFraction = "0.84"
	cageBarThickFraction = "0.22"
	// Small-end rim axial band: near face just beyond the roller small end, far face just inside
	// the small face — both as fractions of the ring width, on the −Z (small-end) side.
	cageRimNearGap = "0.03" // near-face gap beyond the roller small end, fraction of width
	cageRimFarZ    = "0.46" // far-face |z|, fraction of width (just inside the −width/2 small face)
	// cageRimClearance keeps the rim off the raceways (fraction of roller small diameter): its ID
	// sits above the highest cone in the band, its OD below the lowest cup, so it never pierces.
	cageRimClearance = "0.06"
	// cageBarGapFloor is the free half-gap (radians) below which the inter-roller gap is too tight
	// to seat a visible bar, so only the small-end rim is built (a defensive guard for dense or
	// imperial members; all 10 ISO 355 sizes clear it at ≥1.5°).
	cageBarGapFloor = 0.0087 // ≈ 0.5°
)

// cageDerivations lists the cage's derived parameters (appended after the on-apex taper
// derivations, so they can reference apex_arm / pitch_dia / the raceway angles). Every distance is
// a parameter expression, so the cage re-drives with bore/outer_dia/width/contact_angle/roller_count.
func cageDerivations() []struct{ name, expr string } {
	return []struct{ name, expr string }{
		// Roller half-angle β and its constant azimuthal subtense φ = asin(tanβ/tanδ).
		{"roller_half_angle", "atan((tan(contact_angle) - tan(cone_ray_angle)) / 2)"},
		{"roller_subtend", "asin(tan(roller_half_angle) / tan(axis_angle))"},
		// Half-pitch azimuth (the bar's work-plane angle) and the bar's angular half-width.
		{"cage_half_pitch", "180 deg / roller_count"},
		{"bar_half_angle", cageBarFill + " * (cage_half_pitch - roller_subtend)"},
		// Bar meridian: a thin rectangle centred on the mid-plane at the pitch line.
		{"bar_axial", cageBarAxialFraction + " * roller_axial"},
		{"bar_thick", cageBarThickFraction + " * roller_small_dia"},
		{"bar_id", "pitch_dia - bar_thick"},
		{"bar_od", "pitch_dia + bar_thick"},
		// Small-end rim band + radii inside the cone→cup window (ID above the near-face cone, OD
		// below the far-face cup, so the frustum stays clear of both raceways across the band).
		{"cage_rim_near_z", "roller_axial / 2 + " + cageRimNearGap + " * width"},
		{"cage_rim_far_z", cageRimFarZ + " * width"},
		{"cage_rim_id", "2 * (apex_arm - cage_rim_near_z) * tan(cone_ray_angle) + " + cageRimClearance + " * roller_small_dia"},
		{"cage_rim_od", "2 * (apex_arm - cage_rim_far_z) * tan(contact_angle) - " + cageRimClearance + " * roller_small_dia"},
	}
}

// deriveCageParams publishes the cage derived parameters in dependency order.
func deriveCageParams(b *PartBuilder) error {
	for _, d := range cageDerivations() {
		if err := b.DeriveParam(d.name, d.expr); err != nil {
			return err
		}
	}
	return nil
}

// cageBarsFit reports whether the inter-roller gap is wide enough to seat a bridge bar: the free
// half-gap π/roller_count − φ_roller must clear the floor. φ_roller = asin(tanβ/tanδ) is the
// roller's constant azimuthal half-angle (ζ-independent). Mirrors the published param derivation
// so the Go build decision matches the parametric geometry.
func cageBarsFit(rm ResolvedMember) bool {
	alpha := rm.Value("alpha") * math.Pi / 180
	n := rm.Value("Z")
	if n < 3 {
		return false
	}
	coneRay, delta := 0.75*alpha, 0.875*alpha
	tanBeta := (math.Tan(alpha) - math.Tan(coneRay)) / 2
	phiRoller := math.Asin(tanBeta / math.Tan(delta))
	return math.Pi/n-phiRoller > cageBarGapFloor
}

// buildCageBar builds one bridge bar as a thin rectangular ring-section revolved a small angle
// about Z, on a plane through Z at the half-pitch azimuth so the bar sits centred in the gap
// between two rollers. It is created BEFORE the roller circular pattern so that single pattern
// arrays the bar together with the roller — the bar, half a pitch off the roller, lands in every
// gap. Two-sided revolve (±bar_half_angle) centres the wedge on the plane.
func (b *PartBuilder) buildCageBar() error {
	sk, err := b.AngledOrientedSketch("cage_half_pitch")
	if err != nil {
		return err
	}
	if err := sk.GroundedRingSection("bar_id", "bar_od", "bar_axial"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.RevolveTwoSided(sk, "origin/axis/z", "bar_half_angle", "new")
	return err
}

// revolveCageRing revolves the small-end rim: a continuous frustum in the small-end axial overhang,
// clear of the rollers (no roller there) and the rib (at the big end). Built after the rings so the
// roller pattern does not replicate it.
func (b *PartBuilder) revolveCageRing() error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedCageRingSection("cage_rim_id", "cage_rim_od", "cage_rim_near_z", "cage_rim_far_z"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}
