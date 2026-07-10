// SPDX-License-Identifier: GPL-2.0-only

package build

// Self-aligning thrust bearing (ISO 104, 532xx) proportions. Same grooved-washer race as the 511xx
// single-direction thrust bearing, but the HOUSING washer's back is a shallow concave SEAT and a
// separate SEAT washer (concave underside) cradles it, so the bearing can tilt to take up shaft
// misalignment. The seat sphere is derived from a cap-depth fraction of the outer diameter
// (geometry-math-advisor #54) — deliberately flat (large radius) to fit the washer thickness rather
// than the catalog SR. That flatness makes the seat sphere's meridian arc sub-tessellation-floor
// (sagitta < 0.1 mm over the washer width) and numerically degenerate as a sketch arc, so per the
// advisor's sub-floor fallback each concave back is built as the sphere's straight CHORD — a shallow
// cone < 0.1 mm off the true sphere, pinned by its rim stations (well-conditioned). Only the boundary
// dimensions are tabulated (ISO 104); the sphered seat is a design-level representational derivation.
const (
	// thrustSeatCapFraction is the seat sphere's cap depth over the OD radius, as a fraction of the
	// outer diameter — small (a flat seat), so the bowl clears the groove and leaves wall at the bore.
	thrustSeatCapFraction = "0.06"
	// thrustSeatWasherOuter is the seat washer OD as a fraction of the bearing OD (the seat washer is
	// larger, spreading the aligning load).
	thrustSeatWasherOuter = "1.12"
	// thrustSeatClearance is the radial gap between the housing back sphere and the seat washer's
	// concave underside, as a fraction of the height (so the two spheres do not z-fight).
	thrustSeatClearance = "0.02"
	// thrustSeatWasherThick is the seat washer's thickness above the bearing back, as a fraction of
	// the height.
	thrustSeatWasherThick = "0.35"
)

// ThrustSelfAligning generates an ISO 104 self-aligning thrust ball bearing (532xx) representationally:
// a grooved shaft washer, a grooved housing washer with a concave spherical back, a ball complement
// on the pitch circle, and a separate seat washer whose concave underside nests over the housing
// sphere with a hair clearance. Bore, outer diameter and height drive everything; the ball/groove
// geometry and the seat sphere are derived, so the whole bearing re-drives with the size.
type ThrustSelfAligning struct{}

// Kind is the family `generator` binding for self-aligning thrust ball bearings.
func (ThrustSelfAligning) Kind() string { return "thrust_self_aligning" }

// Build publishes the boundary dimensions, derives the ball/groove geometry (shared with the 511xx
// thrust bearing) and the seat sphere, patterns the balls FIRST (while they are the only bodies),
// then revolves the shaft washer, the sphered housing washer and the seat washer.
func (ThrustSelfAligning) Build(b *PartBuilder, rm ResolvedMember) error {
	if err := b.PublishParams(rm); err != nil {
		return err
	}
	if err := deriveThrustParams(b); err != nil {
		return err
	}
	if err := deriveSeatParams(b); err != nil {
		return err
	}
	if err := b.patternBalls(); err != nil {
		return err
	}
	if err := b.revolveGroovedWasher(true); err != nil { // shaft washer (lower), standard grooved
		return err
	}
	if err := b.revolveSphericalBackHousing(); err != nil {
		return err
	}
	return b.revolveSeatWasher()
}

// deriveSeatParams adds the seat sphere derived parameters: the cap depth, the seat sphere radius
// (from the cap depth and the OD, R = (a²+s²)/2s), its axis centre (anchored so the OD rim keeps the
// +height/2 back level), the seat washer OD/top level, and the seat washer's own (slightly smaller)
// sphere radius so its concave underside sits a hair clearance outboard of the housing sphere.
func deriveSeatParams(b *PartBuilder) error {
	derived := []struct{ name, expr string }{
		{"seat_cap", thrustSeatCapFraction + " * outer_dia"},
		{"seat_sphere_r", "((outer_dia / 2) * (outer_dia / 2) + seat_cap * seat_cap) / (2 * seat_cap)"},
		{"seat_centre_z", "height / 2 + sqrt(seat_sphere_r * seat_sphere_r - (outer_dia / 2) * (outer_dia / 2))"},
		{"seat_washer_od", thrustSeatWasherOuter + " * outer_dia"},
		{"seat_washer_top", "height / 2 + " + thrustSeatWasherThick + " * height"},
		// The seat washer underside is a smaller-radius sphere about the same centre, so it sits the
		// clearance outboard of (above) the housing back sphere with a uniform gap.
		{"seat_washer_r", "seat_sphere_r - " + thrustSeatClearance + " * height"},
		// Housing back-sphere rim STATIONS. The seat sphere is deliberately flat (R ≫ washer width), so
		// pinning the back arc by centre+radius is ill-conditioned (a near-flat arc → near-singular solve
		// Jacobian → the meridian collapses). Instead each rim's axial level z=z_c−√(R²−r²) is derived and
		// the rim vertex pinned DIRECTLY by its Euclidean distance from the origin (√(r²+z²)) — a
		// well-conditioned constraint (#54 self-aligning seat).
		{"seat_back_od_z", "seat_centre_z - sqrt(seat_sphere_r * seat_sphere_r - (outer_dia / 2) * (outer_dia / 2))"},
		{"seat_back_bore_z", "seat_centre_z - sqrt(seat_sphere_r * seat_sphere_r - (bore / 2) * (bore / 2))"},
		{"seat_back_od_dist", "sqrt((outer_dia / 2) * (outer_dia / 2) + seat_back_od_z * seat_back_od_z)"},
		{"seat_back_bore_dist", "sqrt((bore / 2) * (bore / 2) + seat_back_bore_z * seat_back_bore_z)"},
		// Seat-washer underside-sphere rim stations (same treatment, its own smaller sphere / larger OD).
		{"seat_under_od_z", "seat_centre_z - sqrt(seat_washer_r * seat_washer_r - (seat_washer_od / 2) * (seat_washer_od / 2))"},
		{"seat_under_bore_z", "seat_centre_z - sqrt(seat_washer_r * seat_washer_r - (bore / 2) * (bore / 2))"},
		{"seat_under_od_dist", "sqrt((seat_washer_od / 2) * (seat_washer_od / 2) + seat_under_od_z * seat_under_od_z)"},
		{"seat_under_bore_dist", "sqrt((bore / 2) * (bore / 2) + seat_under_bore_z * seat_under_bore_z)"},
	}
	for _, d := range derived {
		if err := b.DeriveParam(d.name, d.expr); err != nil {
			return err
		}
	}
	return nil
}

// revolveSphericalBackHousing revolves the housing washer: a grooved ball-facing front (the 511xx
// race) with a concave spherical back, about the Z axis.
func (b *PartBuilder) revolveSphericalBackHousing() error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedSphericalBackWasherSection("bore", "outer_dia", "pitch_dia", "groove_radius",
		"land_offset", "seat_back_od_dist", "seat_back_bore_dist"); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}

// revolveSeatWasher revolves the separate seat (aligning) washer that cradles the housing sphere.
func (b *PartBuilder) revolveSeatWasher() error {
	sk, err := b.Sketch("XZ")
	if err != nil {
		return err
	}
	if err := sk.GroundedSeatWasherSection("bore", "seat_washer_od", "seat_washer_top",
		"seat_under_od_dist", "seat_under_bore_dist"); err != nil {
		return err
	}
	_, err = b.Revolve(sk, "origin/axis/z", "360 deg", "new")
	return err
}
