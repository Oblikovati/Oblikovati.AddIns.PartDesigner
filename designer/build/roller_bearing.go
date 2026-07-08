// SPDX-License-Identifier: GPL-2.0-only

package build

// Cylindrical-roller proportions, mirroring the ball bearing's representational fractions of the
// radial gap. The rollers are straight cylinders standing on the pitch circle with their axis
// parallel to the bearing axis.
const (
	// rollerGapFraction sizes the roller diameter as a fraction of the radial gap (outer_dia − bore)/2.
	rollerGapFraction = "0.3"
	// rollerLengthFraction sizes the roller length as a fraction of the bearing width, leaving the
	// small end clearances a real roller has to the ring guide flanges.
	rollerLengthFraction = "0.8"
)

// RollerBearing generates a cylindrical roller bearing (ISO 15, NU/N series) representationally: a
// solid inner ring, a solid outer ring, and a circular pattern of straight cylindrical rollers on
// the pitch circle, their axes parallel to the bearing axis. Bore, outer diameter and width drive
// everything; the pitch/roller diameters, roller length and race diameters are derived, so the
// bearing re-drives with the size and roller_count drives the pattern. Roller-end chamfers, the
// cage and the ring guide flanges are a tracked refinement.
type RollerBearing struct{}

// Kind is the family `generator` binding for cylindrical roller bearings.
func (RollerBearing) Kind() string { return "roller_bearing" }

// Build publishes bore/outer_dia/width/roller_count, derives the pitch/roller/race parameters, then
// builds the roller complement FIRST (patterned while the rollers are the only bodies, so the
// pattern's whole-body copy does not replicate the rings) and the two rings after.
func (RollerBearing) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if err := deriveRollerParams(b); err != nil {
		return err
	}
	if err := b.patternRollers(); err != nil {
		return err
	}
	if err := b.revolveRing("bore", "inner_race_dia"); err != nil {
		return err
	}
	return b.revolveRing("outer_race_dia", "outer_dia")
}

// deriveRollerParams adds the roller bearing's derived parameters: the shared race diameters plus
// the roller diameter (a fraction of the radial gap) and length (a fraction of the width).
func deriveRollerParams(b *PartBuilder) error {
	if err := deriveRaceParams(b); err != nil {
		return err
	}
	if err := b.DeriveParam("roller_dia", raceGap+" * "+rollerGapFraction); err != nil {
		return err
	}
	return b.DeriveParam("roller_length", "width * "+rollerLengthFraction)
}

// patternRollers extrudes one cylindrical roller (a circle of roller_dia at the pitch radius,
// extruded symmetric about the mid-plane so it is centred on the bearing width) then circular-
// patterns it roller_count times about the Z axis into the full roller complement.
func (b *PartBuilder) patternRollers() error {
	sk, err := b.Sketch("XY")
	if err != nil {
		return err
	}
	if err := sk.GroundedOffsetCircle("pitch_dia", "roller_dia"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	roller, err := b.ExtrudeNamed(sk, "roller_length", "new", "symmetric")
	if err != nil {
		return err
	}
	return b.PatternCircular(roller, "roller_count")
}
