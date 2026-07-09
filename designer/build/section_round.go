// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// GroundedRingSection builds the rectangular cross-section of a bearing ring on the XZ plane (X
// radial, Z axial) and fully constrains it to DOF 0: the box spans radially from innerDia/2 to
// outerDia/2 and is centred axially on the origin (Z from −width/2 to width/2), so revolving it
// about the Z axis yields a ring centred on the mid-plane where the balls sit. innerDia/outerDia/
// width are parameter expressions.
func (s *SketchContext) GroundedRingSection(innerDia, outerDia, width string) error {
	// Seed a radial box straddling the mid-plane; the dimensions below drive it to true size.
	res, err := s.b.api.Sketch().AddRectangle(s.index, []float64{2, -0.5}, []float64{3, 0.5}, false)
	if err != nil {
		return fmt.Errorf("add ring section: %w", err)
	}
	if len(res.PointIDs) < 4 || len(res.EntityIDs) < 4 {
		return fmt.Errorf("ring section reply short: corners=%d, edges=%d", len(res.PointIDs), len(res.EntityIDs))
	}
	return s.constrainRingSection(res.PointIDs, res.EntityIDs, innerDia, outerDia, width)
}

// constrainRingSection pins the ring box (corners BL,BR,TR,TL; edges bottom,right,top,left) to
// DOF 0: axis-aligned, its inner (left) edge at innerDia/2 from the axis, its radial width set, and
// centred axially by the half-width offset of the bottom edge from the grounded origin.
func (s *SketchContext) constrainRingSection(p, e []uint64, innerDia, outerDia, width string) error {
	bl, br, tl := p[0], p[1], p[3]
	bottom, left := e[0], e[3]
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	if err := rectAxisConstraints(con, p[0], p[1], p[2], p[3]); err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	if _, err := dim.Offset(o, left, half(innerDia)); err != nil {
		return fmt.Errorf("place ring at inner radius: %w", err)
	}
	if _, err := dim.Distance(bl, br, "(("+outerDia+") - ("+innerDia+")) / 2"); err != nil {
		return fmt.Errorf("dimension ring radial width: %w", err)
	}
	if _, err := dim.Offset(o, bottom, half(width)); err != nil {
		return fmt.Errorf("centre ring axially: %w", err)
	}
	if _, err := dim.Distance(bl, tl, width); err != nil {
		return fmt.Errorf("dimension ring width: %w", err)
	}
	return nil
}

// GroundedOffsetCircle adds a circle whose centre sits on the +X axis at centreRadiusExpr from the
// grounded origin, with diameter diameterExpr — the profile a cylindrical roller is extruded from
// (a cylinder standing at the pitch radius, its axis parallel to the bearing axis). It is fully
// constrained (DOF 0): the centre is levelled with the axis and placed by a distance dimension, and
// the diameter dimension sizes it, so both position and size re-drive with the parameters. Unlike
// GroundedCircle (centre pinned at a literal) the centre here is parameter-placed, so the roller
// tracks the pitch circle.
func (s *SketchContext) GroundedOffsetCircle(centreRadiusExpr, diameterExpr string) error {
	const cx = 5.0 // seed centre radius (cm); the distance dimension drives the real value
	sk := s.b.api.Sketch()
	res, err := sk.AddCircleByCenterRadius(s.index, []float64{cx, 0}, "("+diameterExpr+")/2", false)
	if err != nil {
		return fmt.Errorf("add offset circle: %w", err)
	}
	if len(res.PointIDs) == 0 {
		return fmt.Errorf("offset circle returned no centre point (entity %d)", res.EntityID)
	}
	centre := res.PointIDs[0]
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	con, dim := sk.Constrain(s.index), sk.Dimension(s.index)
	if _, err := con.Horizontal(o, centre); err != nil {
		return fmt.Errorf("level offset circle with axis: %w", err)
	}
	if _, err := dim.Distance(o, centre, half(centreRadiusExpr)); err != nil {
		return fmt.Errorf("place offset circle at pitch radius: %w", err)
	}
	if _, err := dim.Diameter(res.EntityID, diameterExpr); err != nil {
		return fmt.Errorf("dimension offset circle diameter %q: %w", diameterExpr, err)
	}
	return nil
}

// GroundedChamferedRodSection builds the half-section a chamfered cylindrical pin is revolved from,
// on the XZ plane (X radial, Z axial): a rectangle from the axis (r=0) out to diameter/2, spanning
// z=0..length, with its two outer corners (the pin ends) chamfered at 45° by chamferExpr. Points
// P0..P5 run from the axis-bottom: P0(0,0), P1(d/2−c,0), P2(d/2,c), P3(d/2,length−c), P4(d/2−c,
// length), P5(0,length). It is fully constrained to DOF 0; revolving it 360° about the Z axis (the
// P5→P0 edge on r=0) yields a cylinder with a lead-in chamfer at each end. The two chamfer edges are
// left as free diagonals pinned only by their endpoints (a 45° chamfer has equal legs, so each
// hypotenuse is chamfer·√2).
func (s *SketchContext) GroundedChamferedRodSection(diameterExpr, lengthExpr, chamferExpr string) error {
	// Seeds set only the branch/topology; the constraints below drive the real size (cm-scale).
	pts := [][]float64{{0, 0}, {0.4, 0}, {0.5, 0.1}, {0.5, 3.9}, {0.4, 4}, {0, 4}}
	p, e, err := s.closedPolyline(pts)
	if err != nil {
		return err
	}
	return s.constrainChamferedRod(p, e, diameterExpr, lengthExpr, chamferExpr)
}

