// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// SketchContext is a handle to a live sketch, offering the constrained, parameter-driven
// primitives generators build profiles from. Its helpers encode the DOF-0 discipline
// (pin + dimension), and AssertFullyConstrained verifies it.
type SketchContext struct {
	b     *PartBuilder
	index int
}

// Index is the host sketch index, for direct client calls a generator needs beyond the
// helpers here.
func (s *SketchContext) Index() int { return s.index }

// GroundedCircle adds a circle centred at (cx, cy), pins its centre, and drives its diameter
// by diameterExpr (a parameter name or formula) — a fully-constrained circular profile
// (centre fixes 2 DOF, the diameter dimension the 3rd). The bare centre-radius entity only
// captures its radius at construction; the diameter dimension is what binds it to the
// parameter, so editing the parameter re-drives the circle.
func (s *SketchContext) GroundedCircle(cx, cy float64, diameterExpr string) error {
	sk := s.b.api.Sketch()
	res, err := sk.AddCircleByCenterRadius(s.index, []float64{cx, cy}, "("+diameterExpr+")/2", false)
	if err != nil {
		return fmt.Errorf("add circle: %w", err)
	}
	if len(res.PointIDs) == 0 {
		return fmt.Errorf("circle returned no centre point (entity %d)", res.EntityID)
	}
	if _, err := sk.Constrain(s.index).Fix(res.PointIDs[0]); err != nil {
		return fmt.Errorf("fix circle centre: %w", err)
	}
	if _, err := sk.Dimension(s.index).Diameter(res.EntityID, diameterExpr); err != nil {
		return fmt.Errorf("dimension circle diameter %q: %w", diameterExpr, err)
	}
	return nil
}

// hexFlatSeed is the arbitrary construction span (cm) the hexagon is drawn at before its
// across-corners dimension drives it to the member's true size — a non-degenerate seed the
// solver scales, exactly as GroundedCircle's literal centre is a seed the diameter dimension
// overrides. The dimension, not this literal, is authoritative.
const hexFlatSeed = 1.0

// GroundedHexagon adds a regular hexagon centred on the origin with one flat facing +X, then
// fully constrains it (DOF 0) so acrossFlatsExpr — a parameter name or formula — drives its
// wrench size. AddPolygon already makes it a rigid *regular* hexagon (a construction
// circumscribed circle pins the vertices to one radius, equal edges make it regular); this
// pins the remaining four DOF: the centre (Fix, 2), the rotation (the +X flat is Vertical, 1)
// and the size (1). Size is set by an across-corners distance dimension: a hexagon's
// corner-to-corner span is its across-flats over cos 30°, so the dimension expression is
// derived from the across-flats parameter and re-drives the head when that parameter changes.
func (s *SketchContext) GroundedHexagon(acrossFlatsExpr string) error {
	res, err := s.b.api.Sketch().AddPolygon(s.index, []float64{0, 0}, []float64{hexFlatSeed, 0}, 6, "circumscribed", false)
	if err != nil {
		return fmt.Errorf("add hexagon: %w", err)
	}
	if len(res.PointIDs) < 7 {
		return fmt.Errorf("hexagon returned %d points, want 6 corners + centre", len(res.PointIDs))
	}
	return s.constrainHexagon(res.PointIDs, acrossFlatsExpr)
}

// constrainHexagon pins a regular hexagon (its PointIDs are the six corners in order followed
// by the centre) to DOF 0 from the across-flats parameter. Corners 0 and 3 are diametrically
// opposite, so their distance is the across-corners span; corners 0 and 1 span the +X flat,
// so making that edge vertical fixes the rotation.
func (s *SketchContext) constrainHexagon(pointIDs []uint64, acrossFlatsExpr string) error {
	corners, center := pointIDs[:6], pointIDs[6]
	con := s.b.api.Sketch().Constrain(s.index)
	if _, err := con.Fix(center); err != nil {
		return fmt.Errorf("fix hexagon centre: %w", err)
	}
	if _, err := con.Vertical(corners[0], corners[1]); err != nil {
		return fmt.Errorf("pin hexagon rotation: %w", err)
	}
	span := "(" + acrossFlatsExpr + ") / cos(30 deg)"
	if _, err := s.b.api.Sketch().Dimension(s.index).Distance(corners[0], corners[3], span); err != nil {
		return fmt.Errorf("dimension hexagon across-corners %q: %w", span, err)
	}
	return nil
}

// AssertFullyConstrained fails when the sketch is not DOF-0, so a generator that leaves a
// profile under-constrained is caught rather than silently producing a floppy part. The
// host computes the real DOF from the constraint solver.
func (s *SketchContext) AssertFullyConstrained() error {
	st, err := s.b.api.Sketch().ConstraintStatus(s.index)
	if err != nil {
		return fmt.Errorf("constraint status of sketch %d: %w", s.index, err)
	}
	if st.DOF != 0 {
		return fmt.Errorf("sketch %d under-constrained: DOF=%d (want 0)", s.index, st.DOF)
	}
	return nil
}
