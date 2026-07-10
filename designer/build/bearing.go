// SPDX-License-Identifier: GPL-2.0-only

package build

// Ball-complement proportions. A representational deep-groove ball bearing (ISO 15) is not a
// dimensioned rolling assembly — its balls and race diameters are not tabulated per size — so the
// ball diameter and race diameters are derived from the tabulated bore/outer_dia/width by fixed
// fractions of the radial gap. These place the balls on the pitch circle and leave each ring a
// solid annulus the balls sit between, which reads correctly and re-drives with the size.
const (
	// ballGapFraction sizes the ball diameter as a fraction of the radial gap (outer_dia − bore),
	// small enough that each ring keeps a solid wall once its race clears the ball crest.
	ballGapFraction = "0.28"
	// raceClearanceFraction is the extra radial-gap fraction each race is set beyond the rolling
	// element's crest circle, so the ring solid clears the element instead of overlapping it. The
	// element crest sits at pitch_dia ± element_dia; the race is pushed one more clearance past that.
	// Kept just large enough to clear the outer ring's inner-surface tessellation facets (which bulge
	// toward the element) so the elements read as seated against the races, not floating with a gap.
	raceClearanceFraction = "0.012"
)

// BallBearing generates a deep-groove ball bearing (ISO 15: 60/62/63 series) representationally: an
// inner ring and an outer ring each carrying a ground race groove, with a circular pattern of balls
// nested in the grooves on the pitch circle between them. The three tabulated dimensions — bore,
// outer diameter, width — drive everything; the pitch/ball/groove/shoulder diameters are derived
// parameters, so editing the bore re-drives the whole assembly. The ball nests in the groove with a
// uniform clearance (the groove arc sits just outside the ball surface), so the rings and balls stay
// independent bodies — no boolean. Two flat 2Z shields (one per face) are added when they fit axially
// (see shieldsFit); rubber seals are a further tracked refinement (#53).
type BallBearing struct{}

// Kind is the family `generator` binding for deep-groove ball bearings.
func (BallBearing) Kind() string { return "ball_bearing" }

// Build publishes bore/outer_dia/width/ball_count, derives the pitch, ball and race diameters,
// then builds the ball complement FIRST and the two rings after. Order matters: a circular pattern
// of a new-body source copies every body present at each occurrence (the kernel's whole-body
// replicate path for non-boolean sources), so the balls must be patterned while they are the only
// bodies — otherwise the rings would be replicated too. The rings are added afterwards as their own
// solids, revolved about the Z axis.
func (BallBearing) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if err := deriveBearingParams(b); err != nil {
		return err
	}
	if err := b.patternBalls(); err != nil {
		return err
	}
	if err := b.revolveGroovedRing("bore", "inner_shoulder_dia", true); err != nil {
		return err
	}
	if err := b.revolveGroovedRing("outer_dia", "outer_shoulder_dia", false); err != nil {
		return err
	}
	return b.revolveShields(rm)
}

// raceGap is the radial gap (outer_dia − bore) every rolling bearing's derived diameters are
// scaled from — the space the rings and rolling elements share.
const raceGap = "(outer_dia - bore)"

// derivePitchDia adds the pitch circle, midway between bore and outer diameter — the circle the
// rolling elements are centred on. Every rolling bearing derives its ring and element geometry
// from it.
func derivePitchDia(b *PartBuilder) error {
	return b.DeriveParam("pitch_dia", "(bore + outer_dia) / 2")
}

// deriveRacesClearing adds the two race diameters so each ring solid just clears the named rolling-
// element diameter rather than overlapping it. A rolling element centred on the pitch circle reaches
// a crest circle of diameter pitch_dia ± element_dia; the race is set one raceClearanceFraction of
// the gap beyond that crest, leaving a small clearance. The element diameter param must already be
// derived. Both ball and roller bearings size their rings from this.
func deriveRacesClearing(b *PartBuilder, elementDia string) error {
	clr := raceGap + " * " + raceClearanceFraction
	if err := b.DeriveParam("outer_race_dia", "pitch_dia + "+elementDia+" + "+clr); err != nil {
		return err
	}
	return b.DeriveParam("inner_race_dia", "pitch_dia - "+elementDia+" - "+clr)
}

// Race-groove proportions (see the geometry-math-advisor derivation, #53). The groove arc is a
// conformity multiple of the ball diameter (grooveConformity ≈ r_g / ball_dia), so r_g sits just
// outside the ball surface and the ball nests in the groove with a uniform clearance and no boolean.
// The raceway land (shoulder) sits shoulderFactor·r_g off the pitch circle radially, leaving a
// positive shoulder land on each side of the groove across the whole ISO 15 size range.
const (
	grooveConformity   = "0.52" // r_g = 0.52·ball_dia ⇒ r_g ≈ 1.04·ball_radius (design band 0.515–0.53)
	grooveShoulderTwoK = "1.1"  // 2k with shoulder factor k = 0.55; shoulder_dia = pitch_dia ∓ 2k·r_g
)

