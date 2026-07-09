// SPDX-License-Identifier: GPL-2.0-only

package build

// Tapered-roller proportions. A tapered-roller bearing (ISO 355) is a cone (inner ring) and a cup
// (outer ring) whose raceways are truncated cones, with tapered rollers between them. Its DEFINING
// property is that the cup-raceway cone, cone-raceway cone and every roller cone share ONE common
// apex O on the bearing axis — the on-apex condition that lets the rollers roll without sliding. So
// every raceway and roller generator is the SAME ray from O; the roller diameter is derived from the
// apex geometry, not chosen. Only the boundary dimensions and the contact angle are tabulated.
//
// With the ISO contact angle α = cup-raceway angle from the bearing axis, choose a small roller
// half-angle β = α/8 (a modelled proportion, not tabulated); then the pitch-cone / roller-axis angle
// is δ = α − β and the cone-raceway angle is γ_cone = α − 2β. See the geometry-math-advisor
// derivation (#54): the earlier model offset the roller centre at tan α over the full span, which is
// only parallel-not-coapical (the roller-pitch apex and race apexes land at different axial points).
const (
	// taperRollerAxialFraction is the roller's axial span as a fraction of the ring width.
	taperRollerAxialFraction = "0.65"
	// The three on-apex cone angles as fractions of the contact angle α, for a roller half-angle
	// β = α/8: γ_cone = α − 2β = 0.75·α (cone raceway) and δ = α − β = 0.875·α (roller axis / pitch
	// cone). The cup raceway is α itself, so it needs no fraction.
	taperConeRayFraction = "0.75"  // γ_cone = α − 2β
	taperAxisFraction    = "0.875" // δ = α − β
	// The raceways are collinear with the roller surfaces (the shared apex rays), so the clearance
	// only has to cover the tessellation facet error, and that error is ASYMMETRIC. The cone is a
	// revolved OUTER surface: its facets recede toward the axis, AWAY from the rollers, so a hairline
	// clearance keeps the rollers reading as seated. The cup is a revolved INNER surface: its facets
	// bulge toward the rollers, so it needs more clearance or the ring pokes through the rollers.
	taperConeClearance = "0.02" // fraction of big-end roller diameter, cone (inner race) side
	taperCupClearance  = "0.05" // fraction of big-end roller diameter, cup (outer race) side
	// Cone big-rib (retaining flange) proportions. The rib foot sits a small axial gap beyond the
	// roller big end; the crest stands proud of the roller centre but clears the cup.
	taperRibFootGap = "0.04" // rib-foot axial gap beyond the roller big end, as a fraction of width
	taperRibProud   = "0.8"  // rib crest above the roller centre, as a fraction of roller_big_dia
	taperRibCupGap  = "0.3"  // rib-crest-to-cup radial gap, as a fraction of roller_big_dia
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

// deriveTaperParams adds the on-apex derived geometry: the three raceway/roller cone angles, the
// common apex arm, the roller-end stations measured from the apex, the raceway diameters as the
// shared apex rays (2·ζ·tan γ), the roller centre/diameter (circles centred at the raceway radial
// midpoint, so the roller touches both races at both ends), the cone big-rib bounds, and the ring
// raceway diameters at the ring faces. Every raceway and roller generator is the SAME ray from the
// apex, so the construction is provably on-apex (see the geometry-math-advisor derivation, #54).
//
// The apex is far off-sketch (≈ p/tan δ ≈ 100+ mm), so it is never materialised as a point — only
// the tan-expressions reference it. The roller diameter is DERIVED here (it falls out of the apex
// rays), not chosen from the radial gap as the earlier parallel-not-coapical model did.
func deriveTaperParams(b *PartBuilder) error {
	for _, d := range taperDerivations() {
		if err := b.DeriveParam(d.name, d.expr); err != nil {
			return err
		}
	}
	return nil
}

// taperDerivations lists the on-apex derived parameters in dependency order.
func taperDerivations() []struct{ name, expr string } {
	return []struct{ name, expr string }{
		{"pitch_dia", "(bore + outer_dia) / 2"},
		{"roller_axial", "width * " + taperRollerAxialFraction},
		// On-apex cone angles: cup ray = α; roller half-angle β = α/8; cone ray = α−2β; axis δ = α−β.
		{"cone_ray_angle", "contact_angle * " + taperConeRayFraction},
		{"axis_angle", "contact_angle * " + taperAxisFraction},
		// Common apex: the pitch cone meets the axis this far on the small-end side (apex never drawn).
		{"apex_arm", "(pitch_dia / 2) / tan(axis_angle)"},
		// Roller-end stations from the apex (big end farther out at +Z, small end nearer the apex).
		{"zeta_big", "apex_arm + roller_axial / 2"},
		{"zeta_small", "apex_arm - roller_axial / 2"},
		// Raceway diameters at the roller ends = 2·ζ·tan γ — the SAME apex rays for races and rollers.
		{"cone_big_dia", "2 * zeta_big * tan(cone_ray_angle)"},
		{"cone_small_dia", "2 * zeta_small * tan(cone_ray_angle)"},
		{"cup_big_dia", "2 * zeta_big * tan(contact_angle)"},
		{"cup_small_dia", "2 * zeta_small * tan(contact_angle)"},
		// Roller circles centred at the raceway radial midpoint (touch both races at both ends).
		{"roller_big_pos", "(cone_big_dia + cup_big_dia) / 2"},
		{"roller_small_pos", "(cone_small_dia + cup_small_dia) / 2"},
		{"roller_big_dia", "(cup_big_dia - cone_big_dia) / 2"},
		{"roller_small_dia", "(cup_small_dia - cone_small_dia) / 2"},
		// Method-C large-end sphere radius (centred at the apex) — published for a future domed roller.
		{"roller_sphere_r", "zeta_big / cos(contact_angle)"},
		// Asymmetric tessellation clearances so each raceway sits just off the rollers (see constants).
		{"cone_clr", "roller_big_dia * " + taperConeClearance},
		{"cup_clr", "roller_big_dia * " + taperCupClearance},
		// Cone big rib: foot a small axial gap beyond the roller big end; crest proud of the roller
		// centre but clear of the cup (min guards a shallow bearing whose window would otherwise close).
		{"rib_inner_z", "roller_axial / 2 + width * " + taperRibFootGap},
		{"rib_crest_dia", "min(roller_big_pos + " + taperRibProud + " * roller_big_dia, " +
			"cup_big_dia - " + taperRibCupGap + " * roller_big_dia)"},
		// Ring raceway diameters at the ring faces = the apex rays, offset by the clearance. The cone
		// runs from the small-end face to the rib foot; the cup spans the full width.
		{"cone_bottom_dia", "2 * (apex_arm - width / 2) * tan(cone_ray_angle) - cone_clr"},
		{"cone_rib_dia", "2 * (apex_arm + rib_inner_z) * tan(cone_ray_angle) - cone_clr"},
		{"cup_bottom_dia", "2 * (apex_arm - width / 2) * tan(contact_angle) + cup_clr"},
		{"cup_top_dia", "2 * (apex_arm + width / 2) * tan(contact_angle) + cup_clr"},
	}
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

// revolveCone revolves the inner ring (straight bore, sloped raceway, big-end retaining rib) about
// Z. The raceway runs from the small-end face up to the rib foot; beyond it the rib rises to the
// crest, guiding the roller big ends.
func (b *PartBuilder) revolveCone() error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedRibbedConeSection("bore", "cone_bottom_dia", "cone_rib_dia",
		"rib_crest_dia", "rib_inner_z", "width"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}

// revolveCup revolves the outer ring (straight OD, sloped inner raceway) about Z. The raceway is the
// cup apex ray: cup_bottom_dia at the small-end face (−width/2), cup_top_dia at the big end.
func (b *PartBuilder) revolveCup() error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedCupSection("outer_dia", "cup_bottom_dia", "cup_top_dia", "width"); err != nil {
		return err
	}
	if err := sk.AssertFullyConstrained(); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}
