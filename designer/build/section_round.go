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