// deriveBearingParams adds the ball bearing's derived parameters: the pitch circle, the ball
// diameter (a fraction of the radial gap), the ground race-groove radius, and the two raceway
// shoulder diameters that flank the groove.
func deriveBearingParams(b *PartBuilder) error {
	if err := derivePitchDia(b); err != nil {
		return err
	}
	if err := b.DeriveParam("ball_dia", raceGap+" * "+ballGapFraction); err != nil {
		return err
	}
	if err := b.DeriveParam("groove_radius", "ball_dia * "+grooveConformity); err != nil {
		return err
	}
	if err := b.DeriveParam("inner_shoulder_dia", "pitch_dia - "+grooveShoulderTwoK+" * groove_radius"); err != nil {
		return err
	}
	if err := b.DeriveParam("outer_shoulder_dia", "pitch_dia + "+grooveShoulderTwoK+" * groove_radius"); err != nil {
		return err
	}
	return deriveShieldParams(b)
}

// shieldThickFraction caps a 2Z shield's axial thickness as a fraction of the bearing width.
const shieldThickFraction = "0.12"

// The 2Z shield's absolute clearances are UNIT-BEARING (mm). The parameter engine refuses to add a
// bare unitless number to a length quantity, so "ball_dia / 2 + 0.2" silently evaluated to 0 and
// collapsed both shields to zero volume (#53). Carrying "mm" makes each term a length. ISO-15
// bearings are metric; a bare constant added to a length elsewhere must do the same.
const (
	shieldEquatorClr  = "0.2 mm" // near face held this far outboard of the ball equator
	shieldEndClr      = "0.4 mm" // thickness kept this far off the ring end face
	shieldRadialInset = "0.3 mm" // radial span held this far inside each raceway shoulder
)

// deriveShieldParams adds the 2Z shield band: near face just outboard of the ball equator, thickness
// capped by the axial slack, and the radial span a hair inside the two raceway shoulders.
func deriveShieldParams(b *PartBuilder) error {
	dims := []struct{ name, expr string }{
		{"shield_near_z", "ball_dia / 2 + " + shieldEquatorClr},
		{"shield_thick", "min(width * " + shieldThickFraction + ", width / 2 - ball_dia / 2 - " + shieldEndClr + ")"},
		{"shield_far_z", "shield_near_z + shield_thick"},
		{"shield_id", "inner_shoulder_dia + " + shieldRadialInset},
		{"shield_od", "outer_shoulder_dia - " + shieldRadialInset},
	}
	for _, d := range dims {
		if err := b.DeriveParam(d.name, d.expr); err != nil {
			return err
		}
	}
	return nil
}

// shieldsFit reports whether both 2Z shields fit axially between the ball equator and the ring end
// faces: axial slack width/2 - ball_dia/2 must exceed clearance+inset+min-thickness. Else no shields.
func shieldsFit(rm ResolvedMember) bool {
	gap := rm.Value("D") - rm.Value("d")
	ballDia, width := 0.28*gap, rm.Value("B")
	const sMin = 0.70 // shield_clr_ax 0.2 + ring_inset 0.2 + t_min 0.3
	return width/2-ballDia/2 >= sMin
}

// revolveShields revolves the two metal shields (one per face) after the grooved rings, when they
// fit; each is a flat annulus spanning the groove mouth, axially outboard of the ball equator.
func (b *PartBuilder) revolveShields(rm ResolvedMember) error {
	if !shieldsFit(rm) {
		return nil
	}
	for _, negZ := range []bool{false, true} {
		if err := b.revolveOneShield(negZ); err != nil {
			return err
		}
	}
	return nil
}

// revolveOneShield builds and revolves the shield on one bearing face (negZ selects which).
func (b *PartBuilder) revolveOneShield(negZ bool) error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedShieldSection("shield_id", "shield_od", "shield_near_z", "shield_far_z", negZ); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}

// revolveRing builds one ring as a solid of revolution: a rectangular radial section from innerDia
// to outerDia, `width` tall and centred on the mid-plane, revolved a full turn about the Z axis.
func (b *PartBuilder) revolveRing(innerDia, outerDia string) error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedRingSection(innerDia, outerDia, "width"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}

// revolveGroovedRing builds one ring as a solid of revolution whose raceway carries a ground race
// groove: a grooved meridian section (far edge → axial faces → shoulders → groove arc), revolved a
// full turn about the Z axis. edgeDia is the ring's far cylindrical edge (bore/outer_dia) and
// shoulderDia its raceway land; innerRing picks whether the groove dips toward the axis or away.
func (b *PartBuilder) revolveGroovedRing(edgeDia, shoulderDia string, innerRing bool) error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedGroovedRingSection(edgeDia, shoulderDia, "pitch_dia", "groove_radius", "width", innerRing); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}

// patternBalls revolves a single ball (a semicircle of ball_dia centred on the X axis at the pitch
// radius, swept about that axis) then circular-patterns it ball_count times about the Z axis into
// the full ball complement on the pitch circle.
func (b *PartBuilder) patternBalls() error {
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := sk.GroundedBallSection("pitch_dia", "ball_dia"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	ball, err := b.Revolve(sk, "origin/axis/x", "360 deg", "new")
	if err != nil {
		return err
	}
	return b.PatternCircular(ball, "ball_count")
}
