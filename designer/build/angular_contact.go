// SPDX-License-Identifier: GPL-2.0-only

package build

// AngularContact generates an angular-contact ball bearing (ISO 15, 7000 series) representationally:
// two grounded rings whose race grooves are offset axially in OPPOSITE directions so the ball-to-race
// contact normal is tilted at the contact angle α from the radial plane — the geometry that lets the
// bearing carry a one-directional thrust load. Each ring carries a tall retaining shoulder on the
// contact side and a relieved low shoulder on the counterbore side (which exposes the balls and lets
// the bearing be assembled). Bore, outer diameter, width and contact angle drive everything; the
// pitch/ball/groove diameters and the axial groove offset are derived, so the bearing re-drives with
// the size.
//
// It differs from the deep-groove BallBearing by the axially-offset grooves and the asymmetric
// shoulders: the deep-groove bearing's grooves sit on the mid-plane with symmetric shoulders (radial
// contact only), whereas here the ±α tilt and the relieved counterbore are the visible
// angular-contact signature.
type AngularContact struct{}

// Kind is the family `generator` binding for angular-contact ball bearings.
func (AngularContact) Kind() string { return "angular_contact" }

// Build publishes bore/outer_dia/width/ball_count/contact_angle, derives the pitch/ball/groove
// diameters, the axial groove offset for the contact angle, and each ring's asymmetric shoulder
// diameters, patterns the balls FIRST (while they are the only bodies, so the pattern's whole-body
// copy does not replicate the rings), then revolves the inner and outer tilted-groove rings.
func (AngularContact) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if err := deriveAngularParams(b); err != nil {
		return err
	}
	if err := b.patternBalls(); err != nil {
		return err
	}
	if err := b.revolveAngularRing("bore", "inner_high_shoulder_dia", "inner_relief_shoulder_dia", "inner_groove_dia", true); err != nil {
		return err
	}
	return b.revolveAngularRing("outer_dia", "outer_high_shoulder_dia", "outer_relief_shoulder_dia", "outer_groove_dia", false)
}

// The angular-contact shoulder proportions (geometry-math-advisor #54, Method C). The groove centre
// is displaced along the tilted contact normal by (r_g − R) — the ball-groove clearance radius — so
// the ball nests tangent at the α-tilted contact point. The retaining shoulder is the deep-groove
// land (0.55·r_g off the groove centre); the relieved shoulder is opened out to 0.85·r_g so the low
// counterbore just clears the ball crest on that face, exposing the balls.
const (
	angularHighShoulderTwoK   = grooveShoulderTwoK // 1.1 = 2·0.55, the retaining (tall) shoulder
	angularReliefShoulderTwoK = "1.7"              // 2·0.85, the relieved (low) counterbore shoulder
)

// deriveAngularParams adds the angular-contact derived parameters on top of the shared ball-bearing
// ones: the axial and radial components of the groove-centre offset for the contact angle, each
// ring's groove-centre diameter (pitch ∓ 2·radial-offset), and the asymmetric high/relief shoulder
// diameters that flank each groove. The offset magnitude is (groove_radius − ball_dia/2), the
// ball-to-groove clearance radius; multiplied by sin/cos α it tilts the contact normal by α.
func deriveAngularParams(b *PartBuilder) error {
	if err := deriveBearingParams(b); err != nil { // pitch_dia, ball_dia, groove_radius, shoulders
		return err
	}
	const offset = "(groove_radius - ball_dia / 2)" // r_g − R, the ball-groove clearance radius
	if err := b.DeriveParam("groove_axial_offset", offset+" * sin(contact_angle)"); err != nil {
		return err
	}
	if err := b.DeriveParam("groove_radial_offset", offset+" * cos(contact_angle)"); err != nil {
		return err
	}
	if err := deriveAngularGrooveDias(b); err != nil {
		return err
	}
	return deriveAngularShoulderDias(b)
}

// deriveAngularGrooveDias adds each ring's groove-centre diameter: the outer groove bulges out to
// pitch + 2·radial-offset, the inner groove dips in to pitch − 2·radial-offset, so the two grooves
// straddle the pitch circle and their contact normals converge on the ball at ±α.
func deriveAngularGrooveDias(b *PartBuilder) error {
	if err := b.DeriveParam("outer_groove_dia", "pitch_dia + 2 * groove_radial_offset"); err != nil {
		return err
	}
	return b.DeriveParam("inner_groove_dia", "pitch_dia - 2 * groove_radial_offset")
}

// deriveAngularShoulderDias adds the asymmetric shoulder diameters for both rings: the tall retaining
// shoulder flanks the groove centre by 2·0.55·r_g and the relieved low shoulder by 2·0.85·r_g, on
// the outward side for the outer ring and the inward side for the inner ring.
func deriveAngularShoulderDias(b *PartBuilder) error {
	shoulders := []struct{ name, base, twoK, sign string }{
		{"outer_high_shoulder_dia", "outer_groove_dia", angularHighShoulderTwoK, " + "},
		{"outer_relief_shoulder_dia", "outer_groove_dia", angularReliefShoulderTwoK, " + "},
		{"inner_high_shoulder_dia", "inner_groove_dia", angularHighShoulderTwoK, " - "},
		{"inner_relief_shoulder_dia", "inner_groove_dia", angularReliefShoulderTwoK, " - "},
	}
	for _, s := range shoulders {
		if err := b.DeriveParam(s.name, s.base+s.sign+s.twoK+" * groove_radius"); err != nil {
			return err
		}
	}
	return nil
}

// revolveAngularRing builds one ring as a solid of revolution whose raceway carries an axially-offset
// ground race groove (the α-tilt) between an asymmetric tall retaining shoulder and a relieved low
// shoulder: an angular meridian section revolved a full turn about the Z axis. farDia is the ring's
// far cylindrical edge (bore/outer_dia); innerRing picks whether the groove dips toward the axis or
// away and which axial side carries the tall shoulder.
func (b *PartBuilder) revolveAngularRing(farDia, highShoulderDia, reliefShoulderDia, grooveDia string, innerRing bool) error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedAngularRingSection(farDia, highShoulderDia, reliefShoulderDia, grooveDia,
		"groove_axial_offset", "groove_radius", "width", innerRing); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}