// constrainChamferedRod pins the six-point rod half-section to DOF 0 (see GroundedChamferedRodSection
// for the point layout). e[i] joins p[i]→p[i+1]; e[2] is the right (outer-radius) side.
func (s *SketchContext) constrainChamferedRod(p, e []uint64, dia, length, chamfer string) error {
	con := s.b.api.Sketch().Constrain(s.index)
	if _, err := con.Fix(p[0]); err != nil {
		return fmt.Errorf("fix chamfered-rod axis corner: %w", err)
	}
	// Bottom & top edges horizontal, axis & outer-radius edges vertical; the two chamfers stay free.
	horiz := [][]uint64{{p[0], p[1]}, {p[4], p[5]}}
	vert := [][]uint64{{p[5], p[0]}, {p[2], p[3]}}
	if err := alignLevels(con, horiz, vert); err != nil {
		return err
	}
	return s.dimensionChamferedRod(p, e, dia, length, chamfer)
}

// dimensionChamferedRod sizes the rod half-section: length up the axis, radius to the outer edge, the
// two chamfer feet (radius − chamfer), and the two 45° chamfer hypotenuses (chamfer·√2).
func (s *SketchContext) dimensionChamferedRod(p, e []uint64, dia, length, chamfer string) error {
	dim := s.b.api.Sketch().Dimension(s.index)
	half := "(" + dia + ") / 2"
	edge := half + " - (" + chamfer + ")"
	hyp := "(" + chamfer + ") * sqrt(2)"
	if _, err := dim.Distance(p[0], p[5], length); err != nil {
		return fmt.Errorf("dimension rod length: %w", err)
	}
	if _, err := dim.Offset(p[0], e[2], half); err != nil {
		return fmt.Errorf("dimension rod radius: %w", err)
	}
	rodDims := []struct {
		a, b uint64
		expr string
	}{
		{p[0], p[1], edge}, {p[4], p[5], edge}, {p[1], p[2], hyp}, {p[3], p[4], hyp},
	}
	for _, d := range rodDims {
		if _, err := dim.Distance(d.a, d.b, d.expr); err != nil {
			return fmt.Errorf("dimension chamfered-rod %q: %w", d.expr, err)
		}
	}
	return nil
}

// GroundedBallSection builds the half-disk a bearing ball is revolved from: a semicircular arc of
// diameter diameterExpr centred on the X axis at radius centreRadiusExpr from the origin, bulging
// to +Y, closed by its diameter line back along the X axis. Revolved 360° about the X axis (which
// runs along that diameter) it sweeps a sphere centred at (centreRadius, 0, 0); a circular pattern
// about Z then arrays the ball complement. The closing line is required — the host will not form a
// solid from an open (arc-only) profile even when the ends sit on the axis.
func (s *SketchContext) GroundedBallSection(centreRadiusExpr, diameterExpr string) error {
	const cx, r = 5.0, 1.0 // seed centre radius / ball radius (cm)
	sk := s.b.api.Sketch()
	arc, err := sk.AddArcByCenterStartEnd(s.index,
		[]float64{cx, 0}, []float64{cx - r, 0}, []float64{cx + r, 0}, false, false)
	if err != nil {
		return fmt.Errorf("add ball arc: %w", err)
	}
	if len(arc.PointIDs) < 3 {
		return fmt.Errorf("ball arc reply had %d points, want centre+start+end", len(arc.PointIDs))
	}
	line, err := sk.AddLine(s.index, []float64{cx + r, 0}, []float64{cx - r, 0}, false)
	if err != nil {
		return fmt.Errorf("add ball diameter line: %w", err)
	}
	if len(line.PointIDs) < 2 {
		return fmt.Errorf("ball line reply had %d points, want two ends", len(line.PointIDs))
	}
	if err := s.joinBallDiameter(arc.PointIDs, line.PointIDs); err != nil {
		return err
	}
	return s.constrainBallArc(arc.EntityID, arc.PointIDs, centreRadiusExpr, diameterExpr)
}

// joinBallDiameter makes the closing line's ends coincident with the arc's end (line start) and
// start (line end), closing the half-disk into one region without adding net DOF (each coincident
// removes the two the shared point added).
func (s *SketchContext) joinBallDiameter(arcPts, linePts []uint64) error {
	con := s.b.api.Sketch().Constrain(s.index)
	if _, err := con.Coincident(linePts[0], arcPts[2]); err != nil {
		return fmt.Errorf("close ball diameter at arc end: %w", err)
	}
	if _, err := con.Coincident(linePts[1], arcPts[1]); err != nil {
		return fmt.Errorf("close ball diameter at arc start: %w", err)
	}
	return nil
}

// constrainBallArc pins the ball arc (points centre, start, end) to DOF 0: the centre on the X axis
// at centreRadius from the grounded origin, the two ends level with the centre (on the X axis), and
// the arc radius at half the ball diameter — a semicircle spanning the ball's diameter on the axis.
func (s *SketchContext) constrainBallArc(arcID uint64, p []uint64, centreRadius, diameter string) error {
	centre, start, end := p[0], p[1], p[2]
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	if _, err := con.Horizontal(o, centre); err != nil {
		return fmt.Errorf("level ball centre with axis: %w", err)
	}
	if _, err := dim.Distance(o, centre, half(centreRadius)); err != nil {
		return fmt.Errorf("place ball at pitch radius: %w", err)
	}
	if _, err := con.Horizontal(centre, start); err != nil {
		return fmt.Errorf("start on axis: %w", err)
	}
	if _, err := con.Horizontal(centre, end); err != nil {
		return fmt.Errorf("end on axis: %w", err)
	}
	if _, err := dim.Radius(arcID, half(diameter)); err != nil {
		return fmt.Errorf("dimension ball radius: %w", err)
	}
	return nil
}
