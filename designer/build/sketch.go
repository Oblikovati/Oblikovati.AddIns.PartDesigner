// SPDX-License-Identifier: GPL-2.0-only

package build

import (
	"fmt"

	"oblikovati.org/api/client"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

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

// rectHalfSeed is the arbitrary construction half-extent (cm) the centred rectangle is drawn at
// before its dimensions drive it to the member's true size — a non-degenerate seed the solver
// scales, exactly as GroundedCircle's literal centre is a seed the diameter dimension overrides.
const rectHalfSeed = 1.0

// GroundedRectangle adds an axis-aligned rectangle centred on the sketch origin and fully
// constrains it (DOF 0) so widthExpr (its X extent) and heightExpr (its Y extent) — parameter
// names or formulas — drive its size. It is the profile every rectangular structural stock
// shares (flat bar now; the flanges and web of an I-beam later build on the same centred-profile
// convention, so the extruded solid's neutral axis is the sketch origin).
//
// The centre-variant rectangle seeds the four corners symmetric about the origin; the axis
// constraints make them a rigid rectangle; the two side dimensions set the size and the two
// offset dimensions (from a grounded origin point to the bottom and left edges, each at half the
// size) pin the centre back onto the origin.
func (s *SketchContext) GroundedRectangle(widthExpr, heightExpr string) error {
	res, err := s.b.api.Sketch().AddEntity(wire.AddSketchEntityArgs{
		SketchIndex: s.index, Kind: string(types.SketchEntityRectangle), Variant: "center",
		Points: [][]float64{{0, 0}, {rectHalfSeed, rectHalfSeed}},
	})
	if err != nil {
		return fmt.Errorf("add centred rectangle: %w", err)
	}
	if len(res.PointIDs) < 4 || len(res.EntityIDs) < 4 {
		return fmt.Errorf("rectangle reply short: corners=%d, edges=%d (want 4 each)", len(res.PointIDs), len(res.EntityIDs))
	}
	return s.constrainCenteredRectangle(res.PointIDs, res.EntityIDs, widthExpr, heightExpr)
}

// constrainCenteredRectangle pins a rectangle (corners in order bottom-left, bottom-right,
// top-right, top-left; edges in order bottom, right, top, left) to DOF 0 centred on the origin.
// The axis constraints make it a rigid rectangle (4 DOF: centre x/y + width/height); the two side
// dimensions fix the size; the two offset dimensions from the grounded origin point to the bottom
// and left edges (each half the corresponding size) fix the centre at the origin.
func (s *SketchContext) constrainCenteredRectangle(corners, edges []uint64, widthExpr, heightExpr string) error {
	bl, br, tr, tl := corners[0], corners[1], corners[2], corners[3]
	bottom, left := edges[0], edges[3]
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	if err := rectAxisConstraints(con, bl, br, tr, tl); err != nil {
		return err
	}
	origin, err := s.b.api.Sketch().AddPoint(s.index, []float64{0, 0})
	if err != nil {
		return fmt.Errorf("add rectangle centre point: %w", err)
	}
	if len(origin.PointIDs) < 1 {
		return fmt.Errorf("rectangle centre point reply had no point id")
	}
	o := origin.PointIDs[0]
	if _, err := con.Fix(o); err != nil {
		return fmt.Errorf("fix rectangle centre: %w", err)
	}
	return centeredRectangleDimensions(dim, o, bl, br, tl, bottom, left, widthExpr, heightExpr)
}

// centeredRectangleDimensions sizes the rectangle (width across the bottom edge, height up the
// left edge) and centres it on the grounded origin point via the two half-size offset dimensions.
func centeredRectangleDimensions(dim client.Dimension, o, bl, br, tl, bottom, left uint64, widthExpr, heightExpr string) error {
	if _, err := dim.Distance(bl, br, widthExpr); err != nil {
		return fmt.Errorf("dimension rectangle width %q: %w", widthExpr, err)
	}
	if _, err := dim.Distance(bl, tl, heightExpr); err != nil {
		return fmt.Errorf("dimension rectangle height %q: %w", heightExpr, err)
	}
	if _, err := dim.Offset(o, bottom, "("+heightExpr+") / 2"); err != nil {
		return fmt.Errorf("centre rectangle vertically: %w", err)
	}
	if _, err := dim.Offset(o, left, "("+widthExpr+") / 2"); err != nil {
		return fmt.Errorf("centre rectangle horizontally: %w", err)
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

// coilSectionInner, coilSectionOuter, coilSectionTop are the arbitrary construction size (cm) the
// cross-section box is drawn at before its dimensions drive it — non-degenerate seeds the solver
// scales, exactly as GroundedCircle's literal centre is a seed the diameter overrides.
const (
	coilSectionInner = 0.5
	coilSectionOuter = 1.0
	coilSectionTop   = 0.2
)

// GroundedRadialSection adds the rectangular cross-section a coil sweeps into a helical (spring)
// washer: a box spanning radially from inner_dia/2 to outer_dia/2 and thickness tall, sitting at
// that radius from a grounded origin point on a plane containing the coil axis (XZ). It is fully
// constrained (DOF 0), so the section re-drives with the washer's parameters. AddRectangle only
// makes four coincident-cornered lines (it lands under-constrained), so the constraints below
// axis-align the edges, level the section with the axis, and dimension its position and size.
func (s *SketchContext) GroundedRadialSection(innerDiaExpr, outerDiaExpr, thicknessExpr string) error {
	sk := s.b.api.Sketch()
	origin, err := sk.AddPoint(s.index, []float64{0, 0})
	if err != nil {
		return fmt.Errorf("add section origin point: %w", err)
	}
	rect, err := sk.AddRectangle(s.index, []float64{coilSectionInner, 0}, []float64{coilSectionOuter, coilSectionTop}, false)
	if err != nil {
		return fmt.Errorf("add section rectangle: %w", err)
	}
	if len(origin.PointIDs) < 1 || len(rect.PointIDs) < 4 {
		return fmt.Errorf("section reply short: origin points=%d, corners=%d", len(origin.PointIDs), len(rect.PointIDs))
	}
	return s.constrainRadialSection(origin.PointIDs[0], rect.PointIDs, innerDiaExpr, outerDiaExpr, thicknessExpr)
}

// constrainRadialSection pins the cross-section (corners in order bottom-left, bottom-right,
// top-right, top-left) to DOF 0: the origin point is grounded, the edges are axis-aligned, the
// bottom-left corner sits level with the origin, and the dimensions place and size the box.
func (s *SketchContext) constrainRadialSection(o uint64, p []uint64, innerDia, outerDia, thickness string) error {
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	bl, br, tr, tl := p[0], p[1], p[2], p[3]
	if _, err := con.Ground(o); err != nil {
		return fmt.Errorf("ground section origin: %w", err)
	}
	if err := rectAxisConstraints(con, bl, br, tr, tl); err != nil {
		return err
	}
	if _, err := con.Horizontal(o, bl); err != nil {
		return fmt.Errorf("level section with axis: %w", err)
	}
	return sectionDimensions(dim, o, bl, br, tl, innerDia, outerDia, thickness)
}

// sectionDimensions drives the cross-section's radial position and size from the parameters: the
// bottom-left corner at inner_dia/2 from the axis, the radial width (outer_dia−inner_dia)/2, and
// the thickness — so the swept ring spans the member's inner and outer diameters.
func sectionDimensions(dim client.Dimension, o, bl, br, tl uint64, innerDia, outerDia, thickness string) error {
	if _, err := dim.Distance(o, bl, "("+innerDia+") / 2"); err != nil {
		return fmt.Errorf("position section at inner radius: %w", err)
	}
	if _, err := dim.Distance(bl, br, "(("+outerDia+") - ("+innerDia+")) / 2"); err != nil {
		return fmt.Errorf("dimension section radial width: %w", err)
	}
	if _, err := dim.Distance(bl, tl, thickness); err != nil {
		return fmt.Errorf("dimension section thickness: %w", err)
	}
	return nil
}

// rectAxisConstraints makes both horizontal edges horizontal and both vertical edges vertical, so
// a positioned corner plus the two side dimensions leaves a rigid rectangle (DOF 0).
func rectAxisConstraints(con client.Constrain, bl, br, tr, tl uint64) error {
	if _, err := con.Horizontal(bl, br); err != nil {
		return fmt.Errorf("bottom edge horizontal: %w", err)
	}
	if _, err := con.Horizontal(tl, tr); err != nil {
		return fmt.Errorf("top edge horizontal: %w", err)
	}
	if _, err := con.Vertical(bl, tl); err != nil {
		return fmt.Errorf("left edge vertical: %w", err)
	}
	if _, err := con.Vertical(br, tr); err != nil {
		return fmt.Errorf("right edge vertical: %w", err)
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
