// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// GroundedGroovedRingSection builds one deep-groove ball-bearing ring's revolved meridian section
// (XZ plane, X = radius, Z = axial) with a concave ground race groove on the raceway edge facing
// the ball, fully constrained to DOF 0 and centred axially on the origin. edgeDia is the
// ring's far cylindrical edge (bore for the inner ring, outer_dia for the outer ring); shoulderDia
// is the raceway land diameter; pitchDia locates the groove-arc centre on the pitch circle; and
// grooveRadius is the groove arc radius (a conformity multiple of the ball diameter, so the arc sits
// just outside the ball surface and the ball nests in it with a uniform clearance — no boolean).
//
// The outline is 5 straight edges + 1 groove arc: the far edge, two axial faces, two shoulder lands,
// and the groove arc dipping between the shoulders (see geometry-math-advisor derivation, #53). The
// groove arc is NOT tangent to the shoulders (Tangent is a solver no-op); its endpoints simply sit
// Coincident on the axis-aligned shoulder lines, and its centre is pinned to the origin (Horizontal
// + radial Distance) exactly once — the endpoints' axial position then follows from the arc radius.
func (s *SketchContext) GroundedGroovedRingSection(edgeDia, shoulderDia, pitchDia, grooveRadius, width string, innerRing bool) error {
	if err := s.disableSketchInference(); err != nil {
		return err
	}
	lines, arc, err := s.addGroovedRingEntities(innerRing)
	if err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	if err := s.closeGroovedRingLoop(lines, arc); err != nil {
		return err
	}
	if err := s.orientGroovedRing(lines); err != nil {
		return err
	}
	if err := s.sizeGroovedRing(o, lines, arc, edgeDia, shoulderDia, pitchDia, grooveRadius, width); err != nil {
		return err
	}
	return s.assertNoRedundancy("grooved ring section")
}

// groovedRingSeeds returns the 5 edges and 1 groove arc of the grooved-ring outline at seed
// coordinates (cm). The inner ring's groove dips toward the axis (−X); the outer ring's groove
// bulges away from it (+X). Seeds only pick the solver branch — the constraints drive the true size.
// Order: far edge, top axial face, top shoulder, [groove arc], bottom shoulder, bottom axial face.
func groovedRingSeeds(inner bool) (lineSeeds [][2][]float64, arcSeed [3][]float64, ccw bool) {
	const pit, w2, zs = 1.925, 0.75, 0.167 // pitch radius, half-width, groove half-axial-span
	edge, sh := 2.6, 2.035                 // outer-ring defaults
	ccw = false
	if inner {
		edge, sh, ccw = 1.25, 1.815, true
	}
	lineSeeds = [][2][]float64{
		{{edge, -w2}, {edge, w2}}, // 0 far edge (bore / outer_dia)
		{{edge, w2}, {sh, w2}},    // 1 top axial face
		{{sh, w2}, {sh, zs}},      // 2 top shoulder land
		{{sh, -zs}, {sh, -w2}},    // 3 bottom shoulder land
		{{sh, -w2}, {edge, -w2}},  // 4 bottom axial face
	}
	arcSeed = [3][]float64{{pit, 0}, {sh, zs}, {sh, -zs}} // centre, start(top shoulder), end(bottom)
	return lineSeeds, arcSeed, ccw
}

