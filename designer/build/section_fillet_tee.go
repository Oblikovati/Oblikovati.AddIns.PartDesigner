// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// GroundedFilletedTeeSection builds an EN 10055 equal-flange tee (flange on top, stem hanging down,
// symmetric about the Y axis) with a real concave root fillet at each of the two flange-stem
// junctions, fully constrained to DOF 0 and centred on the origin. h/b/tw/tf/r are parameter
// expressions (height, flange width, stem thickness, flange thickness, root radius).
//
// It reuses the I-section's proven filleted-section recipe (see section_fillet.go): disable host
// inference, orient every edge explicitly, and centre-pin each arc axis-aligned — the two root
// fillets are literally the I-section's two top fillets (arc 0 right, arc 1 left), so they share
// orientRootFillets with startOnFlange = {true, false}. The flange toes stay sharp: EN 10055 rounds
// them, but the published sectional area is reproduced by the root fillets alone (the two toe radii
// net out below tabulation precision), so modelling only the roots keeps the extruded area faithful
// to the standard's tabulated value (e.g. T 50×50×6 → 5.66 cm²).
func (s *SketchContext) GroundedFilletedTeeSection(h, b, tw, tf, r string) error {
	if err := s.disableSketchInference(); err != nil {
		return err
	}
	lines, arcs, err := s.addFilletedTeeEntities()
	if err != nil {
		return err
	}
	o, err := s.groundedOrigin()
	if err != nil {
		return err
	}
	if err := s.closeFilletedTeeLoop(lines, arcs); err != nil {
		return err
	}
	if err := s.orientRootFillets(arcs); err != nil {
		return err
	}
	if err := s.orientFilletedTeeEdges(lines); err != nil {
		return err
	}
	if err := s.sizeFilletedTeeSection(o, lines, arcs, h, b, tw, tf, r); err != nil {
		return err
	}
	return s.assertNoRedundancy("filleted tee section")
}

// teeFilletSeeds are the 8 straight edges and 2 root-fillet arcs of the filleted-tee outline at
// seed coordinates (cm) for a representative T 60×60×7 (H=3, B=3, W=0.35, F=2.3, r=0.2). Walking
// clockwise from the top-left flange corner; the arcs replace the two reentrant flange-stem
// corners. The seeds only pick the solver branch — the constraints drive the true size.
func teeFilletSeeds() (lineSeeds [][2][]float64, arcSeeds [][3][]float64) {
	lineSeeds = [][2][]float64{
		{{-3, 3}, {3, 3}},           // 0 flange top
		{{3, 3}, {3, 2.3}},          // 1 flange right end
		{{3, 2.3}, {0.55, 2.3}},     // 2 flange underside (right)
		{{0.35, 2.1}, {0.35, -3}},   // 3 stem right face
		{{0.35, -3}, {-0.35, -3}},   // 4 stem bottom
		{{-0.35, -3}, {-0.35, 2.1}}, // 5 stem left face
		{{-0.55, 2.3}, {-3, 2.3}},   // 6 flange underside (left)
		{{-3, 2.3}, {-3, 3}},        // 7 flange left end
	}
	arcSeeds = [][3][]float64{
		{{0.55, 2.1}, {0.55, 2.3}, {0.35, 2.1}},    // 0 right root: centre, start(underside), end(stem) — startOnFlange
		{{-0.55, 2.1}, {-0.35, 2.1}, {-0.55, 2.3}}, // 1 left root: centre, start(stem), end(underside)
	}
	return lineSeeds, arcSeeds
}

// addFilletedTeeEntities lays down the 8 edges and 2 fillet arcs at their seeds and returns their
// records. Arcs are wound CCW (start→end sweeps the quarter into the air quadrant).
func (s *SketchContext) addFilletedTeeEntities() ([]filletLine, []filletArc, error) {
	lineSeeds, arcSeeds := teeFilletSeeds()
	sk := s.b.api.Sketch()
	lines := make([]filletLine, 0, len(lineSeeds))
	for _, p := range lineSeeds {
		res, err := sk.AddLine(s.index, p[0], p[1], false)
		if err != nil || len(res.PointIDs) < 2 {
			return nil, nil, fmt.Errorf("add filleted-tee edge: %w (points=%d)", err, len(res.PointIDs))
		}
		lines = append(lines, filletLine{res.EntityID, res.PointIDs[0], res.PointIDs[1]})
	}
	arcs := make([]filletArc, 0, len(arcSeeds))
	for _, p := range arcSeeds {
		res, err := sk.AddArcByCenterStartEnd(s.index, p[0], p[1], p[2], true, false)
		if err != nil || len(res.PointIDs) < 3 {
			return nil, nil, fmt.Errorf("add filleted-tee root fillet: %w (points=%d)", err, len(res.PointIDs))
		}
		arcs = append(arcs, filletArc{res.EntityID, res.PointIDs[0], res.PointIDs[1], res.PointIDs[2]})
	}
	return lines, arcs, nil
}

