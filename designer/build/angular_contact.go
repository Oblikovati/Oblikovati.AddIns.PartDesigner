// SPDX-License-Identifier: GPL-2.0-only

package build

// AngularContact generates an angular-contact ball bearing (ISO 15, 7000 series) representationally:
// a plain inner ring, an outer ring whose inner raceway is relieved to a low shoulder on one face
// (the characteristic angular-contact counterbore that carries the axial load and lets the balls be
// assembled), and a ball complement between them. Bore, outer diameter, width and contact angle
// drive everything; the pitch/ball/race diameters are derived, so the bearing re-drives with the
// size. Race grooves and the exact contact geometry are a tracked refinement.
//
// It differs from the deep-groove BallBearing by the relieved outer ring: the low shoulder on the
// front face exposes the balls, which is how an angular-contact bearing is recognised and how it
// takes a one-directional thrust load.
type AngularContact struct{}

// Kind is the family `generator` binding for angular-contact ball bearings.
func (AngularContact) Kind() string { return "angular_contact" }

// Build publishes bore/outer_dia/width/ball_count/contact_angle, derives the pitch/ball/race
// diameters and the relieved-shoulder diameter, patterns the balls FIRST (while they are the only
// bodies, so the pattern's whole-body copy does not replicate the rings), revolves the plain inner
// ring, then revolves the outer ring as a cup with the low front shoulder.
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
	if err := b.revolveRing("bore", "inner_race_dia"); err != nil {
		return err
	}
	return b.revolveAngularOuterRing()
}

// deriveAngularParams adds the derived diameters: the shared race diameters and ball diameter (as
// for a deep-groove bearing) plus the relieved outer-shoulder diameter — the outer raceway's low
// front shoulder, opened out to the ball crest so the balls show on that face.
func deriveAngularParams(b *PartBuilder) error {
	if err := deriveBearingParams(b); err != nil { // pitch_dia, ball_dia, inner/outer_race_dia
		return err
	}
	// The relieved (low) shoulder is opened OUTWARD from the outer race — halfway to the outside
	// diameter — so the front face clears the ball crest and exposes the balls (the counterbore that
	// lets an angular-contact bearing be assembled and carry one-directional thrust). Opening it out,
	// not in, is what distinguishes the relieved face from the retaining high shoulder on the back.
	return b.DeriveParam("relief_dia", "(outer_race_dia + outer_dia) / 2")
}

// revolveAngularOuterRing revolves the outer ring as a cup: the outside diameter is the straight
// (vertical) edge, and the inner raceway runs from the full outer-race shoulder on the back face
// (−width/2) out to the relieved low shoulder on the front face (+width/2), so the balls are
// exposed from the front — the angular-contact form.
func (b *PartBuilder) revolveAngularOuterRing() error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedCupSection("outer_dia", "outer_race_dia", "relief_dia", "width"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}