// addGroovedRingEntities lays down the 5 edges and the groove arc at their seeds and returns records.
func (s *SketchContext) addGroovedRingEntities(inner bool) ([]filletLine, filletArc, error) {
	lineSeeds, arcSeed, ccw := groovedRingSeeds(inner)
	sk := s.b.api.Sketch()
	lines := make([]filletLine, 0, len(lineSeeds))
	for _, p := range lineSeeds {
		res, err := sk.AddLine(s.index, p[0], p[1], false)
		if err != nil || len(res.PointIDs) < 2 {
			return nil, filletArc{}, fmt.Errorf("add grooved-ring edge: %w (points=%d)", err, len(res.PointIDs))
		}
		lines = append(lines, filletLine{res.EntityID, res.PointIDs[0], res.PointIDs[1]})
	}
	res, err := sk.AddArcByCenterStartEnd(s.index, arcSeed[0], arcSeed[1], arcSeed[2], ccw, false)
	if err != nil || len(res.PointIDs) < 3 {
		return nil, filletArc{}, fmt.Errorf("add grooved-ring groove arc: %w (points=%d)", err, len(res.PointIDs))
	}
	return lines, filletArc{res.EntityID, res.PointIDs[0], res.PointIDs[1], res.PointIDs[2]}, nil
}

// closeGroovedRingLoop joins the outline into one closed region (6 junctions): far edge → top face →
// top shoulder → groove arc → bottom shoulder → bottom face → back to the far edge.
func (s *SketchContext) closeGroovedRingLoop(l []filletLine, a filletArc) error {
	con := s.b.api.Sketch().Constrain(s.index)
	joins := [][2]uint64{
		{l[0].to, l[1].from}, {l[1].to, l[2].from}, {l[2].to, a.from}, {a.to, l[3].from},
		{l[3].to, l[4].from}, {l[4].to, l[0].from},
	}
	for _, j := range joins {
		if _, err := con.Coincident(j[0], j[1]); err != nil {
			return fmt.Errorf("close grooved-ring loop at %d-%d: %w", j[0], j[1], err)
		}
	}
	return nil
}

// orientGroovedRing orients the five straight edges: the far edge and the two shoulder lands
// vertical (axis-parallel cylindrical surfaces), the two axial faces horizontal.
func (s *SketchContext) orientGroovedRing(l []filletLine) error {
	con := s.b.api.Sketch().Constrain(s.index)
	for _, i := range []int{0, 2, 3} { // far edge + two shoulders
		if _, err := con.Vertical(l[i].from, l[i].to); err != nil {
			return fmt.Errorf("orient grooved-ring edge %d vertical: %w", i, err)
		}
	}
	for _, i := range []int{1, 4} { // two axial faces
		if _, err := con.Horizontal(l[i].from, l[i].to); err != nil {
			return fmt.Errorf("orient grooved-ring edge %d horizontal: %w", i, err)
		}
	}
	return nil
}

// sizeGroovedRing pins the far edge to its radius, the shoulders to the raceway land radius, the
// axial faces to ±width/2, and the groove arc by centre-pinning it to the origin (Horizontal +
// radial Distance to the pitch circle) plus its radius. The arc endpoints then land on the shoulder
// lines by the loop coincidences, so their axial position is derived, not dimensioned.
func (s *SketchContext) sizeGroovedRing(o uint64, l []filletLine, a filletArc, edgeDia, shoulderDia, pitchDia, grooveRadius, width string) error {
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	if _, err := dim.Offset(o, l[0].id, half(edgeDia)); err != nil {
		return fmt.Errorf("place grooved ring far edge: %w", err)
	}
	for _, i := range []int{2, 3} {
		if _, err := dim.Offset(o, l[i].id, half(shoulderDia)); err != nil {
			return fmt.Errorf("place grooved-ring shoulder %d: %w", i, err)
		}
	}
	for _, i := range []int{1, 4} {
		if _, err := dim.Offset(o, l[i].id, half(width)); err != nil {
			return fmt.Errorf("centre grooved-ring face %d axially: %w", i, err)
		}
	}
	if _, err := con.Horizontal(o, a.centre); err != nil {
		return fmt.Errorf("level groove arc centre with axis: %w", err)
	}
	if _, err := dim.Distance(o, a.centre, half(pitchDia)); err != nil {
		return fmt.Errorf("place groove arc centre on pitch circle: %w", err)
	}
	if _, err := dim.Radius(a.id, grooveRadius); err != nil {
		return fmt.Errorf("dimension groove radius %q: %w", grooveRadius, err)
	}
	return nil
}