// closeFilletedTeeLoop joins the outline into one closed region, walking the 10 line/arc junctions
// clockwise: line0→line1→line2→arc0→line3→line4→line5→arc1→line6→line7→line0.
func (s *SketchContext) closeFilletedTeeLoop(l []filletLine, a []filletArc) error {
	con := s.b.api.Sketch().Constrain(s.index)
	joins := [][2]uint64{
		{l[0].to, l[1].from}, {l[1].to, l[2].from}, {l[2].to, a[0].from}, {a[0].to, l[3].from},
		{l[3].to, l[4].from}, {l[4].to, l[5].from}, {l[5].to, a[1].from}, {a[1].to, l[6].from},
		{l[6].to, l[7].from}, {l[7].to, l[0].from},
	}
	for _, j := range joins {
		if _, err := con.Coincident(j[0], j[1]); err != nil {
			return fmt.Errorf("close filleted-tee loop at %d-%d: %w", j[0], j[1], err)
		}
	}
	return nil
}

// orientFilletedTeeEdges gives every straight edge an explicit horizontal/vertical constraint: the
// four horizontal edges (flange top, the two flange undersides, stem bottom) and the four vertical
// edges (the two flange ends, the two stem faces). With inference disabled the arc centre-pins alone
// do not imply the arc-adjacent edges' orientation, so all eight are stated explicitly.
func (s *SketchContext) orientFilletedTeeEdges(l []filletLine) error {
	con := s.b.api.Sketch().Constrain(s.index)
	for _, i := range []int{0, 2, 4, 6} { // flange top, undersides, stem bottom
		if _, err := con.Horizontal(l[i].from, l[i].to); err != nil {
			return fmt.Errorf("orient tee edge %d horizontal: %w", i, err)
		}
	}
	for _, i := range []int{1, 3, 5, 7} { // flange ends, stem faces
		if _, err := con.Vertical(l[i].from, l[i].to); err != nil {
			return fmt.Errorf("orient tee edge %d vertical: %w", i, err)
		}
	}
	return nil
}

// sizeFilletedTeeSection shares one radius across the two root fillets, dimensions it, and pins
// EVERY straight edge by an absolute offset from the grounded origin — flange top / stem bottom at
// half the height, the flange ends at half the width, the stem faces at half the stem thickness, and
// the two flange undersides at h/2−tf. Positioning every edge absolutely (never chained to another
// edge) avoids loop-closure redundancy; the offsets both size and centre the section, and the fillet
// tangent points follow from the arcs.
func (s *SketchContext) sizeFilletedTeeSection(o uint64, l []filletLine, a []filletArc, h, b, tw, tf, r string) error {
	con, dim := s.b.api.Sketch().Constrain(s.index), s.b.api.Sketch().Dimension(s.index)
	if _, err := con.EqualRadius(a[0].id, a[1].id); err != nil {
		return fmt.Errorf("equal root fillet radius: %w", err)
	}
	if _, err := dim.Radius(a[0].id, r); err != nil {
		return fmt.Errorf("dimension root fillet radius %q: %w", r, err)
	}
	fin := "(" + h + ") / 2 - (" + tf + ")"
	offsets := []edgeOffset{
		{l[0].id, half(h)}, {l[4].id, half(h)}, // flange top / stem bottom
		{l[1].id, half(b)}, {l[7].id, half(b)}, // flange ends
		{l[2].id, fin}, {l[6].id, fin}, // flange undersides
		{l[3].id, half(tw)}, {l[5].id, half(tw)}, // stem faces
	}
	return applyEdgeOffsets(dim, o, offsets)
}
