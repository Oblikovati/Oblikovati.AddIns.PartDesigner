// SPDX-License-Identifier: GPL-2.0-only

package build

// Thrust-bearing proportions. A single-direction thrust ball bearing (ISO 104) is two washer races
// with a ball complement between them; only the boundary dimensions (bore, outer diameter, total
// height) are tabulated, so the ball diameter, groove radius and land offset are derived from the
// height. Each washer carries a ground race groove on its ball-facing face: the groove arc is
// concentric with the ball (centre on the mid-plane at the pitch radius), so the ball nests in it
// with a uniform clearance and the two washers' grooves together cradle the ball — no boolean.
const (
	// thrustBallHeightFraction sizes the ball at a fraction of the total height. The ball sinks into
	// the two grooves, so it can be a larger fraction than the flat-washer model used, while still
	// clearing its neighbours on the pitch circle across the whole 511xx range.
	thrustBallHeightFraction = "0.45"
	// thrustGrooveConformity is r_g / ball_dia — the groove arc as a conformity multiple of the ball
	// diameter (design band 0.51–0.54), so r_g sits just outside the ball surface and it nests with a
	// uniform clearance.
	thrustGrooveConformity = "0.53"
	// thrustLandFraction is land_offset / r_g: the ball-facing land face sits this fraction of the
	// groove radius off the mid-plane, so the groove is a channel of depth (1 − thrustLandFraction)·r_g
	// cut below the land, leaving positive inner/outer land annuli across the range.
	thrustLandFraction = "0.7"
)

// ThrustBearing generates a single-direction thrust ball bearing (ISO 104, 511xx series)
// representationally: a shaft washer and a housing washer (annular races with a ground race groove
// on each ball-facing face) with a circular pattern of balls between them on the pitch circle, the
// whole stack centred on the mid-plane so the balls sit at z = 0 and each washer occupies one axial
// end. Bore, outer diameter and height drive everything; the pitch/ball diameters, groove radius and
// land offset are derived. The self-aligning sphered seat is a variant (532xx) not tabulated here.
type ThrustBearing struct{}

// Kind is the family `generator` binding for thrust ball bearings.
func (ThrustBearing) Kind() string { return "thrust_bearing" }

// Build publishes bore/outer_dia/height/ball_count, derives the pitch/ball diameters, groove radius
// and land offset, patterns the ball complement FIRST (while the balls are the only bodies, so the
// pattern's whole-body copy does not replicate the washers), then revolves the two grooved washers.
func (ThrustBearing) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if err := deriveThrustParams(b); err != nil {
		return err
	}
	if err := b.patternBalls(); err != nil {
		return err
	}
	// Shaft washer below the mid-plane, housing washer above it; the balls nest in both grooves.
	if err := b.revolveGroovedWasher(true); err != nil {
		return err
	}
	return b.revolveGroovedWasher(false)
}

// deriveThrustParams adds the thrust bearing's derived parameters: the pitch circle, the ball
// diameter (a fraction of the height), the ground groove radius (a conformity multiple of the ball
// diameter) and the land offset (the |z| of each washer's ball-facing land face, a fraction of the
// groove radius, so the groove is a channel cut below the land that the ball nests in).
func deriveThrustParams(b *PartBuilder) error {
	derived := []struct{ name, expr string }{
		{"pitch_dia", "(bore + outer_dia) / 2"},
		{"ball_dia", "height * " + thrustBallHeightFraction},
		{"groove_radius", "ball_dia * " + thrustGrooveConformity},
		{"land_offset", "groove_radius * " + thrustLandFraction},
	}
	for _, d := range derived {
		if err := b.DeriveParam(d.name, d.expr); err != nil {
			return err
		}
	}
	return nil
}

// revolveGroovedWasher revolves one washer race with a ground race groove on its ball-facing face:
// a grooved washer meridian (back face → OD edge → lands → groove arc), revolved a full turn about
// the Z axis. grooveDown picks the shaft (lower) washer; false the housing (upper) washer.
func (b *PartBuilder) revolveGroovedWasher(grooveDown bool) error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedGroovedWasherSection("bore", "outer_dia", "pitch_dia", "groove_radius",
		"land_offset", "height / 2", grooveDown); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}
