// SPDX-License-Identifier: GPL-2.0-only

package build

// Ball-complement proportions. A representational deep-groove ball bearing (ISO 15) is not a
// dimensioned rolling assembly — its balls and race diameters are not tabulated per size — so the
// ball diameter and race diameters are derived from the tabulated bore/outer_dia/width by fixed
// fractions of the radial gap. These place the balls on the pitch circle and leave each ring a
// solid annulus the balls sit between, which reads correctly and re-drives with the size.
const (
	// ballGapFraction sizes the ball at a fraction of the radial gap (outer_dia − bore)/2, so a
	// ball spans most of the space between the rings without touching either race face.
	ballGapFraction = "0.3"
	// raceInsetFraction is how far each ring's race face is set back from the pitch circle,
	// as a fraction of the radial gap — clearance so the ring solids don't swallow the balls.
	raceInsetFraction = "0.21"
)

// BallBearing generates a deep-groove ball bearing (ISO 15: 60/62/63 series) representationally: a
// solid inner ring, a solid outer ring, and a circular pattern of balls on the pitch circle
// between them. The three tabulated dimensions — bore, outer diameter, width — drive everything;
// the pitch/ball/race diameters are derived parameters, so editing the bore re-drives the whole
// assembly. Race grooves and seals are a tracked refinement; the milestone calls the balls
// representational and patterned circularly.
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
	if err := b.revolveRing("bore", "inner_race_dia"); err != nil {
		return err
	}
	return b.revolveRing("outer_race_dia", "outer_dia")
}

// deriveBearingParams adds the formula parameters the geometry references: the pitch circle
// (midway between bore and outer diameter), the ball diameter (a fraction of the radial gap), and
// the two race diameters (the pitch circle inset by a fraction of the gap, so each ring is a solid
// annulus clearing the balls).
func deriveBearingParams(b *PartBuilder) error {
	const gap = "(outer_dia - bore)"
	derived := []struct{ name, expr string }{
		{"pitch_dia", "(bore + outer_dia) / 2"},
		{"ball_dia", gap + " * " + ballGapFraction},
		{"inner_race_dia", "pitch_dia - " + gap + " * " + raceInsetFraction},
		{"outer_race_dia", "pitch_dia + " + gap + " * " + raceInsetFraction},
	}
	for _, d := range derived {
		if err := b.DeriveParam(d.name, d.expr); err != nil {
			return err
		}
	}
	return nil
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
