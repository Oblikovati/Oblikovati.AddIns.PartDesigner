// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// GroundedGroovedWasherSection builds one thrust-ball-bearing washer's revolved meridian section (XZ
// plane, X = radius, Z = axial) — a flat annular race (bore → outer_dia) with a concave ground race
// groove cut into its ball-facing face at the pitch radius, fully constrained to DOF 0. The groove
// arc is concentric with the ball (its centre sits on the mid-plane at the pitch radius), so the ball
// nests in it with a uniform clearance (grooveRadius > ball radius) and the two washers' grooves
// together cradle the ball — the thrust analogue of the deep-groove race groove, on a flat face.
//
// bore / outerDia are the washer's radial extents; pitchDia locates the groove-arc centre radially;
// grooveRadius is the groove arc radius (a conformity multiple of the ball diameter); landOffset is
// the |z| of the ball-facing land face; backOffset is the |z| of the washer's back face (height/2).
// grooveDown picks the washer: the shaft (lower) washer's land and groove sit below the mid-plane,
// the housing (upper) washer's above it.
//
// The outline is 5 straight edges + 1 groove arc: back face, outer-diameter edge, outer land, inner
// land, bore edge, and the groove arc dipping from the lands into the washer. The arc is NOT tangent
// to the lands (Tangent is a solver no-op); its endpoints ride Coincident on the land lines and its
// centre is pinned to the origin (Horizontal to the mid-plane + radial Distance) exactly once.
func (s *SketchContext) GroundedGroovedWasherSection(bore, outerDia, pitchDia, grooveRadius, landOffset, backOffset string, grooveDown bool) error {
	if err := s.disableSketchInference(); err != nil {
		return err
	}
	lines, arc, err := s.addGroovedWasherEntities(grooveDown)
	if err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	if err := s.closeGroovedWasherLoop(lines, arc); err != nil {
		return err
	}
	if err := s.orientGroovedWasher(lines); err != nil {
		return err
	}
	if err := s.sizeGroovedWasher(o, lines, arc, groovedWasherDims{
		bore: bore, outerDia: outerDia, pitchDia: pitchDia,
		grooveRadius: grooveRadius, landOffset: landOffset, backOffset: backOffset,
	}); err != nil {
		return err
	}
	return s.assertNoRedundancy("grooved washer section")
}

// groovedWasherDims bundles the six parameter expressions the grooved washer section pins, keeping
// sizeGroovedWasher within the argument budget while naming each role at the call site.
type groovedWasherDims struct {
	bore, outerDia, pitchDia, grooveRadius, landOffset, backOffset string
}

// groovedWasherSeeds returns the 5 edges and 1 groove arc of the washer outline at seed coordinates
// (cm), sized on the 51106 (30×47×11). grooveDown mirrors the seeds across the mid-plane: the shaft
// washer's material sits below Z=0, the housing washer's above. The groove arc dips from the land
// into the washer, floor at the arc's Z-extreme; the seeds only pick the branch.
// Order: back face, OD edge, outer land, inner land, bore edge, [groove arc].
func groovedWasherSeeds(grooveDown bool) (lineSeeds [][2][]float64, arcSeed [3][]float64, ccw bool) {
	const bore2, od2, pit, back, land, w = 1.5, 2.35, 1.925, 0.55, 0.184, 0.187
	sign := 1.0 // housing washer: land/back/groove above the mid-plane
	ccw = true
	if grooveDown { // shaft washer: below the mid-plane, groove dips the other way
		sign, ccw = -1.0, false
	}
	bk, ld := sign*back, sign*land
	lineSeeds = [][2][]float64{
		{{bore2, bk}, {od2, bk}},     // 0 back face
		{{od2, bk}, {od2, ld}},       // 1 outer-diameter edge
		{{od2, ld}, {pit + w, ld}},   // 2 outer land
		{{pit - w, ld}, {bore2, ld}}, // 3 inner land
		{{bore2, ld}, {bore2, bk}},   // 4 bore edge
	}
	arcSeed = [3][]float64{{pit, 0}, {pit + w, ld}, {pit - w, ld}} // centre on mid-plane, start/end on the land
	return lineSeeds, arcSeed, ccw
}

