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
