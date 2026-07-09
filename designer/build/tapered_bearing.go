// SPDX-License-Identifier: GPL-2.0-only

package build

// Tapered-roller proportions. A tapered-roller bearing (ISO 355) is a cone (inner ring) and a cup
// (outer ring) whose raceways are truncated cones, with tapered rollers between them tilted by the
// contact angle. Only the boundary dimensions and the contact angle are tabulated, so the roller
// and race geometry are derived from bore/outer_dia/width and the angle.
const (
	// taperRollerAxialFraction is the roller's axial span as a fraction of the ring width.
	taperRollerAxialFraction = "0.65"
	// taperRollerBigFraction sizes the roller's big-end diameter as a fraction of the radial gap.
	taperRollerBigFraction = "0.17"
	// taperRollerSmallRatio is the small-end diameter as a fraction of the big-end diameter.
	taperRollerSmallRatio = "0.62"
	// The raceways are collinear with the roller surfaces (not merely tangent at the ends), so the
	// clearance only has to cover the tessellation facet error, and that error is ASYMMETRIC. The
	// cone is a revolved OUTER surface: its facets are chords that recede toward the axis, AWAY from
	// the rollers, so a hairline clearance keeps the rollers reading as seated against the inner race.
	// The cup is a revolved INNER surface: its facets bulge toward the axis, TOWARD the rollers, so it
	// needs enough clearance to clear that intrusion or the ring pokes through the rollers.
	taperConeClearance = "0.02" // fraction of big-end roller diameter, cone (inner race) side
	taperCupClearance  = "0.05" // fraction of big-end roller diameter, cup (outer race) side
)

// TaperedRoller generates a tapered-roller bearing (ISO 355, 302xx/303xx/313xx series)
// representationally: a cone (inner ring) and a cup (outer ring) with truncated-cone raceways, and
// a circular pattern of tapered rollers between them, each roller a frustum tilted by the contact
// angle. Bore, outer diameter, width and contact angle drive everything; the roller and race
// diameters are derived, so the bearing re-drives with the size. Roller-end sphere ends, the cage
// and the exact on-apex geometry are a tracked refinement.
type TaperedRoller struct{}

// Kind is the family `generator` binding for tapered-roller bearings.
func (TaperedRoller) Kind() string { return "tapered_roller" }

// Build publishes bore/outer_dia/width/roller_count/contact_angle, derives the roller and race
// geometry, patterns the tapered rollers FIRST (while they are the only bodies, so the pattern's
// whole-body copy does not replicate the rings), then revolves the cone and cup.
func (TaperedRoller) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if err := deriveTaperParams(b); err != nil {
		return err
	}
	if err := b.patternTaperedRollers(); err != nil {
		return err
	}
	if err := b.revolveCone(); err != nil {
		return err
	}
	return b.revolveCup()
}

// deriveTaperParams adds the derived geometry: the pitch circle, the roller axial span, the big/
// small roller diameters, the big/small roller-centre diameters (offset radially by the contact
// angle so the roller tilts), and the four race diameters.
//
// The raceways must be COLLINEAR with the roller surfaces, not merely tangent at the roller ends.
// The roller inner-surface crest is cone_big at the big end (z = +roller_axial/2) and cone_small at
// the small end (z = −roller_axial/2); the cone raceway is that same line, but the ring section
// spans the full width, so the slope is extrapolated from the roller span (roller_axial) out to the
// width — cone_half scales the half-spread by width/roller_axial. Setting the raceway only tangent
// at the roller ends (the naive `roller_big_pos − roller_big_dia` at ±width/2) makes the raceway
// slope shallower than the roller, so the tilted roller pokes through the ring near its ends. The
// cup mirrors this on the outer surface. Each raceway is then offset by taper_clr (cone inward, cup
// outward) so the rings sit just clear of the rollers.
func deriveTaperParams(b *PartBuilder) error {
	derived := []struct{ name, expr string }{
		{"pitch_dia", "(bore + outer_dia) / 2"},
		{"roller_axial", "width * " + taperRollerAxialFraction},
		{"roller_big_dia", raceGap + " * " + taperRollerBigFraction},
		{"roller_small_dia", "roller_big_dia * " + taperRollerSmallRatio},
		// The roller centre moves out by roller_axial·tan(angle) from small to big end, tilting it.
		{"roller_big_pos", "pitch_dia + roller_axial * tan(contact_angle)"},
		{"roller_small_pos", "pitch_dia - roller_axial * tan(contact_angle)"},
		// Roller inner/outer crest diameters at the roller ends — the raceway tangent points.
		{"cone_big", "roller_big_pos - roller_big_dia"},
		{"cone_small", "roller_small_pos - roller_small_dia"},
		{"cup_big", "roller_big_pos + roller_big_dia"},
		{"cup_small", "roller_small_pos + roller_small_dia"},
		// Asymmetric clearances so each raceway sits just off the rollers (see the constants).
		{"cone_clr", "roller_big_dia * " + taperConeClearance},
		{"cup_clr", "roller_big_dia * " + taperCupClearance},
		// Raceway lines collinear with the roller surfaces, extrapolated from the roller span to the
		// full ring width (width/roller_axial), then offset by the clearance (cone inward, cup outward).
		{"cone_mid", "(cone_big + cone_small) / 2"},
		{"cone_half", "((cone_big - cone_small) / 2) * (width / roller_axial)"},
		{"cup_mid", "(cup_big + cup_small) / 2"},
		{"cup_half", "((cup_big - cup_small) / 2) * (width / roller_axial)"},
		{"cone_back_dia", "cone_mid + cone_half - cone_clr"},
		{"cone_front_dia", "cone_mid - cone_half - cone_clr"},
		{"cup_back_dia", "cup_mid + cup_half + cup_clr"},
		{"cup_front_dia", "cup_mid - cup_half + cup_clr"},
	}
	for _, d := range derived {
		if err := b.DeriveParam(d.name, d.expr); err != nil {
			return err
		}
	}
	return nil
}

// patternTaperedRollers lofts one tapered roller — a frustum from the big-end circle (on the +Z
// offset plane, centred at the big-end radius) to the small-end circle (on the −Z plane, centred at
// the smaller radius), so it tilts in the radial-axial plane — then circular-patterns it
// roller_count times about the Z axis into the roller complement.
func (b *PartBuilder) patternTaperedRollers() error {
	big, err := b.OffsetPlaneSketch("roller_axial / 2")
	if err != nil {
		return err
	}
	if err := big.GroundedOffsetCircle("roller_big_pos", "roller_big_dia"); err != nil {
		return err
	}
	if err := big.AssertFullyConstrained(); err != nil {
		return err
	}
	small, err := b.OffsetPlaneSketch("-roller_axial / 2")
	if err != nil {
		return err
	}
	if err := small.GroundedOffsetCircle("roller_small_pos", "roller_small_dia"); err != nil {
		return err
	}
	if err := small.AssertFullyConstrained(); err != nil {
		return err
	}
	roller, err := b.LoftNamed(big, small, "new")
	if err != nil {
		return err
	}
	return b.PatternCircular(roller, "roller_count")
}

// revolveCone revolves the inner ring (straight bore, sloped outer raceway) about Z.
func (b *PartBuilder) revolveCone() error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedConeSection("bore", "cone_front_dia", "cone_back_dia", "width"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}

// revolveCup revolves the outer ring (straight OD, sloped inner raceway) about Z.
func (b *PartBuilder) revolveCup() error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedCupSection("outer_dia", "cup_front_dia", "cup_back_dia", "width"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}
