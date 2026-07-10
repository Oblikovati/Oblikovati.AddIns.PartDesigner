// SPDX-License-Identifier: GPL-2.0-only

package build

import "math"

// Cylindrical-roller proportions, mirroring the ball bearing's representational fractions of the
// radial gap. The rollers are straight cylinders standing on the pitch circle with their axis
// parallel to the bearing axis.
const (
	// rollerGapFraction sizes the roller diameter as a fraction of the radial gap (outer_dia − bore),
	// small enough that each ring keeps a solid wall once its race clears the roller crest.
	rollerGapFraction = "0.28"
	// rollerLengthFraction sizes the roller length as a fraction of the bearing width, leaving the
	// small end clearances a real roller has to the ring guide flanges.
	rollerLengthFraction = "0.8"
)

// RollerBearing generates a cylindrical roller bearing (ISO 15, NU/N series) representationally: a
// plain inner ring, an outer ring that is an inward-opening ⊐ channel with two integral guide
// flanges (when flangesFit allows it, else a plain ring), and a circular pattern of cylindrical
// rollers on the pitch circle — each roller revolved about its own centerline with 45° end
// chamfers built into the meridian (when rollerChamferFits allows it, else a plain cylinder), axes
// parallel to the bearing axis. Bore, outer diameter and width drive everything; the pitch/roller
// diameters, roller length, race diameters and flange band are derived, so the bearing re-drives
// with the size and roller_count drives the pattern. The cage is a tracked refinement (#53).
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
	if err := b.patternRollers(rm); err != nil {
		return err
	}
	if err := b.revolveRing("bore", "inner_race_dia"); err != nil {
		return err
	}
	return b.revolveFlangedOuterRing(rm)
}

// deriveRollerParams adds the roller bearing's derived parameters: the pitch circle, the roller
// diameter (a fraction of the radial gap) and length (a fraction of the width), and the two race
// diameters set to clear the roller crest.
func deriveRollerParams(b *PartBuilder) error {
	if err := derivePitchDia(b); err != nil {
		return err
	}
	if err := b.DeriveParam("roller_dia", raceGap+" * "+rollerGapFraction); err != nil {
		return err
	}
	if err := b.DeriveParam("roller_length", "width * "+rollerLengthFraction); err != nil {
		return err
	}
	if err := deriveRacesClearing(b, "roller_dia"); err != nil {
		return err
	}
	if err := deriveFlangeParams(b); err != nil {
		return err
	}
	return deriveRollerChamferParams(b)
}

const rollerChamferFraction = "0.1" // 45deg chamfer leg as a fraction of roller_dia

// deriveRollerChamferParams adds the roller-end chamfer leg (45deg, equal axial & radial leg).
func deriveRollerChamferParams(b *PartBuilder) error {
	return b.DeriveParam("roller_chamfer", "roller_dia * "+rollerChamferFraction)
}

// rollerChamferFits reports whether a visible 45deg end chamfer fits: leg below half the roller
// radius (end disc stays real) and above the visibility floor. Else the roller is built plain.
func rollerChamferFits(rm ResolvedMember) bool {
	gap := rm.Value("D") - rm.Value("d")
	rollerDia := 0.28 * gap
	leg := 0.10 * rollerDia
	const cMin, epsClr = 0.15, 0.10
	return leg < rollerDia/2-epsClr && leg >= cMin
}

// buildRoller builds one cylindrical roller standing on the pitch circle as a body of revolution
// about its own centerline: chamfered when rollerChamferFits, else a plain cylinder. Returns the
// roller feature name so the caller can pattern it.
func (b *PartBuilder) buildRoller(rm ResolvedMember) (string, error) {
	if !rollerChamferFits(rm) {
		return b.buildPlainRoller()
	}
	sk, err := b.Sketch("XZ")
	if err != nil {
		return "", err
	}
	if err := sk.GroundedChamferedRollerSection(half("pitch_dia"), "roller_dia", "roller_length", "roller_chamfer"); err != nil {
		return "", err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return "", err
	}
	return b.RevolveAboutCenterline(sk, "360 deg", "new")
}

// Flange proportions: the axial clearance from a roller end to the flange inner face, as an
// absolute floor or a fraction of the roller length — whichever is larger.
const flangeAxialClrFraction = "0.02"

// deriveFlangeParams adds the outer-ring guide-flange band: the roller-end→flange clearance, the
// flange inner-face |z|, and the flange bore diameter (pitch_dia = mid roller-end annulus, so the
// rib dips roller_dia/2 below the roller crest yet keeps a land above the plain inner ring).
func deriveFlangeParams(b *PartBuilder) error {
	if err := b.DeriveParam("flange_axial_clr", "max(0.1, roller_length * "+flangeAxialClrFraction+")"); err != nil {
		return err
	}
	if err := b.DeriveParam("flange_inner_z", "roller_length / 2 + flange_axial_clr"); err != nil {
		return err
	}
	return b.DeriveParam("flange_bore_dia", "pitch_dia")
}

// flangesFit reports whether the outer ring can carry integral guide flanges: a positive land above
// the inner ring, a real locating overlap, and a visible axial overhang band. Mirrors the parametric
// guard so the Go build decision matches the geometry. Units: mm (tabulated columns).
func flangesFit(rm ResolvedMember) bool {
	d, D, B := rm.Value("d"), rm.Value("D"), rm.Value("B")
	gap := D - d
	rollerDia, rollerLen := 0.28*gap, 0.8*B
	land := (rollerDia + 0.012*gap) / 2
	overlap := rollerDia / 2
	axialClr := math.Max(0.10, 0.02*rollerLen)
	band := B/2 - rollerLen/2 - axialClr
	const epsClr = 0.10
	return land >= epsClr && overlap >= epsClr && band >= epsClr
}

// revolveFlangedOuterRing revolves the outer ring as an inward-opening channel carrying two integral
// guide flanges; falls back to a plain ring when flangesFit is false.
func (b *PartBuilder) revolveFlangedOuterRing(rm ResolvedMember) error {
	if !flangesFit(rm) {
		return b.revolveRing("outer_race_dia", "outer_dia")
	}
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedFlangedRingSection("outer_dia", "outer_race_dia", "flange_bore_dia", "flange_inner_z", "width"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}

// buildPlainRoller extrudes one plain cylindrical roller (a circle of roller_dia at the pitch
// radius, extruded symmetric about the mid-plane so it is centred on the bearing width) — the
// fallback buildRoller uses when rollerChamferFits is false. Returns the created feature's name so
// the caller can pattern it.
func (b *PartBuilder) buildPlainRoller() (string, error) {
	sk, err := b.Sketch("XY")
	if err != nil {
		return "", err
	}
	if err := sk.GroundedOffsetCircle("pitch_dia", "roller_dia"); err != nil {
		return "", err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return "", err
	}
	return b.ExtrudeNamed(sk, "roller_length", "new", "symmetric")
}

// patternRollers builds one cylindrical roller (chamfered when rollerChamferFits, else plain) then
// circular-patterns it roller_count times about the Z axis into the full roller complement. The
// roller is built BEFORE the pattern so a future cage bar (#53 Task 3) can be added here, between
// the build and the single whole-body PatternCircular call.
func (b *PartBuilder) patternRollers(rm ResolvedMember) error {
	roller, err := b.buildRoller(rm)
	if err != nil {
		return err
	}
	return b.PatternCircular(roller, "roller_count")
}
