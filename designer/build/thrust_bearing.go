// SPDX-License-Identifier: GPL-2.0-only

package build

// Thrust-bearing proportions. A single-direction thrust ball bearing (ISO 104) is two flat washer
// races with a ball complement between them; only the boundary dimensions (bore, outer diameter,
// total height) are tabulated, so the ball diameter and washer thickness are derived from the
// height as fixed fractions — the balls fill most of the axial gap and each washer takes the rest.
const (
	// thrustBallHeightFraction sizes the ball at a fraction of the total height. It is smaller than
	// (1 − 2·thrustWasherFraction) so the ball crest stays clear of both washer faces instead of
	// touching them (a tangent face interpenetrates the ball under tessellation).
	thrustBallHeightFraction = "0.4"
	// thrustWasherFraction sizes each washer's thickness as a fraction of the total height; the two
	// washers plus the ball leave a small axial clearance rather than filling the height exactly.
	thrustWasherFraction = "0.28"
)

// ThrustBearing generates a single-direction thrust ball bearing (ISO 104, 511xx series)
// representationally: a shaft washer and a housing washer (flat annular races) with a circular
// pattern of balls between them on the pitch circle, the whole stack centred on the mid-plane so
// the balls sit at z = 0 and each washer occupies one axial end. Bore, outer diameter and height
// drive everything; the pitch/ball diameters and washer thickness are derived. Grooved raceways
// and the alignment seat are a tracked refinement.
type ThrustBearing struct{}

// Kind is the family `generator` binding for thrust ball bearings.
func (ThrustBearing) Kind() string { return "thrust_bearing" }

// Build publishes bore/outer_dia/height/ball_count, derives the pitch/ball diameters and washer
// thickness, patterns the ball complement FIRST (while the balls are the only bodies, so the
// pattern's whole-body copy does not replicate the washers), then extrudes the two washers.
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
	// Shaft washer at the bottom end, housing washer at the top end; the balls sit between them.
	if err := b.washerAt("-height / 2", "washer_thickness"); err != nil {
		return err
	}
	return b.washerAt("height / 2 - washer_thickness", "washer_thickness")
}

// deriveThrustParams adds the thrust bearing's derived parameters: the pitch circle, the ball
// diameter (a fraction of the height) and the washer thickness (a fraction of the height). The two
// washers plus the ball diameter sum to less than the height, so the balls sit clear of both washer
// faces at the mid-plane rather than tangent to them.
func deriveThrustParams(b *PartBuilder) error {
	derived := []struct{ name, expr string }{
		{"pitch_dia", "(bore + outer_dia) / 2"},
		{"ball_dia", "height * " + thrustBallHeightFraction},
		{"washer_thickness", "height * " + thrustWasherFraction},
	}
	for _, d := range derived {
		if err := b.DeriveParam(d.name, d.expr); err != nil {
			return err
		}
	}
	return nil
}

// washerAt extrudes one flat annular washer race (bore .. outer_dia) of the given thickness from a
// work plane offset planeOffsetExpr along +Z: the outside diameter as a new solid, the bore cut
// through the washer. The bore cut is a small central cylinder that never reaches the balls (they
// sit at the pitch radius), so it removes only the washer's bore.
func (b *PartBuilder) washerAt(planeOffsetExpr, thicknessExpr string) error {
	outer, err := b.OffsetPlaneSketch(planeOffsetExpr)
	if err != nil {
		return err
	}
	if err := outer.GroundedCircle(0, 0, "outer_dia"); err != nil {
		return err
	}
	if err := b.ExtrudeDirected(outer, thicknessExpr, "new", ""); err != nil {
		return err
	}
	bore, err := b.OffsetPlaneSketch(planeOffsetExpr)
	if err != nil {
		return err
	}
	if err := bore.GroundedCircle(0, 0, "bore"); err != nil {
		return err
	}
	return b.ExtrudeDirected(bore, thicknessExpr, "cut", "")
}