// addGroovedWasherEntities lays down the 5 edges and the groove arc at their seeds.
func (s *SketchContext) addGroovedWasherEntities(grooveDown bool) ([]filletLine, filletArc, error) {
	lineSeeds, arcSeed, ccw := groovedWasherSeeds(grooveDown)
	sk := s.b.api.Sketch()
	lines := make([]filletLine, 0, len(lineSeeds))
	for _, p := range lineSeeds {
		res, err := sk.AddLine(s.index, p[0], p[1], false)
		if err != nil || len(res.PointIDs) < 2 {
			return nil, filletArc{}, fmt.Errorf("add grooved-washer edge: %w (points=%d)", err, len(res.PointIDs))
		}
		lines = append(lines, filletLine{res.EntityID, res.PointIDs[0], res.PointIDs[1]})
	}
	res, err := sk.AddArcByCenterStartEnd(s.index, arcSeed[0], arcSeed[1], arcSeed[2], ccw, false)
	if err != nil || len(res.PointIDs) < 3 {
		return nil, filletArc{}, fmt.Errorf("add grooved-washer groove arc: %w (points=%d)", err, len(res.PointIDs))
	}
	return lines, filletArc{res.EntityID, res.PointIDs[0], res.PointIDs[1], res.PointIDs[2]}, nil
}

// closeGroovedWasherLoop joins the outline into one closed region (6 junctions): back face → OD edge
// → outer land → groove arc → inner land → bore edge → back to the back face.
func (s *SketchContext) closeGroovedWasherLoop(l []filletLine, a filletArc) error {
	con := s.b.api.Sketch().Constrain(s.index)
	joins := [][2]uint64{
		{l[0].to, l[1].from}, {l[1].to, l[2].from}, {l[2].to, a.from}, {a.to, l[3].from},
		{l[3].to, l[4].from}, {l[4].to, l[0].from},
	}
	for _, j := range joins {
		if _, err := con.Coincident(j[0], j[1]); err != nil {
			return fmt.Errorf("close grooved-washer loop at %d-%d: %w", j[0], j[1], err)
		}
	}
	return nil
}

// orientGroovedWasher orients the five straight edges: the back face and the two lands horizontal
// (axis-perpendicular annular faces), the outer-diameter and bore edges vertical (axis-parallel
// cylindrical surfaces).
func (s *SketchContext) orientGroovedWasher(l []filletLine) error {
	con := s.b.api.Sketch().Constrain(s.index)
	for _, i := range []int{0, 2, 3} { // back face + two lands
		if _, err := con.Horizontal(l[i].from, l[i].to); err != nil {
			return fmt.Errorf("orient grooved-washer edge %d horizontal: %w", i, err)
		}
	}
	for _, i := range []int{1, 4} { // outer-diameter + bore edges
		if _, err := con.Vertical(l[i].from, l[i].to); err != nil {
			return fmt.Errorf("orient grooved-washer edge %d vertical: %w", i, err)
		}
	}
	return nil
}

// sizeGroovedWasher pins the back face to ±height/2, the two lands to ±land_offset, the OD and bore
// edges to their radii, and the groove arc by centre-pinning it to the origin (Horizontal to the
// mid-plane + radial Distance to the pitch circle) plus its radius. The arc endpoints then land on
// the land lines by the loop coincidences, so their radial position is derived, not dimensioned.
func (s *SketchContext) sizeGroovedWasher(o uint64, l []filletLine, a filletArc, d groovedWasherDims) error {
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	if _, err := dim.Offset(o, l[0].id, d.backOffset); err != nil {
		return fmt.Errorf("place grooved-washer back face: %w", err)
	}
	for _, i := range []int{2, 3} {
		if _, err := dim.Offset(o, l[i].id, d.landOffset); err != nil {
			return fmt.Errorf("place grooved-washer land %d: %w", i, err)
		}
	}
	if _, err := dim.Offset(o, l[1].id, half(d.outerDia)); err != nil {
		return fmt.Errorf("place grooved-washer outer-diameter edge: %w", err)
	}
	if _, err := dim.Offset(o, l[4].id, half(d.bore)); err != nil {
		return fmt.Errorf("place grooved-washer bore edge: %w", err)
	}
	if _, err := con.Horizontal(o, a.centre); err != nil {
		return fmt.Errorf("level groove arc centre with the mid-plane: %w", err)
	}
	if _, err := dim.Distance(o, a.centre, half(d.pitchDia)); err != nil {
		return fmt.Errorf("place groove arc centre on the pitch circle: %w", err)
	}
	if _, err := dim.Radius(a.id, d.grooveRadius); err != nil {
		return fmt.Errorf("dimension washer groove radius %q: %w", d.grooveRadius, err)
	}
	return nil
}
