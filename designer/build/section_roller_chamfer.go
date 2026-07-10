// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// rollerRodAxisSeed/rollerRodHalfLenSeed are the seed cm coordinates for the chamfered-rod
// meridian's axis corners (branch-picking only; the constraints below drive the real size).
const rollerRodAxisSeed, rollerRodHalfLenSeed = 3.0, 2.0

// GroundedChamferedRollerSection builds a cylindrical roller's meridian on the XZ plane: the SAME
// six-point chamfered-rod half-section as GroundedChamferedRodSection (section_round.go) — a
// rectangle with two 45° end chamfers — but standing on a centerline at X=axisXExpr (the pitch
// radius) instead of the sketch's own r=0 axis, spanning Z symmetrically about the mid-plane
// (±lengthExpr/2) so the ±roller_length/2 × pitch_dia±roller_dia envelope the plain-cylinder roller
// used is preserved. Revolving it about that centerline (RevolveAboutCenterline) turns the meridian
// into a roller whose two end faces carry the 45° chamfer built into the profile — no fragile
// post-hoc edge references (#53).
//
// disableSketchInference MUST run first: the host's auto-inference would otherwise add redundant
// H/V/coincidence relations to the polyline as it is committed, letting the solver settle on a
// degenerate configuration (the #1591 lesson the filleted I-section already hit).
func (s *SketchContext) GroundedChamferedRollerSection(axisXExpr, diameterExpr, lengthExpr, chamferExpr string) error {
	if err := s.disableSketchInference(); err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	pts := rollerRodSeedPoints()
	p, e, err := s.closedPolyline(pts)
	if err != nil {
		return err
	}
	if err := s.addRollerCenterline(pts[0], pts[5], p[0], p[5]); err != nil {
		return err
	}
	if err := s.anchorRollerRodAxis(o, p, e, axisXExpr, lengthExpr); err != nil {
		return err
	}
	return s.dimensionChamferedRod(p, e, diameterExpr, lengthExpr, chamferExpr)
}

// rollerRodSeedPoints seeds the six-point chamfered-rod outline (P0..P5, see
// GroundedChamferedRodSection) at the pitch-radius branch: P0/P5 the axis-side corners (bottom,
// top), P1..P4 the chamfered outer corners. Coordinates are cm; only the topology/branch matters —
// the constraints in GroundedChamferedRollerSection drive the real size.
func rollerRodSeedPoints() [][]float64 {
	const a, h = rollerRodAxisSeed, rollerRodHalfLenSeed
	return [][]float64{
		{a, -h}, {a + 0.4, -h}, {a + 0.5, -h + 0.1},
		{a + 0.5, h - 0.1}, {a + 0.4, h}, {a, h},
	}
}

// addRollerCenterline adds the sketch's centerline through the given seeds, then welds its two
// points onto the profile's own axis-side corners (p0, p5) — so the revolve axis exactly follows
// the meridian's axis edge instead of drifting as a separate, independently-solved pair of points
// (which would leave the sketch under-constrained: a construction line's endpoints are ordinary
// solver variables, see model/sketch/solve_sketch.go variables()).
func (s *SketchContext) addRollerCenterline(seedA, seedB []float64, p0, p5 uint64) error {
	cl, err := s.b.api.Sketch().AddCenterline(s.index, seedA, seedB)
	if err != nil {
		return fmt.Errorf("add roller centerline: %w", err)
	}
	if len(cl.PointIDs) < 2 {
		return fmt.Errorf("roller centerline reply had %d points, want 2", len(cl.PointIDs))
	}
	con := s.b.api.Sketch().Constrain(s.index)
	if _, err := con.Coincident(cl.PointIDs[0], p0); err != nil {
		return fmt.Errorf("weld centerline start to roller axis corner: %w", err)
	}
	if _, err := con.Coincident(cl.PointIDs[1], p5); err != nil {
		return fmt.Errorf("weld centerline end to roller axis corner: %w", err)
	}
	return nil
}

// anchorRollerRodAxis pins the rod's axis-side edge (p5→p0) to the pitch-radius centerline instead
// of GroundedChamferedRodSection's literal Fix: level-aligned like the plain rod, then placed by two
// offset dimensions from the grounded origin — axisXExpr radially (GroundedRingSection's idiom) and
// half(lengthExpr) axially — so the roller re-drives with pitch_dia and roller_length.
func (s *SketchContext) anchorRollerRodAxis(o uint64, p, e []uint64, axisXExpr, lengthExpr string) error {
	con := s.b.api.Sketch().Constrain(s.index)
	horiz := [][]uint64{{p[0], p[1]}, {p[4], p[5]}}
	vert := [][]uint64{{p[5], p[0]}, {p[2], p[3]}}
	if err := alignLevels(con, horiz, vert); err != nil {
		return err
	}
	dim := s.b.api.Sketch().Dimension(s.index)
	if _, err := dim.Offset(o, e[5], axisXExpr); err != nil {
		return fmt.Errorf("place roller axis at pitch radius: %w", err)
	}
	if _, err := dim.Offset(o, e[0], half(lengthExpr)); err != nil {
		return fmt.Errorf("centre roller axially: %w", err)
	}
	return nil
}
